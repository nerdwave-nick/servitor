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
	w.step = stepRite
	f := newForm(m.t)
	f.addText("name", "Name of the rite", w.d.name, "mouse-autohide-toggle",
		"Also the file name in the Librarium. Letters, digits, '.', '_', '-'.",
		func(s string) error {
			if !config.ValidName(s) {
				return errors.New("a rite's name must be letters, digits, '.', '_' and '-'")
			}
			if _, exists := m.s.Librarium.Scriptures[s]; exists && s != w.d.origName {
				return errors.New("a rite of this name is already recorded")
			}
			return nil
		})
	f.addText("description", "Purpose", w.d.description,
		"Hide the mouse cursor after inactivity", "Optional. Shown in the census and in completions.", nil)
	f.addText("states", "Aspects", w.d.states, "on, off",
		"The aspects the rite may take, separated by commas.",
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
		return nil, m.notify(toastInfo, "The consecration is abandoned.")
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
			return w, m.notify(toastErr, "A rite without vessels is an empty prayer. Add one with a.")
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
	v := w.vwork
	f := newForm(m.t)
	if w.vpage == 0 {
		cPrefix, cSuffix := config.DefaultComment(v.file)
		f.addText("file", "Vessel (target file)", v.file, "~/.config/niri/util.kdl",
			"~ and $VARS are expanded. Relative paths are resolved against the Librarium.",
			func(s string) error {
				if strings.TrimSpace(s) == "" {
					return errors.New("a vessel must be named")
				}
				return nil
			})
		f.addText("guard", "Ward", v.guard, w.d.name,
			"Identifies the sanctum within the vessel. Defaults to the rite's name.",
			func(s string) error {
				if strings.ContainsAny(s, " \t") {
					return errors.New("a ward admits no whitespace")
				}
				return nil
			})
		f.addText("comment", "Comment glyph", v.comment, strings.TrimSpace(cPrefix),
			"Leave empty to divine it from the extension.", nil)
		f.addText("comment_end", "Closing glyph", v.commentEnd, cSuffix,
			"Only for block comments such as */ or -->.", nil)
		f.addToggle("create", "Consecrate the vessel if absent", v.create, "space toggles")
		f.addText("inscriptions", "Inscriptions (metadata keys)", v.inscriptions, "reason, mode!",
			"Comma separated; a trailing ! makes an inscription mandatory.",
			func(s string) error { _, _, err := parseInscriptions(s); return err })
		return w.setForm(m, f)
	}
	states, _ := parseStates(w.d.states)
	st := states[w.vpage-1]
	f.addArea("value", fmt.Sprintf("Scripture for aspect %q", st), v.values[st],
		"The lines the sanctum holds in this aspect. Empty is allowed.")
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
