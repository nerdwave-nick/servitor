package vox

import (
	"errors"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// TestCompose_TheFallOfAnObservanceNamesIt: the fall of an observance of a
// state rite is told as a fallen incantation's, its heresy naming the
// observance; a reversion that faltered names every observance and subject
// that could not be undone.
func TestCompose_TheFallOfAnObservanceNamesIt(t *testing.T) {
	v := placeholder.Values{Rite: "dnd", Aspect: "on", Tidings: placeholder.Tidings{Step: 2, Steps: 1,
		StepKind: "incantation", StepTarget: "notify-send silence",
		Heresy: `the observance "herald-silence" fell: it ended bearing the death-mark 3`}}
	out := &invocation.Outcome{Verdict: invocation.Faltered, Reversions: []invocation.Reversion{
		{Verse: invocation.Verse{Observance: "herald-silence"}, Heresy: errors.New("x")},
		{Verse: invocation.Verse{Subject: "mako"}}, {Verse: invocation.Verse{Observance: "veil"}, Heresy: errors.New("y")}}}
	for i := range templates[invocation.Failure] {
		m := Compose(invocation.Proclamation{Tidings: invocation.Failure, Values: v, Outcome: out}, func(int) int { return i })
		if !strings.Contains(m.Flavour, `observance "herald-silence" fell`) || !strings.HasSuffix(m.Body,
			"\nThe reversion faltered at observance herald-silence, observance veil; the rite lies corrupted.") {
			t.Errorf("failure template %d composed %+v", i, m)
		}
	}
}
