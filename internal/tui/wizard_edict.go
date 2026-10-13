package tui

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// The natures of a rite chosen on the Rite page: one that bears aspects,
// or an edict.
const (
	natureAspects = "aspects"
	natureEdict   = "edict"
)

// oncePage writes the words of an incantation or litany of an edict, each
// spoken once.
func (w *wizard) oncePage(m *model) tea.Cmd {
	f := newForm(m.t)
	for _, v := range w.varying() {
		key, name := v[0], v[1]
		label := strings.ToUpper(name[:1]) + name[1:]
		switch key {
		case "offerings":
			list, _ := w.work.offerings.value(librarium.Fallback)
			f.addArea("offerings", "Offerings, one per line", strings.Join(list, "\n"),
				"Given to the scroll exactly as written, each illuminated.")
		case "command":
			text, _ := w.work.command.value(librarium.Fallback)
			f.addText(key, label, text, "", "Placeholders such as {{rite.name}} and {{inscription.<name>}} are illuminated.",
				required("the "+name+" must be spoken"))
		default:
			text, _ := w.textOf(key).value(librarium.Fallback)
			f.addText(key, label, text, "", "Optional. Spoken to undo this verse should it or a later one fall.", nil)
		}
	}
	return w.setForm(m, pageAspect, 0, f)
}

// keepOnce keeps the words of the edict's step, each spoken once.
func (w *wizard) keepOnce() {
	for _, v := range w.varying() {
		if key := v[0]; key == "offerings" {
			w.work.offerings.setOnce(lines(w.form.get(key)))
		} else {
			w.textOf(key).setOnce(strings.TrimSpace(w.form.get(key)))
		}
	}
}

// kinds are the kinds of step the rite may bear: an edict only speaks,
// recites and sends word, for it holds no aspect to keep, and an edict of
// pattern Mark II sends word by the vox-casts of the rite. A rite of
// aspects of Mark II bears subjects, of the designations the wizard writes.
func (w *wizard) kinds() []librarium.Kind {
	switch {
	case w.station == stationObservances:
		return observanceDesignations
	case w.d.bearsSubjects():
		return subjectDesignations
	case w.d.edict && w.d.markII:
		// the vox-casts of an edict of Mark II are a list of the rite
		return []librarium.Kind{librarium.KindIncantation, librarium.KindLitany}
	case !w.d.edict:
		return librarium.Kinds
	}
	return slices.DeleteFunc(slices.Clone(librarium.Kinds), func(k librarium.Kind) bool { return !edictBears(k) })
}

// edictBears reports whether an edict may bear a step of kind k.
func edictBears(k librarium.Kind) bool {
	return k == librarium.KindIncantation || k == librarium.KindLitany || k == librarium.KindVoxCast
}
