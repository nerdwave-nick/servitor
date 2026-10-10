package cli

import (
	"bytes"
	"os"
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

// themeJSON is a rite whose auspex, were it ever awakened, leaves a mark.
const themeJSON = `{"pattern": "Mark I", "purpose": "Color theme", "aspects": ["dark", "light", "auto"],
  "auspex": "touch $TARGET.auspex; echo dark",
  "liturgy": [{"sanctum": "$TARGET", "ward": "theme", "scripture": {"*": "{{aspect}}"}}]}`

func TestCompletion_Callbacks(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/theme.json", themeJSON)
	e.addRite("rites/broken.json", `{`)
	e.mustRun("invoke", "mouse-autohide-toggle", "off", "-s")

	cases := []struct {
		args []string
		want []string
	}{
		{[]string{"invoke", ""}, []string{"mouse-autohide-toggle", "theme"}},
		{[]string{"invoke", "th"}, []string{"theme"}},
		{[]string{"invoke", "theme", ""}, []string{"dark", "light", "auto"}},
		{[]string{"invoke", "mouse-autohide-toggle", ""}, []string{"on", "off"}},
		{[]string{"invoke", "mouse-autohide-toggle", "on", "--example-key", ""}, []string{"example-value", "other-value"}},
		{[]string{"invoke", "mouse-autohide-toggle", "on", "--reason", ""}, nil},
		{[]string{"invoke", "unknown", ""}, nil},
		{[]string{"augury", ""}, []string{"broken", "mouse-autohide-toggle", "theme"}},
		{[]string{"augury", "mouse-autohide-toggle", ""}, []string{"aspect", "standing", "desecrated", "former", "reason", "example-key"}},
		{[]string{"augury", "broken", ""}, []string{"aspect", "standing", "desecrated", "former"}},
		{[]string{"augury", "theme", "--is", ""}, []string{"dark", "light", "auto"}},
		{[]string{"inquisition", "theme", ""}, []string{"broken", "mouse-autohide-toggle"}},
	}
	for _, c := range cases {
		got, dir := e.complete(c.args...)
		if !slices.Equal(got, c.want) || dir != noFileComp {
			t.Errorf("complete %v = %v %s, want %v %s", c.args, got, dir, c.want, noFileComp)
		}
	}
	if _, err := os.Stat(e.target + ".auspex"); err == nil {
		t.Fatal("completion awoke an auspex")
	}
}

func TestCompletion_Runes(t *testing.T) {
	e := newEnv(t)
	cases := map[string][]string{
		"invoke mouse-autohide-toggle on --": {"--reason", "--example-key", "--foresee", "--silence"},
		"augury mouse-autohide-toggle --":    {"--is"},
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
		for _, forsaken := range []string{"--dry-run", "--quiet", "--config", "--no-grimdark", "--json", "--no-files", "--help", "--per-file"} {
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
	want := []string{"augury", "census", "cogitator", "completion", "expound", "inquisition", "invoke"}
	if !slices.Equal(got, want) {
		t.Fatalf("rituals = %v, want %v", got, want)
	}
}

func TestCompletion_DescriptionsAndTheCurrentAspect(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/theme.json", themeJSON)
	e.mustRun("invoke", "mouse-autohide-toggle", "on", "-s")
	e.mustRun("invoke", "theme", "light", "-s")
	if err := os.Remove(e.target + ".auspex"); err != nil {
		t.Fatal(err) // the invocation heeded the auspex
	}
	for args, want := range map[string]string{
		"invoke mouse-autohide-toggle":                  "on\tcurrent aspect\n",
		"augury mouse-autohide-toggle --is":             "on\tcurrent aspect\n",
		"invoke theme":                                  "light\tcurrent aspect\n",
		"augury":                                        "mouse-autohide-toggle\tHide the cursor after inactivity\n",
		"augury mouse-autohide-toggle":                  "reason\twhy the rite was invoked\n",
		"invoke mouse-autohide-toggle on --example-key": "example-value\tdecreed for on\n",
	} {
		out := e.mustRun(append(append([]string{"__complete"}, strings.Fields(args)...), "")...)
		if !strings.Contains(out, want) {
			t.Errorf("__complete %s lacks %q:\n%s", args, want, out)
		}
	}
	if _, err := os.Stat(e.target + ".auspex"); err == nil {
		t.Fatal("completion awoke an auspex")
	}
}

func TestCompletion_LibrariumRuneIsHonoured(t *testing.T) {
	e := newEnv(t)
	other := newEnv(t)
	other.addRite("rites/only-here.json", `{"pattern": "Mark I", "aspects": ["x"], "liturgy": []}`)
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
