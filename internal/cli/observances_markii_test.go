package cli

import (
	"os"
	"strings"
	"testing"
)

// dndIIJSON is a do-not-disturb rite of pattern Mark II: a veil spoken
// before its machine-spirit is set, and a herald after it, into the aspect
// on alone. $HERALD is the herald's words.
const dndIIJSON = `{"pattern": "Mark II", "aspects": ["on", "off"], "vox-cast": ["progress", "success"],
  "subjects": [{"name": "mako", "designation": "machine-spirit", "omens": {"on": "do-not-disturb", "*": "default"},
    "auspex": {"scan": "cat $TARGET.mode"}, "enact": {"*": "echo {{omen}} > $TARGET.mode"},
    "introit": ["veil"], "recessional": ["herald-silence"]}],
  "observances": [
    {"name": "veil", "designation": "incantation", "words": "echo veil >> $TARGET.log",
     "reversion": "echo unveil >> $TARGET.log"},
    {"name": "herald-silence", "designation": "incantation", "to": ["on"], "words": "$HERALD",
     "reversion": "echo unsilence >> $TARGET.log"}]}`

func (e *env) addDnd(herald string) {
	e.t.Helper()
	e.addRite("rites/dnd.json", strings.ReplaceAll(dndIIJSON, "$HERALD", herald))
	if err := os.WriteFile(e.target+".mode", []byte("default\n"), 0o644); err != nil {
		e.t.Fatal(err)
	}
}

// TestInvokeII_TellsTheObservancesSpoken: the report tells each observance
// spoken in its place among the subjects, yet the desktop counts subjects
// alone; foresight tells the observances that would be spoken.
func TestInvokeII_TellsTheObservancesSpoken(t *testing.T) {
	e := newEnv(t)
	e.addDnd("echo silence >> $TARGET.log")

	foreseen := lines(e.mustRun("invoke", "dnd", "on", "--foresee"))
	want := []string{
		"+++ Foreseen: the rite dnd, off → on +++",
		"observance veil · incantation",
		`  spoken in the tongue bash, with the patience of 30s: bash -c "echo veil >> ` + e.target + `.log"`,
		"  should the rite fall, its reversion: echo unveil >> " + e.target + ".log",
		`subject mako · machine-spirit: "default" → "do-not-disturb", changing`,
	}
	if len(foreseen) < len(want)+4 || foreseen[len(foreseen)-4] != "observance herald-silence · incantation" {
		t.Fatalf("foresight:\n%s", strings.Join(foreseen, "\n"))
	}
	for i, w := range want {
		if foreseen[i] != w {
			t.Errorf("line %d: %q, want %q", i, foreseen[i], w)
		}
	}

	got := lines(e.mustRun("invoke", "dnd", "on"))

	if len(got) != 5 || got[1] != "  ✔ observance veil · incantation" || got[2] != "  ✔ subject mako · machine-spirit" ||
		got[3] != "  ✔ observance herald-silence · incantation" || !strings.HasPrefix(got[4], "✠ ") {
		t.Fatalf("stdout:\n%s", strings.Join(got, "\n"))
	}
	if b, _ := os.ReadFile(e.target + ".log"); string(b) != "veil\nsilence\n" {
		t.Fatalf("log %q", b)
	}
	if n := e.missives(); len(n) != 2 || !strings.HasSuffix(n[0], "\nsubject 1 / 1\n") {
		t.Fatalf("the desktop heard:\n%s", e.notified())
	}

	if got := lines(e.mustRun("invoke", "dnd", "off")); len(got) != 4 || got[3] == "  ✔ observance herald-silence · incantation" {
		t.Fatalf("the herald guarded by \"to\" was spoken out of on:\n%s", strings.Join(got, "\n"))
	}
}

// TestInvokeII_AFallenObservanceIsLamentedByItsName: an observance that
// falls is lamented by its name; it and everything before it are undone in
// reverse order, and the chronicle and the desktop name it.
func TestInvokeII_AFallenObservanceIsLamentedByItsName(t *testing.T) {
	e := newEnv(t)
	e.addDnd("exit 3")

	_, errOut, code := e.run("invoke", "dnd", "on")

	if code != markReverted {
		t.Fatalf("exit %d\nstderr: %s", code, errOut)
	}
	for _, want := range []string{
		`the rite "dnd" fell at observance herald-silence · incantation: the observance "herald-silence" fell: ` +
			"it ended bearing the death-mark 3",
		"  observance herald-silence · incantation is undone\n  subject mako · machine-spirit is undone\n" +
			"  observance veil · incantation is undone",
		"Every deed is undone",
	} {
		if !strings.Contains(errOut, want) {
			t.Errorf("the lament lacks %q:\n%s", want, errOut)
		}
	}
	grimdark(t, errOut)
	if b, _ := os.ReadFile(e.target + ".log"); string(b) != "veil\nunsilence\nunveil\n" {
		t.Fatalf("log %q", b)
	}
	if b, _ := os.ReadFile(e.target + ".mode"); string(b) != "default\n" {
		t.Fatalf("the subject was not restored: %q", b)
	}
	if n := e.notified(); !strings.Contains(n, "-u critical") || !strings.Contains(n, `observance "herald-silence"`) {
		t.Fatalf("the desktop heard:\n%s", n)
	}
	chronicled := e.chronicleLines()
	if len(chronicled) != 1 || !strings.Contains(chronicled[0], `"fell_at":{"observance":"herald-silence"}`) ||
		!strings.Contains(chronicled[0], `{"observance":"veil","verdict":"triumph"}`) {
		t.Fatalf("chronicle:\n%s", strings.Join(chronicled, "\n"))
	}
}
