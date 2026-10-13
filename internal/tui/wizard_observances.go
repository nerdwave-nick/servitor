package tui

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// observanceDesignations are the designations of an observance, as the
// kinds of step that write them.
var observanceDesignations = []librarium.Kind{librarium.KindIncantation, librarium.KindLitany}

// updateObservances handles the keys of the observances of a rite of
// aspects of pattern Mark II: a adds one by its designation, enter amends
// one, d strikes it — and every subject's naming of it — J and K reorder
// them.
func (w *wizard) updateObservances(m *model, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return w, nil
	}
	if w.choosing {
		return w, w.updateChoosing(m, k.String())
	}
	var struck string
	if n := len(w.d.observances); n > 0 && (k.String() == "d" || k.String() == "x") {
		struck = w.d.observances[w.cursor].name
	}
	var arranged bool
	if w.d.observances, arranged = arrange(w.d.observances, &w.cursor, k.String()); arranged {
		if struck != "" {
			w.d.rename(struck, "")
		}
		return w, nil
	}
	switch k.String() {
	case "esc":
		w.station = stationLiturgy
		w.cursor = 0
	case "a", "n":
		w.choosing, w.kind = true, 0
	case "enter", "e":
		if len(w.d.observances) > 0 {
			return w, w.editObservance(m, w.cursor, w.d.observances[w.cursor])
		}
	case "tab", "ctrl+s", "right", "l":
		w.station, w.seal = stationSeal, newSealing(m, w.d)
	}
	return w, nil
}

// rename renames the observance old to name wherever a subject names it,
// or strikes it there when name is "".
func (d *riteDraft) rename(old, name string) {
	for i := range d.subjects {
		s := &d.subjects[i]
		s.introit, s.recessional = renamed(s.introit, old, name), renamed(s.recessional, old, name)
	}
}

// editObservance writes the observance od at place i of the observances
// (-1 for a new one): its essence, its further rites, its words, then its
// guards.
func (w *wizard) editObservance(m *model, i int, od observanceDraft) tea.Cmd {
	w.observance = od
	return w.editStep(m, i, od.step)
}

// observanceFields offer the observance its name, fit and its own among
// the observances.
func (w *wizard) observanceFields(f *form) {
	f.addText("name", "Name of the observance", w.observance.name, "herald-silence",
		"Named by the subjects that speak it, in the tidings and the chronicle. Letters, digits, '.', '_', '-'.",
		func(s string) error {
			s = strings.TrimSpace(s)
			switch {
			case s == "":
				return errors.New("an observance must be named")
			case !librarium.ValidName(s):
				return errors.New("an observance's name must be letters, digits, '.', '_' and '-'")
			}
			for i, o := range w.d.observances {
				if o.name == s && i != w.widx {
					return errors.New("another observance of the rite bears this name")
				}
			}
			return nil
		})
}

// finishObservance places the observance written among the observances,
// and follows its new name wherever a subject names it.
func (w *wizard) finishObservance() tea.Cmd {
	od := w.observance
	od.step = w.work
	if w.widx >= 0 {
		if old := w.d.observances[w.widx].name; old != od.name {
			w.d.rename(old, od.name)
		}
	}
	w.d.observances = place(w.d.observances, w.widx, &w.cursor, od)
	w.form = nil
	return nil
}

// guardsPage writes the guards of the observance: the aspects it is
// spoken into, and out of.
func (w *wizard) guardsPage(m *model) tea.Cmd {
	f := newForm(m.t)
	for _, g := range []struct {
		key, label string
		list       []string
	}{{"to", "Spoken into aspect %q", w.observance.to}, {"from", "Spoken out of aspect %q", w.observance.from}} {
		for _, a := range w.d.aspectList() {
			f.addToggle(g.key+":"+a, fmt.Sprintf(g.label, a), slices.Contains(g.list, a),
				"Turned for none, it is spoken whatever the aspect. space turns it.")
		}
	}
	return w.setForm(m, pageGuards, 0, f)
}

// keepGuards keeps the guards of the observance.
func (w *wizard) keepGuards() {
	chosen := func(key string) []string {
		var aspects []string
		for _, a := range w.d.aspectList() {
			if w.form.get(key+":"+a) == "true" {
				aspects = append(aspects, a)
			}
		}
		return aspects
	}
	w.observance.to = keptChoice(w.observance.to, chosen("to"))
	w.observance.from = keptChoice(w.observance.from, chosen("from"))
}

// Where a subject names an observance: nowhere, or as one of its introits
// or recessionals.
const (
	ritesUnheeded    = "unheeded"
	ritesIntroit     = "introit"
	ritesRecessional = "recessional"
)

// ritesPage writes which observances the subject names as its introits
// and recessionals, one choice for each observance.
func (w *wizard) ritesPage(m *model) tea.Cmd {
	f := newForm(m.t)
	for _, o := range w.d.observances {
		at := ritesUnheeded
		switch {
		case slices.Contains(w.subject.introit, o.name):
			at = ritesIntroit
		case slices.Contains(w.subject.recessional, o.name):
			at = ritesRecessional
		}
		f.addChoice("rite:"+o.name, fmt.Sprintf("Observance %q", o.name),
			[]string{ritesUnheeded, ritesIntroit, ritesRecessional}, at,
			"Spoken before the subject changes, or after it changed; space turns it.")
	}
	return w.setForm(m, pageRites, 0, f)
}

// keepRites keeps the introits and recessionals of the subject, in the
// order of the observances, or as written while the same are chosen.
func (w *wizard) keepRites() {
	var introit, recessional []string
	for _, o := range w.d.observances {
		switch w.form.get("rite:" + o.name) {
		case ritesIntroit:
			introit = append(introit, o.name)
		case ritesRecessional:
			recessional = append(recessional, o.name)
		}
	}
	w.subject.introit = keptChoice(w.subject.introit, introit)
	w.subject.recessional = keptChoice(w.subject.recessional, recessional)
}

// observancesView shows the observances of the rite in their order.
func (w *wizard) observancesView(m *model, width, height int) string {
	t := m.t
	lines := []string{t.text.Render("The observances of the rite " + w.d.name + ", spoken as its subjects name them:"), ""}
	if len(w.d.observances) == 0 {
		lines = append(lines, t.dim.Render("  No observance is declared yet. Press a to add one, or tab to pass on."))
	}
	avail := max(1, height-len(lines))
	start := max(0, min(w.cursor-avail/2, len(w.d.observances)-avail))
	for i := start; i < min(len(w.d.observances), start+avail); i++ {
		od := w.d.observances[i]
		line := t.dim.Render(fmt.Sprint(i+1)) + " " + t.accent.Render(kindGlyph(od.step.kind)) + " " +
			t.text.Render(od.name) + t.dim.Render(guard("to", od.to)+guard("from", od.from)+
			"  "+stepWords(od.step.toStep(nil), ""))
		lines = append(lines, w.marked(m, i == w.cursor, line, width))
	}
	return strings.Join(lines, "\n")
}
