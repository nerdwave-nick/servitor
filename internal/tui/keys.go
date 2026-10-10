package tui

import (
	"os"
	"strings"
)

var userHome = os.UserHomeDir

// binding documents one key of the overview.
type binding struct {
	keys   string
	text   string
	footer bool // shown in the footer hint line
}

var overviewBindings = []binding{
	{"↑/k ↓/j", "choose a rite", false},
	{"g G", "first / last rite", false},
	{"enter", "invoke", true},
	{"space", "cycle aspect", true},
	{"n", "consecrate", true},
	{"e", "amend", true},
	{"c", "replicate", false},
	{"d", "excommunicate", true},
	{"o", "open in $EDITOR", false},
	{"i", "inquisition", false},
	{"O", "words of the last invocation", false},
	{"/", "filter", false},
	{"r", "reload the Librarium", false},
	{"?", "lore", true},
	{"q", "retreat", true},
}

func (m *model) keyHints() string {
	if m.screen != nil {
		return m.hint(screenHints(m))
	}
	if m.typing {
		return m.hint([][2]string{{"enter", "seal the filter"}, {"esc", "abandon"}})
	}
	var pairs [][2]string
	for _, b := range overviewBindings {
		if b.footer {
			pairs = append(pairs, [2]string{strings.Fields(b.keys)[0], b.text})
		}
	}
	return m.hint(pairs)
}

func (m *model) hint(pairs [][2]string) string {
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = m.t.key.Render(p[0]) + " " + m.t.dim.Render(p[1])
	}
	return " " + strings.Join(parts, m.t.dim.Render(" · "))
}

// screenHints returns the key hints of the active screen.
func screenHints(m *model) [][2]string {
	if h, ok := m.screen.(interface{ hints(*model) [][2]string }); ok {
		return h.hints(m)
	}
	return [][2]string{{"esc", "withdraw"}}
}

// helpText renders the full key catalogue for the help overlay.
func helpText(m *model) string {
	var b strings.Builder
	for _, bd := range overviewBindings {
		b.WriteString(m.t.key.Render(padRight(bd.keys, 9)) + " " + m.t.text.Render(bd.text) + "\n")
	}
	b.WriteString("\n" + m.t.dim.Render("In forms: tab/↑↓ move between fields · enter next · ctrl+s seal · esc go back"))
	return b.String()
}

func padRight(s string, n int) string {
	if len([]rune(s)) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len([]rune(s)))
}
