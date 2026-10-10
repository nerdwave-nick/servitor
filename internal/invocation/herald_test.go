package invocation

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// hearer is a herald that keeps every proclamation.
type hearer struct{ heard []Proclamation }

func (h *hearer) Proclaim(p Proclamation) { h.heard = append(h.heard, p) }

func TestHerald_HearsProgressOfTheRealStepBeforeAndSuccess(t *testing.T) {
	fx := newFixture(t)
	util := write(t, fx.data, "util.kdl", "", 0o644)
	r := fx.rite(t, `{"vox-cast": "progress"},
	  {"sanctum": "$DATA/util.kdl", "scripture": "1"},
	  {"vox-cast": "progress"},
	  {"tether": "$DATA/current-theme", "anchor": "$DATA/themes/{{aspect}}"},
	  {"incantation": "true"},
	  {"vox-cast": "progress"},
	  {"vox-cast": "success"}`)
	h := &hearer{}

	out, _ := perform(t, r, Options{Aspect: "on", Former: "off", Inscriptions: map[string]string{"reason": "gaming"}, Herald: h})

	if out.Verdict != Triumph {
		t.Fatalf("verdict %q", out.Verdict)
	}
	type heard struct {
		tidings      Tidings
		step, steps  int
		kind, target string
		verse        int
	}
	want := []heard{
		{Progress, 0, 3, "", "", 0},
		{Progress, 1, 3, "sanctum", util, 2},
		{Progress, 3, 3, "incantation", "true", 5},
		{Success, 3, 3, "incantation", "true", 5},
	}
	if len(h.heard) != len(want) {
		t.Fatalf("heard %d proclamations, want %d: %+v", len(h.heard), len(want), h.heard)
	}
	for i, w := range want {
		p := h.heard[i]
		got := heard{p.Tidings, p.Values.Tidings.Step, p.Values.Tidings.Steps,
			p.Values.Tidings.StepKind, p.Values.Tidings.StepTarget, p.Verse.Number}
		if got != w {
			t.Errorf("proclamation %d: got %+v, want %+v", i, got, w)
		}
		if p.Values.Aspect != "on" || p.Values.Former != "off" || p.Values.Rite != "mouse" ||
			p.Values.Inscriptions["reason"] != "gaming" || p.Outcome != nil || p.Values.Tidings.Heresy != "" {
			t.Errorf("proclamation %d carries %+v", i, p)
		}
	}
}

func TestHerald_HearsTheFallUnbiddenAfterTheReversions(t *testing.T) {
	fx := newFixture(t)
	write(t, fx.data, "util.kdl", "", 0o644)
	r := fx.rite(t, `{"sanctum": "$DATA/util.kdl", "scripture": "1"},
	  {"vox-cast": "progress"},
	  {"incantation": "echo woe; exit 3", "reversion": "exit 1"},
	  {"vox-cast": "success"}`)
	h := &hearer{}

	out, _ := perform(t, r, Options{Aspect: "on", Herald: h})

	if out.Verdict != Faltered || len(h.heard) != 2 {
		t.Fatalf("verdict %q, heard %+v", out.Verdict, h.heard)
	}
	if p := h.heard[0]; p.Tidings != Progress || p.Values.Tidings.Step != 1 || p.Values.Tidings.Steps != 2 {
		t.Fatalf("first proclamation %+v", p)
	}
	p := h.heard[1]
	tid := p.Values.Tidings
	if p.Tidings != Failure || tid.Step != 2 || tid.Steps != 2 || tid.StepKind != "incantation" ||
		tid.StepTarget != "echo woe; exit 3" || p.Verse.Number != 3 {
		t.Fatalf("failure proclaimed as %+v", p)
	}
	if tid.Heresy == "" || tid.Heresy != out.Fell.Heresy.Error() {
		t.Fatalf("heresy %q, fell %v", tid.Heresy, out.Fell.Heresy)
	}
	if p.Outcome == nil || p.Outcome.Verdict != Faltered || len(p.Outcome.Reversions) != 2 ||
		p.Outcome.Reversions[0].Heresy == nil || !strings.Contains(p.Outcome.Fell.Output, "woe") {
		t.Fatalf("outcome proclaimed as %+v", p.Outcome)
	}
}

func TestHerald_HearsNothingOfATriumphWithoutVoxCasts(t *testing.T) {
	fx := newFixture(t)
	write(t, fx.data, "util.kdl", "", 0o644)
	h := &hearer{}
	if out, _ := perform(t, fx.rite(t, mouseSanctum), Options{Aspect: "on", Herald: h}); out.Verdict != Triumph {
		t.Fatalf("verdict %q", out.Verdict)
	}
	if len(h.heard) != 0 {
		t.Fatalf("heard %+v", h.heard)
	}
}

func TestHerald_NilHearsNothingAndHindersNothing(t *testing.T) {
	fx := newFixture(t)
	write(t, fx.data, "util.kdl", "", 0o644)
	r := fx.rite(t, `{"sanctum": "$DATA/util.kdl", "scripture": "1"},
	  {"vox-cast": "progress"},
	  {"incantation": "exit 1"}`)
	out, _ := perform(t, r, Options{Aspect: "on"})
	if out.Verdict != Reverted || len(out.Deeds) != 3 {
		t.Fatalf("outcome %+v", out)
	}
}

// watcher is a hearer that also watches every real step begin.
type watcher struct {
	hearer
	begun []string
}

func (w *watcher) Begin(v Verse, step, steps int) {
	w.begun = append(w.begun, fmt.Sprintf("%d %s %d/%d", v.Number, v.Kind.Key(), step, steps))
}

func TestWatcher_SeesEveryRealStepBegin(t *testing.T) {
	fx := newFixture(t)
	write(t, fx.data, "util.kdl", "", 0o644)
	r := fx.rite(t, `{"vox-cast": "progress"},
	  {"sanctum": "$DATA/util.kdl", "scripture": "1"},
	  {"vox-cast": "progress"},
	  {"incantation": "true"},
	  {"incantation": "exit 3"},
	  {"incantation": "true"},
	  {"vox-cast": "success"}`)
	w := &watcher{}

	out, _ := perform(t, r, Options{Aspect: "on", Herald: w})

	if out.Verdict != Reverted {
		t.Fatalf("verdict %q", out.Verdict)
	}
	want := []string{"2 sanctum 1/4", "4 incantation 2/4", "5 incantation 3/4"}
	if !slices.Equal(w.begun, want) {
		t.Fatalf("saw begun %q, want %q", w.begun, want)
	}
	if len(w.heard) != 3 || w.heard[1].Tidings != Progress || w.heard[2].Tidings != Failure {
		t.Fatalf("the watcher heard %+v", w.heard)
	}
}

// witness is a hearer that also sees every real step performed, keeping
// what it saw and heard in one order.
type witness struct {
	hearer
	seen []string
}

func (w *witness) Proclaim(p Proclamation) {
	w.hearer.Proclaim(p)
	w.seen = append(w.seen, string(p.Tidings))
}

func (w *witness) Performed(v Verse, step, steps int) {
	w.seen = append(w.seen, fmt.Sprintf("%d %s %d/%d", v.Number, v.Kind.Key(), step, steps))
}

func TestWitness_SeesEveryRealStepPerformedBeforeTheVoxCastsAfterIt(t *testing.T) {
	fx := newFixture(t)
	write(t, fx.data, "util.kdl", "", 0o644)
	r := fx.rite(t, `{"vox-cast": "progress"},
	  {"sanctum": "$DATA/util.kdl", "scripture": "1"},
	  {"vox-cast": "progress"},
	  {"incantation": "true"},
	  {"incantation": "exit 3"},
	  {"incantation": "true"}`)
	w := &witness{}

	out, _ := perform(t, r, Options{Aspect: "on", Herald: w})

	if out.Verdict != Reverted {
		t.Fatalf("verdict %q", out.Verdict)
	}
	want := []string{"progress", "2 sanctum 1/4", "progress", "4 incantation 2/4", "failure"}
	if !slices.Equal(w.seen, want) {
		t.Fatalf("saw %q, want %q", w.seen, want)
	}
}
