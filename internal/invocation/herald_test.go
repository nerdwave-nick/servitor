package invocation

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// hearer is a herald that keeps every proclamation.
type hearer struct{ heard []Proclamation }

func (h *hearer) Proclaim(p Proclamation) { h.heard = append(h.heard, p) }

// TestHerald_HearsProgressOfEverySubjectAndSuccess: a rite bearing the
// vox-casts progress and success proclaims each subject once it is set,
// counted among the subjects that change, and its triumph at the end;
// observances count for nothing.
func TestHerald_HearsProgressOfEverySubjectAndSuccess(t *testing.T) {
	fx := newFixture(t)
	util := write(t, fx.data, "util.kdl", "", 0o644)
	theme := filepath.Join(fx.data, "current-theme")
	r := fx.riteWith(t, `"vox-cast": ["progress", "success"],
  "observances": [{"name": "o", "designation": "incantation", "words": "true"}],`,
		`{"name": "util", "designation": "sanctum", "vessel": "$DATA/util.kdl", "omens": {"*": "1"}},
	  {"name": "theme", "designation": "tether", "tether": "$DATA/current-theme", "omens": {"*": "$DATA/themes/{{aspect}}"},
	   "recessional": ["o"]}`)
	h := &hearer{}

	out, _ := perform(t, r, Options{Aspect: "on", Former: "off", Inscriptions: map[string]string{"reason": "gaming"}, Herald: h})

	if out.Verdict != Triumph {
		t.Fatalf("verdict %q", out.Verdict)
	}
	type heard struct {
		tidings               Tidings
		step, steps           int
		kind, target, subject string
	}
	want := []heard{
		{Progress, 1, 2, "sanctum", util, "util"},
		{Progress, 2, 2, "tether", theme, "theme"},
		{Success, 2, 2, "tether", theme, "theme"},
	}
	if len(h.heard) != len(want) {
		t.Fatalf("heard %d proclamations, want %d: %+v", len(h.heard), len(want), h.heard)
	}
	for i, w := range want {
		p := h.heard[i]
		got := heard{p.Tidings, p.Values.Tidings.Step, p.Values.Tidings.Steps,
			p.Values.Tidings.StepKind, p.Values.Tidings.StepTarget, p.Verse.Subject}
		if got != w {
			t.Errorf("proclamation %d: got %+v, want %+v", i, got, w)
		}
		if p.Values.Aspect != "on" || p.Values.Former != "off" || p.Values.Rite != "mouse" ||
			p.Values.Inscriptions["reason"] != "gaming" || p.Outcome != nil || p.Values.Tidings.Heresy != "" ||
			p.Values.Tidings.Subject != w.subject {
			t.Errorf("proclamation %d carries %+v", i, p)
		}
	}
}

// TestHerald_HearsTheFallUnbiddenAfterTheReversions: the fall of an
// observance is proclaimed unbidden once every reversion is done, naming
// what fell and how the reversion went.
func TestHerald_HearsTheFallUnbiddenAfterTheReversions(t *testing.T) {
	fx := newFixture(t)
	write(t, fx.data, "util.kdl", "", 0o644)
	r := fx.riteWith(t, `"vox-cast": ["progress", "success"],
  "observances": [{"name": "woe", "designation": "incantation", "words": "echo woe; exit 3", "reversion": "exit 1"}],`,
		`{"name": "util", "designation": "sanctum", "vessel": "$DATA/util.kdl", "omens": {"*": "1"}},
	  {"name": "x", "designation": "machine-spirit", "enact": {"*": "true"}, "introit": ["woe"]}`)
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
		tid.StepTarget != "echo woe; exit 3" || p.Verse.Observance != "woe" {
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

// TestHerald_HearsNothingOfATriumphWithoutVoxCasts: a rite bearing no
// vox-cast proclaims nothing of its triumph.
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

// TestHerald_NilHearsNothingAndHindersNothing: without a herald the
// invocation is performed and reverted all the same.
func TestHerald_NilHearsNothingAndHindersNothing(t *testing.T) {
	fx := newFixture(t)
	write(t, fx.data, "util.kdl", "", 0o644)
	r := fx.following(t, `{"name": "util", "designation": "sanctum", "vessel": "$DATA/util.kdl", "omens": {"*": "1"}}`,
		`{"designation": "incantation", "words": "exit 1"}`)
	out, _ := perform(t, r, Options{Aspect: "on"})
	if out.Verdict != Reverted || len(out.Deeds) != 2 {
		t.Fatalf("outcome %+v", out)
	}
}

// fallingSubjects are a sanctum and three machine-spirits never read, the
// second of which falls when set into "on"; each is restored by enacting
// the omen of the former aspect, "off".
const fallingSubjects = `{"name": "util", "designation": "sanctum", "vessel": "$DATA/util.kdl", "omens": {"*": "1"}},
  {"name": "a", "designation": "machine-spirit", "enact": {"*": "true"}},
  {"name": "b", "designation": "machine-spirit", "enact": {"on": "exit 3", "*": "true"}},
  {"name": "c", "designation": "machine-spirit", "enact": {"*": "true"}}`

// watcher is a hearer that also watches every real step begin.
type watcher struct {
	hearer
	begun []string
}

func (w *watcher) Begin(v Verse, step, steps int) {
	w.begun = append(w.begun, fmt.Sprintf("%s %d/%d", named(v), step, steps))
}

// TestWatcher_SeesEveryRealStepBegin: a watcher sees every subject begun,
// counted among those that change, until one falls.
func TestWatcher_SeesEveryRealStepBegin(t *testing.T) {
	fx := newFixture(t)
	write(t, fx.data, "util.kdl", "", 0o644)
	r := fx.riteWith(t, `"vox-cast": ["progress", "success"],`, fallingSubjects)
	w := &watcher{}

	out, _ := perform(t, r, Options{Aspect: "on", Former: "off", Herald: w})

	if out.Verdict != Reverted {
		t.Fatalf("verdict %q", out.Verdict)
	}
	want := []string{"util 1/4", "a 2/4", "b 3/4"}
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
	w.seen = append(w.seen, fmt.Sprintf("%s %d/%d", named(v), step, steps))
}

// TestWitness_SeesEveryRealStepPerformedBeforeTheVoxCastsAfterIt: a witness
// sees every subject performed before the progress it is proclaimed by.
func TestWitness_SeesEveryRealStepPerformedBeforeTheVoxCastsAfterIt(t *testing.T) {
	fx := newFixture(t)
	write(t, fx.data, "util.kdl", "", 0o644)
	r := fx.riteWith(t, `"vox-cast": ["progress"],`, fallingSubjects)
	w := &witness{}

	out, _ := perform(t, r, Options{Aspect: "on", Former: "off", Herald: w})

	if out.Verdict != Reverted {
		t.Fatalf("verdict %q", out.Verdict)
	}
	want := []string{"util 1/4", "progress", "a 2/4", "progress", "failure"}
	if !slices.Equal(w.seen, want) {
		t.Fatalf("saw %q, want %q", w.seen, want)
	}
}
