package tui

import (
	"maps"
	"slices"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// varied is a value of a step that may vary per aspect, as the wizard
// edits it: each aspect either speaks a value of its own or shares the
// value written under "*" — the same for every aspect.
type varied[T any] struct {
	own       map[string]T // aspect → its own value
	shared    T            // the value under "*"
	hasShared bool
	apart     bool // aspects not yet given a value begin with one of their own
}

func newVaried[T any](apart bool) varied[T] { return varied[T]{own: map[string]T{}, apart: apart} }

// variedFrom is the aspect map m as the wizard edits it.
func variedFrom[T any](m librarium.AspectMap[T]) varied[T] {
	v := newVaried[T](false)
	for _, e := range m.Entries() {
		if e.Aspect == librarium.Fallback {
			v.shared, v.hasShared = e.Value, true
		} else {
			v.own[e.Aspect] = e.Value
		}
	}
	return v
}

// value is what aspect holds, and whether it is the value shared by every
// aspect rather than its own.
func (v *varied[T]) value(aspect string) (T, bool) {
	if x, ok := v.own[aspect]; ok {
		return x, false
	}
	if !v.hasShared && v.apart {
		var zero T
		return zero, false
	}
	return v.shared, true
}

// set gives aspect the value x: its own, or — when same — the value every
// aspect shares.
func (v *varied[T]) set(aspect string, x T, same bool) {
	if v.own == nil {
		v.own = map[string]T{}
	}
	if same {
		delete(v.own, aspect)
		v.shared, v.hasShared = x, true
		return
	}
	v.own[aspect] = x
}

// setOnce gives the value x once for all, as an edict writes it: what any
// aspect spoke of its own is forgotten.
func (v *varied[T]) setOnce(x T) {
	v.own = map[string]T{}
	v.shared, v.hasShared = x, true
}

// spoken are the aspects a value written for an edict, which declares
// none, still speaks of its own — kept from the rite it was, for the
// Inquisition to denounce — followed by "*".
func (v varied[T]) spoken() []string {
	return append(slices.Sorted(maps.Keys(v.own)), librarium.Fallback)
}

// ownAfter reports whether an aspect after aspects[i] speaks a value of
// its own.
func (v varied[T]) ownAfter(aspects []string, i int) bool {
	for _, a := range aspects[i+1:] {
		if _, ok := v.own[a]; ok {
			return true
		}
	}
	return false
}

// toMap writes the value as an aspect map over aspects: one value when
// every aspect shares it, else each aspect's own followed by "*" for the
// aspects that share. A value never given stays unwritten, and an aspect
// given none is left for the Inquisition to denounce. Without aspects, as
// for an edict, the value is written once unless aspects spoke their own.
func (v varied[T]) toMap(aspects []string) librarium.AspectMap[T] {
	if len(aspects) == 0 {
		aspects = v.spoken()
	}
	var entries []librarium.Entry[T]
	sharing := false
	for _, a := range aspects {
		if x, ok := v.own[a]; ok {
			entries = append(entries, librarium.Entry[T]{Aspect: a, Value: x})
		} else {
			sharing = true
		}
	}
	switch {
	case len(entries) == 0 && !v.hasShared:
		return librarium.AspectMap[T]{}
	case len(entries) == 0:
		return librarium.Uniform(v.shared)
	case sharing && v.hasShared:
		entries = append(entries, librarium.Entry[T]{Aspect: librarium.Fallback, Value: v.shared})
	}
	return librarium.PerAspect(entries...)
}
