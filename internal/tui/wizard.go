package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// station is a stage of the consecration wizard: Rite → Liturgy → Seal,
// or Rite → Subjects → Seal for a rite of aspects of pattern Mark II.
type station int

const (
	stationRite    station = iota
	stationLiturgy         // the liturgy, or the subjects
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
	pageOmens                // the omens of the subject, a row for each aspect
)

// wizard is the guided, full-screen flow for consecrating, amending and
// replicating rites: of pattern Mark I, and of pattern Mark II the edicts
// and the rites of sanctums, transcriptions and tethers.
type wizard struct {
	d        riteDraft
	station  station
	page     pageKind
	index    int   // the inscription or aspect of the page
	form     *form // nil while the liturgy or the subjects are shown
	cursor   int   // the chosen step of the liturgy, or subject
	choosing bool  // the kind of a new step, or designation of a subject, is being chosen
	kind     int   // the kind chosen among w.kinds()
	work     stepDraft
	widx     int // the place of work in the liturgy or subjects; -1 for a new one
	// subject is the subject being written, while work holds its step.
	subject subjectDraft
	seal    *sealing
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
	if !wizardWrites(rite) {
		// Sealing it here would lose what the wizard cannot write.
		return m.notify(toastErr, fmt.Sprintf("The consecration wizard knows not yet the machine-spirits and "+
			"observances of pattern %q; amend the rite %s with o ($EDITOR).", librarium.PatternMarkII, rite.Name))
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
	case w.form == nil && w.d.bearsSubjects():
		return w.updateSubjects(m, msg)
	case w.form == nil:
		return w.updateLiturgy(m, msg)
	}
	res, cmd := w.form.update(msg)
	if key := w.riteTurned(); w.page == pageRite && key != "" {
		return w, w.turnRite(m, key)
	}
	if key := w.omensTurned(); w.page == pageOmens && key != "" {
		return w, w.turnOmens(m, key)
	}
	if w.page == pageEssence && bearsRelics(w.work.kind) && (w.form.get("exalted") == "true") != w.form.has("relics") {
		return w, w.turnExaltation(m)
	}
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
	case pageOmens:
		if w.work.further && hasFurther(w.work.kind) {
			return w, w.furtherPage(m)
		}
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
		w.keepRite()
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
		return w.wordsPage(m)
	case pageFurther:
		for _, k := range furtherKeys[w.work.kind] {
			w.work.fixed[k] = strings.TrimSpace(f.get(k))
		}
		return w.wordsPage(m)
	case pageOmens:
		w.omenGrid().keep(f, w.d.aspectList())
		return w.finishSubject()
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
	n := len(w.d.steps)
	if w.d.bearsSubjects() {
		n = len(w.d.subjects)
	}
	w.cursor = max(0, min(w.cursor, n-1))
	return nil
}

// wordsPage turns to what the step or subject holds per aspect: the pages
// of the aspects of a step, or the omens of a subject.
func (w *wizard) wordsPage(m *model) tea.Cmd {
	if w.d.bearsSubjects() {
		return w.omensPage(m)
	}
	return w.aspectPage(m, 0)
}
