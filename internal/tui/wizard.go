package tui

import (
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// station is a stage of the consecration wizard: Rite → Liturgy → Seal.
type station int

const (
	stationRite station = iota
	stationLiturgy
	stationSeal
)

// pageKind is what the wizard's current form writes.
type pageKind int

const (
	pageRite        pageKind = iota
	pageInscription          // one inscription, index
	pageEssence              // the essential fields of the step
	pageFurther              // the further rites of the step
	pageAspect               // the aspect-varying fields of the step for aspect index
)

// wizard is the guided, full-screen flow for consecrating, amending and
// replicating rites of pattern Mark I.
type wizard struct {
	d        riteDraft
	station  station
	page     pageKind
	index    int   // the inscription or aspect of the page
	form     *form // nil while the liturgy itself is shown
	cursor   int   // the chosen step of the liturgy
	choosing bool  // the kind of a new step is being chosen
	kind     int   // the kind chosen among librarium.Kinds
	work     stepDraft
	widx     int // the place of work in the liturgy; -1 for a new step
	seal     *sealing
}

func (m *model) openWizard(d riteDraft) tea.Cmd {
	w := &wizard{d: d}
	m.screen = w
	return w.ritePage(m)
}

// amend opens the consecration wizard upon the rite of the row, to amend
// it (e) or to replicate it (c).
func (m *model) amend(key string, r *row) tea.Cmd {
	rite := r.rite()
	if rite == nil {
		return m.notify(toastErr, "The consecration wizard cannot read a heretical rite; purify it with o ($EDITOR) "+
			"or excommunicate it with d.")
	}
	d := riteDraftFrom(rite)
	if key == "c" {
		d.origName, d.origPath, d.name = "", "", r.name+"-copy"
	}
	return m.openWizard(d)
}

func (w *wizard) fullscreen() bool { return true }

func (w *wizard) bodyHeight(m *model) int { return max(5, m.bodyH-7) }

func (w *wizard) setForm(m *model, page pageKind, index int, f *form) tea.Cmd {
	f.setWidth(min(80, m.width-8))
	f.height = w.bodyHeight(m)
	w.form, w.page, w.index = f, page, index
	return f.start()
}

func (w *wizard) update(m *model, msg tea.Msg) (screen, tea.Cmd) {
	switch {
	case w.station == stationSeal:
		return w.updateSeal(m, msg)
	case w.form == nil:
		return w.updateLiturgy(m, msg)
	}
	res, cmd := w.form.update(msg)
	if w.page == pageEssence && w.work.kind == librarium.KindSanctum {
		// The glyphs follow the vessel's extension unless spoken.
		glyph, closing := librarium.DefaultGlyphs(w.form.get("sanctum"))
		w.form.setPlaceholder("glyph", glyph)
		w.form.setPlaceholder("closing-glyph", closing)
	}
	switch res {
	case formCancel:
		return w.back(m)
	case formSubmit:
		return w, w.onward(m)
	}
	return w, cmd
}

// back withdraws to the page before the current one.
func (w *wizard) back(m *model) (screen, tea.Cmd) {
	switch w.page {
	case pageRite:
		return nil, m.notify(toastInfo, "The consecration is abandoned.")
	case pageInscription:
		if w.index > 0 {
			return w, w.inscriptionPage(m, w.index-1)
		}
		return w, w.ritePage(m)
	case pageEssence:
		w.form = nil
		return w, nil
	case pageFurther:
		return w, w.essencePage(m)
	}
	switch {
	case w.index > 0:
		return w, w.aspectPage(m, w.index-1)
	case w.work.further && hasFurther(w.work.kind):
		return w, w.furtherPage(m)
	}
	return w, w.essencePage(m)
}

// onward keeps what the current page holds and turns to the next.
func (w *wizard) onward(m *model) tea.Cmd {
	f := w.form
	switch w.page {
	case pageRite:
		d := &w.d
		d.name, d.purpose, d.aspects = strings.TrimSpace(f.get("name")), f.get("purpose"), f.get("aspects")
		d.auspex, d.patience, d.tongue = f.get("auspex"), f.get("patience"), f.get("tongue")
		keys, _ := parseKeys(f.get("inscriptions"))
		d.declare(keys)
		return w.afterInscription(m, -1)
	case pageInscription:
		in := &w.d.inscriptions[w.index]
		in.purpose, in.mandatory = f.get("purpose"), f.get("mandatory") == "true"
		in.decrees = map[string]string{}
		for _, a := range append(w.d.aspectList(), librarium.Fallback) {
			in.decrees[a] = strings.TrimSpace(f.get("decree:" + a))
		}
		return w.afterInscription(m, w.index)
	case pageEssence:
		w.keepEssence()
		switch {
		case w.work.kind == librarium.KindVoxCast:
			return w.finishStep()
		case w.work.further && hasFurther(w.work.kind):
			return w.furtherPage(m)
		}
		return w.aspectPage(m, 0)
	case pageFurther:
		for _, k := range furtherKeys[w.work.kind] {
			w.work.fixed[k] = strings.TrimSpace(f.get(k))
		}
		return w.aspectPage(m, 0)
	}
	if w.keepAspect() {
		return w.finishStep()
	}
	return w.aspectPage(m, w.index+1)
}

// afterInscription turns to the inscription after i, or to the liturgy.
func (w *wizard) afterInscription(m *model, i int) tea.Cmd {
	if i+1 < len(w.d.inscriptions) {
		return w.inscriptionPage(m, i+1)
	}
	w.station, w.form = stationLiturgy, nil
	w.cursor = max(0, min(w.cursor, len(w.d.steps)-1))
	return nil
}

func (w *wizard) ritePage(m *model) tea.Cmd {
	w.station = stationRite
	d := w.d
	f := newForm(m.t)
	f.addText("name", "Name of the rite", d.name, "mouse-autohide-toggle",
		"Also the name of its scripture in the Librarium. Letters, digits, '.', '_', '-'.",
		func(s string) error {
			s = strings.TrimSpace(s)
			if !librarium.ValidName(s) {
				return errors.New("a rite's name must be letters, digits, '.', '_' and '-'")
			}
			if _, exists := m.s.Librarium.Scriptures[s]; exists && s != d.origName {
				return errors.New("a rite of this name is already recorded")
			}
			return nil
		})
	f.addText("purpose", "Purpose", d.purpose, "Hide the cursor of the machine",
		"Optional. Told in the census, the cogitator and the completions.", nil)
	f.addText("aspects", "Aspects", d.aspects, "on, off",
		"The aspects the rite may bring the machine into, separated by commas.",
		func(s string) error { _, err := parseAspects(s); return err })
	f.addText("inscriptions", "Inscriptions", strings.Join(d.keys(), ", "), "reason, mode",
		"Optional, separated by commas; each is spoken as a rune and illuminated as {{inscription.<name>}}.",
		func(s string) error { _, err := parseKeys(s); return err })
	f.addText("auspex", "Auspex", d.auspex, "makoctl mode | grep -q do-not-disturb && echo on || echo off",
		"Optional. A command that speaks the aspect the machine stands in.", nil)
	f.addText("patience", "Patience of the auspex", d.patience, "2s",
		"How long the auspex may labour, such as 500ms or 5s.", validPatience)
	f.addText("tongue", "Tongue", d.tongue, "bash",
		"The program that speaks the rite's commands; unspoken, the tongue of the settings, or bash.", validWord)
	return w.setForm(m, pageRite, 0, f)
}

func (w *wizard) inscriptionPage(m *model, i int) tea.Cmd {
	in := w.d.inscriptions[i]
	f := newForm(m.t)
	f.addText("purpose", "Purpose", in.purpose, "why the rite was invoked",
		"Optional. Told beside its rune in the completions.", nil)
	f.addToggle("mandatory", "Mandatory", in.mandatory,
		"An invocation lacking this inscription, with no decree for its aspect, is refused.")
	for _, a := range w.d.aspectList() {
		f.addText("decree:"+a, fmt.Sprintf("Decree for aspect %q", a), in.decrees[a], "",
			"Inscribed when the rite is invoked into this aspect and no rune speaks otherwise; empty decrees nothing.", nil)
	}
	f.addText("decree:*", "Decree for every other aspect (*)", in.decrees[librarium.Fallback], "",
		"Serves every aspect without a decree of its own.", nil)
	return w.setForm(m, pageInscription, i, f)
}
