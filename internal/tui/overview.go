package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/lexicon"
	"github.com/nerdwave-nick/servitor/internal/librarium"
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
		m.screen = newCodexScreen()
	case "i":
		m.screen = newTextScreen("Verdict of the Inquisition", m.verdict())
	case "O":
		return m.showWords()
	case "h":
		m.screen = newChronicleScreen(m, nameOf(r))
	case "n":
		return m.openWizard(newRiteDraft())
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
	switch key {
	case "d":
		m.screen = newDeleteScreen(r)
		return nil
	case "o":
		return m.openEditor(r)
	case "e", "c":
		return m.amend(key, r)
	}
	if r.rite() == nil {
		return m.notify(toastErr, "This rite is heretical. Purify it with o ($EDITOR) or excommunicate it with d.")
	}
	if key == "enter" {
		m.screen = newInvokeScreen(r)
		return nil
	}
	return m.cycle(r)
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

// cycle invokes the aspect after the current one, inscribed by its decrees.
func (m *model) cycle(r *row) tea.Cmd {
	aspects := r.rite().Aspects
	next := aspects[0]
	for i, a := range aspects {
		if a == r.reading.Aspect {
			next = aspects[(i+1)%len(aspects)]
		}
	}
	return m.invoke(r.rite(), next, nil)
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

// verdict renders the Inquisition's findings upon the whole Librarium.
func (m *model) verdict() string {
	t := m.t
	found, _ := m.s.Inquire(nil, false)
	if len(found) == 0 {
		return t.ok.Render("No heresy was found. The Emperor protects.")
	}
	var b strings.Builder
	for _, f := range found {
		sev, st := lexicon.Impurity, t.warn
		if f.Severity == librarium.Heresy {
			sev, st = lexicon.Heresy, t.danger
		}
		loc := fmt.Sprintf("%s:%d:%d", shortPath(f.Scripture), f.Line, f.Column)
		b.WriteString(st.Render(sev) + " " + t.dim.Render(loc) + "\n  " + t.text.Render(f.Message) + "\n")
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
		glyph, state := m.standingGlyph(r.reading.Standing)
		if r.reading.Aspect != "" {
			state = t.accent.Render(r.reading.Aspect)
		}
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

// standingGlyph returns the colored glyph and name of a standing.
func (m *model) standingGlyph(s augury.Standing) (string, string) {
	t := m.t
	switch s {
	case augury.Performed:
		return t.ok.Render(t.glyphApplied), t.ok.Render(string(s))
	case augury.Dormant:
		return t.dim.Render(t.glyphDormant), t.dim.Render(string(s))
	case augury.Heretical:
		return t.danger.Render(t.glyphBroken), t.danger.Render(string(s))
	}
	return t.warn.Render(t.glyphCorrupt), t.warn.Render(string(s))
}
