package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// panel draws a rounded box of the given outer size with title embedded in
// the top border, lazygit style. Content lines are clipped to fit.
func (t *theme) panel(title, content string, width, height int, focused bool) string {
	if width < 4 || height < 2 {
		return ""
	}
	bc := t.p.border
	if focused {
		bc = t.p.accent
	}
	b := lipgloss.NewStyle().Foreground(bc)
	inner := width - 2

	var top string
	if title != "" {
		ts := t.dim
		if focused {
			ts = t.accent.Bold(true)
		}
		title = " " + truncate(title, inner-3) + " "
		top = b.Render("╭─") + ts.Render(title) + b.Render(strings.Repeat("─", max(0, inner-1-lipgloss.Width(title)))+"╮")
	} else {
		top = b.Render("╭" + strings.Repeat("─", inner) + "╮")
	}
	lines := strings.Split(content, "\n")
	rows := make([]string, 0, height)
	rows = append(rows, top)
	for i := 0; i < height-2; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		rows = append(rows, b.Render("│")+fit(line, inner)+b.Render("│"))
	}
	rows = append(rows, b.Render("╰"+strings.Repeat("─", inner)+"╯"))
	return strings.Join(rows, "\n")
}

// fit pads or truncates a (possibly styled) line to exactly width cells.
func fit(line string, width int) string {
	w := lipgloss.Width(line)
	if w > width {
		return lipgloss.NewStyle().MaxWidth(width).Render(line)
	}
	return line + strings.Repeat(" ", width-w)
}

// truncate shortens plain text to width cells, adding an ellipsis.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r))+1 > width {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

// modal renders content in a framed box sized to the content (bounded by
// maxW) for display on top of the screen.
func (t *theme) modal(title, content string, maxW int) string {
	lines := strings.Split(content, "\n")
	w := lipgloss.Width(title) + 6
	for _, l := range lines {
		w = max(w, lipgloss.Width(l)+4)
	}
	w = min(w, maxW)
	padded := make([]string, len(lines))
	for i, l := range lines {
		padded[i] = " " + l
	}
	return t.panel(title, strings.Join(padded, "\n"), w, len(lines)+2, true)
}

// truncateLeft shortens plain text to width cells, keeping the end.
func truncateLeft(s string, width int) string {
	if width <= 1 || lipgloss.Width(s) <= width {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r))+1 > width {
		r = r[1:]
	}
	return "…" + string(r)
}
