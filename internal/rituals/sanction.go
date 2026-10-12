package rituals

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// SanctionRoot is where the servitor reads the store of sanctions when it is
// given none; the trials lay it in a hall of their own.
var SanctionRoot = sanction.DefaultRoot

// sanctions is the store of the master's sanctions: Sanctions, else the
// store at SanctionRoot of the uid the servitor serves.
func (s *Servitor) sanctions() sanction.Store {
	if s.Sanctions != nil {
		return *s.Sanctions
	}
	return sanction.Store{Root: SanctionRoot, UID: s.master()}
}

// master is the uid whose sanctions the servitor reads: its own, or, roused
// as root by sudo, the uid of the one who spoke sudo.
func (s *Servitor) master() int {
	if os.Geteuid() == 0 {
		if v, ok := s.env()("SUDO_UID"); ok {
			if uid, err := strconv.Atoi(v); err == nil && uid > 0 {
				return uid
			}
		}
	}
	return os.Getuid()
}

// Unit draws the exalted unit of the rite r for the master; the error is
// sanction.ErrUnexalted for a rite without exalted verses.
func (s *Servitor) Unit(r *librarium.Rite) (*sanction.Unit, error) {
	return sanction.Build(r, sanction.Options{UID: s.sanctions().UID, Orders: s.Orders()})
}

// standing tells how the rite name, read as r (nil when heretical), stands
// towards its sanction, with the sanction itself. A rite without exalted
// verses stands towards none, even when a forsaken sanction bears its name;
// a heretical rite whose sanction is kept has lapsed. The lament is a
// sanction that cannot be read, which lapses an exalted rite.
func (s *Servitor) standing(name string, r *librarium.Rite) (sanction.Standing, *sanction.Sanction, error) {
	if r != nil && !r.Exalted() {
		return sanction.None, nil, nil
	}
	kept, lament := s.sanctions().Read(name)
	switch {
	case r == nil && kept == nil && lament == nil:
		return sanction.None, nil, nil
	case r == nil:
		return sanction.Lapsed, kept, lament
	case lament != nil:
		return sanction.Lapsed, nil, lament
	}
	u, err := s.Unit(r)
	if err != nil {
		u = nil
	}
	return sanction.Judge(u, kept), kept, nil
}

// Sanctioned is one sanction of the master as `servitor sanction` lists it;
// it marshals as its binharic.
type Sanctioned struct {
	Rite         string            `json:"rite"`
	Standing     sanction.Standing `json:"standing"` // sanctioned, lapsed or forsaken
	Sigil        string            `json:"sigil"`
	SanctionedAt time.Time         `json:"sanctioned_at"`
}

// ListSanctions lists every sanction of the master by the name of its rite,
// with its standing: forsaken when its rite is gone or holds no exalted
// verse. Laments are sanctions that cannot be read.
func (s *Servitor) ListSanctions() ([]Sanctioned, []error) {
	kept, laments := s.sanctions().List()
	out := []Sanctioned{}
	for _, k := range kept {
		e := Sanctioned{Rite: k.Rite, Standing: sanction.Forsaken, Sigil: k.Sigil, SanctionedAt: k.SanctionedAt}
		if _, recorded := s.Librarium.Scriptures[k.Rite]; recorded {
			r := s.Librarium.Rites[k.Rite]
			if st, _, _ := s.standing(k.Rite, r); st != sanction.None {
				e.Standing = st
			}
		}
		out = append(out, e)
	}
	return out, laments
}

// ForsakenSanction is the lament of a forsaken sanction.
type ForsakenSanction struct {
	Rite string
	Gone bool // the rite is gone; else it holds no exalted verse
}

func (f *ForsakenSanction) Error() string {
	why := fmt.Sprintf("no rite named %q is recorded in the Librarium any more", f.Rite)
	if !f.Gone {
		why = fmt.Sprintf("the rite %q holds no exalted verse any more", f.Rite)
	}
	return fmt.Sprintf("the sanction of %q is forsaken: %s, and the sanction serves nothing — abjure it: "+
		"servitor abjure %s", f.Rite, why, f.Rite)
}

// forsaken laments every forsaken sanction of the master, by the name of
// its rite.
func (s *Servitor) forsaken() []error {
	kept, _ := s.sanctions().List()
	var out []error
	for _, k := range kept {
		_, recorded := s.Librarium.Scriptures[k.Rite]
		r := s.Librarium.Rites[k.Rite]
		if !recorded || (r != nil && !r.Exalted()) {
			out = append(out, &ForsakenSanction{Rite: k.Rite, Gone: !recorded})
		}
	}
	return out
}

// sanction notes an exalted rite that stands unsanctioned, or whose
// sanction lapsed, at the "exalted" of its first exalted verse.
func (e *examiner) sanction(s *Servitor) {
	standing, kept, lament := s.standing(e.rite.Name, e.rite)
	verses := e.rite.ExaltedVerses()
	if len(verses) == 0 {
		return
	}
	ptr := fmt.Sprintf("/liturgy/%d/exalted", verses[0]-1)
	name := e.rite.Name
	switch standing {
	case sanction.Unsanctioned:
		e.note(librarium.Impurity, ptr, "the rite %q holds exalted verses, yet it stands unsanctioned: its master "+
			"never sanctioned them, and every invocation is refused before any proof is asked — sanction them: "+
			"servitor sanction %s", name, name)
	case sanction.Lapsed:
		why := "its exalted verses are no longer those its master sanctioned"
		if kept != nil {
			why = fmt.Sprintf("its exalted verses are no longer those its master sanctioned under the sigil %s",
				sanction.Short(kept.Sigil))
		}
		if lament != nil {
			why = lament.Error()
		}
		e.note(librarium.Impurity, ptr, "the sanction of the rite %q has lapsed: %s, and every invocation is "+
			"refused before any proof is asked — sanction them anew: servitor sanction %s", name, why, name)
	}
}

// forsakenFindings are the impurities of the forsaken sanctions of the rites
// wanted, each found at the head of its sanction.
func (s *Servitor) forsakenFindings(wanted func(string) bool) librarium.Findings {
	var out librarium.Findings
	for _, err := range s.forsaken() {
		f := err.(*ForsakenSanction)
		if wanted(f.Rite) {
			out = append(out, librarium.Finding{Severity: librarium.Impurity, Scripture: s.sanctions().Path(f.Rite),
				Rite: f.Rite, Position: librarium.Position{Line: 1, Column: 1}, Message: f.Error()})
		}
	}
	return out
}
