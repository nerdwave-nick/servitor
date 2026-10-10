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
	fieldChoice
)

// field is one input of a form.
type field struct {
	key, label, hint string
	kind             fieldKind
	input            textinput.Model
	area             textarea.Model
	on               bool
	choices          []string // the values of a choice
	pick             int      // the chosen one
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
	case fieldChoice:
		return f.choices[f.pick]
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

// addChoice adds a choice among choices, value chosen first.
func (f *form) addChoice(key, label string, choices []string, value, hint string) *form {
	fl := &field{key: key, label: label, hint: hint, kind: fieldChoice, choices: choices}
	for i, c := range choices {
		if c == value {
			fl.pick = i
		}
	}
	f.fields = append(f.fields, fl)
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
		case "space", "x", "right", "l", "left", "h":
			if r, ok := f.turn(cur, k.String()); ok {
				return r, nil
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

// turn changes a toggle or a choice by key; ok is false when the key
// belongs to the focused input.
func (f *form) turn(fl *field, key string) (formResult, bool) {
	switch {
	case fl.kind == fieldToggle && (key == "space" || key == "x"):
		fl.on = !fl.on
	case fl.kind == fieldChoice && (key == "left" || key == "h"):
		fl.pick = (fl.pick + len(fl.choices) - 1) % len(fl.choices)
	case fl.kind == fieldChoice && key != "x":
		fl.pick = (fl.pick + 1) % len(fl.choices)
	default:
		return formContinue, false
	}
	return formContinue, true
}
