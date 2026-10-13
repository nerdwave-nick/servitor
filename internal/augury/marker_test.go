package augury

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAugur_TheMarkerSaysWhatTheAuspexGave: the augury's auspex marker tells
// heeded from mute and from a rite whose subjects keep no auspex, while the
// aspect and standing are still judged by the intersection of whatever
// omens exist.
func TestAugur_TheMarkerSaysWhatTheAuspexGave(t *testing.T) {
	for _, tc := range []struct {
		name     string
		keys     string
		auspex   string
		want     Heed
		aspect   string
		standing Standing
	}{
		{"an auspex naming an aspect", ``, `{"scan": "echo on"}`, Heeded, "on", Performed},
		{"an auspex that falls", ``, `{"scan": "echo off; exit 1"}`, Mute, "on", Performed},
		{"an auspex naming no declared omen", ``, `{"scan": "echo porpl"}`, Mute, "on", Performed},
		{"an impatient auspex", ``, `{"scan": "sleep 3; echo off", "patience": "100ms"}`, Mute, "on", Performed},
		{"an auspex in a tongue spoken nowhere", `
  "tongue": "no-such-tongue-of-the-machine",`, `{"scan": "echo off"}`, Mute, "on", Performed},
		{"no auspex", ``, ``, NoAuspex, "on", Performed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newFixture(t)
			subjects := current + ", " + chant
			if tc.auspex != "" {
				subjects = current + ", " + spirit("chant", "", tc.auspex)
			}
			r := fx.rite(t, `"on", "off"`, tc.keys, subjects)
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
// and marked forgone; a rite whose subjects keep none is still marked "".
func TestAugur_TheMarkerOfAForgoneAuspex(t *testing.T) {
	fx := newFixture(t)
	mark := filepath.Join(fx.data, "awakened")
	r := fx.rite(t, `"on", "off"`, "", spirit("chant", "", `{"scan": "touch `+mark+`; sleep 3; exit 1"}`))
	a := fx.augur(t, r, Options{ForgoAuspex: true})
	if _, err := os.Stat(mark); err == nil {
		t.Error("a forgone auspex was awakened")
	}
	if a.Auspex != Forgone {
		t.Errorf("auspex %q, want forgone", a.Auspex)
	}
	plain := fx.rite(t, `"on", "off"`, ``, chant)
	if a := fx.augur(t, plain, Options{ForgoAuspex: true}); a.Auspex != NoAuspex {
		t.Errorf("a rite without auspex, forgone: auspex %q, want \"\"", a.Auspex)
	}
}

// TestAugur_EdictsAndHereticsBearNoMarker: an edict keeps no auspex, and a
// heretical rite's cannot be read; both are marked "".
func TestAugur_EdictsAndHereticsBearNoMarker(t *testing.T) {
	fx := newFixture(t)
	r := fx.edict(t, `{"designation": "incantation", "words": "true"}`)
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
