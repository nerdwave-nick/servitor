package vox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

func TestControllingTerminal_IsWhatOpens(t *testing.T) {
	orig := ttyPath
	t.Cleanup(func() { ttyPath = orig })

	ttyPath = filepath.Join(t.TempDir(), "absent", "tty")
	if _, ok := ControllingTerminal(); ok {
		t.Fatal("a terminal that cannot be opened was found")
	}

	ttyPath = filepath.Join(t.TempDir(), "tty")
	if err := os.WriteFile(ttyPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if cols, ok := ControllingTerminal(); !ok || cols != 0 {
		t.Fatalf("a terminal of unknown width is found as %d columns (%v)", cols, ok)
	}
}

// rite writes and loads a rite of aspects on and off with the progress and
// success vox-casts and subjects; $DATA is a temporary place for vessels.
func rite(t *testing.T, subjects string) *librarium.Rite {
	t.Helper()
	data := t.TempDir()
	if err := os.WriteFile(filepath.Join(data, "util.kdl"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "rites", "mouse.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	scripture := `{"pattern": "Mark II", "aspects": ["on", "off"], "vox-cast": ["progress", "success"], "subjects": [` +
		strings.ReplaceAll(subjects, "$DATA", data) + `]}`
	if err := os.WriteFile(path, []byte(scripture), 0o644); err != nil {
		t.Fatal(err)
	}
	r, findings := librarium.LoadFile(path)
	if findings.Heretical() {
		t.Fatalf("rite is heretical:\n%v", findings)
	}
	return r
}

func invoke(t *testing.T, r *librarium.Rite, h *Herald) invocation.Outcome {
	t.Helper()
	inv, err := invocation.Prepare(r, invocation.Options{Aspect: "on", Herald: h})
	if err != nil {
		t.Fatal(err)
	}
	out, err := inv.Perform()
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	return out
}

// TestInvocation_OneNotificationFromProgressToTriumph: the progress of
// every subject set and the triumph are one notification upon the desktop,
// each replacing the one before, its value counting the subjects done.
func TestInvocation_OneNotificationFromProgressToTriumph(t *testing.T) {
	s := newStub(t)
	r := rite(t, `{"name": "util", "designation": "sanctum", "vessel": "$DATA/util.kdl", "omens": {"*": "1"}},
	  {"name": "a", "designation": "machine-spirit", "enact": {"*": "true"}},
	  {"name": "b", "designation": "machine-spirit", "enact": {"*": "true"}},
	  {"name": "c", "designation": "machine-spirit", "enact": {"*": "true"}}`)

	if out := invoke(t, r, quiet(Config{})); out.Verdict != invocation.Triumph {
		t.Fatalf("verdict %q", out.Verdict)
	}

	calls := s.calls(t)
	if len(calls) != 5 {
		t.Fatalf("calls %q", calls)
	}
	if _, ok := flag(calls[0], "-r"); ok {
		t.Errorf("the first vox-cast replaces something: %q", calls[0])
	}
	for _, c := range calls[1:] {
		if id, _ := flag(c, "-r"); id != "42" {
			t.Errorf("a later vox-cast replaces not the first: %q", c)
		}
	}
	if v, _ := flag(calls[0], "-h"); v != "int:value:25" || !strings.HasSuffix(calls[0][len(calls[0])-1], "subject 1 / 4") ||
		!strings.Contains(calls[0][len(calls[0])-1], "util") {
		t.Errorf("first progress %q", calls[0])
	}
	if v, _ := flag(calls[3], "-h"); v != "int:value:100" || !strings.HasSuffix(calls[3][len(calls[3])-1], "subject 4 / 4") {
		t.Errorf("last progress %q", calls[3])
	}
}
