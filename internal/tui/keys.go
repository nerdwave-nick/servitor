package tui

import (
	"os"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/lexicon"
)

var userHome = os.UserHomeDir

// binding documents one key of the overview.
type binding struct {
	keys        string
	grim, plain string
	footer      bool // shown in the footer hint line
}

var overviewBindings = []binding{
	{"↑/k ↓/j", "choose a rite", "move", false},
	{"g G", "first / last rite", "first / last", false},
	{"enter", "invoke", "apply", true},
	{"space", "cycle aspect", "next state", true},
	{"n", "consecrate", "new", true},
	{"e", "amend", "edit", true},
	{"c", "replicate", "clone", false},
	{"d", "excommunicate", "delete", true},
	{"o", "open in $EDITOR", "open in $EDITOR", false},
	{"i", "inquisition", "verify", false},
	{"/", "filter", "filter", false},
	{"r", "reload the Librarium", "reload", false},
	{"t", "toggle the liturgy (grimdark)", "toggle vocabulary (grimdark)", false},
	{"?", "lore", "help", true},
	{"q", "retreat", "quit", true},
}

func (b binding) text(l *lexicon.Lexicon) string { return l.P(b.grim, b.plain) }

func (m *model) keyHints() string {
	if m.screen != nil {
		return m.hint(screenHints(m))
	}
	if m.typing {
		return m.hint([][2]string{{"enter", m.lex.P("seal the filter", "apply filter")}, {"esc", m.lex.P("abandon", "clear")}})
	}
	var pairs [][2]string
	for _, b := range overviewBindings {
		if b.footer {
			pairs = append(pairs, [2]string{strings.Fields(b.keys)[0], b.text(m.lex)})
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
	return [][2]string{{"esc", m.lex.P("withdraw", "close")}}
}

// helpText renders the full key catalogue for the help overlay.
func helpText(m *model) string {
	var b strings.Builder
	for _, bd := range overviewBindings {
		b.WriteString(m.t.key.Render(padRight(bd.keys, 9)) + " " + m.t.text.Render(bd.text(m.lex)) + "\n")
	}
	b.WriteString("\n" + m.t.dim.Render(m.lex.P(
		"In forms: tab/↑↓ move between fields · enter next · ctrl+s seal · esc go back",
		"In forms: tab/↑↓ move between fields · enter next · ctrl+s save · esc back")))
	return b.String()
}

func padRight(s string, n int) string {
	if len([]rune(s)) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len([]rune(s)))
}
