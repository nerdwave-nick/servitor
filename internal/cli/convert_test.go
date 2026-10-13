package cli

import (
	"encoding/json"

	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/tui"
)

const markISettingsJSON = "{\n  // the standing orders\n  \"pattern\": \"Mark I\",\n  \"vox\": \"off\"\n}\n"

// markIMouseJSON is the rite of mouseJSON, written in pattern Mark I.
const markIMouseJSON = `{
  "pattern": "Mark I",
  "purpose": "Hide the cursor after inactivity",
  "aspects": ["on", "off"],
  "inscriptions": {
    "reason": {"purpose": "why the rite was invoked"}, // speaks of its purpose
    "example-key": {"mandatory": true, "decrees": {"on": "example-value", "off": "other-value"}},
  },
  "liturgy": [
    {"sanctum": "$TARGET", "scripture": {"on": "cursor {\n    hide-after-inactive-ms 400\n}", "off": ""}},
    {"vox-cast": "success"},
  ],
}`

// converting writes the rite and the settings of the Librarium in pattern
// Mark I, which every ritual reads converted; it returns the scripture of
// the rite and of the settings, as written.
func (e *env) converting() (rite, settings string) {
	e.t.Helper()
	e.addRite("rites/mouse-autohide-toggle.json", markIMouseJSON)
	e.addRite("servitor.json", markISettingsJSON)
	return e.scripture("rites/mouse-autohide-toggle.json"), markISettingsJSON
}

func (e *env) scripture(rel string) string {
	e.t.Helper()
	b, err := os.ReadFile(filepath.Join(e.cfgDir, rel))
	if err != nil {
		e.t.Fatal(err)
	}
	return string(b)
}

// unconverted fails unless the Librarium is as written: no scripture
// converted, no copy kept.
func (e *env) unconverted(rite, settings, after string) {
	e.t.Helper()
	if e.scripture("rites/mouse-autohide-toggle.json") != rite || e.scripture("servitor.json") != settings {
		e.t.Errorf("%s: the Librarium was converted on disk", after)
	}
	for _, dir := range []string{".mark-i", "rites/.mark-i"} {
		if _, err := os.Stat(filepath.Join(e.cfgDir, dir)); err == nil {
			e.t.Errorf("%s: a copy was kept in %s", after, dir)
		}
	}
}

// TestConvert_TheFirstRitualThatMayWriteConverts: the first ritual that
// reads the Librarium and may write converts every scripture of pattern
// Mark I on disk, keeps each original in .mark-i/, and tells each in one
// line upon stderr, even in binharic; a second ritual converts nothing.
// The settings keep their comments.
func TestConvert_TheFirstRitualThatMayWriteConverts(t *testing.T) {
	e := newEnv(t)
	rite, settings := e.converting()
	before := e.targetContent()
	out, errOut, code := e.run("census", "--binharic")
	if code != 0 || !json.Valid([]byte(out)) {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	lines := strings.Split(strings.TrimSpace(errOut), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "⋯ the settings are converted") ||
		!strings.HasPrefix(lines[1], `⋯ the rite "mouse-autohide-toggle" is converted to pattern "Mark II"`) ||
		!strings.Contains(lines[1], filepath.Join(e.cfgDir, "rites", ".mark-i", "mouse-autohide-toggle.json")) {
		t.Errorf("the tidings of the conversion are:\n%s", errOut)
	}
	grimdark(t, errOut)
	if e.scripture("rites/.mark-i/mouse-autohide-toggle.json") != rite || e.scripture(".mark-i/servitor.json") != settings {
		t.Error("the originals are not kept in .mark-i/")
	}
	if got := e.scripture("servitor.json"); got != strings.Replace(settings, `"Mark I"`, `"Mark II"`, 1) {
		t.Errorf("the settings are rewritten as:\n%s", got)
	}
	if !strings.Contains(e.scripture("rites/mouse-autohide-toggle.json"), `"designation": "sanctum"`) {
		t.Error("the rite is not converted on disk")
	}
	if e.targetContent() != before {
		t.Error("the conversion touched a vessel")
	}
	if _, errOut, code := e.run("census"); code != 0 || errOut != "" {
		t.Errorf("a converted Librarium is converted anew: exit %d\n%s", code, errOut)
	}
}

// TestConvert_SilenceMutesTheTidings: --silence mutes the tidings of the
// conversion, which is still written.
func TestConvert_SilenceMutesTheTidings(t *testing.T) {
	e := newEnv(t)
	rite, _ := e.converting()
	_, errOut, _ := e.run("invoke", "mouse-autohide-toggle", "on", "--silence")
	if strings.Contains(errOut, "is converted") {
		t.Errorf("silence did not mute the tidings:\n%s", errOut)
	}
	if e.scripture("rites/.mark-i/mouse-autohide-toggle.json") != rite {
		t.Error("the silenced ritual did not convert")
	}
}

// TestConvert_InMemoryAlone: completion, foresight, the reading rituals
// under sudo and the exalted servitor write nothing: completion and
// foresight read the rites converted in memory, and the scripture stays of
// pattern Mark I until a ritual that may write reads it.
func TestConvert_InMemoryAlone(t *testing.T) {
	e := newEnv(t)
	rite, settings := e.converting()
	out, errOut, code := e.run("__complete", "invoke", "mouse-autohide-toggle", "")
	if code != 0 || !strings.Contains(out, "on") || strings.Contains(errOut, "is converted") {
		t.Errorf("completion: exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	e.unconverted(rite, settings, "completion")
	// only the rite converted in memory is foreseen subject by subject: its
	// sanctum became the subject sanctum-1
	vessel := e.targetContent()
	out, errOut, code = e.run("invoke", "mouse-autohide-toggle", "on", "--foresee")
	if code != 0 || !strings.Contains(out, "\nsubject sanctum-1 · sanctum ") || strings.Contains(out, "verse ") ||
		strings.Contains(errOut, "is converted") || e.targetContent() != vessel {
		t.Errorf("foresight does not read the rite converted in memory alone: exit %d\nstdout: %s\nstderr: %s",
			code, out, errOut)
	}
	e.unconverted(rite, settings, "foresight")
	if out, errOut, code := e.run("__exalt", "mouse-autohide-toggle", "0123456789ab", "on"); code == 0 {
		t.Errorf("the exalted servitor was roused: %s %s", out, errOut)
	}
	e.unconverted(rite, settings, "the exalted servitor")
	keepWorld(t)
	asRoot(t)
	t.Setenv("SUDO_USER", "adept")
	t.Setenv("SUDO_HOME", t.TempDir())
	for _, args := range [][]string{{"census"}, {"augury", "mouse-autohide-toggle"}, {"inquisition"}} {
		out, errOut, code := e.run(args...)
		if code != 0 || strings.Contains(errOut, "is converted") || !strings.Contains(out+errOut, "1 rite") &&
			!strings.Contains(out, "mouse-autohide-toggle") {
			t.Errorf("%v under sudo: exit %d\nstdout: %s\nstderr: %s", args, code, out, errOut)
		}
		e.unconverted(rite, settings, strings.Join(args, " ")+" under sudo")
	}
}

// TestConvert_TheCogitatorTellsByItself: the cogitator converts and tells
// the conversion in a toast of its own; the command line that awakens it
// writes nothing before it.
func TestConvert_TheCogitatorTellsByItself(t *testing.T) {
	e := newEnv(t)
	origRun, origTerm := runTUI, isTerminal
	t.Cleanup(func() { runTUI, isTerminal = origRun, origTerm })
	awakened := 0
	runTUI = func(tui.Options) error { awakened++; return nil }
	isTerminal = func() bool { return true }
	e.mustRun("cogitator")
	rite, settings := e.converting()
	for _, args := range [][]string{{"cogitator"}, nil} {
		if _, errOut, code := e.run(args...); code != 0 || errOut != "" {
			t.Fatalf("%v: exit %d\n%s", args, code, errOut)
		}
		e.unconverted(rite, settings, "the awakening of the cogitator")
	}
	if awakened != 3 {
		t.Errorf("the cogitator was awakened %d times, want 3", awakened)
	}
}
