package augury

import (
	"bytes"
	"encoding/json"

	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// MarshalJSON writes the augury in the shape of the codex: rite, aspect,
// standing, desecrated, former (only when desecrated), inscriptions,
// last_rite (null when never invoked), omens and taint (lists, never null).
func (a Augury) MarshalJSON() ([]byte, error) {
	type binharic struct {
		Rite         string            `json:"rite"`
		Aspect       string            `json:"aspect"`
		Standing     Standing          `json:"standing"`
		Desecrated   bool              `json:"desecrated"`
		Former       string            `json:"former,omitempty"`
		Inscriptions map[string]string `json:"inscriptions"`
		LastRite     *LastRite         `json:"last_rite"`
		Omens        []Omen            `json:"omens"`
		Taint        []Taint           `json:"taint"`
	}
	b := binharic{Rite: a.Rite, Aspect: a.Aspect, Standing: a.Standing, Desecrated: a.Desecrated,
		Inscriptions: a.Inscriptions, LastRite: a.LastRite, Omens: a.Omens, Taint: a.Taint}
	if a.Desecrated {
		b.Former = a.Former
	}
	if b.Inscriptions == nil {
		b.Inscriptions = map[string]string{}
	}
	if b.Omens == nil {
		b.Omens = []Omen{}
	}
	if b.Taint == nil {
		b.Taint = []Taint{}
	}
	return json.Marshal(b)
}

// MarshalJSON writes a step's omen as its verse, its own key and the aspect
// ({"verse": 2, "tether": "…", "aspect": "porpl"}), and the auspex's as
// {"auspex": true, "aspect": "porpl"}.
func (o Omen) MarshalJSON() ([]byte, error) {
	if o.Auspex {
		return json.Marshal(struct {
			Auspex bool   `json:"auspex"`
			Aspect string `json:"aspect"`
		}{true, o.Aspect})
	}
	return versed(o.Verse, "aspect", o.Aspect)
}

// MarshalJSON writes a taint as the sanctum's verse and its own key
// ({"verse": 1, "sanctum": "…"}).
func (t Taint) MarshalJSON() ([]byte, error) { return versed(t.Verse) }

// versed writes {"verse": n, "<key>": target} followed by the further
// fields, given as name, value pairs.
func versed(v invocation.Verse, fields ...string) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString(`{"verse":`)
	n, _ := json.Marshal(v.Number)
	b.Write(n)
	pairs := append([]string{v.Kind.Key(), v.Target}, fields...)
	for i := 0; i+1 < len(pairs); i += 2 {
		key, _ := json.Marshal(pairs[i])
		val, _ := json.Marshal(pairs[i+1])
		b.WriteByte(',')
		b.Write(key)
		b.WriteByte(':')
		b.Write(val)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}
