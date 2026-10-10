package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// kindWords tell what each kind of step does, in the order of the codex.
var kindWords = map[librarium.Kind]string{
	librarium.KindSanctum:       "keep a warded region within a vessel",
	librarium.KindTranscription: "transcribe a whole vessel anew, or strike it",
	librarium.KindTether:        "bind a name to an anchor",
	librarium.KindIncantation:   "speak one line of command in a tongue",
	librarium.KindLitany:        "recite a scroll, given its offerings",
	librarium.KindVoxCast:       "proclaim progress or triumph",
}

func (w *wizard) updateLiturgy(m *model, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return w, nil
	}
	if w.choosing {
		return w, w.updateChoosing(m, k.String())
	}
	steps := w.d.steps
	n := len(steps)
	switch k.String() {
	case "esc":
		return w, w.ritePage(m)
	case "up", "k":
		w.cursor = max(0, w.cursor-1)
	case "down", "j":
		w.cursor = max(0, min(n-1, w.cursor+1))
	case "g", "home":
		w.cursor = 0
	case "G", "end":
		w.cursor = max(0, n-1)
	case "a", "n":
		w.choosing, w.kind = true, 0
	case "enter", "e":
		if n > 0 {
			return w, w.editStep(m, w.cursor, steps[w.cursor])
		}
	case "d", "x":
		if n > 0 {
			w.d.steps = append(steps[:w.cursor], steps[w.cursor+1:]...)
			w.cursor = max(0, min(w.cursor, len(w.d.steps)-1))
		}
	case "K":
		if w.cursor > 0 {
			steps[w.cursor], steps[w.cursor-1] = steps[w.cursor-1], steps[w.cursor]
			w.cursor--
		}
	case "J":
		if w.cursor < n-1 {
			steps[w.cursor], steps[w.cursor+1] = steps[w.cursor+1], steps[w.cursor]
			w.cursor++
		}
	case "tab", "ctrl+s", "right", "l":
		if n == 0 {
			return w, m.notify(toastErr, "A rite without a liturgy is an empty prayer. Add a step with a.")
		}
		w.station, w.seal = stationSeal, newSealing(m, w.d)
	}
	return w, nil
}

func (w *wizard) updateChoosing(m *model, key string) tea.Cmd {
	kinds := librarium.Kinds
	switch key {
	case "esc", "q":
		w.choosing = false
	case "up", "k":
		w.kind = max(0, w.kind-1)
	case "down", "j":
		w.kind = min(len(kinds)-1, w.kind+1)
	case "enter":
		w.choosing = false
		return w.editStep(m, -1, newStepDraft(kinds[w.kind]))
	default:
		if len(key) == 1 && key[0] >= '1' && int(key[0]-'1') < len(kinds) {
			w.choosing = false
			return w.editStep(m, -1, newStepDraft(kinds[key[0]-'1']))
		}
	}
	return nil
}

// editStep writes the step sd at place i of the liturgy (-1 for a new one).
func (w *wizard) editStep(m *model, i int, sd stepDraft) tea.Cmd {
	w.widx, w.work = i, sd
	w.work.fixed = make(map[string]string, len(sd.fixed))
	for k, v := range sd.fixed {
		w.work.fixed[k] = v
	}
	w.work.scripture.own = copyMap(sd.scripture.own)
	w.work.anchor.own = copyMap(sd.anchor.own)
	w.work.command.own = copyMap(sd.command.own)
	w.work.reversion.own = copyMap(sd.reversion.own)
	w.work.offerings.own = copyMap(sd.offerings.own)
	return w.essencePage(m)
}

func copyMap[T any](src map[string]T) map[string]T {
	dst := make(map[string]T, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// liturgyView shows the steps of the liturgy, or the kinds to choose from.
func (w *wizard) liturgyView(m *model, width, height int) string {
	t := m.t
	if w.choosing {
		lines := []string{t.text.Render("Choose the kind of the new step:"), ""}
		for i, k := range librarium.Kinds {
			line := fmt.Sprintf("%d %s %-14s", i+1, kindGlyph(k), k.Key()) + t.dim.Render(kindWords[k])
			lines = append(lines, w.marked(m, i == w.kind, line, width))
		}
		return strings.Join(lines, "\n")
	}
	lines := []string{t.text.Render("The liturgy of the rite " + w.d.name + ", performed in order:"), ""}
	if len(w.d.steps) == 0 {
		lines = append(lines, t.dim.Render("  No step is written yet. Press a to add one."))
	}
	aspects := w.d.aspectList()
	avail := max(1, height-len(lines))
	start := max(0, min(w.cursor-avail/2, len(w.d.steps)-avail))
	for i := start; i < min(len(w.d.steps), start+avail); i++ {
		sd := w.d.steps[i]
		step := sd.toStep(aspects)
		line := t.dim.Render(fmt.Sprint(i+1)) + " " + t.accent.Render(kindGlyph(sd.kind)) + " " +
			t.text.Render(stepWords(step, ""))
		lines = append(lines, w.marked(m, i == w.cursor, line, width))
	}
	return strings.Join(lines, "\n")
}

// marked shows a line of a list, the chosen one marked.
func (w *wizard) marked(m *model, chosen bool, line string, width int) string {
	line = lipgloss.NewStyle().MaxWidth(width - 2).Render(line)
	if chosen {
		return m.t.accent.Render("▸ ") + line
	}
	return "  " + line
}
