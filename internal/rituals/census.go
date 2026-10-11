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

// Census reads every rite of the Librarium, heretical or not, by name.
// forgoAuspex leaves every auspex unawakened. Laments are data-slates
// that could not be read.
func (s *Servitor) Census(forgoAuspex bool) (entries []Entry, laments []error) {
	entries = []Entry{}
	for _, name := range s.Librarium.Names() {
		r, err := s.Augur(name, forgoAuspex)
		if err != nil {
			laments = append(laments, err)
			continue
		}
		if r.Lament != nil {
			laments = append(laments, r.Lament)
		}
		e := Entry{Rite: name, Edict: r.Edict, Aspect: r.Aspect, Standing: r.Standing, Auspex: r.Auspex,
			Aspects: []string{}, Desecrated: r.Desecrated, LastRite: r.LastRite,
			RecordedIn: s.Librarium.Scriptures[name][0]}
		if r.Rite != nil {
			e.Purpose = r.Rite.Purpose
			if !r.Rite.Edict {
				e.Aspects = r.Rite.Aspects
			}
		}
		entries = append(entries, e)
	}
	return entries, laments
}
