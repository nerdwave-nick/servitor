package cli

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// elder is where the scriptures of pattern Mark I lie that the conversion
// is judged by: the owner's three rites, and the example rites as they were
// written before pattern Mark II.
const elder = "../librarium/testdata/conversion"

// convertedRites are rites of pattern Mark I, laid in a Librarium of their
// own (owner or examples) and converted on its first reading, with their
// aspects in the order invoked, back to the first; none for an edict. The
// exalted example split-lock asks the machine's own authority and is judged
// in the VM trial.
var convertedRites = []struct {
	hall, rite string
	aspects    []string
}{
	{"owner", "mouse-autohide-toggle", []string{"on", "off", "on"}},
	{"owner", "dnd", []string{"on", "off", "on"}},
	{"owner", "gaming", []string{"on", "off", "on"}},
	{"examples", "mouse-autohide-toggle", []string{"on", "off", "on"}},
	{"examples", "theme", []string{"porpl", "finii", "default", "porpl"}},
	{"examples", "screenshot", nil},
}

// TestConverted_EveryRiteOfMarkIMovesBetweenItsAspects: the owner's rites and
// the example rites of pattern Mark I are converted by the first ritual that
// reads them, each original kept in rites/.mark-i/, upon a home of their own
// and with every program they call stood in for (makoctl, notify-send,
// sysctl, scx_cosmos, killall, tee, niri, grim, slurp, wl-copy), so that no
// word reaches the machine itself. Each rite of aspects is then invoked into
// each aspect and back with triumph; invoked again into the same aspect, it
// leaves the machine as it was and changes no subject it can read — only a
// machine-spirit without an auspex, which conversion made of a verse of
// speech, is enacted anew, as every machine-spirit that cannot be read is —
// and the augury reads it performed. The edict is proclaimed whole each time.
func TestConverted_EveryRiteOfMarkIMovesBetweenItsAspects(t *testing.T) {
	for _, hall := range []string{"owner", "examples"} {
		t.Run(hall, func(t *testing.T) {
			e := newConverted(t, hall)
			_, errOut, code := e.run("census")
			if code != 0 {
				t.Fatalf("census: exit %d\n%s", code, errOut)
			}
			for _, c := range convertedRites {
				if c.hall != hall {
					continue
				}
				if !strings.Contains(errOut, `the rite "`+c.rite+`" is converted to pattern "Mark II"`) {
					t.Errorf("the census did not tell the conversion of %s:\n%s", c.rite, errOut)
				}
				original, _ := os.ReadFile(filepath.Join(elder, hall, c.rite+".json"))
				if kept := e.scripture("rites/.mark-i/" + c.rite + ".json"); kept != string(original) {
					t.Errorf("the original of %s is not kept in rites/.mark-i/", c.rite)
				}
				if !strings.Contains(e.scripture("rites/"+c.rite+".json"), `"pattern": "Mark II"`) {
					t.Errorf("%s is not converted on disk", c.rite)
				}
			}
			grimdark(t, errOut)
			for _, c := range convertedRites {
				if c.hall != hall {
					continue
				}
				t.Run(c.rite, func(t *testing.T) { e.movesBetween(t, c.rite, c.aspects) })
			}
		})
	}
}

// convertedEnv is a Librarium of rites of pattern Mark I, upon a home of
// its own whose files the stand-ins keep the machine's state in.
type convertedEnv struct {
	*env
	home string
}

// newConverted lays the scriptures of pattern Mark I of hall in a new
// Librarium, beside the scroll the example theme recites, and stands in for
// every program they call.
func newConverted(t *testing.T, hall string) convertedEnv {
	t.Helper()
	e := convertedEnv{env: newEnv(t), home: t.TempDir()}
	if err := os.Remove(filepath.Join(e.cfgDir, "rites", "mouse-autohide-toggle.json")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", e.home)
	for _, c := range convertedRites {
		if c.hall == hall {
			b, err := os.ReadFile(filepath.Join(elder, hall, c.rite+".json"))
			if err != nil {
				t.Fatal(err)
			}
			e.addRite("rites/"+c.rite+".json", string(b))
		}
	}
	scroll, err := os.ReadFile(filepath.Join(examples, "rites", "scripts", "recite-hooks"))
	if err != nil {
		t.Fatal(err)
	}
	e.put(t, filepath.Join(e.cfgDir, "rites", "scripts", "recite-hooks"), string(scroll), 0o755)
	for _, hall := range []string{".config/niri", ".local/share/nfluff", "pictures"} {
		if err := os.MkdirAll(filepath.Join(e.home, hall), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, theme := range []string{"default", "finii", "porpl"} {
		e.put(t, filepath.Join(e.home, ".config/nfluff/themes", theme, "hooks"),
			`basename "$THEME_PATH" >> "$HOME/awakened"`+"\n", 0o644)
	}
	for name, body := range map[string]string{
		"niri": `exit 0`,
		"makoctl": `case "$1 $2 $3" in
"mode ") cat "$HOME/mako-mode" 2>/dev/null || echo default ;;
"mode -a dnd") echo dnd > "$HOME/mako-mode" ;;
"mode -r dnd") echo default > "$HOME/mako-mode" ;;
*) exit 2 ;;
esac`,
		"notify-send": `printf '%s\n' "$*" >> "$HOME/heralded"; echo 7`,
		"sysctl":      `[ "$1" = -wq ] || exit 2; echo "${2#*=}" > "$HOME/split-lock"`,
		"scx_cosmos":  `echo cosmos > "$HOME/scheduler"`,
		"killall":     `[ "$1" = scx_cosmos ] && echo eevdf > "$HOME/scheduler"`,
		"tee":         `cat > "$HOME/smt"`,
		"grim":        `for a; do last=$a; done; echo visage > "$last"`,
		"slurp":       `echo "0,0 64x64"`,
		"wl-copy":     `cat > /dev/null`,
	} {
		e.put(t, filepath.Join(e.bin, name), "#!/bin/sh\n"+body+"\n", 0o755)
	}
	return e
}

func (e convertedEnv) put(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}

// machine is the state the stand-ins keep, and every vessel of the home.
func (e convertedEnv) machine(t *testing.T) string {
	t.Helper()
	var state []string
	for _, rel := range []string{"mako-mode", "split-lock", "scheduler", "smt", ".config/niri/util.kdl",
		".local/share/nfluff/current-theme"} {
		p := filepath.Join(e.home, rel)
		b, err := os.ReadFile(p)
		if link, lerr := os.Readlink(p); lerr == nil {
			b, err = []byte("→ "+link), nil
		}
		if err == nil {
			state = append(state, rel+": "+string(b))
		}
	}
	return strings.Join(state, "\n")
}

// movesBetween invokes rite into each of aspects, and again into each; an
// edict (no aspects) is proclaimed twice.
func (e convertedEnv) movesBetween(t *testing.T, rite string, aspects []string) {
	if aspects == nil {
		for range 2 {
			e.invoke(t, rite)
			if got := e.augur(t, rite, "standing"); got != "proclaimed" {
				t.Fatalf("the edict %s stands %q", rite, got)
			}
		}
		return
	}
	unread := e.unread(t, rite)
	for _, aspect := range aspects {
		e.invoke(t, rite, aspect)
		if changed := e.changed(t); len(changed) == 0 {
			t.Fatalf("invoke %s %s changed nothing", rite, aspect)
		}
		before := e.machine(t)
		e.invoke(t, rite, aspect)
		if changed := e.changed(t); !slices.Equal(changed, unread) {
			t.Fatalf("invoke %s %s again changed %v; only the unread machine-spirits %v are enacted anew",
				rite, aspect, changed, unread)
		}
		if after := e.machine(t); after != before {
			t.Fatalf("invoke %s %s again changed the machine:\n%s\nbecame\n%s", rite, aspect, before, after)
		}
		if got := e.augur(t, rite, "aspect"); got != aspect {
			t.Fatalf("augury %s aspect = %q, want %q", rite, got, aspect)
		}
		if got := e.augur(t, rite, "standing"); got != "performed" {
			t.Fatalf("augury %s standing = %q, want performed", rite, got)
		}
	}
}

// unread names the subjects of the converted rite that cannot be read: its
// machine-spirits without an auspex.
func (e convertedEnv) unread(t *testing.T, rite string) []string {
	t.Helper()
	var scripture struct {
		Subjects []struct {
			Name        string `json:"name"`
			Designation string `json:"designation"`
			Auspex      any    `json:"auspex"`
		} `json:"subjects"`
	}
	data := e.scripture("rites/" + rite + ".json")
	if err := json.Unmarshal([]byte(stripRemarks(data)), &scripture); err != nil {
		t.Fatalf("the converted %s cannot be read: %v\n%s", rite, err, data)
	}
	var unread []string
	for _, s := range scripture.Subjects {
		if s.Designation == "machine-spirit" && s.Auspex == nil {
			unread = append(unread, s.Name)
		}
	}
	return unread
}

// stripRemarks strikes the lines of remark a conversion writes.
func stripRemarks(jsonc string) string {
	var lines []string
	for _, l := range strings.Split(jsonc, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(l), "//") {
			lines = append(lines, l)
		}
	}
	return strings.Join(lines, "\n")
}

func (e convertedEnv) invoke(t *testing.T, args ...string) {
	t.Helper()
	if out, errb, code := e.run(append([]string{"invoke"}, args...)...); code != 0 {
		t.Fatalf("invoke %v: exit %d\nstdout: %s\nstderr: %s", args, code, out, errb)
	}
}

func (e convertedEnv) augur(t *testing.T, rite, key string) string {
	t.Helper()
	out, errb, code := e.run("augury", rite, key)
	if code != 0 {
		t.Fatalf("augury %s %s: exit %d\n%s", rite, key, code, errb)
	}
	return strings.TrimSpace(out)
}

// changed is what the last line of the chronicle names as changed; the
// invocation must have ended in triumph.
func (e convertedEnv) changed(t *testing.T) []string {
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
	return line.Changed
}
