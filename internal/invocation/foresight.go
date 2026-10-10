package invocation

import (
	"io/fs"
	"strings"
	"time"
)

// Foresight tells what one step would do. Exactly one of its changes is set
// for the steps that change the machine.
type Foresight struct {
	Verse
	Vessel *VesselChange // sanctums and transcriptions
	Tether *TetherChange // tethers
	Speech *Speech       // incantations and litanies
}

// Speech is what an incantation or litany will speak, every placeholder
// rendered.
type Speech struct {
	// Tongue is the tongue of the step: step > rite > settings > bash. The
	// reversion is always spoken in it; a litany recited by its own shebang
	// speaks only its reversion in it.
	Tongue string
	// Shebang is true for a litany whose scroll may be executed: it is
	// recited by its own shebang rather than through the tongue.
	Shebang bool
	// Words are the incantation's command, or the path of the litany's scroll.
	Words string
	// Offerings are a litany's offerings, exactly as they will be handed over.
	Offerings []string
	// Argv is the program and every argument exactly as they will be spoken.
	Argv []string
	// Reversion is the command spoken in the tongue should the invocation
	// fall; "" when there is none.
	Reversion string
	// Patience is how long the step, and its reversion, may labour.
	Patience time.Duration
}

// VesselChange is what a sanctum or transcription does to its vessel.
type VesselChange struct {
	Path            string // the vessel written, its symbolic links followed
	Before, After   string // scripture before and after; "" when absent
	Existed, Exists bool   // whether the vessel stands before and after
	BeforeSeal      fs.FileMode
	AfterSeal       fs.FileMode
}

// Changed reports whether the vessel changes at all.
func (c VesselChange) Changed() bool {
	return c.Existed != c.Exists || c.Before != c.After || (c.Exists && c.BeforeSeal != c.AfterSeal)
}

// Diff returns the lines that differ, prefixed with "- " and "+ ", between
// the common beginning and the common end of the scripture.
func (c VesselChange) Diff() []string { return diffLines(c.Before, c.After) }

// TetherChange is what a tether does to the name it binds.
type TetherChange struct {
	Name     string // the bound name
	From, To string // the anchor before and after; "" when unbound
	// Displaced is true when a vessel that was no tether is cast down by
	// zeal to make room.
	Displaced bool
}

// Changed reports whether the tether changes at all.
func (c TetherChange) Changed() bool { return c.From != c.To || c.Displaced }

func diffLines(before, after string) []string {
	if before == after {
		return nil
	}
	a, b := strings.Split(before, "\n"), strings.Split(after, "\n")
	p := 0
	for p < len(a) && p < len(b) && a[p] == b[p] {
		p++
	}
	s := 0
	for s < len(a)-p && s < len(b)-p && a[len(a)-1-s] == b[len(b)-1-s] {
		s++
	}
	var out []string
	for _, l := range a[p : len(a)-s] {
		out = append(out, "- "+l)
	}
	for _, l := range b[p : len(b)-s] {
		out = append(out, "+ "+l)
	}
	return out
}
