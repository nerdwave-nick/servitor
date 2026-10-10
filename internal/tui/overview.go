package tui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/nerdwave-nick/servitor/internal/config"
	"github.com/nerdwave-nick/servitor/internal/engine"
	"github.com/nerdwave-nick/servitor/internal/lexicon"
)

func (m *model) updateOverview(msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	if m.typing {
		return m.updateFilter(k)
	}
	r := m.current()
	switch k.String() {
	case "q":
		return tea.Quit
	case "up", "k":
		m.cursor = max(0, m.cursor-1)
	case "down", "j":
		m.cursor = min(len(m.rows)-1, m.cursor+1)
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = max(0, len(m.rows)-1)
	case "/":
		m.typing = true
		return m.filter.Focus()
	case "esc":
		if m.filter.Value() != "" {
			m.filter.SetValue("")
			m.applyFilter(nameOf(r))
		}
	case "r":
		m.reload("")
		return m.notify(toastInfo, "The Librarium has been re-read.")
	case "?":
		m.screen = newTextScreen("Catalogue of Sacred Keys", helpText(m))
	case "i":
		m.screen = newTextScreen("Verdict of the Inquisition", m.verdict())
	case "n":
		return m.openWizard(draft{states: "on, off"})
	case "enter", "space", "e", "c", "d", "o":
		return m.rowAction(k.String(), r)
	}
	return nil
}

// rowAction runs an action that needs a selected rite.
func (m *model) rowAction(key string, r *row) tea.Cmd {
	if r == nil {
		return m.notify(toastInfo, "No rite is chosen. Consecrate one with n.")
	}
	if r.sw == nil && key != "d" && key != "o" {
		return m.notify(toastErr, "This rite is heretical. Purify it with o ($EDITOR) or excommunicate it with d.")
	}
	switch key {
	case "enter":
		m.screen = newInvokeScreen(m, r.sw)
	case "space":
		return m.cycle(r)
	case "e":
		return m.openWizard(draftFrom(r.sw))
	case "c":
		d := draftFrom(r.sw)
		d.origName, d.origPath, d.name = "", "", r.name+"-copy"
		return m.openWizard(d)
	case "d":
		m.screen = newDeleteScreen(r)
	case "o":
		return m.openEditor(r)
	}
	return nil
}

func (m *model) updateFilter(k tea.KeyPressMsg) tea.Cmd {
	switch k.String() {
	case "enter":
		m.typing = false
		m.filter.Blur()
		return nil
	case "esc":
		m.typing = false
		m.filter.Blur()
		m.filter.SetValue("")
		m.applyFilter(nameOf(m.current()))
		return nil
	}
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(k)
	m.applyFilter(nameOf(m.current()))
	return cmd
}

func nameOf(r *row) string {
	if r == nil {
		return ""
	}
	return r.name
}

// cycle applies the aspect after the current one with configured metadata.
func (m *model) cycle(r *row) tea.Cmd {
	next := r.sw.States[0]
	if cur := r.status.State(); cur != "" {
		for i, s := range r.sw.States {
			if s == cur {
				next = r.sw.States[(i+1)%len(r.sw.States)]
			}
		}
	}
	return m.perform(r.sw, next, nil)
}

// perform applies a state and reloads with a toast describing the outcome.
func (m *model) perform(sw *config.Switch, state string, overrides map[string]string) tea.Cmd {
	res, err := engine.Apply(sw, state, overrides, engine.Options{})
	if err != nil {
		return m.notify(toastErr, "The rite falters: "+err.Error())
	}
	prev := res.Previous
	if prev == "" {
		prev = "dormant"
	}
	changed := 0
	for _, c := range res.Changes {
		if c.Changed() {
			changed++
		}
	}
	m.reload(sw.Name)
	return m.notify(toastOK, fmt.Sprintf("Rite %s performed: %s → %s · %d vessel(s) sanctified. The Omnissiah is pleased.", sw.Name, prev, state, changed))
}

func (m *model) openEditor(r *row) tea.Cmd {
	if len(r.paths) == 0 {
		return nil
	}
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	parts := strings.Fields(editor)
	cmd := exec.Command(parts[0], append(parts[1:], r.paths[0])...)
	name := r.name
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		t := &toast{kind: toastInfo, text: "The scripture has been amended by hand."}
		if err != nil {
			t = &toast{kind: toastErr, text: err.Error()}
		}
		return reloadMsg{selected: name, toast: t}
	})
}

// verdict renders all diagnostics of the configuration and target files.
func (m *model) verdict() string {
	t := m.t
	diags := append(config.Diagnostics{}, m.set.Diags...)
	for _, name := range m.set.Names() {
		diags = append(diags, engine.CheckTargets(m.set.Switches[name])...)
	}
	diags.Sort()
	if len(diags) == 0 {
		return t.ok.Render("No heresy was found. The Emperor protects.")
	}
	var b strings.Builder
	for _, d := range diags {
		sev, st := lexicon.Impurity, t.warn
		if d.Severity == config.SevError {
			sev, st = lexicon.Heresy, t.danger
		}
		loc := shortPath(d.File)
		if d.Line > 0 {
			loc += fmt.Sprintf(":%d:%d", d.Line, d.Col)
		}
		b.WriteString(st.Render(sev) + " " + t.dim.Render(loc) + "\n  " + t.text.Render(d.Message) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *model) overview(h int) string {
	listW := min(42, max(28, m.width/3))
	detailW := m.width - listW
	title := fmt.Sprintf("%s (%d)", "Rites", len(m.rows))
	if m.typing || m.filter.Value() != "" {
		title += " " + m.filter.Value()
	}
	list := m.t.panel(title, m.listView(listW-2, h-2), listW, h, true)
	dTitle := "Scripture"
	if r := m.current(); r != nil {
		dTitle = r.name
	}
	detail := m.t.panel(dTitle, indentLines(m.detailView(detailW-4), " "), detailW, h, false)
	return lipgloss.JoinHorizontal(lipgloss.Top, list, detail)
}

func (m *model) listView(w, h int) string {
	t := m.t
	var lines []string
	if m.typing {
		lines = append(lines, " "+m.filter.View())
		h--
	}
	if len(m.rows) == 0 {
		lines = append(lines, "", t.dim.Render(" "+"The Librarium is empty."),
			t.dim.Render(" "+"Press n to consecrate a rite."))
		return strings.Join(lines, "\n")
	}
	start := max(0, min(m.cursor-h/2, len(m.rows)-h))
	for i := start; i < min(len(m.rows), start+h); i++ {
		r := m.rows[i]
		glyph, state := m.statusGlyph(r)
		name := truncate(r.name, w-lipgloss.Width(state)-5)
		gap := max(1, w-3-lipgloss.Width(name)-lipgloss.Width(state))
		line := " " + glyph + " " + name + strings.Repeat(" ", gap) + state
		if i == m.cursor {
			line = t.accent.Render(t.glyphCursor) + t.selected.Render(fit(strings.TrimPrefix(line, " "), w-1))
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// statusGlyph returns the colored status glyph and state label of a row.
func (m *model) statusGlyph(r row) (string, string) {
	t := m.t
	switch {
	case r.sw == nil:
		return t.danger.Render(t.glyphBroken), t.danger.Render(lexicon.Heretical)
	case r.status.Err == nil:
		return t.ok.Render(t.glyphApplied), t.accent.Render(r.status.State())
	case errors.Is(r.status.Err, engine.ErrNotApplied):
		return t.dim.Render(t.glyphDormant), t.dim.Render(lexicon.Dormant)
	default:
		return t.warn.Render(t.glyphCorrupt), t.warn.Render(lexicon.Corrupted)
	}
}
