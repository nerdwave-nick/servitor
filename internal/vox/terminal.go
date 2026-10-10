package vox

import (
	"os"

	"github.com/charmbracelet/x/term"
)

// ttyPath is the controlling terminal of the process.
var ttyPath = "/dev/tty"

// ControllingTerminal tells whether the servitor has a controlling terminal
// (/dev/tty can be opened) and how many columns it holds, 0 when it does
// not say. There is none when the servitor is invoked from a hotkey or a
// launcher.
func ControllingTerminal() (cols int, ok bool) {
	f, err := os.OpenFile(ttyPath, os.O_WRONLY, 0)
	if err != nil {
		return 0, false
	}
	if w, _, err := term.GetSize(f.Fd()); err == nil && w > 0 {
		cols = w
	}
	_ = f.Close()
	return cols, true
}
