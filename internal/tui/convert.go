package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
)

// converted tells in a toast what the conversion of the Librarium wrote:
// the one scripture converted, or how many, each named; a lament before
// all. Init lets the toast of the first reading fade; a later one stays
// until the next toast.
func (m *model) converted(told rituals.Converted) {
	var t toast
	switch n := len(told.Tidings); {
	case len(told.Laments) > 0:
		t = toast{kind: toastErr, text: told.Laments[0].Error()}
	case n == 1:
		t = toast{kind: toastInfo, text: told.Tidings[0]}
	case n > 1:
		t = toast{kind: toastInfo, text: fmt.Sprintf("%d scriptures are converted to pattern %q: %s", n,
			librarium.PatternMarkII, strings.Join(told.Tidings, " · "))}
	default:
		return
	}
	t.id = m.toast.id + 1
	m.toast = t
}

// Init awakens the auspexes of the first reading, and lets its toast fade.
func (m *model) Init() tea.Cmd {
	m.watch()
	if m.toast.text == "" {
		return nil
	}
	return fade(m.toast.id)
}
