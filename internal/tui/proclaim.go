package tui

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
)

// proclaimScreen is the invoke screen of an edict, which has no aspect to
// choose: it asks for the edict's inscriptions when it bears any, then
// for confirmation. Only enter upon the confirmation proclaims the edict;
// a single key from the overview never does.
type proclaimScreen struct {
	rite       *librarium.Rite
	form       *form // the inscriptions; nil when the edict bears none
	confirming bool
}

// proclaim opens the invoke screen of the edict r.
func (m *model) proclaim(r *librarium.Rite) tea.Cmd {
	s := &proclaimScreen{rite: r, confirming: len(r.Inscriptions) == 0}
	m.screen = s
	if s.confirming {
		return nil
	}
	s.form = m.inscriptionForm(r, "")
	return s.form.start()
}

func (s *proclaimScreen) fullscreen() bool { return false }

func (s *proclaimScreen) update(m *model, msg tea.Msg) (screen, tea.Cmd) {
	k, isKey := msg.(tea.KeyPressMsg)
	if !s.confirming {
		if isKey && k.String() == "ctrl+p" {
			return s.foresee(m), nil
		}
		res, cmd := s.form.update(msg)
		switch res {
		case formCancel:
			return nil, nil
		case formSubmit:
			s.confirming = true
			return s, nil
		}
		return s, cmd
	}
	if !isKey {
		return s, nil
	}
	switch k.String() {
	case "esc", "q":
		if s.form == nil {
			return nil, nil
		}
		s.confirming = false // back to the inscriptions, as written
	case "p":
		return s.foresee(m), nil
	case "enter":
		cmd := m.invoke(s.rite, "", runesOf(s.form))
		return m.screen, cmd // the running panel
	}
	return s, nil
}

// foresee shows what proclaiming the edict would do; nothing is performed.
func (s *proclaimScreen) foresee(m *model) screen {
	res, err := m.s.Invoke(context.Background(),
		rituals.Petition{Rite: s.rite.Name, Runes: runesOf(s.form), Foresee: true})
	ts := newTextScreen("Foresight of the edict "+s.rite.Name, m.foresight(res, err))
	ts.back = s
	return ts
}

func (s *proclaimScreen) view(m *model) string {
	t := m.t
	title := "Proclaim the edict " + s.rite.Name
	if !s.confirming {
		head := t.text.Render("Inscriptions for the edict ") + t.accent.Bold(true).Render(s.rite.Name)
		return t.modal(title, head+"\n\n"+s.form.view(), m.width-4)
	}
	b := []string{
		t.text.Render("The edict ") + t.accent.Bold(true).Render(s.rite.Name) +
			t.text.Render(" is to be proclaimed: every verse of its liturgy is performed anew."),
	}
	if s.form != nil {
		b = append(b, "", t.label.Render(padRight("INSCRIBED", labelW))+m.inscribed(nonEmpty(runesOf(s.form))))
	}
	b = append(b, "", t.dim.Render("Press enter once more to proclaim it, p to foresee it, esc to retreat."))
	return t.modal(title, strings.Join(b, "\n"), m.width-4)
}

func (s *proclaimScreen) hints(m *model) [][2]string {
	if !s.confirming {
		return [][2]string{{"enter", "onward to the proclaiming"}, {"tab", "next inscription"},
			{"ctrl+p", "foresee"}, {"esc", "retreat"}}
	}
	return [][2]string{{"enter", "proclaim the edict"}, {"p", "foresee"}, {"esc", "retreat"}}
}

// nonEmpty are the inscriptions that bear a value; an emptied one is
// borne by none.
func nonEmpty(ins map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range ins {
		if v != "" {
			out[k] = v
		}
	}
	return out
}
