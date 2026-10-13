package augury

import (
	"path/filepath"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanctum"
)

// power is a rite of three aspects whose smt shares its omen between two
// of them, and whose profile names its aspect.
func power(t *testing.T, fx fixture) *librarium.Rite {
	t.Helper()
	return fx.stateRite(t, `"performance", "balanced", "powersave"`, "",
		spirit("smt", `{"performance": "off", "*": "on"}`, `{"scan": "cat $DATA/smt"}`)+", "+
			spirit("profile", "", `{"scan": "cat $DATA/profile"}`))
}

// TestAugurMarkII_OmensAgreeByIntersection: the fitting aspects of a state
// rite are the intersection of the omens of every subject that yielded
// one, and every row of the table of agreement holds.
func TestAugurMarkII_OmensAgreeByIntersection(t *testing.T) {
	for _, tc := range []struct {
		name         string
		smt, profile string // "" leaves the state file absent: its subject is mute
		slate        string // the data-slate's aspect; "-" for no data-slate
		aspect       string
		standing     Standing
		fitting      []string
		desecrated   bool
	}{
		{"the hidden wrong subject", "on", "performance", "-", "", Corrupted, nil, false},
		{"the hidden wrong subject beside a data-slate", "on", "performance", "performance", "", Corrupted, nil, false},
		{"one aspect", "off", "performance", "-", "performance", Performed, []string{"performance"}, false},
		{"one aspect the data-slate recorded", "on", "balanced", "balanced", "balanced", Performed, []string{"balanced"}, false},
		{"one aspect the data-slate did not record", "on", "balanced", "powersave", "balanced", Desecrated, []string{"balanced"}, true},
		{"several, the data-slate's among them", "on", "", "powersave", "powersave", Performed, []string{"balanced", "powersave"}, false},
		{"several, the data-slate's not among them", "on", "", "performance", "", Desecrated, []string{"balanced", "powersave"}, true},
		{"several, no data-slate", "on", "", "-", "", Dormant, []string{"balanced", "powersave"}, false},
		{"no omen, the data-slate's aspect", "", "", "balanced", "balanced", Performed, nil, false},
		{"no omen and no data-slate", "", "", "-", "", Dormant, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newFixture(t)
			r := power(t, fx)
			for name, shows := range map[string]string{"smt": tc.smt, "profile": tc.profile} {
				if shows != "" {
					fx.shows(t, name, shows)
				}
			}
			if tc.slate != "-" {
				fx.slate(t, r, tc.slate, invocation.Triumph, nil)
			}
			a := fx.augur(t, r, Options{})
			judged(t, a, tc.aspect, tc.standing, tc.fitting...)
			former := ""
			if tc.desecrated {
				former = tc.slate
			}
			if a.Desecrated != tc.desecrated || a.Former != former {
				t.Errorf("desecrated %v former %q, want %v %q", a.Desecrated, a.Former, tc.desecrated, former)
			}
		})
	}
}

// TestAugurMarkII_TheLastInvocationFalteredCorrupts: whatever the omens
// agree on, a last invocation that faltered leaves the rite corrupted.
func TestAugurMarkII_TheLastInvocationFalteredCorrupts(t *testing.T) {
	fx := newFixture(t)
	r := power(t, fx)
	fx.shows(t, "smt", "off")
	fx.slate(t, r, "performance", invocation.Faltered, nil)
	judged(t, fx.augur(t, r, Options{}), "performance", Corrupted, "performance")
}

// builtins is a state rite of three aspects holding a subject of every
// built-in designation.
func builtins(t *testing.T, fx fixture) *librarium.Rite {
	t.Helper()
	return fx.stateRite(t, `"on", "off", "auto"`, "", `
	  {"name": "niri", "designation": "sanctum", "vessel": "$DATA/niri.kdl", "ward": "gaming", "glyph": "#",
	   "omens": {"on": "hide", "*": "show"}},
	  {"name": "flag", "designation": "transcription", "vessel": "$DATA/flag", "omens": {"on": "raised", "*": null}},
	  {"name": "theme", "designation": "tether", "tether": "$DATA/current",
	   "omens": {"auto": "$DATA/themes/off", "*": "$DATA/themes/{{aspect}}"}}`)
}

// warded writes the sanctum of the rite "gaming" holding content and
// marked with aspect into the vessel niri.kdl.
func (fx fixture) warded(t *testing.T, aspect, content string) {
	t.Helper()
	m := sanctum.Marker{Glyph: "#", Ward: "gaming"}
	write(t, filepath.Join(fx.data, "niri.kdl"), sanctum.Render(m, "", aspect, content))
}

// TestAugurMarkII_BuiltInSubjectsYieldSets: a sanctum's marker names one
// aspect, a transcription and a tether fit every aspect whose omen they
// match; the rite stands in the intersection.
func TestAugurMarkII_BuiltInSubjectsYieldSets(t *testing.T) {
	fx := newFixture(t)
	r := builtins(t, fx)
	link(t, filepath.Join(fx.data, "themes", "off"), filepath.Join(fx.data, "current"))
	a := fx.augur(t, r, Options{})
	if got := readings(a); got != "flag=off|auto theme=off|auto" {
		t.Errorf("readings %q", got)
	}
	judged(t, a, "", Dormant, "off", "auto")

	fx.warded(t, "auto", "show")
	a = fx.augur(t, r, Options{})
	if got := readings(a); got != "niri=auto flag=off|auto theme=off|auto" {
		t.Errorf("readings %q", got)
	}
	judged(t, a, "auto", Performed, "auto")
	if a.Auspex != NoAuspex || len(a.Taint) != 0 {
		t.Errorf("heed %q taint %v", a.Auspex, a.Taint)
	}

	fx.warded(t, "auto", "hide")
	a = fx.augur(t, r, Options{})
	if a.Standing != Tainted || len(a.Taint) != 1 || a.Taint[0].Subject != "niri" {
		t.Errorf("standing %q taint %+v, want the subject niri tainted", a.Standing, a.Taint)
	}
}

// TestAugurMarkII_SingleAspectOmensReadAsMarkI: when every omen names one
// aspect, a state rite reads exactly as the same liturgy of Mark I.
func TestAugurMarkII_SingleAspectOmensReadAsMarkI(t *testing.T) {
	type machine struct{ marked, held, anchored string }
	for _, tc := range []struct {
		name    string
		machine machine
		slate   string
	}{
		{"agreeing", machine{"on", "hide", "on"}, "-"},
		{"agreeing on the data-slate's", machine{"off", "show", "off"}, "off"},
		{"desecrated", machine{"off", "show", "off"}, "on"},
		{"disagreeing", machine{"on", "hide", "off"}, "on"},
		{"tainted", machine{"on", "show", "on"}, "on"},
		{"nothing to read", machine{}, "off"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newFixture(t)
			markII := fx.stateRite(t, `"on", "off"`, "", `
	  {"name": "niri", "designation": "sanctum", "vessel": "$DATA/niri.kdl", "ward": "gaming", "glyph": "#",
	   "omens": {"on": "hide", "off": "show"}},
	  {"name": "theme", "designation": "tether", "tether": "$DATA/current", "omens": {"*": "$DATA/themes/{{aspect}}"}}`)
			markI := fx.load(t, "gaming", `{"pattern": "Mark I", "aspects": ["on", "off"], "liturgy": [
	  {"sanctum": "$DATA/niri.kdl", "ward": "gaming", "glyph": "#", "scripture": {"on": "hide", "off": "show"}},
	  {"tether": "$DATA/current", "anchor": "$DATA/themes/{{aspect}}"}]}`)
			if tc.machine.marked != "" {
				fx.warded(t, tc.machine.marked, tc.machine.held)
				link(t, filepath.Join(fx.data, "themes", tc.machine.anchored), filepath.Join(fx.data, "current"))
			}
			if tc.slate != "-" {
				fx.slate(t, markI, tc.slate, invocation.Triumph, nil)
			}
			want, got := fx.augur(t, markI, Options{}), fx.augur(t, markII, Options{})
			if got.Aspect != want.Aspect || got.Standing != want.Standing || got.Desecrated != want.Desecrated ||
				got.Former != want.Former || len(got.Taint) != len(want.Taint) || len(got.Omens) != len(want.Omens) {
				t.Errorf("Mark II read %+v\nMark I read %+v", got, want)
			}
		})
	}
}

// TestAugurMarkII_AnEdictReadsAsAnEdict: an edict of pattern Mark II is
// judged from its data-slate alone, as any edict.
func TestAugurMarkII_AnEdictReadsAsAnEdict(t *testing.T) {
	fx := newFixture(t)
	r := fx.load(t, "capture", `{"pattern": "Mark II", "edict": true, "liturgy": [
	  {"designation": "incantation", "words": "true"}]}`)
	if a := fx.augur(t, r, Options{}); !a.Edict || a.Standing != Dormant || a.Auspex != NoAuspex {
		t.Errorf("augury %+v, want a dormant edict", a)
	}
	fx.slate(t, r, "", invocation.Reverted, nil)
	if a := fx.augur(t, r, Options{}); a.Standing != Proclaimed {
		t.Errorf("standing %q, want proclaimed", a.Standing)
	}
}
