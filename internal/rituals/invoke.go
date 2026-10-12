package rituals

import (
	"context"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/chronicle"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// Petition asks for one invocation: of a rite into Aspect, or of an edict,
// which takes no aspect (Aspect "").
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
	// Commence, when not nil, is told once the pre-flight has passed and the
	// liturgy is about to be performed, with the result so far: the rite,
	// the options and how the rite stood before. It is never told of an
	// invocation refused or foreseen.
	Commence func(Result)
}

// Result is what came of a petition.
type Result struct {
	Rite *librarium.Rite // nil when no such rite may be invoked
	// Options are those the invocation was prepared with: the aspect, the
	// former aspect read by augury (neither for an edict), the inscriptions
	// resolved, those on the data-slate, and the orders.
	Options invocation.Options
	// Former is how the rite stood before: when Options.Former is empty it
	// tells whether the rite lay dormant or its aspect could not be read.
	// It is empty for an edict, whose aspect is never read.
	Former    augury.Standing
	Foresight []invocation.Foresight // what every step that passed the pre-flight would do
	Outcome   *invocation.Outcome    // nil when foreseen or refused
	// Entry is the line this performance added to the chronicle, nil when
	// foreseen or refused. It is borne even when the chronicle could not be
	// written; that is lamented.
	Entry *chronicle.Entry
	// Laments are what went amiss around the invocation without undoing it:
	// a data-slate that could not be read or kept, a chronicle that could
	// not be written.
	Laments []error
}

// Invoke performs the petition, or foresees it. The error tells why
// nothing was performed: *Unrecorded, *Heretical, invocation.Heresies of
// the inscriptions or the pre-flight (the result's Foresight is filled for
// the steps that passed) or of the exalted verses of a rite, which are
// never performed (see unexalted), or *Forsaken. A performance that fell is no
// error: its Outcome tells the verdict. Only performances are kept on the
// data-slate and in the chronicle; a refused or foreseen invocation is not.
//
// An invocation that is not foreseen is held Steadfast throughout: the
// departure of the one who invoked it neither halts it nor leaves it half
// performed. It holds dominion over its rite and over every vessel and
// tethered name it writes from before the augury of its former aspect and
// its pre-flight until its verdict is kept, awaiting another invocation that
// holds any of it (see holdDominion); foresight holds and awaits none.
func (s *Servitor) Invoke(ctx context.Context, p Petition) (Result, error) {
	if !p.Foresee {
		defer Steadfast()()
	}
	r, err := s.Rite(p.Rite)
	if err != nil {
		return Result{}, err
	}
	res := Result{Rite: r}
	if !p.Foresee {
		if err := unexalted(r); err != nil {
			res.Options = invocation.Options{Aspect: p.Aspect, Edict: r.Edict}
			return res, err
		}
	}
	inscriptions, err := invocation.ResolveInscriptions(r, p.Aspect, p.Runes)
	if err != nil {
		return res, err
	}
	var held *dominions
	if !p.Foresee {
		held, err = s.holdDominion(ctx, r, p, inscriptions, &res)
		if err != nil {
			return res, err
		}
		defer held.relinquish()
	}
	laments := len(res.Laments)
	inv, err := s.prepare(r, p, inscriptions, &res)
	for err == nil && !held.covers(res.Foresight) {
		// The pre-flight revealed places beyond those held: they are
		// seized, and what was read without dominion over them is read anew.
		res.Laments = res.Laments[:laments]
		if err := held.seize(res.Foresight); err != nil {
			return res, err
		}
		inv, err = s.prepare(r, p, inscriptions, &res)
	}
	if err != nil || p.Foresee {
		return res, err
	}
	if p.Commence != nil {
		p.Commence(res)
	}
	out, err := inv.PerformContext(ctx)
	if err != nil {
		return res, err
	}
	res.Outcome = &out
	at := s.now()
	res.lament(augury.Record(s.env(), r.Name, out, res.Options, at))
	entry := chronicle.New(at, r.Name, res.Options, out)
	res.Entry = &entry
	res.lament(chronicle.Append(res.Options.Orders.Chronicle, entry))
	return res, nil
}

// prepare reads how the rite stands, or for an edict the inscriptions on
// its data-slate, and runs the pre-flight, filling res's Options, Former and
// Foresight.
func (s *Servitor) prepare(r *librarium.Rite, p Petition, inscriptions map[string]string,
	res *Result) (*invocation.Invocation, error) {
	env, orders := s.env(), s.Orders()
	res.Options = invocation.Options{Aspect: p.Aspect, Edict: r.Edict, Inscriptions: inscriptions, Orders: orders,
		Herald: p.Herald}
	if r.Edict {
		// An edict stands in no aspect to be read: only the inscriptions
		// on its data-slate are wanted.
		slate, lament := augury.ReadSlate(env, r.Name)
		res.lament(lament)
		if slate != nil {
			res.Options.Recorded = slate.Inscriptions
		}
	} else {
		former, lament := augury.Augur(r, augury.Options{Target: p.Aspect, ForgoAuspex: p.Foresee, Orders: orders, Env: env})
		res.lament(lament)
		res.Former = former.Standing
		res.Options.Former, res.Options.Recorded = former.Aspect, former.Inscriptions
	}
	inv, err := invocation.Prepare(r, res.Options)
	res.Foresight = inv.Foresee()
	return inv, err
}

func (r *Result) lament(err error) {
	if err != nil {
		r.Laments = append(r.Laments, err)
	}
}
