package tui

import (
	"fmt"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

func (w *wizard) view(m *model) string {
	t := m.t
	width, height := m.width, m.bodyH
	title := "Consecration of a new " + w.d.noun()
	if w.d.origName != "" {
		title = "Amendment of the " + w.d.noun() + " " + w.d.origName
	}
	inner := width - 4
	var body string
	switch {
	case w.station == stationSeal:
		body = w.sealView(m, inner, height-5)
	case w.form != nil:
		body = w.formHeading(m) + "\n\n" + w.form.view()
	default:
		body = w.liturgyView(m, inner, height-5)
	}
	content := " " + w.stations(m) + "\n\n" + indentLines(body, " ")
	return t.panel(title, content, width, height, true)
}

func indentLines(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}

// stations renders the breadcrumb of the wizard.
func (w *wizard) stations(m *model) string {
	t := m.t
	stations := []station{stationRite, stationLiturgy, stationSeal}
	names := map[station]string{stationRite: "Rite", stationLiturgy: "Liturgy", stationObservances: "Observances",
		stationSeal: "Seal"}
	if w.d.bearsSubjects() {
		stations = []station{stationRite, stationLiturgy, stationObservances, stationSeal}
		names[stationLiturgy] = "Subjects"
	}
	parts := make([]string, len(stations))
	for i, s := range stations {
		n := names[s]
		label := fmt.Sprintf("%d %s", i+1, n)
		switch {
		case s == w.station:
			parts[i] = t.selected.Render(" " + label + " ")
		case s < w.station:
			parts[i] = t.ok.Render("✔ " + n)
		default:
			parts[i] = t.dim.Render(label)
		}
	}
	return strings.Join(parts, t.dim.Render("  ─  "))
}

// formHeading tells what the current page writes.
func (w *wizard) formHeading(m *model) string {
	t := m.t
	switch w.page {
	case pageRite:
		pattern := "\n" + t.dim.Render("Its scripture is written in pattern "+librarium.PatternMarkII+".")
		if w.d.edict {
			return t.text.Render("Speak the name and purpose of the edict, the inscriptions it bears and its "+
				"vox-casts.") + pattern
		}
		return t.text.Render("Speak the name, purpose and aspects of the rite, or declare it an edict, the "+
			"inscriptions it bears and its vox-casts.") + pattern
	case pageInscription:
		return t.bold.Render(fmt.Sprintf("Inscription %q", w.d.inscriptions[w.index].key)) +
			t.dim.Render(fmt.Sprintf("  ·  %d/%d", w.index+1, len(w.d.inscriptions)))
	}
	if w.d.bearsSubjects() {
		return w.courseHeading(m)
	}
	verse := "New verse"
	if w.widx >= 0 {
		verse = fmt.Sprintf("Verse %d", w.widx+1)
	}
	k := w.work.kind
	pages := 2 // the essence, and the words spoken once
	if w.work.further && hasFurther(k) {
		pages++
	}
	page, at := "essence", 1
	switch w.page {
	case pageFurther:
		page, at = "further rites", 2
	case pageAspect:
		page, at = w.varying()[0][1], pages
	}
	head := t.bold.Render(verse) + t.dim.Render("  ·  ") + t.accent.Render(kindGlyph(k)+" "+k.Key()) +
		t.dim.Render("  ·  ") + t.dim.Render(page) + t.dim.Render(fmt.Sprintf("  ·  %d/%d", at, pages))
	if w.page == pageEssence {
		head += "\n" + t.dim.Render("Its "+w.varying()[0][1]+" is spoken on the page that follows.")
	}
	return head
}

func (w *wizard) hints(m *model) [][2]string {
	switch {
	case w.station == stationSeal:
		return [][2]string{{"enter", "seal into the Librarium"}, {"j/k g/G", "scroll"}, {"esc", "back"}}
	case w.form != nil:
		return [][2]string{{"tab", "next field"}, {"enter", "onward"}, {"ctrl+s", "seal page"}, {"esc", "back"}}
	case w.choosing:
		return [][2]string{{"j/k", "choose"}, {fmt.Sprintf("1-%d", len(w.kinds())), "swift choice"}, {"enter", "take it"},
			{"esc", "withdraw"}}
	case w.station == stationObservances:
		return [][2]string{{"a", "add an observance"}, {"enter", "amend"}, {"d", "strike"},
			{"J/K", "reorder"}, {"tab", "onward to the seal"}, {"esc", "back"}}
	case w.d.bearsSubjects():
		return [][2]string{{"a", "add a subject"}, {"enter", "amend"}, {"d", "strike"},
			{"J/K", "reorder"}, {"tab", "onward to the observances"}, {"esc", "back"}}
	}
	return [][2]string{{"a", "add a step"}, {"enter", "amend"}, {"d", "cast out"},
		{"J/K", "reorder"}, {"tab", "onward to the seal"}, {"esc", "back"}}
}
