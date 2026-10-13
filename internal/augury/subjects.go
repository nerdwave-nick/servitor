package augury

import (
	"context"
	"slices"
	"sync"

	"github.com/nerdwave-nick/servitor/internal/invocation"
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
	a.Readings = make([]invocation.Reading, len(s.rite.Subjects))
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
		case *librarium.MachineSpirit:
			if m.Auspex != nil && !m.Auspex.Exalted {
				spirits = append(spirits, i)
			}
		default:
			seen = s.builtIn(sub)
		}
		if len(seen) > 0 {
			omens[i] = subjectOmen(sub, readSpirit{}, aspectsOf(seen))
			a.Readings[i].Aspects = aspectsOf(seen)
		}
	}
	a.hearSpirits(ctx, s, opts, spirits, omens)
	for _, o := range omens {
		if o != nil {
			a.Omens = append(a.Omens, *o)
		}
	}
}

// builtIn returns every aspect a transcription or tether subject fits.
func (s *seer) builtIn(sub librarium.Subject) []sighting {
	switch m := sub.Matter.(type) {
	case *librarium.Transcription:
		return s.transcribed(m)
	case *librarium.Tether:
		return s.bound(m)
	}
	return nil
}

// invoked lets the seer render the omens of the aspect an invocation reads
// a state rite for with the invocation's own inscriptions.
func (s *seer) invoked(opts Options) {
	s.target, s.inscriptions = opts.Target, opts.Inscriptions
}

// ReadSubject reads the subject at index i of the state rite r anew, as an
// augury reads it, while it is invoked into opts.Target: what an invocation
// proves a subject it set by. An exalted auspex is never spoken here, nor a
// forgone one; a sanctum reads only the aspect its marker records.
func ReadSubject(ctx context.Context, r *librarium.Rite, i int, opts Options) invocation.Reading {
	slate, _ := ReadSlate(opts.env(), r.Name)
	s := newSeer(r, slate)
	s.invoked(opts)
	sub := r.Subjects[i]
	switch m := sub.Matter.(type) {
	case *librarium.Sanctum:
		if omen, _ := s.sanctum(i, m); omen != nil {
			return invocation.Reading{Aspects: []string{omen.Aspect}}
		}
	case *librarium.MachineSpirit:
		if m.Auspex != nil && !m.Auspex.Exalted && !opts.ForgoAuspex {
			read := s.spirit(sub, opts).hear(ctx, opts.Places)
			return invocation.Reading{Omen: read.omen, Aspects: read.aspects}
		}
	default:
		return invocation.Reading{Aspects: aspectsOf(s.builtIn(sub))}
	}
	return invocation.Reading{}
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
		a.Readings[i] = invocation.Reading{Omen: readings[n].omen, Aspects: readings[n].aspects}
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
