package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/tui"
)

// asRoot makes the servitor bear the machine's own authority (euid 0); no
// test ever truly runs as root.
func asRoot(t *testing.T) {
	t.Helper()
	orig := geteuid
	t.Cleanup(func() { geteuid = orig })
	geteuid = func() int { return 0 }
}

// keepWorld lets the servitor cast away the variables of the environment
// that place its world (HOME and every XDG_*) and restores them after t.
func keepWorld(t *testing.T) {
	t.Helper()
	for _, kv := range os.Environ() {
		if k, v, _ := strings.Cut(kv, "="); k == "HOME" || strings.HasPrefix(k, "XDG_") {
			t.Setenv(k, v)
		}
	}
}

// TestAsRoot_PerformingRitualsAreRefused: a servitor bearing the machine's
// own authority invokes nothing, proclaims nothing and awakens no
// cogitator: it refuses with the death-mark 2 before anything is touched.
func TestAsRoot_PerformingRitualsAreRefused(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", edictJSON)
	origRun, origTerm := runTUI, isTerminal
	t.Cleanup(func() { runTUI, isTerminal = origRun, origTerm })
	awakened := false
	runTUI = func(tui.Options) error { awakened = true; return nil }
	isTerminal = func() bool { return true }
	asRoot(t)
	before := e.targetContent()
	for _, tc := range []struct {
		args []string
		deed string
	}{
		{[]string{"invoke", "mouse-autohide-toggle", "on"}, `the invocation of the rite "mouse-autohide-toggle" into "on"`},
		{[]string{"invoke", "mouse-autohide-toggle", "off", "--binharic"}, `the invocation of the rite "mouse-autohide-toggle" into "off"`},
		{[]string{"invoke", "capture"}, `the proclaiming of the edict "capture"`},
		{[]string{"cogitator"}, "the awakening of the cogitator"},
		{nil, "the awakening of the cogitator"},
	} {
		out, errOut, code := e.run(tc.args...)
		if code != 2 || out != "" || !strings.Contains(errOut, tc.deed+" is refused") ||
			!strings.Contains(errOut, "the machine's own authority") {
			t.Errorf("servitor %v as root: exit %d\nstdout: %s\nstderr: %s", tc.args, code, out, errOut)
		}
	}
	if awakened {
		t.Error("the cogitator was awakened as root")
	}
	if got := e.targetContent(); got != before {
		t.Errorf("the vessel was touched as root:\n%s", got)
	}
	if _, err := os.Stat(e.target + ".edict"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the edict was proclaimed as root: %v", err)
	}
	if lines := e.chronicleLines(); lines != nil {
		t.Errorf("the chronicle was written as root: %v", lines)
	}
	if _, err := os.Stat(filepath.Join(e.state, "servitor", "data-slates")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a data-slate was written as root: %v", err)
	}
}

// TestAsRoot_ReadingRitualsServe: the rituals that only read still serve a
// servitor bearing the machine's own authority.
func TestAsRoot_ReadingRitualsServe(t *testing.T) {
	e := newEnv(t)
	asRoot(t)
	for _, args := range [][]string{
		{"invoke", "mouse-autohide-toggle", "on", "--foresee"}, {"augury", "mouse-autohide-toggle"},
		{"census"}, {"inquisition", "--spare-vessels"}, {"expound", "invoke"}, {"invoke"}, {"--version"},
	} {
		if out, errOut, code := e.run(args...); code != 0 {
			t.Errorf("servitor %v as root: exit %d\nstdout: %s\nstderr: %s", args, code, out, errOut)
		}
	}
}

// veilJSON is a rite whose vessel lies in the home of the one who invokes it.
const veilJSON = `{"pattern": "Mark I", "purpose": "Veil the cursor", "aspects": ["on", "off"],
  "liturgy": [{"sanctum": "~/util.kdl", "scripture": {"on": "cursor {}", "off": ""}}]}`

// decoyJSON is a rite of root's own Librarium.
const decoyJSON = `{"pattern": "Mark I", "aspects": ["on", "off"], "liturgy": [{"incantation": "true"}]}`

// sudoWorld is an invoker's world and root's, side by side.
type sudoWorld struct {
	e                *env
	invoker, rootDir string // the homes
}

// newSudoWorld raises the invoker adept's world — a Librarium in its
// default hall, holding veil, invoked into on as adept — and root's own,
// whose Librarium holds decoy, then rouses the servitor as root under
// sudo: HOME and the XDG variables are root's.
func newSudoWorld(t *testing.T) *sudoWorld {
	t.Helper()
	keepWorld(t)
	w := &sudoWorld{e: newEnv(t), invoker: t.TempDir(), rootDir: t.TempDir()}
	for _, k := range []string{"XDG_CONFIG_HOME", "XDG_STATE_HOME", "SUDO_USER", "SUDO_HOME", "SUDO_UID"} {
		t.Setenv(k, "")
	}
	t.Setenv("HOME", w.invoker)
	writeFile(t, filepath.Join(w.invoker, ".config", "servitor", "rites", "veil.json"), veilJSON)
	writeFile(t, filepath.Join(w.invoker, "util.kdl"), "input {}\n")
	if out, errOut, code := w.execute("invoke", "veil", "on"); code != 0 {
		t.Fatalf("adept's invocation: exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	writeFile(t, filepath.Join(w.rootDir, ".xdg-config", "servitor", "rites", "decoy.json"), decoyJSON)
	t.Setenv("HOME", w.rootDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(w.rootDir, ".xdg-config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(w.rootDir, ".xdg-state"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(w.rootDir, ".xdg-data"))
	t.Setenv("SUDO_USER", "adept")
	t.Setenv("SUDO_HOME", w.invoker)
	asRoot(t)
	return w
}

// execute runs servitor without --librarium, as the faithful would.
func (w *sudoWorld) execute(args ...string) (stdout, stderr string, code int) {
	var out, errb bytes.Buffer
	code = Execute(args, &out, &errb)
	return out.String(), errb.String(), code
}

// census returns the rites of the census and the aspect each stands in.
func (w *sudoWorld) census(t *testing.T) map[string]string {
	t.Helper()
	out, errOut, code := w.execute("census", "--binharic")
	if code != 0 {
		t.Fatalf("census: exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	var entries []struct{ Rite, Aspect, Standing string }
	if err := json.Unmarshal([]byte(out), &entries); err != nil {
		t.Fatalf("census: %v\n%s", err, out)
	}
	got := map[string]string{}
	for _, e := range entries {
		got[e.Rite] = e.Aspect + " " + e.Standing
	}
	return got
}

// TestUnderSudo_ReadingRitualsReadTheInvokersWorld: roused as root by sudo,
// the reading rituals read the Librarium, vessels and data-slates of the
// invoker (SUDO_USER, SUDO_HOME), heed none of root's XDG variables, and
// write nothing.
func TestUnderSudo_ReadingRitualsReadTheInvokersWorld(t *testing.T) {
	w := newSudoWorld(t)
	before := snapshot(t, w.rootDir, w.invoker)
	if got := w.census(t); len(got) != 1 || got["veil"] != "on performed" {
		t.Errorf("census under sudo: %v, want veil on performed and no decoy", got)
	}
	if out, errOut, code := w.execute("augury", "veil", "aspect"); code != 0 || out != "on\n" {
		t.Errorf("augury under sudo: exit %d %q\n%s", code, out, errOut)
	}
	if out, errOut, code := w.execute("invoke", "veil", "off", "--foresee"); code != 0 ||
		!strings.Contains(out, filepath.Join(w.invoker, "util.kdl")) && !strings.Contains(out, "~/util.kdl") {
		t.Errorf("foresight under sudo: exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	if out, errOut, code := w.execute("inquisition", "veil"); code != 0 || out != "" {
		t.Errorf("inquisition under sudo: exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	if out, _, code := w.execute("expound", "invoke"); code != 0 || !strings.Contains(out, "veil") ||
		strings.Contains(out, "decoy") {
		t.Errorf("expound under sudo: exit %d, names veil and not decoy?\n%s", code, out)
	}
	if after := snapshot(t, w.rootDir, w.invoker); after != before {
		t.Errorf("the reading rituals wrote under sudo:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if out, errOut, code := w.execute("invoke", "veil", "off"); code != 2 || !strings.Contains(errOut, `"veil" into "off" is refused`) {
		t.Errorf("a performing invocation under sudo: exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
}

// TestUnderSudo_TheInvokersHome: without SUDO_HOME the invoker's home is
// that of SUDO_USER; a SUDO_USER whose home cannot be found is refused,
// rather than root's world read in its place; without SUDO_USER, and for a
// servitor that does not bear the machine's authority, the world is its own.
func TestUnderSudo_TheInvokersHome(t *testing.T) {
	t.Run("home of SUDO_USER", func(t *testing.T) {
		w := newSudoWorld(t)
		t.Setenv("SUDO_HOME", "")
		lookupAs(t, map[string]string{"adept": w.invoker})
		if got := w.census(t); len(got) != 1 || got["veil"] != "on performed" {
			t.Errorf("census: %v, want veil alone", got)
		}
	})
	t.Run("no home to be found", func(t *testing.T) {
		w := newSudoWorld(t)
		t.Setenv("SUDO_HOME", "")
		lookupAs(t, nil)
		out, errOut, code := w.execute("census")
		if code != 2 || out != "" || !strings.Contains(errOut, `"adept"`) {
			t.Errorf("census: exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
		}
	})
	t.Run("without SUDO_USER", func(t *testing.T) {
		w := newSudoWorld(t)
		t.Setenv("SUDO_USER", "")
		if got := w.census(t); len(got) != 1 || got["decoy"] == "" {
			t.Errorf("census: %v, want root's decoy alone", got)
		}
	})
	t.Run("not as root", func(t *testing.T) {
		w := newSudoWorld(t)
		geteuid = func() int { return 1000 }
		if got := w.census(t); len(got) != 1 || got["decoy"] == "" {
			t.Errorf("census: %v, want the servitor's own decoy alone", got)
		}
	})
}

// lookupAs makes homes the homes of the faithful.
func lookupAs(t *testing.T, homes map[string]string) {
	t.Helper()
	orig := lookupHome
	t.Cleanup(func() { lookupHome = orig })
	lookupHome = func(name string) (string, error) {
		if h, ok := homes[name]; ok {
			return h, nil
		}
		return "", errors.New("unknown user " + name)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// snapshot tells every entry beneath the halls with its mode, size and
// content, one per line.
func snapshot(t *testing.T, halls ...string) string {
	t.Helper()
	var b strings.Builder
	for _, hall := range halls {
		err := filepath.WalkDir(hall, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			b.WriteString(p + " " + info.Mode().String())
			if d.Type().IsRegular() {
				data, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				b.WriteString(" " + string(data))
			}
			b.WriteString("\n")
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return b.String()
}
