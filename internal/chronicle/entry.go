package chronicle

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"maps"
	"slices"
	"time"

	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// Entry is one invocation as the chronicle records it, one line of JSON.
type Entry struct {
	At           time.Time          `json:"at"` // RFC 3339, whole seconds, UTC
	Rite         string             `json:"rite"`
	Edict        bool               `json:"edict,omitzero"`   // an edict was proclaimed
	Aspect       string             `json:"aspect,omitempty"` // "" only for an edict
	Former       string             `json:"former,omitempty"` // "" when the former aspect was unknown, and for an edict
	Inscriptions map[string]string  `json:"inscriptions"`
	Verdict      invocation.Verdict `json:"verdict"`
	FellAt       *Verse             `json:"fell_at,omitempty"` // nil when no step fell
	Heresy       string             `json:"heresy,omitempty"`  // why the step fell
	// Reversions are the steps undone, from the fallen step back to the
	// first.
	Reversions []Reversion `json:"reversions,omitempty"`
	// Changed names, for a state rite of pattern Mark II alone, the
	// subjects it set and proved, in written order; [] when it changed
	// nothing, and absent for every other rite.
	Changed []string `json:"changed,omitzero"`
	// Intercessor and Sigil name, for an invocation with exalted verses
	// alone, the intercessor that vouched for its master and the first
	// twelve letters of the sigil of the plan the exalted servitor
	// performed.
	Intercessor string `json:"intercessor,omitempty"`
	Sigil       string `json:"sigil,omitempty"`
}

// Verse names a step as the chronicle writes it: its verse and its own key
// holding its target, e.g. {"verse": 3, "incantation": "source …/hooks"};
// or a subject or an observance of a state rite by its name alone,
// {"subject": "smt"}, {"observance": "herald-silence"}.
type Verse struct {
	Number     int    // counted from one; zero for a subject or an observance
	Key        string // the step's own key, e.g. "tether"
	Target     string // what the step's own key held, rendered and resolved
	Subject    string // the subject's name; "" for a step
	Observance string // the observance's name; "" for a step
}

// Reversion is the verdict upon undoing one step, one subject or one
// observance: triumph, or faltered when the reversion fell.
type Reversion struct {
	Verse      int                `json:"verse,omitzero"`
	Subject    string             `json:"subject,omitzero"`
	Observance string             `json:"observance,omitzero"`
	Verdict    invocation.Verdict `json:"verdict"`
}

// New records an invocation of rite performed with opts that ended in out,
// at the moment at.
func New(at time.Time, rite string, opts invocation.Options, out invocation.Outcome) Entry {
	e := Entry{
		At:           at.UTC().Truncate(time.Second),
		Rite:         rite,
		Edict:        opts.Edict,
		Aspect:       opts.Aspect,
		Former:       opts.Former,
		Inscriptions: map[string]string{},
		Verdict:      out.Verdict,
	}
	maps.Copy(e.Inscriptions, opts.Inscriptions)
	if f := out.Fell; f != nil {
		e.FellAt = &Verse{Number: f.Number, Key: f.Kind.Key(), Target: f.Target}
		if f.Subject != "" || f.Observance != "" {
			e.FellAt = &Verse{Subject: f.Subject, Observance: f.Observance}
		}
		if f.Heresy != nil {
			e.Heresy = f.Heresy.Error()
		}
	}
	for _, r := range out.Reversions {
		v := invocation.Triumph
		if r.Heresy != nil {
			v = invocation.Faltered
		}
		e.Reversions = append(e.Reversions, Reversion{Verse: r.Number, Subject: r.Subject, Observance: r.Observance,
			Verdict: v})
	}
	e.Changed = slices.Clone(out.Changed)
	return e
}

// subjectKey and observanceKey are the keys naming a subject or an
// observance in place of a verse.
const (
	subjectKey    = "subject"
	observanceKey = "observance"
)

// MarshalJSONTo writes the verse first, then the step's own key; a subject
// or an observance by its name alone.
func (v Verse) MarshalJSONTo(enc *jsontext.Encoder) error {
	tokens := []jsontext.Token{
		jsontext.BeginObject,
		jsontext.String("verse"), jsontext.Int(int64(v.Number)),
		jsontext.String(v.Key), jsontext.String(v.Target),
		jsontext.EndObject,
	}
	if key, name := v.named(); key != "" {
		tokens = []jsontext.Token{jsontext.BeginObject, jsontext.String(key), jsontext.String(name), jsontext.EndObject}
	}
	for _, t := range tokens {
		if err := enc.WriteToken(t); err != nil {
			return err
		}
	}
	return nil
}

// named is the key and the name of a subject or an observance; "" for a
// step.
func (v Verse) named() (key, name string) {
	switch {
	case v.Subject != "":
		return subjectKey, v.Subject
	case v.Observance != "":
		return observanceKey, v.Observance
	}
	return "", ""
}

// errStrangeVerse denounces a fell_at that names no single step.
var errStrangeVerse = errors.New("the chronicle names no single step in this verse")

// UnmarshalJSONFrom reads a verse and the one key beside it, whatever step
// it names, or a subject or an observance named alone.
func (v *Verse) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var fields map[string]jsontext.Value
	if err := json.UnmarshalDecode(dec, &fields); err != nil {
		return err
	}
	var got Verse
	for key, name := range map[string]*string{subjectKey: &got.Subject, observanceKey: &got.Observance} {
		if raw, ok := fields[key]; ok && len(fields) == 1 {
			if err := json.Unmarshal(raw, name); err != nil {
				return err
			}
			*v = got
			return nil
		}
	}
	for k, raw := range fields {
		if k == "verse" {
			if err := json.Unmarshal(raw, &got.Number); err != nil {
				return err
			}
			continue
		}
		if got.Key != "" {
			return errStrangeVerse
		}
		got.Key = k
		if err := json.Unmarshal(raw, &got.Target); err != nil {
			return err
		}
	}
	*v = got
	return nil
}
