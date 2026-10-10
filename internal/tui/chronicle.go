package tui

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/nerdwave-nick/servitor/internal/chronicle"
	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// chronicleScreen is the chronicle as the cogitator reads it: every
// invocation of the current and the rotated chronicle, newest first, one
// line each; / filters, enter reads the chosen entry whole.
type chronicleScreen struct {
	path    string            // the chronicle, as the orders place it
	all     []chronicle.Entry // newest first
	shown   []chronicle.Entry // those that answer the filter
	lament  error             // why the chronicle could not be read
	cursor  int
	filter  textinput.Model
	typing  bool        // the filter is being written
	reading *textScreen // the entry read whole, nil while choosing
}

// newChronicleScreen reads the chronicle of m's orders, filtered to the rite
// chosen in the overview ("" for none).
func newChronicleScreen(m *model, rite string) *chronicleScreen {
	f := textinput.New()
	f.Prompt = "/ "
	f.SetValue(rite)
	s := &chronicleScreen{path: m.s.Orders().Chronicle, filter: f}
	s.all, s.lament = chronicle.Read(s.path)
	s.applyFilter()
	return s
}

func (s *chronicleScreen) fullscreen() bool { return true }

// applyFilter keeps the entries whose rite, aspects or verdict contain the
// filter, regardless of case.
func (s *chronicleScreen) applyFilter() {
	q := strings.ToLower(s.filter.Value())
	s.shown = s.shown[:0]
	for _, e := range s.all {
		hay := strings.ToLower(strings.Join([]string{e.Rite, e.Aspect, former(e.Former), string(e.Verdict)}, " "))
		if strings.Contains(hay, q) {
			s.shown = append(s.shown, e)
		}
	}
	s.cursor = min(s.cursor, max(0, len(s.shown)-1))
}

func (s *chronicleScreen) update(m *model, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return s, nil
	}
	if s.reading != nil {
		next, cmd := s.reading.update(m, k)
		if next == nil {
			s.reading = nil
		}
		return s, cmd
	}
	if s.typing {
		return s, s.updateFilter(k)
	}
	switch k.String() {
	case "esc", "q", "h":
		return nil, nil
	case "down", "j":
		s.cursor = min(max(0, len(s.shown)-1), s.cursor+1)
	case "up", "k":
		s.cursor = max(0, s.cursor-1)
	case "g", "home":
		s.cursor = 0
	case "G", "end":
		s.cursor = max(0, len(s.shown)-1)
	case "/":
		s.typing = true
		return s, s.filter.Focus()
	case "enter":
		if s.cursor < len(s.shown) {
			s.reading = newTextScreen("Entry of the chronicle", m.chronicled(s.shown[s.cursor]))
		}
	}
	return s, nil
}

// updateFilter writes the filter; enter seals it, esc casts it away.
func (s *chronicleScreen) updateFilter(k tea.KeyPressMsg) tea.Cmd {
	switch k.String() {
	case "enter":
		s.typing = false
		s.filter.Blur()
		return nil
	case "esc":
		s.typing = false
		s.filter.Blur()
		s.filter.SetValue("")
		s.applyFilter()
		return nil
	}
	var cmd tea.Cmd
	s.filter, cmd = s.filter.Update(k)
	s.applyFilter()
	return cmd
}

func (s *chronicleScreen) view(m *model) string {
	t := m.t
	title := "Chronicle (" + strconv.Itoa(len(s.shown)) + ")"
	if !s.typing && s.filter.Value() != "" {
		title += " / " + s.filter.Value()
	}
	h := m.bodyH
	lines := []string{" " + t.dim.Render("kept in "+truncateLeft(shortPath(s.path), m.width-14))}
	if s.typing {
		lines = append(lines, " "+s.filter.View())
	}
	lines = append(lines, "")
	lines = append(lines, s.listLines(m, m.width-2, h-2-len(lines))...)
	page := t.panel(title, strings.Join(lines, "\n"), m.width, h, true)
	if s.reading == nil {
		return page
	}
	box := s.reading.view(m)
	x := max(0, (m.width-lipgloss.Width(box))/2)
	return lipgloss.NewCompositor(lipgloss.NewLayer(page), lipgloss.NewLayer(box).X(x).Y(1).Z(1)).Render()
}

// listLines renders the entries in sight, h lines at most, w cells wide.
func (s *chronicleScreen) listLines(m *model, w, h int) []string {
	t := m.t
	switch {
	case s.lament != nil:
		return wrap([]string{t.danger.Render("The chronicle cannot be read: " + s.lament.Error())}, w-2)
	case len(s.all) == 0:
		return []string{" " + t.dim.Render("The chronicle is silent: no invocation has yet been recorded.")}
	case len(s.shown) == 0:
		return []string{" " + t.dim.Render("No entry of the chronicle answers the filter.")}
	}
	riteW, turnW := 0, 0
	for _, e := range s.shown {
		riteW = max(riteW, lipgloss.Width(e.Rite))
		turnW = max(turnW, lipgloss.Width(turn(e)))
	}
	h = max(1, h)
	start := max(0, min(s.cursor-h/2, len(s.shown)-h))
	var lines []string
	for i := start; i < min(len(s.shown), start+h); i++ {
		e := s.shown[i]
		line := t.dim.Render(e.At.Local().Format("2006-01-02 15:04")) + "  " +
			t.text.Render(padRight(e.Rite, riteW)) + "  " + t.accent.Render(padRight(turn(e), turnW)) + "  " +
			m.verdictMark(e.Verdict)
		if i == s.cursor {
			lines = append(lines, t.accent.Render(t.glyphCursor)+t.selected.Render(fit(line, w-1)))
		} else {
			lines = append(lines, " "+line)
		}
	}
	return lines
}

func (s *chronicleScreen) hints(m *model) [][2]string {
	switch {
	case s.reading != nil:
		return s.reading.hints(m)
	case s.typing:
		return [][2]string{{"enter", "seal the filter"}, {"esc", "cast the filter away"}}
	}
	return [][2]string{{"j/k", "choose an entry"}, {"enter", "read it whole"}, {"/", "filter"}, {"esc", "withdraw"}}
}

// turn names the aspects an invocation turned the rite from and to.
func turn(e chronicle.Entry) string { return former(e.Former) + " → " + e.Aspect }

// verdictMark renders a verdict with its glyph: ✔ for triumph, ✖ otherwise.
func (m *model) verdictMark(v invocation.Verdict) string {
	switch v {
	case invocation.Triumph:
		return m.t.ok.Render("✔ " + string(v))
	case invocation.Reverted:
		return m.t.warn.Render("✖ " + string(v))
	}
	return m.t.danger.Render("✖ " + string(v))
}
