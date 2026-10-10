package invocation

import (
	"io/fs"
	"strings"
)

// Foresight tells what one step would do. Exactly one of its changes is set
// for the steps that change the machine.
type Foresight struct {
	Verse
	Vessel *VesselChange // sanctums and transcriptions
	Tether *TetherChange // tethers
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
