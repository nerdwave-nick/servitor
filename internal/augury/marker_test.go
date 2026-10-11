package augury

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAugur_TheMarkerSaysWhatTheAuspexGave: the augury's auspex marker tells
// heeded from mute and from a rite that keeps no auspex, while the aspect
// and standing are still judged by the agreement of whatever omens exist.
func TestAugur_TheMarkerSaysWhatTheAuspexGave(t *testing.T) {
	tether := `{"tether": "$DATA/current", "anchor": "$DATA/themes/{{aspect}}"}`
	for _, tc := range []struct {
		name     string
		keys     string
		want     Heed
		aspect   string
		standing Standing
	}{
		{"an auspex naming an aspect", `
  "auspex": "echo on",`, Heeded, "on", Performed},
		{"an auspex that falls", `
  "auspex": "echo off; exit 1",`, Mute, "on", Performed},
		{"an auspex naming no declared aspect", `
  "auspex": "echo porpl",`, Mute, "on", Performed},
		{"an impatient auspex", `
  "auspex": {"rite": "sleep 3; echo off", "patience": "100ms"},`, Mute, "on", Performed},
		{"an auspex in a tongue spoken nowhere", `
  "auspex": "echo off",
  "tongue": "no-such-tongue-of-the-machine",`, Mute, "on", Performed},
		{"no auspex", ``, NoAuspex, "on", Performed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newFixture(t)
			r := fx.rite(t, `"on", "off"`, tc.keys, tether)
			link(t, filepath.Join(fx.data, "themes", "on"), filepath.Join(fx.data, "current"))
			a := fx.augur(t, r, Options{})
			if a.Auspex != tc.want {
				t.Errorf("auspex %q, want %q", a.Auspex, tc.want)
			}
			standing(t, a, tc.aspect, tc.standing)
		})
	}
}

// TestAugur_TheMarkerOfAForgoneAuspex: a forgone auspex is never awakened
// and marked forgone; a rite without one is still marked "".
func TestAugur_TheMarkerOfAForgoneAuspex(t *testing.T) {
	fx := newFixture(t)
	mark := filepath.Join(fx.data, "awakened")
	r := fx.rite(t, `"on", "off"`, `
  "auspex": "touch `+mark+`; sleep 3; exit 1",`, `{"vox-cast": "success"}`)
	a := fx.augur(t, r, Options{ForgoAuspex: true})
	if _, err := os.Stat(mark); err == nil {
		t.Error("a forgone auspex was awakened")
	}
	if a.Auspex != Forgone {
		t.Errorf("auspex %q, want forgone", a.Auspex)
	}
	plain := fx.rite(t, `"on", "off"`, ``, `{"vox-cast": "success"}`)
	if a := fx.augur(t, plain, Options{ForgoAuspex: true}); a.Auspex != NoAuspex {
		t.Errorf("a rite without auspex, forgone: auspex %q, want \"\"", a.Auspex)
	}
}

// TestAugur_EdictsAndHereticsBearNoMarker: an edict keeps no auspex, and a
// heretical rite's cannot be read; both are marked "".
func TestAugur_EdictsAndHereticsBearNoMarker(t *testing.T) {
	fx := newFixture(t)
	r := fx.edict(t, `{"incantation": "true"}`)
	for name, opts := range map[string]Options{"heeding": {}, "forgoing": {ForgoAuspex: true}} {
		if a := fx.augur(t, r, opts); a.Auspex != NoAuspex {
			t.Errorf("an edict, %s: auspex %q, want \"\"", name, a.Auspex)
		}
		opts.Env = fx.env
		if a, _ := Heretic("mouse", opts); a.Auspex != NoAuspex {
			t.Errorf("a heretic, %s: auspex %q, want \"\"", name, a.Auspex)
		}
	}
}
