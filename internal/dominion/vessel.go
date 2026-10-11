package dominion

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// Vessel is the place of the dominion over the vessel or tethered name at
// path, which Resolve has made whole:
// $XDG_STATE_HOME/servitor/dominions/vessels/<sha256 hex of path>.
func Vessel(env librarium.Environment, path string) string {
	sum := sha256.Sum256([]byte(path))
	return filepath.Join(librarium.StateDir(env), Dir, "vessels", hex.EncodeToString(sum[:]))
}

// Resolve makes path, a vessel or tethered name a liturgy writes, whole:
// absolute, with every hall above it followed through its tethers, so that
// a vessel reached through a tethered hall and the same vessel named
// directly are one. The last name itself is not followed: a sanctum or
// transcription has already followed its vessel's tethers to the place it
// rewrites, and a tether rebinds the name itself. A hall that does not
// stand yet is reckoned through the hall above it.
func Resolve(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = filepath.Clean(path)
	}
	return filepath.Join(hall(filepath.Dir(abs)), filepath.Base(abs))
}

// hall follows every tether of the hall dir; when it does not stand, the
// hall above it is followed and its name kept.
func hall(dir string) string {
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		return real
	}
	above := filepath.Dir(dir)
	if above == dir {
		return dir
	}
	return filepath.Join(hall(above), filepath.Base(dir))
}
