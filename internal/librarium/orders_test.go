package librarium

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// environ is an Environment holding exactly vars.
func environ(vars map[string]string) Environment {
	return func(key string) (string, bool) {
		v, ok := vars[key]
		return v, ok
	}
}

func TestResolve_ChronicleRuneOverEnvironmentOverSettingsOverDefault(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	const home = "/home/adept"
	cases := []struct {
		name     string
		rune     string
		env      map[string]string
		settings string
		want     string
	}{
		{"rune over all", "/rune.jsonl", map[string]string{EnvChronicle: "/env.jsonl"}, "/settings.jsonl", "/rune.jsonl"},
		{"environment over settings", "", map[string]string{EnvChronicle: "/env.jsonl"}, "/settings.jsonl", "/env.jsonl"},
		{"settings over default", "", nil, "/settings.jsonl", "/settings.jsonl"},
		{"default in the state home", "", map[string]string{"XDG_STATE_HOME": "/state"}, "", "/state/servitor/chronicle.jsonl"},
		{"default without state home", "", nil, "", home + "/.local/state/servitor/chronicle.jsonl"},
		{"default with relative state home", "", map[string]string{"XDG_STATE_HOME": "state"}, "", home + "/.local/state/servitor/chronicle.jsonl"},
		{"default with empty state home", "", map[string]string{"XDG_STATE_HOME": ""}, "", home + "/.local/state/servitor/chronicle.jsonl"},
		{"empty rune and environment are not given", "", map[string]string{EnvChronicle: ""}, "/settings.jsonl", "/settings.jsonl"},
		{"rune relative to the working directory", "c.jsonl", nil, "", filepath.Join(cwd, "c.jsonl")},
		{"rune from home", "~/r.jsonl", nil, "", home + "/r.jsonl"},
		{"environment relative to the working directory", "", map[string]string{EnvChronicle: "e.jsonl"}, "", filepath.Join(cwd, "e.jsonl")},
		{"environment from home", "", map[string]string{EnvChronicle: "~/e.jsonl"}, "", home + "/e.jsonl"},
		{"settings relative to the Librarium", "", nil, "chronicles/c.jsonl", "/lib/chronicles/c.jsonl"},
		{"settings from home", "", nil, "~/s.jsonl", home + "/s.jsonl"},
		{"settings with variables of the environment", "", map[string]string{"XDG_STATE_HOME": "/state"}, "$XDG_STATE_HOME/s.jsonl", "/state/s.jsonl"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := map[string]string{"HOME": home}
			for k, v := range c.env {
				env[k] = v
			}
			s := &Settings{Path: "/lib/servitor.json", Chronicle: c.settings}
			if got := s.Resolve(Runes{Chronicle: c.rune}, environ(env)).Chronicle; got != c.want {
				t.Fatalf("chronicle %q, want %q", got, c.want)
			}
		})
	}
}

func TestResolve_SettingsOnlyOrdersOverDefault(t *testing.T) {
	// tongue, vox and patience have neither rune nor variable of the
	// environment; no such variable may sway them.
	env := environ(map[string]string{"HOME": "/home/adept", "SERVITOR_TONGUE": "sh", "SERVITOR_VOX": "off", "SERVITOR_PATIENCE": "1s"})
	written := &Settings{Path: "/lib/servitor.json", Tongue: "zsh", Vox: VoxOff, Patience: 5 * time.Second}
	o := written.Resolve(Runes{}, env)
	if o.Tongue != "zsh" || o.Vox != VoxOff || o.Patience != 5*time.Second {
		t.Fatalf("orders %+v must follow the settings", o)
	}
	o = (&Settings{Path: "/lib/servitor.json"}).Resolve(Runes{}, env)
	if o.Tongue != DefaultTongue || o.Vox != VoxNotifySend || o.Patience != DefaultPatience {
		t.Fatalf("orders %+v must fall back to the defaults", o)
	}
	if DefaultTongue != "bash" || DefaultPatience != 30*time.Second {
		t.Fatalf("defaults: tongue %q, patience %v", DefaultTongue, DefaultPatience)
	}
}

func TestResolve_WithoutSettingsYieldsDefaults(t *testing.T) {
	var s *Settings
	o := s.Resolve(Runes{}, environ(map[string]string{"XDG_STATE_HOME": "/state"}))
	want := Orders{Tongue: "bash", Chronicle: "/state/servitor/chronicle.jsonl", Vox: VoxNotifySend, Patience: 30 * time.Second}
	if o != want {
		t.Fatalf("orders %+v, want %+v", o, want)
	}
}

func TestStateDir_FollowsXDG(t *testing.T) {
	cases := map[string]struct {
		env  map[string]string
		want string
	}{
		"absolute state home": {map[string]string{"XDG_STATE_HOME": "/state", "HOME": "/h"}, "/state/servitor"},
		"relative state home": {map[string]string{"XDG_STATE_HOME": "rel", "HOME": "/h"}, "/h/.local/state/servitor"},
		"no state home":       {map[string]string{"HOME": "/h"}, "/h/.local/state/servitor"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := StateDir(environ(c.env)); got != c.want {
				t.Fatalf("StateDir = %q, want %q", got, c.want)
			}
		})
	}
}
