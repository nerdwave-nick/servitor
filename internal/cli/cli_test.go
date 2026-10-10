package cli

import (
	"bytes"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type env struct {
	t              *testing.T
	cfgDir, target string
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
	t.Setenv("SERVITOR_LIBRARIUM", "")
	t.Setenv("SERVITOR_CONFIG", "")
	t.Setenv("SERVITOR_NO_GRIMDARK", "")
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

func TestInvokeAndAugury_UserScenario(t *testing.T) {
	e := newEnv(t)
	out := e.mustRun("invoke", "mouse-autohide-toggle", "on", "--reason", "gaming remnant")
	if !strings.Contains(out, "+++ Rite mouse-autohide-toggle performed: (dormant) → on +++") ||
		!strings.Contains(out, "sanctified") || !strings.Contains(out, "The Omnissiah is pleased.") {
		t.Fatalf("summary: %s", out)
	}
	if !strings.Contains(e.targetContent(), `// begin servitor managed -- mouse-autohide-toggle -- state|on example-key|example-value reason|"gaming remnant"`) {
		t.Fatalf("target:\n%s", e.targetContent())
	}
	want := "{\n  \"state\": \"on\",\n  \"example-key\": \"example-value\",\n  \"reason\": \"gaming remnant\"\n}\n"
	if got := e.mustRun("augury", "mouse-autohide-toggle"); got != want {
		t.Fatalf("augury:\n%s\nwant\n%s", got, want)
	}

	e.mustRun("invoke", "mouse-autohide-toggle", "off")
	want = "{\n  \"state\": \"off\",\n  \"example-key\": \"other-value\"\n}\n"
	if got := e.mustRun("augury", "mouse-autohide-toggle"); got != want {
		t.Fatalf("augury after off:\n%s", got)
	}
	if got := e.mustRun("augury", "mouse-autohide-toggle", "state"); got != "off\n" {
		t.Fatalf("augury state = %q", got)
	}
	if got := e.mustRun("augury", "mouse-autohide-toggle", "reason"); got != "\n" {
		t.Fatalf("uninscribed key should print empty line, got %q", got)
	}
}

func TestAugury_IsAndExitCodes(t *testing.T) {
	e := newEnv(t)
	if _, errOut, code := e.run("augury", "mouse-autohide-toggle"); code != 2 || !strings.Contains(errOut, "lies dormant") {
		t.Fatalf("dormant: code=%d stderr=%s", code, errOut)
	}
	if _, _, code := e.run("augury", "mouse-autohide-toggle", "--is", "on"); code != 1 {
		t.Fatalf("--is on before invoke: code=%d", code)
	}
	e.mustRun("invoke", "mouse-autohide-toggle", "on", "-s")
	cases := []struct {
		args []string
		code int
		err  string
	}{
		{[]string{"augury", "mouse-autohide-toggle", "--is", "on"}, 0, ""},
		{[]string{"augury", "mouse-autohide-toggle", "--is", "off"}, 1, ""},
		{[]string{"augury", "mouse-autohide-toggle", "--is", "nope"}, 2, `knows no aspect "nope"`},
		{[]string{"augury", "nope"}, 2, `no rite named "nope"`},
		{[]string{"augury", "mouse-autohide-toggle", "nokey"}, 2, `bears no inscription "nokey"`},
		{[]string{"augury"}, 1, "accepts between 1 and 2 arg(s)"},
	}
	for _, c := range cases {
		out, errOut, code := e.run(c.args...)
		if code != c.code || !strings.Contains(errOut, c.err) || (c.code == 0 && out != "") {
			t.Errorf("%v: code=%d out=%q err=%q", c.args, code, out, errOut)
		}
	}
}

func TestAugury_PerFile(t *testing.T) {
	e := newEnv(t)
	e.mustRun("invoke", "mouse-autohide-toggle", "on")
	var files []map[string]any
	if err := json.Unmarshal([]byte(e.mustRun("augury", "mouse-autohide-toggle", "--per-file")), &files); err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0]["file"] != e.target || files[0]["present"] != true {
		t.Fatalf("per-file = %v", files)
	}
}

func TestInvoke_Errors(t *testing.T) {
	e := newEnv(t)
	e.addSwitch("switches/broken.json", `{"states": []}`)
	cases := []struct {
		args []string
		err  string
	}{
		{[]string{"invoke", "mouse-autohide-toggle", "maybe"}, `servitor ✠ the rite "mouse-autohide-toggle" knows no aspect "maybe"`},
		{[]string{"invoke", "mouse-autohide-toggle"}, "the rite demands exactly one aspect (on, off)"},
		{[]string{"invoke", "mouse-autohide", "on"}, "no rite named \"mouse-autohide\" is recorded in the Librarium"},
		{[]string{"invoke", "mouse-autohide", "on"}, "Perhaps you sought:\n\tmouse-autohide-toggle"},
		{[]string{"invoke", "broken", "on"}, `the rite "broken" is tainted by heresy`},
		{[]string{"invoke", "mouse-autohide-toggle", "on", "--example-key", ""}, `"example-key" is required`},
		{[]string{"invoke", "mouse-autohide-toggle", "on", "--bogus", "x"}, "unknown rune: --bogus"},
	}
	for _, c := range cases {
		if _, errOut, code := e.run(c.args...); code == 0 || !strings.Contains(errOut, c.err) {
			t.Errorf("%v: code=%d stderr=%q", c.args, code, errOut)
		}
	}
	if e.targetContent() != "input {}\n" {
		t.Fatal("vessel modified by failing invocations")
	}
}

func TestInvoke_HelpAndForesee(t *testing.T) {
	e := newEnv(t)
	help := e.mustRun("invoke", "--help")
	if !strings.Contains(help, "mouse-autohide-toggle Hide the cursor after inactivity") {
		t.Fatalf("invoke help lacks rite list:\n%s", help)
	}
	if got := e.mustRun("invoke"); got != help {
		t.Fatal("bare 'invoke' should print help")
	}
	sub := e.mustRun("invoke", "mouse-autohide-toggle", "--help")
	for _, want := range []string{"Aspects: on, off", e.target, "--reason string", "why the switch was flipped",
		"--example-key string", "-f, --foresee", "-s, --silence"} {
		if !strings.Contains(sub, want) {
			t.Errorf("invoke help missing %q:\n%s", want, sub)
		}
	}
	for _, flag := range []string{"--foresee", "-f"} {
		out := e.mustRun("invoke", "mouse-autohide-toggle", "on", flag)
		if !strings.Contains(out, "The augury foresees changes to "+e.target) || e.targetContent() != "input {}\n" {
			t.Fatalf("%s output:\n%s", flag, out)
		}
	}
}

func TestInvoke_Silence(t *testing.T) {
	e := newEnv(t)
	for _, flag := range []string{"--silence", "-s"} {
		if out := e.mustRun("invoke", "mouse-autohide-toggle", "on", flag); out != "" {
			t.Fatalf("%s should suppress the summary, got %q", flag, out)
		}
	}
}

func TestCensus(t *testing.T) {
	e := newEnv(t)
	e.addSwitch("switches/broken.json", `{`)
	out := e.mustRun("census")
	for _, want := range []string{`RITE\s+ASPECT\s+ASPECTS\s+PURPOSE`, `mouse-autohide-toggle\s+\(dormant\)\s+on\|off`, `broken\s+\(heretical\)`} {
		if !regexp.MustCompile(want).MatchString(out) {
			t.Errorf("census missing %s:\n%s", want, out)
		}
	}
	e.mustRun("invoke", "mouse-autohide-toggle", "off")
	for _, flag := range []string{"--binharic", "--json"} {
		var entries []listEntry
		if err := json.Unmarshal([]byte(e.mustRun("census", flag)), &entries); err != nil {
			t.Fatalf("%s: %v", flag, err)
		}
		if len(entries) != 2 || entries[1].Name != "mouse-autohide-toggle" || entries[1].State != "off" || entries[0].Status != "invalid" {
			t.Fatalf("%s entries = %+v", flag, entries)
		}
	}
	empty := &env{t: t, cfgDir: t.TempDir()}
	if _, errOut, code := empty.run("census"); code != 0 || !strings.Contains(errOut, "holds no rites") {
		t.Fatalf("empty census: %d %s", code, errOut)
	}
}

func TestInquisition(t *testing.T) {
	e := newEnv(t)
	out, errOut, code := e.run("inquisition")
	if code != 0 || out != "" || !strings.Contains(errOut, "+++ The Inquisition examined 1 rite(s)") {
		t.Fatalf("clean inquisition: code=%d out=%q err=%q", code, out, errOut)
	}
	e.addSwitch("switches/mouse-autohide-toggle.jsonc", mouseJSON)
	e.addSwitch("switches/bad.json", "{\n  \"states\": [\"on\"],\n  \"files\": [{\"file\": \"/nonexistent/x\", \"values\": [{\"state\": \"off\", \"value\": \"\"}]}]\n}")
	out, _, code = e.run("inquisition")
	for _, want := range []string{
		`switch name "mouse-autohide-toggle" is defined by multiple files`,
		"bad.json:3:", ": heresy: ", `state "off" is not declared`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("inquisition output missing %q:\n%s", want, out)
		}
	}
	if code != 1 {
		t.Fatalf("inquisition with heresy: code=%d", code)
	}
	for _, flag := range []string{"--binharic", "--json"} {
		out, _, _ = e.run("inquisition", "bad", flag)
		var diags []map[string]any
		if err := json.Unmarshal([]byte(out), &diags); err != nil || len(diags) == 0 {
			t.Fatalf("%s: %v %q", flag, err, out)
		}
		for _, d := range diags {
			if d["switch"] != "bad" {
				t.Fatalf("filter by name leaked %v", d)
			}
		}
	}
	if _, errOut, code := e.run("inquisition", "nope"); code == 0 || !strings.Contains(errOut, `no rite named "nope"`) {
		t.Fatalf("unknown name: %d %s", code, errOut)
	}
}

func TestInquisition_Vessels(t *testing.T) {
	e := newEnv(t)
	e.addSwitch("switches/missing.json", `{"states": ["a"], "files": [{"file": "/nonexistent/f", "values": [{"state": "a", "value": ""}]}]}`)
	e.mustRun("invoke", "mouse-autohide-toggle", "on")
	e.writeTarget(strings.Replace(e.targetContent(), "400", "999", 1))
	out, _, code := e.run("inquisition")
	if code != 0 || !strings.Contains(out, "the vessel does not exist") || !strings.Contains(out, "tainted by unsanctioned hands") ||
		!strings.Contains(out, ": impurity: ") {
		t.Fatalf("code=%d out:\n%s", code, out)
	}
	if out, _, _ := e.run("inquisition", "--spare-vessels"); out != "" {
		t.Fatalf("--spare-vessels should skip vessel checks: %s", out)
	}
	e.writeTarget("// begin servitor managed -- mouse-autohide-toggle\n")
	if out, _, code := e.run("inquisition"); code != 1 || !strings.Contains(out, "has no end marker") {
		t.Fatalf("malformed: code=%d out=%s", code, out)
	}
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
