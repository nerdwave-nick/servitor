package cli

import (
	"bytes"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// examples is the example Librarium shown to the faithful.
const examples = "../../examples"

// spokenExamples are the example rites a trial may speak, with their aspects
// in the order invoked, back to the first. The exalted ones (gaming and
// split-lock) ask the machine's own authority and are judged, never spoken.
var spokenExamples = []struct {
	rite    string
	aspects []string
}{
	{"theme", []string{"porpl", "finii", "default", "porpl"}},
	{"mouse-autohide-toggle", []string{"on", "off", "on"}},
	{"do-not-disturb", []string{"on", "off", "on"}},
}

// TestExamples_EverySpokenRiteMovesBetweenItsAspects: the example rites
// that need no exaltation are invoked into each aspect and back with
// triumph, upon a home of their own and with the programs they call stood
// in for (niri, makoctl, notify-send). A second invocation into the same
// aspect changes nothing, and the augury reads each one performed.
func TestExamples_EverySpokenRiteMovesBetweenItsAspects(t *testing.T) {
	e := newExamples(t)
	for _, ex := range spokenExamples {
		t.Run(ex.rite, func(t *testing.T) {
			for _, aspect := range ex.aspects {
				e.invoke(t, ex.rite, aspect)
				if changed := e.lastChanged(t); len(changed) == 0 {
					t.Fatalf("invoke %s %s changed nothing", ex.rite, aspect)
				}
				e.invoke(t, ex.rite, aspect)
				if changed := e.lastChanged(t); len(changed) != 0 {
					t.Fatalf("invoke %s %s again changed %v", ex.rite, aspect, changed)
				}
				if got := e.augury(t, ex.rite, "aspect"); got != aspect {
					t.Fatalf("augury %s aspect = %q, want %q", ex.rite, got, aspect)
				}
				if got := e.augury(t, ex.rite, "standing"); got != "performed" {
					t.Fatalf("augury %s standing = %q, want performed", ex.rite, got)
				}
			}
		})
	}
	if got := e.read(t, "awakened"); got != "porpl\nfinii\ndefault\nporpl\n" {
		t.Errorf("the theme's litany awakened %q", got)
	}
	if got := e.read(t, ".config/niri/util.kdl"); !strings.Contains(got, "hide-after-inactive-ms 400") {
		t.Errorf("the cursor's sanctum ends as\n%s", got)
	}
	if got := strings.Count(e.read(t, "heralded"), "The heralds fall silent"); got != 2 {
		t.Errorf("the heralds were silenced %d times, want 2 (only when entering on)", got)
	}
}

// TestExamples_TheEdictIsProclaimedWhole: the example edict captures the
// visage at every proclaiming, the region chosen or whole.
func TestExamples_TheEdictIsProclaimedWhole(t *testing.T) {
	e := newExamples(t)
	e.invoke(t, "screenshot")
	e.invoke(t, "screenshot", "--region", "whole")
	if got := e.read(t, "captured"); got != "-g 0,0 64x64\n\n" {
		t.Errorf("grim heard %q", got)
	}
	if got := e.augury(t, "screenshot", "standing"); got != "proclaimed" {
		t.Errorf("the edict stands %q", got)
	}
}

type exampleEnv struct {
	*env
	home string
}

// newExamples lends the example Librarium a home of its own and stands in
// for every program its spoken rites call upon. The stand-ins keep what
// they heard in the home.
func newExamples(t *testing.T) exampleEnv {
	t.Helper()
	e := exampleEnv{env: newEnv(t), home: t.TempDir()}
	t.Setenv("HOME", e.home)
	for _, hall := range []string{".config/niri", ".local/share/nfluff", "pictures"} {
		if err := os.MkdirAll(filepath.Join(e.home, hall), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, theme := range []string{"default", "finii", "porpl"} {
		hall := filepath.Join(e.home, ".config/nfluff/themes", theme)
		if err := os.MkdirAll(hall, 0o755); err != nil {
			t.Fatal(err)
		}
		e.write(t, filepath.Join(hall, "hooks"), `basename "$THEME_PATH" >> "$HOME/awakened"`+"\n", 0o644)
	}
	for name, body := range map[string]string{
		"niri": `exit 0`,
		"makoctl": `case "$1 $2" in
"mode ") cat "$HOME/mako-mode" 2>/dev/null || echo default ;;
"mode -s") echo "$3" > "$HOME/mako-mode" ;;
*) exit 2 ;;
esac`,
		"notify-send": `printf '%s\n' "$*" >> "$HOME/heralded"; echo 7`,
		"grim":        `for a; do last=$a; done; [ "$1" = -g ] && echo "$1 $2" >> "$HOME/captured" || echo >> "$HOME/captured"; echo visage > "$last"`,
		"slurp":       `echo "0,0 64x64"`,
		"wl-copy":     `cat > /dev/null`,
	} {
		e.write(t, filepath.Join(e.bin, name), "#!/bin/sh\n"+body+"\n", 0o755)
	}
	return e
}

func (e exampleEnv) write(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}

func (e exampleEnv) read(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(e.home, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func (e exampleEnv) servitor(args ...string) (string, string, int) {
	var out, errb bytes.Buffer
	code := Execute(append([]string{"--librarium", examples}, args...), &out, &errb)
	return out.String(), errb.String(), code
}

func (e exampleEnv) invoke(t *testing.T, args ...string) {
	t.Helper()
	if out, errb, code := e.servitor(append([]string{"invoke"}, args...)...); code != 0 {
		t.Fatalf("invoke %v: exit %d\nstdout: %s\nstderr: %s", args, code, out, errb)
	}
}

func (e exampleEnv) augury(t *testing.T, rite, key string) string {
	t.Helper()
	out, errb, code := e.servitor("augury", rite, key)
	if code != 0 {
		t.Fatalf("augury %s %s: exit %d\n%s", rite, key, code, errb)
	}
	return strings.TrimSpace(out)
}

// lastChanged is what the last line of the chronicle names as changed.
func (e exampleEnv) lastChanged(t *testing.T) []string {
	t.Helper()
	lines := e.chronicleLines()
	if len(lines) == 0 {
		t.Fatal("the chronicle is empty")
	}
	var line struct {
		Verdict string   `json:"verdict"`
		Changed []string `json:"changed"`
	}
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &line); err != nil {
		t.Fatal(err)
	}
	if line.Verdict != "triumph" {
		t.Fatalf("the last invocation ended %q: %s", line.Verdict, lines[len(lines)-1])
	}
	return slices.Clip(line.Changed)
}
