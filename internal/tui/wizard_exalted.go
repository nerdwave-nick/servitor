package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// exaltedHints tell what the exaltation of a verse, subject or observance
// does.
var exaltedHints = map[string]string{
	"verse":      "Performed",
	"subject":    "Set",
	"observance": "Spoken",
}

// exaltedNoun names what the step being written is exalted as: a verse, a
// subject or an observance.
func (w *wizard) exaltedNoun() string {
	switch {
	case w.station == stationObservances:
		return "observance"
	case w.d.bearsSubjects():
		return "subject"
	}
	return "verse"
}

// addExaltation offers the step sd its exaltation as the noun it is
// written as, and exalted speech or an exalted machine-spirit its relics.
func addExaltation(f *form, sd stepDraft, noun string) {
	exalted := sd.fixed["exalted"] == "true"
	f.addToggle("exalted", "Exalted "+noun, exalted, exaltedHints[noun]+" with the machine's own authority, by an "+
		"exalted servitor from the sanction of its master; space turns it.")
	if exalted && bearsRelics(sd.kind) {
		f.addArea("relics", "Relics, one per line", strings.Join(sd.relics, "\n"),
			"The scriptures the exalted speech reads beside its scroll, each reckoned from the hall of the "+
				"rite's scripture and lying within it.")
	}
}

// bearsRelics reports whether a step of kind k, once exalted, bears relics:
// speech, and the enact words of a machine-spirit.
func bearsRelics(k librarium.Kind) bool {
	return k == librarium.KindIncantation || k == librarium.KindLitany || k == kindMachineSpirit
}

// turnExaltation unveils the relics of an incantation or litany once it is
// exalted, and veils them once it is not, keeping all the page holds.
func (w *wizard) turnExaltation(m *model) tea.Cmd {
	w.keepEssence()
	w.essencePage(m)
	for i, fl := range w.form.fields {
		if fl.key == "exalted" {
			return w.form.setFocus(i)
		}
	}
	return nil
}

// keptRelics are the relics written one per line in text. Written as none,
// relics read as an empty list stay so, and are otherwise unwritten.
func keptRelics(text string, old []string) []string {
	var relics []string
	for _, l := range strings.Split(text, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			relics = append(relics, l)
		}
	}
	if relics == nil && old != nil && len(old) == 0 {
		return old
	}
	return relics
}
