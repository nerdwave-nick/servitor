package tui

import (
	"image/color"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

// palette holds the colors of one vocabulary.
type palette struct {
	accent, text, dim, border, ok, warn, danger, barBg, barFg, selBg color.Color
}

var (
	// Crimson, brass and bone: the colors of the forge.
	grimPalette = palette{
		accent: lipgloss.Color("#D4A62A"), text: lipgloss.Color("#E8DCC4"), dim: lipgloss.Color("#8C7F68"),
		border: lipgloss.Color("#5E4630"), ok: lipgloss.Color("#9BC53D"), warn: lipgloss.Color("#E8913A"),
		danger: lipgloss.Color("#E5484D"), barBg: lipgloss.Color("#5A0F0E"), barFg: lipgloss.Color("#F2E6C9"),
		selBg: lipgloss.Color("#3A1A12"),
	}
	plainPalette = palette{
		accent: lipgloss.Color("#7AA2F7"), text: lipgloss.Color("#C0CAF5"), dim: lipgloss.Color("#737AA2"),
		border: lipgloss.Color("#3B4261"), ok: lipgloss.Color("#9ECE6A"), warn: lipgloss.Color("#E0AF68"),
		danger: lipgloss.Color("#F7768E"), barBg: lipgloss.Color("#24283B"), barFg: lipgloss.Color("#C0CAF5"),
		selBg: lipgloss.Color("#2E3550"),
	}
)

// theme holds the derived styles and glyphs.
type theme struct {
	p palette

	text, dim, accent, ok, warn, danger, bold lipgloss.Style
	bar, barAccent, key, selected, label      lipgloss.Style

	glyphApplied, glyphDormant, glyphCorrupt, glyphBroken, glyphCursor string
	glyphCurrent, glyphOther, glyphLogo                                string
}

func newTheme(grim bool) *theme {
	p := plainPalette
	if grim {
		p = grimPalette
	}
	s := lipgloss.NewStyle
	t := &theme{
		p:         p,
		text:      s().Foreground(p.text),
		dim:       s().Foreground(p.dim),
		accent:    s().Foreground(p.accent),
		ok:        s().Foreground(p.ok),
		warn:      s().Foreground(p.warn),
		danger:    s().Foreground(p.danger),
		bold:      s().Foreground(p.text).Bold(true),
		bar:       s().Background(p.barBg).Foreground(p.barFg),
		barAccent: s().Background(p.barBg).Foreground(p.accent).Bold(true),
		key:       s().Foreground(p.accent).Bold(true),
		selected:  s().Background(p.selBg).Foreground(p.text).Bold(true),
		label:     s().Foreground(p.accent).Bold(true),

		glyphApplied: "●", glyphDormant: "○", glyphCorrupt: "◐", glyphBroken: "✖",
		glyphCursor: "▌", glyphCurrent: "◆", glyphOther: "◇", glyphLogo: "◈",
	}
	if grim {
		t.glyphBroken, t.glyphLogo = "✠", "⚙"
	}
	return t
}

func (t *theme) inputStyles() textinput.Styles {
	st := textinput.DefaultDarkStyles()
	st.Focused.Text, st.Blurred.Text = t.bold, t.text
	st.Focused.Placeholder, st.Blurred.Placeholder = t.dim.Faint(true), t.dim.Faint(true)
	st.Cursor.Color = t.p.accent
	return st
}

func (t *theme) areaStyles() textarea.Styles {
	st := textarea.DefaultDarkStyles()
	for _, s := range []*textarea.StyleState{&st.Focused, &st.Blurred} {
		s.Text, s.Placeholder = t.text, t.dim.Faint(true)
		s.CursorLine = t.text
		s.EndOfBuffer = t.dim
	}
	st.Focused.Prompt, st.Blurred.Prompt = t.accent, t.dim
	st.Cursor.Color = t.p.accent
	return st
}
