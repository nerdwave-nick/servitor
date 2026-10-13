package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/rituals"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// provenMsg tells that a sanctioning or an abjuring asked for the master's
// proof and ended.
type provenMsg struct {
	rite    string
	exalted string // what the rite holds exalted, as exaltedOf names it
	abjured bool   // an abjuring; else a sanctioning
	sigil   string
	err     error
	words   string  // what the intercessor and the exalted servitor spoke upon stderr
	laments []error // what went amiss after it, undoing nothing
}

// sanctionScreen shows the master what a rite holds exalted as it
// would be sanctioned — the whole unit beneath its sigil, or only what
// changed since the sanction that stands — or why none may be. enter asks
// the intercessor for the one proof that lays the sanction, a for the one
// that abjures the sanction kept under the rite's name; while the
// intercessor asks, the screen awaits it.
type sanctionScreen struct {
	*textScreen
	rite     string
	exalted  string            // what the rite holds exalted, as exaltedOf names it
	offering *rituals.Offering // nil when nothing may be sanctioned
	kept     bool              // a sanction is laid under the rite's name
	proving  bool
}

// openSanction opens the sanction of the rite of r.
func (m *model) openSanction(r *row) {
	o, err := m.s.Offer(r.name)
	s := &sanctionScreen{rite: r.name, exalted: exaltedOf(r.rite(), true), offering: o,
		kept: r.sanction != sanction.None && r.sanction != sanction.Unsanctioned}
	s.textScreen = newTextScreen("Sanction of "+r.name, m.offered(r.name, o, err, s.kept))
	m.screen = s
}

// offered renders what the master beholds before the proof.
func (m *model) offered(name string, o *rituals.Offering, err error, kept bool) string {
	t := m.t
	var b []string
	if err != nil {
		for _, l := range strings.Split(err.Error(), "\n") {
			b = append(b, t.danger.Render(l))
		}
		if kept {
			b = append(b, "", t.dim.Render("a abjures the sanction kept under its name, with one proof of the master."))
		}
		return strings.Join(b, "\n")
	}
	if o.Lament != nil {
		b = append(b, t.warn.Render(o.Lament.Error()), "")
	}
	shown, exalted := o.Shown(), exaltedOf(o.Rite, true)
	switch {
	case o.Unchanged():
		b = append(b, t.ok.Render(fmt.Sprintf("✠ The %s of %s stand sanctioned already under the sigil %s.",
			exalted, name, sanction.Short(o.Sigil()))), t.text.Render("Nothing is to be sanctioned anew."), "")
		shown = sanction.Show(o.Unit)
	case o.Kept == nil:
		b = append(b, t.bold.Render(fmt.Sprintf("✠ The %s of %s, offered for sanction under the sigil %s:",
			exalted, name, o.Sigil())), "")
	default:
		b = append(b, t.bold.Render(fmt.Sprintf("✠ The %s of %s changed since they were sanctioned under the sigil "+
			"%s; offered anew under the sigil %s:", exalted, name, sanction.Short(o.Kept.Sigil), o.Sigil())), "")
	}
	for _, l := range shown {
		st := t.text
		switch {
		case strings.HasPrefix(l, "- "):
			st = t.danger
		case strings.HasPrefix(l, "+ "):
			st = t.ok
		}
		b = append(b, st.Render(l))
	}
	if !o.Unchanged() {
		b = append(b, "", t.accent.Render("⋯ The intercessor asks the master's proof; the proof is the sanction."))
	}
	return strings.Join(b, "\n")
}

func (s *sanctionScreen) update(m *model, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok || s.proving {
		return s, nil
	}
	switch k.String() {
	case "enter":
		if s.offering == nil || s.offering.Unchanged() {
			return s, nil
		}
		o, sv := s.offering, m.s
		s.prove(m, false, func(x rituals.Exaltation) provenMsg {
			return provenMsg{rite: s.rite, exalted: s.exalted, sigil: o.Sigil(), err: sv.Sanction(o, x)}
		})
	case "a":
		if !s.kept {
			return s, nil
		}
		sv := m.s
		s.prove(m, true, func(x rituals.Exaltation) provenMsg {
			_, err := sv.Abjure(s.rite, x)
			return provenMsg{rite: s.rite, exalted: s.exalted, abjured: true, err: err}
		})
	default:
		next, cmd := s.textScreen.update(m, msg)
		if next != s.textScreen { // withdrawn
			return next, cmd
		}
	}
	return s, nil
}

// prove asks the master's proof for the deed in its own goroutine, through
// the cogitator's intercession, and tells how it ended.
func (s *sanctionScreen) prove(m *model, abjure bool, deed func(rituals.Exaltation) provenMsg) {
	s.proving = true
	c, send := m.intercede(), m.send
	m.running.Add(1)
	go func() {
		defer m.running.Done()
		msg := deed(*c.exaltation())
		msg.abjured = abjure
		msg.words = c.words.String()
		msg.laments = appendLament(nil, c.forgotten())
		send(msg)
	}()
}

func (s *sanctionScreen) view(m *model) string {
	if !s.proving {
		return s.textScreen.view(m)
	}
	return m.t.modal(s.title, m.t.accent.Render("⋯ The exalted servitor awaits the master's proof, which the "+
		"intercessor asks."), m.width-4)
}

func (s *sanctionScreen) hints(m *model) [][2]string {
	if s.proving {
		return [][2]string{{"⋯", "the intercessor asks the master's proof"}}
	}
	var h [][2]string
	if s.offering != nil && !s.offering.Unchanged() {
		h = append(h, [2]string{"enter", "sanction with one proof"})
	}
	if s.kept {
		h = append(h, [2]string{"a", "abjure the sanction"})
	}
	return append(h, [2]string{"j/k", "scroll the scroll"}, [2]string{"esc", "withdraw"})
}

// proven ends the sanctioning or abjuring: triumph is told in a toast, a
// refusal as a heresy, with the words the intercessor spoke.
func (m *model) proven(msg provenMsg) tea.Cmd {
	m.screen = nil
	m.reload(msg.rite)
	if msg.err != nil {
		deed, untouched := "The sanctioning of "+msg.rite+" is refused", "Nothing was written."
		if msg.abjured {
			deed, untouched = "The abjuring of the sanction of "+msg.rite+" is refused", "Nothing was struck."
		}
		b := []string{m.t.danger.Render("✠ " + msg.err.Error()), "", m.t.text.Render(untouched)}
		b = append(b, m.spoken(msg.words)...)
		for _, l := range msg.laments {
			b = append(b, m.t.warn.Render(l.Error()))
		}
		m.screen = newTextScreen(deed, strings.Join(b, "\n"))
		return nil
	}
	text := fmt.Sprintf("The %s of %s stand sanctioned under the sigil %s.", msg.exalted, msg.rite,
		sanction.Short(msg.sigil))
	if msg.abjured {
		text = "The sanction of " + msg.rite + " is abjured; its " + msg.exalted + " are performed no more " +
			"until they are sanctioned anew."
	}
	if len(msg.laments) > 0 {
		return m.notify(toastErr, text+" Yet: "+msg.laments[0].Error())
	}
	return m.notify(toastOK, text)
}
