package augury

import (
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// stateEnv is an environment whose XDG_STATE_HOME is a temporary directory.
func stateEnv(t *testing.T) (librarium.Environment, string) {
	t.Helper()
	state := t.TempDir()
	vars := map[string]string{"XDG_STATE_HOME": state, "HOME": t.TempDir()}
	return func(k string) (string, bool) { v, ok := vars[k]; return v, ok }, state
}

var noon = time.Date(2026, 10, 10, 13, 30, 0, 0, time.UTC)

func TestSlatePath_LiesUnderTheStateDirectory(t *testing.T) {
	env, state := stateEnv(t)
	if got, want := SlatePath(env, "theme"), filepath.Join(state, "servitor", "data-slates", "theme.json"); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestWriteSlate_WritesTheShapeOfTheCodexAtomically(t *testing.T) {
	env, state := stateEnv(t)
	s := Slate{Aspect: "porpl", Inscriptions: map[string]string{"reason": "gaming remnant"},
		LastRite: &LastRite{Verdict: invocation.Triumph, At: noon}}
	if err := WriteSlate(env, "theme", s); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(state, "servitor", "data-slates")
	data, err := os.ReadFile(filepath.Join(dir, "theme.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"aspect":"porpl","inscriptions":{"reason":"gaming remnant"},` +
		`"last_rite":{"verdict":"triumph","at":"2026-10-10T13:30:00Z"}}` + "\n"
	if string(data) != want {
		t.Errorf("got  %s\nwant %s", data, want)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("the data-slates hall holds more than the slate: %v", entries)
	}
	if info, _ := os.Stat(filepath.Join(dir, "theme.json")); info.Mode().Perm() != 0o644 {
		t.Errorf("seal %v, want 0644", info.Mode().Perm())
	}
}

func TestReadSlate_ReadsWhatWasWritten(t *testing.T) {
	env, _ := stateEnv(t)
	if s, err := ReadSlate(env, "theme"); s != nil || err != nil {
		t.Fatalf("a rite never invoked has a slate: %+v %v", s, err)
	}
	want := Slate{Aspect: "on", Inscriptions: map[string]string{"mode": "auto-hide"},
		LastRite: &LastRite{Verdict: invocation.Reverted, At: noon}}
	if err := WriteSlate(env, "theme", want); err != nil {
		t.Fatal(err)
	}
	if err := WriteSlate(env, "theme", want); err != nil {
		t.Fatalf("a slate cannot be written anew: %v", err)
	}
	got, err := ReadSlate(env, "theme")
	if err != nil || got == nil {
		t.Fatalf("got %+v, %v", got, err)
	}
	if got.Aspect != want.Aspect || !maps.Equal(got.Inscriptions, want.Inscriptions) ||
		got.LastRite == nil || *got.LastRite != *want.LastRite {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestReadSlate_DenouncesAGarbledSlate(t *testing.T) {
	env, state := stateEnv(t)
	p := filepath.Join(state, "servitor", "data-slates", "theme.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("{not binharic"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := ReadSlate(env, "theme")
	if err == nil {
		t.Fatalf("a garbled slate was read: %+v", s)
	}
	if plain := regexp.MustCompile(`(?i)\b(invalid|error|character|looking|unexpected)\b`).FindString(err.Error()); plain != "" {
		t.Errorf("%q speaks the plain word %q", err, plain)
	}
}

func TestSlateOf_RecordsWhereTheInvocationLeftTheRite(t *testing.T) {
	opts := invocation.Options{Aspect: "porpl", Former: "default",
		Inscriptions: map[string]string{"reason": "new"}, Recorded: map[string]string{"reason": "old"}}
	local := time.Date(2026, 10, 10, 15, 30, 0, 987654321, time.FixedZone("CEST", 2*60*60))
	for _, tc := range []struct {
		verdict      invocation.Verdict
		aspect       string
		inscriptions map[string]string
	}{
		{invocation.Triumph, "porpl", map[string]string{"reason": "new"}},
		{invocation.Reverted, "default", map[string]string{"reason": "old"}},
		{invocation.Faltered, "default", map[string]string{"reason": "old"}},
	} {
		t.Run(string(tc.verdict), func(t *testing.T) {
			s := SlateOf(invocation.Outcome{Verdict: tc.verdict}, opts, local)
			if s.Aspect != tc.aspect || !maps.Equal(s.Inscriptions, tc.inscriptions) {
				t.Errorf("got aspect %q inscriptions %v, want %q %v", s.Aspect, s.Inscriptions, tc.aspect, tc.inscriptions)
			}
			if s.LastRite == nil || s.LastRite.Verdict != tc.verdict || !s.LastRite.At.Equal(noon) ||
				s.LastRite.At.Location() != time.UTC {
				t.Errorf("last rite %+v, want %s at %v in UTC", s.LastRite, tc.verdict, noon)
			}
		})
	}
}

func TestRecord_WritesTheSlateOfAnInvocation(t *testing.T) {
	env, _ := stateEnv(t)
	opts := invocation.Options{Aspect: "on"}
	if err := Record(env, "mouse", invocation.Outcome{Verdict: invocation.Triumph}, opts, noon); err != nil {
		t.Fatal(err)
	}
	s, err := ReadSlate(env, "mouse")
	if err != nil || s == nil {
		t.Fatalf("got %+v, %v", s, err)
	}
	if s.Aspect != "on" || s.Inscriptions == nil || len(s.Inscriptions) != 0 || s.LastRite.Verdict != invocation.Triumph {
		t.Errorf("got %+v", s)
	}
	data, _ := os.ReadFile(SlatePath(env, "mouse"))
	if want := `{"aspect":"on","inscriptions":{},"last_rite":{"verdict":"triumph","at":"2026-10-10T13:30:00Z"}}` + "\n"; string(data) != want {
		t.Errorf("got  %s\nwant %s", data, want)
	}
}
