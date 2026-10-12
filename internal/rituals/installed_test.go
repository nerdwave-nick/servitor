package rituals

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/hallowed"
	"github.com/nerdwave-nick/servitor/internal/intercessor"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// The trials never judge the machine's own /usr/bin/servitor: a servitor
// given no world of intercession finds the servitor installed in a hall of
// the trials' own, the invoker of the trials standing in for the root.
func TestMain(m *testing.M) {
	Summon = func(cmd *exec.Cmd, _ *os.File) (func() error, error) {
		return nil, fmt.Errorf("the trials never call upon %s", cmd.Path)
	}
	dir, err := os.MkdirTemp("", "servitor-trials-enshrined-")
	if err == nil {
		Intercession, err = enshrine(dir)
	}
	if err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// enshrine lays a servitor installed as the root's in the hall root, and
// returns the world that finds it there.
func enshrine(root string) (intercessor.World, error) {
	w := intercessor.Machine()
	me := uint32(os.Getuid())
	w.Ground = hallowed.Ground{Root: root, Sovereign: func(uid uint32) bool { return uid == me }}
	// the trials never call upon sudo, run0 or pkexec of the machine
	w.LookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	if err := os.Chmod(root, 0o755); err != nil {
		return w, err
	}
	if err := os.MkdirAll(filepath.Join(root, "usr", "bin"), 0o755); err != nil {
		return w, err
	}
	return w, os.WriteFile(filepath.Join(root, w.Servitor), []byte("\x7fELF"), 0o755)
}

// intercede lets the servitor find the servitor installed in a hall of the
// trial's own, marred by mar, for as long as the trial lasts.
func intercede(t *testing.T, mar func(root string)) {
	t.Helper()
	root := t.TempDir()
	w, err := enshrine(root)
	if err != nil {
		t.Fatal(err)
	}
	mar(root)
	former := Intercession
	t.Cleanup(func() { Intercession = former })
	Intercession = w
}

// TestInquire_TheInstalledServitorOfExaltedRites: while a rite examined
// holds exalted verses, the servitor installed at /usr/bin/servitor that is
// missing, or does not stand where the root alone may write, is an impurity
// of the machine; --spare-vessels spares it, and a Librarium without
// exalted verses never needs it.
func TestInquire_TheInstalledServitorOfExaltedRites(t *testing.T) {
	words := []string{"/usr/bin/servitor", "install the servitor anew", "exalted verses"}
	for what, c := range map[string]struct {
		mar   func(root string)
		names []string
		spare bool
		noted bool
	}{
		"missing": {mar: func(root string) { _ = os.Remove(filepath.Join(root, "usr/bin/servitor")) },
			noted: true},
		"beneath a hall others may write": {mar: func(root string) { _ = os.Chmod(filepath.Join(root, "usr"), 0o777) },
			noted: true},
		"others may write it": {mar: func(root string) {
			_ = os.Chmod(filepath.Join(root, "usr/bin/servitor"), 0o777)
		}, noted: true},
		"missing, for an exalted rite named": {mar: func(root string) {
			_ = os.Remove(filepath.Join(root, "usr/bin/servitor"))
		}, names: []string{"plain", "fresh"}, noted: true},
		"missing, for rites without exalted verses": {mar: func(root string) {
			_ = os.Remove(filepath.Join(root, "usr/bin/servitor"))
		}, names: []string{"plain"}},
		"missing, sparing the vessels": {mar: func(root string) {
			_ = os.Remove(filepath.Join(root, "usr/bin/servitor"))
		}, spare: true},
		"enshrined": {mar: func(string) {}},
	} {
		t.Run(what, func(t *testing.T) {
			sc := newSanctuary(t)
			intercede(t, c.mar)
			found := inquire(t, sc.s, c.spare, c.names...)
			got := denounced(found, "", librarium.Impurity, words...)
			switch {
			case c.noted && len(got) != 1:
				t.Fatalf("noted %d times in\n%v", len(got), found)
			case !c.noted && len(got) != 0:
				t.Fatalf("noted %v", got)
			case c.noted && (got[0].Scripture != "/usr/bin/servitor" || got[0].Line != 1 || got[0].Column != 1):
				t.Errorf("noted at %s:%d:%d", got[0].Scripture, got[0].Line, got[0].Column)
			}
		})
	}
	fx := newFixture(t)
	fx.vessel("util.kdl", "input {}\n")
	fx.rite("mouse", mouse)
	intercede(t, func(root string) { _ = os.Remove(filepath.Join(root, "usr/bin/servitor")) })
	if fs := inquire(t, fx.servitor(), false); len(fs) != 0 {
		t.Fatalf("a Librarium without exalted verses: %v", fs)
	}
}
