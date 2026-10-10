package tui

import "strings"

func (f *form) view() string {
	t := f.t
	var blocks []string
	for i, fl := range f.fields {
		focused := i == f.focus
		marker, label := "  ", t.dim.Render(fl.label)
		if focused {
			marker, label = t.accent.Render("▸ "), t.label.Render(fl.label)
		}
		var b strings.Builder
		b.WriteString(marker + label + "\n")
		switch fl.kind {
		case fieldText:
			b.WriteString("  " + fl.input.View())
		case fieldArea:
			for _, l := range strings.Split(fl.area.View(), "\n") {
				b.WriteString("  " + l + "\n")
			}
		case fieldToggle:
			box := t.dim.Render("[ ] no")
			if fl.on {
				box = t.ok.Render("[✔] yes")
			}
			b.WriteString("  " + box)
		case fieldChoice:
			b.WriteString("  " + f.choiceView(fl))
		}
		switch {
		case fl.err != "":
			b.WriteString("\n  " + t.danger.Render("✖ "+fl.err))
		case fl.hint != "" && focused:
			b.WriteString("\n  " + t.dim.Render(fl.hint))
		}
		blocks = append(blocks, strings.TrimRight(b.String(), "\n"))
	}
	return f.window(blocks)
}

// window shows as many field blocks as fit, keeping the focused one visible.
func (f *form) window(blocks []string) string {
	if f.height <= 0 {
		return strings.Join(blocks, "\n\n")
	}
	heights := make([]int, len(blocks))
	for i, b := range blocks {
		heights[i] = strings.Count(b, "\n") + 2
	}
	start, used := f.focus, heights[f.focus]
	for start > 0 && used+heights[start-1] <= f.height {
		start--
		used += heights[start]
	}
	end := f.focus + 1
	for end < len(blocks) && used+heights[end] <= f.height {
		used += heights[end]
		end++
	}
	out := strings.Join(blocks[start:end], "\n\n")
	if start > 0 {
		out = f.t.dim.Render("  ↑ more") + "\n" + out
	}
	if end < len(blocks) {
		out += "\n" + f.t.dim.Render("  ↓ more")
	}
	return out
}

// choiceView shows every choice, the chosen one marked.
func (f *form) choiceView(fl *field) string {
	parts := make([]string, len(fl.choices))
	for i, c := range fl.choices {
		parts[i] = f.t.dim.Render("◇ " + c)
		if i == fl.pick {
			parts[i] = f.t.accent.Render("◆ " + c)
		}
	}
	return strings.Join(parts, "  ")
}
