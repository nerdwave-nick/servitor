package augury

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// Subjects of every built-in designation, and a machine-spirit that cannot
// be read.
const (
	niri    = `{"name": "niri", "designation": "sanctum", "vessel": "$DATA/niri.kdl", "consecrate": true, "omens": {"on": "hide", "off": "show"}}`
	flag    = `{"name": "flag", "designation": "transcription", "vessel": "$DATA/flag", "omens": {"on": "raised", "off": null}}`
	current = `{"name": "theme", "designation": "tether", "tether": "$DATA/current", "omens": {"*": "$DATA/themes/{{aspect}}"}}`
	chant   = `{"name": "chant", "designation": "machine-spirit", "enact": {"*": "true"}}`
)

// TestAugur_EveryBuiltInSubjectGivesAnOmen: a sanctum, a transcription
// and a tether each yield the aspect the servitor brought them into; a
// machine-spirit without an auspex yields none. The augury keeps the
// data-slate's inscriptions and last rite.
func TestAugur_EveryBuiltInSubjectGivesAnOmen(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off"`, "", strings.Join([]string{niri, flag, current, chant}, ", "))
	for _, aspect := range []string{"on", "off"} {
		fx.invoke(t, r, aspect, map[string]string{"reason": "gaming remnant"})
		a := fx.augur(t, r, Options{})
		standing(t, a, aspect, Performed)
		if got, want := readings(a), "niri="+aspect+" flag="+aspect+" theme="+aspect; got != want {
			t.Errorf("readings %s, want %s", got, want)
		}
		if a.Desecrated || a.Former != "" || len(a.Taint) != 0 {
			t.Errorf("a rite performed by the servitor is desecrated or tainted: %+v", a)
		}
		if !maps.Equal(a.Inscriptions, map[string]string{"reason": "gaming remnant"}) {
			t.Errorf("inscriptions %v are not the data-slate's", a.Inscriptions)
		}
		if a.LastRite == nil || a.LastRite.Verdict != invocation.Triumph || !a.LastRite.At.Equal(noon) {
			t.Errorf("last rite %+v", a.LastRite)
		}
	}
}

// TestAugur_Standing: the standing of a rite of aspects by its omens, its
// data-slate and its last verdict.
func TestAugur_Standing(t *testing.T) {
	for _, tc := range []struct {
		name     string
		subjects string
		before   func(t *testing.T, fx fixture, r *librarium.Rite)
		aspect   string
		standing Standing
	}{
		{"dormant: no omen and no data-slate", current + ", " + chant,
			func(*testing.T, fixture, *librarium.Rite) {}, "", Dormant},
		{"performed: no omen, but the data-slate's aspect", chant,
			func(t *testing.T, fx fixture, r *librarium.Rite) { fx.invoke(t, r, "off", nil) }, "off", Performed},
		{"dormant: the first invocation was reverted", chant,
			func(t *testing.T, fx fixture, r *librarium.Rite) { fx.slate(t, r, "", invocation.Reverted, nil) }, "", Dormant},
		{"corrupted: the omens fit no aspect together", niri + ", " + current,
			func(t *testing.T, fx fixture, r *librarium.Rite) {
				fx.invoke(t, r, "on", nil)
				link(t, filepath.Join(fx.data, "themes", "off"), filepath.Join(fx.data, "current"))
			}, "", Corrupted},
		{"corrupted: the last invocation faltered", niri,
			func(t *testing.T, fx fixture, r *librarium.Rite) {
				fx.invoke(t, r, "on", nil)
				fx.slate(t, r, "on", invocation.Faltered, nil)
			}, "on", Corrupted},
		{"corrupted: faltered without omen", chant,
			func(t *testing.T, fx fixture, r *librarium.Rite) { fx.slate(t, r, "off", invocation.Faltered, nil) },
			"off", Corrupted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newFixture(t)
			r := fx.rite(t, `"on", "off"`, "", tc.subjects)
			tc.before(t, fx, r)
			standing(t, fx.augur(t, r, Options{}), tc.aspect, tc.standing)
		})
	}
}

// TestAugur_DesecratedWhenTheOmensAgreeOnAnotherAspect: omens agreeing on
// an aspect the data-slate did not record desecrate the rite, naming the
// former aspect.
func TestAugur_DesecratedWhenTheOmensAgreeOnAnotherAspect(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off"`, "", current)
	fx.invoke(t, r, "on", nil)
	link(t, filepath.Join(fx.data, "themes", "off"), filepath.Join(fx.data, "current"))
	a := fx.augur(t, r, Options{})
	standing(t, a, "off", Desecrated)
	if !a.Desecrated || a.Former != "on" {
		t.Errorf("desecrated %v former %q, want true \"on\"", a.Desecrated, a.Former)
	}
}

// TestAugur_TaintedWhenOtherHandsAlterASanctum: a sanctum whose scripture
// other hands altered taints the rite, naming the subject and its vessel;
// an empty sanctum is never tainted.
func TestAugur_TaintedWhenOtherHandsAlterASanctum(t *testing.T) {
	fx := newFixture(t)
	vessel := filepath.Join(fx.data, "niri.kdl")
	r := fx.rite(t, `"on", "off"`, "", chant+`,
  {"name": "niri", "designation": "sanctum", "vessel": "$DATA/niri.kdl", "consecrate": true,
   "omens": {"on": ["hide", "{{inscription.reason}}"], "off": ""}}`)
	fx.invoke(t, r, "on", map[string]string{"reason": "gaming remnant"})
	if a := fx.augur(t, r, Options{}); a.Standing != Performed || len(a.Taint) != 0 {
		t.Fatalf("an untouched sanctum is tainted: %+v", a)
	}
	data, err := os.ReadFile(vessel)
	if err != nil {
		t.Fatal(err)
	}
	write(t, vessel, strings.Replace(string(data), "hide", "reveal", 1))
	a := fx.augur(t, r, Options{})
	standing(t, a, "on", Tainted)
	if len(a.Taint) != 1 || a.Taint[0].Subject != "niri" || a.Taint[0].Vessel != vessel {
		t.Errorf("taint %+v, want the subject niri in %s", a.Taint, vessel)
	}
	fx.invoke(t, r, "off", nil)
	if a := fx.augur(t, r, Options{}); a.Standing != Performed || len(a.Taint) != 0 {
		t.Errorf("an empty sanctum is tainted: %+v", a)
	}
}

// TestAugur_TranscriptionNullIsAbsence: a transcription fits the aspect
// whose scripture its vessel holds, one final newline forgiven, and the
// aspect of null when its vessel is absent.
func TestAugur_TranscriptionNullIsAbsence(t *testing.T) {
	fx := newFixture(t)
	path := filepath.Join(fx.data, "flag")
	r := fx.rite(t, `"on", "off"`, "", `{"name": "flag", "designation": "transcription", "vessel": "$DATA/flag",
  "omens": {"on": ["raised", ""], "off": null}}`)
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T)
		want  string
	}{
		{"absent vessel", func(*testing.T) {}, "flag=off"},
		{"vessel of on", func(t *testing.T) { write(t, path, "raised\n") }, "flag=on"},
		{"one final newline forgiven", func(t *testing.T) { write(t, path, "raised") }, "flag=on"},
		{"scripture of no aspect", func(t *testing.T) { write(t, path, "lowered\n") }, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Remove(path)
			tc.setup(t)
			if got := readings(fx.augur(t, r, Options{})); got != tc.want {
				t.Errorf("readings %q, want %q", got, tc.want)
			}
		})
	}
}

// TestAugur_TetherComparedWithEveryAnchor: a tether fits the aspect whose
// anchor its bound name leads to, resolved and cleaned, and the aspect of
// null when nothing stands at its name.
func TestAugur_TetherComparedWithEveryAnchor(t *testing.T) {
	fx := newFixture(t)
	name := filepath.Join(fx.data, "current")
	r := fx.rite(t, `"on", "off"`, "", `{"name": "theme", "designation": "tether", "tether": "$DATA/current",
  "omens": {"on": "$DATA/themes//on/", "off": null}}`)
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T)
		want  string
	}{
		{"unbound name", func(*testing.T) {}, "theme=off"},
		{"bound to the anchor", func(t *testing.T) { link(t, filepath.Join(fx.data, "themes", "on"), name) }, "theme=on"},
		{"bound relatively", func(t *testing.T) { link(t, "themes/../themes/on", name) }, "theme=on"},
		{"bound elsewhere", func(t *testing.T) { link(t, filepath.Join(fx.data, "themes", "porpl"), name) }, ""},
		{"a vessel stands at the name", func(t *testing.T) { write(t, name, "x") }, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Remove(name)
			tc.setup(t)
			if got := readings(fx.augur(t, r, Options{})); got != tc.want {
				t.Errorf("readings %q, want %q", got, tc.want)
			}
		})
	}
}

// TestAugur_UnreadableVesselGivesNoOmen: vessels the machine spirit will
// not let the servitor read yield no omen; the data-slate names the aspect.
func TestAugur_UnreadableVesselGivesNoOmen(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("the machine spirit denies nothing to root")
	}
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off"`, "", `
  {"name": "niri", "designation": "sanctum", "vessel": "$DATA/niri.kdl", "consecrate": true, "omens": {"*": "x"}},
  `+flag)
	fx.invoke(t, r, "on", nil)
	for _, p := range []string{"niri.kdl", "flag"} {
		if err := os.Chmod(filepath.Join(fx.data, p), 0); err != nil {
			t.Fatal(err)
		}
	}
	a := fx.augur(t, r, Options{})
	if got := readings(a); got != "" {
		t.Errorf("unreadable vessels gave readings %s", got)
	}
	standing(t, a, "on", Performed)
}

// TestAugur_ObservedFieldsRenderWithTheSlatesInscriptions: omens are
// rendered with the inscriptions the data-slate recorded; without it the
// inscription is unknown and no omen fits.
func TestAugur_ObservedFieldsRenderWithTheSlatesInscriptions(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off"`, "", `{"name": "theme", "designation": "tether", "tether": "$DATA/current",
  "omens": {"*": "$DATA/{{inscription.place}}/{{aspect}}"}}`)
	if err := os.Mkdir(filepath.Join(fx.data, "far"), 0o755); err != nil {
		t.Fatal(err)
	}
	fx.invoke(t, r, "on", map[string]string{"place": "far"})
	if got := readings(fx.augur(t, r, Options{})); got != "theme=on" {
		t.Errorf("readings %q, want the tether's omen of on", got)
	}
	if err := os.Remove(SlatePath(fx.env, r.Name)); err != nil {
		t.Fatal(err)
	}
	if got := readings(fx.augur(t, r, Options{})); got != "" {
		t.Errorf("without the data-slate the inscription is unknown, yet readings %q were read", got)
	}
}

// TestAugur_ObservedFieldsRenderWithTheDecreesOfEveryAspect: an aspect
// the data-slate did not record renders its omen with its decrees.
func TestAugur_ObservedFieldsRenderWithTheDecreesOfEveryAspect(t *testing.T) {
	fx := newFixture(t)
	r := fx.load(t, "mouse", `{
  "pattern": "Mark II",
  "aspects": ["on", "off"],
  "inscriptions": {"mode": {"decrees": {"on": "auto-hide", "off": "always-visible"}}},
  "subjects": [{"name": "mode", "designation": "transcription", "vessel": "$DATA/mode",
                "omens": {"*": "mode {{inscription.mode}}"}}]
}`)
	fx.invoke(t, r, "on", map[string]string{"mode": "auto-hide"})
	write(t, filepath.Join(fx.data, "mode"), "mode always-visible")
	a := fx.augur(t, r, Options{})
	if got := readings(a); got != "mode=off" {
		t.Errorf("readings %q, want the transcription's omen of off", got)
	}
	standing(t, a, "off", Desecrated)
}

// TestAugur_ASanctumOfAnUndeclaredAspectGivesNoOmen: a marker naming an
// aspect the rite does not declare yields no omen.
func TestAugur_ASanctumOfAnUndeclaredAspectGivesNoOmen(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off"`, "", `{"name": "niri", "designation": "sanctum", "vessel": "$DATA/niri.kdl", "omens": {"*": "x"}}`)
	write(t, filepath.Join(fx.data, "niri.kdl"),
		"// +++ begin of sanctum mouse -- aspect|porpl +++\nx\n// +++ end of sanctum mouse +++\n")
	if got := readings(fx.augur(t, r, Options{})); got != "" {
		t.Errorf("readings %q, want none", got)
	}
}

// TestAugur_AGarbledSlateIsLamentedAndIgnored: a data-slate that cannot be
// read is lamented, and the rite read as if it had none.
func TestAugur_AGarbledSlateIsLamentedAndIgnored(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off"`, "", chant)
	write(t, SlatePath(fx.env, r.Name), "{garbled")
	a, err := Augur(r, Options{Env: fx.env})
	if err == nil {
		t.Error("a garbled data-slate went unlamented")
	}
	standing(t, a, "", Dormant)
}

// TestHeretic_ReadsOnlyTheDataSlate: a heretical rite is read from its
// data-slate alone.
func TestHeretic_ReadsOnlyTheDataSlate(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off"`, "", chant)
	fx.invoke(t, r, "on", map[string]string{"reason": "x"})
	a, err := Heretic("mouse", Options{Env: fx.env})
	if err != nil {
		t.Fatal(err)
	}
	standing(t, a, "", Heretical)
	if a.Rite != "mouse" || a.Inscriptions["reason"] != "x" || a.LastRite == nil || len(a.Omens) != 0 {
		t.Errorf("got %+v", a)
	}
}
