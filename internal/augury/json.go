package augury

import (
	"encoding/json"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// MarshalJSON writes the augury in the shape of the codex: rite, edict
// (false), aspect, standing, sanction, auspex, desecrated, former (only when
// desecrated), inscriptions, canon, last_rite (null when never invoked),
// fitting, omens and taint (lists, never null). An edict's is rite, edict
// (true), standing, sanction, auspex (always ""), inscriptions, canon and
// last_rite alone.
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
		Fitting      []string          `json:"fitting"`
		Omens        []Omen            `json:"omens"`
		Taint        []Taint           `json:"taint"`
	}
	b := binharic{Rite: a.Rite, Aspect: a.Aspect, Standing: a.Standing, Sanction: a.Sanction, Auspex: a.Auspex, Desecrated: a.Desecrated,
		Inscriptions: inscriptions, Canon: canon, LastRite: a.LastRite, Fitting: a.Fitting, Omens: a.Omens, Taint: a.Taint}
	if a.Desecrated {
		b.Former = a.Former
	}
	if b.Fitting == nil {
		b.Fitting = []string{}
	}
	if b.Omens == nil {
		b.Omens = []Omen{}
	}
	if b.Taint == nil {
		b.Taint = []Taint{}
	}
	return json.Marshal(b)
}

// MarshalJSON writes a subject's omen as its name, its designation, the
// omen read when one was, every aspect it fits and the aspect when it fits
// one ({"subject": "smt", "designation": "machine-spirit", "omen": "off",
// "aspects": ["on"], "aspect": "on"}), followed by "stale": true and
// "read_at" for an omen taken from the data-slate's record.
func (o Omen) MarshalJSON() ([]byte, error) {
	var readAt *time.Time
	if o.Stale {
		readAt = &o.ReadAt
	}
	return json.Marshal(struct {
		Subject     string     `json:"subject"`
		Designation string     `json:"designation"`
		Omen        string     `json:"omen,omitempty"`
		Aspects     []string   `json:"aspects"`
		Aspect      string     `json:"aspect,omitempty"`
		Stale       bool       `json:"stale,omitempty"`
		ReadAt      *time.Time `json:"read_at,omitempty"`
	}{o.Subject, o.Designation, o.Read, o.Aspects, o.Aspect, o.Stale, readAt})
}

// MarshalJSON writes a taint as the sanctum subject's name and designation
// ({"subject": "niri", "designation": "sanctum"}).
func (t Taint) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Subject     string `json:"subject"`
		Designation string `json:"designation"`
	}{t.Subject, librarium.DesignationSanctum})
}
