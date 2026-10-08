package tui

import (
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/config"
)

type wizardStep int

const (
	stepRite wizardStep = iota
	stepVessels
	stepVessel // editing one vessel: page 0 = settings, 1..n = one page per state
	stepReview
)

// wizard is the guided, full-screen flow for creating and editing switches.
type wizard struct {
	d       draft
	step    wizardStep
	form    *form
	vcursor int    // selected vessel in the list
	vidx    int    // vessel being edited, -1 for a new one
	vpage   int    // page of the vessel editor
	vwork   vessel // vessel being edited
	review  *review
}

func (m *model) openWizard(d draft) tea.Cmd {
	w := &wizard{d: d}
	m.screen = w
	return w.enterRite(m)
}

func (w *wizard) fullscreen() bool { return true }

func (w *wizard) bodyHeight(m *model) int { return max(5, m.bodyH-7) }

func (w *wizard) enterRite(m *model) tea.Cmd {
	l := m.lex
	w.step = stepRite
	f := newForm(m.t)
	f.addText("name", l.P("Name of the rite", "Name"), w.d.name, "mouse-autohide-toggle",
		l.P("Also the file name in the Librarium. Letters, digits, '.', '_', '-'.", "Also the file name. Letters, digits, '.', '_', '-'."),
		func(s string) error {
			if !config.ValidName(s) {
				return errors.New(l.P("a rite's name must be letters, digits, '.', '_' and '-'", "use letters, digits, '.', '_' and '-'"))
			}
			if _, exists := m.set.Files[s]; exists && s != w.d.origName {
				return errors.New(l.P("a rite of this name is already recorded", "a switch with this name already exists"))
			}
			return nil
		})
	f.addText("description", l.P("Purpose", "Description"), w.d.description,
		"Hide the mouse cursor after inactivity", l.P("Optional. Shown in the census and in completions.", "Optional. Shown in lists and completions."), nil)
	f.addText("states", l.P("Aspects", "States"), w.d.states, "on, off",
		l.P("The aspects the rite may take, separated by commas.", "Comma separated list of states."),
		func(s string) error { _, err := parseStates(s); return err })
	return w.setForm(m, f)
}

func (w *wizard) setForm(m *model, f *form) tea.Cmd {
	f.setWidth(min(80, m.width-8))
	f.height = w.bodyHeight(m)
	w.form = f
	return f.start()
}

func (w *wizard) update(m *model, msg tea.Msg) (screen, tea.Cmd) {
	switch w.step {
	case stepVessels:
		return w.updateVessels(m, msg)
	case stepReview:
		return w.updateReview(m, msg)
	}
	res, cmd := w.form.update(msg)
	if w.step == stepVessel && w.vpage == 0 {
		// The default comment style follows the target file's extension.
		prefix, suffix := config.DefaultComment(w.form.get("file"))
		w.form.setPlaceholder("comment", prefix)
		w.form.setPlaceholder("comment_end", suffix)
	}
	switch {
	case res == formCancel && w.step == stepRite:
		return nil, m.notify(toastInfo, m.lex.P("The consecration is abandoned.", "Discarded."))
	case res == formCancel:
		return w, w.vesselBack(m)
	case res == formSubmit && w.step == stepRite:
		w.d.name, w.d.description, w.d.states = w.form.get("name"), w.form.get("description"), w.form.get("states")
		w.step = stepVessels
		if len(w.d.vessels) == 0 {
			return w, w.editVessel(m, -1)
		}
		return w, nil
	case res == formSubmit:
		return w, w.vesselNext(m)
	}
	return w, cmd
}

func (w *wizard) updateVessels(m *model, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return w, nil
	}
	n := len(w.d.vessels)
	switch k.String() {
	case "esc":
		return w, w.enterRite(m)
	case "up", "k":
		w.vcursor = max(0, w.vcursor-1)
	case "down", "j":
		w.vcursor = min(n-1, w.vcursor+1)
	case "a", "n":
		return w, w.editVessel(m, -1)
	case "enter", "e":
		if n > 0 {
			return w, w.editVessel(m, w.vcursor)
		}
	case "d", "x":
		if n > 0 {
			w.d.vessels = append(w.d.vessels[:w.vcursor], w.d.vessels[w.vcursor+1:]...)
			w.vcursor = max(0, min(w.vcursor, len(w.d.vessels)-1))
		}
	case "K":
		if w.vcursor > 0 {
			v := w.d.vessels
			v[w.vcursor], v[w.vcursor-1] = v[w.vcursor-1], v[w.vcursor]
			w.vcursor--
		}
	case "J":
		if w.vcursor < n-1 {
			v := w.d.vessels
			v[w.vcursor], v[w.vcursor+1] = v[w.vcursor+1], v[w.vcursor]
			w.vcursor++
		}
	case "tab", "ctrl+s", "right", "l":
		if n == 0 {
			return w, m.notify(toastErr, m.lex.P("A rite without vessels is an empty prayer. Add one with a.", "Add at least one file with a."))
		}
		w.step, w.review = stepReview, newReview(m, w.d)
	}
	return w, nil
}

func (w *wizard) editVessel(m *model, idx int) tea.Cmd {
	w.step, w.vidx, w.vpage = stepVessel, idx, 0
	w.vwork = newVessel()
	if idx >= 0 {
		w.vwork = w.d.vessels[idx]
	}
	return w.vesselPage(m)
}

// vesselPage builds the form of the current page of the vessel editor.
func (w *wizard) vesselPage(m *model) tea.Cmd {
	l, v := m.lex, w.vwork
	f := newForm(m.t)
	if w.vpage == 0 {
		cPrefix, cSuffix := config.DefaultComment(v.file)
		f.addText("file", l.P("Vessel (target file)", "Target file"), v.file, "~/.config/niri/util.kdl",
			l.P("~ and $VARS are expanded. Relative paths are resolved against the Librarium.", "~ and $VARS are expanded; relative paths are resolved against the config directory."),
			func(s string) error {
				if strings.TrimSpace(s) == "" {
					return errors.New(l.P("a vessel must be named", "required"))
				}
				return nil
			})
		f.addText("guard", l.P("Ward", "Guard"), v.guard, w.d.name,
			l.P("Identifies the sanctum within the vessel. Defaults to the rite's name.", "Identifies the block in the file. Defaults to the switch name."),
			func(s string) error {
				if strings.ContainsAny(s, " \t") {
					return errors.New(l.P("a ward admits no whitespace", "no whitespace allowed"))
				}
				return nil
			})
		f.addText("comment", l.P("Comment glyph", "Comment prefix"), v.comment, strings.TrimSpace(cPrefix),
			l.P("Leave empty to divine it from the extension.", "Leave empty to infer it from the file extension."), nil)
		f.addText("comment_end", l.P("Closing glyph", "Comment suffix"), v.commentEnd, cSuffix,
			l.P("Only for block comments such as */ or -->.", "Only for block comments such as */ or -->."), nil)
		f.addToggle("create", l.P("Consecrate the vessel if absent", "Create the file if missing"), v.create, l.P("space toggles", "space toggles"))
		f.addText("inscriptions", l.P("Inscriptions (metadata keys)", "Metadata keys"), v.inscriptions, "reason, mode!",
			l.P("Comma separated; a trailing ! makes an inscription mandatory.", "Comma separated; a trailing ! marks a required key."),
			func(s string) error { _, _, err := parseInscriptions(s); return err })
		return w.setForm(m, f)
	}
	states, _ := parseStates(w.d.states)
	st := states[w.vpage-1]
	f.addArea("value", fmt.Sprintf(l.P("Scripture for aspect %q", "Content for state %q"), st), v.values[st],
		l.P("The lines the sanctum holds in this aspect. Empty is allowed.", "The managed block content in this state. May be empty."))
	keys, required, _ := parseInscriptions(v.inscriptions)
	for _, k := range keys {
		label := k
		if required[k] {
			label += " *"
		}
		f.addText("meta:"+k, label, v.meta[st][k], "", v.descs[k], nil)
	}
	return w.setForm(m, f)
}

// vesselNext stores the current page and advances, finishing on the last page.
func (w *wizard) vesselNext(m *model) tea.Cmd {
	states, _ := parseStates(w.d.states)
	if w.vpage == 0 {
		v := &w.vwork
		v.file, v.guard = strings.TrimSpace(w.form.get("file")), strings.TrimSpace(w.form.get("guard"))
		v.comment, v.commentEnd = w.form.get("comment"), w.form.get("comment_end")
		v.create, v.inscriptions = w.form.get("create") == "true", w.form.get("inscriptions")
	} else {
		st := states[w.vpage-1]
		w.vwork.values[st] = w.form.get("value")
		keys, _, _ := parseInscriptions(w.vwork.inscriptions)
		w.vwork.meta[st] = map[string]string{}
		for _, k := range keys {
			w.vwork.meta[st][k] = w.form.get("meta:" + k)
		}
	}
	if w.vpage < len(states) {
		w.vpage++
		return w.vesselPage(m)
	}
	if w.vidx < 0 {
		w.d.vessels = append(w.d.vessels, w.vwork)
		w.vcursor = len(w.d.vessels) - 1
	} else {
		w.d.vessels[w.vidx] = w.vwork
	}
	w.step = stepVessels
	return nil
}

func (w *wizard) vesselBack(m *model) tea.Cmd {
	if w.vpage == 0 {
		w.step = stepVessels
		return nil
	}
	w.vpage--
	return w.vesselPage(m)
}
