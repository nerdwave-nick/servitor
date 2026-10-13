package sanction

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strconv"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// Unit is the canonical exalted unit of a rite: everything its exalted
// verses do, for every aspect. Its JSON is the shape of the sanction.
type Unit struct {
	Pattern string `json:"pattern"` // Pattern; PatternMarkI for a sanction of old
	UID     int    `json:"uid"`     // the master's
	Rite    string `json:"rite"`
	Edict   bool   `json:"edict"` // an edict has one plan, of aspect ""
	// Environment is what exalted speech hears from its environment.
	Environment Environment `json:"environment"`
	// Aspects holds one plan per aspect, in the rite's order.
	Aspects []Plan `json:"aspects"`
}

// Environment names the variables of the environment exalted speech hears:
// those the unit declares, and the pattern of those the exalted servitor
// grants besides (Pattern, which grants SERVITOR_OMEN too). A new pattern
// lapses every sanction.
type Environment struct {
	Pattern string   `json:"pattern"` // Pattern; PatternMarkI for a sanction of old
	Keys    []string `json:"keys"`    // sorted
}

// Plan is what the exalted verses do for one aspect.
type Plan struct {
	Aspect string `json:"aspect"` // "" for an edict
	// Inscriptions are the words of every inscription spliced into the
	// exalted verses, from their decrees for the aspect ("" when none).
	Inscriptions map[string]string `json:"inscriptions"`
	Liturgy      []Verse           `json:"liturgy,omitzero"` // the exalted verses alone, in written order
	// Subjects and Observances are those of a state rite of pattern Mark
	// II: its exalted subjects, and the exalted observances the aspect
	// admits by "to", each in written order. Such a plan holds no Liturgy.
	Subjects    []Verse `json:"subjects,omitzero"`
	Observances []Verse `json:"observances,omitzero"`
}

// Verse is one exalted verse, resolved for the aspect of its plan. Only the
// fields of its kind are written. It is also one exalted subject or
// observance of a state rite of pattern Mark II (see unit_markii.go),
// named by Subject or Observance in place of its verse: a sanctum,
// transcription or tether subject bears the fields of its kind, an
// observance those of its incantation or litany, and a machine-spirit,
// whose Kind is none, its Omen and Enact words beside its speech.
type Verse struct {
	Verse int // counted from one, as written; 0 for a subject or an observance
	Kind  librarium.Kind
	// Target is what the step's own key holds, rendered: the vessel of a
	// sanctum or transcription (from the root), the bound name of a tether,
	// the command of an incantation, the scroll of a litany as written.
	Target string

	Ward, Glyph, ClosingGlyph string // a sanctum's, resolved
	Consecrate                bool   // a sanctum's

	// Scripture is what a sanctum or transcription places in its vessel:
	// inline scripture rendered, or the bytes of its tome (illuminated when
	// it says so). Null removes a transcription's vessel.
	Scripture []byte
	Null      bool
	Seal      *fs.FileMode // a transcription's; nil when unwritten
	Zeal      bool         // a transcription's or tether's

	// Anchor is where a tether leads, resolved; Null unbinds it.
	Anchor string

	Scroll    []byte   // the bytes of a litany's scroll
	Offerings []string // a litany's, rendered

	// The speech of an incantation or litany.
	Tongue    string        // absolute
	Patience  time.Duration // resolved
	Reversion string        // rendered; "" reverts nothing
	Relics    []Relic       // in written order

	// Subject and Observance name the subject or observance of a state rite
	// the item is; Place is a subject's index among the rite's subjects.
	Subject, Observance string
	Place               int
	// Omen is the omen a machine-spirit is set to for the plan's aspect,
	// and Enact the words that set it into each omen it declares,
	// rendered for that aspect: a rollback enacts the omen read before,
	// which may be any of them.
	Omen  string
	Enact map[string]string
	// From are the former aspects an observance admits, nil when it admits
	// every one; Introit and Recessional the places of the subjects that
	// name it so.
	From                 []string
	Introit, Recessional []int
	// Auspex is the privileged auspex of an exalted machine-spirit, which
	// the exalted servitor alone speaks; nil when its auspex is common, or
	// it has none.
	Auspex *Auspex
}

// Auspex is the privileged auspex of an exalted machine-spirit, resolved
// for the aspect of its plan: its scan, or its trials keyed by every
// declared omen that bears one (each hearing its omen as {{omen}}), and
// its patience. NamesAspects tells that the spirit declares no omens, so
// that its scan prints the names of aspects, each an omen.
type Auspex struct {
	Scan         string
	Trials       map[string]string
	Patience     time.Duration
	NamesAspects bool
}

// Relic is one scripture exalted speech reads beside its scroll.
type Relic struct {
	Relic     string `json:"relic"`     // as written: a path within the rite's hall
	Scripture []byte `json:"scripture"` // its bytes
}

// Sigil is the sigil of the whole unit: the SHA-256 of its canonical JSON.
func (u *Unit) Sigil() string { return digest(u.Canonical()) }

// SigilOf is the sigil of the unit narrowed to the plan of aspect ("" for an
// edict, whose one plan makes it the sigil of the whole); ok is false for
// an aspect the unit holds no plan for.
func (u *Unit) SigilOf(aspect string) (sigil string, ok bool) {
	n, ok := u.Narrow(aspect)
	if !ok {
		return "", false
	}
	return n.Sigil(), true
}

// Narrow is the unit holding the plan of aspect alone.
func (u *Unit) Narrow(aspect string) (*Unit, bool) {
	for _, p := range u.Aspects {
		if p.Aspect == aspect {
			n := *u
			n.Aspects = []Plan{p}
			return &n, true
		}
	}
	return nil, false
}

// Canonical is the unit written as canonical JSON (RFC 8785).
func (u *Unit) Canonical() []byte {
	b, err := json.Marshal(u.written(), json.Deterministic(true))
	if err != nil {
		panic(fmt.Sprintf("sanction: a unit that cannot be written: %v", err)) // every field is writable
	}
	v := jsontext.Value(b)
	if err := v.Canonicalize(); err != nil {
		panic(fmt.Sprintf("sanction: a unit that cannot be canonicalized: %v", err))
	}
	return v
}

// written is the unit as it is written, with no list or map null: the plan
// of a state rite of pattern Mark II holds its subjects and observances,
// every other its liturgy.
func (u *Unit) written() *Unit {
	w := *u
	w.Environment.Keys = orEmpty(w.Environment.Keys)
	w.Aspects = slices.Clone(orEmpty(w.Aspects))
	for i, p := range w.Aspects {
		if p.Inscriptions == nil {
			p.Inscriptions = map[string]string{}
		}
		if w.State() {
			p.Liturgy, p.Subjects, p.Observances = nil, orEmpty(p.Subjects), orEmpty(p.Observances)
		} else {
			p.Liturgy, p.Subjects, p.Observances = orEmpty(p.Liturgy), nil, nil
		}
		w.Aspects[i] = p
	}
	return &w
}

// State tells that the unit is that of a state rite, whose plans hold
// subjects and observances.
func (u *Unit) State() bool { return u.Pattern == Pattern && !u.Edict }

func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// The writings of each kind of verse.
type (
	sanctumVerse struct {
		Verse        int    `json:"verse"`
		Sanctum      string `json:"sanctum"`
		Ward         string `json:"ward"`
		Glyph        string `json:"glyph"`
		ClosingGlyph string `json:"closing-glyph"`
		Consecrate   bool   `json:"consecrate"`
		Scripture    []byte `json:"scripture"`
	}
	transcriptionVerse struct {
		Verse         int     `json:"verse"`
		Transcription string  `json:"transcription"`
		Scripture     *[]byte `json:"scripture"` // null removes the vessel
		Seal          *string `json:"seal"`      // octal, e.g. "0644"
		Zeal          bool    `json:"zeal"`
	}
	tetherVerse struct {
		Verse  int     `json:"verse"`
		Tether string  `json:"tether"`
		Anchor *string `json:"anchor"` // null unbinds it
		Zeal   bool    `json:"zeal"`
	}
	incantationVerse struct {
		Verse       int     `json:"verse"`
		Incantation string  `json:"incantation"`
		Tongue      string  `json:"tongue"`
		Patience    string  `json:"patience"`
		Reversion   string  `json:"reversion"`
		Relics      []Relic `json:"relics"`
	}
	litanyVerse struct {
		Verse     int      `json:"verse"`
		Litany    string   `json:"litany"`
		Scroll    []byte   `json:"scroll"`
		Offerings []string `json:"offerings"`
		Tongue    string   `json:"tongue"`
		Patience  string   `json:"patience"`
		Reversion string   `json:"reversion"`
		Relics    []Relic  `json:"relics"`
	}
)

// MarshalJSON writes the verse with its number, its own key and the fields
// of its kind; a subject or observance as unit_markii.go tells.
func (v Verse) MarshalJSON() ([]byte, error) {
	if v.Subject != "" || v.Observance != "" {
		return v.marshalItem()
	}
	var w any
	switch v.Kind {
	case librarium.KindSanctum:
		w = sanctumVerse{v.Verse, v.Target, v.Ward, v.Glyph, v.ClosingGlyph, v.Consecrate, orEmpty(v.Scripture)}
	case librarium.KindTranscription:
		t := transcriptionVerse{Verse: v.Verse, Transcription: v.Target, Zeal: v.Zeal}
		if !v.Null {
			sc := orEmpty(v.Scripture)
			t.Scripture = &sc
		}
		if v.Seal != nil {
			s := fmt.Sprintf("%04o", uint32(*v.Seal))
			t.Seal = &s
		}
		w = t
	case librarium.KindTether:
		t := tetherVerse{Verse: v.Verse, Tether: v.Target, Zeal: v.Zeal}
		if !v.Null {
			t.Anchor = &v.Anchor
		}
		w = t
	case librarium.KindIncantation:
		w = incantationVerse{v.Verse, v.Target, v.Tongue, v.Patience.String(), v.Reversion, orEmpty(v.Relics)}
	case librarium.KindLitany:
		w = litanyVerse{v.Verse, v.Target, orEmpty(v.Scroll), orEmpty(v.Offerings), v.Tongue, v.Patience.String(),
			v.Reversion, orEmpty(v.Relics)}
	default:
		return nil, fmt.Errorf("a %s is never exalted", v.Kind.Key())
	}
	return json.Marshal(w, json.Deterministic(true))
}

// UnmarshalJSON reads a verse strictly: exactly one own key of an exalted
// kind, and no field its kind does not write.
func (v *Verse) UnmarshalJSON(data []byte) error {
	var keys map[string]jsontext.Value
	if err := json.Unmarshal(data, &keys); err != nil {
		return err
	}
	if _, item := keys["designation"]; item {
		return v.unmarshalItem(data, keys)
	}
	var kinds []librarium.Kind
	for _, k := range librarium.Kinds {
		if _, ok := keys[k.Key()]; ok {
			kinds = append(kinds, k)
		}
	}
	if len(kinds) != 1 {
		return fmt.Errorf("a verse of a sanction bears exactly one own key, yet it bears %d (%v)", len(kinds),
			slices.Sorted(maps.Keys(keys)))
	}
	strict := json.RejectUnknownMembers(true)
	*v = Verse{Kind: kinds[0]}
	var err error
	switch v.Kind {
	case librarium.KindSanctum:
		var w sanctumVerse
		err = json.Unmarshal(data, &w, strict)
		v.Verse, v.Target, v.Ward, v.Glyph, v.ClosingGlyph, v.Consecrate, v.Scripture =
			w.Verse, w.Sanctum, w.Ward, w.Glyph, w.ClosingGlyph, w.Consecrate, w.Scripture
	case librarium.KindTranscription:
		var w transcriptionVerse
		if err = json.Unmarshal(data, &w, strict); err == nil {
			v.Verse, v.Target, v.Zeal, v.Null = w.Verse, w.Transcription, w.Zeal, w.Scripture == nil
			if w.Scripture != nil {
				v.Scripture = *w.Scripture
			}
			if w.Seal != nil {
				seal, perr := strconv.ParseUint(*w.Seal, 8, 32)
				m := fs.FileMode(seal)
				v.Seal, err = &m, perr
			}
		}
	case librarium.KindTether:
		var w tetherVerse
		if err = json.Unmarshal(data, &w, strict); err == nil {
			v.Verse, v.Target, v.Zeal, v.Null = w.Verse, w.Tether, w.Zeal, w.Anchor == nil
			if w.Anchor != nil {
				v.Anchor = *w.Anchor
			}
		}
	case librarium.KindIncantation:
		var w incantationVerse
		if err = json.Unmarshal(data, &w, strict); err == nil {
			v.Verse, v.Target, v.Tongue, v.Reversion, v.Relics = w.Verse, w.Incantation, w.Tongue, w.Reversion, w.Relics
			v.Patience, err = time.ParseDuration(w.Patience)
		}
	case librarium.KindLitany:
		var w litanyVerse
		if err = json.Unmarshal(data, &w, strict); err == nil {
			v.Verse, v.Target, v.Scroll, v.Offerings, v.Tongue, v.Reversion, v.Relics =
				w.Verse, w.Litany, w.Scroll, w.Offerings, w.Tongue, w.Reversion, w.Relics
			v.Patience, err = time.ParseDuration(w.Patience)
		}
	default:
		return fmt.Errorf("a %s is never exalted", v.Kind.Key())
	}
	return err
}
