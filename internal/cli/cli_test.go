package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type env struct {
	t              *testing.T
	cfgDir, target string
	state, bin     string // the state home; the place of the notify-send stub
}

const mouseJSON = `{
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

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{t: t, cfgDir: t.TempDir(), state: t.TempDir(), bin: t.TempDir()}
	e.target = filepath.Join(t.TempDir(), "util.kdl")
	if err := os.WriteFile(e.target, []byte("input {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e.addRite("rites/mouse-autohide-toggle.json", mouseJSON)
	t.Setenv("SERVITOR_LIBRARIUM", "")
	t.Setenv("SERVITOR_CONFIG", "")
	t.Setenv("SERVITOR_NO_GRIMDARK", "")
	t.Setenv("SERVITOR_CHRONICLE", "")
	t.Setenv("XDG_STATE_HOME", e.state)
	stub := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + filepath.Join(e.bin, "notified") + "\necho 7\n"
	if err := os.WriteFile(filepath.Join(e.bin, "notify-send"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", e.bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	e.terminal(-1)
	return e
}

// terminal lends the servitor a controlling terminal of cols columns (0:
// of unknown width); a negative cols means none.
func (e *env) terminal(cols int) {
	orig := controllingTerminal
	e.t.Cleanup(func() { controllingTerminal = orig })
	controllingTerminal = func() (int, bool) { return max(0, cols), cols >= 0 }
}

// notified returns the arguments of every notify-send call, one per line.
func (e *env) notified() string {
	b, _ := os.ReadFile(filepath.Join(e.bin, "notified"))
	return string(b)
}

func (e *env) addRite(rel, body string) string {
	e.t.Helper()
	p := filepath.Join(e.cfgDir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		e.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(strings.ReplaceAll(body, "$TARGET", e.target)), 0o644); err != nil {
		e.t.Fatal(err)
	}
	return p
}

// run executes servitor with --librarium pointing at the test Librarium.
func (e *env) run(args ...string) (stdout, stderr string, code int) {
	e.t.Helper()
	var out, errb bytes.Buffer
	code = Execute(append([]string{"--librarium", e.cfgDir}, args...), &out, &errb)
	return out.String(), errb.String(), code
}

func (e *env) mustRun(args ...string) string {
	e.t.Helper()
	out, errOut, code := e.run(args...)
	if code != 0 {
		e.t.Fatalf("servitor %v: exit %d\nstdout: %s\nstderr: %s", args, code, out, errOut)
	}
	return out
}

func (e *env) targetContent() string {
	e.t.Helper()
	b, err := os.ReadFile(e.target)
	if err != nil {
		e.t.Fatal(err)
	}
	return string(b)
}

func (e *env) writeTarget(content string) {
	e.t.Helper()
	if err := os.WriteFile(e.target, []byte(content), 0o644); err != nil {
		e.t.Fatal(err)
	}
}

func (e *env) chronicleLines() []string {
	b, err := os.ReadFile(filepath.Join(e.state, "servitor", "chronicle.jsonl"))
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(b)), "\n")
}

func TestResolveLibrarium(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	t.Setenv("SERVITOR_LIBRARIUM", "")
	t.Setenv("SERVITOR_CONFIG", "/ignored")
	cases := map[string][]string{
		"/xdg/servitor": {"invoke", "x"},
		"/a":            {"--librarium", "/a", "invoke"},
		"/b":            {"invoke", "--librarium=/b"},
		"/c":            {"-l", "/c"},
		"/d":            {"-l/d"},
		"/e":            {"-l=/e"},
	}
	for want, args := range cases {
		if got := resolveConfigDir(args); got != want {
			t.Errorf("%v: got %s want %s", args, got, want)
		}
	}
	if got := resolveConfigDir([]string{"-l", "/first", "--librarium", "/last"}); got != "/last" {
		t.Errorf("last occurrence must win, got %s", got)
	}
	if got := resolveConfigDir([]string{"--", "--librarium", "/x"}); got != "/xdg/servitor" {
		t.Errorf("args after -- must be ignored, got %s", got)
	}
	if got := resolveConfigDir([]string{"--config", "/old", "-c", "/old"}); got != "/xdg/servitor" {
		t.Errorf("the forsaken --config rune must not choose the Librarium, got %s", got)
	}
	t.Setenv("HOME", "/home/adept")
	if got := resolveConfigDir([]string{"-l", "~/lib"}); got != "/home/adept/lib" {
		t.Errorf("~ must be the home, got %s", got)
	}
	t.Setenv("SERVITOR_LIBRARIUM", "/env")
	if got := resolveConfigDir(nil); got != "/env" {
		t.Errorf("env: got %s", got)
	}
	if got := resolveConfigDir([]string{"-l", "/flag"}); got != "/flag" {
		t.Errorf("rune must win over env, got %s", got)
	}
}

func TestLibrarium_RuneAndEnvironmentChooseTheRites(t *testing.T) {
	e := newEnv(t)
	var out, errb bytes.Buffer
	if code := Execute([]string{"-l", e.cfgDir, "census"}, &out, &errb); code != 0 || !strings.Contains(out.String(), "mouse-autohide-toggle") {
		t.Fatalf("-l: code=%d out=%q err=%q", code, out.String(), errb.String())
	}
	t.Setenv("SERVITOR_LIBRARIUM", e.cfgDir)
	out.Reset()
	if code := Execute([]string{"census"}, &out, &errb); code != 0 || !strings.Contains(out.String(), "mouse-autohide-toggle") {
		t.Fatalf("SERVITOR_LIBRARIUM: code=%d out=%q err=%q", code, out.String(), errb.String())
	}
}
