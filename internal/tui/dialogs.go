package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
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

// wrapped are the lines wrapped to the width of the viewscreen.
func (s *textScreen) wrapped(m *model) []string { return wrap(s.lines, m.width-8) }

// wrap wraps every (possibly styled) line to width cells.
func wrap(lines []string, width int) []string {
	var out []string
	for _, l := range lines {
		out = append(out, strings.Split(ansi.Wrap(l, max(20, width), ""), "\n")...)
	}
	return out
}

func (s *textScreen) update(m *model, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return s, nil
	}
	maxOff := max(0, len(s.wrapped(m))-s.visible(m))
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
	lines := s.wrapped(m)
	s.offset = min(s.offset, max(0, len(lines)-s.visible(m)))
	end := min(len(lines), s.offset+s.visible(m))
	body := strings.Join(lines[s.offset:end], "\n")
	if len(lines) > s.visible(m) {
		body += "\n" + m.t.dim.Render(strings.Repeat("─", 8)+" "+strconv.Itoa(s.offset+1)+"–"+strconv.Itoa(end)+"/"+strconv.Itoa(len(lines)))
	}
	return m.t.modal(s.title, body, m.width-4)
}

func (s *textScreen) hints(m *model) [][2]string {
	return [][2]string{{"j/k", "scroll the scroll"}, {"esc", "withdraw"}}
}

// deleteScreen confirms the excommunication of a rite, optionally purging
// its sanctums.
type deleteScreen struct{ r row }

func newDeleteScreen(r *row) *deleteScreen { return &deleteScreen{r: *r} }

func (s *deleteScreen) fullscreen() bool { return false }

func (s *deleteScreen) update(m *model, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return s, nil
	}
	switch k.String() {
	case "esc", "n", "q":
		return nil, m.notify(toastInfo, "Mercy is shown. The rite endures.")
	case "y":
		return nil, m.remove(s.r, false)
	case "p":
		if s.r.rite() != nil {
			return nil, m.remove(s.r, true)
		}
	}
	return s, nil
}

func (m *model) remove(r row, purge bool) tea.Cmd {
	ex, err := m.s.Excommunicate(r.name, purge)
	m.reload("")
	if err != nil {
		return m.notify(toastErr, "The excommunication falters: "+err.Error())
	}
	msg := "The rite " + r.name + " is excommunicated. Its sanctums remain."
	if purge {
		msg = fmt.Sprintf("The rite %s is excommunicated and its sanctums purged from %d vessel(s). "+
			"Exterminatus complete.", r.name, len(ex.Purged))
	}
	return m.notify(toastOK, msg)
}

func (s *deleteScreen) view(m *model) string {
	t := m.t
	lines := []string{
		t.bold.Render("Excommunicate the rite " + s.r.name + "?"),
		"",
		t.key.Render("y") + "  " + t.text.Render("strike it from the Librarium (sanctums remain in their vessels)"),
	}
	if s.r.rite() != nil {
		lines = append(lines, t.key.Render("p")+"  "+t.text.Render("purge its sanctums from every vessel, then strike it"),
			"   "+t.dim.Render("(transcribed vessels and tethers stand untouched)"))
	}
	lines = append(lines, t.key.Render("n")+"  "+t.text.Render("show mercy"))
	for _, p := range s.r.paths {
		lines = append(lines, "", t.dim.Render(shortPath(p)))
	}
	return m.t.modal("Excommunication", strings.Join(lines, "\n"), m.width-4)
}

func (s *deleteScreen) hints(m *model) [][2]string {
	if s.r.rite() == nil {
		return [][2]string{{"y", "excommunicate"}, {"n", "mercy"}}
	}
	return [][2]string{{"y", "excommunicate"}, {"p", "purge"}, {"n", "mercy"}}
}
