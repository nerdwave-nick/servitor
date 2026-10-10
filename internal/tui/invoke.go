package tui

import (
	"context"
	"errors"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
)

// invokeScreen lets the user choose an aspect and inscribe the invocation
// before it is performed, and foresee it.
type invokeScreen struct {
	rite    *librarium.Rite
	cursor  int
	current string
	form    *form // non-nil once an aspect of a rite with inscriptions was chosen
}

func newInvokeScreen(r *row) *invokeScreen {
	s := &invokeScreen{rite: r.rite(), current: r.reading.Aspect}
	for i, a := range s.rite.Aspects {
		if a == s.current {
			s.cursor = (i + 1) % len(s.rite.Aspects) // preselect the next aspect
		}
	}
	return s
}

func (s *invokeScreen) fullscreen() bool { return false }

func (s *invokeScreen) aspect() string { return s.rite.Aspects[s.cursor] }

func (s *invokeScreen) update(m *model, msg tea.Msg) (screen, tea.Cmd) {
	if s.form != nil {
		if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "ctrl+p" {
			return s.foresee(m), nil
		}
		res, cmd := s.form.update(msg)
		switch res {
		case formCancel:
			s.form = nil
		case formSubmit:
			cmd := m.invoke(s.rite, s.aspect(), s.runes())
			return m.screen, cmd // the running panel
		}
		return s, cmd
	}
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return s, nil
	}
	n := len(s.rite.Aspects)
	switch key := k.String(); key {
	case "esc", "q":
		return nil, nil
	case "up", "k":
		s.cursor = (s.cursor - 1 + n) % n
	case "down", "j", "tab":
		s.cursor = (s.cursor + 1) % n
	case "p":
		return s.foresee(m), nil
	case "enter", "space":
		return s.choose(m)
	default:
		if i, err := strconv.Atoi(key); err == nil && i >= 1 && i <= n {
			s.cursor = i - 1
			return s.choose(m)
		}
	}
	return s, nil
}

// choose invokes the aspect at once, or asks for the inscriptions first.
func (s *invokeScreen) choose(m *model) (screen, tea.Cmd) {
	if len(s.rite.Inscriptions) == 0 {
		cmd := m.invoke(s.rite, s.aspect(), nil)
		return m.screen, cmd // the running panel
	}
	f := newForm(m.t)
	for _, in := range s.rite.Inscriptions {
		label := in.Key
		if in.Mandatory {
			label += " *"
		}
		decree, _ := in.Decrees.For(s.aspect())
		f.addText(in.Key, label, decree, "", in.Purpose, nil)
	}
	f.setWidth(min(60, m.width-12))
	s.form = f
	return s, f.start()
}

// runes are the inscriptions as written in the form; an emptied one clears
// its decree. Without the form the decrees stand.
func (s *invokeScreen) runes() map[string]string {
	if s.form == nil {
		return nil
	}
	out := map[string]string{}
	for _, fl := range s.form.fields {
		out[fl.key] = fl.value()
	}
	return out
}

// foresee shows what invoking the chosen aspect would do; nothing is
// performed, not even the auspex.
func (s *invokeScreen) foresee(m *model) screen {
	res, err := m.s.Invoke(context.Background(),
		rituals.Petition{Rite: s.rite.Name, Aspect: s.aspect(), Runes: s.runes(), Foresee: true})
	ts := newTextScreen("Foresight of aspect "+s.aspect(), m.foresight(res, err))
	ts.back = s
	return ts
}

func (s *invokeScreen) view(m *model) string {
	t := m.t
	title := "Invoke " + s.rite.Name
	if s.form != nil {
		head := t.text.Render("Inscriptions for aspect ") + t.accent.Bold(true).Render(s.aspect())
		return t.modal(title, head+"\n\n"+s.form.view(), m.width-4)
	}
	lines := []string{t.text.Render("Choose the aspect to invoke:"), ""}
	for i, a := range s.rite.Aspects {
		num := t.dim.Render(strconv.Itoa(i+1) + " ")
		note := ""
		if a == s.current {
			note = t.dim.Render("  (current aspect)")
		}
		if i == s.cursor {
			lines = append(lines, t.accent.Render("▸ ")+num+t.selected.Render(" "+a+" ")+note)
		} else {
			lines = append(lines, "  "+num+t.text.Render(a)+note)
		}
	}
	return t.modal(title, strings.Join(lines, "\n"), m.width-4)
}

func (s *invokeScreen) hints(m *model) [][2]string {
	if s.form != nil {
		return [][2]string{{"enter", "perform the rite"}, {"tab", "next inscription"},
			{"ctrl+p", "foresee"}, {"esc", "back"}}
	}
	return [][2]string{{"enter", "choose"}, {"1-9", "swift choice"},
		{"p", "foresee"}, {"esc", "withdraw"}}
}

// foresight renders what an invocation would do, or why the pre-flight
// forbids it.
func (m *model) foresight(res rituals.Result, err error) string {
	t := m.t
	var b []string
	var hs invocation.Heresies
	switch {
	case errors.As(err, &hs):
		b = append(b, t.danger.Render("The pre-flight forbids this invocation:"))
		for _, h := range hs {
			b = append(b, t.danger.Render("  "+h.Error()))
		}
		b = append(b, "")
	case err != nil:
		return t.danger.Render(err.Error())
	}
	for _, f := range res.Foresight {
		b = append(b, t.bold.Render(verseName(f.Verse)))
		for _, line := range foreseen(f) {
			st := t.text
			switch {
			case strings.HasPrefix(line, "  - "):
				st = t.danger
			case strings.HasPrefix(line, "  + "):
				st = t.ok
			}
			b = append(b, st.Render(line))
		}
	}
	return strings.Join(append(b, t.dim.Render("Nothing was performed.")), "\n")
}
