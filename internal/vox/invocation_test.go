package vox

import (
	"bytes"
	"io"
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
	if w := ControllingTerminal(); w != nil {
		t.Fatalf("a terminal that cannot be opened was found: %v", w)
	}

	ttyPath = filepath.Join(t.TempDir(), "tty")
	if err := os.WriteFile(ttyPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	h := New(Config{Pick: first})
	h.Proclaim(success())
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(ttyPath)
	if err != nil || !strings.Contains(string(b), "theme → porpl") {
		t.Fatalf("the terminal heard %q (%v)", b, err)
	}
}

// rite writes and loads a rite of aspects on and off with liturgy; $DATA
// is a temporary place for vessels.
func rite(t *testing.T, liturgy string) *librarium.Rite {
	t.Helper()
	data := t.TempDir()
	if err := os.WriteFile(filepath.Join(data, "util.kdl"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "rites", "mouse.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	scripture := `{"pattern": "Mark I", "aspects": ["on", "off"], "liturgy": [` +
		strings.ReplaceAll(liturgy, "$DATA", data) + `]}`
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

func TestInvocation_OneNotificationFromProgressToTriumph(t *testing.T) {
	s := newStub(t)
	r := rite(t, `{"sanctum": "$DATA/util.kdl", "scripture": "1"},
	  {"vox-cast": "progress"},
	  {"incantation": "true"},
	  {"incantation": "true"},
	  {"incantation": "true"},
	  {"vox-cast": "progress"},
	  {"vox-cast": "success"}`)

	if out := invoke(t, r, withoutTerminal(Config{})); out.Verdict != invocation.Triumph {
		t.Fatalf("verdict %q", out.Verdict)
	}

	calls := s.calls(t)
	if len(calls) != 3 {
		t.Fatalf("calls %q", calls)
	}
	if _, ok := flag(calls[0], "-r"); ok {
		t.Errorf("the first vox-cast replaces something: %q", calls[0])
	}
	if id, _ := flag(calls[2], "-r"); id != "42" {
		t.Errorf("the triumph replaces not the progress: %q", calls[2])
	}
	if v, _ := flag(calls[0], "-h"); v != "int:value:25" || !strings.HasSuffix(calls[0][len(calls[0])-1], "step 1 / 4") ||
		!strings.Contains(calls[0][len(calls[0])-1], "util.kdl") {
		t.Errorf("first progress %q", calls[0])
	}
	if v, _ := flag(calls[1], "-h"); v != "int:value:100" || !strings.HasSuffix(calls[1][len(calls[1])-1], "step 4 / 4") {
		t.Errorf("second progress %q", calls[1])
	}
}

func TestInvocation_TheFallIsPrintedOnTheTerminal(t *testing.T) {
	s := newStub(t)
	r := rite(t, `{"sanctum": "$DATA/util.kdl", "scripture": "1"},
	  {"vox-cast": "progress"},
	  {"incantation": "exit 3", "reversion": "true"},
	  {"vox-cast": "success"}`)
	var tty bytes.Buffer
	h := New(Config{Pick: first, Terminal: func() io.Writer { return &tty }})

	if out := invoke(t, r, h); out.Verdict != invocation.Reverted {
		t.Fatalf("verdict %q", out.Verdict)
	}

	lines := strings.Split(strings.TrimSuffix(tty.String(), "\n"), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "step 1 / 2") ||
		!strings.Contains(lines[1], "exit 3") || !strings.Contains(lines[1], "Every deed is undone") {
		t.Fatalf("the terminal heard:\n%s", tty.String())
	}
	if calls := s.calls(t); len(calls) != 0 {
		t.Fatalf("the desktop heard %q", calls)
	}
	t.Log("\n" + tty.String())
}
