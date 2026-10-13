package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/nerdwave-nick/servitor/internal/exalted"
	"github.com/nerdwave-nick/servitor/internal/hallowed"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// altar is the world in which the trials sanction and abjure: stub
// intercessors in a hall of the trial's own, which record their words and
// vouch for the master or not; an exalted servitor raised within the
// trials' own process over a true socketpair, its authority and the root
// stood in for by the invoker of the trials; and a store beneath a hall of
// the trial's own standing in for /var/lib. No intercessor of the machine
// is ever called upon.
type altar struct {
	t        *testing.T
	bin      string
	store    sanction.Store
	journal  *ledger
	cfg      exalted.Config
	attended bool
	vouch    bool
	protocol int // the protocol the exalted servitor names; 0 is its own

	mu    sync.Mutex
	calls [][]string // the words of every intercessor called upon
}

// ledger is a journal of the exalted servitor kept in memory.
type ledger struct {
	mu      sync.Mutex
	entries []exalted.Entry
}

func (l *ledger) Record(e exalted.Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, e)
	return nil
}

func (l *ledger) lines() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []string
	for _, e := range l.entries {
		out = append(out, e.String())
	}
	return out
}

// record is where a stub intercessor records its words.
const record = "SERVITOR_TRIAL_RECORD"

// stubIntercessor records its words, one per NUL, and vouches for the
// master unless SERVITOR_TRIAL_SPURN is set.
const stubIntercessor = "#!/bin/sh\nprintf '%s\\0' \"$0\" \"$@\" > \"$" + record + "\"\n" +
	"[ -z \"$SERVITOR_TRIAL_SPURN\" ] || exit 1\n"

// raiseAltar raises the altar of the trial with the intercessors present.
func (e *env) raiseAltar(present ...string) *altar {
	t := e.t
	t.Helper()
	a := &altar{t: t, bin: t.TempDir(), journal: &ledger{}, attended: true, vouch: true}
	for _, name := range present {
		if err := os.WriteFile(filepath.Join(a.bin, name), []byte(stubIntercessor), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	lib := filepath.Join(t.TempDir(), "var", "lib")
	if err := os.MkdirAll(lib, 0o755); err != nil {
		t.Fatal(err)
	}
	formerRoot, formerWorld, formerSummon := rituals.SanctionRoot, rituals.Intercession, rituals.Summon
	t.Cleanup(func() {
		rituals.SanctionRoot, rituals.Intercession, rituals.Summon = formerRoot, formerWorld, formerSummon
	})
	rituals.SanctionRoot = filepath.Join(lib, "servitor", "sanctions")
	a.store = sanction.Store{Root: rituals.SanctionRoot, UID: os.Getuid()}

	w := formerWorld
	w.LookPath = func(name string) (string, error) {
		p := filepath.Join(a.bin, name)
		if _, err := os.Stat(p); err != nil {
			return "", err
		}
		return p, nil
	}
	w.Attended = func() bool { return a.attended }
	rituals.Intercession = w
	rituals.Summon = a.summon

	self := filepath.Join(t.TempDir(), "exe")
	if err := os.Symlink(filepath.Join(w.Ground.Root, w.Servitor), self); err != nil {
		t.Fatal(err)
	}
	me := uint32(os.Getuid())
	mine := func(uid uint32) bool { return uid == me }
	a.cfg = exalted.Config{
		Exalted: func() bool { return true },
		Getenv: func(k string) (string, bool) {
			return map[string]string{"SUDO_UID": strconv.Itoa(int(me))}[k], k == "SUDO_UID"
		},
		Store:     rituals.SanctionRoot,
		Sovereign: mine,
		Ground:    hallowed.Ground{Root: w.Ground.Root, Sovereign: mine, Lord: hallowed.Owner{UID: me, GID: uint32(os.Getgid())}},
		Self:      self,
		Journal:   a.journal,
		Lament:    io.Discard,
		Voice:     func(exalted.Sitting) (exalted.Speaker, error) { return nil, errors.New("no speech in these trials") },
		Kin:       func(int) (int, error) { return os.Getgid(), nil },
		Now:       func() time.Time { return sanctionedAt },
	}
	return a
}

// summon calls upon the stub intercessor as the machine would, never
// lending it the channel; when it vouches, the exalted servitor it would
// have raised is raised within the trials upon the channel, roused by the
// words that follow the servitor in the intercessor's.
func (a *altar) summon(cmd *exec.Cmd, end *os.File) (func() error, error) {
	if filepath.Dir(cmd.Path) != a.bin {
		a.t.Fatalf("the trials would call upon %s", cmd.Path)
	}
	rec := filepath.Join(a.t.TempDir(), "words")
	stub := exec.Command(cmd.Path, cmd.Args[1:]...)
	stub.Env = append(os.Environ(), record+"="+rec)
	if !a.vouch {
		stub.Env = append(stub.Env, "SERVITOR_TRIAL_SPURN=1")
	}
	ran := stub.Run()
	words, err := os.ReadFile(rec)
	if err != nil {
		a.t.Fatalf("the stub recorded nothing: %v", err)
	}
	called := strings.Split(strings.TrimSuffix(string(words), "\x00"), "\x00")
	a.mu.Lock()
	a.calls = append(a.calls, called)
	a.mu.Unlock()
	if ran != nil {
		return func() error { return ran }, nil
	}
	i := slices.Index(called, exalted.Mode)
	if i < 0 || called[i-1] != "/usr/bin/servitor" {
		a.t.Fatalf("the intercessor raises no exalted servitor: %q", called)
	}
	fd, err := unix.Dup(int(end.Fd()))
	if err != nil {
		return nil, err
	}
	conn := os.NewFile(uintptr(fd), "exalted servitor")
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = conn.Close() }()
		if a.protocol != 0 {
			fmt.Fprintf(conn, `{"to":"handshake","verdict":"refused","protocol":%d}`+"\n", a.protocol)
			return
		}
		exalted.Serve(called[i+1:], conn, conn, a.cfg)
	}()
	return func() error { <-done; return nil }, nil
}

// called are the words of every intercessor called upon, its path first.
func (a *altar) called() [][]string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.calls)
}

// sudoWords are the words sudo is called upon with to sanction the rite
// under sigil.
func (a *altar) sudoWords(rite, sigil string) []string {
	short := sanction.Short(sigil)
	return []string{filepath.Join(a.bin, "sudo"), "-k", "-p", "servitor: authenticate to sanction " + rite + " (sigil " +
		short + "): ", "--", "/usr/bin/servitor", "__exalt", "--sanction", rite, short}
}

// TestSanction_LaysTheSanctionWithOneProof: the sanctioning of a rite shows
// its master the unit and its sigil, calls upon the intercessor once, and
// the exalted servitor lays the sanction in the store, the master's to read
// and the root's to write; the rite then stands sanctioned. A unit
// unchanged is not sanctioned anew, and no proof is asked; a changed one
// shows only its difference.
func TestSanction_LaysTheSanctionWithOneProof(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/split-lock.json", splitLockJSON)
	a := e.raiseAltar("sudo", "run0", "pkexec")
	u, err := (&rituals.Servitor{Librarium: e.lib(), Sanctions: &a.store}).Unit(e.lib().Rites["split-lock"])
	if err != nil {
		t.Fatal(err)
	}

	out, errOut, code := e.run("sanction", "split-lock")
	if code != 0 || errOut != "" {
		t.Fatalf("sanction: exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	for _, words := range []string{u.Sigil(), "observance exalt · incantation", "words: echo {{aspect}} >> " + e.target +
		".exalted", "stand sanctioned under the sigil " + sanction.Short(u.Sigil())} {
		if !strings.Contains(out, strings.ReplaceAll(words, "{{aspect}}", "on")) {
			t.Errorf("the master is not shown %q:\n%s", words, out)
		}
	}
	grimdark(t, out)
	if calls := a.called(); len(calls) != 1 || !slices.Equal(calls[0], a.sudoWords("split-lock", u.Sigil())) {
		t.Errorf("the intercessors were called upon as %q\nwant once as %q", calls, a.sudoWords("split-lock", u.Sigil()))
	}
	kept, err := a.store.Read("split-lock")
	if err != nil || kept == nil || kept.Sigil != u.Sigil() {
		t.Fatalf("the store keeps %+v, %v", kept, err)
	}
	for path, seal := range map[string]fs.FileMode{filepath.Dir(a.store.Root): 0o755, a.store.Root: 0o755,
		filepath.Dir(a.store.Path("split-lock")): 0o750, a.store.Path("split-lock"): 0o640} {
		if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != seal ||
			int(fi.Sys().(*syscall.Stat_t).Gid) != os.Getgid() {
			t.Errorf("%s: %v, %v; want sealed %04o", path, fi.Mode(), err, seal)
		}
	}
	short := sanction.Short(u.Sigil())
	if lines := a.journal.lines(); len(lines) != 1 || lines[0] != fmt.Sprintf(`uid=%d rite="split-lock" sigil=%s `+
		"former=none deed=sanction verdict=sanctioned", os.Getuid(), short) {
		t.Errorf("journaled %q", lines)
	}
	if got := e.mustRun("augury", "split-lock", "sanction"); got != "sanctioned\n" {
		t.Errorf("the rite stands %q", got)
	}

	out, _, code = e.run("sanction", "split-lock")
	if code != 0 || !strings.Contains(out, "sanctioned already under the sigil "+short) || len(a.called()) != 1 {
		t.Errorf("unchanged: exit %d, %d calls\n%s", code, len(a.called()), out)
	}

	e.addRite("rites/split-lock.json", strings.Replace(splitLockJSON, "echo {{aspect}}", "echo anew {{aspect}}", 1))
	out, errOut, code = e.run("sanction", "split-lock")
	if code != 0 || len(a.called()) != 2 {
		t.Fatalf("changed: exit %d, %d calls\nstdout: %s\nstderr: %s", code, len(a.called()), out, errOut)
	}
	if !strings.Contains(out, "under the sigil "+short) || !strings.Contains(out, "\n-       words: echo on") ||
		!strings.Contains(out, "\n+       words: echo anew on") || strings.Contains(out, "subject veil") {
		t.Errorf("the difference is not shown:\n%s", out)
	}
	if lines := a.journal.lines(); len(lines) != 2 || !strings.Contains(lines[1], "former="+short) {
		t.Errorf("journaled %q", lines)
	}
}

// TestSanction_NeedsATerminal: a sanctioning no terminal attends is refused
// with the death-mark 2, and no intercessor is called upon; an unchanged
// unit still says so and exits 0.
func TestSanction_NeedsATerminal(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/split-lock.json", splitLockJSON)
	a := e.raiseAltar("sudo", "run0", "pkexec")
	a.attended = false
	out, errOut, code := e.run("sanction", "split-lock")
	if code != 2 || out != "" || !strings.Contains(errOut, "no terminal attends") || len(a.called()) != 0 {
		t.Errorf("exit %d, %d calls\nstdout: %s\nstderr: %s", code, len(a.called()), out, errOut)
	}
	grimdark(t, errOut)
	if _, err := os.Stat(a.store.Root); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the store was raised: %v", err)
	}
	a.attended = true
	e.mustRun("sanction", "split-lock")
	a.attended = false
	if out, _, code := e.run("sanction", "split-lock"); code != 0 || !strings.Contains(out, "sanctioned already") {
		t.Errorf("unchanged without a terminal: exit %d\n%s", code, out)
	}
}

// TestSanction_ADismissedProofWritesNothing: when the intercessor does not
// vouch for the master, the sanctioning ends with the death-mark 1, and
// nothing is written.
func TestSanction_ADismissedProofWritesNothing(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/split-lock.json", splitLockJSON)
	a := e.raiseAltar("sudo")
	a.vouch = false
	out, errOut, code := e.run("sanction", "split-lock")
	if code != 1 || !strings.Contains(errOut, "did not vouch") || len(a.called()) != 1 {
		t.Errorf("exit %d, %d calls\nstdout: %s\nstderr: %s", code, len(a.called()), out, errOut)
	}
	grimdark(t, errOut)
	if kept, err := a.store.Read("split-lock"); kept != nil || err != nil || len(a.journal.lines()) != 0 {
		t.Errorf("the store keeps %v, %v; journaled %q", kept, err, a.journal.lines())
	}
}

// TestSanction_TheInquisitionComesFirst: a heretical rite, a rite without
// exalted verses and a rite never recorded are refused with the death-mark
// 2 before any proof; the words name a sanction kept under such a name as
// forsaken, and its abjuring.
func TestSanction_TheInquisitionComesFirst(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/split-lock.json", splitLockJSON)
	e.addRite("rites/heretic.json", strings.Replace(splitLockJSON, `"exalted": true`, `"exalted": "yes"`, 1))
	a := e.raiseAltar("sudo", "run0", "pkexec")
	for _, as := range []string{"gone", "mouse-autohide-toggle"} {
		e.sanctify(a.store, "split-lock", as)
	}
	for rite, words := range map[string][]string{
		"heretic":               {"heresy", "servitor inquisition heretic"},
		"mouse-autohide-toggle": {"holds nothing exalted", "forsaken", "servitor abjure mouse-autohide-toggle"},
		"gone":                  {`no rite named "gone"`, "forsaken", "servitor abjure gone"},
		"never":                 {`no rite named "never"`},
	} {
		out, errOut, code := e.run("sanction", rite)
		if code != 2 || out != "" || len(a.called()) != 0 {
			t.Errorf("%s: exit %d\nstdout: %s\nstderr: %s", rite, code, out, errOut)
		}
		for _, w := range words {
			if !strings.Contains(errOut, w) {
				t.Errorf("%s: the words do not name %q:\n%s", rite, w, errOut)
			}
		}
		if rite == "never" && strings.Contains(errOut, "forsaken") {
			t.Errorf("never: %s", errOut)
		}
	}
}

// TestSanction_RefusesWhenNoIntercessorCanVouch: no intercessor to be
// found, an installed servitor amiss or one of another protocol, and the
// binharic, refuse with the death-mark 2 and write nothing.
func TestSanction_RefusesWhenNoIntercessorCanVouch(t *testing.T) {
	for what, c := range map[string]struct {
		present []string
		mar     func(a *altar)
		args    []string
		words   string
	}{
		"no intercessor":        {words: "no intercessor is found"},
		"another protocol":      {present: []string{"sudo"}, mar: func(a *altar) { a.protocol = 99 }, words: "install the servitor anew"},
		"in binharic":           {present: []string{"sudo"}, args: []string{"--binharic"}, words: "binharic"},
		"an installed servitor": {present: []string{"sudo"}, mar: func(a *altar) { rituals.Intercession.Servitor = "/usr/bin/xenos" }, words: "/usr/bin/xenos"},
	} {
		t.Run(what, func(t *testing.T) {
			e := newEnv(t)
			e.addRite("rites/split-lock.json", splitLockJSON)
			a := e.raiseAltar(c.present...)
			if c.mar != nil {
				c.mar(a)
			}
			out, errOut, code := e.run(append([]string{"sanction", "split-lock"}, c.args...)...)
			if code != 2 || !strings.Contains(errOut, c.words) {
				t.Errorf("exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
			}
			grimdark(t, errOut)
			if kept, _ := a.store.Read("split-lock"); kept != nil {
				t.Error("a sanction was laid")
			}
		})
	}
}

// lib is the test Librarium as it stands.
func (e *env) lib() *librarium.Librarium { return librarium.Load(e.cfgDir) }
