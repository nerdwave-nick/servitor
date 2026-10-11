package vox

import (
	"math/rand/v2"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// Config is what a herald is raised with.
type Config struct {
	// Vox is the vox of the settings (librarium.Orders.Vox): "off" silences
	// the desktop.
	Vox string
	// Pick chooses a template: an index below n; nil chooses at random.
	Pick func(n int) int
}

// Herald sends the vox-casts of one invocation to the desktop, as a single
// notification replaced in place, unless the vox is "off"; its caller
// decides by the vox whether an invocation upon a terminal is heard. It never fails the invocation: a missing or fallen
// notify-send stays silent. Raise one herald per invocation and Close it
// afterwards. It is not safe for concurrent use.
type Herald struct {
	cfg     Config
	desktop desktop
}

// New raises a herald for one invocation.
func New(c Config) *Herald {
	if c.Pick == nil {
		c.Pick = rand.IntN
	}
	return &Herald{cfg: c}
}

// Proclaim composes the vox-cast of p and sends it. It satisfies
// invocation.Herald.
func (h *Herald) Proclaim(p invocation.Proclamation) { h.Send(Compose(p, h.cfg.Pick)) }

// Send sends m, composed before, to the desktop.
func (h *Herald) Send(m Message) {
	if h.cfg.Vox != librarium.VoxOff {
		h.desktop.send(m, false)
	}
}

// Close lets the herald rest after the invocation: a progress left on the
// desktop, which never expires while the rite runs, is sent once more so
// that it may expire.
func (h *Herald) Close() error {
	h.desktop.rest()
	return nil
}
