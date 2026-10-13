package rituals

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/invocation"
)

func TestInvoke_PerformsKeepsTheDataSlateAndChronicles(t *testing.T) {
	fx := newFixture(t)
	util := fx.vessel("util.kdl", "input {}\n")
	fx.rite("mouse", mouse)
	h := &herald{}

	res, err := fx.servitor().Invoke(context.Background(),
		Petition{Rite: "mouse", Aspect: "on", Runes: map[string]string{"reason": "gaming remnant"}, Herald: h})

	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome == nil || res.Outcome.Verdict != invocation.Triumph || len(res.Laments) != 0 {
		t.Fatalf("result %+v", res)
	}
	if !strings.Contains(fx.read(util), "hide-after-inactive-ms 400") {
		t.Fatalf("vessel:\n%s", fx.read(util))
	}
	if got := fx.read(filepath.Join(fx.data, "said")); got != "on-gaming remnant\n" {
		t.Fatalf("the incantation said %q", got)
	}
	if !slices.Equal(h.heard, []invocation.Tidings{invocation.Success}) {
		t.Fatalf("the herald heard %v", h.heard)
	}
	slate := fx.slate("mouse")
	if slate == nil || slate.Aspect != "on" || slate.Inscriptions["reason"] != "gaming remnant" ||
		slate.LastRite.Verdict != invocation.Triumph {
		t.Fatalf("data-slate %+v", slate)
	}

	res, err = fx.servitor().Invoke(context.Background(), Petition{Rite: "mouse", Aspect: "off"})
	if err != nil || res.Outcome.Verdict != invocation.Triumph {
		t.Fatalf("second invocation: %v %+v", err, res)
	}
	if res.Options.Former != "on" || res.Options.Inscriptions["reason"] != "restored" ||
		res.Options.Recorded["reason"] != "gaming remnant" {
		t.Fatalf("options %+v", res.Options)
	}
	entries := fx.chronicled()
	if len(entries) != 2 {
		t.Fatalf("chronicled %d entries", len(entries))
	}
	newest, oldest := entries[0], entries[1]
	if oldest.Aspect != "on" || oldest.Former != "" || oldest.Inscriptions["reason"] != "gaming remnant" ||
		newest.Aspect != "off" || newest.Former != "on" || newest.Verdict != invocation.Triumph ||
		!newest.At.Equal(time.Date(2026, 10, 10, 13, 30, 0, 0, time.UTC)) {
		t.Fatalf("chronicle %+v", entries)
	}
}

func TestInvoke_TheChronicleRuneAndEnvironmentPlaceTheChronicle(t *testing.T) {
	fx := newFixture(t)
	fx.vessel("util.kdl", "")
	fx.rite("mouse", mouse)
	s := fx.servitor()
	s.Runes.Chronicle = filepath.Join(fx.data, "by-rune.jsonl")
	if _, err := s.Invoke(context.Background(), Petition{Rite: "mouse", Aspect: "on"}); err != nil {
		t.Fatal(err)
	}
	if !exists(s.Runes.Chronicle) || exists(filepath.Join(fx.state, "servitor", "chronicle.jsonl")) {
		t.Fatal("the rune did not place the chronicle")
	}
}

// TestInvoke_ForeseeingPerformsNothing: foresight tells the subject that
// would change and the observance that would be spoken, and touches,
// keeps and chronicles nothing.
func TestInvoke_ForeseeingPerformsNothing(t *testing.T) {
	fx := newFixture(t)
	util := fx.vessel("util.kdl", "input {}\n")
	fx.rite("mouse", mouse)

	res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "mouse", Aspect: "on", Foresee: true})

	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != nil || len(res.Foresight) != 2 || res.Foresight[0].Vessel == nil ||
		!strings.Contains(res.Foresight[0].Vessel.After, "hide-after-inactive-ms 400") ||
		res.Foresight[1].Speech == nil || !strings.Contains(res.Foresight[1].Speech.Words, "echo on-") {
		t.Fatalf("result %+v", res)
	}
	if fx.read(util) != "input {}\n" || exists(filepath.Join(fx.data, "said")) {
		t.Fatal("foreseeing touched the machine")
	}
	if fx.slate("mouse") != nil || len(fx.chronicled()) != 0 {
		t.Fatal("a foreseen invocation was recorded")
	}
}

// TestInvoke_TheAuspexSeesTheTargetDuringAnInvocation: the auspex of a
// machine-spirit hears the aspect invoked as SERVITOR_ASPECT, and the omen
// it reads names the former aspect.
func TestInvoke_TheAuspexSeesTheTargetDuringAnInvocation(t *testing.T) {
	fx := newFixture(t)
	fx.vessel("cursor", "off\n")
	fx.rite("mouse", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "cursor", "designation": "machine-spirit",
	   "auspex": {"scan": "echo $SERVITOR_ASPECT > $DATA/auspex; cat $DATA/cursor"},
	   "enact": {"*": "echo {{omen}} > $DATA/cursor"}}]}`)

	res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "mouse", Aspect: "on"})

	if err != nil {
		t.Fatal(err)
	}
	if got := fx.read(filepath.Join(fx.data, "auspex")); got != "on\n" {
		t.Fatalf("the auspex saw %q", got)
	}
	if res.Options.Former != "off" {
		t.Fatalf("former %q, though the auspex, the only omen, read off", res.Options.Former)
	}
}

// TestInvoke_AFallIsRevertedKeptAndChronicled: an observance that falls
// restores the sanctum already set, is proclaimed a failure, and the
// reversion is kept on the data-slate and chronicled with the observance
// that fell.
func TestInvoke_AFallIsRevertedKeptAndChronicled(t *testing.T) {
	fx := newFixture(t)
	util := fx.vessel("util.kdl", "input {}\n")
	fx.rite("mouse", falling)
	h := &herald{}

	res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "mouse", Aspect: "on", Herald: h})

	if err != nil {
		t.Fatal(err)
	}
	out := res.Outcome
	if out == nil || out.Verdict != invocation.Reverted || out.Fell == nil || out.Fell.Observance != "fall" ||
		out.Fell.Output != "dying\n" {
		t.Fatalf("outcome %+v", out)
	}
	if fx.read(util) != "input {}\n" {
		t.Fatal("the sanctum was not reverted")
	}
	if !slices.Equal(h.heard, []invocation.Tidings{invocation.Failure}) {
		t.Fatalf("the herald heard %v", h.heard)
	}
	if s := fx.slate("mouse"); s == nil || s.Aspect != "" || s.LastRite.Verdict != invocation.Reverted {
		t.Fatalf("data-slate %+v", s)
	}
	entries := fx.chronicled()
	if len(entries) != 1 || entries[0].Verdict != invocation.Reverted || entries[0].FellAt == nil ||
		entries[0].FellAt.Observance != "fall" || entries[0].Heresy == "" {
		t.Fatalf("chronicle %+v", entries)
	}
}

// TestInvoke_AHaltRevertsAndIsChronicled: an invocation halted before its
// first subject is set touches nothing, and is chronicled reverted.
func TestInvoke_AHaltRevertsAndIsChronicled(t *testing.T) {
	fx := newFixture(t)
	util := fx.vessel("util.kdl", "input {}\n")
	fx.rite("mouse", mouse)
	ctx, halt := context.WithCancel(context.Background())
	halt()

	res, err := fx.servitor().Invoke(ctx, Petition{Rite: "mouse", Aspect: "on"})

	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome == nil || res.Outcome.Verdict != invocation.Reverted || res.Outcome.Fell.Subject != "util" {
		t.Fatalf("outcome %+v", res.Outcome)
	}
	if fx.read(util) != "input {}\n" || exists(filepath.Join(fx.data, "said")) {
		t.Fatal("a halted invocation touched the machine")
	}
	if entries := fx.chronicled(); len(entries) != 1 || entries[0].Verdict != invocation.Reverted {
		t.Fatalf("chronicle %+v", entries)
	}
}

func TestInvoke_ARefusalChangesAndKeepsNothing(t *testing.T) {
	fx := newFixture(t)
	util := fx.vessel("util.kdl", "input {}\n")
	fx.rite("mouse", strings.Replace(mouse, `"purpose": "why the rite was invoked"`,
		`"purpose": "why the rite was invoked", "mandatory": true`, 1))
	fx.rite("lost", strings.Replace(mouse, "$DATA/util.kdl", "$DATA/no-hall/util.kdl", 1))
	s := fx.servitor()

	cases := []struct {
		name string
		p    Petition
	}{
		{"mandatory inscription without word", Petition{Rite: "mouse", Aspect: "on"}},
		{"rune of no inscription", Petition{Rite: "mouse", Aspect: "on", Runes: map[string]string{"reason": "x", "mood": "y"}}},
		{"unknown aspect", Petition{Rite: "mouse", Aspect: "maybe"}},
		{"the pre-flight", Petition{Rite: "lost", Aspect: "on"}},
		{"the pre-flight, foreseen", Petition{Rite: "lost", Aspect: "on", Foresee: true}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := s.Invoke(context.Background(), c.p)
			var hs invocation.Heresies
			if !errors.As(err, &hs) || len(hs) == 0 || res.Outcome != nil {
				t.Fatalf("err %v, result %+v", err, res)
			}
			grimdark(t, err.Error())
		})
	}
	if fx.read(util) != "input {}\n" || fx.slate("mouse") != nil || fx.slate("lost") != nil || len(fx.chronicled()) != 0 {
		t.Fatal("a refused invocation touched the machine or was recorded")
	}
}

func TestInvoke_OnlyRitesFitToBeInvoked(t *testing.T) {
	fx := newFixture(t)
	fx.rite("broken", broken)
	s := fx.servitor()

	_, err := s.Invoke(context.Background(), Petition{Rite: "nope", Aspect: "on"})
	var unrecorded *Unrecorded
	if !errors.As(err, &unrecorded) || unrecorded.Name != "nope" || !strings.Contains(err.Error(), `no rite named "nope"`) {
		t.Fatalf("unknown rite: %v", err)
	}
	_, err = s.Invoke(context.Background(), Petition{Rite: "broken", Aspect: "on"})
	var heretical *Heretical
	if !errors.As(err, &heretical) || len(heretical.Findings) == 0 ||
		!strings.Contains(err.Error(), `the rite "broken" is tainted by heresy`) || !strings.Contains(err.Error(), "broken.json:1:") {
		t.Fatalf("heretical rite: %v", err)
	}
	grimdark(t, err.Error())
}

func TestInvoke_CommenceIsToldOnceTheLiturgyIsAboutToBePerformed(t *testing.T) {
	fx := newFixture(t)
	fx.vessel("util.kdl", "input {}\n")
	fx.rite("mouse", mouse)
	fx.rite("lost", strings.Replace(mouse, "$DATA/util.kdl", "$DATA/no-hall/util.kdl", 1))
	s := fx.servitor()
	h := &herald{}
	var told []Result
	commence := func(res Result) {
		if len(h.heard) != 0 {
			t.Errorf("commence was told after the herald heard %v", h.heard)
		}
		told = append(told, res)
	}

	for _, p := range []Petition{
		{Rite: "lost", Aspect: "on"},
		{Rite: "mouse", Aspect: "on", Foresee: true},
		{Rite: "mouse", Aspect: "maybe"},
	} {
		p.Commence = commence
		if _, err := s.Invoke(context.Background(), p); err != nil && p.Foresee {
			t.Fatal(err)
		}
	}
	if len(told) != 0 {
		t.Fatalf("commence was told of an invocation never performed: %+v", told)
	}

	res, err := s.Invoke(context.Background(), Petition{Rite: "mouse", Aspect: "on", Herald: h, Commence: commence})
	if err != nil || res.Outcome == nil {
		t.Fatalf("%v %+v", err, res)
	}
	if len(told) != 1 || told[0].Rite.Name != "mouse" || told[0].Options.Aspect != "on" ||
		told[0].Options.Former != "" || told[0].Former != res.Former || told[0].Outcome != nil {
		t.Fatalf("commence was told %+v", told)
	}
}
