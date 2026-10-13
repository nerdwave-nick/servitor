package augury

import (
	"context"
	"slices"
	"sync"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// readSubjects reads the omen of every subject of a state rite of pattern
// Mark II, in written order. A sanctum's marker names one aspect; a
// transcription and a tether fit every aspect whose omen they match; a
// machine-spirit fits what its auspex reads. The auspexes of every subject
// labour at once, unless forgone; an exalted auspex is never spoken, and
// a machine-spirit without an auspex cannot be read. A subject that fits
// no aspect yields no omen.
func (a *Augury) readSubjects(ctx context.Context, s *seer, opts Options) {
	omens := make([]*Omen, len(s.rite.Subjects))
	var spirits []int // the subjects whose auspex is to be heard
	for i, sub := range s.rite.Subjects {
		var seen []sighting
		switch m := sub.Matter.(type) {
		case *librarium.Sanctum:
			omen, taint := s.sanctum(i, m)
			if omen != nil {
				seen = []sighting{{aspect: omen.Aspect}}
			}
			if taint != nil {
				// a state rite has no verses: the taint keeps only its vessel
				taint.Number, taint.Subject = 0, sub.Name
				a.Taint = append(a.Taint, *taint)
			}
		case *librarium.Transcription:
			seen = s.transcribed(m)
		case *librarium.Tether:
			seen = s.bound(m)
		case *librarium.MachineSpirit:
			if m.Auspex != nil && !m.Auspex.Exalted {
				spirits = append(spirits, i)
			}
		}
		if len(seen) > 0 {
			omens[i] = subjectOmen(sub, readSpirit{}, aspectsOf(seen))
		}
	}
	a.hearSpirits(ctx, s, opts, spirits, omens)
	for _, o := range omens {
		if o != nil {
			a.Omens = append(a.Omens, *o)
		}
	}
}

// hearSpirits awakens the auspexes of the machine-spirits at the indices
// spirits together, and keeps the omen of each that fits an aspect at its
// index of omens. The heed tells of them alone: none, forgone, mute when
// any fitted no aspect, else heeded.
func (a *Augury) hearSpirits(ctx context.Context, s *seer, opts Options, spirits []int, omens []*Omen) {
	switch {
	case len(spirits) == 0:
		return
	case opts.ForgoAuspex:
		a.Auspex = Forgone
		return
	}
	readings := make([]readSpirit, len(spirits))
	var wg sync.WaitGroup
	for n, i := range spirits {
		hearing := s.spirit(s.rite.Subjects[i], opts) // prepared here: the seer is not shared
		wg.Go(func() { readings[n] = hearing.hear(ctx, opts.Places) })
	}
	wg.Wait()
	a.Auspex = Heeded
	for n, i := range spirits {
		if len(readings[n].aspects) == 0 {
			a.Auspex = Mute
			continue
		}
		omens[i] = subjectOmen(s.rite.Subjects[i], readings[n], readings[n].aspects)
	}
}

func subjectOmen(sub librarium.Subject, r readSpirit, aspects []string) *Omen {
	o := &Omen{Subject: sub.Name, Designation: sub.Designation(), Read: r.omen, Aspects: aspects}
	if len(aspects) == 1 {
		o.Aspect = aspects[0]
	}
	return o
}

func aspectsOf(seen []sighting) []string {
	aspects := make([]string, len(seen))
	for i, s := range seen {
		aspects[i] = s.aspect
	}
	return aspects
}

// intersect names the aspect and standing of a state rite: its fitting
// aspects are those every omen fits. None corrupts it; one names its
// aspect, desecrated when the data-slate recorded another; several name
// the data-slate's aspect when it is among them, else none, the rite
// desecrated when the data-slate recorded another and dormant when it
// recorded none. Without omen the data-slate names the aspect, as for
// Mark I; taint and a faltered last invocation weigh as for Mark I.
func (a *Augury) intersect(slate *Slate) {
	a.Fitting = []string{}
	if len(a.Omens) == 0 {
		a.judge(slate)
		return
	}
	fitting := a.Omens[0].Aspects
	for _, o := range a.Omens[1:] {
		fitting = slices.DeleteFunc(slices.Clone(fitting), func(aspect string) bool {
			return !slices.Contains(o.Aspects, aspect)
		})
	}
	recorded := ""
	if slate != nil {
		recorded = slate.Aspect
	}
	switch {
	case len(fitting) == 0:
		a.Standing = Corrupted
	case len(fitting) == 1, !slices.Contains(fitting, recorded):
		a.Fitting, a.Standing = fitting, Dormant
		if len(fitting) == 1 {
			a.Aspect, a.Standing = fitting[0], Performed
		}
		if recorded != "" && recorded != a.Aspect {
			a.Desecrated, a.Former, a.Standing = true, recorded, Desecrated
		}
	default:
		a.Fitting, a.Aspect, a.Standing = fitting, recorded, Performed
	}
	a.weigh(slate)
}
