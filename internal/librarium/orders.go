package librarium

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EnvChronicle names the variable of the environment that places the
// chronicle, ranking below the rune --chronicle and above the settings.
const EnvChronicle = "SERVITOR_CHRONICLE"

// Defaults of the orders the settings do not write.
const (
	DefaultTongue   = "bash"
	DefaultVox      = VoxNotifySend
	DefaultPatience = 30 * time.Second
)

// Environment looks up a variable of the environment; os.LookupEnv serves.
type Environment func(key string) (string, bool)

// Runes are the values of the servitor's runes that sway the settings; ""
// is a rune not given.
type Runes struct {
	Chronicle string // --chronicle
}

// Orders are the settings resolved: every order holds the value it obeys.
type Orders struct {
	Tongue    string        // the tongue of incantations and litanies naming none
	Chronicle string        // absolute path of the chronicle
	Vox       string        // VoxNotifySend or VoxOff
	Patience  time.Duration // the patience of incantations and litanies naming none
}

// Resolve resolves each order by precedence rune > environment > settings >
// default. Only the chronicle has a rune and a variable of the environment;
// tongue, vox and patience come from the settings or their defaults. A rune
// or variable given empty counts as not given. Paths expand a leading "~"
// and $VARS through env; a relative path from a rune or the environment
// resolves against the working directory, one from the settings against the
// Librarium. Nil settings resolve as unwritten ones.
func (s *Settings) Resolve(r Runes, env Environment) Orders {
	if s == nil {
		s = &Settings{}
	}
	o := Orders{Tongue: s.Tongue, Vox: s.Vox, Patience: s.Patience}
	if o.Tongue == "" {
		o.Tongue = DefaultTongue
	}
	if o.Vox == "" {
		o.Vox = DefaultVox
	}
	if o.Patience <= 0 {
		o.Patience = DefaultPatience
	}
	switch c, _ := env(EnvChronicle); {
	case r.Chronicle != "":
		o.Chronicle = resolvePath(r.Chronicle, "", env)
	case c != "":
		o.Chronicle = resolvePath(c, "", env)
	case s.Chronicle != "":
		o.Chronicle = resolvePath(s.Chronicle, filepath.Dir(s.Path), env)
	default:
		o.Chronicle = filepath.Join(StateDir(env), "chronicle.jsonl")
	}
	return o
}

// StateDir returns the servitor's state directory: $XDG_STATE_HOME/servitor,
// or ~/.local/state/servitor when XDG_STATE_HOME is unset or relative.
func StateDir(env Environment) string {
	if base, _ := env("XDG_STATE_HOME"); filepath.IsAbs(base) {
		return filepath.Join(base, "servitor")
	}
	home, _ := env("HOME")
	return filepath.Join(home, ".local", "state", "servitor")
}

// resolvePath expands a leading "~" and $VARS in p through env and resolves
// a relative result against base, or the working directory when base is "".
func resolvePath(p, base string, env Environment) string {
	if home, ok := env("HOME"); ok && home != "" && (p == "~" || strings.HasPrefix(p, "~/")) {
		p = home + p[1:]
	}
	p = os.Expand(p, func(k string) string { v, _ := env(k); return v })
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	if base == "" {
		if abs, err := filepath.Abs(p); err == nil {
			return abs
		}
	}
	return filepath.Join(base, p)
}
