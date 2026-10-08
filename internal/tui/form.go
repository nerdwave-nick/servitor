package tui

import (
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type fieldKind int

const (
	fieldText fieldKind = iota
	fieldArea
	fieldToggle
)

// field is one input of a form.
type field struct {
	key, label, hint string
	kind             fieldKind
	input            textinput.Model
	area             textarea.Model
	on               bool
	validate         func(string) error
	err              string
}

func (f *field) value() string {
	switch f.kind {
	case fieldArea:
		return f.area.Value()
	case fieldToggle:
		if f.on {
			return "true"
		}
		return ""
	}
	return f.input.Value()
}

type formResult int

const (
	formContinue formResult = iota
	formSubmit
	formCancel
)

// form is a vertical list of fields with keyboard navigation and validation.
type form struct {
	t      *theme
	fields []*field
	focus  int
	width  int
	height int // available lines for the fields, 0 = unlimited
}

func newForm(t *theme) *form { return &form{t: t} }

func (f *form) addText(key, label, value, placeholder, hint string, validate func(string) error) *form {
	in := textinput.New()
	in.Prompt = ""
	in.SetStyles(f.t.inputStyles())
	in.Placeholder = placeholder
	in.SetValue(value)
	f.fields = append(f.fields, &field{key: key, label: label, hint: hint, kind: fieldText, input: in, validate: validate})
	return f
}

func (f *form) addArea(key, label, value, hint string) *form {
	ta := textarea.New()
	ta.Prompt = "┃ "
	ta.SetStyles(f.t.areaStyles())
	ta.ShowLineNumbers = false
	ta.SetHeight(min(max(3, strings.Count(value, "\n")+2), 8))
	ta.SetValue(value)
	f.fields = append(f.fields, &field{key: key, label: label, hint: hint, kind: fieldArea, area: ta})
	return f
}

func (f *form) addToggle(key, label string, on bool, hint string) *form {
	f.fields = append(f.fields, &field{key: key, label: label, hint: hint, kind: fieldToggle, on: on})
	return f
}

// setPlaceholder changes the placeholder of a text field.
func (f *form) setPlaceholder(key, placeholder string) {
	for _, fl := range f.fields {
		if fl.key == key && fl.kind == fieldText {
			fl.input.Placeholder = placeholder
		}
	}
}

// get returns the value of the field with key.
func (f *form) get(key string) string {
	for _, fl := range f.fields {
		if fl.key == key {
			return fl.value()
		}
	}
	return ""
}

func (f *form) setWidth(w int) {
	f.width = w
	for _, fl := range f.fields {
		switch fl.kind {
		case fieldText:
			fl.input.SetWidth(max(10, w-4))
		case fieldArea:
			fl.area.SetWidth(max(10, w-2))
		}
	}
}

// start focuses the first field.
func (f *form) start() tea.Cmd { return f.setFocus(0) }

func (f *form) setFocus(i int) tea.Cmd {
	if len(f.fields) == 0 {
		return nil
	}
	f.focus = (i + len(f.fields)) % len(f.fields)
	var cmd tea.Cmd
	for j, fl := range f.fields {
		fl.input.Blur()
		fl.area.Blur()
		if j == f.focus {
			switch fl.kind {
			case fieldText:
				cmd = fl.input.Focus()
			case fieldArea:
				cmd = fl.area.Focus()
			}
		}
	}
	return cmd
}

// validateField checks one field, recording its error.
func (f *form) validateField(fl *field) bool {
	fl.err = ""
	if fl.validate != nil {
		if err := fl.validate(fl.value()); err != nil {
			fl.err = err.Error()
			return false
		}
	}
	return true
}

func (f *form) validateAll() bool {
	first := -1
	for i, fl := range f.fields {
		fl.err = ""
		if fl.validate != nil {
			if err := fl.validate(fl.value()); err != nil {
				fl.err = err.Error()
				if first < 0 {
					first = i
				}
			}
		}
	}
	if first >= 0 {
		f.setFocus(first)
		return false
	}
	return true
}

func (f *form) submit() (formResult, tea.Cmd) {
	if f.validateAll() {
		return formSubmit, nil
	}
	return formContinue, nil
}

// update handles navigation keys and forwards everything else to the
// focused input.
func (f *form) update(msg tea.Msg) (formResult, tea.Cmd) {
	if len(f.fields) == 0 {
		return formSubmit, nil
	}
	cur := f.fields[f.focus]
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc":
			return formCancel, nil
		case "ctrl+s":
			return f.submit()
		case "tab":
			return formContinue, f.setFocus(f.focus + 1)
		case "shift+tab":
			return formContinue, f.setFocus(f.focus - 1)
		case "down":
			if cur.kind != fieldArea {
				return formContinue, f.setFocus(f.focus + 1)
			}
		case "up":
			if cur.kind != fieldArea {
				return formContinue, f.setFocus(f.focus - 1)
			}
		case "enter":
			if cur.kind != fieldArea {
				if f.focus == len(f.fields)-1 {
					return f.submit()
				}
				if !f.validateField(cur) {
					return formContinue, nil
				}
				return formContinue, f.setFocus(f.focus + 1)
			}
		case "space", "x":
			if cur.kind == fieldToggle {
				cur.on = !cur.on
				return formContinue, nil
			}
		}
	}
	var cmd tea.Cmd
	switch cur.kind {
	case fieldText:
		cur.input, cmd = cur.input.Update(msg)
	case fieldArea:
		cur.area, cmd = cur.area.Update(msg)
	}
	return formContinue, cmd
}

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
