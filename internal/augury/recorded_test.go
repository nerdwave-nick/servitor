package augury

import (
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// remembering is a state rite whose smt is read by its scan, whose nft is
// read by an exalted auspex alone, and whose flag is a transcription.
func remembering(t *testing.T, fx fixture) *librarium.Rite {
	t.Helper()
	return fx.stateRite(t, `"on", "off"`, ``, spirit("smt", `{"on": "off", "*": "on"}`, `{"scan": "cat $DATA/smt"}`)+`,
	  {"name": "nft", "designation": "machine-spirit", "exalted": true,
	   "auspex": {"exalted": true, "scan": "touch $DATA/spoken; echo on"}, "enact": {"*": "true"}},
	  {"name": "flag", "designation": "transcription", "vessel": "$DATA/flag", "omens": {"on": "raised", "*": null}}`)
}

// recorded writes the data-slate of r: the aspect and the omens recorded
// in boot.
func (fx fixture) recorded(t *testing.T, r *librarium.Rite, aspect, boot string, omens map[string]string) {
	t.Helper()
	s := Slate{Aspect: aspect, LastRite: &LastRite{Verdict: invocation.Triumph, At: noon}, Boot: boot,
		Omens: map[string]Recorded{}}
	for name, omen := range omens {
		s.Omens[name] = Recorded{Omen: omen, At: noon.Add(-time.Minute)}
	}
	if err := WriteSlate(fx.env, r.Name, s); err != nil {
		t.Fatal(err)
	}
}

// stale renders the stale omens of a compactly, as readings does.
func stale(a Augury) string {
	var only Augury
	for _, o := range a.Omens {
		if o.Stale {
			if !o.ReadAt.Equal(noon.Add(-time.Minute)) {
				return "read at " + o.ReadAt.String()
			}
			only.Omens = append(only.Omens, o)
		}
	}
	return readings(only)
}

// TestAugurMarkII_RecordedOmensServeWhereNoLiveOmenIsHeard: the omen the
// data-slate recorded for a subject serves, marked stale with the moment it
// was read, where its live omen is not heard: an exalted auspex always, any
// auspex forgone. It is never used during an invocation, nor when recorded
// in another boot, nor when it is no longer a declared omen; and the
// exalted auspex is never spoken.
func TestAugurMarkII_RecordedOmensServeWhereNoLiveOmenIsHeard(t *testing.T) {
	fx := newFixture(t)
	r := remembering(t, fx)
	fx.shows(t, "smt", "off\n")
	fx.shows(t, "flag", "raised")
	both := map[string]string{"smt": "off", "nft": "on"}

	for _, tc := range []struct {
		name  string
		boot  string
		omens map[string]string
		opts  Options
		want  string // the stale omens
		heed  Heed
	}{
		{"exalted auspex", Boot(), both, Options{}, "nft=on(on)", Heeded},
		{"forgone", Boot(), both, Options{ForgoAuspex: true}, "smt=on(off) nft=on(on)", Forgone},
		{"invoked", Boot(), both, Options{Target: "on"}, "", Heeded},
		{"another boot", "another-boot", both, Options{ForgoAuspex: true}, "", Forgone},
		{"no boot", "", both, Options{ForgoAuspex: true}, "", Forgone},
		{"no longer declared", Boot(), map[string]string{"smt": "maybe", "nft": "auto"}, Options{ForgoAuspex: true}, "", Forgone},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx.recorded(t, r, "on", tc.boot, tc.omens)

			a := fx.augur(t, r, tc.opts)

			if got := stale(a); got != tc.want {
				t.Errorf("stale omens %q, want %q (omens %s)", got, tc.want, readings(a))
			}
			if a.Stale() != (tc.want != "") || a.Auspex != tc.heed {
				t.Errorf("stale %v auspex %q", a.Stale(), a.Auspex)
			}
			judged(t, a, "on", Performed, "on")
		})
	}
	if exists(fx.data + "/spoken") {
		t.Fatal("an exalted auspex was spoken")
	}
}

// TestAugurMarkII_AStaleOmenAloneNeverDesecrates: stale omens fit aspects
// as live ones do, yet a rite is desecrated only when a live omen bears
// witness against the data-slate.
func TestAugurMarkII_AStaleOmenAloneNeverDesecrates(t *testing.T) {
	fx := newFixture(t)
	r := fx.stateRite(t, `"on", "off"`, ``, spirit("smt", `{"on": "off", "*": "on"}`, `{"scan": "cat $DATA/smt"}`)+`,
	  {"name": "nft", "designation": "machine-spirit", "exalted": true,
	   "auspex": {"exalted": true, "scan": "echo on"}, "enact": {"*": "true"}}`)
	fx.recorded(t, r, "off", Boot(), map[string]string{"nft": "on"})

	a := fx.augur(t, r, Options{ForgoAuspex: true}) // the stale omen of nft alone
	judged(t, a, "off", Performed)
	if a.Desecrated || !a.Stale() {
		t.Errorf("desecrated %v stale %v", a.Desecrated, a.Stale())
	}

	fx.shows(t, "smt", "off\n") // a live omen of on bears witness
	a = fx.augur(t, r, Options{})
	judged(t, a, "on", Desecrated, "on")
	if a.Former != "off" || !a.Stale() {
		t.Errorf("former %q stale %v", a.Former, a.Stale())
	}

	fx.shows(t, "smt", "on\n") // live off against stale on
	if a = fx.augur(t, r, Options{}); a.Standing != Corrupted {
		t.Errorf("smt off, nft on: %q (omens %s)", a.Standing, readings(a))
	}
}
