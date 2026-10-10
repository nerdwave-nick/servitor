package cli

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

// complete runs cobra's hidden __complete command and returns the candidate
// values (descriptions stripped) and the directive line.
func (e *env) complete(args ...string) (values []string, directive string) {
	e.t.Helper()
	out := e.mustRun(append([]string{"__complete"}, args...)...)
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.HasPrefix(line, ":") {
			directive = line
			continue
		}
		v, _, _ := strings.Cut(line, "\t")
		values = append(values, v)
	}
	return values, directive
}

const noFileComp = ":4"

func TestCompletion_Callbacks(t *testing.T) {
	e := newEnv(t)
	e.addSwitch("switches/theme.json", `{"description": "Color theme", "states": ["dark", "light", "auto"],
	  "files": [{"file": "$TARGET", "values": [{"state": "dark", "value": "d"}, {"state": "light", "value": "l"}, {"state": "auto", "value": "a"}]}]}`)
	e.mustRun("invoke", "mouse-autohide-toggle", "off")

	cases := []struct {
		args []string
		want []string
	}{
		{[]string{"invoke", ""}, []string{"mouse-autohide-toggle", "theme"}},
		{[]string{"invoke", "th"}, []string{"theme"}},
		{[]string{"invoke", "theme", ""}, []string{"dark", "light", "auto"}},
		{[]string{"invoke", "mouse-autohide-toggle", ""}, []string{"on", "off"}},
		{[]string{"invoke", "mouse-autohide-toggle", "on", "--example-key", ""}, []string{"example-value", "other-value"}},
		{[]string{"invoke", "unknown", ""}, nil},
		{[]string{"augury", ""}, []string{"mouse-autohide-toggle", "theme"}},
		{[]string{"augury", "mouse-autohide-toggle", ""}, []string{"state", "example-key", "reason"}},
		{[]string{"augury", "theme", "--is", ""}, []string{"dark", "light", "auto"}},
		{[]string{"inquisition", "theme", ""}, []string{"mouse-autohide-toggle"}},
	}
	for _, c := range cases {
		got, dir := e.complete(c.args...)
		if !slices.Equal(got, c.want) || dir != noFileComp {
			t.Errorf("complete %v = %v %s, want %v %s", c.args, got, dir, c.want, noFileComp)
		}
	}
}

func TestCompletion_Runes(t *testing.T) {
	e := newEnv(t)
	cases := map[string][]string{
		"invoke mouse-autohide-toggle on --": {"--reason", "--example-key", "--foresee", "--silence"},
		"census --":                          {"--binharic"},
		"inquisition --":                     {"--binharic", "--spare-vessels"},
		"--":                                 {"--version"},
	}
	for line, want := range cases {
		got, _ := e.complete(strings.Fields(line)...)
		for _, w := range want {
			if !slices.Contains(got, w) {
				t.Errorf("complete %q missing %s: %v", line, w, got)
			}
		}
		for _, forsaken := range []string{"--dry-run", "--quiet", "--config", "--no-grimdark", "--json", "--no-files", "--help"} {
			if slices.Contains(got, forsaken) {
				t.Errorf("complete %q offers %s: %v", line, forsaken, got)
			}
		}
	}
}

func TestCompletion_LibrariumRune(t *testing.T) {
	newEnv(t)
	var out, errb bytes.Buffer
	if code := Execute([]string{"__complete", "-"}, &out, &errb); code != 0 {
		t.Fatalf("code=%d err=%s", code, errb.String())
	}
	for _, want := range []string{"--librarium\t", "-l\t"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("root rune completion missing %q:\n%s", want, out.String())
		}
	}
}

func TestCompletion_Rituals(t *testing.T) {
	e := newEnv(t)
	got, _ := e.complete("")
	want := []string{"augury", "census", "cogitator", "completion", "inquisition", "invoke"}
	if !slices.Equal(got, want) {
		t.Fatalf("rituals = %v, want %v", got, want)
	}
}

func TestCompletion_DescriptionsAndCurrentState(t *testing.T) {
	e := newEnv(t)
	e.mustRun("invoke", "mouse-autohide-toggle", "on")
	out := e.mustRun("__complete", "invoke", "mouse-autohide-toggle", "")
	if !strings.Contains(out, "on\tcurrent aspect\n") {
		t.Fatalf("current state not marked:\n%s", out)
	}
	out = e.mustRun("__complete", "augury", "")
	if !strings.Contains(out, "mouse-autohide-toggle\tHide the cursor after inactivity") {
		t.Fatalf("rite purpose missing:\n%s", out)
	}
}

func TestCompletion_LibrariumRuneIsHonoured(t *testing.T) {
	e := newEnv(t)
	other := newEnv(t)
	other.addSwitch("switches/only-here.json", `{"states": ["x"], "files": [{"file": "$TARGET", "values": [{"state": "x", "value": ""}]}]}`)
	got, _ := e.complete("--librarium", other.cfgDir, "invoke", "")
	if !slices.Contains(got, "only-here") {
		t.Fatalf("--librarium not used during completion: %v", got)
	}
}

func TestCompletion_Scripts(t *testing.T) {
	e := newEnv(t)
	for _, shell := range []string{"bash", "zsh", "fish"} {
		out := e.mustRun("completion", shell)
		if !strings.Contains(out, "__complete") || !strings.Contains(out, "servitor") {
			t.Errorf("%s completion script looks wrong", shell)
		}
	}
}
