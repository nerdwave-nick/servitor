package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (w *wizard) view(m *model) string {
	t := m.t
	width, height := m.width, m.bodyH
	title := "Consecration of a new rite"
	if w.d.origName != "" {
		title = "Amendment of the rite " + w.d.origName
	}
	inner := width - 4
	var body string
	switch w.step {
	case stepRite, stepVessel:
		body = w.formHeading(m) + "\n\n" + w.form.view()
	case stepVessels:
		body = w.vesselsView(m, inner)
	case stepReview:
		body = w.reviewView(m, inner, height-5)
	}
	content := " " + w.steps(m) + "\n\n" + indentLines(body, " ")
	return t.panel(title, content, width, height, true)
}

func indentLines(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}

// steps renders the breadcrumb of the wizard.
func (w *wizard) steps(m *model) string {
	t := m.t
	names := []string{"Rite", "Vessels", "Seal"}
	active := map[wizardStep]int{stepRite: 0, stepVessels: 1, stepVessel: 1, stepReview: 2}[w.step]
	parts := make([]string, len(names))
	for i, n := range names {
		label := fmt.Sprintf("%d %s", i+1, n)
		switch {
		case i == active:
			parts[i] = t.selected.Render(" " + label + " ")
		case i < active:
			parts[i] = t.ok.Render("✔ " + n)
		default:
			parts[i] = t.dim.Render(label)
		}
	}
	return strings.Join(parts, t.dim.Render("  ─  "))
}

func (w *wizard) formHeading(m *model) string {
	t := m.t
	if w.step == stepRite {
		return t.text.Render("Speak the name, purpose and aspects of the rite.")
	}
	states, _ := parseStates(w.d.states)
	which := "New vessel"
	if w.vidx >= 0 {
		which = "Vessel " + fmt.Sprint(w.vidx+1)
	}
	page := "settings"
	if w.vpage > 0 {
		page = "aspect " + t.accent.Render(states[w.vpage-1])
	}
	return t.bold.Render(which) + t.dim.Render(fmt.Sprintf("  ·  %s  ·  %d/%d", page, w.vpage+1, len(states)+1))
}

func (w *wizard) vesselsView(m *model, width int) string {
	t := m.t
	lines := []string{t.text.Render("The vessels whose sanctums this rite keeps:"), ""}
	if len(w.d.vessels) == 0 {
		lines = append(lines, t.dim.Render("  No vessels yet. Press a to add one."))
	}
	for i, v := range w.d.vessels {
		s := v.summary(t, "ward", w.d.name, width-2)
		if i == w.vcursor {
			lines = append(lines, t.accent.Render("▸ ")+lipgloss.NewStyle().MaxWidth(width-2).Render(s))
		} else {
			lines = append(lines, "  "+lipgloss.NewStyle().MaxWidth(width-2).Render(s))
		}
	}
	return strings.Join(lines, "\n")
}

func (w *wizard) hints(m *model) [][2]string {
	switch w.step {
	case stepVessels:
		return [][2]string{{"a", "add vessel"}, {"enter", "amend"}, {"d", "cast out"},
			{"J/K", "reorder"}, {"tab", "onward to the seal"}, {"esc", "back"}}
	case stepReview:
		return [][2]string{{"enter", "seal into the Librarium"}, {"j/k", "scroll"}, {"esc", "back"}}
	}
	return [][2]string{{"tab", "next field"}, {"enter", "onward"},
		{"ctrl+s", "seal page"}, {"esc", "back"}}
}
