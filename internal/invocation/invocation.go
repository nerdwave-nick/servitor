// Package invocation performs a rite of pattern Mark I, or a state rite of
// pattern Mark II, into one aspect, or proclaims an edict, which bears none,
// of pattern Mark I or Mark II. A state rite is performed by its subjects
// rather than a liturgy (see subjects.go): only those that do not already
// show the omen of the aspect are set, each proven by a reading afresh.
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
// interface. Incantations and litanies are spoken by real processes (see
// speak.go); vox-casts and failures are proclaimed to the Options' Herald
// (see herald.go). An edict of pattern Mark II bears its vox-casts as a key
// of the rite, not as steps: progress after each verse, success at the end.
package invocation

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// Options are what one invocation is performed with. The caller resolves
// them; the invocation reads no settings, runes or data-slates itself.
type Options struct {
	Aspect string // the aspect to invoke; "" for an edict
	Former string // the aspect before; "" when unknown, and for an edict
	// Edict tells that the invocation proclaims an edict, which bears no
	// aspect: the data-slate and the chronicle record it so. It must agree
	// with the rite's own mark (librarium.Rite.Edict).
	Edict        bool
	Inscriptions map[string]string // the inscriptions of this invocation, already resolved
	// Recorded are the inscriptions on the data-slate, with which earlier
	// invocations illuminated their scripture; a transcription recognises
	// its own vessel by them too.
	Recorded map[string]string
	// Orders are the settings resolved; they give incantations and litanies
	// the tongue and patience their step and rite do not name. Zero orders
	// speak bash with librarium.DefaultPatience.
	Orders librarium.Orders
	// Herald hears the vox-casts and the failure of the performance; nil
	// hears nothing.
	Herald Herald
	// Readings are what every subject of a state rite of pattern Mark II
	// was read to show before the invocation, in written order; a subject
	// beyond them, or a machine-spirit without an auspex, was not read.
	Readings []Reading
	// Reader reads one subject of the state rite anew (its index in the
	// rite's subjects) once it is set, to prove it; nil proves no
	// machine-spirit, and each then falls.
	Reader Reader
}

// Reading is what one subject of a state rite was read to show.
type Reading struct {
	// Omen is the one omen a machine-spirit was read to show: what its scan
	// printed, even when that is no omen declared, or the one omen whose
	// trial succeeded; "" when none, or several.
	Omen string
	// Aspects are the aspects whose omen it shows, in the order of the
	// rite's; none when it is mute, or was not read.
	Aspects []string
	// At is when it was read; zero when the reader did not tell.
	At time.Time
	// Exalted marks a reading the exalted servitor made, the invoker
	// being unable to read the subject itself.
	Exalted bool
}

// Reader reads the subject at index i of a state rite's subjects anew,
// while it is invoked.
type Reader func(ctx context.Context, i int) Reading

// Verse names one step of the liturgy: its place, its kind and what its own
// key holds, rendered and resolved.
type Verse struct {
	Number int // counted from one
	Kind   librarium.Kind
	Target string
	// Exalted tells that the step is exalted: performed by the exalted
	// servitor alone, from the sanction of its master.
	Exalted bool
	// Subject names the subject of a state rite of pattern Mark II the
	// verse sets; Number is then zero. A built-in subject keeps its Kind
	// and Target, a machine-spirit bears neither.
	Subject string
}

// performer is a step made ready by the pre-flight.
type performer interface {
	// verse names the step.
	verse() Verse
	// foresee tells what the step will do, as planned in the pre-flight.
	foresee() Foresight
	// perform does the step, capturing what it changes.
	perform(ctx context.Context) error
	// revert undoes what perform changed; reverted is false when there
	// was nothing to undo.
	revert() (reverted bool, err error)
}

// Invocation is a rite made ready to be performed into one aspect.
type Invocation struct {
	rite      *librarium.Rite
	opts      Options
	steps     []performer
	foresight []Foresight
	heresies  Heresies
	performed bool
}

// Prepare runs the pre-flight of rite r for opts. The returned invocation
// can always be foreseen; when the error (of type Heresies) is not nil it
// cannot be performed. r must not be heretical.
func Prepare(r *librarium.Rite, opts Options) (*Invocation, error) {
	inv := &Invocation{rite: r, opts: opts}
	if h := unfit(r, opts); h != nil {
		inv.heresies = Heresies{*h}
		return inv, inv.heresies
	}
	p := &preflight{rite: r, opts: opts, world: newShadow(), values: values(r, opts, opts.Inscriptions)}
	p.heresies = unmet(r, opts.Aspect, opts.Inscriptions)
	if r.MarkII && !r.Edict {
		inv.subjects(p)
	}
	for i, s := range r.Liturgy {
		p.index, p.exalted = i, librarium.IsExalted(s)
		if step := p.prepare(s); step != nil {
			if p.exalted {
				step = &exaltedStep{planned: step}
			}
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

// unfit denounces options that do not fit r: an aspect r does not know, or
// any aspect at all for an edict, which is proclaimed by its name alone.
func unfit(r *librarium.Rite, opts Options) *Heresy {
	h := func(ptr, message string) *Heresy {
		return &Heresy{Finding: librarium.Finding{
			Severity: librarium.Heresy, Scripture: r.Path, Rite: r.Name, Position: r.Locate(ptr), Message: message,
		}}
	}
	switch {
	case r.Edict && (!opts.Edict || opts.Aspect != "" || opts.Former != ""):
		return h("/edict", fmt.Sprintf("the edict %q bears no aspect and is proclaimed whole, by its name alone; "+
			"no aspect may be invoked upon it", r.Name))
	case !r.Edict && (opts.Edict || !r.HasAspect(opts.Aspect)):
		return h("/aspects", fmt.Sprintf("the rite %q knows no aspect %q; it may be invoked only into %s",
			r.Name, opts.Aspect, quoteAll(r.Aspects)))
	}
	return nil
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
func (inv *Invocation) Perform() (Outcome, error) { return inv.PerformContext(context.Background()) }

// PerformContext is Perform, halted when ctx is done: the incantation or
// litany being spoken is slain with its whole process group and falls, no
// further step is begun, and the steps performed are reverted as after any
// fall. Reversions are never halted; each labours within its own patience.
// An invocation holding exalted verses is refused (see PerformExalted).
func (inv *Invocation) PerformContext(ctx context.Context) (Outcome, error) {
	return inv.PerformExalted(ctx, nil)
}

// utterer is a performer whose commands utter words worth keeping.
type utterer interface {
	uttered() (said, unsaid string) // what perform and revert uttered
}

func uttered(step performer) (said, unsaid string) {
	if u, ok := step.(utterer); ok {
		return u.uttered()
	}
	return "", ""
}

// performAll performs steps in order, telling c (which may be nil) of
// every step performed and of the fall, and then finishes the work of the
// exalted servitor (finish, nil without one). A finish unheard after a
// triumph is a fall at the last exalted step: every step is reverted. Only
// then is the triumph told.
func performAll(ctx context.Context, steps []performer, c *crier, finish func() error) Outcome {
	out := Outcome{Verdict: Triumph}
	for k, step := range steps {
		begun := ctx.Err() == nil
		err := errHalted
		if begun {
			c.begin(step)
			err = step.perform(ctx)
		}
		said, _ := uttered(step)
		out.Deeds = append(out.Deeds, Deed{Verse: step.verse(), Output: said, Heresy: err})
		if err == nil {
			c.performed(step)
			continue
		}
		out.Fell = &Fall{Verse: step.verse(), Heresy: err, Output: said}
		from := k
		if !begun {
			from = k - 1 // a step never begun has nothing to revert
		}
		revertFrom(&out, steps, from)
		c.fell(step, out)
		break
	}
	if finish != nil {
		err := finish()
		if last := lastExalted(steps); err != nil && out.Fell == nil && last >= 0 {
			out.Fell = &Fall{Verse: steps[last].verse(), Heresy: fmt.Errorf("the exalted servitor never heard "+
				"the finish, and reverts by itself what it performed: %w", err)}
			revertFrom(&out, steps, len(steps)-1)
			c.fell(steps[last], out)
		}
	}
	c.triumphed(out)
	return out
}

// revertFrom reverts steps from … 0, in reverse order, into out: a
// failing reversion does not stop the others.
func revertFrom(out *Outcome, steps []performer, from int) {
	out.Verdict = Reverted
	for j := from; j >= 0; j-- {
		reverted, err := steps[j].revert()
		if !reverted && err == nil {
			continue
		}
		_, unsaid := uttered(steps[j])
		out.Reversions = append(out.Reversions, Reversion{Verse: steps[j].verse(), Heresy: err, Output: unsaid})
		if err != nil {
			out.Verdict = Faltered
		}
	}
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
