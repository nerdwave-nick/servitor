package tui

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// pages are the pages the subject or observance being written passes, in
// order: its essence and further rites; then the words of an observance
// and its guards; or the omens of a subject, the auspex and enact words of
// a machine-spirit, and the observances it names, while the rite bears any.
func (w *wizard) pages() []pageKind {
	pages := []pageKind{pageEssence}
	if w.work.further && hasFurther(w.work.kind) {
		pages = append(pages, pageFurther)
	}
	if w.station == stationObservances {
		return append(pages, pageAspect, pageGuards)
	}
	pages = append(pages, pageOmens)
	if w.work.kind == kindMachineSpirit {
		pages = append(pages, pageAuspex, pageEnact)
	}
	if len(w.d.observances) > 0 {
		pages = append(pages, pageRites)
	}
	return pages
}

// pageNames name the pages of a subject or observance in its heading.
var pageNames = map[pageKind]string{pageEssence: "essence", pageFurther: "further rites", pageOmens: "omens",
	pageAuspex: "auspex", pageEnact: "enact", pageRites: "observances", pageAspect: "words", pageGuards: "guards"}

// course turns by pages along the pages of the subject or observance
// being written: back to its list from the first, and placing it among
// them once the last is left.
func (w *wizard) course(m *model, by int) tea.Cmd {
	pages := w.pages()
	at := slices.Index(pages, w.page) + by
	switch {
	case at < 0:
		w.form = nil
		return nil
	case at >= len(pages) && w.station == stationObservances:
		return w.finishObservance()
	case at >= len(pages):
		return w.finishSubject()
	}
	return w.openPage(m, pages[at])
}

// openPage shows the page p of the subject or observance being written.
func (w *wizard) openPage(m *model, p pageKind) tea.Cmd {
	switch p {
	case pageEssence:
		return w.essencePage(m)
	case pageFurther:
		return w.furtherPage(m)
	case pageOmens, pageAuspex, pageEnact:
		return w.gridPage(m, p)
	case pageRites:
		return w.ritesPage(m)
	case pageGuards:
		return w.guardsPage(m)
	}
	return w.oncePage(m)
}

// keepPage keeps what the page of the subject or observance holds.
func (w *wizard) keepPage() {
	switch w.page {
	case pageEssence:
		w.keepEssence()
	case pageFurther:
		w.keepFurther()
	case pageOmens, pageAuspex, pageEnact:
		w.gridOf(w.page).keep(w.form)
	case pageRites:
		w.keepRites()
	case pageGuards:
		w.keepGuards()
	case pageAspect:
		w.keepOnce()
	}
}

// courseHeading tells which subject or observance the page writes, and
// which of its pages.
func (w *wizard) courseHeading(m *model) string {
	t := m.t
	noun, pages := "subject", w.pages()
	if w.station == stationObservances {
		noun = "observance"
	}
	head := "New " + noun
	if w.widx >= 0 {
		head = fmt.Sprintf("%s%s %d", strings.ToUpper(noun[:1]), noun[1:], w.widx+1)
	}
	page := pageNames[w.page]
	if w.page == pageAspect {
		page = w.varying()[0][1]
	}
	k := w.work.kind
	head = t.bold.Render(head) + t.dim.Render("  ·  ") + t.accent.Render(kindGlyph(k)+" "+kindKey(k)) +
		t.dim.Render("  ·  ") + t.dim.Render(page) +
		t.dim.Render(fmt.Sprintf("  ·  %d/%d", slices.Index(pages, w.page)+1, len(pages)))
	words := courseWords[w.page]
	if w.page == pageEssence {
		words = "Its omens, the mark each aspect leaves upon it, are written on the page of omens."
		if w.station == stationObservances {
			words = "Its " + w.varying()[0][1] + " is spoken on a page that follows."
		}
	}
	if words != "" {
		head += "\n" + t.dim.Render(words)
	}
	return head
}

// courseWords tell what each page of a subject or observance writes.
var courseWords = map[pageKind]string{
	pageOmens: `Each aspect bears an omen of its own, or the omen of every other aspect, written under "*".`,
	pageAuspex: "The auspex reads the omen the subject shows: by its scan, or by a trial for each omen. " +
		"A subject never read is set at every invocation.",
	pageEnact: `Each omen is set by words of its own, or by the words of every other omen, written under "*".`,
	pageRites: "An introit is spoken once before the first changing subject that names it; a recessional, " +
		"once after the last.",
	pageGuards: "Unturned, the observance is spoken into every aspect, and out of every aspect.",
}
