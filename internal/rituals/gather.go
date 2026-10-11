package rituals

import "context"

// maxAuspexes is how many auspexes the servitor keeps awake at once when it
// augurs many rites: an auspex mostly waits upon other machines, so many may
// wait together, but not without bound.
const maxAuspexes = 16

// augural is what the servitor learns of one rite of a gathering.
type augural struct {
	at     int    // the rite's place among the names gathered
	entry  *Entry // nil when the rite could not be read
	lament error  // why it could not be read, or a data-slate that could not be
}

// gather augurs the rites names together, keeping at most maxAuspexes
// auspexes awake at once; a rite that keeps no auspex, or whose auspex is
// forgone, waits for none. It tells settled of every rite as it settles,
// in the order they settle, one at a time on the caller's goroutine, and
// returns once every rite has settled. When ctx ends, every auspex still
// awake is slain, and none is awakened after.
func (s *Servitor) gather(ctx context.Context, names []string, forgoAuspex bool, settled func(augural)) {
	heard := make(chan augural, len(names)) // no augury waits upon the listener
	awake := make(chan struct{}, maxAuspexes)
	for at, name := range names {
		go func() {
			if s.awakens(name, forgoAuspex) {
				awake <- struct{}{}
				defer func() { <-awake }()
			}
			heard <- s.augural(ctx, at, name, forgoAuspex)
		}()
	}
	for range names {
		settled(<-heard)
	}
}

// awakens tells whether the augury of the rite name awakens its auspex.
func (s *Servitor) awakens(name string, forgoAuspex bool) bool {
	r := s.Librarium.Rites[name]
	return !forgoAuspex && r != nil && !r.Edict && r.Auspex != nil
}

// augural reads the rite name, the at-th of a gathering.
func (s *Servitor) augural(ctx context.Context, at int, name string, forgoAuspex bool) augural {
	r, err := s.augur(ctx, name, forgoAuspex)
	if err != nil {
		return augural{at: at, lament: err}
	}
	e := s.entry(name, r)
	return augural{at: at, entry: &e, lament: r.Lament}
}

// read gathers the rites names and returns what it learnt of each, in the
// order of names.
func (s *Servitor) read(ctx context.Context, names []string, forgoAuspex bool) []augural {
	read := make([]augural, len(names))
	s.gather(ctx, names, forgoAuspex, func(a augural) { read[a.at] = a })
	return read
}
