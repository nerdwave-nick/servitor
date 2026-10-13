package tui

import (
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// designationWords tell what a subject of each designation keeps.
var designationWords = map[librarium.Kind]string{
	librarium.KindSanctum:       "keep a warded region within a vessel",
	librarium.KindTranscription: "keep a whole vessel, or its absence",
	librarium.KindTether:        "keep a name bound to an anchor, or unbound",
}

// updateSubjects handles the keys of the subjects of a rite of aspects of
// pattern Mark II: a adds one by its designation, enter amends one, d
// strikes it, J and K reorder them.
func (w *wizard) updateSubjects(m *model, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return w, nil
	}
	if w.choosing {
		return w, w.updateChoosing(m, k.String())
	}
	var arranged bool
	if w.d.subjects, arranged = arrange(w.d.subjects, &w.cursor, k.String()); arranged {
		return w, nil
	}
	switch k.String() {
	case "esc":
		return w, w.ritePage(m)
	case "a", "n":
		w.choosing, w.kind = true, 0
	case "enter", "e":
		if len(w.d.subjects) > 0 {
			return w, w.editSubject(m, w.cursor, w.d.subjects[w.cursor])
		}
	case "tab", "ctrl+s", "right", "l":
		if len(w.d.subjects) == 0 {
			return w, m.notify(toastErr, "A rite without subjects has nothing to read or set. Add a subject with a.")
		}
		w.station, w.seal = stationSeal, newSealing(m, w.d)
	}
	return w, nil
}

// editSubject writes the subject sd at place i of the subjects (-1 for a
// new one): its essence, its further rites, then its omens.
func (w *wizard) editSubject(m *model, i int, sd subjectDraft) tea.Cmd {
	w.subject = sd
	return w.editStep(m, i, sd.step)
}

// subjectFields offer the subject its name, fit and its own among the
// subjects, and its purpose.
func (w *wizard) subjectFields(f *form) {
	f.addText("name", "Name of the subject", w.subject.name, "smt",
		"Named in the tidings and the chronicle. Letters, digits, '.', '_', '-'.", func(s string) error {
			s = strings.TrimSpace(s)
			switch {
			case s == "":
				return errors.New("a subject must be named")
			case !librarium.ValidName(s):
				return errors.New("a subject's name must be letters, digits, '.', '_' and '-'")
			}
			for i, o := range w.d.subjects {
				if o.name == s && i != w.widx {
					return errors.New("another subject of the rite bears this name")
				}
			}
			return nil
		})
	f.addText("purpose", "Purpose of the subject", w.subject.purpose, "",
		"Optional. Words for the reader of the scripture.", nil)
}

// finishSubject places the subject written among the subjects.
func (w *wizard) finishSubject() tea.Cmd {
	sd := w.subject
	sd.step = w.work
	w.d.subjects = place(w.d.subjects, w.widx, &w.cursor, sd)
	w.form = nil
	return nil
}

// arrange handles the keys that choose, strike and reorder among list at
// cursor; arranged is false for any other key.
func arrange[T any](list []T, cursor *int, key string) (_ []T, arranged bool) {
	n := len(list)
	switch key {
	case "up", "k":
		*cursor = max(0, *cursor-1)
	case "down", "j":
		*cursor = max(0, min(n-1, *cursor+1))
	case "g", "home":
		*cursor = 0
	case "G", "end":
		*cursor = max(0, n-1)
	case "d", "x":
		if n > 0 {
			list = append(list[:*cursor], list[*cursor+1:]...)
			*cursor = max(0, min(*cursor, len(list)-1))
		}
	case "K":
		if *cursor > 0 {
			list[*cursor], list[*cursor-1] = list[*cursor-1], list[*cursor]
			*cursor--
		}
	case "J":
		if *cursor < n-1 {
			list[*cursor], list[*cursor+1] = list[*cursor+1], list[*cursor]
			*cursor++
		}
	default:
		return list, false
	}
	return list, true
}

// place puts x at its place i of list, or, new (i < 0), after the cursor,
// which then marks it.
func place[T any](list []T, i int, cursor *int, x T) []T {
	if i >= 0 {
		list[i] = x
		return list
	}
	at := 0
	if len(list) > 0 {
		at = *cursor + 1
	}
	*cursor = at
	return append(list[:at], append([]T{x}, list[at:]...)...)
}

// subjectsView shows the subjects of the rite in their order.
func (w *wizard) subjectsView(m *model, width, height int) string {
	t := m.t
	lines := []string{t.text.Render("The subjects of the rite " + w.d.name + ", read and set in order:"), ""}
	if len(w.d.subjects) == 0 {
		lines = append(lines, t.dim.Render("  No subject is declared yet. Press a to add one."))
	}
	aspects := w.d.aspectList()
	avail := max(1, height-len(lines))
	start := max(0, min(w.cursor-avail/2, len(w.d.subjects)-avail))
	for i := start; i < min(len(w.d.subjects), start+avail); i++ {
		sd := w.d.subjects[i]
		line := t.dim.Render(fmt.Sprint(i+1)) + " " + t.accent.Render(kindGlyph(sd.step.kind)) + " " +
			t.text.Render(sd.name) + t.dim.Render("  "+stepWords(sd.step.toStep(aspects), ""))
		lines = append(lines, w.marked(m, i == w.cursor, line, width))
	}
	return strings.Join(lines, "\n")
}
