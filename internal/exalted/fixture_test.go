package exalted

import (
	"bytes"
	"encoding/json/v2"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/nerdwave-nick/servitor/internal/hallowed"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// patience is how long a trial waits for what must come to pass.
const patience = 10 * time.Second

// secret is a word only the scripture of the trials' vessels holds: no line
// of the channel and no entry of the journal may ever carry it.
const secret = "OMNISSIAH-ONLY-SEES-THIS"

// me is the invoker of the trials, who stands in for the root and for the
// master alike.
var me = os.Getuid()

// ledger is a journal kept in memory.
type ledger struct {
	mu      sync.Mutex
	entries []Entry
}

func (l *ledger) Record(e Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, e)
	return nil
}

func (l *ledger) all() []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.entries)
}

// deeds are the deeds journaled with their verses or items and verdicts,
// e.g. "perform conf triumph".
func (l *ledger) deeds() []string {
	var out []string
	for _, e := range l.all() {
		d := e.Deed
		if e.Verse > 0 {
			d += " " + strconv.Itoa(e.Verse)
		}
		if e.Subject+e.Observance != "" {
			d += " " + e.Subject + e.Observance
		}
		out = append(out, d+" "+string(e.Verdict))
	}
	return out
}

// shrine is the world of a trial: a ground, a store and dominions of its
// own in temporary halls, and a journal in memory.
type shrine struct {
	t       *testing.T
	ground  string
	journal *ledger
	cfg     Config
	unit    *sanction.Unit
}

// hallowUnit is the unit of the rite "hallow" for the trials: the subject
// conf (place 1) transcribes a new scroll, lock (place 3) rewrites a
// sanctum, theme (place 4) binds a tether; the subject at place 2, a
// common one, is no part of it.
func hallowUnit() *sanction.Unit {
	plan := func(aspect string) sanction.Plan {
		return sanction.Plan{Aspect: aspect, Inscriptions: map[string]string{}, Subjects: []sanction.Verse{
			{Subject: "conf", Place: 0, Kind: librarium.KindTranscription, Target: "/etc/sysctl.d/90-hallow.conf",
				Scripture: []byte("kernel.hallow = " + aspect + " # " + secret + "\n")},
			{Subject: "lock", Place: 2, Kind: librarium.KindSanctum, Target: "/etc/hallow.conf", Ward: "hallow",
				Glyph: "#", Scripture: []byte("mode = " + aspect + " " + secret)},
			{Subject: "theme", Place: 3, Kind: librarium.KindTether, Target: "/etc/hallow-theme",
				Anchor: "/usr/share/hallow/" + aspect},
		}}
	}
	return &sanction.Unit{Pattern: sanction.Pattern, UID: me, Rite: "hallow",
		Environment: sanction.Environment{Pattern: sanction.Pattern, Keys: []string{"SERVITOR_ASPECT",
			"SERVITOR_FORMER_ASPECT", "SERVITOR_INSCRIPTION_WHO", "SERVITOR_OMEN", "SERVITOR_RITE"}},
		Aspects: []sanction.Plan{plan("on"), plan("off")}}
}

// elderUnit is hallowUnit as a rite of pattern Mark I was drawn: its
// subjects were verses of a liturgy, numbered by their places.
func elderUnit() *sanction.Unit {
	u := hallowUnit()
	u.Pattern, u.Environment.Pattern = sanction.PatternMarkI, sanction.PatternMarkI
	u.Environment.Keys = slices.DeleteFunc(u.Environment.Keys, func(k string) bool { return k == "SERVITOR_OMEN" })
	for i := range u.Aspects {
		p := &u.Aspects[i]
		p.Liturgy, p.Subjects = p.Subjects, nil
		for j := range p.Liturgy {
			p.Liturgy[j].Verse, p.Liturgy[j].Subject = p.Liturgy[j].Place+1, ""
			p.Liturgy[j].Place = 0
		}
	}
	return u
}

// The exalted subjects of hallowUnit.
var (
	conf  = subject("conf")
	lock  = subject("lock")
	theme = subject("theme")
)

// original is what /etc/hallow.conf holds before any verse.
const original = "base = " + secret + "\n"

// newShrine raises the world of a trial for u (hallowUnit when nil), its
// sanction laid in the store.
func newShrine(t *testing.T, u *sanction.Unit) *shrine {
	t.Helper()
	if u == nil {
		u = hallowUnit()
	}
	sh := &shrine{t: t, ground: t.TempDir(), journal: &ledger{}, unit: u}
	for _, hall := range []string{"/etc/sysctl.d", "/usr/share/hallow", "/usr/bin"} {
		must(t, os.MkdirAll(sh.at(hall), 0o755))
	}
	// The servitor installed in the ground, and the bond that names it as
	// /proc/self/exe names the scroll the exalted servitor runs from.
	sh.scroll("/usr/bin/servitor", "\x7fELF", 0o755)
	self := filepath.Join(t.TempDir(), "exe")
	must(t, os.Symlink(sh.at("/usr/bin/servitor"), self))
	sh.scroll("/etc/hallow.conf", original, 0o600)
	sh.scroll("/usr/share/hallow/on", "on", 0o644)
	sh.scroll("/usr/share/hallow/off", "off", 0o644)
	store := t.TempDir()
	sh.lay(store, u)
	mine := func(uid uint32) bool { return uid == uint32(me) }
	sh.cfg = Config{
		Exalted:   func() bool { return true },
		Getenv:    env(map[string]string{"SUDO_UID": strconv.Itoa(me)}),
		Store:     store,
		Sovereign: mine,
		Ground: hallowed.Ground{Root: sh.ground, Sovereign: mine,
			Lord: hallowed.Owner{UID: uint32(me), GID: uint32(os.Getgid())}},
		Self:      self,
		Dominions: filepath.Join(t.TempDir(), "dominions"),
		Journal:   sh.journal,
		Lament:    io.Discard,
	}
	return sh
}

func env(m map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) { v, ok := m[k]; return v, ok }
}

// lay lays the sanction of u in the store at root as the store keeps it.
func (sh *shrine) lay(root string, u *sanction.Unit) {
	data, err := json.Marshal(sanction.New(u, time.Date(2026, 10, 12, 18, 0, 0, 0, time.UTC)))
	must(sh.t, err)
	p := sanction.Store{Root: root, UID: me}.Path(u.Rite)
	must(sh.t, os.MkdirAll(filepath.Dir(p), 0o750))
	must(sh.t, os.WriteFile(p, data, 0o640))
}

func (sh *shrine) at(path string) string { return filepath.Join(sh.ground, path) }

func (sh *shrine) scroll(path, content string, mode fs.FileMode) {
	must(sh.t, os.WriteFile(sh.at(path), []byte(content), mode))
	must(sh.t, os.Chmod(sh.at(path), mode))
}

// look tells what stands at path: "absent", "→ anchor", or the scripture
// with its seal.
func (sh *shrine) look(path string) string {
	fi, err := os.Lstat(sh.at(path))
	switch {
	case err != nil:
		return "absent"
	case fi.Mode()&fs.ModeSymlink != 0:
		a, _ := os.Readlink(sh.at(path))
		return "→ " + a
	}
	data, err := os.ReadFile(sh.at(path))
	must(sh.t, err)
	return strconv.FormatUint(uint64(fi.Mode().Perm()), 8) + " " + string(data)
}

// untouched asserts that the ground stands as it was raised.
func (sh *shrine) untouched() {
	sh.t.Helper()
	for path, want := range map[string]string{
		"/etc/sysctl.d/90-hallow.conf": "absent",
		"/etc/hallow.conf":             "600 " + original,
		"/etc/hallow-theme":            "absent",
	} {
		if got := sh.look(path); got != want {
			sh.t.Errorf("%s holds %q, want %q", path, got, want)
		}
	}
	if entries, _ := os.ReadDir(sh.at("/etc")); len(entries) != 2 {
		sh.t.Errorf("/etc holds %v", entries)
	}
}

// sigil is the whole sigil of the plan of aspect.
func (sh *shrine) sigil(aspect string) string {
	s, ok := sh.unit.SigilOf(aspect)
	if !ok {
		sh.t.Fatalf("no plan of %q", aspect)
	}
	return s
}

// invoker is the invoker's end of a session.
type invoker struct {
	t     *testing.T
	conn  *os.File
	ch    *Channel
	heard *bytes.Buffer // every byte the invoker heard
	mark  chan int
}

// rouse rouses an exalted servitor over a socketpair, as argv names it.
func (sh *shrine) rouse(args ...string) *invoker {
	sh.t.Helper()
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	must(sh.t, err)
	ours, theirs := os.NewFile(uintptr(fds[0]), "invoker"), os.NewFile(uintptr(fds[1]), "exalted")
	return sh.rouseOn(ours, theirs, theirs, args)
}

// rouseOn rouses an exalted servitor hearing in and answering on out; ours
// is the invoker's end.
func (sh *shrine) rouseOn(ours, in, out *os.File, args []string) *invoker {
	inv := &invoker{t: sh.t, conn: ours, heard: &bytes.Buffer{}, mark: make(chan int, 1)}
	inv.ch = NewChannel(io.TeeReader(ours, inv.heard), ours)
	go func() {
		m := Serve(args, in, out, sh.cfg)
		_ = in.Close()
		if out != in {
			_ = out.Close()
		}
		inv.mark <- m
	}()
	sh.t.Cleanup(func() { _ = ours.Close() })
	return inv
}

// rouseFor rouses an exalted servitor for the aspect of hallow.
func (sh *shrine) rouseFor(aspect string) *invoker {
	return sh.rouse("hallow", sanction.Short(sh.sigil(aspect)), aspect)
}

// handshake is the true handshake of the aspect.
func (sh *shrine) handshake(aspect string) Handshake {
	return Handshake{Protocol: Protocol, UID: me, Rite: "hallow", Aspect: aspect, Sigil: sh.sigil(aspect),
		Former: "off", Inscriptions: map[string]string{"who": "x; echo INJECTED"}}
}

// accepted rouses an exalted servitor for the aspect and is accepted.
func (sh *shrine) accepted(aspect string) *invoker {
	sh.t.Helper()
	inv := sh.rouseFor(aspect)
	if r := inv.say(sh.handshake(aspect)); r.Verdict != Accepted {
		sh.t.Fatalf("the session was not accepted: %+v", r)
	}
	return inv
}

// say speaks v and hears the reply. An exalted servitor that refused
// before it heard may have ended already; its refusal is heard all the same.
func (inv *invoker) say(v any) Reply {
	inv.t.Helper()
	_ = inv.ch.Send(v)
	return inv.hear()
}

// hear hears the next reply.
func (inv *invoker) hear() Reply {
	inv.t.Helper()
	var r Reply
	if err := inv.ch.Receive(&r); err != nil {
		inv.t.Fatalf("no reply was heard: %v", err)
	}
	return r
}

// perform performs the verse or item it, which must triumph.
func (inv *invoker) perform(it *Item) Reply {
	inv.t.Helper()
	r := inv.say(Request{Perform: it})
	if r.Verdict != Triumph || r.To != "perform" || r.Verse != it.Verse || r.Subject != it.Subject ||
		r.Observance != it.Observance {
		inv.t.Fatalf("%s: %+v", it.Named(), r)
	}
	return r
}

// ended awaits the death-mark of the exalted servitor.
func (inv *invoker) ended() int {
	inv.t.Helper()
	select {
	case m := <-inv.mark:
		return m
	case <-time.After(patience):
		inv.t.Fatal("the exalted servitor never ended")
	}
	return -1
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
