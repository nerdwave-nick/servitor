package cli

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// geteuid tells the effective uid the servitor bears; replaced in tests.
var geteuid = os.Geteuid

// lookupHome finds the home of the faithful named name; replaced in tests.
var lookupHome = func(name string) (string, error) {
	u, err := user.Lookup(name)
	if err != nil {
		return "", err
	}
	return u.HomeDir, nil
}

// bearsMachinesAuthority tells whether the servitor was roused as root.
func bearsMachinesAuthority() bool { return geteuid() == 0 }

// unexalted refuses deed, with the death-mark 2, when the servitor bears the
// machine's own authority: as root the servitor only reads. Every ritual
// that performs calls it before it touches anything — invoke unless
// foreseen, the cogitator, sanction and abjure — naming its deed, e.g.
// `the sanctioning of the rite "x"`.
func unexalted(deed string) error {
	if !bearsMachinesAuthority() {
		return nil
	}
	return refused(fmt.Errorf("%s is refused, and nothing was touched: the servitor bears the machine's own "+
		"authority, and with it the servitor performs nothing. Rouse the servitor as yourself, never through sudo, "+
		"run0 or pkexec; as root it only reads — foresight, augury, census, inquisition and expound", deed))
}

// inheritInvokersWorld makes a servitor roused as root by sudo (SUDO_USER)
// read the world of the one who roused it rather than root's: HOME becomes
// the invoker's home — SUDO_HOME when it is absolute, else the home of
// SUDO_USER — and every XDG_* variable, root's own, is cast away, so that
// the Librarium, the settings, the vessels written from "~", the data-slates
// and the chronicle are the invoker's. Without SUDO_USER, or not as root,
// the servitor's own world is read. It fails when the invoker's home cannot
// be found, for root's world must not be read in its place.
//
// It changes the servitor's own environment, once, before the Librarium is
// read, so that every reading of the world (and every auspex) sees the same.
func inheritInvokersWorld() error {
	invoker := os.Getenv("SUDO_USER")
	if !bearsMachinesAuthority() || invoker == "" {
		return nil
	}
	home := os.Getenv("SUDO_HOME")
	if !filepath.IsAbs(home) {
		found, err := lookupHome(invoker)
		if err != nil || !filepath.IsAbs(found) {
			return fmt.Errorf("the servitor was roused as root by %q, yet the home of %q cannot be found, and "+
				"root's own Librarium is not read in its place; nothing was read", invoker, invoker)
		}
		home = found
	}
	for _, kv := range os.Environ() {
		if k, _, _ := strings.Cut(kv, "="); strings.HasPrefix(k, "XDG_") {
			if err := os.Unsetenv(k); err != nil {
				return err
			}
		}
	}
	return os.Setenv("HOME", filepath.Clean(home))
}
