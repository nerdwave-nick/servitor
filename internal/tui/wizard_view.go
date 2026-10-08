package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (w *wizard) view(m *model) string {
	t, l := m.t, m.lex
	width, height := m.width, m.bodyH
	title := l.P("Consecration of a new rite", "New switch")
	if w.d.origName != "" {
		title = l.P("Amendment of the rite ", "Edit ") + w.d.origName
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
	t, l := m.t, m.lex
	names := []string{l.P("Rite", "Switch"), l.P("Vessels", "Files"), l.P("Seal", "Review")}
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
	t, l := m.t, m.lex
	if w.step == stepRite {
		return t.text.Render(l.P("Speak the name, purpose and aspects of the rite.", "Name, description and states of the switch."))
	}
	states, _ := parseStates(w.d.states)
	which := l.P("New vessel", "New file")
	if w.vidx >= 0 {
		which = l.P("Vessel ", "File ") + fmt.Sprint(w.vidx+1)
	}
	page := l.P("settings", "settings")
	if w.vpage > 0 {
		page = l.P("aspect ", "state ") + t.accent.Render(states[w.vpage-1])
	}
	return t.bold.Render(which) + t.dim.Render(fmt.Sprintf("  ·  %s  ·  %d/%d", page, w.vpage+1, len(states)+1))
}

func (w *wizard) vesselsView(m *model, width int) string {
	t, l := m.t, m.lex
	lines := []string{t.text.Render(l.P("The vessels whose sanctums this rite keeps:", "Files managed by this switch:")), ""}
	if len(w.d.vessels) == 0 {
		lines = append(lines, t.dim.Render(l.P("  No vessels yet. Press a to add one.", "  No files yet. Press a to add one.")))
	}
	for i, v := range w.d.vessels {
		s := v.summary(t, l.Guard, w.d.name, width-2)
		if i == w.vcursor {
			lines = append(lines, t.accent.Render("▸ ")+lipgloss.NewStyle().MaxWidth(width-2).Render(s))
		} else {
			lines = append(lines, "  "+lipgloss.NewStyle().MaxWidth(width-2).Render(s))
		}
	}
	return strings.Join(lines, "\n")
}

func (w *wizard) hints(m *model) [][2]string {
	l := m.lex
	switch w.step {
	case stepVessels:
		return [][2]string{{"a", l.P("add vessel", "add")}, {"enter", l.P("amend", "edit")}, {"d", l.P("cast out", "remove")},
			{"J/K", l.P("reorder", "reorder")}, {"tab", l.P("onward to the seal", "review")}, {"esc", l.P("back", "back")}}
	case stepReview:
		return [][2]string{{"enter", l.P("seal into the Librarium", "save")}, {"j/k", l.P("scroll", "scroll")}, {"esc", l.P("back", "back")}}
	}
	return [][2]string{{"tab", l.P("next field", "next field")}, {"enter", l.P("onward", "next")},
		{"ctrl+s", l.P("seal page", "save page")}, {"esc", l.P("back", "back")}}
}
