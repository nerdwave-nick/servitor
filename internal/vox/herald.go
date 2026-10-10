package vox

import (
	"fmt"
	"io"
	"math/rand/v2"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// Config is what a herald is raised with.
type Config struct {
	// Vox is the vox of the settings (librarium.Orders.Vox): "off" silences
	// the desktop; a terminal still hears every vox-cast.
	Vox string
	// Terminal returns the controlling terminal to print to, or nil when
	// there is none; nil means ControllingTerminal.
	Terminal func() io.Writer
	// Pick chooses a template: an index below n; nil chooses at random.
	Pick func(n int) int
}

// Herald sends the vox-casts of one invocation. With a controlling terminal
// it prints them there, one line each; without one it sends them as a
// single desktop notification replaced in place, unless the vox is "off".
// It never fails the invocation: a missing or fallen notify-send stays
// silent. Raise one herald per invocation and Close it afterwards. It is
// not safe for concurrent use.
type Herald struct {
	cfg     Config
	decided bool      // whether the terminal was looked for
	term    io.Writer // the controlling terminal; nil when there is none
	desktop desktop
}

// New raises a herald for one invocation.
func New(c Config) *Herald {
	if c.Terminal == nil {
		c.Terminal = ControllingTerminal
	}
	if c.Pick == nil {
		c.Pick = rand.IntN
	}
	return &Herald{cfg: c}
}

// Proclaim sends the vox-cast of p. It satisfies invocation.Herald.
func (h *Herald) Proclaim(p invocation.Proclamation) {
	m := Compose(p, h.cfg.Pick)
	if !h.decided {
		h.decided, h.term = true, h.cfg.Terminal()
	}
	switch {
	case h.term != nil:
		fmt.Fprintf(h.term, "+++ %s +++ %s\n", m.Summary, strings.ReplaceAll(m.Body, "\n", " · "))
	case h.cfg.Vox != librarium.VoxOff:
		h.desktop.send(m, false)
	}
}

// Close lets the herald rest after the invocation: a progress left on the
// desktop, which never expires while the rite runs, is sent once more so
// that it may expire; the controlling terminal is closed when the herald
// opened it.
func (h *Herald) Close() error {
	h.desktop.rest()
	if c, ok := h.term.(io.Closer); ok {
		h.term = nil
		return c.Close()
	}
	return nil
}
