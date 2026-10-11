package rituals

import (
	"github.com/nerdwave-nick/servitor/internal/augury"
)

// Entry is one rite of the census; it marshals as the census's binharic.
type Entry struct {
	Rite       string           `json:"rite"`
	Edict      bool             `json:"edict"` // an edict bears no aspect: its aspect is "", its aspects []
	Aspect     string           `json:"aspect"`
	Standing   augury.Standing  `json:"standing"`
	Auspex     augury.Heed      `json:"auspex"` // what the auspex gave
	Aspects    []string         `json:"aspects"`
	Purpose    string           `json:"purpose"`
	Desecrated bool             `json:"desecrated"`
	LastRite   *augury.LastRite `json:"last_rite"`
	RecordedIn string           `json:"recorded_in"` // the rite's scripture; the first, when recorded twice
}

// Census reads every rite of the Librarium, heretical or not, by name. It
// augurs them together, at most maxAuspexes auspexes awake at once.
// forgoAuspex leaves every auspex unawakened. Laments are data-slates
// that could not be read, in the order of their rites.
func (s *Servitor) Census(forgoAuspex bool) (entries []Entry, laments []error) {
	names := s.Librarium.Names()
	read := make([]augural, len(names))
	s.gather(names, forgoAuspex, func(a augural) { read[a.at] = a })
	entries = []Entry{}
	for _, a := range read {
		if a.lament != nil {
			laments = append(laments, a.lament)
		}
		if a.entry != nil {
			entries = append(entries, *a.entry)
		}
	}
	return entries, laments
}

// entry is the census entry of the rite name read as r.
func (s *Servitor) entry(name string, r Reading) Entry {
	e := Entry{Rite: name, Edict: r.Edict, Aspect: r.Aspect, Standing: r.Standing, Auspex: r.Auspex,
		Aspects: []string{}, Desecrated: r.Desecrated, LastRite: r.LastRite,
		RecordedIn: s.Librarium.Scriptures[name][0]}
	if r.Rite != nil {
		e.Purpose = r.Rite.Purpose
		if !r.Rite.Edict {
			e.Aspects = r.Rite.Aspects
		}
	}
	return e
}
