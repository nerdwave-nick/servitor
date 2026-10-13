package rituals

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/exalted"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// behold bids the exalted servitor e read the subjects of the state rite r
// the invoker cannot read (unread), and prepares the invocation anew with
// what it told, so that its plan is whole before the first item: each
// reading is marked as the exalted servitor's, at the moment it was told.
// A privileged machine-spirit fits the aspects whose omens it was told to
// show; a hidden sanctum, transcription or tether fits the aspect invoked
// when it stands as that aspect demands. The privileged subjects the
// invocation sets are then proven by what the exalted servitor told of
// them (see exalter.reader).
func (s *Servitor) behold(r *librarium.Rite, p Petition, inscriptions map[string]string, res *Result, e *exalter,
	unread []string) (*invocation.Invocation, error) {
	sights, err := e.read(unread)
	if err != nil {
		return nil, fmt.Errorf("%s is refused, and nothing was touched: the subjects only the exalted servitor reads "+
			"were not read: %w", e.deed, err)
	}
	opts, at := s.auguryOf(r, p, inscriptions), s.now()
	readings := make([]invocation.Reading, len(r.Subjects))
	copy(readings, res.Options.Readings)
	for _, sight := range sights {
		i := slices.IndexFunc(r.Subjects, func(sub librarium.Subject) bool { return sub.Name == sight.Subject })
		var read invocation.Reading
		switch {
		case r.Subjects[i].Designation() == librarium.DesignationMachineSpirit:
			read = augury.Fitting(r, i, opts, sight.Omens)
		case sight.Unchanged:
			read.Aspects = []string{p.Aspect}
		}
		read.At, read.Exalted = at, true
		readings[i] = read
	}
	res.Options.Readings = readings
	res.Options.Reader = e.reader(r, opts, res.Options.Reader, s.now)
	inv, err := invocation.Prepare(r, res.Options)
	res.Foresight = inv.Foresee()
	return inv, err
}

// read bids the exalted servitor read the subjects named, and hears what
// each shows: one sight for each, in the order named.
func (e *exalter) read(names []string) ([]exalted.Sight, error) {
	r, err := e.ask(exalted.Request{Read: names}, "read", exalted.Item{})
	if err != nil {
		return nil, err
	}
	told := make([]string, len(r.Sights))
	for i, sight := range r.Sights {
		told[i] = sight.Subject
	}
	if r.Verdict != exalted.Beheld || !slices.Equal(told, names) {
		e.lose(fmt.Errorf("%w: the exalted servitor answered the read of %q with %q, telling of %q", errLost, names,
			r.Verdict, told))
		return nil, e.lost
	}
	return r.Sights, nil
}

// reader reads the subjects of r anew to prove them: a privileged
// machine-spirit by the omens the exalted servitor told it showed once it
// set or restored it, which proved it, marked as its reading at the moment
// now tells; none when it told none. Every other subject is read by read.
func (e *exalter) reader(r *librarium.Rite, opts augury.Options, read invocation.Reader,
	now func() time.Time) invocation.Reader {
	return func(ctx context.Context, i int) invocation.Reading {
		ms, ok := r.Subjects[i].Matter.(*librarium.MachineSpirit)
		if !ok || ms.Auspex == nil || !ms.Auspex.Exalted {
			return read(ctx, i)
		}
		name := r.Subjects[i].Name
		omens, proven := e.proofs[name]
		delete(e.proofs, name)
		if !proven {
			return invocation.Reading{}
		}
		got := augury.Fitting(r, i, opts, omens)
		got.At, got.Exalted = now(), true
		return got
	}
}
