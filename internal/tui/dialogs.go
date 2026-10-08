package tui

import (
	"errors"
	"os"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/engine"
)

// textScreen is a scrollable read-only modal.
type textScreen struct {
	title  string
	lines  []string
	offset int
	back   screen // screen to return to on close, nil for the overview
}

func newTextScreen(title, body string) *textScreen {
	return &textScreen{title: title, lines: strings.Split(body, "\n")}
}

func (s *textScreen) fullscreen() bool { return false }

func (s *textScreen) visible(m *model) int { return max(3, m.height-10) }

func (s *textScreen) update(m *model, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return s, nil
	}
	maxOff := max(0, len(s.lines)-s.visible(m))
	switch k.String() {
	case "esc", "q", "enter", "?":
		return s.back, nil
	case "down", "j":
		s.offset = min(maxOff, s.offset+1)
	case "up", "k":
		s.offset = max(0, s.offset-1)
	case "pgdown", "ctrl+d", "space":
		s.offset = min(maxOff, s.offset+s.visible(m)/2)
	case "pgup", "ctrl+u":
		s.offset = max(0, s.offset-s.visible(m)/2)
	}
	return s, nil
}

func (s *textScreen) view(m *model) string {
	end := min(len(s.lines), s.offset+s.visible(m))
	body := strings.Join(s.lines[s.offset:end], "\n")
	if len(s.lines) > s.visible(m) {
		body += "\n" + m.t.dim.Render(strings.Repeat("─", 8)+" "+strconv.Itoa(s.offset+1)+"–"+strconv.Itoa(end)+"/"+strconv.Itoa(len(s.lines)))
	}
	return m.t.modal(s.title, body, m.width-4)
}

func (s *textScreen) hints(m *model) [][2]string {
	return [][2]string{{"j/k", m.lex.P("scroll the scroll", "scroll")}, {"esc", m.lex.P("withdraw", "close")}}
}

// deleteScreen confirms deleting a definition, optionally purging blocks.
type deleteScreen struct{ r row }

func newDeleteScreen(r *row) *deleteScreen { return &deleteScreen{r: *r} }

func (s *deleteScreen) fullscreen() bool { return false }

func (s *deleteScreen) update(m *model, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return s, nil
	}
	l := m.lex
	switch k.String() {
	case "esc", "n", "q":
		return nil, m.notify(toastInfo, l.P("Mercy is shown. The rite endures.", "Cancelled."))
	case "y":
		return nil, m.remove(s.r, false)
	case "p":
		if s.r.sw != nil {
			return nil, m.remove(s.r, true)
		}
	}
	return s, nil
}

func (m *model) remove(r row, purge bool) tea.Cmd {
	l := m.lex
	if purge {
		if _, err := engine.Purge(r.sw, engine.Options{}); err != nil {
			return m.notify(toastErr, l.P("The purge falters: ", "Purge failed: ")+err.Error())
		}
	}
	for _, p := range r.paths {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			m.reload("")
			return m.notify(toastErr, err.Error())
		}
	}
	m.reload("")
	msg := l.P("The rite "+r.name+" is excommunicated. Its sanctums remain.", r.name+" deleted; managed blocks were kept.")
	if purge {
		msg = l.P("The rite "+r.name+" is excommunicated and its sanctums purged. Exterminatus complete.", r.name+" deleted and its blocks removed.")
	}
	return m.notify(toastOK, msg)
}

func (s *deleteScreen) view(m *model) string {
	t, l := m.t, m.lex
	lines := []string{
		t.bold.Render(l.P("Excommunicate the rite ", "Delete ") + s.r.name + "?"),
		"",
		t.key.Render("y") + "  " + t.text.Render(l.P("strike it from the Librarium (sanctums remain in their vessels)", "delete the definition, keep the managed blocks")),
	}
	if s.r.sw != nil {
		lines = append(lines, t.key.Render("p")+"  "+t.text.Render(l.P("purge its sanctums from every vessel, then strike it", "remove its managed blocks from all files, then delete it")))
	}
	lines = append(lines, t.key.Render("n")+"  "+t.text.Render(l.P("show mercy", "cancel")))
	for _, p := range s.r.paths {
		lines = append(lines, "", t.dim.Render(shortPath(p)))
	}
	return m.t.modal(l.P("Excommunication", "Delete"), strings.Join(lines, "\n"), m.width-4)
}

func (s *deleteScreen) hints(m *model) [][2]string {
	return [][2]string{{"y", m.lex.P("excommunicate", "delete")}, {"p", m.lex.P("purge", "purge")}, {"n", m.lex.P("mercy", "cancel")}}
}
