package librarium

import (
	"slices"

	"github.com/tailscale/hujson"
)

// aspectMap decodes a value that may vary per aspect: an object keyed by
// aspect (with Fallback) or a single value read by one. isValue tells objects
// that are single values (such as tomes) from aspect maps; nil means every
// object is an aspect map. When full, every declared aspect must be served.
func aspectMap[T any](d *decoder, v *hujson.Value, what string, full bool,
	isValue func(*hujson.Value) bool, one func(*hujson.Value, string) (T, bool)) AspectMap[T] {
	if _, isObj := v.Value.(*hujson.Object); !isObj || (isValue != nil && isValue(v)) {
		val, _ := one(v, what)
		return Uniform(val)
	}
	m := AspectMap[T]{entries: []Entry[T]{}}
	fs, _ := d.object(v, what, nil)
	for _, f := range fs {
		if f.name != Fallback && len(d.aspects) > 0 && !slices.Contains(d.aspects, f.name) {
			d.heresy(f.key, "%s names the aspect %q, which the rite never declared (it declares %s); every "+
				"key of an aspect map must be a declared aspect, or %q to serve every aspect not named",
				what, f.name, quoteAll(d.aspects), Fallback)
		}
		entry := what + " for the aspect " + quoteAll([]string{f.name})
		if f.name == Fallback {
			entry = what + " for every aspect not named otherwise"
		}
		val, _ := one(f.val, entry)
		m.entries = append(m.entries, Entry[T]{Aspect: f.name, Value: val})
	}
	if missing := m.unserved(d.aspects); full && len(missing) > 0 {
		d.heresy(v, "%s gives no value for %s and holds no %q to serve %s; were the rite invoked into "+
			"such an aspect, the servitor would not know what to do", what, aspectList(missing), Fallback,
			pronoun(missing))
	}
	return m
}

// unserved returns the aspects the map holds no value for.
func (m AspectMap[T]) unserved(aspects []string) []string {
	var missing []string
	for _, a := range aspects {
		if _, ok := m.For(a); !ok {
			missing = append(missing, a)
		}
	}
	return missing
}

func aspectList(aspects []string) string {
	if len(aspects) == 1 {
		return "the aspect " + quoteAll(aspects)
	}
	return "the aspects " + quoteAll(aspects)
}

func pronoun(aspects []string) string {
	if len(aspects) == 1 {
		return "it"
	}
	return "them"
}

// unDecreed notes a mandatory inscription that some aspect has no decree for.
func (d *decoder) unDecreed(key *hujson.Value, in Inscription) {
	missing := in.Decrees.unserved(d.aspects)
	if len(missing) == 0 {
		return
	}
	d.impurity(key, "the inscription %q is mandatory, yet no decree serves %s; an invocation into %s "+
		"will fail unless the rune --%s is given each time", in.Key, aspectList(missing),
		pronoun(missing), in.Key)
}
