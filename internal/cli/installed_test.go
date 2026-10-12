package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/hallowed"
	"github.com/nerdwave-nick/servitor/internal/intercessor"
	"github.com/nerdwave-nick/servitor/internal/rituals"
)

// enshrinedHall names, to the processes the trials awaken, the hall where
// the trials laid the servitor installed.
const enshrinedHall = "SERVITOR_TEST_ENSHRINED"

// enshrineTrials lets the servitor of every trial, and of every process the
// trials awaken as the servitor, find the servitor installed in a hall of
// the trials' own, never the machine's own /usr/bin; the invoker of the
// trials stands in for the root. It returns what tears that hall down; a
// process awakened by the trials finds the hall they laid, and tears down
// nothing.
func enshrineTrials() func() {
	if dir := os.Getenv(enshrinedHall); dir != "" {
		w := intercessor.Machine()
		w.Ground = trialGround(dir)
		rituals.Intercession = w
		return func() {}
	}
	dir, err := os.MkdirTemp("", "servitor-trials-enshrined-")
	if err == nil {
		rituals.Intercession, err = enshrine(dir)
	}
	if err == nil {
		err = os.Setenv(enshrinedHall, dir)
	}
	if err != nil {
		panic(err)
	}
	return func() { _ = os.RemoveAll(dir) }
}

// trialGround is the ground in the hall root, where the invoker of the
// trials stands in for the root.
func trialGround(root string) hallowed.Ground {
	me := uint32(os.Getuid())
	return hallowed.Ground{Root: root, Sovereign: func(uid uint32) bool { return uid == me }}
}

// enshrine lays a servitor installed as the root's in the hall root, and
// returns the world that finds it there.
func enshrine(root string) (intercessor.World, error) {
	w := intercessor.Machine()
	w.Ground = trialGround(root)
	if err := os.Chmod(root, 0o755); err != nil {
		return w, err
	}
	if err := os.MkdirAll(filepath.Join(root, "usr", "bin"), 0o755); err != nil {
		return w, err
	}
	return w, os.WriteFile(filepath.Join(root, w.Servitor), []byte("\x7fELF"), 0o755)
}

// TestInquisition_NotesAServitorInstalledAmiss: while a rite holds exalted
// verses, the Inquisition notes the servitor installed at /usr/bin/servitor
// that is missing, plainly and in binharic, as an impurity of the machine
// that exits 0.
func TestInquisition_NotesAServitorInstalledAmiss(t *testing.T) {
	e, _ := sanctuary(t)
	root := t.TempDir()
	w, err := enshrine(root)
	if err != nil {
		t.Fatal(err)
	}
	former := rituals.Intercession
	t.Cleanup(func() { rituals.Intercession = former })
	rituals.Intercession = w

	finding := "/usr/bin/servitor:1:1: impurity: the rites "
	out, _, code := e.run("inquisition")
	if code != 0 || strings.Contains(out, finding) {
		t.Fatalf("an enshrined servitor: exit %d\n%s", code, out)
	}
	if err := os.Remove(filepath.Join(root, "usr/bin/servitor")); err != nil {
		t.Fatal(err)
	}
	out, _, code = e.run("inquisition")
	if code != 0 || !strings.Contains(out, finding) || !strings.Contains(out, "install the servitor anew") {
		t.Fatalf("a missing servitor: exit %d\n%s", code, out)
	}
	out, _, _ = e.run("inquisition", "--binharic")
	if !strings.Contains(out, `"scripture": "/usr/bin/servitor"`) || !strings.Contains(out, `"judgement": "impurity"`) {
		t.Errorf("binharic:\n%s", out)
	}
	if out, _, _ = e.run("inquisition", "--spare-vessels"); strings.Contains(out, finding) {
		t.Errorf("--spare-vessels:\n%s", out)
	}
}
