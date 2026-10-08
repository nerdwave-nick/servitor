package tui

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/config"
	"github.com/nerdwave-nick/servitor/internal/engine"
)

// invokeScreen lets the user pick a state and adjust metadata before applying.
type invokeScreen struct {
	sw      *config.Switch
	cursor  int
	current string
	form    *form // non-nil once a state with metadata keys was chosen
}

func newInvokeScreen(m *model, sw *config.Switch) *invokeScreen {
	s := &invokeScreen{sw: sw, current: m.current().status.State()}
	for i, st := range sw.States {
		if st == s.current {
			s.cursor = (i + 1) % len(sw.States) // preselect the next aspect
		}
	}
	return s
}

func (s *invokeScreen) fullscreen() bool { return false }

func (s *invokeScreen) state() string { return s.sw.States[s.cursor] }

func (s *invokeScreen) update(m *model, msg tea.Msg) (screen, tea.Cmd) {
	if s.form != nil {
		if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "ctrl+p" {
			return s.preview(m), nil
		}
		res, cmd := s.form.update(msg)
		switch res {
		case formCancel:
			s.form = nil
		case formSubmit:
			return nil, m.perform(s.sw, s.state(), s.overrides())
		}
		return s, cmd
	}
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return s, nil
	}
	switch key := k.String(); key {
	case "esc", "q":
		return nil, nil
	case "up", "k":
		s.cursor = (s.cursor - 1 + len(s.sw.States)) % len(s.sw.States)
	case "down", "j", "tab":
		s.cursor = (s.cursor + 1) % len(s.sw.States)
	case "p":
		return s.preview(m), nil
	case "enter", "space":
		return s.choose(m)
	default:
		if n, err := strconv.Atoi(key); err == nil && n >= 1 && n <= len(s.sw.States) {
			s.cursor = n - 1
			return s.choose(m)
		}
	}
	return s, nil
}

// choose either applies the state directly or asks for metadata first.
func (s *invokeScreen) choose(m *model) (screen, tea.Cmd) {
	keys := s.sw.MetaKeys()
	if len(keys) == 0 {
		return nil, m.perform(s.sw, s.state(), nil)
	}
	f := newForm(m.t)
	for _, k := range sortedKeys(keys) {
		label := k
		if keys[k].Required() {
			label += " *"
		}
		f.addText(k, label, s.configured(k), "", keys[k].Description, nil)
	}
	f.setWidth(min(60, m.width-12))
	s.form = f
	return s, f.start()
}

// configured returns the value prescribed for key in the chosen state.
func (s *invokeScreen) configured(key string) string {
	for _, f := range s.sw.Files {
		if v, ok := f.ValueFor(s.state()); ok {
			if val, ok := v.Meta[key]; ok {
				return val
			}
		}
	}
	return ""
}

func (s *invokeScreen) overrides() map[string]string {
	out := map[string]string{}
	if s.form == nil {
		return out
	}
	for _, fl := range s.form.fields {
		if v := fl.value(); v != s.configured(fl.key) {
			out[fl.key] = v
		}
	}
	return out
}

// preview shows the dry-run diff of the chosen state.
func (s *invokeScreen) preview(m *model) screen {
	l, t := m.lex, m.t
	res, err := engine.Apply(s.sw, s.state(), s.overrides(), engine.Options{DryRun: true})
	var b strings.Builder
	if err != nil {
		b.WriteString(t.danger.Render(err.Error()))
	}
	for _, c := range res.Changes {
		b.WriteString(t.bold.Render(shortPath(c.Path)) + "\n")
		diff := engine.DiffLines(c.Before, c.After)
		if len(diff) == 0 {
			b.WriteString(t.dim.Render(l.P("  undisturbed", "  unchanged")) + "\n")
		}
		for _, line := range diff {
			st := t.ok
			if strings.HasPrefix(line, "-") {
				st = t.danger
			}
			b.WriteString(st.Render("  "+line) + "\n")
		}
	}
	ts := newTextScreen(l.P("Augury of aspect ", "Preview of ")+s.state(), strings.TrimRight(b.String(), "\n"))
	ts.back = s
	return ts
}

func (s *invokeScreen) view(m *model) string {
	t, l := m.t, m.lex
	title := l.P("Invoke ", "Apply ") + s.sw.Name
	if s.form != nil {
		head := t.text.Render(l.P("Inscriptions for aspect ", "Metadata for state ")) + t.accent.Bold(true).Render(s.state())
		return t.modal(title, head+"\n\n"+s.form.view(), m.width-4)
	}
	lines := []string{t.text.Render(l.P("Choose the aspect to invoke:", "Choose the state to apply:")), ""}
	for i, st := range s.sw.States {
		num := t.dim.Render(strconv.Itoa(i+1) + " ")
		label := st
		if st == s.current {
			label += t.dim.Render(l.P("  (current aspect)", "  (current)"))
		}
		if i == s.cursor {
			lines = append(lines, t.accent.Render("▸ ")+num+t.selected.Render(" "+st+" ")+strings.TrimPrefix(label, st))
		} else {
			lines = append(lines, "  "+num+t.text.Render(label))
		}
	}
	return t.modal(title, strings.Join(lines, "\n"), m.width-4)
}

func (s *invokeScreen) hints(m *model) [][2]string {
	l := m.lex
	if s.form != nil {
		return [][2]string{{"enter", l.P("perform the rite", "apply")}, {"tab", l.P("next inscription", "next field")},
			{"ctrl+p", l.P("augury", "preview")}, {"esc", l.P("back", "back")}}
	}
	return [][2]string{{"enter", l.P("choose", "choose")}, {"1-9", l.P("swift choice", "quick pick")},
		{"p", l.P("augury (preview)", "preview")}, {"esc", l.P("withdraw", "cancel")}}
}
