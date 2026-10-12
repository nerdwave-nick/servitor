package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestExalt_RefusesWithoutTheMachinesAuthority: the hidden mode of the
// exalted servitor truly checks that it bears the machine's own authority
// (no trial runs as root, so it refuses here with the death-mark 2), and it
// is roused before the servitor would read the world of the invoker sudo
// names: the exalted servitor never takes SUDO_USER's home.
func TestExalt_RefusesWithoutTheMachinesAuthority(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("the trials run as root; the real refusal cannot be seen")
	}
	keepWorld(t)
	asRoot(t)
	t.Setenv("SUDO_USER", "adept")
	t.Setenv("SUDO_HOME", "/home/adept")
	t.Setenv("XDG_STATE_HOME", "/kept/state")
	home := os.Getenv("HOME")
	var out, errb bytes.Buffer
	code := Execute([]string{"__exalt", "hallow", "0123456789ab", "on"}, &out, &errb)
	if code != 2 || out.Len() != 0 || !strings.Contains(errb.String(), "only with the machine's own authority") {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, out.String(), errb.String())
	}
	if got := os.Getenv("HOME"); got != home {
		t.Errorf("the exalted servitor took the home %q", got)
	}
	if got := os.Getenv("XDG_STATE_HOME"); got != "/kept/state" {
		t.Errorf("the exalted servitor cast away XDG_STATE_HOME: %q", got)
	}
}

// TestExalt_IsNeverListed: the hidden mode stands in no help, no index of
// the codex and no completion.
func TestExalt_IsNeverListed(t *testing.T) {
	e := newEnv(t)
	for _, args := range [][]string{{"--help"}, {"expound"}, {"__complete", ""}, {"__complete", "__"}} {
		out, errOut, _ := e.run(args...)
		if strings.Contains(out+errOut, "__exalt") {
			t.Errorf("servitor %v names the hidden mode:\n%s%s", args, out, errOut)
		}
	}
}
