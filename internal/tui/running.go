package tui

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"os/signal"
	"strings"
	"syscall"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
	"github.com/nerdwave-nick/servitor/internal/vox"
)

// What a running invocation tells the cogitator from its own goroutine.
type (
	begunMsg struct {
		verse       invocation.Verse
		step, steps int
	}
	voxMsg     struct{ vox.Message }
	invokedMsg struct {
		res rituals.Result
		err error
		// hungUp tells that the terminal hung up while the invocation was
		// performed: the cogitator ends now that it has ended.
		hungUp bool
		// words are what the intercessor and the exalted servitor spoke
		// upon stderr.
		words string
	}
)

// herald hears a running invocation for the cogitator: the vox-casts are
// composed as the desktop would hear them and shown in the running panel,
// never sent to the desktop or printed to the terminal.
type herald struct{ send func(tea.Msg) }

func (h herald) Proclaim(p invocation.Proclamation) { h.send(voxMsg{vox.Compose(p, rand.IntN)}) }

func (h herald) Begin(v invocation.Verse, step, steps int) {
	h.send(begunMsg{verse: v, step: step, steps: steps})
}

// runningScreen shows an invocation while it is performed: its vox-casts,
// "step x / n" and the step being performed; for a state rite of pattern
// Mark II, "subject x / n" of the changing subjects, the subject being set
// with the omen it is set to, and the observances spoken. ctrl+c halts it.
type runningScreen struct {
	rite, aspect string
	edict        bool            // an edict is proclaimed, into no aspect
	scripture    *librarium.Rite // the rite performed
	state        bool            // a state rite of pattern Mark II is invoked
	step, steps  int
	verse        *invocation.Verse
	plans        map[string]invocation.SubjectPlan // of a state rite's subjects, once drawn
	spoken       []string                          // the observances spoken
	casts        []vox.Message
	halt         context.CancelFunc
	halting      bool
}

// isRunning tells that an invocation is performed, or that the
// intercessor asks the master's proof for a sanctioning or an abjuring:
// the cogitator does not end before it has ended.
func (m *model) isRunning() bool {
	switch s := m.screen.(type) {
	case *runningScreen:
		return true
	case *sanctionScreen:
		return s.proving
	}
	return false
}

// invoke performs rite into aspect, inscribed by runes, in its own
// goroutine, and shows the running panel until it has ended. The
// invocation's words are captured, never printed: the cogitator owns the
// screen.
//
// Like every invocation it is steadfast (rituals.Steadfast): a hangup
// neither halts it nor ends the cogitator before it has ended. An
// interrupt or SIGTERM sent to the cogitator halts and reverts it, as
// ctrl+c does.
//
// Its exalted verses are performed by the exalted servitor alone, raised
// through the cogitator's intercession; run0 is bidden forget the master's
// proof once it has ended.
func (m *model) invoke(r *librarium.Rite, aspect string, runes map[string]string) tea.Cmd {
	ctx, halt := context.WithCancel(context.Background())
	rs := &runningScreen{rite: r.Name, aspect: aspect, edict: r.Edict, scripture: r, state: r.MarkII && !r.Edict,
		halt: halt}
	for _, step := range r.Liturgy {
		if step.Kind() != librarium.KindVoxCast {
			rs.steps++
		}
	}
	m.screen = rs
	s, send, c := m.s, m.send, m.intercede()
	p := rituals.Petition{Rite: r.Name, Aspect: aspect, Runes: runes, Herald: herald{send},
		Exaltation: c.exaltation(), Commence: commence(r, send)}
	m.running.Add(1)
	go func() {
		defer m.running.Done()
		defer halt()
		halted, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()
		hangup := heedHangup()
		res, err := s.Invoke(halted, p)
		res.Laments = appendLament(res.Laments, c.forgotten())
		send(invokedMsg{res: res, err: err, hungUp: hangup(), words: c.words.String()})
	}()
	return nil
}

// heedHangup listens for a hangup until the returned heard is asked
// whether one came. A hangup ignored since the servitor awoke stays
// ignored, and is never heard.
func heedHangup() (heard func() bool) {
	if signal.Ignored(syscall.SIGHUP) {
		return func() bool { return false }
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	return func() bool {
		signal.Stop(c)
		return len(c) > 0
	}
}

// invoked ends the running panel: triumph is told in a toast, a fall or a
// refusal in a modal. A rite with exalted verses that is refused before
// any verse — no intercessor found, the servitor not installed, a proof
// dismissed, an exalted servitor of another forging or one that refused —
// is told as a heresy, with the words its intercessor spoke.
func (m *model) invoked(msg invokedMsg) tea.Cmd {
	m.screen = nil
	res, err := msg.res, msg.err
	name, edict := "", false
	if res.Rite != nil {
		name, edict = res.Rite.Name, res.Rite.Edict
	}
	m.reload(name)
	var hs invocation.Heresies
	switch {
	case errors.As(err, &hs):
		forbids := "The pre-flight forbids invoking " + name + " into " + res.Options.Aspect
		if edict {
			forbids = "The pre-flight forbids the proclaiming of the edict " + name
		}
		m.screen = newTextScreen("The pre-flight forbids the invocation",
			m.t.text.Render(forbids+"; nothing was touched.")+"\n\n"+m.foresight(res, err))
		return nil
	case err != nil && res.Rite != nil && res.Rite.Exalted():
		b := append([]string{m.t.danger.Render("✠ " + err.Error()), "", m.t.text.Render("Nothing was touched: " +
			"not one verse of the " + riteNoun(edict) + " " + name + " was performed.")}, m.spoken(msg.words)...)
		for _, l := range res.Laments {
			b = append(b, m.t.warn.Render(l.Error()))
		}
		m.screen = newTextScreen("No exalted servitor was raised", strings.Join(b, "\n"))
		return nil
	case err != nil:
		return m.notify(toastErr, err.Error())
	}
	m.last = &res
	if out := res.Outcome; out.Verdict != invocation.Triumph {
		body := m.fallen(res)
		if words := m.spoken(msg.words); words != nil {
			body += "\n" + strings.Join(words, "\n")
		}
		m.screen = newTextScreen("The "+riteNoun(edict)+" "+name+" has fallen", body)
		return nil
	}
	text := fmt.Sprintf("The rite %s is performed: %s → %s. The Omnissiah is pleased.", name,
		former(res.Options.Former), res.Options.Aspect)
	if edict {
		text = "The edict " + name + " is proclaimed. The Omnissiah is pleased."
	}
	if len(res.Laments) > 0 {
		return m.notify(toastErr, text+" Yet: "+res.Laments[0].Error())
	}
	return m.notify(toastOK, text)
}

func (s *runningScreen) fullscreen() bool { return false }

func (s *runningScreen) update(m *model, msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case begunMsg:
		s.step, s.steps, s.verse = msg.step, msg.steps, &msg.verse
	case commencedMsg:
		s.plans, s.steps = msg.plans, changing(msg.plans)
	case spokenMsg:
		s.spoken = append(s.spoken, msg.verse.Observance)
	case voxMsg:
		s.casts = append(s.casts, msg.Message)
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" && !s.halting {
			s.halting = true
			s.halt()
		}
	}
	return s, nil
}

func (s *runningScreen) view(m *model) string {
	t := m.t
	w := min(72, m.width-8)
	unit := "step"
	if s.state {
		unit = "subject"
	}
	b := []string{t.bold.Render(fmt.Sprintf("%s %d / %d", unit, s.step, s.steps)) + "  " + progressBar(max(0, s.step-1), s.steps, w-16)}
	switch {
	case s.verse != nil:
		b = append(b, t.accent.Render("▸ ")+t.text.Render(truncate(s.setting(*s.verse), w-2)))
	case s.state:
		b = append(b, t.dim.Render("  the subjects await their setting"))
	default:
		b = append(b, t.dim.Render("  the liturgy awaits its first step"))
	}
	if len(s.spoken) > 0 {
		b = append(b, t.dim.Render(truncate("  observances spoken: "+strings.Join(s.spoken, ", "), w)))
	}
	if len(s.casts) > 0 {
		b = append(b, "", t.label.Render("VOX-CASTS"))
	}
	for _, c := range s.casts[max(0, len(s.casts)-4):] {
		for _, line := range strings.Split(c.Body, "\n") {
			b = append(b, t.text.Render(truncate(line, w)))
		}
	}
	if s.halting {
		b = append(b, "", t.warn.Render("Halting the "+riteNoun(s.edict)+"; its deeds are being undone…"))
	}
	title := "Invocation of " + s.rite + " → " + s.aspect
	if s.edict {
		title = "Proclamation of the edict " + s.rite
	}
	return t.modal(title, strings.Join(b, "\n"), m.width-4)
}

// setting names the step being performed: a subject of a state rite with
// its designation, and a machine-spirit with the omen read and the omen it
// is set to.
func (s *runningScreen) setting(v invocation.Verse) string {
	if !s.state {
		return verseName(v)
	}
	name := named(s.scripture, v)
	if p, ok := s.plans[v.Subject]; ok && spirit(s.scripture, v) {
		name += fmt.Sprintf(": %s → %q", readOmen(p), p.Target)
	}
	return name
}

func (s *runningScreen) hints(m *model) [][2]string {
	return [][2]string{{"ctrl+c", "halt the " + riteNoun(s.edict)}}
}

// appendLament appends the lament err, when there is one.
func appendLament(laments []error, err error) []error {
	if err == nil {
		return laments
	}
	return append(laments, err)
}

// progressBar is a bar of width cells filled for done of all; the step
// being performed is not yet done.
func progressBar(done, all, width int) string {
	width = max(4, width)
	filled := width
	if all > 0 {
		filled = done * width / all
	}
	return "[" + strings.Repeat("■", filled) + strings.Repeat("·", width-filled) + "]"
}
