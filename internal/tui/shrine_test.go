package tui

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/sys/unix"

	"github.com/nerdwave-nick/servitor/internal/exalted"
	"github.com/nerdwave-nick/servitor/internal/hallowed"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// shrine is the world in which the cogitator's trials sanction, abjure and
// perform exalted rites: stub intercessors in a hall of the trial's own,
// which append their words to a scroll and vouch for the master or not; an
// exalted servitor raised within the trials' own process over a true
// socketpair, its authority and the root stood in for by the invoker of the
// trials; a store beneath a hall of the trial's own standing in for
// /var/lib; and, in place of the terminal the cogitator lends sudo, a
// scroll of the trial's own. No intercessor of the machine is ever called
// upon.
type shrine struct {
	h       *harness
	bin     string
	calls   string // the scroll the stubs append their words to
	store   sanction.Store
	cfg     exalted.Config
	journal *ledger

	mu       sync.Mutex
	lent     int          // how often the cogitator lent its terminal
	terminal bytes        // what was spoken upon the terminal while it was lent
	detached []bool       // whether each intercessor was summoned in a session of its own
	marks    []int        // the death-marks of the exalted servitors raised
	summon   func() error // when not nil, how the next exalted servitor answers instead
}

// bytes is a scroll safe to be written from several goroutines.
type bytes struct {
	mu sync.Mutex
	b  strings.Builder
}

func (b *bytes) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *bytes) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
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

func (l *ledger) deeds() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []string
	for _, e := range l.entries {
		d := e.Deed
		if e.Verse > 0 {
			d += " " + strconv.Itoa(e.Verse)
		}
		out = append(out, d+" "+string(e.Verdict))
	}
	return out
}

// The stubs record each call as one line of words parted by \x1f, and
// vouch for the master unless spurnVar is set: then they lament upon stderr
// as an intercessor would, and end with 1.
const (
	callsVar = "SERVITOR_TRIAL_CALLS"
	spurnVar = "SERVITOR_TRIAL_SPURN"
	stub     = "#!/bin/sh\nprintf '%s\\037' \"$0\" \"$@\" >> \"$" + callsVar + "\"\nprintf '\\n' >> \"$" + callsVar +
		"\"\n[ -z \"$" + spurnVar + "\" ] || { echo 'the proof of the master was refused thrice' >&2; exit 1; }\n"
)

// raiseShrine raises the shrine of the trial with the intercessors present,
// and re-reads the Librarium within it.
func (h *harness) raiseShrine(present ...string) *shrine {
	t := h.t
	t.Helper()
	tongue, err := exec.LookPath("bash")
	if err == nil {
		tongue, err = filepath.EvalSymlinks(tongue)
	}
	if err == nil {
		_, err = hallowed.Machine.Program(tongue)
	}
	if err != nil {
		t.Skipf("no bash of the root upon this machine: %v", err)
	}
	s := &shrine{h: h, bin: t.TempDir(), calls: filepath.Join(t.TempDir(), "calls"), journal: &ledger{}}
	t.Setenv(callsVar, s.calls)
	for _, name := range present {
		mustDo(t, os.WriteFile(filepath.Join(s.bin, name), []byte(stub), 0o755))
	}
	lib := filepath.Join(t.TempDir(), "var", "lib")
	mustDo(t, os.MkdirAll(lib, 0o755))
	formerRoot, formerWorld, formerSummon := rituals.SanctionRoot, rituals.Intercession, rituals.Summon
	t.Cleanup(func() {
		rituals.SanctionRoot, rituals.Intercession, rituals.Summon = formerRoot, formerWorld, formerSummon
	})
	rituals.SanctionRoot = filepath.Join(lib, "servitor", "sanctions")
	s.store = sanction.Store{Root: rituals.SanctionRoot, UID: os.Getuid()}
	w := formerWorld
	w.LookPath = func(name string) (string, error) {
		p := filepath.Join(s.bin, name)
		if _, err := os.Stat(p); err != nil {
			return "", exec.ErrNotFound
		}
		return p, nil
	}
	w.Attended = func() bool { return true } // the cogitator stands at a terminal
	rituals.Intercession, rituals.Summon = w, s.raise

	self := filepath.Join(t.TempDir(), "exe")
	mustDo(t, os.Symlink(filepath.Join(w.Ground.Root, w.Servitor), self))
	me := uint32(os.Getuid())
	mine := func(uid uint32) bool { return uid == me }
	getenv := func(k string) (string, bool) {
		v, ok := map[string]string{"SUDO_UID": strconv.Itoa(int(me)), "LANG": "C.UTF-8"}[k]
		return v, ok
	}
	s.cfg = exalted.Config{
		Exalted:   func() bool { return true },
		Getenv:    getenv,
		Store:     rituals.SanctionRoot,
		Sovereign: mine,
		Ground:    hallowed.Ground{Root: w.Ground.Root, Sovereign: mine, Lord: hallowed.Owner{UID: me, GID: uint32(os.Getgid())}},
		Self:      self,
		Dominions: filepath.Join(t.TempDir(), "dominions"),
		Journal:   s.journal,
		Lament:    io.Discard,
		Voice: exalted.Speech{Halls: filepath.Join(t.TempDir(), "run", "servitor"), Sovereign: mine,
			Tongues: hallowed.Machine, Getenv: getenv}.Voice,
		Kin: func(int) (int, error) { return os.Getgid(), nil },
	}
	h.m.exec = s.lend
	h.keys("r")
	return s
}

func mustDo(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// lend stands in for the terminal the cogitator lends: the command runs
// as bubbletea would run it, the scroll of the trial being its terminal,
// and what it ends with is delivered as bubbletea delivers it.
func (s *shrine) lend(c tea.ExecCommand, fn tea.ExecCallback) tea.Cmd {
	return func() tea.Msg {
		s.mu.Lock()
		s.lent++
		s.mu.Unlock()
		c.SetStdin(strings.NewReader(""))
		c.SetStdout(io.Discard)
		c.SetStderr(&s.terminal)
		go func() { s.h.inbox <- fn(c.Run()) }()
		return nil
	}
}

// raise calls upon the stub intercessor as the machine would, never lending
// it the channel, its stderr that the cogitator laid; when it vouches, the
// exalted servitor it would have raised is raised within the trials upon
// the channel, roused by the words that follow the servitor in the
// intercessor's, speaking its laments upon that same stderr.
func (s *shrine) raise(cmd *exec.Cmd, end *os.File) (func() error, error) {
	t := s.h.t
	if filepath.Dir(cmd.Path) != s.bin {
		t.Errorf("the trials would call upon %s", cmd.Path)
		return nil, errors.New("no intercessor of the machine")
	}
	s.mu.Lock()
	s.detached = append(s.detached, cmd.SysProcAttr != nil && cmd.SysProcAttr.Setsid)
	instead := s.summon
	s.mu.Unlock()
	called := exec.Command(cmd.Path, cmd.Args[1:]...)
	called.Stderr = cmd.Stderr
	if err := called.Run(); err != nil {
		return func() error { return err }, nil
	}
	if instead != nil {
		_, _ = fmt.Fprintf(end, `{"to":"handshake","verdict":"refused","protocol":99}`+"\n")
		return instead, nil
	}
	i := slices.Index(cmd.Args, exalted.Mode)
	if i < 1 || cmd.Args[i-1] != "/usr/bin/servitor" {
		t.Fatalf("the intercessor raises no exalted servitor: %q", cmd.Args)
	}
	fd, err := unix.Dup(int(end.Fd()))
	if err != nil {
		return nil, err
	}
	conn := os.NewFile(uintptr(fd), "exalted servitor")
	cfg := s.cfg
	if cmd.Stderr != nil {
		cfg.Lament = cmd.Stderr
	}
	done := make(chan int, 1)
	go func() {
		defer func() { _ = conn.Close() }()
		done <- exalted.Serve(cmd.Args[i+1:], conn, conn, cfg)
	}()
	return func() error {
		m := <-done
		s.mu.Lock()
		s.marks = append(s.marks, m)
		s.mu.Unlock()
		if m != 0 {
			return fmt.Errorf("the exalted servitor ended with %d", m)
		}
		return nil
	}, nil
}

// called are the words of every intercessor called upon, its path first.
func (s *shrine) called() [][]string {
	s.h.t.Helper()
	data, err := os.ReadFile(s.calls)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	mustDo(s.h.t, err)
	var out [][]string
	for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		out = append(out, strings.Split(strings.TrimSuffix(line, "\x1f"), "\x1f"))
	}
	return out
}

// lends is how often the cogitator lent its terminal.
func (s *shrine) lends() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lent
}

// sanctify lays the sanction of the rite name as it stands now.
func (s *shrine) sanctify(name string) *sanction.Sanction {
	t := s.h.t
	t.Helper()
	lib := librarium.Load(s.h.dir)
	u, err := (&rituals.Servitor{Librarium: lib, Sanctions: &s.store}).Unit(lib.Rites[name])
	mustDo(t, err)
	kept := sanction.New(u, time.Now())
	data, err := json.Marshal(kept)
	mustDo(t, err)
	p := s.store.Path(name)
	mustDo(t, os.MkdirAll(filepath.Dir(p), 0o750))
	mustDo(t, os.WriteFile(p, data, 0o640))
	s.h.keys("r")
	return kept
}

// awaitProven delivers the messages of a sanctioning or an abjuring until
// it has ended.
func (h *harness) awaitProven() {
	h.t.Helper()
	for {
		if _, ended := h.next().(provenMsg); ended {
			return
		}
	}
}

// said is what the verses of hallowRite said, line by line.
func (h *harness) said() []string {
	data, err := os.ReadFile(filepath.Join(h.dataDir, "said"))
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

// listLine is what the line of the rite name in the list of rites shows
// after its name: its aspect or standing, and the mark of its sanction.
func (h *harness) listLine(name string) string {
	h.t.Helper()
	for _, line := range strings.Split(h.screen(), "\n") {
		r := []rune(line)
		if len(r) < 40 {
			continue
		}
		if _, after, ok := strings.Cut(string(r[:40]), " "+name+" "); ok {
			return after
		}
	}
	h.t.Fatalf("the list names no rite %q:\n%s", name, h.screen())
	return ""
}

// hallowRite speaks, before its one subject changes, a common observance,
// then an exalted one that tells who speaks it and utters words of its own;
// both revert by word.
const hallowRite = `{
  "pattern": "Mark II",
  "purpose": "Hallow the machine",
  "aspects": ["on", "off"],
  "vox-cast": ["success"],
  "subjects": [
    {"name": "mark", "designation": "machine-spirit", "auspex": {"scan": "cat $DATA/mark"},
     "enact": {"*": "echo {{omen}} > $DATA/mark"}, "introit": ["common", "hallowed"]}
  ],
  "observances": [
    {"name": "common", "designation": "incantation", "words": "echo 1 >> $DATA/said",
     "reversion": "echo undo 1 >> $DATA/said"},
    {"name": "hallowed", "designation": "incantation",
     "words": "echo exalted {{aspect}} $HOME >> $DATA/said; echo spoken {{aspect}}",
     "reversion": "echo undo exalted >> $DATA/said", "exalted": true}
  ]
}`

// TestOverview_ShowsTheSanctionOfEveryRite: the list marks every rite with
// exalted verses by its sanction standing — ✠ sanctioned, ☓ lapsed, ○
// unsanctioned — and a rite whose sanction is forsaken by †; the detail
// tells the standing and the sigil, and marks every exalted verse ✠. A
// sanction whose rite is gone, and a rite without exalted verses or
// sanction, trouble nothing.
func TestOverview_ShowsTheSanctionOfEveryRite(t *testing.T) {
	h := newHarness(t)
	s := h.raiseShrine("sudo")
	h.writeRite("hallow", hallowRite)
	h.writeRite("lapsed", hallowRite)
	h.writeRite("plain", hallowRite)
	h.writeRite("gone", hallowRite)
	kept := s.sanctify("hallow")
	s.sanctify("lapsed")
	s.sanctify("plain")
	s.sanctify("gone")
	h.writeRite("lapsed", strings.Replace(hallowRite, "echo spoken", "echo uttered", 1))
	h.writeRite("plain", strings.Replace(hallowRite, `, "exalted": true`, "", 1))
	h.writeRite("unsanctioned", hallowRite)
	mustDo(t, os.Remove(filepath.Join(h.dir, "rites", "gone.json")))
	h.keys("r")

	for name, glyph := range map[string]string{"hallow": "✠", "lapsed": "☓", "unsanctioned": "○", "plain": "†"} {
		if line := h.listLine(name); !strings.Contains(line, glyph) {
			t.Errorf("the line of %s lacks %s: %q", name, glyph, line)
		}
	}
	if line := h.listLine("mouse"); strings.ContainsAny(line, "✠☓†") {
		t.Errorf("a rite without exalted verses bears a sanction: %q", line)
	}
	// hallow, lapsed, mouse, plain, unsanctioned
	h.keys("g")
	h.mustShow("SANCTION", "✠ sanctioned", "sigil "+sanction.Short(kept.Sigil), "» common · echo 1 >>",
		"» ✠ hallowed · echo exalted {{aspect}}")
	h.keys("j")
	h.mustShow("☓ lapsed since sigil", "s to sanction anew")
	h.keys("j")
	h.mustNotShow("SANCTION")
	h.keys("j")
	h.mustShow("† forsaken", "abjure")
	h.keys("j")
	h.mustShow("○ unsanctioned", "s to sanction its exalted subjects")

	// foresight marks the exalted verse and tells the standing
	h.keys("g", "enter", "p")
	h.mustShow("Foresight of aspect", "stand sanctioned under the sigil", "observance hallowed · incantation",
		"Nothing was performed.")
	if len(s.called()) != 0 {
		t.Errorf("the intercessor was called upon: %q", s.called())
	}
}

// TestSanction_ShowsTheUnitThenAsksOneProof: s shows the master the exalted
// verses of the rite as they would be sanctioned, under their sigil, and
// enter asks the intercessor for the one proof that lays them: sudo is lent
// the terminal for it, once, and the cogitator takes it back. A rite
// changed since shows only its difference, beneath the sigil it bore; an
// unchanged one is told and asks no proof. a abjures the sanction with one
// proof more.
func TestSanction_ShowsTheUnitThenAsksOneProof(t *testing.T) {
	h := newHarness(t)
	s := h.raiseShrine("sudo", "run0")
	h.writeRite("hallow", hallowRite)
	h.send(tea.WindowSizeMsg{Width: 160, Height: 70})
	h.keys("r", "g", "s")
	h.mustShow("Sanction of hallow", "offered for sanction under the sigil", "echo exalted on",
		"The intercessor asks the master's proof")
	h.send(tea.WindowSizeMsg{Width: 120, Height: 36}) // the list of rites as wide as listLine reads it
	if len(s.called()) != 0 {
		t.Fatalf("the intercessor was called upon before enter: %q", s.called())
	}
	h.keys("enter")
	h.mustShow("awaits the master's proof")
	h.awaitProven()
	h.mustShow("stand sanctioned under the sigil")
	lib := librarium.Load(h.dir)
	u, err := (&rituals.Servitor{Librarium: lib, Sanctions: &s.store}).Unit(lib.Rites["hallow"])
	mustDo(t, err)
	short := sanction.Short(u.Sigil())
	want := []string{filepath.Join(s.bin, "sudo"), "-k", "-p", "servitor: authenticate to sanction hallow (sigil " +
		short + "): ", "--", "/usr/bin/servitor", exalted.Mode, exalted.SanctionWord, "hallow", short}
	if calls := s.called(); len(calls) != 1 || !slices.Equal(calls[0], want) {
		t.Fatalf("the intercessors were called upon as %q, want once as %q", calls, want)
	}
	if n := s.lends(); n != 1 {
		t.Errorf("the terminal was lent %d times", n)
	}
	if kept, lament := s.store.Read("hallow"); kept == nil || lament != nil || kept.Sigil != u.Sigil() {
		t.Fatalf("the sanction laid: %+v, %v", kept, lament)
	}
	if line := h.listLine("hallow"); !strings.Contains(line, "✠") {
		t.Errorf("the rite is not marked sanctioned: %q", line)
	}

	// unchanged: told, and no proof is asked
	h.keys("s")
	h.mustShow("stand sanctioned already under the sigil "+short, "Nothing is to be sanctioned anew")
	h.keys("enter")
	if n := len(s.called()); n != 1 {
		t.Fatalf("an unchanged unit asked a proof: %d calls", n)
	}
	h.keys("esc")

	// changed: only the difference is shown, beneath both sigils
	h.writeRite("hallow", strings.Replace(hallowRite, "echo spoken", "echo uttered", 1))
	h.keys("r", "s")
	h.mustShow("changed since they were sanctioned under the sigil "+short, "- ", "+ ", "uttered")
	h.keys("enter")
	h.awaitProven()
	if n := len(s.called()); n != 2 {
		t.Fatalf("the intercessor was called upon %d times", n)
	}
	if !strings.Contains(strings.Join(s.journal.deeds(), "\n"), "sanction sanctioned") {
		t.Errorf("journaled %q", s.journal.deeds())
	}

	// abjure, with one proof
	h.keys("s", "a")
	h.awaitProven()
	h.mustShow("The sanction of hallow is abjured")
	calls := s.called()
	if len(calls) != 3 || !slices.Equal(calls[2][len(calls[2])-3:], []string{exalted.Mode, exalted.AbjureWord, "hallow"}) {
		t.Fatalf("abjured as %q", calls)
	}
	if kept, lament := s.store.Read("hallow"); kept != nil || lament != nil {
		t.Errorf("the sanction stands: %+v, %v", kept, lament)
	}
	if line := h.listLine("hallow"); !strings.Contains(line, "○") {
		t.Errorf("the rite is not marked unsanctioned: %q", line)
	}
	if n := s.lends(); n != 3 {
		t.Errorf("the terminal was lent %d times", n)
	}
}

// TestSanction_RefusalsAndForsakenSanctions: a rite without exalted verses
// offers nothing to sanction, yet its forsaken sanction may be abjured; a
// heretical rite and a dismissed proof are told as heresies with the words
// the intercessor spoke. Nothing panics, and nothing is laid.
func TestSanction_RefusalsAndForsakenSanctions(t *testing.T) {
	h := newHarness(t)
	s := h.raiseShrine("sudo")
	h.writeRite("hallow", hallowRite)
	s.sanctify("hallow")
	h.writeRite("hallow", strings.Replace(hallowRite, `, "exalted": true`, "", 1))
	h.keys("r", "g", "s")
	h.mustShow("holds no exalted verse", "forsaken", "abjure")
	h.keys("enter")
	if len(s.called()) != 0 {
		t.Fatal("a rite without exalted verses asked a proof")
	}
	h.keys("a")
	h.awaitProven()
	h.mustShow("The sanction of hallow is abjured")
	if calls := s.called(); len(calls) != 1 || calls[0][len(calls[0])-1] != "hallow" {
		t.Fatalf("abjured as %q", calls)
	}

	// a rite of no exalted verse and no sanction: nothing to sanction, nothing to abjure
	h.keys("j", "s")
	h.mustShow("holds no exalted verse")
	h.keys("a", "esc")

	// a dismissed proof
	h.writeRite("hallow", hallowRite)
	t.Setenv(spurnVar, "1")
	h.keys("r", "g", "s", "enter")
	h.awaitProven()
	h.mustShow("did not vouch", "the proof of the master was refused thrice", "Nothing was written")
	if kept, _ := s.store.Read("hallow"); kept != nil {
		t.Error("a dismissed proof laid a sanction")
	}
	h.keys("esc")

	// a heretical rite
	h.writeRite("hallow", strings.Replace(hallowRite, `"exalted": true`, `"exalted": "yes"`, 1))
	h.keys("r", "g", "s")
	h.mustShow("heres")
	h.keys("enter", "a", "esc")
	if n := len(s.called()); n != 2 {
		t.Errorf("the intercessor was called upon %d times", n)
	}
}

// TestInvoke_TheCogitatorPerformsExaltedVerses: the cogitator performs a
// sanctioned rite's exalted verses through the exalted servitor alone,
// raised with one proof after the pre-flight: sudo is lent the terminal
// for it and the cogitator takes it back, while the channel stays upon its
// socketpair. The exalted speech hears the root's home, never its
// invoker's; its words are kept for O; the chronicle names the intercessor.
func TestInvoke_TheCogitatorPerformsExaltedVerses(t *testing.T) {
	h := newHarness(t)
	s := h.raiseShrine("sudo", "run0")
	h.writeRite("hallow", hallowRite)
	kept := s.sanctify("hallow")
	sigil, _ := kept.SigilOf("on")
	h.keys("g", "enter", "1")
	h.await()
	h.mustShow("The rite hallow is performed: dormant → on")
	if said := h.said(); len(said) != 2 || said[0] != "1" || said[1] != "exalted on /root" {
		t.Errorf("said %q", said)
	}
	want := []string{filepath.Join(s.bin, "sudo"), "-k", "-p", "servitor: authenticate to perform hallow (sigil " +
		sanction.Short(sigil) + "): ", "--", "/usr/bin/servitor", exalted.Mode, "hallow", sanction.Short(sigil), "on"}
	if calls := s.called(); len(calls) != 1 || !slices.Equal(calls[0], want) {
		t.Fatalf("the intercessors were called upon as %q, want once as %q", calls, want)
	}
	if n := s.lends(); n != 1 {
		t.Errorf("the terminal was lent %d times", n)
	}
	if got := s.journal.deeds(); !slices.Equal(got, []string{"accept accepted", "perform triumph",
		"finish finished"}) {
		t.Errorf("journaled %q", got)
	}
	if line := h.listLine("hallow"); !strings.Contains(line, "on") || !strings.Contains(line, "✠") {
		t.Errorf("the rite stands as %q", line)
	}
	h.keys("O")
	h.mustShow("observance hallowed", "spoken on")
	h.keys("esc", "h")
	h.mustShow("hallow", "dormant → on")
}

// TestInvoke_Run0IsBiddenForget: an invocation through run0 lends no
// terminal — run0 is summoned in a session of its own, so that only the
// graphical agent asks — and once it has ended, run0 is bidden forget the
// master's proof (run0 -k), so that the next asks anew. sudo, which forgets
// as it is called, is bidden nothing.
func TestInvoke_Run0IsBiddenForget(t *testing.T) {
	h := newHarness(t)
	s := h.raiseShrine("run0")
	h.writeRite("hallow", hallowRite)
	kept := s.sanctify("hallow")
	sigil, _ := kept.SigilOf("on")
	h.keys("g", "enter", "1")
	h.await()
	h.mustShow("The rite hallow is performed")
	run0 := filepath.Join(s.bin, "run0")
	want := [][]string{{run0, "--unit=servitor-hallow-" + sanction.Short(sigil)[:8], "--pipe", "--chdir=/",
		"/usr/bin/servitor", exalted.Mode, "hallow", sanction.Short(sigil), "on"}, {run0, "-k"}}
	if calls := s.called(); !slices.EqualFunc(calls, want, slices.Equal) {
		t.Fatalf("the intercessors were called upon as %q, want %q", calls, want)
	}
	s.mu.Lock()
	detached := slices.Clone(s.detached)
	s.mu.Unlock()
	if s.lends() != 0 || !slices.Equal(detached, []bool{true}) {
		t.Errorf("lent the terminal %d times; summoned in a session of its own: %v", s.lends(), detached)
	}

	// sudo is bidden nothing
	mustDo(t, os.WriteFile(filepath.Join(s.bin, "sudo"), []byte(stub), 0o755))
	h.keys("enter", "2")
	h.await()
	if calls := s.called(); len(calls) != 3 || filepath.Base(calls[2][0]) != "sudo" {
		t.Fatalf("called %q", calls)
	}
}

// TestInvoke_TheFallShowsTheWordsOfTheExaltedServitor: an exalted verse
// that falls opens the lament with the words its speech uttered, captured
// by the exalted servitor, and every reversion; a proof the master
// dismissed is told as a heresy with the words the intercessor spoke, and
// nothing is touched.
func TestInvoke_TheFallShowsTheWordsOfTheExaltedServitor(t *testing.T) {
	h := newHarness(t)
	s := h.raiseShrine("sudo")
	h.writeRite("hallow", strings.Replace(hallowRite, "echo spoken {{aspect}}", "echo doomed {{aspect}}; exit 3", 1))
	s.sanctify("hallow")
	h.keys("g", "enter", "1")
	h.await()
	h.mustShow("The rite hallow has fallen", "It fell at observance hallowed · incantation", "doomed on", "REVERSIONS",
		"✔ observance hallowed is undone", "✔ observance common is undone", "Every deed is undone")
	if said := h.said(); len(said) != 4 || said[3] != "undo 1" {
		t.Errorf("said %q", said)
	}
	h.keys("esc")

	mustDo(t, os.Remove(filepath.Join(h.dataDir, "said")))
	t.Setenv(spurnVar, "1")
	h.keys("enter", "1")
	h.await()
	h.mustShow("No exalted servitor was raised", "did not vouch", "the proof of the master was refused thrice",
		"Nothing was touched")
	if said := h.said(); said != nil {
		t.Errorf("a dismissed proof touched the machine: %q", said)
	}
	if n := len(s.called()); n != 2 {
		t.Errorf("the intercessor was called upon %d times", n)
	}
}

// TestInvoke_TheCogitatorShowsRefusalsAsHeresies: a rite whose sanction
// lapsed or that stands unsanctioned, an intercessor nowhere to be found,
// a servitor not installed and an exalted servitor of another forging each
// refuse the invocation before any verse is performed, shown as heresies.
// Nothing is touched and nothing panics.
func TestInvoke_TheCogitatorShowsRefusalsAsHeresies(t *testing.T) {
	for _, c := range []struct {
		name    string
		present []string
		mar     func(h *harness, s *shrine)
		calls   int
		words   []string
	}{
		{name: "unsanctioned", present: []string{"sudo"}, mar: func(h *harness, s *shrine) {
			mustDo(h.t, os.Remove(s.store.Path("hallow")))
		}, words: []string{"The pre-flight forbids", "unsanctioned", "servitor sanction hallow"}},
		{name: "lapsed", present: []string{"sudo"}, mar: func(h *harness, s *shrine) {
			h.writeRite("hallow", strings.Replace(hallowRite, "echo spoken", "echo uttered", 1))
		}, words: []string{"The pre-flight forbids", "lapsed", "servitor sanction"}},
		{name: "no intercessor", words: []string{"No exalted servitor was raised", "no intercessor is found"}},
		{name: "not installed", present: []string{"sudo"}, mar: func(h *harness, _ *shrine) {
			me := uint32(os.Getuid())
			rituals.Intercession.Ground = hallowed.Ground{Root: h.t.TempDir(),
				Sovereign: func(uid uint32) bool { return uid == me }}
		}, words: []string{"No exalted servitor was raised", "/usr/bin/servitor", "makepkg -si"}},
		{name: "another forging", present: []string{"sudo"}, mar: func(_ *harness, s *shrine) {
			s.summon = func() error { return errors.New("the exalted servitor ended with 2") }
		}, calls: 1, words: []string{"No exalted servitor was raised", "anew"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness(t)
			s := h.raiseShrine(c.present...)
			h.writeRite("hallow", hallowRite)
			s.sanctify("hallow")
			if c.mar != nil {
				c.mar(h, s)
			}
			h.keys("r", "g", "enter", "1")
			h.await()
			h.mustShow(c.words...)
			if said := h.said(); said != nil {
				t.Errorf("touched: said %q", said)
			}
			if n := len(s.called()); n != c.calls {
				t.Errorf("the intercessor was called upon %d times", n)
			}
			h.keys("esc", "i", "esc", "s", "esc", "space")
			h.await()
		})
	}
}
