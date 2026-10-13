package rituals

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/dominion"
	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// gated is a rite whose first machine-spirit tells its beginning, and whose
// second waits at $DATA/gate before it tells its end; neither can be read,
// so both are set at every invocation.
const gated = `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
  {"name": "begin", "designation": "machine-spirit",
   "enact": {"*": "echo begin-{{aspect}} >> $DATA/log; touch $DATA/began-{{aspect}}"}},
  {"name": "end", "designation": "machine-spirit",
   "enact": {"*": "while [ ! -e $DATA/gate ]; do sleep 0.01; done; echo end-{{aspect}} >> $DATA/log"}}
]}`

// gatedEdict is gated as an edict; its inscription who stands for the aspect.
const gatedEdict = `{"pattern": "Mark II", "edict": true, "inscriptions": {"who": {"mandatory": true}}, "liturgy": [
  {"designation": "incantation", "words": "echo begin-{{inscription.who}} >> $DATA/log; touch $DATA/began-{{inscription.who}}"},
  {"designation": "incantation",
   "words": "while [ ! -e $DATA/gate ]; do sleep 0.01; done; echo end-{{inscription.who}} >> $DATA/log"}
]}`

// awaitingHerald hears every proclamation and tells each dominion awaited.
type awaitingHerald struct {
	heard   []invocation.Proclamation
	awaited chan struct{}
}

func newAwaitingHerald() *awaitingHerald { return &awaitingHerald{awaited: make(chan struct{}, 4)} }

func (h *awaitingHerald) Proclaim(p invocation.Proclamation) {
	h.heard = append(h.heard, p)
	if p.Tidings == invocation.Dominion {
		h.awaited <- struct{}{}
	}
}

// dominions are the proclamations of dominion heard.
func (h *awaitingHerald) dominions() []invocation.Proclamation {
	var out []invocation.Proclamation
	for _, p := range h.heard {
		if p.Tidings == invocation.Dominion {
			out = append(out, p)
		}
	}
	return out
}

// invoked is an invocation performed in a goroutine of its own.
type invoked struct {
	res  Result
	err  error
	done chan struct{}
}

func (fx *fixture) invoking(ctx context.Context, p Petition) *invoked {
	in := &invoked{done: make(chan struct{})}
	go func() {
		defer close(in.done)
		in.res, in.err = fx.servitor().Invoke(ctx, p)
	}()
	return in
}

// patience is how long a test waits for what must come to pass.
const patience = 10 * time.Second

func (fx *fixture) ended(in *invoked) {
	fx.t.Helper()
	select {
	case <-in.done:
	case <-time.After(patience):
		fx.t.Fatal("the invocation never ended")
	}
}

func (fx *fixture) told(h *awaitingHerald) {
	fx.t.Helper()
	select {
	case <-h.awaited:
	case <-time.After(patience):
		fx.t.Fatal("the awaiting of dominion was never told")
	}
}

// appeared waits until the vessel rel stands.
func (fx *fixture) appeared(rel string) {
	fx.t.Helper()
	deadline := time.Now().Add(patience)
	for !exists(filepath.Join(fx.data, rel)) {
		if time.Now().After(deadline) {
			fx.t.Fatalf("%s never appeared", rel)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (fx *fixture) openGate() { fx.vessel("gate", "") }

// TestInvoke_TwoInvocationsOfOneRiteNeverInterleave: the second invocation
// tells the awaiting of dominion once, waits until the first has its
// verdict, then performs; the first tells nothing of dominion.
func TestInvoke_TwoInvocationsOfOneRiteNeverInterleave(t *testing.T) {
	cases := []struct {
		name, scripture string
		edict           bool
	}{{"rite", gated, false}, {"edict", gatedEdict, true}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fx := newFixture(t)
			fx.rite("gated", c.scripture)
			t.Cleanup(fx.openGate) // no incantation outlives its test
			petition := func(aspect string, h invocation.Herald) Petition {
				if c.edict {
					return Petition{Rite: "gated", Runes: map[string]string{"who": aspect}, Herald: h}
				}
				return Petition{Rite: "gated", Aspect: aspect, Herald: h}
			}
			first, second := newAwaitingHerald(), newAwaitingHerald()

			a := fx.invoking(context.Background(), petition("on", first))
			fx.appeared("began-on")
			b := fx.invoking(context.Background(), petition("off", second))
			fx.told(second)
			if exists(filepath.Join(fx.data, "began-off")) {
				t.Fatal("the second invocation began while the first held dominion")
			}
			fx.openGate()
			fx.ended(a)
			fx.ended(b)

			for _, in := range []*invoked{a, b} {
				if in.err != nil || in.res.Outcome == nil || in.res.Outcome.Verdict != invocation.Triumph {
					t.Fatalf("%v %+v", in.err, in.res)
				}
			}
			if got := fx.read(filepath.Join(fx.data, "log")); got != "begin-on\nend-on\nbegin-off\nend-off\n" {
				t.Fatalf("the liturgies interleaved:\n%s", got)
			}
			if d := first.dominions(); len(d) != 0 {
				t.Fatalf("the first invocation told of dominion: %+v", d)
			}
			d := second.dominions()
			if len(d) != 1 {
				t.Fatalf("the second invocation told of dominion %d times", len(d))
			}
			wantAspect := "off"
			if c.edict {
				wantAspect = ""
			}
			if d[0].Edict != c.edict || d[0].Values.Rite != "gated" || d[0].Values.Aspect != wantAspect {
				t.Fatalf("the awaiting was told as %+v", d[0])
			}
		})
	}
}

// TestInvoke_AnUncontendedInvocationHoldsDominionAndTellsNothingNew: the
// place of dominion over the rite is raised, and the herald hears what it
// heard before dominion was known.
func TestInvoke_AnUncontendedInvocationHoldsDominionAndTellsNothingNew(t *testing.T) {
	fx := newFixture(t)
	fx.rite("mouse", mouse)
	fx.vessel("util.kdl", "")
	h := &herald{}
	res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "mouse", Aspect: "on", Herald: h})
	if err != nil || res.Outcome.Verdict != invocation.Triumph {
		t.Fatalf("%v %+v", err, res)
	}
	if len(h.heard) != 1 || h.heard[0] != invocation.Success {
		t.Fatalf("the herald heard %v", h.heard)
	}
	if !exists(filepath.Join(fx.state, "servitor", "dominions", "rites", "mouse")) {
		t.Fatal("no dominion was held over the rite")
	}
}

// TestInvoke_RitesOfDifferentDominionsOverlap: a rite is performed whole
// while another waits at its gate, and tells nothing of dominion.
func TestInvoke_RitesOfDifferentDominionsOverlap(t *testing.T) {
	fx := newFixture(t)
	fx.rite("gated", gated)
	fx.rite("mouse", mouse)
	fx.vessel("util.kdl", "")
	t.Cleanup(fx.openGate)
	a := fx.invoking(context.Background(), Petition{Rite: "gated", Aspect: "on"})
	fx.appeared("began-on")

	h := newAwaitingHerald()
	b := fx.invoking(context.Background(), Petition{Rite: "mouse", Aspect: "on", Herald: h})
	fx.ended(b)
	if b.err != nil || b.res.Outcome.Verdict != invocation.Triumph || len(h.dominions()) != 0 {
		t.Fatalf("%v %+v %+v", b.err, b.res, h.heard)
	}
	if exists(filepath.Join(fx.data, "log")) && strings.Contains(fx.read(filepath.Join(fx.data, "log")), "end-on") {
		t.Fatal("the gated rite ended before the other was performed")
	}
	fx.openGate()
	fx.ended(a)
}

// TestInvoke_ForeseeingNeverAwaitsDominion: foresight touches nothing, so
// it waits for no dominion.
func TestInvoke_ForeseeingNeverAwaitsDominion(t *testing.T) {
	fx := newFixture(t)
	fx.rite("mouse", mouse)
	fx.vessel("util.kdl", "")
	held := fx.hold("mouse")
	defer held.Relinquish()
	h := newAwaitingHerald()
	in := fx.invoking(context.Background(), Petition{Rite: "mouse", Aspect: "on", Foresee: true, Herald: h})
	fx.ended(in)
	if in.err != nil || len(in.res.Foresight) == 0 || len(h.heard) != 0 {
		t.Fatalf("%v %+v %+v", in.err, in.res, h.heard)
	}
}

// TestInvoke_HaltedWhileAwaitingDominionPerformsNothing: an interrupt
// while dominion is awaited ends the invocation untouched and unchronicled.
func TestInvoke_HaltedWhileAwaitingDominionPerformsNothing(t *testing.T) {
	fx := newFixture(t)
	fx.rite("mouse", mouse)
	fx.vessel("util.kdl", "")
	held := fx.hold("mouse")
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
	grimdark(t, in.err.Error())
	if exists(filepath.Join(fx.data, "said")) || fx.read(filepath.Join(fx.data, "util.kdl")) != "" {
		t.Fatal("a halted invocation touched the machine")
	}
	if _, err := os.Stat(filepath.Join(fx.state, "servitor", "chronicle.jsonl")); err == nil {
		t.Fatal("a halted invocation was chronicled")
	}
}

// hold takes dominion over the rite as another invocation would.
func (fx *fixture) hold(rite string) *dominion.Dominion {
	fx.t.Helper()
	d, err := dominion.Seize(context.Background(), []string{dominion.Rite(os.LookupEnv, rite)}, nil)
	if err != nil {
		fx.t.Fatal(err)
	}
	return d
}

// TestInvoke_ADominionThatCannotBeHeldIsLamented: with no place for
// dominion beneath the state home, the invocation is performed without it,
// as it is without a data-slate.
func TestInvoke_ADominionThatCannotBeHeldIsLamented(t *testing.T) {
	fx := newFixture(t)
	fx.rite("mouse", mouse)
	fx.vessel("util.kdl", "")
	fx.write(fx.state, "servitor/dominions", "a stone where the dominions should lie")
	res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "mouse", Aspect: "on"})
	if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Triumph {
		t.Fatalf("%v %+v", err, res)
	}
	if len(res.Laments) == 0 || !strings.Contains(res.Laments[0].Error(), "no dominion could be held over the rite \"mouse\"") {
		t.Fatalf("laments %v", res.Laments)
	}
	own, _, _ := strings.Cut(res.Laments[0].Error(), ": ") // the machine's own words follow
	grimdark(t, own)
}
