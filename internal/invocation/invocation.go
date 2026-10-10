// Package invocation performs a rite of pattern Mark I into one aspect.
//
// Prepare runs the pre-flight: it renders every placeholder, reads every
// tome afresh, and examines every vessel and tether, without changing
// anything. It plans the steps one after another against a shadow of the
// machine, so a step sees what the steps before it will have done. Foresee
// tells what the invocation would do; Perform does it, step by step in
// written order, and when step k falls it reverts steps k … 1 in reverse
// order and reports every reversion.
//
// Relative vessels, tethers, anchors and tomes resolve against the
// directory of the rite's scripture (librarium.Rite.ResolvePath).
//
// Steps of every kind join the same ordered loop through the performer
// interface; incantations and litanies are not yet spoken, and vox-casts are
// not yet sent.
package invocation

import (
	"errors"
	"fmt"
	"maps"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// Options are what one invocation is performed with. The caller resolves
// them; the invocation reads no settings, runes or data-slates itself.
type Options struct {
	Aspect       string            // the aspect to invoke
	Former       string            // the aspect before; "" when unknown
	Inscriptions map[string]string // the inscriptions of this invocation, already resolved
	// Recorded are the inscriptions on the data-slate, with which earlier
	// invocations illuminated their scripture; a transcription recognises
	// its own vessel by them too.
	Recorded map[string]string
}

// Verse names one step of the liturgy: its place, its kind and what its own
// key holds, rendered and resolved.
type Verse struct {
	Number int // counted from one
	Kind   librarium.Kind
	Target string
}

// performer is a step made ready by the pre-flight.
type performer interface {
	// verse names the step.
	verse() Verse
	// foresee tells what the step will do, as planned in the pre-flight.
	foresee() Foresight
	// perform does the step, capturing what it changes.
	perform() error
	// revert undoes what perform changed; reverted is false when there
	// was nothing to undo.
	revert() (reverted bool, err error)
}

// Invocation is a rite made ready to be performed into one aspect.
type Invocation struct {
	rite      *librarium.Rite
	steps     []performer
	foresight []Foresight
	heresies  Heresies
	performed bool
}

// Fall is the step that failed and why.
type Fall struct {
	Verse
	Heresy error
}

// Reversion is the outcome of undoing one step; Heresy is nil when it
// triumphed.
type Reversion struct {
	Verse
	Heresy error
}

// Outcome is how a performance ended.
type Outcome struct {
	Fell       *Fall       // nil when every step was performed
	Reversions []Reversion // in the order performed: from the fallen step back to the first
}

// Prepare runs the pre-flight of rite r for opts. The returned invocation
// can always be foreseen; when the error (of type Heresies) is not nil it
// cannot be performed. r must not be heretical.
func Prepare(r *librarium.Rite, opts Options) (*Invocation, error) {
	inv := &Invocation{rite: r}
	if !r.HasAspect(opts.Aspect) {
		inv.heresies = Heresies{{Finding: librarium.Finding{
			Severity: librarium.Heresy, Scripture: r.Path, Rite: r.Name, Position: r.Locate("/aspects"),
			Message: fmt.Sprintf("the rite %q knows no aspect %q; it may be invoked only into %s",
				r.Name, opts.Aspect, quoteAll(r.Aspects)),
		}}}
		return inv, inv.heresies
	}
	p := &preflight{rite: r, opts: opts, world: newShadow(), values: values(r, opts, opts.Inscriptions)}
	for i, s := range r.Liturgy {
		p.index = i
		if step := p.prepare(s); step != nil {
			inv.steps = append(inv.steps, step)
			inv.foresight = append(inv.foresight, step.foresee())
		}
	}
	inv.heresies = p.heresies
	if len(inv.heresies) > 0 {
		return inv, inv.heresies
	}
	return inv, nil
}

// values are the placeholder values of an invocation of r.
func values(r *librarium.Rite, opts Options, inscriptions map[string]string) placeholder.Values {
	return placeholder.Values{Aspect: opts.Aspect, Former: opts.Former, Rite: r.Name, Inscriptions: maps.Clone(inscriptions)}
}

// Foresee tells what every step that passed the pre-flight would do, in
// written order. Nothing is touched.
func (inv *Invocation) Foresee() []Foresight { return append([]Foresight{}, inv.foresight...) }

// Heresies returns what the pre-flight found.
func (inv *Invocation) Heresies() Heresies { return append(Heresies{}, inv.heresies...) }

// ErrPerformed is returned when an invocation is performed a second time.
var ErrPerformed = errors.New("this invocation has already been performed; prepare it anew")

// Perform performs the steps in written order. When step k falls, steps
// k … 1 are reverted in reverse order — the fallen step too, since it may
// have done part of its work — and a failing reversion does not stop the
// others. The error is not nil, and nothing is performed, when the
// pre-flight found heresy or the invocation was performed before.
func (inv *Invocation) Perform() (Outcome, error) {
	if len(inv.heresies) > 0 {
		return Outcome{}, inv.heresies
	}
	if inv.performed {
		return Outcome{}, ErrPerformed
	}
	inv.performed = true
	return performAll(inv.steps), nil
}

func performAll(steps []performer) Outcome {
	var out Outcome
	for k, step := range steps {
		err := step.perform()
		if err == nil {
			continue
		}
		out.Fell = &Fall{Verse: step.verse(), Heresy: err}
		for j := k; j >= 0; j-- {
			if reverted, err := steps[j].revert(); reverted || err != nil {
				out.Reversions = append(out.Reversions, Reversion{Verse: steps[j].verse(), Heresy: err})
			}
		}
		break
	}
	return out
}

func quoteAll(names []string) string {
	out := ""
	for i, n := range names {
		switch {
		case i == 0:
		case i == len(names)-1:
			out += " or "
		default:
			out += ", "
		}
		out += fmt.Sprintf("%q", n)
	}
	return out
}
