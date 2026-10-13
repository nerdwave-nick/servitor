package librarium

import (
	"fmt"
	"slices"

	"github.com/tailscale/hujson"
)

// markIIDecoding is what the decoder keeps of a scripture of pattern Mark II
// to judge once every key is read: where each subject and observance is
// named, and every observance a subject names.
type markIIDecoding struct {
	subjects     []named
	observances  []named
	references   []reference
	observanceAt []int // the index of each of the rite's Observances in the scripture
}

// named is one subject or observance as named in the scripture.
type named struct {
	name string
	at   *hujson.Value
}

// reference is one observance a subject names in its introit or recessional.
type reference struct {
	subject, observance string
	recessional         bool
	at                  *hujson.Value
}

// decodeMarkII reads the rest of a rite of pattern Mark II: a rite of
// aspects with its subjects and observances, or an edict with its liturgy.
func (d *decoder) decodeMarkII(root *hujson.Value) *Rite {
	r := d.rite
	r.MarkII = true
	all, _ := d.object(root, "a rite", nil)
	var fs fields
	for _, m := range all {
		switch {
		case m.name == "auspex":
			d.heresy(m.key, "a rite of pattern %q bears no \"auspex\" of its own: each subject is read by itself, "+
				"a sanctum by its marker, a tether by its anchor, a transcription by its vessel, and a "+
				"machine-spirit by the \"auspex\" among its own keys — move these words into the \"scan\" of the "+
				"auspex of a machine-spirit among \"subjects\", or strike them", PatternMarkII)
		case !slices.Contains(MarkIIRiteKeys, m.name):
			d.unknownKey(m, "a rite", MarkIIRiteKeys)
		default:
			fs = append(fs, m)
		}
	}
	if m, ok := fs.get(SchemaKey); ok {
		r.Schema, _ = d.str(m.val, `the "$schema" of a rite`)
	}
	if m, ok := fs.get("purpose"); ok {
		r.Purpose, _ = d.str(m.val, `the "purpose" of a rite`)
	}
	d.decodeAspectsOrEdict(root, fs)
	if m, ok := fs.get("inscriptions"); ok {
		d.decodeInscriptions(m.val)
	}
	d.scope = r.Scope()
	if m, ok := fs.get("tongue"); ok {
		r.Tongue, _ = d.word(m.val, `the "tongue" of a rite`)
	}
	if m, ok := fs.get("vox-cast"); ok {
		d.decodeVoxCasts(m.val)
	}
	if r.Edict {
		d.decodeEdictII(fs)
	} else {
		d.decodeAspectsII(root, fs)
	}
	d.judgeSpeech()
	return r
}

// decodeEdictII reads the liturgy of an edict of pattern Mark II, which
// holds no subject and no observance.
func (d *decoder) decodeEdictII(fs fields) {
	for _, key := range []string{"subjects", "observances"} {
		if m, ok := fs.get(key); ok {
			d.heresy(m.key, "an edict bears no %q: subjects and observances belong to a rite that brings the "+
				"machine into its aspects, yet an edict is proclaimed whole and bears no aspect — write what it "+
				"performs as entries of its \"liturgy\", or strike \"edict\" and declare its \"aspects\"", key)
		}
	}
	if m, ok := fs.get("liturgy"); ok {
		d.decodeEntries(m.val)
	}
}

// decodeAspectsII reads the subjects and observances of a rite of aspects
// of pattern Mark II, which keeps no liturgy.
func (d *decoder) decodeAspectsII(root *hujson.Value, fs fields) {
	if m, ok := fs.get("liturgy"); ok {
		d.heresy(m.key, "a rite of aspects of pattern %q bears no \"liturgy\": what it brings into each aspect "+
			"is declared under \"subjects\", each read and set by itself, and what it does besides as named "+
			"\"observances\" spoken before or after a subject changes — move each step there; only an edict "+
			"keeps a liturgy", PatternMarkII)
	}
	if m, ok := fs.get("observances"); ok {
		d.decodeObservances(m.val)
	}
	if m, ok := fs.get("subjects"); ok {
		d.decodeSubjects(m.val)
	} else if _, hasAspects := fs.get("aspects"); hasAspects {
		d.heresy(root, "the rite declares no \"subjects\"; a rite of aspects reads and sets its subjects, and a "+
			"rite with nothing to read or set can bring the machine into no aspect — declare its subjects, or "+
			"mark it an edict with \"edict\": true should it only ever be proclaimed whole")
	}
	d.judgeObservances()
	d.judgeSubjects()
}

// decodeVoxCasts reads the rite's vox-casts: "progress" and "success", each
// at most once, in any order.
func (d *decoder) decodeVoxCasts(v *hujson.Value) {
	arr, ok := v.Value.(*hujson.Array)
	if !ok {
		d.wrongForm(v, `the "vox-cast" of a rite`, fmt.Sprintf("a list of %q and %q", VoxProgress, VoxSuccess))
		return
	}
	d.rite.VoxCasts = []string{}
	for i := range arr.Elements {
		e := &arr.Elements[i]
		t, ok := d.str(e, `each tiding of the "vox-cast" of a rite`)
		switch {
		case !ok:
		case t != VoxProgress && t != VoxSuccess:
			d.heresy(e, "a vox-cast proclaims only %q or %q; %q is no tiding the codex knows — tidings of "+
				"failure need no vox-cast, for they are proclaimed unbidden", VoxProgress, VoxSuccess, t)
		case slices.Contains(d.rite.VoxCasts, t):
			d.heresy(e, "the vox-cast %q is written twice; each tiding is proclaimed at most once, whatever "+
				"its place in the list — keep only one", t)
		default:
			d.rite.VoxCasts = append(d.rite.VoxCasts, t)
		}
	}
}

// decrees reads the decrees of an inscription: one plain word in an edict,
// an aspect map in a rite of aspects (always an object in pattern Mark II).
func (d *decoder) decrees(v *hujson.Value, what string) AspectMap[string] {
	if d.rite.MarkII && !d.rite.Edict {
		return keyedMap(d, v, what, d.aspectKeying(), false, d.str)
	}
	return aspectMap(d, v, what, false, nil, func(v *hujson.Value, what string) (string, bool) {
		return d.str(v, what)
	})
}

// designation reads the designation of what, one of family; "" when it
// bears none the codex knows, which is denounced.
func (d *decoder) designation(v *hujson.Value, fs fields, what string, family []string) string {
	m, ok := fs.get("designation")
	if !ok {
		d.heresy(v, "%s bears no \"designation\"; it must name the shape it takes, one of %s", what, quoteAll(family))
		return ""
	}
	s, ok := d.str(m.val, `the "designation" of `+what)
	if !ok {
		return ""
	}
	if !slices.Contains(family, s) {
		d.heresy(m.val, "%q is no designation the codex knows for %s; write one of %s", s, what, quoteAll(family))
		return ""
	}
	return s
}

// designated keeps the members of what, of designation, that the codex
// knows there: the common keys and the designation's own. A key of another
// designation of family, and any other key, is denounced.
func (d *decoder) designated(all fields, what, designation string, common, family []string) fields {
	known := slices.Concat(common, designationKeys[designation])
	var fs fields
	for _, m := range all {
		switch owner := ownerOf(m.name, family); {
		case slices.Contains(known, m.name):
			fs = append(fs, m)
		case owner != "" && designation != DesignationMachineSpirit && owner == DesignationMachineSpirit:
			d.heresy(m.key, "%s is %s, which the servitor reads and sets by itself; it bears no %q, which a "+
				"machine-spirit alone bears for the words that read or set it — strike it, or make the subject "+
				"a machine-spirit", what, article(designation), m.name)
		case owner != "":
			d.heresy(m.key, "the key %q belongs to %s, yet %s is %s; strike it, or write the keys the codex "+
				"knows for %s: %s", m.name, article(owner), what, article(designation), article(designation),
				quoteAll(withoutAliases(known)))
		default:
			d.unknownKey(m, what, known)
		}
	}
	return fs
}

// ownerOf names the first designation of family that holds key.
func ownerOf(key string, family []string) string {
	for _, d := range family {
		if slices.Contains(designationKeys[d], key) {
			return d
		}
	}
	return ""
}

// article names a designation with its article: "a sanctum", "an incantation".
func article(designation string) string {
	if designation == DesignationIncantation {
		return "an " + designation
	}
	return "a " + designation
}

// name reads the name of a subject or observance (noun), which must be fit
// and unique among its kind; "" when it cannot be read.
func (d *decoder) name(v *hujson.Value, fs fields, what, noun string, seen *[]named) string {
	m, ok := fs.get("name")
	if !ok {
		d.heresy(v, "%s bears no \"name\"; every %s is named, that the tidings and the chronicle may name it "+
			"and the rite may speak of it", what, noun)
		return ""
	}
	n, ok := d.str(m.val, `the "name" of `+what)
	switch {
	case !ok:
		return ""
	case !ValidName(n):
		d.heresy(m.val, "%q is no fit name for a %s: it must begin with a letter or digit and continue only "+
			"with letters, digits, '.', '_' or '-', as the name of an aspect does", n, noun)
	case slices.ContainsFunc(*seen, func(o named) bool { return o.name == n }):
		d.heresy(m.val, "the %s %q is named twice; each %s of a rite bears a name of its own, lest the servitor "+
			"mistake one for the other", noun, n, noun)
	}
	*seen = append(*seen, named{name: n, at: m.val})
	return n
}
