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

// everyStep is a liturgy with a step of every kind.
const everyStep = `
  {"sanctum": "$DATA/niri.kdl", "consecrate": true, "scripture": {"on": "hide", "off": "show"}},
  {"transcription": "$DATA/flag", "scripture": {"on": "raised", "off": null}},
  {"tether": "$DATA/current", "anchor": "$DATA/themes/{{aspect}}"},
  {"incantation": "true"},
  {"litany": "$DATA/scroll.sh"},
  {"vox-cast": "success"}`

func TestAugur_EveryObservableStepGivesAnOmen(t *testing.T) {
	fx := newFixture(t)
	write(t, filepath.Join(fx.data, "scroll.sh"), "true\n")
	r := fx.rite(t, `"on", "off"`, "", everyStep)
	for _, aspect := range []string{"on", "off"} {
		fx.invoke(t, r, aspect, map[string]string{"reason": "gaming remnant"})
		a := fx.augur(t, r, Options{})
		standing(t, a, aspect, Performed)
		want := "1:sanctum=" + aspect + " 2:transcription=" + aspect + " 3:tether=" + aspect
		if got := omens(a); got != want {
			t.Errorf("omens %s, want %s", got, want)
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
	a := fx.augur(t, r, Options{})
	if len(a.Omens) != 3 {
		t.Fatalf("omens %s, want three", omens(a))
	}
	if got, want := a.Omens[0].Target, filepath.Join(fx.data, "niri.kdl"); got != want {
		t.Errorf("the sanctum's omen names %q, want %q", got, want)
	}
	if got, want := a.Omens[2].Target, filepath.Join(fx.data, "current"); got != want {
		t.Errorf("the tether's omen names %q, want %q", got, want)
	}
}

func TestAugur_Standing(t *testing.T) {
	tether := `{"tether": "$DATA/current", "anchor": "$DATA/themes/{{aspect}}"}`
	sanctum := `{"sanctum": "$DATA/niri.kdl", "consecrate": true, "scripture": {"on": "hide", "off": "show"}}`
	for _, tc := range []struct {
		name     string
		liturgy  string
		before   func(t *testing.T, fx fixture, r *librarium.Rite)
		aspect   string
		standing Standing
	}{
		{"dormant: no omen and no data-slate", tether + `, {"incantation": "true"}`,
			func(*testing.T, fixture, *librarium.Rite) {}, "", Dormant},
		{"performed: no omen, but the data-slate's aspect", `{"incantation": "true"}`,
			func(t *testing.T, fx fixture, r *librarium.Rite) { fx.invoke(t, r, "off", nil) }, "off", Performed},
		{"dormant: the first invocation was reverted", `{"incantation": "true"}`,
			func(t *testing.T, fx fixture, r *librarium.Rite) { fx.slate(t, r, "", invocation.Reverted, nil) }, "", Dormant},
		{"corrupted: the omens disagree", sanctum + ", " + tether,
			func(t *testing.T, fx fixture, r *librarium.Rite) {
				fx.invoke(t, r, "on", nil)
				link(t, filepath.Join(fx.data, "themes", "off"), filepath.Join(fx.data, "current"))
			}, "", Corrupted},
		{"corrupted: the last invocation faltered", sanctum,
			func(t *testing.T, fx fixture, r *librarium.Rite) {
				fx.invoke(t, r, "on", nil)
				fx.slate(t, r, "on", invocation.Faltered, nil)
			}, "on", Corrupted},
		{"corrupted: faltered without omen", `{"incantation": "true"}`,
			func(t *testing.T, fx fixture, r *librarium.Rite) { fx.slate(t, r, "off", invocation.Faltered, nil) },
			"off", Corrupted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newFixture(t)
			r := fx.rite(t, `"on", "off"`, "", tc.liturgy)
			tc.before(t, fx, r)
			standing(t, fx.augur(t, r, Options{}), tc.aspect, tc.standing)
		})
	}
}

func TestAugur_DesecratedWhenTheOmensAgreeOnAnotherAspect(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off"`, "", `{"tether": "$DATA/current", "anchor": "$DATA/themes/{{aspect}}"}`)
	fx.invoke(t, r, "on", nil)
	link(t, filepath.Join(fx.data, "themes", "off"), filepath.Join(fx.data, "current"))
	a := fx.augur(t, r, Options{})
	standing(t, a, "off", Desecrated)
	if !a.Desecrated || a.Former != "on" {
		t.Errorf("desecrated %v former %q, want true \"on\"", a.Desecrated, a.Former)
	}
}

func TestAugur_TaintedWhenOtherHandsAlterASanctum(t *testing.T) {
	fx := newFixture(t)
	vessel := filepath.Join(fx.data, "niri.kdl")
	r := fx.rite(t, `"on", "off"`, "", `{"incantation": "true"},
  {"sanctum": "$DATA/niri.kdl", "consecrate": true, "scripture": {"on": ["hide", "{{inscription.reason}}"], "off": ""}}`)
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
	if len(a.Taint) != 1 || a.Taint[0].Number != 2 || a.Taint[0].Kind != librarium.KindSanctum || a.Taint[0].Target != vessel {
		t.Errorf("taint %+v, want verse 2 sanctum %s", a.Taint, vessel)
	}
	fx.invoke(t, r, "off", nil)
	if a := fx.augur(t, r, Options{}); a.Standing != Performed || len(a.Taint) != 0 {
		t.Errorf("an empty sanctum is tainted: %+v", a)
	}
}

func TestAugur_TranscriptionNullIsAbsence(t *testing.T) {
	fx := newFixture(t)
	flag := filepath.Join(fx.data, "flag")
	r := fx.rite(t, `"on", "off"`, "", `{"transcription": "$DATA/flag", "scripture": {"on": ["raised", ""], "off": null}}`)
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T)
		want  string
	}{
		{"absent vessel", func(*testing.T) {}, "1:transcription=off"},
		{"vessel of on", func(t *testing.T) { write(t, flag, "raised\n") }, "1:transcription=on"},
		{"one final newline forgiven", func(t *testing.T) { write(t, flag, "raised") }, "1:transcription=on"},
		{"scripture of no aspect", func(t *testing.T) { write(t, flag, "lowered\n") }, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Remove(flag)
			tc.setup(t)
			if got := omens(fx.augur(t, r, Options{})); got != tc.want {
				t.Errorf("omens %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAugur_TetherComparedWithEveryAnchor(t *testing.T) {
	fx := newFixture(t)
	name := filepath.Join(fx.data, "current")
	r := fx.rite(t, `"on", "off"`, "", `{"tether": "$DATA/current", "anchor": {"on": "$DATA/themes//on/", "off": null}}`)
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T)
		want  string
	}{
		{"unbound name", func(*testing.T) {}, "1:tether=off"},
		{"bound to the anchor", func(t *testing.T) { link(t, filepath.Join(fx.data, "themes", "on"), name) }, "1:tether=on"},
		{"bound relatively", func(t *testing.T) { link(t, "themes/../themes/on", name) }, "1:tether=on"},
		{"bound elsewhere", func(t *testing.T) { link(t, filepath.Join(fx.data, "themes", "porpl"), name) }, ""},
		{"a vessel stands at the name", func(t *testing.T) { write(t, name, "x") }, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Remove(name)
			tc.setup(t)
			if got := omens(fx.augur(t, r, Options{})); got != tc.want {
				t.Errorf("omens %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAugur_UnreadableVesselGivesNoOmen(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("the machine spirit denies nothing to root")
	}
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off"`, "", `
  {"sanctum": "$DATA/niri.kdl", "consecrate": true, "scripture": "x"},
  {"transcription": "$DATA/flag", "scripture": {"on": "raised", "off": null}}`)
	fx.invoke(t, r, "on", nil)
	for _, p := range []string{"niri.kdl", "flag"} {
		if err := os.Chmod(filepath.Join(fx.data, p), 0); err != nil {
			t.Fatal(err)
		}
	}
	a := fx.augur(t, r, Options{})
	if got := omens(a); got != "" {
		t.Errorf("unreadable vessels gave omens %s", got)
	}
	standing(t, a, "on", Performed)
}

func TestAugur_ObservedFieldsRenderWithTheSlatesInscriptions(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off"`, "", `{"tether": "$DATA/current", "anchor": "$DATA/{{inscription.place}}/{{aspect}}"}`)
	if err := os.Mkdir(filepath.Join(fx.data, "far"), 0o755); err != nil {
		t.Fatal(err)
	}
	fx.invoke(t, r, "on", map[string]string{"place": "far"})
	if got := omens(fx.augur(t, r, Options{})); got != "1:tether=on" {
		t.Errorf("omens %q, want the tether's omen of on", got)
	}
	if err := os.Remove(SlatePath(fx.env, r.Name)); err != nil {
		t.Fatal(err)
	}
	if got := omens(fx.augur(t, r, Options{})); got != "" {
		t.Errorf("without the data-slate the inscription is unknown, yet omens %q were read", got)
	}
}

func TestAugur_ObservedFieldsRenderWithTheDecreesOfEveryAspect(t *testing.T) {
	fx := newFixture(t)
	scripture := `{
  "pattern": "Mark I",
  "aspects": ["on", "off"],
  "inscriptions": {"mode": {"decrees": {"on": "auto-hide", "off": "always-visible"}}},
  "liturgy": [{"transcription": "$DATA/mode", "scripture": "mode {{inscription.mode}}"}]
}`
	path := filepath.Join(fx.lib, "rites", "mouse.json")
	write(t, path, strings.ReplaceAll(scripture, "$DATA", fx.data))
	r, findings := librarium.LoadFile(path)
	if findings.Heretical() {
		t.Fatal(findings)
	}
	fx.invoke(t, r, "on", map[string]string{"mode": "auto-hide"})
	write(t, filepath.Join(fx.data, "mode"), "mode always-visible")
	a := fx.augur(t, r, Options{})
	if got := omens(a); got != "1:transcription=off" {
		t.Errorf("omens %q, want the transcription's omen of off", got)
	}
	standing(t, a, "off", Desecrated)
}

func TestAugur_AnOmenOfSeveralAspectsIsNoOmen(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off", "auto"`, "",
		`{"transcription": "$DATA/flag", "scripture": {"on": "raised", "*": "lowered"}}`)
	write(t, filepath.Join(fx.data, "flag"), "lowered")
	if got := omens(fx.augur(t, r, Options{})); got != "" {
		t.Errorf("omens %q, want none", got)
	}
	write(t, filepath.Join(fx.data, "flag"), "raised")
	if got := omens(fx.augur(t, r, Options{})); got != "1:transcription=on" {
		t.Errorf("omens %q, want the transcription's omen of on", got)
	}
}

func TestAugur_ASanctumOfAnUndeclaredAspectGivesNoOmen(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off"`, "", `{"sanctum": "$DATA/niri.kdl", "scripture": "x"}`)
	write(t, filepath.Join(fx.data, "niri.kdl"),
		"// +++ begin of sanctum mouse -- aspect|porpl +++\nx\n// +++ end of sanctum mouse +++\n")
	if got := omens(fx.augur(t, r, Options{})); got != "" {
		t.Errorf("omens %q, want none", got)
	}
}

func TestAugur_AGarbledSlateIsLamentedAndIgnored(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off"`, "", `{"incantation": "true"}`)
	write(t, SlatePath(fx.env, r.Name), "{garbled")
	a, err := Augur(r, Options{Env: fx.env})
	if err == nil {
		t.Error("a garbled data-slate went unlamented")
	}
	standing(t, a, "", Dormant)
}

func TestHeretic_ReadsOnlyTheDataSlate(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off"`, "", `{"incantation": "true"}`)
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
