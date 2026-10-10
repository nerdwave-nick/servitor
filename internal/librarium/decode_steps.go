package librarium

import (
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/tailscale/hujson"
)

var stepKeys = map[Kind][]string{
	KindSanctum:       {"sanctum", "ward", "glyph", "closing-glyph", "consecrate", "scripture"},
	KindTranscription: {"transcription", "scripture", "seal", "zeal", "force"},
	KindTether:        {"tether", "anchor", "zeal", "force"},
	KindIncantation:   {"incantation", "tongue", "patience", "reversion"},
	KindLitany:        {"litany", "offerings", "tongue", "patience", "reversion"},
	KindVoxCast:       {"vox-cast"},
}

// StepKeys returns the keys a step of kind k may hold, its own key first.
func StepKeys(k Kind) []string { return append([]string{}, stepKeys[k]...) }

func kindOf(key string) Kind {
	for _, k := range Kinds {
		if k.Key() == key {
			return k
		}
	}
	return 0
}

// decodeStep decodes the step at verse; nil when its kind cannot be told.
func (d *decoder) decodeStep(v *hujson.Value, verse int) Step {
	obj, ok := v.Value.(*hujson.Object)
	if !ok {
		d.wrongForm(v, fmt.Sprintf("verse %d of the liturgy", verse), "an object naming one step")
		return nil
	}
	var kind Kind
	for _, m := range obj.Members {
		k := kindOf(m.Name.Value.(hujson.Literal).String())
		switch {
		case k == 0:
		case kind != 0:
			d.heresy(&m.Name, "a step may be of only one kind, yet verse %d names both %q and %q; split it into "+
				"two steps", verse, kind.Key(), k.Key())
			return nil
		default:
			kind = k
		}
	}
	if kind == 0 {
		d.heresy(v, "verse %d of the liturgy names no kind of step; a step must hold exactly one of the keys %s",
			verse, quoteAll(keysOfKinds()))
		return nil
	}
	what := fmt.Sprintf("the %s of verse %d", kind.Key(), verse)
	fs, _ := d.object(v, what, stepKeys[kind])
	st := stepDecoder{d: d, v: v, fs: fs, what: what}
	own, _ := fs.get(kind.Key())
	switch kind {
	case KindSanctum:
		return st.sanctum(own.val)
	case KindTranscription:
		return st.transcription(own.val)
	case KindTether:
		return st.tether(own.val)
	case KindIncantation:
		return &Incantation{Command: st.commands(own.val, what), Utterance: st.utterance()}
	case KindLitany:
		return st.litany(own.val)
	}
	return st.voxCast(own.val)
}

func keysOfKinds() []string {
	keys := make([]string, len(Kinds))
	for i, k := range Kinds {
		keys[i] = k.Key()
	}
	return keys
}

// stepDecoder decodes the fields of one step.
type stepDecoder struct {
	d    *decoder
	v    *hujson.Value
	fs   fields
	what string
}

func (s stepDecoder) field(key string) string { return fmt.Sprintf("the %q of %s", key, s.what) }

// required returns the field key, denouncing its absence.
func (s stepDecoder) required(key, why string) (member, bool) {
	m, ok := s.fs.get(key)
	if !ok {
		s.d.heresy(s.v, "%s holds no %q; %s", s.what, key, why)
	}
	return m, ok
}

func (s stepDecoder) sanctum(own *hujson.Value) *Sanctum {
	st := &Sanctum{}
	st.Vessel, _ = s.d.templated(own, s.what+"'s vessel", false)
	if m, ok := s.fs.get("ward"); ok {
		if st.Ward, ok = s.d.str(m.val, s.field("ward")); ok &&
			(st.Ward == "" || strings.IndexFunc(st.Ward, unicode.IsSpace) >= 0) {
			s.d.heresy(m.val, "the ward %q is unfit: a ward may not be empty and may hold no whitespace, for "+
				"it is written into the sanctum's markers", st.Ward)
		}
	}
	for key, dst := range map[string]*string{"glyph": &st.Glyph, "closing-glyph": &st.ClosingGlyph} {
		if m, ok := s.fs.get(key); ok {
			if *dst, ok = s.d.str(m.val, s.field(key)); ok && (strings.ContainsAny(*dst, "\r\n") ||
				key == "glyph" && strings.TrimSpace(*dst) == "") {
				s.d.heresy(m.val, "%s must be a single line, and a glyph may not be empty, for the sanctum's "+
					"markers are written with it", s.field(key))
			}
		}
	}
	if _, closing := st.Glyphs(); closing != "" && strings.Contains(st.WardFor(s.d.rite.Name), closing) {
		at := s.v
		if m, ok := s.fs.get("ward"); ok {
			at = m.val
		}
		s.d.heresy(at, "the ward %q holds the closing glyph %q, which would end the sanctum's marker in "+
			"its midst", st.WardFor(s.d.rite.Name), closing)
	}
	if m, ok := s.fs.get("consecrate"); ok {
		st.Consecrate = s.d.boolean(m.val, s.field("consecrate"))
	}
	if m, ok := s.required("scripture", "a sanctum must declare the scripture it keeps for each aspect"); ok {
		st.Scripture = s.scripture(m.val, false)
	}
	return st
}

func (s stepDecoder) transcription(own *hujson.Value) *Transcription {
	st := &Transcription{Zeal: s.zeal()}
	st.Vessel, _ = s.d.templated(own, s.what+"'s vessel", false)
	if m, ok := s.required("scripture", "a transcription must declare the scripture of its vessel for "+
		"each aspect, or null where the vessel shall not be"); ok {
		st.Scripture = s.scripture(m.val, true)
	}
	if m, ok := s.fs.get("seal"); ok {
		st.Seal = s.seal(m.val)
	}
	return st
}

func (s stepDecoder) tether(own *hujson.Value) *Tether {
	st := &Tether{Zeal: s.zeal()}
	st.Name, _ = s.d.templated(own, s.what+"'s bound name", false)
	if m, ok := s.required("anchor", "a tether must declare where it leads for each aspect, or null where "+
		"it shall be unbound"); ok {
		st.Anchor = aspectMap(s.d, m.val, s.field("anchor"), true, nil, func(v *hujson.Value, what string) (Anchor, bool) {
			if lit, ok := literal(v); ok && lit.Kind() == 'n' {
				return Anchor{Null: true}, true
			}
			p, ok := s.d.templated(v, what, false)
			return Anchor{Path: p}, ok
		})
	}
	return st
}

func (s stepDecoder) litany(own *hujson.Value) *Litany {
	st := &Litany{Scroll: s.commands(own, s.what+"'s scroll"), Utterance: s.utterance()}
	m, ok := s.fs.get("offerings")
	if !ok {
		return st
	}
	arr, ok := m.val.Value.(*hujson.Array)
	if !ok {
		s.d.wrongForm(m.val, s.field("offerings"), "a list of offerings")
		return st
	}
	st.Offerings = []AspectMap[string]{}
	for i := range arr.Elements {
		what := fmt.Sprintf("offering %d of %s", i+1, s.what)
		st.Offerings = append(st.Offerings, aspectMap(s.d, &arr.Elements[i], what, true, nil, s.text(true)))
	}
	return st
}

func (s stepDecoder) voxCast(own *hujson.Value) *VoxCast {
	t, ok := s.d.str(own, s.what)
	if ok && t != VoxProgress && t != VoxSuccess {
		s.d.heresy(own, "a vox-cast proclaims only %q or %q; %q is no tiding the codex knows — tidings of "+
			"failure need no step, for they are vox-cast unbidden", VoxProgress, VoxSuccess, t)
	}
	return &VoxCast{Tidings: t}
}

// commands decodes the command of an incantation or the scroll of a litany.
func (s stepDecoder) commands(v *hujson.Value, what string) AspectMap[string] {
	return aspectMap(s.d, v, what, true, nil, s.text(false))
}

func (s stepDecoder) text(mayBeEmpty bool) func(*hujson.Value, string) (string, bool) {
	return func(v *hujson.Value, what string) (string, bool) { return s.d.templated(v, what, mayBeEmpty) }
}

func (s stepDecoder) utterance() Utterance {
	var u Utterance
	if m, ok := s.fs.get("tongue"); ok {
		u.Tongue, _ = s.d.word(m.val, s.field("tongue"))
	}
	if m, ok := s.fs.get("patience"); ok {
		u.Patience = s.d.patience(m.val, s.field("patience"))
	}
	if m, ok := s.fs.get("reversion"); ok {
		u.Reversion = aspectMap(s.d, m.val, s.field("reversion"), true, nil, s.text(true))
	}
	return u
}

// zeal decodes "zeal" or its alias "force".
func (s stepDecoder) zeal() bool {
	z, hasZeal := s.fs.get("zeal")
	f, hasForce := s.fs.get("force")
	switch {
	case hasZeal && hasForce:
		s.d.heresy(f.key, "\"zeal\" and \"force\" are one and the same; %s may bear only one of them — "+
			"write only \"zeal\"", s.what)
		return s.d.boolean(z.val, s.field("zeal"))
	case hasForce:
		return s.d.boolean(f.val, s.field("force"))
	case hasZeal:
		return s.d.boolean(z.val, s.field("zeal"))
	}
	return false
}

var sealRe = regexp.MustCompile(`^[0-7]{3,4}$`)

func (s stepDecoder) seal(v *hujson.Value) *fs.FileMode {
	t, ok := s.d.str(v, s.field("seal"))
	if !ok {
		return nil
	}
	if !sealRe.MatchString(t) {
		s.d.heresy(v, "%q is no seal: a seal is written as three or four octal numerals of the old "+
			"cogitators, such as \"0644\" or \"0755\"", t)
		return nil
	}
	n, _ := strconv.ParseUint(t, 8, 32)
	mode := fs.FileMode(n)
	return &mode
}
