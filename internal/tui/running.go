package tui

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"

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
// "step x / n" and the step being performed. ctrl+c halts it.
type runningScreen struct {
	rite, aspect string
	step, steps  int
	verse        *invocation.Verse
	casts        []vox.Message
	halt         context.CancelFunc
	halting      bool
}

func (m *model) isRunning() bool {
	_, ok := m.screen.(*runningScreen)
	return ok
}

// invoke performs rite into aspect, inscribed by runes, in its own
// goroutine, and shows the running panel until it has ended. The
// invocation's words are captured, never printed: the cogitator owns the
// screen.
func (m *model) invoke(r *librarium.Rite, aspect string, runes map[string]string) tea.Cmd {
	ctx, halt := context.WithCancel(context.Background())
	rs := &runningScreen{rite: r.Name, aspect: aspect, halt: halt}
	for _, step := range r.Liturgy {
		if step.Kind() != librarium.KindVoxCast {
			rs.steps++
		}
	}
	m.screen = rs
	s, send := m.s, m.send
	p := rituals.Petition{Rite: r.Name, Aspect: aspect, Runes: runes, Herald: herald{send}}
	go func() {
		defer halt()
		res, err := s.Invoke(ctx, p)
		send(invokedMsg{res: res, err: err})
	}()
	return nil
}

// invoked ends the running panel: triumph is told in a toast, a fall or a
// refusal in a modal.
func (m *model) invoked(msg invokedMsg) tea.Cmd {
	m.screen = nil
	res, err := msg.res, msg.err
	name := ""
	if res.Rite != nil {
		name = res.Rite.Name
	}
	m.reload(name)
	var hs invocation.Heresies
	switch {
	case errors.As(err, &hs):
		m.screen = newTextScreen("The pre-flight forbids the invocation",
			m.t.text.Render("The pre-flight forbids invoking "+name+" into "+res.Options.Aspect+"; nothing was touched.")+
				"\n\n"+m.foresight(res, err))
		return nil
	case err != nil:
		return m.notify(toastErr, err.Error())
	}
	m.last = &res
	if out := res.Outcome; out.Verdict != invocation.Triumph {
		m.screen = newTextScreen("The rite "+name+" has fallen", m.fallen(*out))
		return nil
	}
	text := fmt.Sprintf("The rite %s is performed: %s → %s. The Omnissiah is pleased.", name,
		former(res.Options.Former), res.Options.Aspect)
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
	b := []string{t.bold.Render(fmt.Sprintf("step %d / %d", s.step, s.steps)) + "  " + progressBar(s.step, s.steps, w-16)}
	if s.verse != nil {
		b = append(b, t.accent.Render("▸ ")+t.text.Render(truncate(verseName(*s.verse), w-2)))
	} else {
		b = append(b, t.dim.Render("  the liturgy awaits its first step"))
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
		b = append(b, "", t.warn.Render("Halting the rite; its deeds are being undone…"))
	}
	return t.modal("Invocation of "+s.rite+" → "+s.aspect, strings.Join(b, "\n"), m.width-4)
}

func (s *runningScreen) hints(m *model) [][2]string {
	return [][2]string{{"ctrl+c", "halt the rite"}}
}

// progressBar is a bar of width cells filled for done of all.
func progressBar(done, all, width int) string {
	width = max(4, width)
	filled := width
	if all > 0 {
		filled = done * width / all
	}
	return "[" + strings.Repeat("■", filled) + strings.Repeat("·", width-filled) + "]"
}
