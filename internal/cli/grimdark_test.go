package cli

import (
	"regexp"
	"strings"
	"testing"
)

// TestRootHelpAndVersion: the servitor's own lore is the index of the codex;
// a ritual's lore is its passage, followed by its flavored invocation.
func TestRootHelpAndVersion(t *testing.T) {
	e := newEnv(t)
	help := e.mustRun("--help")
	for _, want := range []string{
		"THE CODEX", "servitor expound <topic>", "Rituals\n", "Runes\n",
		"invoke", "augury", "census", "inquisition", "cogitator", "completion", "expound",
		"librarium", "chronicle", "sanctum", "tether", "pattern",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("root help missing %q", want)
		}
	}
	for _, unwanted := range []string{"no-grimdark", "NO_GRIMDARK", "--config", "SERVITOR_CONFIG", "Available Commands:"} {
		if strings.Contains(help, unwanted) {
			t.Errorf("root help still speaks of %q", unwanted)
		}
	}
	if sub := e.mustRun("census", "--help"); !strings.Contains(sub, "as written in the codex") ||
		!strings.Contains(sub, "Universal Runes:") || !strings.Contains(sub, "--binharic") || !strings.Contains(sub, "-l, --librarium") {
		t.Errorf("census help not flavored:\n%s", sub)
	}
	if out := e.mustRun("--version"); !strings.Contains(out, "blessed be the Omnissiah") {
		t.Fatalf("version: %q", out)
	}
}

// TestHelpAliases_WorkButStayHidden: help, -h and --help reveal the lore but
// are named neither in listings nor in completion.
func TestHelpAliases_WorkButStayHidden(t *testing.T) {
	e := newEnv(t)
	root := e.mustRun("--help")
	if got := e.mustRun("-h"); got != root {
		t.Error("-h must match --help")
	}
	if got := e.mustRun("help"); got != root {
		t.Error("help must match --help")
	}
	census := e.mustRun("census", "--help")
	if got := e.mustRun("help", "census"); got != census {
		t.Error("help census must match census --help")
	}
	listedHelp := regexp.MustCompile(`(?m)^\s+help\s`)
	helpRune := regexp.MustCompile(`(?m)^\s+(-h, )?--help\b`)
	for _, args := range [][]string{{"--help"}, {"census", "--help"}, {"invoke", "--help"},
		{"invoke", "mouse-autohide-toggle", "--help"}, {"completion", "--help"}} {
		out := e.mustRun(args...)
		if listedHelp.MatchString(out) || helpRune.MatchString(out) {
			t.Errorf("%v lists a help alias:\n%s", args, out)
		}
	}
	for _, args := range [][]string{{""}, {"h"}, {"-"}, {"--"}, {"census", "-"}, {"inquisition", "--"},
		{"invoke", "mouse-autohide-toggle", "on", "--"}, {"augury", "mouse-autohide-toggle", "--"}} {
		got, _ := e.complete(args...)
		for _, c := range got {
			if c == "help" || c == "-h" || c == "--help" || c == "--json" {
				t.Errorf("completion %v offers hidden alias %q: %v", args, c, got)
			}
		}
	}
}

// TestPlainNames_AreUnknown: the forsaken plain rituals and runes are no
// longer understood.
func TestPlainNames_AreUnknown(t *testing.T) {
	e := newEnv(t)
	for _, name := range []string{"switch", "profile", "sw", "meta", "list", "ls", "verify", "check", "validate", "tui", "ui"} {
		_, errOut, code := e.run(name, "mouse-autohide-toggle", "on")
		if code == 0 || !strings.Contains(errOut, `servitor ✠ unknown ritual "`+name+`"`) {
			t.Errorf("%s: code=%d stderr=%q", name, code, errOut)
		}
	}
	runes := [][]string{
		{"--no-grimdark", "census"},
		{"census", "--no-grimdark"},
		{"--config", e.cfgDir, "census"},
		{"inquisition", "--no-files"},
		{"invoke", "mouse-autohide-toggle", "on", "--dry-run"},
		{"invoke", "mouse-autohide-toggle", "on", "--quiet"},
	}
	for _, args := range runes {
		_, errOut, code := e.run(args...)
		if code == 0 || !strings.Contains(errOut, "unknown rune: ") || strings.Contains(errOut, "unknown flag") {
			t.Errorf("%v: code=%d stderr=%q", args, code, errOut)
		}
	}
	for _, args := range [][]string{{"invoke", "mouse-autohide-toggle", "on", "-n"}, {"invoke", "mouse-autohide-toggle", "on", "-q"}, {"-c", "x", "census"}} {
		if _, errOut, code := e.run(args...); code == 0 || !strings.Contains(errOut, "unknown rune") {
			t.Errorf("%v: code=%d stderr=%q", args, code, errOut)
		}
	}
	if e.targetContent() != "input {}\n" {
		t.Fatal("vessel modified by forsaken runes")
	}
	got, _ := e.complete("")
	for _, c := range got {
		if strings.Contains(" switch profile sw meta list ls verify check validate tui ui ", " "+c+" ") {
			t.Errorf("completion offers plain ritual %q", c)
		}
	}
}

func TestPlainVocabulary_EnvironmentIsIgnored(t *testing.T) {
	e := newEnv(t)
	want := e.mustRun("--help")
	t.Setenv("SERVITOR_NO_GRIMDARK", "1")
	if got := e.mustRun("--help"); got != want {
		t.Fatal("SERVITOR_NO_GRIMDARK must change nothing")
	}
	if got := e.mustRun("census"); !strings.Contains(got, "RITE") || !strings.Contains(got, "dormant") {
		t.Fatalf("census:\n%s", got)
	}
}

func TestRoot_StartsCogitatorOnlyOnTerminal(t *testing.T) {
	e := newEnv(t)
	var gotDir string
	origRun, origTerm := runTUI, isTerminal
	t.Cleanup(func() { runTUI, isTerminal = origRun, origTerm })
	runTUI = func(dir string) error {
		gotDir = dir
		return nil
	}

	isTerminal = func() bool { return false }
	if out := e.mustRun(); !strings.Contains(out, "THE CODEX") || gotDir != "" {
		t.Fatal("without a terminal servitor must print help, not awaken the cogitator")
	}
	isTerminal = func() bool { return true }
	e.mustRun()
	if gotDir != e.cfgDir {
		t.Fatalf("cogitator not awakened with the Librarium: %q", gotDir)
	}
	gotDir = ""
	e.mustRun("cogitator")
	if gotDir != e.cfgDir {
		t.Fatalf("cogitator ritual: %q", gotDir)
	}
	if _, errOut, code := e.run("bogus"); code == 0 || !strings.Contains(errOut, `unknown ritual "bogus"`) {
		t.Fatalf("unknown ritual: %d %q", code, errOut)
	}
}
