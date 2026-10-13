package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// grid is the omens of a subject as one page writes them: a row for each
// aspect, which bears an omen of its own or the omen of every other aspect,
// written under "*"; the row of "*" stands while an aspect bears it.
type grid interface {
	page(f *form, aspects []string)
	keep(f *form, aspects []string)
}

// omenGrid writes omens of type T: add writes the fields of one row, at an
// aspect or "*", and read reads them back.
type omenGrid[T any] struct {
	v    *varied[T]
	add  func(f *form, at, where string, x T)
	read func(f *form, at string) T
}

func (g omenGrid[T]) page(f *form, aspects []string) {
	sharing := false
	for _, a := range aspects {
		x, same := g.v.value(a)
		f.addToggle("same:"+a, fmt.Sprintf("Aspect %q bears the omen of every other aspect", a), same,
			`Written under "*": the omen of every aspect that bears none of its own. space turns it.`)
		if !same {
			g.add(f, a, fmt.Sprintf("aspect %q", a), x)
		}
		sharing = sharing || same
	}
	if sharing {
		g.add(f, librarium.Fallback, "every other aspect (*)", g.v.shared)
	}
}

// keep keeps the rows of the page. An aspect turned to bear the omen of
// every other aspect gives it its own when none is yet written; one turned
// to bear its own begins with that of every other aspect.
func (g omenGrid[T]) keep(f *form, aspects []string) {
	v := g.v
	if f.has("omen:" + librarium.Fallback) {
		v.shared, v.hasShared = g.read(f, librarium.Fallback), true
	}
	for _, a := range aspects {
		same, shown := f.get("same:"+a) == "true", f.has("omen:"+a)
		switch {
		case same && shown && !v.hasShared:
			v.set(a, g.read(f, a), true)
		case same:
			delete(v.own, a)
		case shown:
			v.set(a, g.read(f, a), false)
		default:
			v.set(a, v.shared, false)
		}
	}
}

// omenGrid is the grid of the omens of the subject being written: the
// anchors of a tether, or the scripture of a sanctum or transcription.
func (w *wizard) omenGrid() grid {
	if w.work.kind == librarium.KindTether {
		return omenGrid[librarium.Anchor]{v: &w.work.anchor, add: addAnchor, read: readAnchor}
	}
	transcription := w.work.kind == librarium.KindTranscription
	return omenGrid[librarium.Scripture]{v: &w.work.scripture, read: readScripture,
		add: func(f *form, at, where string, sc librarium.Scripture) {
			f.addArea("omen:"+at, "Omen of "+where+": its scripture", sc.Text,
				"The lines the vessel holds in this aspect; empty is allowed.")
			f.addText("tome:"+at, "Tome of "+where, sc.Tome, "", "Draw the scripture from this tome instead; "+
				"a relative path lies beside the rite's scripture.", nil)
			f.addToggle("illuminate:"+at, "Illuminate the tome of "+where, sc.Illuminate, "Fill the tome's placeholders.")
			if transcription {
				f.addToggle("null:"+at, "Strike the vessel in "+where, sc.Null, "In this aspect the vessel shall not be.")
			}
		}}
}

func readScripture(f *form, at string) librarium.Scripture {
	sc := librarium.Inline(f.get("omen:" + at))
	if tome := strings.TrimSpace(f.get("tome:" + at)); tome != "" {
		sc = librarium.Tome(tome, f.get("illuminate:"+at) == "true")
	}
	if f.get("null:"+at) == "true" {
		sc = librarium.Scripture{Null: true}
	}
	return sc
}

func addAnchor(f *form, at, where string, an librarium.Anchor) {
	f.addText("omen:"+at, "Omen of "+where+": its anchor", an.Path, "~/.config/nfluff/themes/{{aspect}}",
		"Where the tether leads in this aspect; {{aspect}} names the aspect whose omen it is.", nil)
	f.addToggle("null:"+at, "Unbind the tether in "+where, an.Null, "In this aspect the tether shall lead nowhere.")
}

func readAnchor(f *form, at string) librarium.Anchor {
	if f.get("null:"+at) == "true" {
		return librarium.Anchor{Null: true}
	}
	return librarium.Anchor{Path: strings.TrimSpace(f.get("omen:" + at))}
}

// omensPage writes the omens of the subject on one page.
func (w *wizard) omensPage(m *model) tea.Cmd {
	f := newForm(m.t)
	w.omenGrid().page(f, w.d.aspectList())
	return w.setForm(m, pageOmens, 0, f)
}

// omensTurned names the row of the omens page whose aspect turned to bear
// the omen of every other aspect, or its own; "" when none turned.
func (w *wizard) omensTurned() string {
	for _, a := range w.d.aspectList() {
		if (w.form.get("same:"+a) == "true") == w.form.has("omen:"+a) {
			return "same:" + a
		}
	}
	return ""
}

// turnOmens shows the rows as turned, keeping all the page holds.
func (w *wizard) turnOmens(m *model, key string) tea.Cmd {
	w.omenGrid().keep(w.form, w.d.aspectList())
	w.omensPage(m)
	for i, fl := range w.form.fields {
		if fl.key == key {
			return w.form.setFocus(i)
		}
	}
	return nil
}
