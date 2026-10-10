package rituals

import (
	"github.com/nerdwave-nick/servitor/internal/augury"
)

// Entry is one rite of the census; it marshals as the census's binharic.
type Entry struct {
	Rite       string           `json:"rite"`
	Aspect     string           `json:"aspect"`
	Standing   augury.Standing  `json:"standing"`
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
		e := Entry{Rite: name, Aspect: r.Aspect, Standing: r.Standing, Aspects: []string{},
			Desecrated: r.Desecrated, LastRite: r.LastRite, RecordedIn: s.Librarium.Scriptures[name][0]}
		if r.Rite != nil {
			e.Aspects, e.Purpose = r.Rite.Aspects, r.Rite.Purpose
		}
		entries = append(entries, e)
	}
	return entries, laments
}
