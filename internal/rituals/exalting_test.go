package rituals

import (
	"context"
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

	"golang.org/x/sys/unix"

	"github.com/nerdwave-nick/servitor/internal/exalted"
	"github.com/nerdwave-nick/servitor/internal/hallowed"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// hallow is a rite whose exalted verses stand among common ones: a common
// incantation, an exalted incantation that tells who speaks it and where,
// an exalted transcription of a vessel from the root, and a common
// incantation that falls when $DATA/fall stands. Every verse reverts by
// word.
const hallow = `{
  "pattern": "Mark I",
  "aspects": ["on", "off"],
  "liturgy": [
    {"incantation": "echo 1 >> $DATA/said", "reversion": "echo undo 1 >> $DATA/said"},
    {"incantation": "echo \"exalted {{aspect}} $PWD $HOME $SERVITOR_FORMER_ASPECT\" >> $DATA/said",
     "reversion": "echo \"undo exalted $PWD\" >> $DATA/said", "exalted": true},
    {"transcription": "/etc/servitor-trials-hallow.conf", "scripture": "hallow = {{aspect}}", "exalted": true},
    {"vox-cast": "progress"},
    {"incantation": "test ! -e $DATA/fall && echo 5 >> $DATA/said", "reversion": "echo undo 5 >> $DATA/said"}
  ]
}`

// vesselOfHallow is where the exalted transcription of hallow writes.
const vesselOfHallow = "/etc/servitor-trials-hallow.conf"

// temple is the world of a trial of exalted invocations: a Librarium and a
// store of sanctions of its own, stub intercessors in a hall of its own
// that record their words and vouch for the master or not, and an exalted
// servitor raised within the trials' own process over a true socketpair,
// its authority and the root stood in for by the invoker of the trials,
// acting upon a ground of its own. No intercessor of the machine is ever
// called upon.
type temple struct {
	*fixture
	store   sanction.Store
	ground  string // the root of the exalted servitor's world
	bin     string // the stub intercessors
	cfg     exalted.Config
	journal *ledger
	spurn   bool // the stub intercessor does not vouch for the master

	mu    sync.Mutex
	calls [][]string
	marks []int // the death-marks of the exalted servitors raised
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

// record is where a stub intercessor records its words.
const record = "SERVITOR_TRIAL_RECORD"

// stubIntercessor records its words, one per NUL, and vouches for the
// master unless SERVITOR_TRIAL_SPURN is set.
const stubIntercessor = "#!/bin/sh\nprintf '%s\\0' \"$0\" \"$@\" > \"$" + record + "\"\n" +
	"[ -z \"$SERVITOR_TRIAL_SPURN\" ] || exit 1\n"

// newTemple raises the world of a trial with the intercessors present.
func newTemple(t *testing.T, present ...string) *temple {
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
	tp := &temple{fixture: newFixture(t), ground: t.TempDir(), bin: t.TempDir(), journal: &ledger{}}
	tp.store = sanction.Store{Root: t.TempDir(), UID: os.Getuid()}
	for _, name := range present {
		tp.write(tp.bin, name, stubIntercessor)
		must(t, os.Chmod(filepath.Join(tp.bin, name), 0o755))
	}
	w, err := enshrine(tp.ground)
	must(t, err)
	must(t, os.MkdirAll(filepath.Join(tp.ground, "etc"), 0o755))
	w.Attended = func() bool { return true }
	w.LookPath = func(name string) (string, error) {
		p := filepath.Join(tp.bin, name)
		if _, err := os.Stat(p); err != nil {
			return "", exec.ErrNotFound
		}
		return p, nil
	}
	formerWorld, formerSummon := Intercession, Summon
	t.Cleanup(func() { Intercession, Summon = formerWorld, formerSummon })
	Intercession, Summon = w, tp.summon

	self := filepath.Join(t.TempDir(), "exe")
	must(t, os.Symlink(filepath.Join(tp.ground, w.Servitor), self))
	me := uint32(os.Getuid())
	mine := func(uid uint32) bool { return uid == me }
	getenv := func(k string) (string, bool) {
		v, ok := map[string]string{"SUDO_UID": strconv.Itoa(int(me)), "LANG": "C.UTF-8"}[k]
		return v, ok
	}
	tp.cfg = exalted.Config{
		Exalted:   func() bool { return true },
		Getenv:    getenv,
		Store:     tp.store.Root,
		Sovereign: mine,
		Ground:    hallowed.Ground{Root: tp.ground, Sovereign: mine, Lord: hallowed.Owner{UID: me, GID: uint32(os.Getgid())}},
		Self:      self,
		Dominions: filepath.Join(t.TempDir(), "dominions"),
		Journal:   tp.journal,
		Lament:    io.Discard,
		Voice: exalted.Speech{Halls: filepath.Join(t.TempDir(), "run", "servitor"), Sovereign: mine,
			Tongues: hallowed.Machine, Getenv: getenv}.Voice,
	}
	return tp
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// coded is how an intercessor ended, as the machine tells it.
type coded int

func (c coded) Error() string { return fmt.Sprintf("the intercessor ended with %d", int(c)) }
func (c coded) ExitCode() int { return int(c) }

// summon calls upon the stub intercessor as the machine would, never
// lending it the channel; when it vouches, the exalted servitor it would
// have raised is raised within the trials upon the channel, roused by the
// words that follow the servitor in the intercessor's, and the intercessor
// ends as it ends.
func (tp *temple) summon(cmd *exec.Cmd, end *os.File) (func() error, error) {
	if filepath.Dir(cmd.Path) != tp.bin {
		tp.t.Errorf("the trials would call upon %s", cmd.Path)
		return nil, errors.New("no intercessor of the machine")
	}
	rec := filepath.Join(tp.t.TempDir(), "words")
	stub := exec.Command(cmd.Path, cmd.Args[1:]...)
	stub.Env = append(os.Environ(), record+"="+rec)
	if tp.spurn {
		stub.Env = append(stub.Env, "SERVITOR_TRIAL_SPURN=1")
	}
	ran := stub.Run()
	words, err := os.ReadFile(rec)
	if err != nil {
		tp.t.Fatalf("the stub recorded nothing: %v", err)
	}
	called := strings.Split(strings.TrimSuffix(string(words), "\x00"), "\x00")
	tp.mu.Lock()
	tp.calls = append(tp.calls, called)
	tp.mu.Unlock()
	if ran != nil {
		return func() error { return ran }, nil
	}
	i := slices.Index(called, exalted.Mode)
	if i < 1 || called[i-1] != "/usr/bin/servitor" {
		tp.t.Fatalf("the intercessor raises no exalted servitor: %q", called)
	}
	fd, err := unix.Dup(int(end.Fd()))
	if err != nil {
		return nil, err
	}
	conn := os.NewFile(uintptr(fd), "exalted servitor")
	done := make(chan int, 1)
	go func() {
		defer func() { _ = conn.Close() }()
		done <- exalted.Serve(called[i+1:], conn, conn, tp.cfg)
	}()
	return func() error {
		m := <-done
		tp.mu.Lock()
		tp.marks = append(tp.marks, m)
		tp.mu.Unlock()
		if m != 0 {
			return coded(m)
		}
		return nil
	}, nil
}

// teller is a herald that keeps every proclamation whole.
type teller struct{ told []invocation.Proclamation }

func (h *teller) Proclaim(p invocation.Proclamation) { h.told = append(h.told, p) }

// called are the words of every intercessor called upon, its path first.
func (tp *temple) called() [][]string {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	return slices.Clone(tp.calls)
}

// sanctify lays the sanction of the rite name as it stands now.
func (tp *temple) sanctify(name string) *sanction.Sanction {
	tp.t.Helper()
	s := tp.servitor()
	u, err := s.Unit(s.Librarium.Rites[name])
	must(tp.t, err)
	kept := sanction.New(u, sanctionedAt)
	data, err := json.Marshal(kept)
	must(tp.t, err)
	p := tp.store.Path(name)
	must(tp.t, os.MkdirAll(filepath.Dir(p), 0o750))
	must(tp.t, os.WriteFile(p, data, 0o640))
	return kept
}

// servitor opens the Librarium upon the temple's store.
func (tp *temple) servitor() *Servitor {
	s := tp.fixture.servitor()
	s.Sanctions = &tp.store
	return s
}

// invoke invokes the rite into aspect as the command line does.
func (tp *temple) invoke(rite, aspect string, h *teller) (Result, error) {
	p := Petition{Rite: rite, Aspect: aspect, Exaltation: &Exaltation{}}
	if h != nil {
		p.Herald = h
	}
	return tp.servitor().Invoke(context.Background(), p)
}

// said is what the verses of hallow said, line by line.
func (tp *temple) said() []string {
	p := filepath.Join(tp.data, "said")
	if !exists(p) {
		return nil
	}
	return strings.Split(strings.TrimSpace(tp.read(p)), "\n")
}

// vessel is what the exalted vessel of hallow holds in the ground, with its
// seal; "absent" when none stands.
func (tp *temple) vessel() string {
	p := filepath.Join(tp.ground, vesselOfHallow)
	fi, err := os.Stat(p)
	if err != nil {
		return "absent"
	}
	return fmt.Sprintf("%04o %s", fi.Mode().Perm(), tp.read(p))
}

// TestInvoke_TheExaltedServitorPerformsTheExaltedVerses: the exalted verses
// of a sanctioned rite are performed by the exalted servitor alone, raised
// with one proof of the master after the pre-flight, in written order among
// the common verses: its speech hears the root's home in a working hall of
// its own, never the invoker's, and its vessel lies in its own world. The
// herald hears that the proof is asked first; the chronicle names the
// intercessor and the sigil.
func TestInvoke_TheExaltedServitorPerformsTheExaltedVerses(t *testing.T) {
	tp := newTemple(t, "sudo", "run0", "pkexec")
	tp.rite("hallow", hallow)
	kept := tp.sanctify("hallow")
	sigil, _ := kept.SigilOf("on")

	h := &teller{}
	res, err := tp.invoke("hallow", "on", h)
	if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Triumph {
		t.Fatalf("%v %+v", err, res.Outcome)
	}
	said := tp.said()
	if len(said) != 3 || said[0] != "1" || !strings.HasPrefix(said[1], "exalted on /") ||
		!strings.Contains(said[1], " /root") || said[2] != "5" || strings.Contains(said[1], tp.lib) {
		t.Errorf("said %q", said)
	}
	if got := tp.vessel(); got != "0644 hallow = on" {
		t.Errorf("the exalted vessel holds %q", got)
	}
	if exists(vesselOfHallow) {
		t.Errorf("the machine's own %s was written", vesselOfHallow)
	}
	want := []string{filepath.Join(tp.bin, "sudo"), "-k", "-p", "servitor: authenticate to perform hallow (sigil " +
		sanction.Short(sigil) + "): ", "--", "/usr/bin/servitor", exalted.Mode, "hallow", sanction.Short(sigil), "on"}
	if calls := tp.called(); len(calls) != 1 || !slices.Equal(calls[0], want) {
		t.Errorf("the intercessors were called upon as %q, want once as %q", calls, want)
	}
	if got := tp.journal.deeds(); !slices.Equal(got, []string{"accept accepted", "perform 2 triumph",
		"perform 3 triumph", "finish finished"}) {
		t.Errorf("journaled %q", got)
	}
	if len(h.told) == 0 || h.told[0].Tidings != invocation.Exaltation || h.told[0].Intercessor != "sudo" ||
		h.told[0].Sigil != sanction.Short(sigil) || h.told[0].Values.Rite != "hallow" {
		t.Errorf("the herald heard %+v", h.told)
	}
	entries := tp.chronicled()
	if len(entries) != 1 || entries[0].Intercessor != "sudo" || entries[0].Sigil != sanction.Short(sigil) {
		t.Errorf("chronicled %+v", entries)
	}
	if res.Sanction != sanction.Sanctioned || res.Sigil != sigil || res.Intercessor != "sudo" {
		t.Errorf("the result tells %q %q %q", res.Sanction, res.Sigil, res.Intercessor)
	}

	res, err = tp.invoke("hallow", "off", nil)
	if err != nil || res.Outcome.Verdict != invocation.Triumph || tp.vessel() != "0644 hallow = off" ||
		len(tp.called()) != 2 || !strings.HasSuffix(tp.said()[4], " on") {
		t.Errorf("off: %v %+v, vessel %q, said %q", err, res.Outcome, tp.vessel(), tp.said())
	}
}

// TestInvoke_AFallRevertsBothKindsWithOneProof: when a common verse falls
// after the exalted ones, the exalted servitor reverts its verses among the
// common reversions, from the fallen verse back to the first, and no second
// proof is asked; so too when an exalted verse falls.
func TestInvoke_AFallRevertsBothKindsWithOneProof(t *testing.T) {
	tp := newTemple(t, "sudo")
	tp.rite("hallow", hallow)
	tp.sanctify("hallow")
	tp.write(tp.data, "fall", "")

	res, err := tp.invoke("hallow", "on", nil)
	if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Reverted {
		t.Fatalf("%v %+v", err, res.Outcome)
	}
	said := tp.said()
	if len(said) != 5 || said[0] != "1" || said[2] != "undo 5" || !strings.HasPrefix(said[3], "undo exalted /") ||
		said[4] != "undo 1" {
		t.Errorf("said %q", said)
	}
	if got := tp.vessel(); got != "absent" {
		t.Errorf("the exalted vessel holds %q", got)
	}
	if n := len(tp.called()); n != 1 {
		t.Errorf("the intercessor was called upon %d times", n)
	}
	if got := tp.journal.deeds(); !slices.Equal(got, []string{"accept accepted", "perform 2 triumph",
		"perform 3 triumph", "revert 3 reverted", "revert 2 reverted", "finish finished"}) {
		t.Errorf("journaled %q", got)
	}
	if entries := tp.chronicled(); len(entries) != 1 || entries[0].Verdict != invocation.Reverted ||
		entries[0].Intercessor != "sudo" {
		t.Errorf("chronicled %+v", entries)
	}

	// the exalted vessel cannot be written: the exalted verse falls
	must(t, os.Chmod(filepath.Join(tp.ground, "etc"), 0o775))
	must(t, os.Remove(filepath.Join(tp.data, "fall")))
	res, err = tp.invoke("hallow", "on", nil)
	if err != nil || res.Outcome.Verdict != invocation.Reverted || res.Outcome.Fell.Number != 3 ||
		!strings.Contains(res.Outcome.Fell.Heresy.Error(), "the exalted servitor") || tp.vessel() != "absent" {
		t.Errorf("%v %+v, vessel %q", err, res.Outcome, tp.vessel())
	}
	if said := tp.said(); len(said) != 9 || !strings.HasPrefix(said[7], "undo exalted /") || said[8] != "undo 1" {
		t.Errorf("said %q", said)
	}
}

// TestInvoke_RefusalsTouchNothing: an exalted rite unsanctioned or lapsed,
// lent no intercessor, bearing a rune for an inscription
// spliced into its exalted words or a word exalted speech may not hear, or
// whose dominion cannot be held, is refused before any proof is asked; a
// proof the master dismissed, an intercessor nowhere to be found, and an
// exalted servitor of another forging or one that refuses are refused too.
// Each touches nothing: no verse, no vessel, no data-slate, no chronicle.
func TestInvoke_RefusalsTouchNothing(t *testing.T) {
	spliced := strings.Replace(hallow, `"aspects": ["on", "off"],`, `"aspects": ["on", "off"],
  "inscriptions": {"zone": {"decrees": {"*": "core"}}, "who": {}},`, 1)
	spliced = strings.Replace(spliced, `"hallow = {{aspect}}"`, `"hallow = {{aspect}} {{inscription.zone}}"`, 1)
	for _, c := range []struct {
		name    string
		present []string
		mar     func(tp *temple)
		p       func(p *Petition)
		calls   int
		want    func(err error) bool
		words   string
	}{
		{name: "unsanctioned", present: []string{"sudo"}, mar: func(tp *temple) { _ = os.Remove(tp.store.Path("hallow")) },
			words: "servitor sanction hallow"},
		{name: "lapsed", present: []string{"sudo"}, mar: func(tp *temple) {
			tp.rite("hallow", strings.Replace(hallow, "hallow = {{aspect}}", "hallow anew = {{aspect}}", 1))
		}, words: "servitor sanction hallow"},
		{name: "lent no intercessor", present: []string{"sudo"}, p: func(p *Petition) { p.Exaltation = nil },
			words: "servitor invoke hallow on"},
		{name: "a rune of a spliced inscription", present: []string{"sudo"},
			mar: func(tp *temple) { tp.rite("hallow", spliced); tp.sanctify("hallow") },
			p:   func(p *Petition) { p.Runes = map[string]string{"zone": "core"} }, words: "--zone"},
		{name: "a word exalted speech may not hear", present: []string{"sudo"},
			mar: func(tp *temple) { tp.rite("hallow", spliced); tp.sanctify("hallow") },
			p:   func(p *Petition) { p.Runes = map[string]string{"who": "x\necho INJECTED"} }, words: "newline"},
		{name: "no dominion", present: []string{"sudo"}, mar: func(tp *temple) {
			must(tp.t, os.Chmod(tp.state, 0o500))
			tp.t.Cleanup(func() { _ = os.Chmod(tp.state, 0o755) })
		}, words: "no dominion could be held"},
		{name: "no intercessor", words: "no intercessor is found"},
		{name: "a dismissed proof", present: []string{"sudo"}, mar: func(tp *temple) { tp.spurn = true }, calls: 1,
			want: func(err error) bool { return errors.As(err, new(*Dismissed)) }, words: "did not vouch"},
		{name: "another forging", present: []string{"sudo"}, mar: func(tp *temple) {
			Summon = func(cmd *exec.Cmd, end *os.File) (func() error, error) {
				tp.calls = append(tp.calls, cmd.Args)
				_, _ = fmt.Fprintf(end, `{"to":"handshake","verdict":"refused","protocol":99}`+"\n")
				return func() error { return coded(2) }, nil
			}
		}, calls: 1, want: func(err error) bool { return errors.As(err, new(*exalted.Mismatch)) }, words: "anew"},
		{name: "a sanction the exalted servitor refuses", present: []string{"sudo"},
			mar: func(tp *temple) { tp.cfg.Store = tp.t.TempDir() }, calls: 1,
			want: func(err error) bool { return errors.As(err, new(*Spurned)) }, words: "unsanctioned"},
	} {
		t.Run(c.name, func(t *testing.T) {
			tp := newTemple(t, c.present...)
			tp.rite("hallow", hallow)
			tp.sanctify("hallow")
			if c.mar != nil {
				c.mar(tp)
			}
			p := Petition{Rite: "hallow", Aspect: "on", Exaltation: &Exaltation{}, Herald: &herald{}}
			if c.p != nil {
				c.p(&p)
			}
			res, err := tp.servitor().Invoke(context.Background(), p)
			if err == nil || res.Outcome != nil || res.Entry != nil || !strings.Contains(err.Error(), c.words) ||
				(c.want != nil && !c.want(err)) {
				t.Fatalf("err %v\nresult %+v", err, res)
			}
			grimdark(t, err.Error())
			if n := len(tp.called()); n != c.calls {
				t.Errorf("the intercessor was called upon %d times", n)
			}
			if said := tp.said(); said != nil || tp.vessel() != "absent" {
				t.Errorf("touched: said %q, vessel %q", said, tp.vessel())
			}
			for _, p := range []string{filepath.Join(tp.state, "servitor", "chronicle.jsonl"),
				filepath.Join(tp.state, "servitor", "data-slates")} {
				if exists(p) {
					t.Errorf("%s was written", p)
				}
			}
		})
	}
}

// TestInvoke_AnExaltedEdictIsProclaimed: an edict's exalted verse is
// performed by the exalted servitor, roused with no aspect.
func TestInvoke_AnExaltedEdictIsProclaimed(t *testing.T) {
	tp := newTemple(t, "pkexec")
	tp.rite("reboot", reboot)
	kept := tp.sanctify("reboot")
	res, err := tp.invoke("reboot", "", nil)
	if err != nil || res.Outcome.Verdict != invocation.Triumph {
		t.Fatalf("%v %+v", err, res.Outcome)
	}
	if said := tp.said(); !slices.Equal(said, []string{"exalted"}) {
		t.Errorf("said %q", said)
	}
	want := []string{filepath.Join(tp.bin, "pkexec"), "/usr/bin/servitor", exalted.Mode, "reboot",
		sanction.Short(kept.Sigil)}
	if calls := tp.called(); len(calls) != 1 || !slices.Equal(calls[0], want) {
		t.Errorf("called %q, want %q", calls, want)
	}
}

// TestInvoke_ForesightTellsTheSanction: foresight tells how the plan of the
// aspect stands towards its sanction, marks exalted verses, and never
// raises the exalted servitor.
func TestInvoke_ForesightTellsTheSanction(t *testing.T) {
	tp := newTemple(t, "sudo")
	tp.rite("hallow", hallow)
	for _, st := range []sanction.Standing{sanction.Unsanctioned, sanction.Sanctioned, sanction.Lapsed} {
		switch st {
		case sanction.Sanctioned:
			tp.sanctify("hallow")
		case sanction.Lapsed:
			tp.rite("hallow", strings.Replace(hallow, "hallow = {{aspect}}", "hallow anew = {{aspect}}", 1))
		}
		res, err := tp.servitor().Invoke(context.Background(), Petition{Rite: "hallow", Aspect: "on", Foresee: true,
			Exaltation: &Exaltation{}})
		if err != nil || res.Sanction != st || len(res.Foresight) != 5 || !res.Foresight[1].Exalted ||
			res.Foresight[0].Exalted {
			t.Errorf("%s: %v %+v", st, err, res)
		}
	}
	if len(tp.called()) != 0 || tp.said() != nil {
		t.Error("foresight raised the exalted servitor")
	}
}

// TestInvoke_TellsWhenTheExaltedServitorAnswers: an Exaltation that would
// know when the intercessor asks the master no more (the cogitator, which
// lends sudo its terminal) is told so once: when the exalted servitor first
// answered, before any verse is performed, or when the channel ended
// unanswered because the proof was dismissed. With no intercessor called
// upon it is never told.
func TestInvoke_TellsWhenTheExaltedServitorAnswers(t *testing.T) {
	for _, c := range []struct {
		name    string
		present []string
		spurn   bool
		told    int
	}{
		{name: "vouched", present: []string{"sudo"}, told: 1},
		{name: "dismissed", present: []string{"sudo"}, spurn: true, told: 1},
		{name: "no intercessor"},
	} {
		t.Run(c.name, func(t *testing.T) {
			tp := newTemple(t, c.present...)
			tp.rite("hallow", hallow)
			tp.sanctify("hallow")
			tp.spurn = c.spurn
			told := 0
			var saidWhenTold []string
			x := &Exaltation{Answered: func() { told++; saidWhenTold = tp.said() }}
			res, err := tp.servitor().Invoke(context.Background(), Petition{Rite: "hallow", Aspect: "on",
				Exaltation: x})
			if c.name == "vouched" && (err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Triumph) {
				t.Fatalf("%v %+v", err, res.Outcome)
			}
			if told != c.told || saidWhenTold != nil {
				t.Errorf("told %d times, the verses having said %q", told, saidWhenTold)
			}
		})
	}
}
