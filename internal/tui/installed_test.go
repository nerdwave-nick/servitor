package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/hallowed"
	"github.com/nerdwave-nick/servitor/internal/rituals"
)

// The trials never judge the machine's own /usr/bin/servitor: the servitor
// finds the servitor installed in a hall of the trials' own, the invoker of
// the trials standing in for the root.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "servitor-trials-enshrined-")
	if err == nil {
		err = os.MkdirAll(filepath.Join(dir, "usr", "bin"), 0o755)
	}
	if err == nil {
		err = os.Chmod(dir, 0o755)
	}
	if err == nil {
		err = os.WriteFile(filepath.Join(dir, rituals.Intercession.Servitor), []byte("\x7fELF"), 0o755)
	}
	if err != nil {
		panic(err)
	}
	me := uint32(os.Getuid())
	rituals.Intercession.Ground = hallowed.Ground{Root: dir, Sovereign: func(uid uint32) bool { return uid == me }}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
