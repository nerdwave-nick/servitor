package rituals

import (
	"context"
	"fmt"
	"maps"
	"slices"

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

// maxSeizings bounds how often the dominion over the vessels is seized anew
// because the pre-flight revealed places not yet held (see dominions.covers).
const maxSeizings = 8

// dominions is what an invocation holds: the dominion over its rite, then
// that over every vessel and tethered name its liturgy writes. The nil
// dominions hold nothing and cover everything: foresight seizes none.
type dominions struct {
	ctx      context.Context
	env      librarium.Environment
	rite     *librarium.Rite
	await    func() // tells the awaiting, once for every seizing
	res      *Result
	held     *dominion.Dominion // over the rite
	vessels  *dominion.Dominion
	places   map[string]bool // the resolved places of the vessels sought
	seizings int
	forgone  bool // no dominion could be held: performed without
}

// holdDominion takes the dominion an invocation of r must hold before its
// augury and pre-flight: over the rite itself, then over every vessel and
// tethered name the liturgy writes as rendered with the inscriptions, each
// resolved (dominion.Resolve) and seized in sorted order. Every invocation
// holds at most one rite and seizes vessels in this one order, so no two
// invocations can each hold what the other awaits. An edict writes no vessel
// and seizes none.
//
// When another invocation holds any of it, the herald hears the awaiting
// once (invocation.Dominion), and the invocation waits until it is
// relinquished, or until ctx halts it (*Forsaken). A dominion that cannot
// be held at all, its place beneath the state home unwritable, is lamented,
// and the invocation is performed without it.
func (s *Servitor) holdDominion(ctx context.Context, r *librarium.Rite, p Petition, inscriptions map[string]string,
	res *Result) (*dominions, error) {
	told := false
	d := &dominions{ctx: ctx, env: s.env(), rite: r, res: res, places: map[string]bool{}, await: func() {
		if !told && p.Herald != nil {
			p.Herald.Proclaim(invocation.Proclamation{Tidings: invocation.Dominion, Edict: r.Edict,
				Values: placeholder.Values{Rite: r.Name, Aspect: p.Aspect}})
		}
		told = true
	}}
	held, err := dominion.Seize(ctx, []string{dominion.Rite(d.env, r.Name)}, d.await)
	if err != nil {
		return d, d.unheld(fmt.Sprintf("the rite %q", r.Name), err)
	}
	d.held = held
	if r.Edict {
		return d, nil
	}
	// The places are worked out from the scripture before the augury: the
	// former aspect is not known yet, and covers tells whether the
	// pre-flight reveals places beyond these.
	scouted, _ := invocation.Prepare(r, invocation.Options{Aspect: p.Aspect, Inscriptions: inscriptions})
	if err := d.seize(scouted.Foresee()); err != nil {
		d.relinquish()
		return nil, err
	}
	return d, nil
}

// unheld is what becomes of an invocation whose dominion over what could
// not be seized: Forsaken when it was halted, else a lament and an
// invocation performed without dominion.
func (d *dominions) unheld(what string, err error) error {
	if d.ctx.Err() != nil {
		return &Forsaken{Rite: d.rite.Name, Edict: d.rite.Edict, Err: err}
	}
	d.forgone = true
	d.res.lament(fmt.Errorf("no dominion could be held over %s, so it is performed without: %w", what, err))
	return nil
}

// seize seizes, anew, the dominion over the places held so far and those
// the foresight writes, in sorted order of their resolved paths.
func (d *dominions) seize(foresight []invocation.Foresight) error {
	if d.vessels != nil {
		d.vessels.Relinquish()
		d.vessels = nil
	}
	for _, place := range written(foresight) {
		d.places[place] = true
	}
	d.seizings++
	paths := slices.Sorted(maps.Keys(d.places))
	sought := make([]string, len(paths))
	for i, path := range paths {
		sought[i] = dominion.Vessel(d.env, path)
	}
	vessels, err := dominion.Seize(d.ctx, sought, d.await)
	if err != nil {
		return d.unheld(fmt.Sprintf("the vessels of the rite %q", d.rite.Name), err)
	}
	d.vessels = vessels
	return nil
}

// covers reports whether every place the foresight writes is held. When it
// is not, the dominion over the vessels must be seized anew: the pre-flight
// revealed a place the scripture alone did not (a vessel named by the former
// aspect, or reached through a tether bound meanwhile). After maxSeizings,
// the places left unheld are lamented and the invocation proceeds.
func (d *dominions) covers(foresight []invocation.Foresight) bool {
	if d == nil || d.forgone {
		return true
	}
	for _, place := range written(foresight) {
		if d.places[place] {
			continue
		}
		if d.seizings < maxSeizings {
			return false
		}
		d.res.lament(fmt.Errorf("the vessels of the rite %q shifted with every seizing of their dominion, so it "+
			"is performed without dominion over %s", d.rite.Name, place))
	}
	return true
}

// relinquish lets go of the vessels, then the rite.
func (d *dominions) relinquish() {
	if d == nil {
		return
	}
	if d.vessels != nil {
		d.vessels.Relinquish()
	}
	if d.held != nil {
		d.held.Relinquish()
	}
}

// written are the resolved places the foresight writes: the vessels of
// sanctums and transcriptions, their tethers already followed, and the
// names tethers bind.
func written(foresight []invocation.Foresight) []string {
	var out []string
	for _, f := range foresight {
		switch {
		case f.Vessel != nil:
			out = append(out, dominion.Resolve(f.Vessel.Path))
		case f.Tether != nil:
			out = append(out, dominion.Resolve(f.Tether.Name))
		}
	}
	return out
}
