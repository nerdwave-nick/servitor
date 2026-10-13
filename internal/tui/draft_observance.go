package tui

import (
	"slices"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// observanceDraft is the editable form of one observance of a rite of
// pattern Mark II: its name, its guards, and its speech written as the
// step of an edict is, each value once for every aspect.
type observanceDraft struct {
	name     string
	to, from []string // the aspects it is guarded by; nil when unwritten: every one
	step     stepDraft
}

// observanceFrom is the observance o as the wizard edits it.
func observanceFrom(o librarium.Observance) observanceDraft {
	return observanceDraft{name: o.Name, to: o.To, from: o.From, step: stepFrom(o.Speech)}
}

// toObservance writes the observance, its words written once.
func (od observanceDraft) toObservance() librarium.Observance {
	return librarium.Observance{Name: od.name, To: od.to, From: od.from, Speech: od.step.toStep(nil)}
}

// keptChoice is the choice among a list as written: before when the same
// are chosen, so that its order and its being written stay as read; else
// those chosen, unwritten when none is.
func keptChoice(before, chosen []string) []string {
	same := len(before) == len(chosen)
	for _, c := range chosen {
		same = same && slices.Contains(before, c)
	}
	switch {
	case same:
		return before
	case len(chosen) == 0:
		return nil
	}
	return chosen
}

// renamed is the list of names with old renamed name, as a list of its
// own; struck when name is "".
func renamed(list []string, old, name string) []string {
	if !slices.Contains(list, old) {
		return list
	}
	var out []string
	for _, n := range list {
		switch {
		case n != old:
			out = append(out, n)
		case name != "":
			out = append(out, name)
		}
	}
	return out
}
