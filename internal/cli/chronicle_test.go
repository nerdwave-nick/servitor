package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// heededChronicle runs servitor with args ending in the "probe" ritual and
// returns the chronicle that ritual is ordered to keep.
func heededChronicle(t *testing.T, args ...string) string {
	t.Helper()
	var out, errb bytes.Buffer
	root := NewRootCmd(args, &out, &errb)
	var got string
	root.AddCommand(&cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, _ []string) error {
		got = (*librarium.Settings)(nil).Resolve(runesOf(cmd), os.LookupEnv).Chronicle
		return nil
	}})
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		t.Fatalf("servitor %v: %v\n%s", args, err, errb.String())
	}
	return got
}

func TestChronicleRune_PlacesTheChronicle(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	t.Setenv(librarium.EnvChronicle, "")
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"default in the state home", []string{"probe"}, filepath.Join(state, "servitor", "chronicle.jsonl")},
		{"chronicle before the ritual", []string{"--chronicle", "/a.jsonl", "probe"}, "/a.jsonl"},
		{"chronicle after the ritual", []string{"probe", "--chronicle=/b.jsonl"}, "/b.jsonl"},
		{"hidden alias", []string{"--log", "/c.jsonl", "probe"}, "/c.jsonl"},
		{"the last rune spoken", []string{"--chronicle", "/a.jsonl", "probe", "--log", "/d.jsonl"}, "/d.jsonl"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := heededChronicle(t, tc.args...); got != tc.want {
				t.Fatalf("chronicle %q, want %q", got, tc.want)
			}
		})
	}
	t.Run("rune over environment", func(t *testing.T) {
		t.Setenv(librarium.EnvChronicle, "/env.jsonl")
		if got := heededChronicle(t, "probe"); got != "/env.jsonl" {
			t.Fatalf("without the rune: %q", got)
		}
		if got := heededChronicle(t, "--chronicle", "/rune.jsonl", "probe"); got != "/rune.jsonl" {
			t.Fatalf("with the rune: %q", got)
		}
	})
}

func TestChronicleRune_AliasStaysHidden(t *testing.T) {
	e := newEnv(t)
	for _, args := range [][]string{{"--help"}, {"census", "--help"}} {
		out := e.mustRun(args...)
		if !strings.Contains(out, "--chronicle") || !strings.Contains(out, librarium.EnvChronicle) {
			t.Errorf("%v does not name the chronicle rune:\n%s", args, out)
		}
		if strings.Contains(out, "--log") {
			t.Errorf("%v names the hidden alias --log:\n%s", args, out)
		}
	}
	for _, args := range [][]string{{"--"}, {"census", "--"}, {"invoke", "mouse-autohide-toggle", "on", "--"}} {
		got, _ := e.complete(args...)
		if !slices.Contains(got, "--chronicle") {
			t.Errorf("completion %v lacks --chronicle: %v", args, got)
		}
		if slices.Contains(got, "--log") {
			t.Errorf("completion %v offers hidden alias --log: %v", args, got)
		}
	}
}
