package tui

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// riteDraft is the editable form of a rite, as the consecration wizard
// holds it between its stations. Values are kept as typed; the forms have
// examined them before they are stored. A rite of aspects is written from
// its subjects and observances, an edict from its steps (incantations and
// litanies), each in pattern Mark II; the vox-casts are a list of the rite.
type riteDraft struct {
	origName, origPath string // the rite amended; empty for a new one
	name, purpose      string
	schema             string // the "$schema" kept as read
	aspects            string // separated by commas
	edict              bool   // proclaimed whole, without aspects
	inscriptions       []inscriptionDraft
	voxCasts           []string // the vox-casts of the rite; nil when unwritten
	tongue             string
	steps              []stepDraft       // the liturgy of an edict
	subjects           []subjectDraft    // of a rite of aspects
	observances        []observanceDraft // of a rite of aspects
}

// inscriptionDraft is one declared inscription; its decrees are keyed by
// aspect or "*", and an empty decree is none.
type inscriptionDraft struct {
	key, purpose string
	mandatory    bool
	decrees      map[string]string
}

func newRiteDraft() riteDraft { return riteDraft{aspects: "on, off"} }

// riteDraftFrom is the rite r as the wizard amends it.
func riteDraftFrom(r *librarium.Rite) riteDraft {
	d := riteDraft{origName: r.Name, origPath: r.Path, name: r.Name, purpose: r.Purpose, schema: r.Schema,
		aspects: strings.Join(r.Aspects, ", "), edict: r.Edict, tongue: r.Tongue, voxCasts: r.VoxCasts}
	for _, in := range r.Inscriptions {
		id := inscriptionDraft{key: in.Key, purpose: in.Purpose, mandatory: in.Mandatory, decrees: map[string]string{}}
		for _, e := range in.Decrees.Entries() {
			id.decrees[e.Aspect] = e.Value
		}
		d.inscriptions = append(d.inscriptions, id)
	}
	for _, s := range r.Liturgy {
		d.steps = append(d.steps, stepFrom(s))
	}
	for _, s := range r.Subjects {
		d.subjects = append(d.subjects, subjectFrom(s))
	}
	for _, o := range r.Observances {
		d.observances = append(d.observances, observanceFrom(o))
	}
	return d
}

// bearsSubjects reports whether the draft is written from its subjects: a
// rite of aspects, not an edict.
func (d riteDraft) bearsSubjects() bool { return !d.edict }

// aspectList is the declared aspects; the form has examined them. An
// edict declares none.
func (d riteDraft) aspectList() []string {
	if d.edict {
		return nil
	}
	a, _ := parseAspects(d.aspects)
	return a
}

// noun is what the draft is called: a rite, or an edict.
func (d riteDraft) noun() string {
	if d.edict {
		return "edict"
	}
	return "rite"
}

// keys are the declared inscription keys in order.
func (d riteDraft) keys() []string {
	keys := make([]string, len(d.inscriptions))
	for i, in := range d.inscriptions {
		keys[i] = in.key
	}
	return keys
}

// declare declares the inscriptions keys in order, keeping what was
// written of those declared before.
func (d *riteDraft) declare(keys []string) {
	var next []inscriptionDraft
	for _, k := range keys {
		in := inscriptionDraft{key: k, decrees: map[string]string{}}
		for _, old := range d.inscriptions {
			if old.key == k {
				in = old
			}
		}
		next = append(next, in)
	}
	d.inscriptions = next
}

// toRite writes the rite as it would be recorded at path: an edict without
// aspects, from its steps, whatever the draft kept of subjects; a rite of
// aspects from its subjects and observances.
func (d riteDraft) toRite(path string) *librarium.Rite {
	aspects := d.aspectList()
	r := &librarium.Rite{Name: d.name, Path: path, Schema: d.schema, Purpose: strings.TrimSpace(d.purpose),
		Aspects: aspects, Edict: d.edict, Tongue: strings.TrimSpace(d.tongue), VoxCasts: d.voxCasts,
		MarkII: true} // the pattern of every scripture written, until librarium knows no other
	for _, id := range d.inscriptions {
		in := librarium.Inscription{Key: id.key, Purpose: strings.TrimSpace(id.purpose), Mandatory: id.mandatory}
		v := newVaried[string](false)
		for _, a := range aspects {
			if x := id.decrees[a]; x != "" {
				v.set(a, x, false)
			}
		}
		if x := id.decrees[librarium.Fallback]; x != "" {
			v.shared, v.hasShared = x, true
		}
		if len(v.own) > 0 || v.hasShared {
			in.Decrees = v.toMap(aspects)
		}
		if d.bearsSubjects() {
			in.Decrees = object(in.Decrees)
		}
		r.Inscriptions = append(r.Inscriptions, in)
	}
	if d.bearsSubjects() {
		for _, sd := range d.subjects {
			r.Subjects = append(r.Subjects, sd.toSubject(d))
		}
		for _, od := range d.observances {
			r.Observances = append(r.Observances, od.toObservance())
		}
		return r
	}
	for _, sd := range d.steps {
		r.Liturgy = append(r.Liturgy, sd.toStep(nil))
	}
	return r
}

// splitList parses a list separated by commas or whitespace.
func splitList(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' })
}

func parseAspects(s string) ([]string, error) {
	aspects := splitList(s)
	if len(aspects) == 0 {
		return nil, errors.New("a rite without aspects can never be invoked; name at least one")
	}
	return aspects, names(aspects, "aspect")
}

// parseKeys parses the declared inscription keys.
func parseKeys(s string) ([]string, error) {
	keys := splitList(s)
	if err := names(keys, "inscription"); err != nil {
		return nil, err
	}
	for _, k := range keys {
		if slices.Contains(librarium.ReservedRunes, k) {
			return nil, fmt.Errorf("%q is a rune the servitor bears for its own rituals; choose another name", k)
		}
	}
	return keys, nil
}

// names examines a list of names of what: each fit and spoken once.
func names(list []string, what string) error {
	for i, n := range list {
		switch {
		case !librarium.ValidName(n):
			return fmt.Errorf("%q is no fit name for an %s: letters, digits, '.', '_' and '-'", n, what)
		case slices.Contains(list[:i], n):
			return fmt.Errorf("the %s %q is named twice", what, n)
		}
	}
	return nil
}
