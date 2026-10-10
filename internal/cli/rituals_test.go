package cli

import (
	"bytes"
	"encoding/json/v2"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestAugury_IsAndExitCodes(t *testing.T) {
	e := newEnv(t)
	if out := e.mustRun("augury", "mouse-autohide-toggle"); !strings.Contains(out, `"standing": "dormant"`) {
		t.Fatalf("dormant augury:\n%s", out)
	}
	if _, _, code := e.run("augury", "mouse-autohide-toggle", "--is", "on"); code != 1 {
		t.Fatalf("--is on before invoke: code=%d", code)
	}
	e.mustRun("invoke", "mouse-autohide-toggle", "on", "-s")
	e.addRite("rites/broken.json", `{"pattern": "Mark I", "aspects": []}`)
	cases := []struct {
		args []string
		code int
		err  string
	}{
		{[]string{"augury", "mouse-autohide-toggle", "--is", "on"}, 0, ""},
		{[]string{"augury", "mouse-autohide-toggle", "--is", "off"}, 1, ""},
		{[]string{"augury", "mouse-autohide-toggle", "--is", "nope"}, 2, `knows no aspect "nope"`},
		{[]string{"augury", "mouse-autohide-toggle", "aspect", "--is", "on"}, 2, "--is"},
		{[]string{"augury", "nope"}, 2, `no rite named "nope"`},
		{[]string{"augury", "broken", "--is", "on"}, 2, `the rite "broken" is tainted by heresy`},
		{[]string{"augury", "mouse-autohide-toggle", "nokey"}, 2, `bears no key "nokey"`},
		{[]string{"augury"}, 2, "the augury demands one rite"},
		{[]string{"augury", "mouse-autohide-toggle", "aspect", "more"}, 2, "the augury demands one rite"},
		{[]string{"augury", "mouse-autohide-toggle", "--per-file"}, 2, "unknown rune: --per-file"},
	}
	for _, c := range cases {
		out, errOut, code := e.run(c.args...)
		if code != c.code || !strings.Contains(errOut, c.err) || (c.code != 0 && out != "") || (c.code == 0 && out != "") {
			t.Errorf("%v: code=%d out=%q err=%q", c.args, code, out, errOut)
		}
	}
	if got := e.mustRun("augury", "broken", "standing"); got != "heretical\n" {
		t.Fatalf("heretical standing %q", got)
	}
}

func TestAugury_SpeaksTheJSONOfTheCodex(t *testing.T) {
	e := newEnv(t)
	e.mustRun("invoke", "mouse-autohide-toggle", "on", "-s", "--reason", "gaming remnant")
	out := e.mustRun("augury", "mouse-autohide-toggle")
	want := regexp.MustCompile(`^\{\n  "rite": "mouse-autohide-toggle",\n  "aspect": "on",\n  "standing": "performed",\n` +
		`  "desecrated": false,\n  "inscriptions": \{\n    "example-key": "example-value",\n    "reason": "gaming remnant"\n  \},\n` +
		`  "last_rite": \{\n    "verdict": "triumph",\n    "at": "[0-9T:Z-]+"\n  \},\n` +
		`  "omens": \[\n    \{\n      "verse": 1,\n      "sanctum": "` + regexp.QuoteMeta(e.target) + `",\n      "aspect": "on"\n    \}\n  \],\n` +
		`  "taint": \[\]\n\}\n$`)
	if !want.MatchString(out) {
		t.Fatalf("augury:\n%s", out)
	}
}

func TestCensus(t *testing.T) {
	e := newEnv(t)
	broken := e.addRite("rites/broken.json", `{`)
	out := e.mustRun("census")
	for _, want := range []string{`RITE\s+ASPECT\s+STANDING\s+ASPECTS\s+LAST RITE\s+PURPOSE`,
		`mouse-autohide-toggle\s+·\s+dormant\s+on\|off\s+·\s+Hide the cursor`, `broken\s+·\s+heretical`} {
		if !regexp.MustCompile(want).MatchString(out) {
			t.Errorf("census missing %s:\n%s", want, out)
		}
	}
	e.mustRun("invoke", "mouse-autohide-toggle", "off", "-s")
	if out := e.mustRun("census"); !regexp.MustCompile(`mouse-autohide-toggle\s+off\s+performed\s+on\|off\s+triumph `).MatchString(out) {
		t.Errorf("census after invocation:\n%s", out)
	}
	for _, flag := range []string{"--binharic", "--json"} {
		var entries []map[string]any
		if err := json.Unmarshal([]byte(e.mustRun("census", flag)), &entries); err != nil {
			t.Fatalf("%s: %v", flag, err)
		}
		if len(entries) != 2 || entries[0]["rite"] != "broken" || entries[0]["standing"] != "heretical" ||
			entries[0]["recorded_in"] != broken || entries[1]["aspect"] != "off" || entries[1]["desecrated"] != false {
			t.Fatalf("%s entries = %+v", flag, entries)
		}
		for _, key := range []string{"rite", "aspect", "standing", "aspects", "purpose", "desecrated", "last_rite", "recorded_in"} {
			if _, ok := entries[1][key]; !ok {
				t.Errorf("%s: no key %q in %v", flag, key, entries[1])
			}
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
	e.addRite("rites/mouse-autohide-toggle.jsonc", mouseJSON)
	e.addRite("rites/bad.json", "{\n  \"pattern\": \"Mark I\",\n  \"aspects\": [\"on\"],\n  \"colour\": 1,\n  \"liturgy\": []\n}")
	out, _, code = e.run("inquisition")
	for _, want := range []string{
		`the rite "mouse-autohide-toggle" is recorded in more than one scripture`,
		"bad.json:4:3: heresy: ", `"colour"`,
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
		var found []map[string]any
		if err := json.Unmarshal([]byte(out), &found); err != nil || len(found) == 0 {
			t.Fatalf("%s: %v %q", flag, err, out)
		}
		for _, f := range found {
			if f["rite"] != "bad" || f["judgement"] != "heresy" || f["line"] != 4.0 || f["column"] != 3.0 ||
				!strings.HasSuffix(f["scripture"].(string), "bad.json") || f["denunciation"] == "" {
				t.Fatalf("%s: %v", flag, f)
			}
		}
	}
	if _, errOut, code := e.run("inquisition", "nope"); code == 0 || !strings.Contains(errOut, `no rite named "nope"`) {
		t.Fatalf("unknown name: %d %s", code, errOut)
	}
}

func TestInquisition_Vessels(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/missing.json", `{"pattern": "Mark I", "aspects": ["a"], "liturgy": [{"sanctum": "/nonexistent/f", "scripture": ""}]}`)
	e.mustRun("invoke", "mouse-autohide-toggle", "on", "-s")
	e.writeTarget(strings.Replace(e.targetContent(), "400", "999", 1))
	out, _, code := e.run("inquisition")
	if code != 0 || !strings.Contains(out, "no vessel stands at /nonexistent/f") || !strings.Contains(out, "is tainted") ||
		!strings.Contains(out, ": impurity: ") {
		t.Fatalf("code=%d out:\n%s", code, out)
	}
	if out, _, _ := e.run("inquisition", "--spare-vessels"); out != "" {
		t.Fatalf("--spare-vessels should skip vessel checks: %s", out)
	}
	e.writeTarget("// +++ begin of sanctum mouse-autohide-toggle -- aspect|on +++\n")
	if out, _, code := e.run("inquisition"); code != 1 || !strings.Contains(out, "never sealed") {
		t.Fatalf("broken markers: code=%d out=%s", code, out)
	}
}

// TestInquisition_TheExamplesArePure: the example Librarium shown to the
// faithful passes the inquisition, its vessels spared, and the scroll its
// theme rite recites stands ready to be executed.
func TestInquisition_TheExamplesArePure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var out, errb bytes.Buffer
	code := Execute([]string{"--librarium", "../../examples", "inquisition", "--spare-vessels"}, &out, &errb)
	if code != 0 || out.String() != "" || !strings.Contains(errb.String(), "examined 2 rite(s)") {
		t.Fatalf("code=%d\nstdout: %s\nstderr: %s", code, out.String(), errb.String())
	}
	if info, err := os.Stat("../../examples/rites/scripts/recite-hooks"); err != nil || info.Mode()&0o111 == 0 {
		t.Fatalf("the scroll of the theme rite: %v %v", info, err)
	}
}

// elderJSON is a rite in the elder form that knew no pattern: its vessels
// are "files", its aspects "states".
const elderJSON = `// +++ an elder rite +++
{
  "description": "Hide the cursor after inactivity",
  "states": ["on", "off"],
  "files": [{"file": "$TARGET", "values": [{"state": "on", "value": "hide"}, {"state": "off", "value": ""}]}],
}`

func TestInquisition_ElderScriptureIsHeresy(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/elder.json", elderJSON)
	out, _, code := e.run("inquisition", "elder")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if code != 1 || len(lines) != 1 {
		t.Fatalf("one heresy and nothing else expected: code=%d out:\n%s", code, out)
	}
	for _, want := range []string{
		"elder.json:2:1: heresy: ", `bears no "pattern"`, `"files"`, `"states"`, "elder",
		"nothing of it will be converted", `"Mark I"`, `"liturgy"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("denunciation missing %q:\n%s", want, out)
		}
	}
	if out, _, _ := e.run("inquisition", "elder", "--binharic"); !strings.Contains(out, `"judgement": "heresy"`) ||
		!strings.Contains(out, "nothing of it will be converted") {
		t.Errorf("binharic denunciation: %s", out)
	}
	before := e.targetContent()
	if _, errOut, code := e.run("invoke", "elder", "on"); code == 0 || !strings.Contains(errOut, "tainted by heresy") {
		t.Fatalf("an elder rite must not be invoked: code=%d err=%s", code, errOut)
	}
	if e.targetContent() != before || len(e.chronicleLines()) != 0 {
		t.Fatal("an elder rite touched its vessel or the chronicle")
	}
	if out := e.mustRun("census", "--binharic"); !strings.Contains(out, `"standing": "heretical"`) {
		t.Fatalf("census: %s", out)
	}
}
