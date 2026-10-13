package invocation

// Verdict is how an invocation ended.
type Verdict string

const (
	// Triumph: every step was performed.
	Triumph Verdict = "triumph"
	// Reverted: a step fell and every reversion triumphed.
	Reverted Verdict = "reverted"
	// Faltered: a step fell and at least one reversion fell too.
	Faltered Verdict = "faltered"
)

// Outcome is how a performance ended: its verdict, every step attempted, the
// step that fell and every reversion.
type Outcome struct {
	Verdict Verdict
	// Deeds are the steps attempted, in written order; the last is the
	// fallen step when one fell.
	Deeds      []Deed
	Fell       *Fall       // nil when every step was performed
	Reversions []Reversion // in the order performed: from the fallen step back to the first
	// Changed names, for a state rite of pattern Mark II alone, every
	// subject set and proven, in written order: never nil for one, even
	// when nothing changed; nil for every other rite.
	Changed []string
	// Known are, for a state rite of pattern Mark II alone, the omens its
	// subjects were known to show at the end of the performance, by
	// subject (see known): never nil for one; nil for every other rite.
	Known map[string]Reading
}

// Deed is one step as it was performed.
type Deed struct {
	Verse
	// Output is what an incantation or litany uttered, its standard output
	// and standard error interleaved; "" for every other step.
	Output string
	Heresy error // nil when the step was performed
}

// Fall is the step that failed and why.
type Fall struct {
	Verse
	Heresy error
	Output string // what the fallen incantation or litany uttered
}

// Reversion is the outcome of undoing one step; Heresy is nil when it
// triumphed.
type Reversion struct {
	Verse
	Heresy error
	Output string // what the reversion of an incantation or litany uttered
}
