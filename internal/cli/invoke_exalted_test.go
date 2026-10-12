package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/exalted"
	"github.com/nerdwave-nick/servitor/internal/hallowed"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// hallowJSON keeps a common sanctum in the target, speaks an exalted
// incantation that tells whose home it hears into $TARGET.exalted (its
// reversion falls while $TARGET.falter stands), and a common verse that
// falls while $TARGET.fall stands.
const hallowJSON = `{
  "pattern": "Mark I",
  "aspects": ["on", "off"],
  "liturgy": [
    {"sanctum": "$TARGET", "scripture": {"on": "split-lock 0", "off": ""}},
    {"incantation": "echo \"{{aspect}} $HOME\" >> $TARGET.exalted",
     "reversion": "test ! -e $TARGET.falter && echo undone >> $TARGET.exalted", "exalted": true},
    {"incantation": "test ! -e $TARGET.fall"},
    {"vox-cast": "success"}
  ]
}`

// lendSpeech lends the exalted servitor of the altar exalted speech, in
// working halls of the trial's own, its tongues judged upon the machine,
// and dominions of the trial's own.
func (a *altar) lendSpeech() {
	me := uint32(os.Getuid())
	a.cfg.Voice = exalted.Speech{Halls: filepath.Join(a.t.TempDir(), "run", "servitor"),
		Sovereign: func(uid uint32) bool { return uid == me }, Tongues: hallowed.Machine,
		Getenv: func(string) (string, bool) { return "", false }}.Voice
	a.cfg.Dominions = filepath.Join(a.t.TempDir(), "dominions")
}

// sanctified raises the altar with sudo, lends it speech, lays the hallow
// rite and sanctions it as the master would.
func (e *env) sanctified() (*altar, string) {
	e.t.Helper()
	e.addRite("rites/hallow.json", hallowJSON)
	a := e.raiseAltar("sudo")
	a.lendSpeech()
	e.mustRun("sanction", "hallow")
	kept, err := a.store.Read("hallow")
	if err != nil || kept == nil {
		e.t.Fatalf("no sanction was laid: %v", err)
	}
	sigil, _ := kept.SigilOf("on")
	return a, sanction.Short(sigil)
}

// exaltedSaid is what the exalted verse said into $TARGET.exalted.
func (e *env) exaltedSaid() string {
	b, _ := os.ReadFile(e.target + ".exalted")
	return string(b)
}

// at is the hour of a chronicle line, which every trial reads past.
var at = regexp.MustCompile(`^\{"at":"\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ",`)

// TestInvoke_PerformsExaltedVersesWithOneProof: a sanctioned rite is invoked
// with one proof of the master, its exalted verse spoken by the exalted
// servitor with the root's home, its common verses by the invoker; the
// chronicle line names the intercessor and the sigil, and --binharic prints
// that very line. Upon a terminal the tidings that the proof is asked come
// first, upon stderr in binharic, and silence mutes them.
func TestInvoke_PerformsExaltedVersesWithOneProof(t *testing.T) {
	e := newEnv(t)
	a, sigil := e.sanctified()
	e.terminal(120)

	out, errOut, code := e.run("invoke", "hallow", "on")
	if code != 0 || errOut != "" {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	if got := e.exaltedSaid(); got != "on /root\n" {
		t.Errorf("the exalted verse said %q", got)
	}
	if !strings.Contains(e.targetContent(), "split-lock 0") {
		t.Errorf("the common verse was not performed:\n%s", e.targetContent())
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 || !strings.HasPrefix(lines[0], "⋯ ") || !strings.Contains(lines[0], "sudo") ||
		!strings.Contains(lines[0], sigil) || !strings.Contains(lines[0], "hallow") ||
		!strings.HasPrefix(lines[1], "+++ hallow") {
		t.Errorf("the report:\n%s", out)
	}
	grimdark(t, out)
	if calls := a.called(); len(calls) != 2 || calls[1][len(calls[1])-3] != "hallow" ||
		calls[1][len(calls[1])-2] != sigil || calls[1][len(calls[1])-1] != "on" {
		t.Errorf("the intercessors were called upon as %q", calls)
	}
	want := `"rite":"hallow","aspect":"on","inscriptions":{},"verdict":"triumph","intercessor":"sudo","sigil":"` +
		sigil + `"}`
	if line := e.lastChronicled(); !at.MatchString(line) || at.ReplaceAllString(line, "") != want {
		t.Errorf("chronicled %s\nwant {\"at\":…,%s", line, want)
	}

	offSigil := a.sigil("hallow", "off")
	out, errOut, code = e.run("invoke", "hallow", "off", "--binharic")
	if code != 0 || out != e.lastChronicled()+"\n" || !strings.HasPrefix(errOut, "⋯ ") ||
		!strings.Contains(strings.Split(errOut, "\n")[0], offSigil) {
		t.Errorf("binharic: exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	want = `"rite":"hallow","aspect":"off","former":"on","inscriptions":{},"verdict":"triumph","intercessor":"sudo",` +
		`"sigil":"` + offSigil + "\"}\n"
	if !at.MatchString(out) || at.ReplaceAllString(out, "") != want {
		t.Errorf("binharic printed %s\nwant {\"at\":…,%s", out, want)
	}

	out, _, code = e.run("invoke", "hallow", "on", "--silence")
	if code != 0 || strings.Contains(out, "⋯") {
		t.Errorf("silence: exit %d\n%s", code, out)
	}
	e.terminal(-1)
	out, _, code = e.run("invoke", "hallow", "off")
	if code != 0 || strings.Contains(out, "⋯") || strings.Contains(e.notified(), sigil) {
		t.Errorf("without a terminal: exit %d\n%s\nnotified: %s", code, out, e.notified())
	}
}

// sigil is the first twelve letters of the sigil of the plan of aspect of
// the rite, as sanctioned.
func (a *altar) sigil(rite, aspect string) string {
	a.t.Helper()
	kept, err := a.store.Read(rite)
	if err != nil || kept == nil {
		a.t.Fatalf("no sanction of %s: %v", rite, err)
	}
	sigil, _ := kept.SigilOf(aspect)
	return sanction.Short(sigil)
}

// TestInvoke_TheDeathMarksOfExaltedVerses: an exalted rite bears the
// death-marks of every rite: 1 when a verse falls and every deed is undone
// (the exalted verse by the exalted servitor, with no second proof), 3 when
// its reversion falls too, 2 when the proof is dismissed — and then no
// chronicle line is written and nothing is touched.
func TestInvoke_TheDeathMarksOfExaltedVerses(t *testing.T) {
	e := newEnv(t)
	a, sigil := e.sanctified()

	touch(t, e.target+".fall")
	out, errOut, code := e.run("invoke", "hallow", "on", "--binharic")
	if code != 1 || !strings.Contains(out, `"verdict":"reverted"`) || !strings.Contains(out, `"sigil":"`+sigil+`"`) ||
		e.exaltedSaid() != "on /root\nundone\n" || strings.Contains(e.targetContent(), "split-lock") ||
		len(a.called()) != 2 {
		t.Errorf("fallen: exit %d, said %q, %d calls\nstdout: %s\nstderr: %s", code, e.exaltedSaid(), len(a.called()),
			out, errOut)
	}

	touch(t, e.target+".falter")
	out, errOut, code = e.run("invoke", "hallow", "on", "--binharic")
	if code != 3 || !strings.Contains(out, `"reversions":[{"verse":2,"verdict":"faltered"},{"verse":1,"verdict":"triumph"}]`) ||
		len(a.called()) != 3 {
		t.Errorf("faltered: exit %d, %d calls\nstdout: %s\nstderr: %s", code, len(a.called()), out, errOut)
	}
	_ = os.Remove(e.target + ".fall")
	_ = os.Remove(e.target + ".falter")

	chronicled, said := len(e.chronicleLines()), e.exaltedSaid()
	a.vouch = false
	out, errOut, code = e.run("invoke", "hallow", "on", "--binharic")
	if code != 2 || out != "" || !strings.Contains(errOut, "did not vouch") || len(a.called()) != 4 ||
		len(e.chronicleLines()) != chronicled || e.exaltedSaid() != said || e.targetContent() != "input {}\n" {
		t.Errorf("dismissed: exit %d, %d calls\nstdout: %s\nstderr: %s", code, len(a.called()), out, errOut)
	}
	grimdark(t, errOut)
}

// TestInvoke_ForeseesExaltedVerses: foresight marks the exalted verses ✠,
// tells their sanction standing and never raises the exalted servitor.
func TestInvoke_ForeseesExaltedVerses(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/hallow.json", hallowJSON)
	a := e.raiseAltar("sudo")
	out := e.mustRun("invoke", "hallow", "on", "--foresee")
	if !strings.Contains(out, "verse 2 · incantation") || !strings.Contains(out, ".exalted ✠\n") ||
		strings.Contains(out, "verse 3 · incantation test ! -e "+e.target+".fall ✠") ||
		!strings.Contains(out, "stand unsanctioned") || !strings.Contains(out, "servitor sanction hallow") {
		t.Errorf("foresight:\n%s", out)
	}
	e.mustRun("sanction", "hallow")
	out = e.mustRun("invoke", "hallow", "on", "--foresee")
	if !strings.Contains(out, "stand sanctioned under the sigil") {
		t.Errorf("foresight:\n%s", out)
	}
	grimdark(t, out)
	if len(a.called()) != 1 || e.exaltedSaid() != "" {
		t.Error("foresight raised the exalted servitor")
	}
}
