package cli

import (
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
	e.mustRun("switch", "mouse-autohide-toggle", "off")

	cases := []struct {
		args []string
		want []string
	}{
		{[]string{"switch", ""}, []string{"mouse-autohide-toggle", "theme"}},
		{[]string{"profile", "th"}, []string{"theme"}},
		{[]string{"switch", "theme", ""}, []string{"dark", "light", "auto"}},
		{[]string{"switch", "mouse-autohide-toggle", ""}, []string{"on", "off"}},
		{[]string{"switch", "mouse-autohide-toggle", "on", "--example-key", ""}, []string{"example-value", "other-value"}},
		{[]string{"switch", "unknown", ""}, nil},
		{[]string{"meta", ""}, []string{"mouse-autohide-toggle", "theme"}},
		{[]string{"meta", "mouse-autohide-toggle", ""}, []string{"state", "example-key", "reason"}},
		{[]string{"meta", "theme", "--is", ""}, []string{"dark", "light", "auto"}},
		{[]string{"verify", "theme", ""}, []string{"mouse-autohide-toggle"}},
	}
	for _, c := range cases {
		got, dir := e.complete(c.args...)
		if !slices.Equal(got, c.want) || dir != noFileComp {
			t.Errorf("complete %v = %v %s, want %v %s", c.args, got, dir, c.want, noFileComp)
		}
	}
}

func TestCompletion_MetaKeyFlags(t *testing.T) {
	e := newEnv(t)
	got, _ := e.complete("switch", "mouse-autohide-toggle", "on", "--")
	for _, want := range []string{"--reason", "--example-key", "--dry-run", "--quiet"} {
		if !slices.Contains(got, want) {
			t.Errorf("flag completion missing %s: %v", want, got)
		}
	}
}

func TestCompletion_DescriptionsAndCurrentState(t *testing.T) {
	e := newEnv(t)
	e.mustRun("switch", "mouse-autohide-toggle", "on")
	out := e.mustRun("__complete", "switch", "mouse-autohide-toggle", "")
	if !strings.Contains(out, "on\tcurrent\n") {
		t.Fatalf("current state not marked:\n%s", out)
	}
	out = e.mustRun("__complete", "meta", "")
	if !strings.Contains(out, "mouse-autohide-toggle\tHide the cursor after inactivity") {
		t.Fatalf("switch description missing:\n%s", out)
	}
}

func TestCompletion_ConfigFlagIsHonoured(t *testing.T) {
	e := newEnv(t)
	other := newEnv(t)
	other.addSwitch("switches/only-here.json", `{"states": ["x"], "files": [{"file": "$TARGET", "values": [{"state": "x", "value": ""}]}]}`)
	got, _ := e.complete("--config", other.cfgDir, "switch", "")
	if !slices.Contains(got, "only-here") {
		t.Fatalf("--config not used during completion: %v", got)
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
