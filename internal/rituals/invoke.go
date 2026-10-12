package rituals

import (
	"context"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/chronicle"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
	"github.com/nerdwave-nick/servitor/internal/sanction"
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
	// liturgy is about to be performed (the exalted servitor raised, for a
	// rite with exalted verses), with the result so far: the rite, the
	// options and how the rite stood before. It is never told of an
	// invocation refused or foreseen.
	Commence func(Result)
	// Exaltation is how the exalted servitor is raised to perform the
	// exalted verses of the rite; nil refuses a rite holding any. The
	// command line lends one; the cogitator lends none yet.
	Exaltation *Exaltation
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
	// Sanction is how the plan of the exalted verses for the aspect stands
	// towards their sanction, and Sigil the whole sigil of that plan; both
	// are empty for a rite without exalted verses, and when the plan
	// cannot be drawn. Intercessor names the intercessor that vouches for
	// the master of a performance with exalted verses.
	Sanction    sanction.Standing
	Sigil       string
	Intercessor string
}

// Invoke performs the petition, or foresees it. The error tells why
// nothing was performed: *Unrecorded, *Heretical, invocation.Heresies of
// the inscriptions or the pre-flight (the result's Foresight is filled for
// the steps that passed) or of exalted verses that may not be performed
// (see exaltation), *Forsaken, or a refusal before any exalted verse: no
// dominion held, no intercessor found, *Dismissed, *Spurned or
// *exalted.Mismatch. A performance that fell is no error: its Outcome tells
// the verdict. Only performances are kept on the data-slate and in the
// chronicle; a refused or foreseen invocation is not.
//
// The exalted verses of a rite are never performed by the invoker's
// servitor: after the pre-flight, one exalted servitor is raised through
// the intercessor (one proof of the master), the herald hearing the
// tidings invocation.Exaltation before, and performs and reverts them in
// their turn among the common verses, from the sanction of their master.
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
	exalting, err := s.exaltation(r, p, &res)
	if err != nil {
		res.Options = invocation.Options{Aspect: p.Aspect, Edict: r.Edict}
		return res, err
	}
	inscriptions, err := invocation.ResolveInscriptions(r, p.Aspect, p.Runes)
	if err != nil {
		return res, err
	}
	if exalting != nil {
		if hs := unspeakable(r, inscriptions); len(hs) > 0 {
			return res, hs
		}
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
	if err == nil {
		err = held.amiss()
	}
	if err != nil || p.Foresee {
		return res, err
	}
	var (
		x invocation.Exalter
		e *exalter
	)
	if exalting != nil {
		if e, err = s.exalt(r, p, &res, exalting); err != nil {
			return res, err
		}
		x = e
	}
	if p.Commence != nil {
		p.Commence(res)
	}
	out, err := inv.PerformExalted(ctx, x)
	if e != nil {
		if err != nil {
			e.abandon()
		}
		res.Laments = append(res.Laments, e.laments...)
	}
	if err != nil {
		return res, err
	}
	res.Outcome = &out
	at := s.now()
	res.lament(augury.Record(s.env(), r.Name, out, res.Options, at))
	entry := chronicle.New(at, r.Name, res.Options, out)
	if exalting != nil {
		entry.Intercessor, entry.Sigil = exalting.intercessor.Name, short(exalting.sigil)
	}
	res.Entry = &entry
	res.lament(chronicle.Append(res.Options.Orders.Chronicle, entry))
	return res, nil
}

// exalt tells the herald that the exalted verses await the master's proof
// and raises the exalted servitor through the intercessor.
func (s *Servitor) exalt(r *librarium.Rite, p Petition, res *Result, x *exaltedPlan) (*exalter, error) {
	if p.Herald != nil {
		p.Herald.Proclaim(invocation.Proclamation{Tidings: invocation.Exaltation, Edict: r.Edict,
			Values: placeholder.Values{Rite: r.Name, Aspect: res.Options.Aspect}, Intercessor: x.intercessor.Name,
			Sigil: short(x.sigil)})
	}
	return s.raise(r, res, x, *p.Exaltation)
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
