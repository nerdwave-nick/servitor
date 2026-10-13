package tui

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// ritePage writes the rite itself: its name, purpose and nature, the
// aspects of a rite of aspects (an edict is asked none), its inscriptions,
// its vox-casts and its tongue.
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
	nature := natureAspects
	if d.edict {
		nature = natureEdict
	}
	f.addChoice("edict", "Nature", []string{natureAspects, natureEdict}, nature,
		"A rite brings the machine into one of its aspects; an edict bears none and proclaims its whole "+
			"liturgy anew at every invocation.")
	if !d.edict {
		f.addText("aspects", "Aspects", d.aspects, "on, off",
			"The aspects the rite may bring the machine into, separated by commas.",
			func(s string) error { _, err := parseAspects(s); return err })
	}
	f.addText("inscriptions", "Inscriptions", strings.Join(d.keys(), ", "), "reason, mode",
		"Optional, separated by commas; each is spoken as a rune and illuminated as {{inscription.<name>}}.",
		func(s string) error { _, err := parseKeys(s); return err })
	progress := "Tells of every subject that changed."
	if d.edict {
		progress = "Tells of every verse performed."
	}
	f.addToggle("vox:"+librarium.VoxProgress, "Vox-cast of progress",
		slices.Contains(d.voxCasts, librarium.VoxProgress), progress+" space turns it.")
	f.addToggle("vox:"+librarium.VoxSuccess, "Vox-cast of success",
		slices.Contains(d.voxCasts, librarium.VoxSuccess), "Proclaims the triumph. A fall is proclaimed unbidden.")
	f.addText("tongue", "Tongue", d.tongue, "bash",
		"The program that speaks the rite's commands; unspoken, the tongue of the settings, or bash.", validWord)
	return w.setForm(m, pageRite, 0, f)
}

// keepRite keeps what the Rite page holds.
func (w *wizard) keepRite() {
	f, d := w.form, &w.d
	d.name, d.purpose, d.tongue = strings.TrimSpace(f.get("name")), f.get("purpose"), f.get("tongue")
	d.edict = f.get("edict") == natureEdict
	if f.has("aspects") {
		d.aspects = f.get("aspects")
	}
	d.voxCasts = voxCasts(f.get("vox:"+librarium.VoxProgress) == "true",
		f.get("vox:"+librarium.VoxSuccess) == "true", d.voxCasts)
	if keys, err := parseKeys(f.get("inscriptions")); err == nil {
		d.declare(keys)
	}
}

// voxCasts are the vox-casts chosen; written as before when the same are
// chosen, in their order, and unwritten when none is chosen anew.
func voxCasts(progress, success bool, before []string) []string {
	var chosen []string
	if progress {
		chosen = append(chosen, librarium.VoxProgress)
	}
	if success {
		chosen = append(chosen, librarium.VoxSuccess)
	}
	if len(chosen) == len(before) && !slices.ContainsFunc(chosen, func(v string) bool {
		return !slices.Contains(before, v)
	}) {
		return before
	}
	return chosen
}

// riteTurned names the choice of the Rite page that turned its nature; ""
// when it did not turn.
func (w *wizard) riteTurned() string {
	if (w.form.get("edict") == natureEdict) != w.d.edict {
		return "edict"
	}
	return ""
}

// turnRite turns the Rite page to the nature chosen by the field key,
// keeping all that was written upon it.
func (w *wizard) turnRite(m *model, key string) tea.Cmd {
	old := w.form
	w.keepRite()
	w.ritePage(m)
	at := 0
	for i, fl := range w.form.fields {
		if fl.key == key {
			at = i
		}
		for _, o := range old.fields {
			switch {
			case o.key != fl.key || o.kind != fl.kind:
			case fl.kind == fieldText:
				fl.input.SetValue(o.input.Value())
			case fl.kind == fieldToggle:
				fl.on = o.on
			}
		}
	}
	return w.form.setFocus(at)
}

// inscriptionPage writes the inscription i: its decrees per aspect, or
// one plain decree for an edict.
func (w *wizard) inscriptionPage(m *model, i int) tea.Cmd {
	in := w.d.inscriptions[i]
	f := newForm(m.t)
	f.addText("purpose", "Purpose", in.purpose, "why the rite was invoked",
		"Optional. Told beside its rune in the completions.", nil)
	if w.d.edict {
		f.addToggle("mandatory", "Mandatory", in.mandatory,
			"A proclamation lacking this inscription, with no decree, is refused.")
		f.addText("decree:*", "Decree", in.decrees[librarium.Fallback], "",
			"Inscribed when the edict is proclaimed and no rune speaks otherwise; empty decrees nothing.", nil)
		return w.setForm(m, pageInscription, i, f)
	}
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
