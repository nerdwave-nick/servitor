package rituals

import (
	"context"
	"fmt"

	"github.com/nerdwave-nick/servitor/internal/dominion"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// Forsaken is the lament of an invocation halted while it awaited
// dominion: not one verse was performed, and nothing was touched.
type Forsaken struct {
	Rite  string
	Edict bool
	Err   error // why the waiting ended: the halting of the invocation
}

func (e *Forsaken) Error() string {
	what := fmt.Sprintf("the invocation of the rite %q", e.Rite)
	if e.Edict {
		what = fmt.Sprintf("the proclaiming of the edict %q", e.Rite)
	}
	return what + " was halted while it awaited dominion; not one verse was performed, and nothing was touched"
}

func (e *Forsaken) Unwrap() error { return e.Err }

// holdDominion takes the dominion an invocation of r must hold before its
// pre-flight: over the rite itself. When another invocation holds it, the
// herald hears the awaiting once (invocation.Dominion), and the invocation
// waits until it is relinquished, or until ctx halts it (*Forsaken).
//
// A dominion that cannot be held at all, its place beneath the state home
// unwritable, is lamented, and the invocation is performed without it.
func (s *Servitor) holdDominion(ctx context.Context, r *librarium.Rite, p Petition, res *Result) (relinquish func(), err error) {
	places := []string{dominion.Rite(s.env(), r.Name)}
	d, err := dominion.Seize(ctx, places, func() {
		if p.Herald != nil {
			p.Herald.Proclaim(invocation.Proclamation{Tidings: invocation.Dominion, Edict: r.Edict,
				Values: placeholder.Values{Rite: r.Name, Aspect: p.Aspect}})
		}
	})
	switch {
	case err == nil:
		return d.Relinquish, nil
	case ctx.Err() != nil:
		return nil, &Forsaken{Rite: r.Name, Edict: r.Edict, Err: err}
	}
	res.lament(fmt.Errorf("no dominion could be held over the rite %q, so it is performed without: %w", r.Name, err))
	return func() {}, nil
}
