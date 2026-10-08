package cli

import (
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/lexicon"
)

func grimEnv(t *testing.T) *env {
	t.Helper()
	e := newEnv(t)
	e.grim = true
	return e
}

func TestGrimdark_IsTheDefaultVocabulary(t *testing.T) {
	e := grimEnv(t)
	help := e.mustRun("--help")
	for _, want := range []string{
		"+++ SERVITOR", "Sanctioned Rituals:", "Runes:", "Exempla:",
		"invoke", "augury", "census", "inquisition", "cogitator", "--no-grimdark",
		"reveal the lore of servitor",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("grimdark help missing %q", want)
		}
	}
	if sub := e.mustRun("census", "--help"); !strings.Contains(sub, "Universal Runes:") || !strings.Contains(sub, "Also Known As:") {
		t.Errorf("subcommand help not flavored:\n%s", sub)
	}
	if out := e.mustRun("--version"); !strings.Contains(out, "blessed be the Omnissiah") {
		t.Fatalf("version: %q", out)
	}
}

func TestGrimdark_PlainModeViaFlagAndEnv(t *testing.T) {
	e := grimEnv(t)
	plainHelp := e.mustRun("--no-grimdark", "--help")
	if strings.Contains(plainHelp, "Sanctioned Rituals") || !strings.Contains(plainHelp, "Available Commands:") {
		t.Fatal("--no-grimdark did not select plain vocabulary")
	}
	t.Setenv(lexicon.EnvNoGrimdark, "1")
	if got := e.mustRun("--help"); got != plainHelp {
		t.Fatal("SERVITOR_NO_GRIMDARK=1 should match --no-grimdark")
	}
	if got := e.mustRun("--no-grimdark=false", "--help"); !strings.Contains(got, "Sanctioned Rituals") {
		t.Fatal("--no-grimdark=false must override the environment")
	}
}

func TestGrimdark_CommandNamesWorkInBothModes(t *testing.T) {
	e := grimEnv(t)
	out := e.mustRun("invoke", "mouse-autohide-toggle", "on", "--reason", "for the Emperor")
	if !strings.Contains(out, "+++ Rite mouse-autohide-toggle performed: (dormant) → on +++") ||
		!strings.Contains(out, "sanctified") || !strings.Contains(out, "The Omnissiah is pleased.") {
		t.Fatalf("grimdark summary:\n%s", out)
	}
	e.mustRun("switch", "mouse-autohide-toggle", "off", "-q")
	if got := e.mustRun("augury", "mouse-autohide-toggle", "state"); got != "off\n" {
		t.Fatalf("augury = %q", got)
	}
	e.grim = false
	if got := e.mustRun("augury", "mouse-autohide-toggle", "state"); got != "off\n" {
		t.Fatalf("grimdark name in plain mode = %q", got)
	}
	if got := e.mustRun("census"); !strings.Contains(got, "SWITCH") {
		t.Fatalf("census alias in plain mode:\n%s", got)
	}
}

func TestGrimdark_Messages(t *testing.T) {
	e := grimEnv(t)
	if got := e.mustRun("census"); !strings.Contains(got, "RITE") || !strings.Contains(got, "(dormant)") {
		t.Fatalf("census:\n%s", got)
	}
	_, errOut, code := e.run("invoke", "mouse-autohide-toggle", "maybe")
	if code == 0 || !strings.Contains(errOut, `servitor ✠ the rite "mouse-autohide-toggle" knows no aspect "maybe"`) {
		t.Fatalf("unknown aspect: %q", errOut)
	}
	_, errOut, _ = e.run("augury", "mouse-autohide-toggle")
	if !strings.Contains(errOut, "lies dormant") {
		t.Fatalf("dormant augury: %q", errOut)
	}
	_, errOut, _ = e.run("invoke", "mouse-autohide", "on")
	if !strings.Contains(errOut, "no rite named") || !strings.Contains(errOut, "Perhaps you sought:") {
		t.Fatalf("unknown rite: %q", errOut)
	}
	e.addSwitch("rites/heretic.json", `{"states": ["x"], "files": [{"file": "/x", "values": []}]}`)
	out, errOut, code := e.run("inquisition")
	if code != 1 || !strings.Contains(out, ": heresy: ") || !strings.Contains(errOut, "+++ The Inquisition examined 2 rite(s)") {
		t.Fatalf("inquisition: code=%d out=%q err=%q", code, out, errOut)
	}
	out = e.mustRun("invoke", "mouse-autohide-toggle", "on", "--dry-run")
	if !strings.Contains(out, "The augury foresees changes to") {
		t.Fatalf("dry run: %s", out)
	}
}

func TestRoot_StartsTUIOnlyOnTerminal(t *testing.T) {
	e := grimEnv(t)
	var gotDir string
	var gotGrim bool
	origRun, origTerm := runTUI, isTerminal
	t.Cleanup(func() { runTUI, isTerminal = origRun, origTerm })
	runTUI = func(dir string, lex *lexicon.Lexicon) error {
		gotDir, gotGrim = dir, lex.Grimdark
		return nil
	}

	isTerminal = func() bool { return false }
	if out := e.mustRun(); !strings.Contains(out, "Sanctioned Rituals") || gotDir != "" {
		t.Fatal("without a terminal servitor must print help, not start the TUI")
	}
	isTerminal = func() bool { return true }
	e.mustRun()
	if gotDir != e.cfgDir || !gotGrim {
		t.Fatalf("TUI not started with config dir: %q grim=%v", gotDir, gotGrim)
	}
	gotDir = ""
	e.grim = false
	e.mustRun("tui")
	if gotDir != e.cfgDir || gotGrim {
		t.Fatalf("tui command: %q grim=%v", gotDir, gotGrim)
	}
	if _, errOut, code := e.run("bogus"); code == 0 || !strings.Contains(errOut, "unknown command") {
		t.Fatalf("unknown command: %d %q", code, errOut)
	}
}
