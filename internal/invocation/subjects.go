package invocation

import (
	"context"
	"fmt"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// A state rite of pattern Mark II is invoked by its subjects, not by a
// liturgy. Every subject is examined in written order against the shadow of
// the machine, as a step would be, and planned by what it was read to show
// (Options.Readings): unchanged when it already shows the omen of the
// aspect invoked, else changing. Only changing subjects are performed, in
// written order, each set and then proven: read anew and found to show the
// omen commanded. A fall reverts every subject set, the fallen one too, in
// reverse order: a built-in restores what it captured before writing, a
// machine-spirit enacts the omen it was read to show, or that of the former
// aspect, and is proven again.

// subjects examines every subject of the state rite, foreseeing each and
// making the changing ones ready to be performed.
func (inv *Invocation) subjects(p *preflight) {
	for i, sub := range p.rite.Subjects {
		p.index, p.subject, p.exalted = i, sub.Name, false
		var (
			f    *Foresight
			step performer
		)
		switch m := sub.Matter.(type) {
		case *librarium.Sanctum:
			f, step = p.builtIn(p.sanctum(m))
		case *librarium.Transcription:
			f, step = p.builtIn(p.transcription(m))
		case *librarium.Tether:
			f, step = p.builtIn(p.tether(m))
		case *librarium.MachineSpirit:
			f, step = p.spirit(i, m)
		}
		if f != nil {
			inv.foresight = append(inv.foresight, *f)
		}
		if step != nil {
			inv.steps = append(inv.steps, step)
		}
	}
	p.subject = ""
}

// builtIn plans a sanctum, transcription or tether subject by the change
// the pre-flight planned for it: it is changing when anything would change.
// It returns no foresight for a subject found heretical.
func (p *preflight) builtIn(step performer) (*Foresight, performer) {
	if step == nil {
		return nil, nil
	}
	f := step.foresee()
	changing := f.Vessel != nil && f.Vessel.Changed() || f.Tether != nil && f.Tether.Changed()
	f.Plan = &SubjectPlan{Changing: changing}
	if !changing {
		return &f, nil
	}
	return &f, &builtInSubject{performer: step, aspect: p.opts.Aspect}
}

// settler is a step that can tell, once performed, whether it still has
// anything to change.
type settler interface {
	unsettled() (bool, error)
}

// builtInSubject is a changing sanctum, transcription or tether subject: it
// is performed as its step of Mark I, then proven by planning it anew
// against the machine, which must find nothing left to change.
type builtInSubject struct {
	performer
	aspect string
}

func (s *builtInSubject) perform(ctx context.Context) error {
	if err := s.performer.perform(ctx); err != nil {
		return err
	}
	st, ok := s.performer.(settler)
	if !ok {
		return nil
	}
	unsettled, err := st.unsettled()
	switch {
	case err != nil:
		return err
	case unsettled:
		return fmt.Errorf("the subject %q does not bear the omen of the aspect %q once set: other hands "+
			"altered it as it was written", s.verse().Subject, s.aspect)
	}
	return nil
}

// unsettled plans the vessel anew against the machine.
func (s *vesselStep) unsettled() (bool, error) {
	c, err := s.plan(disk{})
	return c.Changed(), err
}

// unsettled plans the bond anew against the machine.
func (t *tethered) unsettled() (bool, error) {
	c, _, err := t.plan(disk{})
	return c.Changed(), err
}

// changed names the subjects of deeds set and proven, in written order;
// never nil.
func changed(deeds []Deed) []string {
	names := []string{}
	for _, d := range deeds {
		if d.Heresy == nil {
			names = append(names, d.Subject)
		}
	}
	return names
}
