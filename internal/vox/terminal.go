package vox

import (
	"io"
	"os"
)

// ttyPath is the controlling terminal of the process.
var ttyPath = "/dev/tty"

// ControllingTerminal opens the controlling terminal for writing; it
// returns nil when there is none (/dev/tty cannot be opened), as when the
// servitor is invoked from a hotkey or a launcher.
func ControllingTerminal() io.Writer {
	f, err := os.OpenFile(ttyPath, os.O_WRONLY, 0)
	if err != nil {
		return nil
	}
	return f
}
