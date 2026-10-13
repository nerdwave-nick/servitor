package librarium

import (
	"fmt"
	"slices"
	"strings"
)

// converting gathers the rite of pattern Mark II a rite of pattern Mark I
// becomes, the notes written as comments before the values of its
// scripture (by JSON pointer), and what changed in meaning.
type converting struct {
	from    *Rite
	to      *Rite
	notes   map[string][]string
	changes []string
}

// convert converts the rite r of pattern Mark I: sanctums, transcriptions
// and tethers become subjects of their designation; speech becomes
// observances beside them, or machine-spirits in a rite without them;
// vox-casts become the rite's list; an edict's verses take the designation
// form.
func convert(r *Rite) *converting {
	c := &converting{from: r, notes: map[string][]string{}, to: &Rite{Name: r.Name, Path: r.Path, Schema: r.Schema,
		Purpose: r.Purpose, Aspects: r.Aspects, Edict: r.Edict, Inscriptions: r.Inscriptions, Tongue: r.Tongue,
		VoxCasts: voxCastsOf(r.Liturgy)}}
	switch {
	case r.Edict:
		for _, s := range r.Liturgy {
			if s.Kind() != kindVoxCast {
				c.to.Liturgy = append(c.to.Liturgy, s)
			}
		}
	case slices.ContainsFunc(r.Liturgy, builtIn):
		c.observed()
	default:
		c.spirits()
	}
	return c
}

// builtIn reports whether s reads and sets itself: a sanctum, a
// transcription or a tether.
func builtIn(s Step) bool {
	k := s.Kind()
	return k == KindSanctum || k == KindTranscription || k == KindTether
}

// voxCastsOf lists the tidings the vox-casts of liturgy proclaim, progress
// before success; nil when it holds none.
func voxCastsOf(liturgy []Step) []string {
	var out []string
	for _, t := range []string{VoxProgress, VoxSuccess} {
		if slices.ContainsFunc(liturgy, func(s Step) bool { v, ok := s.(*voxCast); return ok && v.Tidings == t }) {
			out = append(out, t)
		}
	}
	return out
}

// nameOf names what verse i of the rite converted becomes: "<kind>-<verse>".
func (c *converting) nameOf(i int) string {
	return fmt.Sprintf("%s-%d", c.from.Liturgy[i].Kind().Key(), c.from.verse(i))
}

// observed converts a rite of sanctums, transcriptions or tethers: each
// becomes a subject; speech becomes observances spoken as the introit of
// the next subject after it, else as the recessional of the last before.
func (c *converting) observed() {
	at := map[int]int{} // the subject each built-in verse becomes
	for i, s := range c.from.Liturgy {
		if builtIn(s) {
			at[i] = len(c.to.Subjects)
			c.to.Subjects = append(c.to.Subjects, Subject{Name: c.nameOf(i), Matter: s.(Matter)})
		}
	}
	for i, s := range c.from.Liturgy {
		if s.Kind() != KindIncantation && s.Kind() != KindLitany {
			continue
		}
		next := slices.IndexFunc(c.from.Liturgy[i+1:], builtIn)
		for _, o := range c.observances(i) {
			c.to.Observances = append(c.to.Observances, o)
			if next >= 0 {
				sub := &c.to.Subjects[at[i+1+next]]
				sub.Introit = append(sub.Introit, o.Name)
			} else {
				sub := &c.to.Subjects[len(c.to.Subjects)-1]
				sub.Recessional = append(sub.Recessional, o.Name)
			}
		}
	}
	if a := c.from.auspex; a != nil {
		c.strikeAuspex(a, "each subject is read by itself, and no machine-spirit is left to bear it")
	}
}

// observances converts the speech of verse i into observances: one, or
// one for every distinct word it speaks per aspect, guarded by "to".
func (c *converting) observances(i int) []Observance {
	groups := c.groups(c.from.Liturgy[i])
	if len(groups) == 1 && len(groups[0]) == len(c.from.Aspects) {
		return []Observance{{Name: c.nameOf(i), Speech: spokenFor(c.from.Liturgy[i], groups[0][0])}}
	}
	out := make([]Observance, len(groups))
	for g, aspects := range groups {
		out[g] = Observance{Name: c.nameOf(i) + "-" + aspects[0], To: aspects,
			Speech: spokenFor(c.from.Liturgy[i], aspects[0])}
	}
	return out
}

// groups gathers the aspects for which the speech s speaks the same words
// (its command or scroll, its offerings and its reversion), in the order
// of the aspects; an aspect it speaks nothing for is in none.
func (c *converting) groups(s Step) [][]string {
	var keys []string
	var groups [][]string
	for _, a := range c.from.Aspects {
		key, ok := wordsFor(s, a)
		if !ok {
			continue
		}
		if g := slices.Index(keys, key); g >= 0 {
			groups[g] = append(groups[g], a)
			continue
		}
		keys, groups = append(keys, key), append(groups, []string{a})
	}
	return groups
}

// wordsFor joins every word the speech s speaks for aspect; false when it
// speaks nothing for it.
func wordsFor(s Step, aspect string) (string, bool) {
	var maps []AspectMap[string]
	var u Utterance
	switch s := s.(type) {
	case *Incantation:
		maps, u = []AspectMap[string]{s.Command}, s.Utterance
	case *Litany:
		maps, u = append([]AspectMap[string]{s.Scroll}, s.Offerings...), s.Utterance
	}
	words := make([]string, 0, len(maps)+1)
	for _, m := range maps {
		w, ok := m.For(aspect)
		if !ok {
			return "", false
		}
		words = append(words, w)
	}
	rev, ok := u.Reversion.For(aspect)
	words = append(words, fmt.Sprint(ok, rev))
	return strings.Join(words, "\x00"), true
}

// spokenFor is the speech s as spoken into aspect, every value of it
// written once.
func spokenFor(s Step, aspect string) Step {
	once := func(m AspectMap[string]) AspectMap[string] {
		v, _ := m.For(aspect)
		return Uniform(v)
	}
	switch s := s.(type) {
	case *Incantation:
		return &Incantation{Command: once(s.Command), Utterance: s.spokenFor(aspect)}
	case *Litany:
		l := &Litany{Scroll: once(s.Scroll), Utterance: s.spokenFor(aspect)}
		for _, o := range s.Offerings {
			l.Offerings = append(l.Offerings, once(o))
		}
		return l
	}
	return s
}

// spokenFor is the utterance as spoken into aspect: its reversion written
// once, or unwritten when it reverts nothing there.
func (u Utterance) spokenFor(aspect string) Utterance {
	rev, ok := u.Reversion.For(aspect)
	u.Reversion = AspectMap[string]{}
	if ok {
		u.Reversion = Uniform(rev)
	}
	return u
}

// strikeAuspex notes the rite's auspex of pattern Mark I as struck, for
// the reason given.
func (c *converting) strikeAuspex(a *riteAuspex, reason string) {
	patience := ""
	if a.Patience > 0 {
		patience = fmt.Sprintf(" (patience %s)", a.Patience)
	}
	c.note("/subjects", fmt.Sprintf("pattern %q read the aspect of the whole rite by the auspex %q%s;", PatternMarkI,
		a.Rite, patience), "it is struck, for "+reason)
	c.changes = append(c.changes, "the auspex of the rite is struck")
}

// note keeps lines of comment before the value at the JSON pointer ptr.
func (c *converting) note(ptr string, lines ...string) {
	c.notes[ptr] = append(c.notes[ptr], lines...)
}

// parseConverting reads the scripture data of the rite name kept at path
// and, when it is pure scripture of pattern Mark I, converts it to pattern
// Mark II: the rite and findings returned are then those of the scripture
// converted, which the Conversion holds to be written.
//
// Heretical scripture of pattern Mark I is not converted: no rite is
// returned, only its findings, and one more heresy tells that it is
// converted once its heresies are purged. Scripture that would be
// heretical once converted is not converted either, and its heresies are
// told upon its "pattern".
func parseConverting(name, path string, data []byte) (*Rite, Findings, *Conversion) {
	r, fs := parse(name, path, data)
	if r == nil || !r.markI {
		return r, fs, nil
	}
	if fs.Heretical() {
		return nil, append(fs, r.unconverted("its heresies are purged")), nil
	}
	c := convert(r)
	w := &writer{notes: c.notes}
	w.riteII(c.to)
	converted, err := w.layout()
	if err != nil {
		return nil, append(fs, r.unconverted(err.Error())), nil
	}
	r2, fs2 := parse(name, path, converted)
	if fs2.Heretical() {
		for _, f := range fs2 {
			if f.Severity == Heresy {
				fs = append(fs, r.unconverted(fmt.Sprintf("this heresy is purged from the scripture it would "+
					"become (at %d:%d): %s", f.Line, f.Column, f.Message)))
			}
		}
		return nil, fs, nil
	}
	return r2, fs2, &Conversion{Rite: name, Path: path, Original: data, Data: converted, Changes: c.changes,
		Exalted: r.Exalted()}
}

// unconverted denounces the scripture of pattern Mark I that is not
// converted until what is told comes to pass.
func (r *Rite) unconverted(until string) Finding {
	return Finding{Severity: Heresy, Scripture: r.Path, Rite: r.Name, Position: r.Locate("/pattern"),
		Message: fmt.Sprintf("the scripture of pattern %q is not converted to pattern %q, which this servitor reads "+
			"in its stead, and so it cannot be invoked; it is converted once %s", PatternMarkI, Pattern, until)}
}
