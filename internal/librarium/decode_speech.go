package librarium

import (
	"fmt"
	"slices"

	"github.com/tailscale/hujson"
)

// decodeObservances reads the observances of a rite of aspects.
func (d *decoder) decodeObservances(v *hujson.Value) {
	arr, ok := v.Value.(*hujson.Array)
	if !ok {
		d.wrongForm(v, `the "observances" of a rite`, "a list of observances")
		return
	}
	for i := range arr.Elements {
		if o := d.decodeObservance(&arr.Elements[i], i); o != nil {
			d.rite.Observances = append(d.rite.Observances, *o)
			d.markII.observanceAt = append(d.markII.observanceAt, i)
		}
	}
}

// decodeObservance reads the observance at index i; nil when its
// designation cannot be told.
func (d *decoder) decodeObservance(v *hujson.Value, i int) *Observance {
	what := fmt.Sprintf("observance %d of the rite", i+1)
	all, ok := d.object(v, what, nil)
	if !ok {
		return nil
	}
	o := &Observance{}
	if o.Name = d.name(v, all, what, "observance", &d.markII.observances); o.Name != "" {
		what = fmt.Sprintf("the observance %q", o.Name)
	}
	designation := d.designation(v, all, what, SpeechDesignations)
	if designation == "" {
		return nil
	}
	fs := d.designated(all, what, designation, ObservanceKeys, SpeechDesignations)
	o.To = d.guard(fs, "to", what)
	o.From = d.guard(fs, "from", what)
	o.Speech = stepDecoder{d: d, v: v, fs: fs, what: what, kind: kindOf(designation), name: o.Name,
		designation: designation}.speech()
	return o
}

// guard reads the aspects an observance is spoken into ("to") or out of
// ("from"); nil when unwritten.
func (d *decoder) guard(fs fields, key, what string) []string {
	m, ok := fs.get(key)
	if !ok {
		return nil
	}
	field := fmt.Sprintf("the %q of %s", key, what)
	arr, ok := m.val.Value.(*hujson.Array)
	if !ok {
		d.wrongForm(m.val, field, "a list of declared aspects")
		return nil
	}
	aspects := []string{}
	for i := range arr.Elements {
		e := &arr.Elements[i]
		a, ok := d.str(e, "each aspect of "+field)
		if !ok {
			continue
		}
		if len(d.aspects) > 0 && !slices.Contains(d.aspects, a) {
			d.heresy(e, "%s names the aspect %q, which the rite never declared (it declares %s); an observance is "+
				"guarded by declared aspects alone", field, a, quoteAll(d.aspects))
		}
		aspects = append(aspects, a)
	}
	return aspects
}

// decodeEntries reads the liturgy of an edict of pattern Mark II: entries
// in the designation form, each named by its verse.
func (d *decoder) decodeEntries(v *hujson.Value) {
	arr, ok := v.Value.(*hujson.Array)
	if !ok {
		d.wrongForm(v, `the "liturgy" of an edict`, "a list of incantations and litanies")
		return
	}
	for i := range arr.Elements {
		e, verse := &arr.Elements[i], i+1
		what := fmt.Sprintf("verse %d of the liturgy", verse)
		all, ok := d.object(e, what, nil)
		if !ok {
			continue
		}
		designation := d.designation(e, all, what, SpeechDesignations)
		if designation == "" {
			continue
		}
		what = fmt.Sprintf("the %s of verse %d", designation, verse)
		fs := d.designated(all, what, designation, []string{"designation"}, SpeechDesignations)
		step := stepDecoder{d: d, v: e, fs: fs, what: what, kind: kindOf(designation), verse: verse}.speech()
		d.rite.Liturgy = append(d.rite.Liturgy, step)
		d.rite.verses = append(d.rite.verses, verse)
	}
}

// speech reads the words of an observance or an edict's entry: an
// incantation or a litany, every value of it written once.
func (s stepDecoder) speech() Step {
	d := s.d
	s.exalted = s.exaltation()
	if s.exalted {
		// The words of exalted speech are fixed at its sanction.
		defer d.within(d.rite.ExaltedScope())()
	}
	u := Utterance{Exalted: s.exalted, Relics: s.relics()}
	if m, ok := s.fs.get("tongue"); ok {
		u.Tongue, _ = d.word(m.val, s.field("tongue"))
	}
	if m, ok := s.fs.get("patience"); ok {
		u.Patience = d.patience(m.val, s.field("patience"))
	}
	if m, ok := s.fs.get("reversion"); ok {
		u.Reversion = s.plain(m.val, s.field("reversion"), true)
	}
	if s.kind == KindIncantation {
		in := &Incantation{Utterance: u}
		if m, ok := s.required("words", "an incantation must hold the words it speaks"); ok {
			in.Command = s.plain(m.val, s.field("words"), false)
		}
		return in
	}
	l := &Litany{Utterance: u}
	if m, ok := s.required("scroll", "a litany must name the scroll it recites"); ok {
		l.Scroll = s.plain(m.val, s.field("scroll"), false)
	}
	if m, ok := s.fs.get("offerings"); ok {
		arr, ok := m.val.Value.(*hujson.Array)
		if !ok {
			d.wrongForm(m.val, s.field("offerings"), "a list of offerings")
			return l
		}
		l.Offerings = []AspectMap[string]{}
		for i := range arr.Elements {
			what := fmt.Sprintf("offering %d of %s", i+1, s.what)
			l.Offerings = append(l.Offerings, s.plain(&arr.Elements[i], what, true))
		}
	}
	return l
}

// plain reads words written once for every aspect: the words of speech in
// pattern Mark II are never an aspect map.
func (s stepDecoder) plain(v *hujson.Value, what string, mayBeEmpty bool) AspectMap[string] {
	if _, isMap := v.Value.(*hujson.Object); isMap {
		if s.d.rite.Edict {
			s.d.heresy(v, "%s is written as an aspect map, yet an edict bears no aspect to key it by: it is "+
				"proclaimed whole, never invoked into an aspect — write one plain value in its stead", what)
		} else {
			s.d.heresy(v, "%s is written as an aspect map, yet the words of an observance are one for every "+
				"aspect it is spoken into — should they differ by aspect, write several observances, each guarded "+
				"by \"to\"", what)
		}
		return Uniform("")
	}
	w, _ := s.d.templated(v, what, mayBeEmpty)
	return Uniform(w)
}
