package tui

import (
	"os"
	"path/filepath"

	"github.com/nerdwave-nick/servitor/internal/rituals"
)

// The trials never read the store of the machine's sanctions: the cogitator
// reads one that is never kept.
func init() { rituals.SanctionRoot = filepath.Join(os.TempDir(), "servitor-trials-keep-no-sanctions") }
