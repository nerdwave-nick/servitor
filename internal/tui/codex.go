package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/codex"
)

// codexScreen is the codex as the cogitator expounds it: the catalogue of
// its own keys, then the index of every passage; enter recites the chosen
// passage, as `servitor expound <topic>` would.
type codexScreen struct {
	chapters []codex.Chapter
	entries  []codex.Passage // the passages of every chapter, in order
	cursor   int
	offset   int
}

func newCodexScreen() *codexScreen {
	s := &codexScreen{chapters: codex.Index()}
	for _, ch := range s.chapters {
		s.entries = append(s.entries, ch.Passages...)
	}
	return s
}

func (s *codexScreen) fullscreen() bool { return false }

func (s *codexScreen) visible(m *model) int { return max(3, m.height-10) }

func (s *codexScreen) update(m *model, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return s, nil
	}
	switch k.String() {
	case "esc", "q", "?":
		return nil, nil
	case "down", "j":
		s.cursor = min(len(s.entries)-1, s.cursor+1)
	case "up", "k":
		s.cursor = max(0, s.cursor-1)
	case "g", "home":
		s.cursor = 0
	case "G", "end":
		s.cursor = len(s.entries) - 1
	case "enter":
		if s.cursor < len(s.entries) {
			passage := newTextScreen("The Codex", strings.TrimRight(s.entries[s.cursor].Recital(), "\n"))
			passage.back = s
			return passage, nil
		}
	}
	return s, nil
}

// lines renders the whole screen and the line of the chosen passage.
func (s *codexScreen) lines(m *model) ([]string, int) {
	t := m.t
	lines := []string{t.accent.Render("Catalogue of Sacred Keys")}
	lines = append(lines, strings.Split(helpText(m), "\n")...)
	lines = append(lines, "", t.accent.Render("Passages of the Codex")+" "+t.dim.Render("— enter recites the chosen one"))
	width := 0
	for _, p := range s.entries {
		width = max(width, len(p.Topic))
	}
	chosen, i := 0, 0
	for _, ch := range s.chapters {
		lines = append(lines, "", t.label.Render(ch.Title))
		for _, p := range ch.Passages {
			mark, name := "  ", t.key.Render(padRight(p.Topic, width))
			if i == s.cursor {
				mark, name, chosen = t.accent.Render(t.glyphCursor)+" ", t.selected.Render(padRight(p.Topic, width)), len(lines)
			}
			epigraph := truncate(p.Epigraph, max(10, m.width-14-width))
			lines = append(lines, mark+name+"  "+t.dim.Render(epigraph))
			i++
		}
	}
	return lines, chosen
}

func (s *codexScreen) view(m *model) string {
	lines, chosen := s.lines(m)
	h := s.visible(m)
	switch {
	case chosen < s.offset:
		s.offset = chosen
	case chosen >= s.offset+h:
		s.offset = chosen - h + 1
	}
	if s.cursor == 0 && chosen < h {
		s.offset = 0 // the first passage shows the catalogue above it
	}
	s.offset = min(s.offset, max(0, len(lines)-h))
	end := min(len(lines), s.offset+h)
	return m.t.modal("The Codex", strings.Join(lines[s.offset:end], "\n"), m.width-4)
}

func (s *codexScreen) hints(*model) [][2]string {
	return [][2]string{{"j/k", "choose a passage"}, {"enter", "recite"}, {"esc", "withdraw"}}
}
