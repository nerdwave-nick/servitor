package augury

import (
	"bytes"
	"encoding/json"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// MarshalJSON writes the augury in the shape of the codex: rite, edict
// (false), aspect, standing, sanction, auspex, desecrated, former (only when
// desecrated), inscriptions, canon, last_rite (null when never invoked),
// fitting (a state rite's alone), omens and taint (lists, never null). An edict's is rite, edict (true),
// standing, sanction, auspex (always ""), inscriptions, canon and last_rite alone.
func (a Augury) MarshalJSON() ([]byte, error) {
	inscriptions := a.Inscriptions
	if inscriptions == nil {
		inscriptions = map[string]string{}
	}
	canon := a.Canon
	if canon == nil {
		canon = []Declaration{}
	}
	if a.Edict {
		return json.Marshal(struct {
			Rite         string            `json:"rite"`
			Edict        bool              `json:"edict"`
			Standing     Standing          `json:"standing"`
			Sanction     sanction.Standing `json:"sanction"`
			Auspex       Heed              `json:"auspex"`
			Inscriptions map[string]string `json:"inscriptions"`
			Canon        []Declaration     `json:"canon"`
			LastRite     *LastRite         `json:"last_rite"`
		}{a.Rite, true, a.Standing, a.Sanction, NoAuspex, inscriptions, canon, a.LastRite})
	}
	type binharic struct {
		Rite         string            `json:"rite"`
		Edict        bool              `json:"edict"`
		Aspect       string            `json:"aspect"`
		Standing     Standing          `json:"standing"`
		Sanction     sanction.Standing `json:"sanction"`
		Auspex       Heed              `json:"auspex"`
		Desecrated   bool              `json:"desecrated"`
		Former       string            `json:"former,omitempty"`
		Inscriptions map[string]string `json:"inscriptions"`
		Canon        []Declaration     `json:"canon"`
		LastRite     *LastRite         `json:"last_rite"`
		Fitting      *[]string         `json:"fitting,omitempty"`
		Omens        []Omen            `json:"omens"`
		Taint        []Taint           `json:"taint"`
	}
	b := binharic{Rite: a.Rite, Aspect: a.Aspect, Standing: a.Standing, Sanction: a.Sanction, Auspex: a.Auspex, Desecrated: a.Desecrated,
		Inscriptions: inscriptions, Canon: canon, LastRite: a.LastRite, Omens: a.Omens, Taint: a.Taint}
	if a.Desecrated {
		b.Former = a.Former
	}
	if a.MarkII {
		fitting := a.Fitting
		if fitting == nil {
			fitting = []string{}
		}
		b.Fitting = &fitting
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
// ({"verse": 2, "tether": "…", "aspect": "porpl"}), the auspex's as
// {"auspex": true, "aspect": "porpl"}, and a subject's as its name, its
// designation, the omen read when one was, every aspect it fits and the
// aspect when it fits one ({"subject": "smt", "designation":
// "machine-spirit", "omen": "off", "aspects": ["on"], "aspect": "on"}).
func (o Omen) MarshalJSON() ([]byte, error) {
	if o.Subject != "" {
		return json.Marshal(struct {
			Subject     string   `json:"subject"`
			Designation string   `json:"designation"`
			Omen        string   `json:"omen,omitempty"`
			Aspects     []string `json:"aspects"`
			Aspect      string   `json:"aspect,omitempty"`
		}{o.Subject, o.Designation, o.Read, o.Aspects, o.Aspect})
	}
	if o.Auspex {
		return json.Marshal(struct {
			Auspex bool   `json:"auspex"`
			Aspect string `json:"aspect"`
		}{true, o.Aspect})
	}
	return versed(o.Verse, "aspect", o.Aspect)
}

// MarshalJSON writes a taint as the sanctum's verse and its own key
// ({"verse": 1, "sanctum": "…"}), or as the sanctum subject's name and
// designation ({"subject": "niri", "designation": "sanctum"}).
func (t Taint) MarshalJSON() ([]byte, error) {
	if t.Subject != "" {
		return json.Marshal(struct {
			Subject     string `json:"subject"`
			Designation string `json:"designation"`
		}{t.Subject, librarium.DesignationSanctum})
	}
	return versed(t.Verse)
}

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
