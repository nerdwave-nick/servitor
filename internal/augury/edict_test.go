package augury

import (
	"maps"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// edict writes the edict "capture" of the given liturgy ($DATA is the
// vessels' place) into the Librarium and loads it.
func (fx fixture) edict(t *testing.T, liturgy string) *librarium.Rite {
	t.Helper()
	path := filepath.Join(fx.lib, "rites", "capture.json")
	write(t, path, strings.ReplaceAll(`{
  "pattern": "Mark I",
  "edict": true,
  "inscriptions": {"region": {"purpose": "the slurp geometry", "decrees": "full"}},
  "liturgy": [`+liturgy+`]
}`, "$DATA", fx.data))
	r, findings := librarium.LoadFile(path)
	if findings.Heretical() || !r.Edict {
		t.Fatalf("the edict is heretical or no edict:\n%v", findings)
	}
	return r
}

// proclaim performs the edict r with inscriptions and records its slate,
// as the invoking servitor does; it must end in triumph.
func (fx fixture) proclaim(t *testing.T, r *librarium.Rite, inscriptions map[string]string) {
	t.Helper()
	opts := invocation.Options{Edict: true, Inscriptions: inscriptions}
	inv, err := invocation.Prepare(r, opts)
	if err != nil {
		t.Fatalf("pre-flight of the edict:\n%v", err)
	}
	out, err := inv.Perform()
	if err != nil || out.Verdict != invocation.Triumph {
		t.Fatalf("proclaiming: err=%v outcome=%+v", err, out)
	}
	if err := Record(fx.env, r.Name, out, opts, noon); err != nil {
		t.Fatal(err)
	}
}

// fall records a proclaiming of the edict r that ended in verdict, as the
// invoking servitor does.
func (fx fixture) fall(t *testing.T, r *librarium.Rite, verdict invocation.Verdict) {
	t.Helper()
	opts := invocation.Options{Edict: true, Inscriptions: map[string]string{"region": "lost"},
		Recorded: map[string]string{"region": "window"}}
	if err := Record(fx.env, r.Name, invocation.Outcome{Verdict: verdict}, opts, noon); err != nil {
		t.Fatal(err)
	}
}

func TestAugur_AnEdictStandsByItsLastVerdict(t *testing.T) {
	triumph := func(t *testing.T, fx fixture, r *librarium.Rite) {
		fx.proclaim(t, r, map[string]string{"region": "window"})
	}
	falls := func(verdicts ...invocation.Verdict) func(*testing.T, fixture, *librarium.Rite) {
		return func(t *testing.T, fx fixture, r *librarium.Rite) {
			for _, v := range verdicts {
				fx.fall(t, r, v)
			}
		}
	}
	for _, tc := range []struct {
		name     string
		before   []func(*testing.T, fixture, *librarium.Rite)
		standing Standing
	}{
		{"dormant: never proclaimed", nil, Dormant},
		{"proclaimed: the last invocation ended in triumph", []func(*testing.T, fixture, *librarium.Rite){triumph}, Proclaimed},
		{"proclaimed: the last invocation was reverted", []func(*testing.T, fixture, *librarium.Rite){falls(invocation.Reverted)}, Proclaimed},
		{"proclaimed: reverted after a triumph", []func(*testing.T, fixture, *librarium.Rite){triumph, falls(invocation.Reverted)}, Proclaimed},
		{"corrupted: the last invocation faltered", []func(*testing.T, fixture, *librarium.Rite){triumph, falls(invocation.Faltered)}, Corrupted},
		{"proclaimed: a reverted invocation after the faltered", []func(*testing.T, fixture, *librarium.Rite){falls(invocation.Faltered, invocation.Reverted)}, Proclaimed},
		{"proclaimed: triumph after the faltered", []func(*testing.T, fixture, *librarium.Rite){falls(invocation.Faltered), triumph}, Proclaimed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newFixture(t)
			write(t, filepath.Join(fx.data, "scroll.sh"), "true\n")
			r := fx.edict(t, `{"incantation": "true"}, {"litany": "$DATA/scroll.sh"}, {"vox-cast": "success"}`)
			for _, b := range tc.before {
				b(t, fx, r)
			}
			a := fx.augur(t, r, Options{})
			if a.Standing != tc.standing || !a.Edict || a.Aspect != "" || a.Desecrated || a.Former != "" {
				t.Errorf("augury %+v, want an edict standing %q, bearing no aspect", a, tc.standing)
			}
			if len(a.Omens) != 0 || len(a.Taint) != 0 {
				t.Errorf("an edict yields omens %s or taint %v", omens(a), a.Taint)
			}
		})
	}
}

func TestAugur_AnEdictKeepsTheInscriptionsAndLastRiteOfItsSlate(t *testing.T) {
	fx := newFixture(t)
	r := fx.edict(t, `{"incantation": "true"}`)
	fx.proclaim(t, r, map[string]string{"region": "window"})
	a := fx.augur(t, r, Options{})
	if !maps.Equal(a.Inscriptions, map[string]string{"region": "window"}) {
		t.Errorf("inscriptions %v are not the data-slate's", a.Inscriptions)
	}
	if a.LastRite == nil || a.LastRite.Verdict != invocation.Triumph || !a.LastRite.At.Equal(noon) {
		t.Errorf("last rite %+v", a.LastRite)
	}
}

func TestAugur_AnEdictReadsNoAspectEvenFromASlateThatRecordsOne(t *testing.T) {
	// The slate of a rite later rewritten by hand as an edict still bears
	// the aspect it last stood in.
	fx := newFixture(t)
	r := fx.edict(t, `{"incantation": "true"}`)
	fx.slate(t, r, "on", invocation.Triumph, nil)
	a := fx.augur(t, r, Options{})
	if a.Aspect != "" || a.Standing != Proclaimed || !a.Edict {
		t.Errorf("augury %+v, want proclaimed without aspect", a)
	}
}

func TestHeretic_AnEdictOfHereticalScriptureStandsHeretical(t *testing.T) {
	fx := newFixture(t)
	r := fx.edict(t, `{"incantation": "true"}`)
	fx.proclaim(t, r, map[string]string{"region": "window"})
	a, err := Heretic(r.Name, Options{Env: fx.env})
	if err != nil || a.Standing != Heretical {
		t.Errorf("augury %+v, %v; want heretical", a, err)
	}
}

func TestAugury_MarshalsAnEdictWithoutAspect(t *testing.T) {
	a := Augury{Rite: "capture", Edict: true, Standing: Proclaimed,
		Inscriptions: map[string]string{"region": "window"},
		LastRite:     &LastRite{Verdict: invocation.Reverted, At: noon}}
	want := `{"rite":"capture","edict":true,"standing":"proclaimed","auspex":"","inscriptions":{"region":"window"},` +
		`"last_rite":{"verdict":"reverted","at":"2026-10-10T13:30:00Z"}}`
	if got := marshal(t, a); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	dormant := `{"rite":"capture","edict":true,"standing":"dormant","auspex":"","inscriptions":{},"last_rite":null}`
	if got := marshal(t, Augury{Rite: "capture", Edict: true, Standing: Dormant}); got != dormant {
		t.Errorf("got  %s\nwant %s", got, dormant)
	}
}

func TestAugury_MarshalsAnEdictReadFromItsSlate(t *testing.T) {
	fx := newFixture(t)
	r := fx.edict(t, `{"incantation": "true"}`)
	fx.proclaim(t, r, map[string]string{"region": "window"})
	fx.fall(t, r, invocation.Faltered)
	want := `{"rite":"capture","edict":true,"standing":"corrupted","auspex":"","inscriptions":{"region":"window"},` +
		`"last_rite":{"verdict":"faltered","at":"2026-10-10T13:30:00Z"}}`
	if got := marshal(t, fx.augur(t, r, Options{})); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
