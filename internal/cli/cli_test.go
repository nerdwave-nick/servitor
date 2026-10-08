package cli

import (
	"bytes"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/lexicon"
)

type env struct {
	t              *testing.T
	cfgDir, target string
	grim           bool // run in the default grimdark vocabulary
}

const mouseJSON = `{
  "description": "Hide the cursor after inactivity",
  "states": ["on", "off"],
  "files": [{
    "file": "$TARGET",
    "values": [
      {"state": "on", "value": "cursor {\n    hide-after-inactive-ms 400\n}", "meta": {"example-key": "example-value"}},
      {"state": "off", "value": "", "meta": {"example-key": "other-value", "reason": ""}},
    ],
    "meta": {
      "reason": {"optional": "true", "description": "why the switch was flipped"}, // optional as default
      "example-key": {"optional": "false"},
    },
  }],
}`

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{t: t, cfgDir: t.TempDir()}
	e.target = filepath.Join(t.TempDir(), "util.kdl")
	if err := os.WriteFile(e.target, []byte("input {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e.addSwitch("rites/mouse-autohide-toggle.json", mouseJSON)
	t.Setenv(EnvConfig, "")
	t.Setenv(lexicon.EnvNoGrimdark, "")
	return e
}

func (e *env) addSwitch(rel, body string) {
	e.t.Helper()
	p := filepath.Join(e.cfgDir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		e.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(strings.ReplaceAll(body, "$TARGET", e.target)), 0o644); err != nil {
		e.t.Fatal(err)
	}
}

// run executes servitor with --config pointing at the test config, in the
// plain vocabulary unless e.grim is set.
func (e *env) run(args ...string) (stdout, stderr string, code int) {
	e.t.Helper()
	var out, errb bytes.Buffer
	pre := []string{"--config", e.cfgDir}
	if !e.grim {
		pre = append(pre, "--no-grimdark")
	}
	code = Execute(append(pre, args...), &out, &errb)
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

func TestSwitchAndMeta_UserScenario(t *testing.T) {
	e := newEnv(t)
	out := e.mustRun("switch", "mouse-autohide-toggle", "on", "--reason", "gaming remnant")
	if !strings.Contains(out, "mouse-autohide-toggle: (unset) -> on") || !strings.Contains(out, "updated") {
		t.Fatalf("summary: %s", out)
	}
	if !strings.Contains(e.targetContent(), `// begin servitor managed -- mouse-autohide-toggle -- state|on example-key|example-value reason|"gaming remnant"`) {
		t.Fatalf("target:\n%s", e.targetContent())
	}
	want := "{\n  \"state\": \"on\",\n  \"example-key\": \"example-value\",\n  \"reason\": \"gaming remnant\"\n}\n"
	if got := e.mustRun("meta", "mouse-autohide-toggle"); got != want {
		t.Fatalf("meta:\n%s\nwant\n%s", got, want)
	}

	e.mustRun("profile", "mouse-autohide-toggle", "off") // "profile" alias
	want = "{\n  \"state\": \"off\",\n  \"example-key\": \"other-value\"\n}\n"
	if got := e.mustRun("meta", "mouse-autohide-toggle"); got != want {
		t.Fatalf("meta after off:\n%s", got)
	}
	if got := e.mustRun("meta", "mouse-autohide-toggle", "state"); got != "off\n" {
		t.Fatalf("meta state = %q", got)
	}
	if got := e.mustRun("meta", "mouse-autohide-toggle", "reason"); got != "\n" {
		t.Fatalf("unset key should print empty line, got %q", got)
	}
}

func TestMeta_IsAndExitCodes(t *testing.T) {
	e := newEnv(t)
	if _, errOut, code := e.run("meta", "mouse-autohide-toggle"); code != 2 || !strings.Contains(errOut, "not applied") {
		t.Fatalf("not applied: code=%d stderr=%s", code, errOut)
	}
	if _, _, code := e.run("meta", "mouse-autohide-toggle", "--is", "on"); code != 1 {
		t.Fatalf("--is on before apply: code=%d", code)
	}
	e.mustRun("switch", "mouse-autohide-toggle", "on", "-q")
	cases := []struct {
		args []string
		code int
		err  string
	}{
		{[]string{"meta", "mouse-autohide-toggle", "--is", "on"}, 0, ""},
		{[]string{"meta", "mouse-autohide-toggle", "--is", "off"}, 1, ""},
		{[]string{"meta", "mouse-autohide-toggle", "--is", "nope"}, 2, `unknown state "nope"`},
		{[]string{"meta", "nope"}, 2, `unknown switch "nope"`},
		{[]string{"meta", "mouse-autohide-toggle", "nokey"}, 2, `no metadata key "nokey"`},
		{[]string{"meta"}, 1, "accepts between 1 and 2 arg(s)"},
	}
	for _, c := range cases {
		out, errOut, code := e.run(c.args...)
		if code != c.code || !strings.Contains(errOut, c.err) || (c.code == 0 && out != "") {
			t.Errorf("%v: code=%d out=%q err=%q", c.args, code, out, errOut)
		}
	}
}

func TestMeta_PerFile(t *testing.T) {
	e := newEnv(t)
	e.mustRun("switch", "mouse-autohide-toggle", "on")
	var files []map[string]any
	if err := json.Unmarshal([]byte(e.mustRun("meta", "mouse-autohide-toggle", "--per-file")), &files); err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0]["file"] != e.target || files[0]["present"] != true {
		t.Fatalf("per-file = %v", files)
	}
}

func TestSwitch_Errors(t *testing.T) {
	e := newEnv(t)
	e.addSwitch("switches/broken.json", `{"states": []}`)
	cases := []struct {
		args []string
		err  string
	}{
		{[]string{"switch", "mouse-autohide-toggle", "maybe"}, `unknown state "maybe"`},
		{[]string{"switch", "mouse-autohide-toggle"}, "expected exactly one state (on, off)"},
		{[]string{"switch", "mouse-autohide", "on"}, "Did you mean this?\n\tmouse-autohide-toggle"},
		{[]string{"switch", "broken", "on"}, `switch "broken" has configuration errors`},
		{[]string{"switch", "mouse-autohide-toggle", "on", "--example-key", ""}, `"example-key" is required`},
		{[]string{"switch", "mouse-autohide-toggle", "on", "--bogus", "x"}, "unknown flag: --bogus"},
	}
	for _, c := range cases {
		if _, errOut, code := e.run(c.args...); code == 0 || !strings.Contains(errOut, c.err) {
			t.Errorf("%v: code=%d stderr=%q", c.args, code, errOut)
		}
	}
	if e.targetContent() != "input {}\n" {
		t.Fatal("target modified by failing commands")
	}
}

func TestSwitch_HelpAndDryRun(t *testing.T) {
	e := newEnv(t)
	help := e.mustRun("switch", "--help")
	if !strings.Contains(help, "mouse-autohide-toggle Hide the cursor after inactivity") {
		t.Fatalf("switch help lacks switch list:\n%s", help)
	}
	if got := e.mustRun("switch"); got != help {
		t.Fatal("bare 'switch' should print help")
	}
	sub := e.mustRun("switch", "mouse-autohide-toggle", "--help")
	for _, want := range []string{"States: on, off", e.target, "--reason string", "why the switch was flipped", "--example-key string", "--dry-run"} {
		if !strings.Contains(sub, want) {
			t.Errorf("switch help missing %q:\n%s", want, sub)
		}
	}
	out := e.mustRun("switch", "mouse-autohide-toggle", "on", "--dry-run")
	if !strings.Contains(out, "would update: "+e.target) || e.targetContent() != "input {}\n" {
		t.Fatalf("dry-run output:\n%s", out)
	}
}

func TestList(t *testing.T) {
	e := newEnv(t)
	e.addSwitch("switches/broken.json", `{`)
	out := e.mustRun("list")
	for _, want := range []string{"SWITCH", "STATE", "mouse-autohide-toggle  (not applied)  on|off", "broken", "(invalid)"} {
		if !strings.Contains(out, want) {
			t.Errorf("list missing %q:\n%s", want, out)
		}
	}
	e.mustRun("switch", "mouse-autohide-toggle", "off")
	var entries []listEntry
	if err := json.Unmarshal([]byte(e.mustRun("list", "--json")), &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[1].Name != "mouse-autohide-toggle" || entries[1].State != "off" || entries[0].Status != "invalid" {
		t.Fatalf("entries = %+v", entries)
	}
	empty := &env{t: t, cfgDir: t.TempDir()}
	if _, errOut, code := empty.run("list"); code != 0 || !strings.Contains(errOut, "no switches found") {
		t.Fatalf("empty list: %d %s", code, errOut)
	}
}

func TestVerify(t *testing.T) {
	e := newEnv(t)
	out, errOut, code := e.run("verify")
	if code != 0 || out != "" || !strings.Contains(errOut, "checked 1 switch(es)") {
		t.Fatalf("clean verify: code=%d out=%q err=%q", code, out, errOut)
	}
	e.addSwitch("switches/mouse-autohide-toggle.jsonc", mouseJSON)
	e.addSwitch("switches/bad.json", "{\n  \"states\": [\"on\"],\n  \"files\": [{\"file\": \"/nonexistent/x\", \"values\": [{\"state\": \"off\", \"value\": \"\"}]}]\n}")
	out, _, code = e.run("verify")
	for _, want := range []string{
		`switch name "mouse-autohide-toggle" is defined by multiple files`,
		"bad.json:3:", `state "off" is not declared`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("verify output missing %q:\n%s", want, out)
		}
	}
	if code != 1 {
		t.Fatalf("verify with errors: code=%d", code)
	}
	out, _, _ = e.run("verify", "bad", "--json")
	var diags []map[string]any
	if err := json.Unmarshal([]byte(out), &diags); err != nil {
		t.Fatal(err)
	}
	for _, d := range diags {
		if d["switch"] != "bad" {
			t.Fatalf("filter by name leaked %v", d)
		}
	}
	if _, errOut, code := e.run("verify", "nope"); code == 0 || !strings.Contains(errOut, `unknown switch "nope"`) {
		t.Fatalf("unknown name: %d %s", code, errOut)
	}
}

func TestVerify_TargetFiles(t *testing.T) {
	e := newEnv(t)
	e.addSwitch("switches/missing.json", `{"states": ["a"], "files": [{"file": "/nonexistent/f", "values": [{"state": "a", "value": ""}]}]}`)
	e.mustRun("switch", "mouse-autohide-toggle", "on")
	e.writeTarget(strings.Replace(e.targetContent(), "400", "999", 1))
	out, _, code := e.run("verify")
	if code != 0 || !strings.Contains(out, "target file does not exist") || !strings.Contains(out, "does not match the configured value") {
		t.Fatalf("code=%d out:\n%s", code, out)
	}
	if out, _, _ := e.run("verify", "--no-files"); out != "" {
		t.Fatalf("--no-files should skip target checks: %s", out)
	}
	e.writeTarget("// begin servitor managed -- mouse-autohide-toggle\n")
	if out, _, code := e.run("verify"); code != 1 || !strings.Contains(out, "has no end marker") {
		t.Fatalf("malformed: code=%d out=%s", code, out)
	}
}

func TestResolveConfigDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	t.Setenv(EnvConfig, "")
	cases := map[string][]string{
		"/xdg/servitor": {"switch", "x"},
		"/a":            {"--config", "/a", "switch"},
		"/b":            {"switch", "--config=/b"},
		"/c":            {"-c", "/c"},
		"/d":            {"-c/d"},
		"/e":            {"-c=/e"},
	}
	for want, args := range cases {
		if got := resolveConfigDir(args); got != want {
			t.Errorf("%v: got %s want %s", args, got, want)
		}
	}
	if got := resolveConfigDir([]string{"-c", "/first", "--config", "/last"}); got != "/last" {
		t.Errorf("last occurrence must win, got %s", got)
	}
	if got := resolveConfigDir([]string{"--", "--config", "/x"}); got != "/xdg/servitor" {
		t.Errorf("args after -- must be ignored, got %s", got)
	}
	t.Setenv(EnvConfig, "/env")
	if got := resolveConfigDir(nil); got != "/env" {
		t.Errorf("env: got %s", got)
	}
	if got := resolveConfigDir([]string{"-c", "/flag"}); got != "/flag" {
		t.Errorf("flag must win over env, got %s", got)
	}
}

func TestRootHelpAndVersion(t *testing.T) {
	e := newEnv(t)
	help := e.mustRun("--help")
	for _, want := range []string{"begin servitor managed", "switch", "meta", "verify", "list", "completion", "--config"} {
		if !strings.Contains(help, want) {
			t.Errorf("root help missing %q", want)
		}
	}
	if out := e.mustRun("--version"); !strings.Contains(out, "servitor version dev") {
		t.Fatalf("version: %q", out)
	}
}
