package librarium

// Fallback is the key of an aspect map that serves every aspect not named.
const Fallback = "*"

// Entry is one value of an aspect map, keyed by an aspect or by Fallback.
type Entry[T any] struct {
	Aspect string
	Value  T
}

// AspectMap is a value of a rite that is either the same for every aspect
// (written as a single value) or one value per aspect, with Fallback serving
// every aspect not named. Entries keep the order they were written in.
//
// The zero AspectMap was never written: it holds no value for any aspect.
type AspectMap[T any] struct {
	entries []Entry[T]
	uniform bool
}

// Uniform is an aspect map written as one value for every aspect.
func Uniform[T any](v T) AspectMap[T] {
	return AspectMap[T]{entries: []Entry[T]{{Aspect: Fallback, Value: v}}, uniform: true}
}

// PerAspect is an aspect map written as an object keyed by aspect.
func PerAspect[T any](entries ...Entry[T]) AspectMap[T] {
	return AspectMap[T]{entries: append([]Entry[T]{}, entries...)}
}

// For returns the value for aspect: its own entry, else the fallback.
func (m AspectMap[T]) For(aspect string) (T, bool) {
	if key, ok := m.Key(aspect); ok {
		for _, e := range m.entries {
			if e.Aspect == key {
				return e.Value, true
			}
		}
	}
	var zero T
	return zero, false
}

// Key returns the key that answers for aspect — aspect itself or Fallback —
// and whether there is one. For a uniform map the key is Fallback.
func (m AspectMap[T]) Key(aspect string) (string, bool) {
	fallback := false
	for _, e := range m.entries {
		switch e.Aspect {
		case aspect:
			if !m.uniform {
				return aspect, true
			}
		case Fallback:
			fallback = true
		}
	}
	return Fallback, fallback
}

// Entries returns the written entries in order. A uniform map has one entry
// keyed by Fallback.
func (m AspectMap[T]) Entries() []Entry[T] { return append([]Entry[T]{}, m.entries...) }

// IsUniform reports whether the map was written as a single value.
func (m AspectMap[T]) IsUniform() bool { return m.uniform }

// IsZero reports whether the map was never written.
func (m AspectMap[T]) IsZero() bool { return m.entries == nil }
