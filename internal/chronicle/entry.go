package chronicle

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"maps"
	"time"

	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// Entry is one invocation as the chronicle records it, one line of JSON.
type Entry struct {
	At           time.Time          `json:"at"` // RFC 3339, whole seconds, UTC
	Rite         string             `json:"rite"`
	Aspect       string             `json:"aspect"`
	Former       string             `json:"former,omitempty"` // "" when the former aspect was unknown
	Inscriptions map[string]string  `json:"inscriptions"`
	Verdict      invocation.Verdict `json:"verdict"`
	FellAt       *Verse             `json:"fell_at,omitempty"` // nil when no step fell
	Heresy       string             `json:"heresy,omitempty"`  // why the step fell
	// Reversions are the steps undone, from the fallen step back to the
	// first.
	Reversions []Reversion `json:"reversions,omitempty"`
}

// Verse names a step as the chronicle writes it: its verse and its own key
// holding its target, e.g. {"verse": 3, "incantation": "source …/hooks"}.
type Verse struct {
	Number int    // counted from one
	Key    string // the step's own key, e.g. "tether"
	Target string // what the step's own key held, rendered and resolved
}

// Reversion is the verdict upon undoing one step: triumph, or faltered when
// the reversion fell.
type Reversion struct {
	Verse   int                `json:"verse"`
	Verdict invocation.Verdict `json:"verdict"`
}

// New records an invocation of rite performed with opts that ended in out,
// at the moment at.
func New(at time.Time, rite string, opts invocation.Options, out invocation.Outcome) Entry {
	e := Entry{
		At:           at.UTC().Truncate(time.Second),
		Rite:         rite,
		Aspect:       opts.Aspect,
		Former:       opts.Former,
		Inscriptions: map[string]string{},
		Verdict:      out.Verdict,
	}
	maps.Copy(e.Inscriptions, opts.Inscriptions)
	if f := out.Fell; f != nil {
		e.FellAt = &Verse{Number: f.Number, Key: f.Kind.Key(), Target: f.Target}
		if f.Heresy != nil {
			e.Heresy = f.Heresy.Error()
		}
	}
	for _, r := range out.Reversions {
		v := invocation.Triumph
		if r.Heresy != nil {
			v = invocation.Faltered
		}
		e.Reversions = append(e.Reversions, Reversion{Verse: r.Number, Verdict: v})
	}
	return e
}

// MarshalJSONTo writes the verse first, then the step's own key.
func (v Verse) MarshalJSONTo(enc *jsontext.Encoder) error {
	for _, t := range []jsontext.Token{
		jsontext.BeginObject,
		jsontext.String("verse"), jsontext.Int(int64(v.Number)),
		jsontext.String(v.Key), jsontext.String(v.Target),
		jsontext.EndObject,
	} {
		if err := enc.WriteToken(t); err != nil {
			return err
		}
	}
	return nil
}

// errStrangeVerse denounces a fell_at that names no single step.
var errStrangeVerse = errors.New("the chronicle names no single step in this verse")

// UnmarshalJSONFrom reads a verse and the one key beside it, whatever step
// it names.
func (v *Verse) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var fields map[string]jsontext.Value
	if err := json.UnmarshalDecode(dec, &fields); err != nil {
		return err
	}
	var got Verse
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
