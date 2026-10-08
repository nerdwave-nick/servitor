package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// userExample is the configuration from the original feature request,
// including comments, trailing commas and string booleans.
const userExample = `{
  "states": ["on","off"],
  "files": [
    {
        "file": "~/.config/niri/util.kdl",
        "values": [
          {"state": "on", "value":"cursor {\n    hide-after-inactive-ms 400\n}", "meta": {"example-key": "example-value"}},
          {"state": "off", "value":"cursor {\n    hide-after-inactive-ms 400\n}", "meta": {"example-key": "other-value", "reason": ""}},
        ],
        "meta": {
            "reason": {"optional":"true"}, // optional as default
            "example-key": {"optional":"false"},
        }
    }
  ]
}`

func writeSwitch(t *testing.T, dir, rel, body string) string {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func findDiag(t *testing.T, set *Set, substr string) Diagnostic {
	t.Helper()
	for _, d := range set.Diags {
		if strings.Contains(d.Message, substr) {
			return d
		}
	}
	t.Fatalf("no diagnostic containing %q in %v", substr, set.Diags)
	return Diagnostic{}
}

func TestLoad_UserExampleIsValid(t *testing.T) {
	dir := t.TempDir()
	writeSwitch(t, dir, "rites/mouse-autohide-toggle.json", userExample)
	set := Load(dir)
	if set.Diags.HasErrors() {
		t.Fatalf("unexpected errors: %v", set.Diags)
	}
	sw := set.Switches["mouse-autohide-toggle"]
	if sw == nil {
		t.Fatalf("switch not loaded: %v", set.Names())
	}
	f := sw.Files[0]
	if f.Guard != "mouse-autohide-toggle" || f.Comment != "//" || f.CommentEnd != "" {
		t.Fatalf("defaults not applied: %+v", f)
	}
	if !f.Meta["example-key"].Required() || f.Meta["reason"].Required() {
		t.Fatal("optional flags parsed incorrectly")
	}
	home, _ := os.UserHomeDir()
	if got := sw.Target(&f); got != filepath.Join(home, ".config/niri/util.kdl") {
		t.Fatalf("target = %s", got)
	}
	if got := sw.MetaValues("example-key"); strings.Join(got, ",") != "example-value,other-value" {
		t.Fatalf("MetaValues = %v", got)
	}
}

func TestLoad_ValueAsLineArray(t *testing.T) {
	dir := t.TempDir()
	writeSwitch(t, dir, "rites/s.jsonc", `{"states":["a"],
		"files":[{"file":"/x.css","values":[{"state":"a","value":["l1","l2"]}]}]}`)
	set := Load(dir)
	sw := set.Switches["s"]
	if sw == nil {
		t.Fatalf("not loaded: %v", set.Diags)
	}
	if sw.Files[0].Values[0].Value != "l1\nl2" || sw.Files[0].CommentEnd != "*/" {
		t.Fatalf("unexpected: %+v", sw.Files[0])
	}
}

func TestLoad_EntryAndExitCommandsAreGone(t *testing.T) {
	dir := t.TempDir()
	writeSwitch(t, dir, "rites/s.json", `{"states":["a"],"exit":"echo hi","files":[]}`)
	findDiag(t, Load(dir), `unknown field "exit"`)
}

func TestLoad_SyntaxErrorHasLocation(t *testing.T) {
	dir := t.TempDir()
	writeSwitch(t, dir, "switches/broken.json", "{\n  \"states\": [\"on\",\n  \"files\": [\n}\n")
	set := Load(dir)
	d := findDiag(t, set, "syntax error")
	if d.Line != 3 || d.Col != 10 || !set.Broken["broken"] {
		t.Fatalf("diag = %+v", d)
	}
}

func TestLoad_DecodeErrorsHaveLocations(t *testing.T) {
	cases := map[string]struct {
		body     string
		msg      string
		line     int
		pointsAt string
	}{
		"unknown field": {"{\"states\":[\"a\"],\n  \"filez\": []}", `unknown field "filez"`, 2, ""},
		"bad bool": {"{\"states\":[\"a\"],\"files\":[{\"file\":\"/x\",\n\"meta\":{\"k\":{\"optional\":\"yes\"}}}]}",
			"/files/0/meta/k/optional", 2, ""},
		"bad type": {"{\n\"states\": \"on\"}", "/states", 2, ""},
	}
	for name, tc := range cases {
		dir := t.TempDir()
		writeSwitch(t, dir, "switches/s.json", tc.body)
		d := findDiag(t, Load(dir), tc.msg)
		if d.Line != tc.line || d.Severity != SevError {
			t.Errorf("%s: diag = %+v", name, d)
		}
	}
}

func TestLoad_SemanticErrors(t *testing.T) {
	dir := t.TempDir()
	writeSwitch(t, dir, "switches/s.json", `{
  "states": ["on", "on", "bad state"],
  "files": [{
    "file": "/tmp/x.kdl",
    "guard": "has space",
    "meta": {"state": {}, "bad key": {}, "req": {"optional": false}},
    "values": [
      {"state": "on", "value": "x", "meta": {"undeclared": "v", "state": "x"}},
      {"state": "nope", "value": "// begin servitor managed -- has space"}
    ]
  }]
}`)
	set := Load(dir)
	for _, want := range []string{
		`duplicate state "on"`, `invalid state "bad state"`,
		`invalid guard "has space"`, `metadata key "state" is reserved`, `invalid metadata key "bad key"`,
		`"undeclared" is not declared`, `"state" is set automatically`, `state "nope" is not declared`,
		`no value configured for state "bad state"`, `required metadata key "req" has no value`,
	} {
		findDiag(t, set, want)
	}
	if d := findDiag(t, set, `state "nope"`); d.Line != 9 {
		t.Fatalf("expected line 9, got %+v", d)
	}
	if len(set.Switches) != 0 || !set.Broken["s"] {
		t.Fatal("invalid switch must not be usable")
	}
}

func TestLoad_EmptyStatesAndFiles(t *testing.T) {
	dir := t.TempDir()
	writeSwitch(t, dir, "switches/s.json", `{}`)
	set := Load(dir)
	findDiag(t, set, "at least one state")
	findDiag(t, set, `at least one entry must be declared in "files"`)
}

func TestLoad_NameClashAcrossDirsAndExtensions(t *testing.T) {
	dir := t.TempDir()
	valid := `{"states":["a"],"files":[{"file":"/x","values":[{"state":"a","value":""}]}]}`
	writeSwitch(t, dir, "switches/dup.json", valid)
	writeSwitch(t, dir, "rites/dup.jsonc", valid)
	writeSwitch(t, dir, "switches/ok.json", strings.Replace(valid, "/x", "/y", 1))
	set := Load(dir)
	findDiag(t, set, `switch name "dup" is defined by multiple files`)
	if set.Switches["dup"] != nil || set.Switches["ok"] == nil {
		t.Fatalf("names = %v", set.Names())
	}
}

func TestLoad_GuardClashAcrossSwitches(t *testing.T) {
	dir := t.TempDir()
	body := `{"states":["a"],"files":[{"file":"/same.kdl","guard":"g","values":[{"state":"a","value":""}]}]}`
	writeSwitch(t, dir, "switches/one.json", body)
	writeSwitch(t, dir, "switches/two.json", body)
	set := Load(dir)
	d := findDiag(t, set, `guard "g" for /same.kdl is used by both`)
	if d.Line == 0 || len(set.Switches) != 0 {
		t.Fatalf("diag=%+v names=%v", d, set.Names())
	}
}

func TestLoad_InvalidFileNameAndRelativePathWarning(t *testing.T) {
	dir := t.TempDir()
	writeSwitch(t, dir, "switches/bad name.json", `{"states":["a"],"files":[{"file":"rel.conf","values":[{"state":"a","value":""}]}]}`)
	set := Load(dir)
	findDiag(t, set, `invalid switch name "bad name"`)
	if d := findDiag(t, set, "relative path"); d.Severity != SevWarning {
		t.Fatalf("expected warning, got %+v", d)
	}
}

func TestLoad_IgnoresOtherFilesAndMissingDir(t *testing.T) {
	dir := t.TempDir()
	writeSwitch(t, dir, "switches/notes.txt", "x")
	writeSwitch(t, dir, "switches/.hidden.json", "x")
	set := Load(dir)
	if len(set.Diags) != 0 || len(set.Switches) != 0 {
		t.Fatalf("unexpected: %v", set.Diags)
	}
	if set := Load(filepath.Join(dir, "missing")); len(set.Diags) != 0 {
		t.Fatalf("missing dir should be empty, got %v", set.Diags)
	}
}

func TestDefaultComment(t *testing.T) {
	for path, want := range map[string]string{"a.KDL": "//", "a.lua": "--", "a.toml": "#", "noext": "#"} {
		if got, _ := DefaultComment(path); got != want {
			t.Errorf("%s: got %q want %q", path, got, want)
		}
	}
}

func TestExpandPathAndDefaultDir(t *testing.T) {
	t.Setenv("FCC_TEST_DIR", "/env")
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	if got := ExpandPath("$FCC_TEST_DIR/a", "/base"); got != "/env/a" {
		t.Fatalf("got %s", got)
	}
	if got := ExpandPath("rel/a", "/base"); got != "/base/rel/a" {
		t.Fatalf("got %s", got)
	}
	if got := DefaultDir(); got != "/xdg/servitor" {
		t.Fatalf("got %s", got)
	}
}

func TestDiagnosticString(t *testing.T) {
	d := Diagnostic{File: "f.json", Line: 2, Col: 3, Severity: SevError, Message: "m"}
	if d.String() != "f.json:2:3: error: m" {
		t.Fatal(d.String())
	}
	if (Diagnostic{File: "f", Severity: SevWarning, Message: "m"}).String() != "f: warning: m" {
		t.Fatal("no-position format wrong")
	}
}
