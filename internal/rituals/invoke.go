package rituals

import (
	"context"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/chronicle"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// Petition asks for one invocation.
type Petition struct {
	Rite, Aspect string
	// Runes are the inscription runes as written (--<key> value); a rune
	// written "" clears the decree.
	Runes map[string]string
	// Foresee reveals what the invocation would do and performs nothing,
	// not even the auspex.
	Foresee bool
	// Herald hears the vox-casts and the fall; nil hears nothing. The
	// caller raises it and lets it rest afterwards.
	Herald invocation.Herald
}

// Result is what came of a petition.
type Result struct {
	Rite *librarium.Rite // nil when no such rite may be invoked
	// Options are those the invocation was prepared with: the aspect, the
	// former aspect read by augury, the inscriptions resolved, those on the
	// data-slate, and the orders.
	Options invocation.Options
	// Former is how the rite stood before: when Options.Former is empty it
	// tells whether the rite lay dormant or its aspect could not be read.
	Former    augury.Standing
	Foresight []invocation.Foresight // what every step that passed the pre-flight would do
	Outcome   *invocation.Outcome    // nil when foreseen or refused
	// Laments are what went amiss around the invocation without undoing it:
	// a data-slate that could not be read or kept, a chronicle that could
	// not be written.
	Laments []error
}

// Invoke performs the petition, or foresees it. The error tells why
// nothing was performed: *Unrecorded, *Heretical, or invocation.Heresies of
// the inscriptions or the pre-flight (the result's Foresight is filled for
// the steps that passed). A performance that fell is no error: its Outcome
// tells the verdict. Only performances are kept on the data-slate and in
// the chronicle; a refused or foreseen invocation is not.
func (s *Servitor) Invoke(ctx context.Context, p Petition) (Result, error) {
	r, err := s.Rite(p.Rite)
	if err != nil {
		return Result{}, err
	}
	res := Result{Rite: r}
	inscriptions, err := invocation.ResolveInscriptions(r, p.Aspect, p.Runes)
	if err != nil {
		return res, err
	}
	env, orders := s.env(), s.Orders()
	former, lament := augury.Augur(r, augury.Options{Target: p.Aspect, ForgoAuspex: p.Foresee, Orders: orders, Env: env})
	res.lament(lament)
	res.Former = former.Standing
	res.Options = invocation.Options{Aspect: p.Aspect, Former: former.Aspect, Inscriptions: inscriptions,
		Recorded: former.Inscriptions, Orders: orders, Herald: p.Herald}
	inv, err := invocation.Prepare(r, res.Options)
	res.Foresight = inv.Foresee()
	if err != nil || p.Foresee {
		return res, err
	}
	out, err := inv.PerformContext(ctx)
	if err != nil {
		return res, err
	}
	res.Outcome = &out
	at := s.now()
	res.lament(augury.Record(env, r.Name, out, res.Options, at))
	res.lament(chronicle.Append(orders.Chronicle, chronicle.New(at, r.Name, res.Options, out)))
	return res, nil
}

func (r *Result) lament(err error) {
	if err != nil {
		r.Laments = append(r.Laments, err)
	}
}
