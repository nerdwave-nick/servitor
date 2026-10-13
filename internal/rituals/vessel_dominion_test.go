package rituals

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/dominion"
	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// keeper is a rite that keeps a sanctum in the vessel %s (consecrating it
// when %t), then, in an observance, tells its beginning and waits at
// $DATA/gate; invoked into "fall" the observance falls there and the rite
// reverts.
const keeper = `{"pattern": "Mark II", "aspects": ["on", "fall"], "subjects": [
  {"name": "kept", "designation": "sanctum", "vessel": "%s", "consecrate": %t, "omens": {"*": "%s {{aspect}}"},
   "recessional": ["wait"]}],
  "observances": [{"name": "wait", "designation": "incantation",
   "words": "touch $DATA/began-%s; while [ ! -e $DATA/gate ]; do sleep 0.01; done; [ {{aspect}} != fall ]"}]}`

// sharer keeps a sanctum in the vessel %s (consecrating it when %t) and is
// done.
const sharer = `{"pattern": "Mark II", "aspects": ["on"], "subjects": [
  {"name": "shared", "designation": "sanctum", "vessel": "%s", "consecrate": %t, "omens": {"*": "%s {{aspect}}"}}]}`

// binder tethers $DATA/current to $DATA/%s, then, in an observance, tells
// its beginning and waits at $DATA/gate.
const binder = `{"pattern": "Mark II", "aspects": ["on"], "subjects": [
  {"name": "current", "designation": "tether", "tether": "$DATA/current", "omens": {"*": "$DATA/%s"},
   "recessional": ["wait"]}],
  "observances": [{"name": "wait", "designation": "incantation",
   "words": "touch $DATA/began-%s; while [ ! -e $DATA/gate ]; do sleep 0.01; done"}]}`

func (fx *fixture) tether(rel, anchor string) {
	fx.t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(fx.data, rel)), 0o755); err != nil {
		fx.t.Fatal(err)
	}
	if err := os.Symlink(anchor, filepath.Join(fx.data, rel)); err != nil {
		fx.t.Fatal(err)
	}
}

func (fx *fixture) triumphed(in *invoked, want invocation.Verdict) {
	fx.t.Helper()
	if in.err != nil || in.res.Outcome == nil || in.res.Outcome.Verdict != want {
		fx.t.Fatalf("not %v: %v %+v", want, in.err, in.res)
	}
}

// holdVessel takes dominion over the vessel at path as another invocation
// would.
func (fx *fixture) holdVessel(path string) *dominion.Dominion {
	fx.t.Helper()
	d, err := dominion.Seize(context.Background(), []string{dominion.Vessel(os.LookupEnv, dominion.Resolve(path))}, nil)
	if err != nil {
		fx.t.Fatal(err)
	}
	return d
}

// TestInvoke_RitesSharingAVesselKeepEachOthersSanctums: while alpha holds
// dominion over the vessel it keeps a sanctum in, beta (another ward in the
// same vessel, however it names it) tells the awaiting once and writes
// nothing; afterwards the vessel holds beta's sanctum, and alpha's unless
// alpha fell and reverted, its reversion sparing beta's sanctum.
func TestInvoke_RitesSharingAVesselKeepEachOthersSanctums(t *testing.T) {
	cases := []struct {
		name, alpha, beta, vessel string
		consecrate                bool
		raise                     func(fx *fixture)
	}{
		{name: "one vessel named alike", alpha: "$DATA/shared.kdl", beta: "$DATA/shared.kdl", vessel: "shared.kdl",
			raise: func(fx *fixture) { fx.vessel("shared.kdl", "") }},
		{name: "a vessel reached through a tether", alpha: "$DATA/link.kdl", beta: "$DATA/shared.kdl",
			vessel: "shared.kdl", raise: func(fx *fixture) { fx.vessel("shared.kdl", ""); fx.tether("link.kdl", "shared.kdl") }},
		{name: "a vessel in a tethered hall", alpha: "$DATA/linked/shared.kdl", beta: "$DATA/hall/shared.kdl",
			vessel: "hall/shared.kdl", raise: func(fx *fixture) { fx.vessel("hall/shared.kdl", ""); fx.tether("linked", "hall") }},
		{name: "a vessel consecrated by the invocation", alpha: "$DATA/linked/new.kdl", beta: "$DATA/hall/new.kdl",
			vessel: "hall/new.kdl", consecrate: true,
			raise: func(fx *fixture) { fx.vessel("hall/.keep", ""); fx.tether("linked", "hall") }},
	}
	for _, c := range cases {
		for _, aspect := range []string{"on", "fall"} {
			t.Run(c.name+"/"+aspect, func(t *testing.T) {
				fx := newFixture(t)
				c.raise(fx)
				fx.rite("alpha", fmt.Sprintf(keeper, c.alpha, c.consecrate, "alpha", "alpha"))
				fx.rite("beta", fmt.Sprintf(sharer, c.beta, c.consecrate, "beta"))
				t.Cleanup(fx.openGate)
				vessel := filepath.Join(fx.data, c.vessel)

				first, second := newAwaitingHerald(), newAwaitingHerald()
				a := fx.invoking(context.Background(), Petition{Rite: "alpha", Aspect: aspect, Herald: first})
				fx.appeared("began-alpha")
				b := fx.invoking(context.Background(), Petition{Rite: "beta", Aspect: "on", Herald: second})
				fx.told(second)
				if strings.Contains(fx.read(vessel), "beta") {
					t.Fatal("beta wrote the vessel while alpha held dominion over it")
				}
				fx.openGate()
				fx.ended(a)
				fx.ended(b)

				want := map[string]invocation.Verdict{"on": invocation.Triumph, "fall": invocation.Reverted}[aspect]
				fx.triumphed(a, want)
				fx.triumphed(b, invocation.Triumph)
				got := fx.read(vessel)
				if !strings.Contains(got, "beta on") || strings.Contains(got, "alpha on") != (aspect == "on") ||
					strings.Contains(got, "alpha fall") {
					t.Fatalf("the vessel holds:\n%s", got)
				}
				if len(first.dominions()) != 0 || len(second.dominions()) != 1 {
					t.Fatalf("the awaiting was told %d and %d times", len(first.dominions()), len(second.dominions()))
				}
			})
		}
	}
}

// TestInvoke_RitesSharingATetheredNameAwaitOneAnother: the tethered name is
// a place the rite writes, so its dominion is awaited too.
func TestInvoke_RitesSharingATetheredNameAwaitOneAnother(t *testing.T) {
	fx := newFixture(t)
	fx.rite("alpha", fmt.Sprintf(binder, "a", "alpha"))
	fx.rite("beta", fmt.Sprintf(binder, "b", "beta"))
	t.Cleanup(fx.openGate)
	a := fx.invoking(context.Background(), Petition{Rite: "alpha", Aspect: "on"})
	fx.appeared("began-alpha")
	h := newAwaitingHerald()
	b := fx.invoking(context.Background(), Petition{Rite: "beta", Aspect: "on", Herald: h})
	fx.told(h)
	if exists(filepath.Join(fx.data, "began-beta")) {
		t.Fatal("beta rebound the tether while alpha held dominion over it")
	}
	fx.openGate()
	fx.ended(a)
	fx.ended(b)
	fx.triumphed(a, invocation.Triumph)
	fx.triumphed(b, invocation.Triumph)
	if anchor, err := os.Readlink(filepath.Join(fx.data, "current")); err != nil || anchor != filepath.Join(fx.data, "b") {
		t.Fatalf("the tether leads to %q (%v)", anchor, err)
	}
}

// TestInvoke_RitesOfDifferentVesselsOverlap: rites that write no vessel or
// tethered name in common are performed side by side.
func TestInvoke_RitesOfDifferentVesselsOverlap(t *testing.T) {
	cases := []struct {
		name, beta string
		raise      func(fx *fixture)
	}{
		{"another vessel", fmt.Sprintf(sharer, "$DATA/other.kdl", false, "beta"),
			func(fx *fixture) { fx.vessel("other.kdl", "") }},
		{"another tethered name", `{"pattern": "Mark II", "aspects": ["on"], "subjects": [
			{"name": "elsewhere", "designation": "tether", "tether": "$DATA/elsewhere", "omens": {"*": "$DATA/shared.kdl"}}]}`,
			func(*fixture) {}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fx := newFixture(t)
			fx.vessel("shared.kdl", "")
			c.raise(fx)
			fx.rite("alpha", fmt.Sprintf(keeper, "$DATA/shared.kdl", false, "alpha", "alpha"))
			fx.rite("beta", c.beta)
			t.Cleanup(fx.openGate)
			a := fx.invoking(context.Background(), Petition{Rite: "alpha", Aspect: "on"})
			fx.appeared("began-alpha")
			h := newAwaitingHerald()
			b := fx.invoking(context.Background(), Petition{Rite: "beta", Aspect: "on", Herald: h})
			fx.ended(b)
			fx.triumphed(b, invocation.Triumph)
			if len(h.dominions()) != 0 {
				t.Fatalf("beta awaited dominion: %+v", h.dominions())
			}
			fx.openGate()
			fx.ended(a)
		})
	}
}

// crossing keeps sanctums in two vessels, in the order written.
const crossing = `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
  {"name": "first", "designation": "sanctum", "vessel": "$DATA/%s.kdl", "omens": {"*": "%s {{aspect}}"}},
  {"name": "second", "designation": "sanctum", "vessel": "$DATA/%s.kdl", "omens": {"*": "%s {{aspect}}"}}
]}`

// TestInvoke_DominionOverVesselsIsSeizedInOneOrder: a rite writing its
// vessels in the order v2, v1 seizes v1 first, so while it awaits v1 it
// holds no dominion over v2, and a rite writing v2 alone is performed
// meanwhile. No two rites can thus hold what the other awaits.
func TestInvoke_DominionOverVesselsIsSeizedInOneOrder(t *testing.T) {
	fx := newFixture(t)
	fx.vessel("v1.kdl", "")
	fx.vessel("v2.kdl", "")
	fx.rite("backwards", fmt.Sprintf(crossing, "v2", "backwards", "v1", "backwards"))
	fx.rite("second", fmt.Sprintf(sharer, "$DATA/v2.kdl", false, "second"))
	held := fx.holdVessel(filepath.Join(fx.data, "v1.kdl"))
	defer held.Relinquish()

	h := newAwaitingHerald()
	b := fx.invoking(context.Background(), Petition{Rite: "backwards", Aspect: "on", Herald: h})
	fx.told(h)
	s := fx.invoking(context.Background(), Petition{Rite: "second", Aspect: "on"})
	fx.ended(s)
	fx.triumphed(s, invocation.Triumph)
	held.Relinquish()
	fx.ended(b)
	fx.triumphed(b, invocation.Triumph)
}

// TestInvoke_RitesWritingVesselsInOpposingOrdersNeverDeadlock: many
// invocations of two rites writing the same vessels in opposing orders, all
// at once, all end and keep both sanctums.
func TestInvoke_RitesWritingVesselsInOpposingOrdersNeverDeadlock(t *testing.T) {
	fx := newFixture(t)
	fx.vessel("v1.kdl", "")
	fx.vessel("v2.kdl", "")
	fx.rite("forwards", fmt.Sprintf(crossing, "v1", "forwards", "v2", "forwards"))
	fx.rite("backwards", fmt.Sprintf(crossing, "v2", "backwards", "v1", "backwards"))
	var all []*invoked
	for i := range 12 {
		aspect := []string{"on", "off"}[i%2]
		for _, rite := range []string{"forwards", "backwards"} {
			all = append(all, fx.invoking(context.Background(), Petition{Rite: rite, Aspect: aspect}))
		}
	}
	for _, in := range all {
		fx.ended(in)
		fx.triumphed(in, invocation.Triumph)
	}
	for _, v := range []string{"v1.kdl", "v2.kdl"} {
		got := fx.read(filepath.Join(fx.data, v))
		if !strings.Contains(got, "forwards") || !strings.Contains(got, "backwards") {
			t.Fatalf("%s holds:\n%s", v, got)
		}
	}
}

// drifting keeps its sanctum in a vessel named by the former aspect, which
// only the augury reveals.
const drifting = `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
  {"name": "drift", "designation": "sanctum", "vessel": "$DATA/from-{{former}}.kdl", "consecrate": true,
   "omens": {"*": "drift {{aspect}}"}}
]}`

// TestInvoke_AVesselRevealedByTheAuguryIsAwaitedToo: a vessel whose path
// the former aspect decides is guarded like any other, and the awaiting is
// told once.
func TestInvoke_AVesselRevealedByTheAuguryIsAwaitedToo(t *testing.T) {
	fx := newFixture(t)
	fx.rite("drift", drifting)
	if _, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "drift", Aspect: "on"}); err != nil {
		t.Fatal(err)
	}
	vessel := filepath.Join(fx.data, "from-on.kdl")
	held := fx.holdVessel(vessel)
	defer held.Relinquish()
	h := newAwaitingHerald()
	in := fx.invoking(context.Background(), Petition{Rite: "drift", Aspect: "off", Herald: h})
	fx.told(h)
	if exists(vessel) {
		t.Fatal("the vessel was written while another held dominion over it")
	}
	held.Relinquish()
	fx.ended(in)
	fx.triumphed(in, invocation.Triumph)
	if !strings.Contains(fx.read(vessel), "drift off") || len(h.dominions()) != 1 {
		t.Fatalf("the vessel holds %q; the awaiting was told %d times", fx.read(vessel), len(h.dominions()))
	}
}

// TestInvoke_AnUncontendedInvocationHoldsDominionOverItsVessel: the place of
// dominion over the vessel, sealed by its resolved path, is raised.
func TestInvoke_AnUncontendedInvocationHoldsDominionOverItsVessel(t *testing.T) {
	fx := newFixture(t)
	fx.rite("mouse", mouse)
	vessel := fx.vessel("util.kdl", "")
	if _, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "mouse", Aspect: "on"}); err != nil {
		t.Fatal(err)
	}
	if !exists(dominion.Vessel(os.LookupEnv, dominion.Resolve(vessel))) {
		t.Fatal("no dominion was held over the vessel")
	}
}

// TestInvoke_HaltedWhileAwaitingAVesselPerformsNothingAndHoldsNothing: an
// interrupt while the dominion over a vessel is awaited ends the invocation
// untouched, and the dominion over its rite is relinquished with it.
func TestInvoke_HaltedWhileAwaitingAVesselPerformsNothingAndHoldsNothing(t *testing.T) {
	fx := newFixture(t)
	fx.rite("mouse", mouse)
	vessel := fx.vessel("util.kdl", "")
	held := fx.holdVessel(vessel)
	defer held.Relinquish()
	ctx, halt := context.WithCancel(context.Background())
	h := newAwaitingHerald()
	in := fx.invoking(ctx, Petition{Rite: "mouse", Aspect: "on", Herald: h})
	fx.told(h)
	halt()
	fx.ended(in)

	var forsaken *Forsaken
	if !errors.As(in.err, &forsaken) || !errors.Is(in.err, context.Canceled) || in.res.Outcome != nil {
		t.Fatalf("%v %+v", in.err, in.res)
	}
	if exists(filepath.Join(fx.data, "said")) || fx.read(vessel) != "" {
		t.Fatal("a halted invocation touched the machine")
	}
	free, err := dominion.Seize(context.Background(), []string{dominion.Rite(os.LookupEnv, "mouse")},
		func() { t.Fatal("the halted invocation still holds dominion over its rite") })
	if err != nil {
		t.Fatal(err)
	}
	free.Relinquish()
}
