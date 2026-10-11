package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// A heresy too long for the viewscreen keeps its colour on every line it is
// broken into, so the scroll of a fallen rite never turns plain mid-sentence.
func TestWrap_AStyledLineKeepsItsStyleOnEveryLine(t *testing.T) {
	th := newTheme()
	heresy := "the invocation was halted at its master's command; the servitor slew " +
		"whatever the step had summoned and began no further step"
	styled := th.danger.Render(heresy)
	pen := styled[:strings.Index(styled, "the")]

	lines := wrap([]string{styled}, 40)

	if len(lines) < 2 {
		t.Fatalf("wrap gave %d line(s), want the heresy broken: %q", len(lines), lines)
	}
	for i, l := range lines {
		if !strings.Contains(l, pen) {
			t.Errorf("line %d lost the colour of the heresy: %q", i+1, l)
		}
	}
	var words []string
	for _, l := range lines {
		words = append(words, strings.Fields(ansi.Strip(l))...)
	}
	if got := strings.Join(words, " "); got != heresy {
		t.Errorf("the words were changed by wrapping:\n got %q\nwant %q", got, heresy)
	}
}
