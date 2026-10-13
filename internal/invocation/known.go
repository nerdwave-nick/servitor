package invocation

// beholder is the step of a subject that can tell, once begun, what the
// subject was last read to show.
type beholder interface {
	// witnessed returns the subject's index among the rite's subjects and
	// what it was last read to show: the proof once set, or the proof of
	// its restoration; zero when that was not proven. begun is false when
	// the step was never begun.
	witnessed() (index int, last Reading, begun bool)
}

// known are the omens the subjects of a rite of aspects are known to show at
// the end of its performance, by subject: those whose last deed was a
// reading that showed one declared omen. That is the reading before for a
// subject unchanged or never reached, the proof of a subject set, or the
// proof of its restoration. A subject set or restored and not proven, one
// that could not be read, and a built-in subject, which shows no omen of
// its own, have none.
func (inv *Invocation) known() map[string]Reading {
	last := make([]Reading, len(inv.rite.Subjects))
	copy(last, inv.opts.Readings)
	for _, step := range inv.steps {
		if w, ok := step.(beholder); ok {
			if i, r, begun := w.witnessed(); begun {
				last[i] = r
			}
		}
	}
	known := map[string]Reading{}
	for i, r := range last {
		if r.Omen != "" && len(r.Aspects) > 0 {
			known[inv.rite.Subjects[i].Name] = r
		}
	}
	return known
}
