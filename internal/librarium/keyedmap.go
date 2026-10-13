package librarium

import (
	"fmt"
	"slices"
	"strings"

	"github.com/tailscale/hujson"

	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// keying tells the keys a map of pattern Mark II is keyed by: the rite's
// declared aspects, or a subject's declared omens.
type keying struct {
	noun  string   // "aspect" or "omen"
	keys  []string // declared once each; nil when they could not be read
	owner string   // who declares them, such as "the rite"
}

func (d *decoder) aspectKeying() keying {
	return keying{noun: "aspect", keys: d.aspects, owner: "the rite"}
}

// list names keys as "the omen "x"" or "the omens "x", "y"".
func (k keying) list(keys []string) string {
	if len(keys) == 1 {
		return "the " + k.noun + " " + quoteAll(keys)
	}
	return "the " + k.noun + "s " + quoteAll(keys)
}

// keyedMap decodes a map of pattern Mark II: always an object, keyed by a
// declared key or by Fallback, which serves every key it does not name.
// When full, every declared key must be served. A map whose keys could not
// be read is not judged by them.
func keyedMap[T any](d *decoder, v *hujson.Value, what string, k keying, full bool,
	one func(*hujson.Value, string) (T, bool)) AspectMap[T] {
	if _, isObj := v.Value.(*hujson.Object); !isObj {
		d.heresy(v, "%s must be written as an object keyed by %s, with %q serving every %s it does not name, "+
			"yet here stands %s; pattern %q knows no single value in its stead — write {%q: …} for one value "+
			"that serves every %s", what, k.noun, Fallback, k.noun, formOf(v), PatternMarkII, Fallback, k.noun)
		return AspectMap[T]{}
	}
	m := AspectMap[T]{entries: []Entry[T]{}}
	fs, _ := d.object(v, what, nil)
	for _, f := range fs {
		if f.name != Fallback && len(k.keys) > 0 && !slices.Contains(k.keys, f.name) {
			d.heresy(f.key, "%s names the %s %q, which %s never declared (it declares %s); every key here must "+
				"be a declared %s, or %q to serve every %s not named", what, k.noun, f.name, k.owner,
				quoteAll(k.keys), k.noun, Fallback, k.noun)
		}
		entry := what + " for the " + k.noun + " " + quoteAll([]string{f.name})
		if f.name == Fallback {
			entry = what + " for every " + k.noun + " not named otherwise"
		}
		val, _ := one(f.val, entry)
		m.entries = append(m.entries, Entry[T]{Aspect: f.name, Value: val})
	}
	if missing := m.unserved(k.keys); full && len(missing) > 0 {
		d.heresy(v, "%s gives no value for %s and holds no %q to serve %s; were the machine brought there, the "+
			"servitor would not know what to do", what, k.list(missing), Fallback, pronoun(missing))
	}
	return m
}

// values are the values the words of the rite are illuminated with when
// aspect is invoked with every inscription at its decree.
func (d *decoder) values(aspect string) placeholder.Values {
	inscriptions := map[string]string{}
	for _, in := range d.rite.Inscriptions {
		inscriptions[in.Key], _ = in.Decrees.For(aspect)
	}
	return placeholder.Values{Aspect: aspect, Rite: d.rite.Name, Inscriptions: inscriptions}
}

// declaredOmens returns the omens a machine-spirit declares: its omens
// rendered for each aspect and trimmed, once each in the order of the
// aspects, or the aspects themselves when it writes none. It is nil when
// they cannot be told, for the omens hold heresy already.
func (d *decoder) declaredOmens(omens AspectMap[string]) []string {
	if omens.IsZero() {
		return slices.Clone(d.aspects)
	}
	var out []string
	for _, a := range d.aspects {
		w, ok := omens.For(a)
		if !ok {
			return nil
		}
		rendered, err := d.scope.Render(w, d.values(a))
		if err != nil {
			return nil
		}
		if o := strings.TrimSpace(rendered); !slices.Contains(out, o) {
			out = append(out, o)
		}
	}
	return out
}

// omenKeying is the keying of the maps of the machine-spirit what, keyed by
// its declared omens.
func omenKeying(what string, omens []string) keying {
	return keying{noun: "omen", keys: omens, owner: what}
}

// subjectAt names a subject in a denunciation before its name is known.
func subjectAt(i int) string { return fmt.Sprintf("subject %d of the rite", i+1) }
