package invocation

import (
	"context"
	"maps"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// Herald hears the vox-casts of an invocation while it is performed: one
// proclamation for every vox-cast step reached, and one, unbidden, when a
// step falls and the reversions are done. Proclaim is called on the
// performing goroutine, in written order; it must not fail the invocation,
// so it returns nothing. A nil herald hears nothing.
type Herald interface {
	Proclaim(Proclamation)
}

// Watcher is a herald that also watches every real step — every step that
// is no vox-cast — as it is begun, for a vessel of the servitor that shows
// the invocation while it is performed. A step never begun, once the
// invocation is halted, is not seen.
type Watcher interface {
	Herald
	// Begin is told before the step of verse v is performed: it is the
	// step-th real step (counted from one) of the liturgy's steps.
	Begin(v Verse, step, steps int)
}

// Witness is a herald that also sees every real step once it is performed,
// before any vox-cast that follows it is proclaimed: a vessel of the
// servitor that tells the faithful, verse by verse, what was done. A step
// that falls is not seen; the fall is proclaimed.
type Witness interface {
	Herald
	// Performed is told once the step of verse v is performed: it is the
	// step-th real step (counted from one) of the liturgy's steps.
	Performed(v Verse, step, steps int)
}

// Tidings name what a proclamation tells.
type Tidings string

const (
	// Progress: a "progress" vox-cast was reached.
	Progress Tidings = librarium.VoxProgress
	// Success: a "success" vox-cast was reached.
	Success Tidings = librarium.VoxSuccess
	// Failure: a step fell; told unbidden once the reversions are done.
	Failure Tidings = "failure"
)

// Proclamation is what a herald hears.
type Proclamation struct {
	Tidings Tidings
	// Values illuminate the herald's vox-cast templates: the invocation's
	// aspect, former aspect, rite and inscriptions, and in Values.Tidings
	// the counting of real steps — those that are not vox-casts. Step is
	// how many real steps were performed (on failure: the ordinal of the
	// fallen one), Steps how many the liturgy holds; StepKind and
	// StepTarget name the real step performed last (on failure: the
	// fallen one; empty when none was), and Heresy why it fell.
	Values placeholder.Values
	// Verse is the real step described, as numbered in the liturgy; zero
	// when no real step came before a progress or success vox-cast.
	Verse Verse
	// Outcome is the invocation's outcome on failure, with its verdict and
	// every reversion; nil otherwise.
	Outcome *Outcome
}

// crier counts the real steps of a performance and tells the herald.
type crier struct {
	herald Herald
	values placeholder.Values
	steps  int   // real steps in the liturgy
	done   int   // real steps performed
	last   Verse // the real step performed last
}

// newCrier counts the real steps among steps; it is nil without a herald.
func newCrier(h Herald, v placeholder.Values, steps []performer) *crier {
	if h == nil {
		return nil
	}
	c := &crier{herald: h, values: v}
	for _, s := range steps {
		if _, vox := s.(voxCast); !vox {
			c.steps++
		}
	}
	return c
}

// performed notes that step was performed and tells a witnessing herald;
// a vox-cast is proclaimed.
func (c *crier) performed(step performer) {
	if c == nil {
		return
	}
	vox, ok := step.(voxCast)
	if !ok {
		c.done++
		c.last = step.verse()
		if w, witnessing := c.herald.(Witness); witnessing {
			w.Performed(c.last, c.done, c.steps)
		}
		return
	}
	c.proclaim(vox.tidings, c.done, c.last, "", nil)
}

// begin tells a watching herald that step is begun; vox-casts are not
// steps it watches.
func (c *crier) begin(step performer) {
	if c == nil {
		return
	}
	w, watching := c.herald.(Watcher)
	if _, vox := step.(voxCast); vox || !watching {
		return
	}
	w.Begin(step.verse(), c.done+1, c.steps)
}

// fell proclaims the failure of step with the outcome of its reversions.
func (c *crier) fell(step performer, out Outcome) {
	if c == nil {
		return
	}
	heresy := ""
	if out.Fell != nil && out.Fell.Heresy != nil {
		heresy = out.Fell.Heresy.Error()
	}
	c.proclaim(Failure, c.done+1, step.verse(), heresy, &out)
}

func (c *crier) proclaim(t Tidings, step int, v Verse, heresy string, out *Outcome) {
	values := c.values
	values.Inscriptions = maps.Clone(values.Inscriptions)
	values.Tidings = placeholder.Tidings{
		Heresy: heresy, Step: step, Steps: c.steps,
		StepKind: v.Kind.Key(), StepTarget: v.Target,
	}
	c.herald.Proclaim(Proclamation{Tidings: t, Values: values, Verse: v, Outcome: out})
}

// voxCast is a vox-cast step; performing it changes nothing, for the crier
// proclaims it.
type voxCast struct {
	v       Verse
	tidings Tidings
}

func (s voxCast) verse() Verse                { return s.v }
func (s voxCast) foresee() Foresight          { return Foresight{Verse: s.v} }
func (voxCast) perform(context.Context) error { return nil }
func (voxCast) revert() (bool, error)         { return false, nil }
