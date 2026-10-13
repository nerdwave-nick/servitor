package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// smtIIJSON is a state rite of pattern Mark II: a machine-spirit read by its
// scan and a sanctum. $ENACT is the machine-spirit's enact map.
const smtIIJSON = `{"pattern": "Mark II", "purpose": "enable or disable gaming optimization",
  "aspects": ["on", "off"], "vox-cast": ["progress", "success"],
  "subjects": [
    {"name": "smt", "designation": "machine-spirit", "omens": {"on": "off", "*": "on"},
     "auspex": {"scan": "cat $TARGET.smt"}, "enact": $ENACT},
    {"name": "util", "designation": "sanctum", "vessel": "$TARGET", "omens": {"on": "hide", "*": ""}}]}`

func (e *env) addSmt(enact string) {
	e.t.Helper()
	e.addRite("rites/gaming.json", strings.ReplaceAll(smtIIJSON, "$ENACT", enact))
	if err := os.WriteFile(e.target+".smt", []byte("on\n"), 0o644); err != nil {
		e.t.Fatal(err)
	}
}

// TestInvokeII_TellsKeepsAndChroniclesEachSubject: a state rite of pattern
// Mark II is told subject by subject, its progress sent to the desktop as
// "subject x / n"; its data-slate keeps the aspect and its chronicle the
// subjects changed. Invoked again into the same aspect it changes nothing
// and still triumphs.
func TestInvokeII_TellsKeepsAndChroniclesEachSubject(t *testing.T) {
	e := newEnv(t)
	e.addSmt(`{"*": "echo {{omen}} > $TARGET.smt"}`)

	out := e.mustRun("invoke", "gaming", "on")

	got := lines(out)
	if len(got) != 4 || got[0] != "+++ gaming · off → on +++" || got[1] != "  ✔ subject smt · machine-spirit" ||
		!strings.HasPrefix(got[2], "  ✔ subject util · sanctum ") || !strings.HasPrefix(got[3], "✠ ") {
		t.Fatalf("stdout:\n%s", out)
	}
	if b, _ := os.ReadFile(e.target + ".smt"); string(b) != "off\n" || !strings.Contains(e.targetContent(), "hide") {
		t.Fatalf("the subjects were not set: %q\n%s", b, e.targetContent())
	}
	notified := e.missives()
	if len(notified) != 3 || !strings.HasSuffix(notified[0], "\nsubject 1 / 2\n") || !strings.Contains(notified[0], "smt") ||
		!strings.HasSuffix(notified[1], "\nsubject 2 / 2\n") || !strings.Contains(notified[1], "util") ||
		!strings.Contains(notified[2], "-- gaming → on ") {
		t.Fatalf("the desktop heard:\n%s", e.notified())
	}
	slate, _ := os.ReadFile(filepath.Join(e.state, "servitor", "data-slates", "gaming.json"))
	if !regexp.MustCompile(`^\{"aspect":"on","inscriptions":\{\},"last_rite":\{"verdict":"triumph","at":"[^"]+"\}\}\n$`).Match(slate) {
		t.Fatalf("data-slate %s", slate)
	}

	again := e.mustRun("invoke", "gaming", "on")

	if got := lines(again); len(got) != 2 || got[0] != "+++ gaming · on → on +++" || !strings.HasPrefix(got[1], "✠ ") {
		t.Fatalf("the second invocation told:\n%s", again)
	}
	if n := e.missives(); len(n) != 4 || !strings.Contains(n[3], "-- gaming → on ") {
		t.Fatalf("the desktop heard:\n%s", e.notified())
	}
	chronicled := e.chronicleLines()
	line := `^\{"at":"[^"]+","rite":"gaming","aspect":"on","former":"(off|on)","inscriptions":\{\},"verdict":"triumph","changed":%s\}$`
	if len(chronicled) != 2 || !regexp.MustCompile(strings.Replace(line, "%s", `\["smt","util"\]`, 1)).MatchString(chronicled[0]) ||
		!regexp.MustCompile(strings.Replace(line, "%s", `\[\]`, 1)).MatchString(chronicled[1]) {
		t.Fatalf("chronicle:\n%s", strings.Join(chronicled, "\n"))
	}
	if out := e.mustRun("augury", "gaming", "standing"); out != "performed\n" {
		t.Fatalf("standing %q", out)
	}
}

// TestInvokeII_AFallIsLamentedBySubjectAndOmen: a subject that shows another
// omen than the one commanded is lamented by its name and both omens, every
// subject set is restored, and the desktop hears the fall.
func TestInvokeII_AFallIsLamentedBySubjectAndOmen(t *testing.T) {
	e := newEnv(t)
	e.addSmt(`{"off": "echo off-ish > $TARGET.smt", "*": "echo {{omen}} > $TARGET.smt"}`)
	e.addRite("rites/gaming.json", strings.Replace(e.readRite("rites/gaming.json"),
		`{"name": "smt"`, `{"name": "util2", "designation": "transcription", "vessel": "$TARGET.t", "omens": {"on": "x", "off": null}},
		 {"name": "smt"`, 1))

	_, errOut, code := e.run("invoke", "gaming", "on")

	if code != markReverted {
		t.Fatalf("exit %d\nstderr: %s", code, errOut)
	}
	for _, want := range []string{
		`the rite "gaming" fell at subject smt · machine-spirit: the subject "smt" shows the omen "off-ish"; "off" was commanded`,
		"subject smt · machine-spirit is undone", "subject util2 · transcription " + homeward(e.target) + ".t is undone",
		"Every deed is undone",
	} {
		if !strings.Contains(errOut, want) {
			t.Errorf("the lament lacks %q:\n%s", want, errOut)
		}
	}
	grimdark(t, errOut)
	if b, _ := os.ReadFile(e.target + ".smt"); string(b) != "on\n" {
		t.Fatalf("the subject was not restored: %q", b)
	}
	if _, err := os.Stat(e.target + ".t"); err == nil {
		t.Fatal("the transcription was not struck again")
	}
	if n := e.notified(); !strings.Contains(n, "-u critical") || !strings.Contains(n, `"off-ish"; "off" was commanded`) ||
		!strings.Contains(n, "smt") {
		t.Fatalf("the desktop heard:\n%s", n)
	}
	chronicled := e.chronicleLines()
	if len(chronicled) != 1 || !strings.Contains(chronicled[0], `"fell_at":{"subject":"smt"}`) ||
		!strings.Contains(chronicled[0], `"changed":["util2"]`) {
		t.Fatalf("chronicle:\n%s", strings.Join(chronicled, "\n"))
	}
}

// TestInvokeII_ForeseesEachSubjectsPlan: foresight reads every subject and
// tells, for each, the omen read and the omen commanded, whether it changes,
// and what would be spoken or written; nothing is performed.
func TestInvokeII_ForeseesEachSubjectsPlan(t *testing.T) {
	e := newEnv(t)
	e.addSmt(`{"*": "echo {{omen}} > $TARGET.smt"}`)
	e.addRite("rites/unread.json", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "u", "designation": "machine-spirit", "omens": {"on": "up", "*": "down"}, "enact": {"*": "true"}},
	  {"name": "m", "designation": "machine-spirit", "auspex": {"scan": "echo sideways"}, "enact": {"*": "true"}}]}`)

	out := e.mustRun("invoke", "gaming", "on", "--foresee")

	want := []string{
		"+++ Foreseen: the rite gaming, off → on +++",
		`subject smt · machine-spirit: "on" → "off", changing`,
		`  spoken in the tongue bash, with the patience of 30s: bash -c "echo off > ` + e.target + `.smt"`,
		"subject util · sanctum " + homeward(e.target) + ": changing",
	}
	got := lines(out)
	if len(got) < len(want)+1 || got[len(got)-1] != "Nothing was performed." {
		t.Fatalf("foresight:\n%s", out)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("line %d: %q, want %q", i, got[i], w)
		}
	}
	e.mustRun("invoke", "gaming", "on", "-s")
	if out := e.mustRun("invoke", "gaming", "on", "-f"); !strings.Contains(out, `subject smt · machine-spirit: "off" → "off", unchanged`) ||
		!strings.Contains(out, "subject util · sanctum "+homeward(e.target)+": unchanged") {
		t.Fatalf("foresight of an unchanged rite:\n%s", out)
	}
	if out := e.mustRun("invoke", "unread", "on", "-f"); !strings.Contains(out, `subject u · machine-spirit: (no auspex) → "up", changing`) ||
		!strings.Contains(out, `subject m · machine-spirit: (mute) → "on", changing`) {
		t.Fatalf("foresight of unread subjects:\n%s", out)
	}
	if len(e.chronicleLines()) != 1 || len(e.missives()) != 3 {
		t.Fatalf("foresight was chronicled or proclaimed: %q %q", e.chronicleLines(), e.notified())
	}
}

func (e *env) readRite(rel string) string {
	e.t.Helper()
	b, err := os.ReadFile(filepath.Join(e.cfgDir, rel))
	if err != nil {
		e.t.Fatal(err)
	}
	return string(b)
}

// missives are the notify-send calls the desktop heard, in order.
func (e *env) missives() []string {
	heard := e.notified()
	if heard == "" {
		return nil
	}
	return strings.Split(strings.TrimPrefix(heard, "-a servitor "), "-a servitor ")
}
