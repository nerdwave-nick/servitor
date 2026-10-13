package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// grid is a page of rows, one for each of its keys — the aspects, or the
// omens of a machine-spirit — where a row bears a value of its own or the
// value of every other row, written under "*"; the row of "*" stands while
// a row bears it.
type grid interface {
	page(f *form)
	keep(f *form)
	// turned names the field whose turning reshapes the page; "" when none
	// turned.
	turned(f *form) string
}

// omenGrid writes values of type T keyed by keys: add writes the fields of
// one row, at a key or "*", and read reads them back. noun names a key
// ("aspect" or "omen"), and shares tells what a row bearing the value of
// every other row bears.
type omenGrid[T any] struct {
	v      *varied[T]
	keys   []string
	noun   string
	shares string
	add    func(f *form, at, where string, x T)
	read   func(f *form, at string) T
}

func (g omenGrid[T]) page(f *form) {
	sharing := false
	title := strings.ToUpper(g.noun[:1]) + g.noun[1:]
	for _, k := range g.keys {
		x, same := g.v.value(k)
		f.addToggle("same:"+k, fmt.Sprintf("%s %q %s", title, k, g.shares), same,
			fmt.Sprintf(`Written under "*", for every %s that bears none of its own. space turns it.`, g.noun))
		if !same {
			g.add(f, k, fmt.Sprintf("%s %q", g.noun, k), x)
		}
		sharing = sharing || same
	}
	if sharing {
		g.add(f, librarium.Fallback, "every other "+g.noun+" (*)", g.v.shared)
	}
}

// keep keeps the rows of the page. A row turned to bear the value of every
// other row gives it its own when none is yet written; one turned to bear
// its own begins with that of every other row.
func (g omenGrid[T]) keep(f *form) {
	v := g.v
	if f.has("omen:" + librarium.Fallback) {
		v.shared, v.hasShared = g.read(f, librarium.Fallback), true
	}
	for _, k := range g.keys {
		same, shown := f.get("same:"+k) == "true", f.has("omen:"+k)
		switch {
		case same && shown && !v.hasShared:
			v.set(k, g.read(f, k), true)
		case same:
			delete(v.own, k)
		case shown:
			v.set(k, g.read(f, k), false)
		default:
			v.set(k, v.shared, false)
		}
	}
}

func (g omenGrid[T]) turned(f *form) string {
	for _, k := range g.keys {
		if (f.get("same:"+k) == "true") == f.has("omen:"+k) {
			return "same:" + k
		}
	}
	return ""
}

// gridOf is the grid of the page p of the subject being written: its omens,
// or the auspex or enact words of a machine-spirit.
func (w *wizard) gridOf(p pageKind) grid {
	if w.work.kind == kindMachineSpirit {
		return w.spiritGrid(p)
	}
	aspects := w.d.aspectList()
	if w.work.kind == librarium.KindTether {
		return omenGrid[librarium.Anchor]{v: &w.work.anchor, keys: aspects, noun: "aspect",
			shares: "bears the omen of every other aspect", add: addAnchor, read: readAnchor}
	}
	transcription := w.work.kind == librarium.KindTranscription
	return omenGrid[librarium.Scripture]{v: &w.work.scripture, keys: aspects, noun: "aspect",
		shares: "bears the omen of every other aspect", read: readScripture,
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

// gridPage writes the grid of the page p on one page.
func (w *wizard) gridPage(m *model, p pageKind) tea.Cmd {
	f := newForm(m.t)
	w.gridOf(p).page(f)
	return w.setForm(m, p, 0, f)
}

// turnGrid shows the page as turned by the field key, keeping all the page
// holds.
func (w *wizard) turnGrid(m *model, key string) tea.Cmd {
	w.gridOf(w.page).keep(w.form)
	w.gridPage(m, w.page)
	for i, fl := range w.form.fields {
		if fl.key == key {
			return w.form.setFocus(i)
		}
	}
	return nil
}
