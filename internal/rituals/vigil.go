package rituals

import (
	"context"

	"github.com/nerdwave-nick/servitor/internal/augury"
)

// Heralding is one word of a vigil: a rite as it stands, or a lament.
type Heralding struct {
	Entry  *Entry // the rite as it stands, whole; nil for a lament
	Rite   string // the rite a lament concerns
	Lament error  // a data-slate that could not be read
}

// Vigil keeps the census as a vigil, telling it as it is learnt. tell
// first hears every rite, by name, as it stands without its auspex: a rite
// whose auspex is yet to be heard is marked Awaited, every other rite is
// told as the census reads it. Then tell hears every awaited rite once
// more, whole, in the order their auspexes settle, as the census reads it.
// A data-slate that cannot be read is lamented before its rite is first
// told. forgoAuspex leaves every auspex unawakened, so every rite is told
// once.
//
// The auspexes are awakened together as the census awakens them. When ctx
// ends, tell hears nothing more, every auspex still awake is slain with its
// process group, and Vigil returns once none labours. tell is heard on the
// caller's goroutine, one heralding at a time.
func (s *Servitor) Vigil(ctx context.Context, forgoAuspex bool, tell func(Heralding)) {
	names := s.Librarium.Names()
	var awaited []string
	for _, a := range s.read(ctx, names, true) {
		if ctx.Err() != nil {
			return
		}
		if a.lament != nil {
			tell(Heralding{Rite: names[a.at], Lament: a.lament})
		}
		if a.entry == nil {
			continue
		}
		if a.entry.Auspex == augury.Forgone && !forgoAuspex {
			a.entry.Auspex = augury.Awaited
			awaited = append(awaited, names[a.at])
		}
		tell(Heralding{Entry: a.entry})
	}
	s.gather(ctx, awaited, false, func(a augural) {
		if a.entry != nil && ctx.Err() == nil {
			tell(Heralding{Entry: a.entry})
		}
	})
}
