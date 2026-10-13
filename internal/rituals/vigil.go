package rituals

import (
	"context"

	"github.com/nerdwave-nick/servitor/internal/augury"
)

// Heralding is one word of a vigil: a rite as it stands, or a lament.
type Heralding struct {
	Entry *Entry // the rite as it stands, whole; nil for a lament
	// Reading is the augury Entry was told from, its auspex marked as
	// Entry's is; nil for a lament.
	Reading *Reading
	Rite    string // the rite a lament concerns
	Lament  error  // a data-slate or sanction that could not be read, or a forsaken sanction
}

// Vigil keeps the census as a vigil, telling it as it is learnt. tell
// first hears every rite, by name, as it stands without its auspex: a rite
// whose auspex is yet to be heard is marked Awaited, every other rite is
// told as the census reads it. Then tell hears every awaited rite once
// more, whole, in the order their auspexes settle, as the census reads it.
// A data-slate that cannot be read is lamented before its rite is first
// told, and every forsaken sanction before the first rite. forgoAuspex leaves every auspex unawakened, so every rite is told
// once.
//
// The auspexes are awakened together as the census awakens them. When ctx
// ends, tell hears nothing more, every auspex still awake is slain with its
// process group, and Vigil returns once none labours. tell is heard on the
// caller's goroutine, one heralding at a time.
func (s *Servitor) Vigil(ctx context.Context, forgoAuspex bool, tell func(Heralding)) {
	s.Attend(ctx, s.Herald(ctx, forgoAuspex, tell), tell)
}

// Herald is the first telling of a vigil: the forsaken sanctions, then
// every rite by name as it stands without its auspex, and it returns the
// rites awaited, by name, for Attend. It awakens no auspex.
func (s *Servitor) Herald(ctx context.Context, forgoAuspex bool, tell func(Heralding)) (awaited []string) {
	names := s.Librarium.Names()
	for _, f := range s.forsaken() {
		tell(Heralding{Rite: f.(*ForsakenSanction).Rite, Lament: f})
	}
	for _, a := range s.read(ctx, names, true) {
		if ctx.Err() != nil {
			return nil
		}
		if a.lament != nil {
			tell(Heralding{Rite: names[a.at], Lament: a.lament})
		}
		if a.entry == nil {
			continue
		}
		if a.entry.Auspex == augury.Forgone && !forgoAuspex {
			a.entry.Auspex, a.reading.Auspex = augury.Awaited, augury.Awaited
			awaited = append(awaited, names[a.at])
		}
		tell(Heralding{Entry: a.entry, Reading: a.reading})
	}
	return awaited
}

// Attend is the rest of a vigil: it awakens the auspexes of the rites
// awaited together, and tells each rite once more, whole, as it settles.
func (s *Servitor) Attend(ctx context.Context, awaited []string, tell func(Heralding)) {
	s.gather(ctx, awaited, false, func(a augural) {
		if a.entry != nil && ctx.Err() == nil {
			tell(Heralding{Entry: a.entry, Reading: a.reading})
		}
	})
}
