package rituals

import (
	"fmt"

	"github.com/nerdwave-nick/servitor/internal/exalted"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// Unkept is the lament for an abjuring of a sanction that is not kept: of
// the rite, or of any rite when Rite is "".
type Unkept struct{ Rite string }

func (u *Unkept) Error() string {
	if u.Rite == "" {
		return "the abjuring of every sanction is refused: no sanction is laid for you — `servitor sanction` lists " +
			"the sanctions laid"
	}
	return fmt.Sprintf("the abjuring of the sanction of %q is refused: no sanction of %q is laid for you — "+
		"`servitor sanction` lists the sanctions laid", u.Rite, u.Rite)
}

// Fallen is the lament of an abjuring of every sanction that fell midway:
// the sanctions of Abjured were struck, the rest stand.
type Fallen struct {
	Abjured []string
	Lament  string
}

func (f *Fallen) Error() string {
	return fmt.Sprintf("the abjuring of every sanction fell midway, the sanctions of %q struck and the rest "+
		"standing: %s", f.Abjured, f.Lament)
}

// Abjure strikes the sanction of the rite name from the store, or every
// sanction of the master when name is "", through the exalted servitor and
// one proof the intercessor demands; it names the rites whose sanctions it
// struck. It strikes forsaken sanctions as any other. A sanction not laid is
// *Unkept, a proof dismissed or failed *Dismissed, an intercessor not found
// refused as Intercession.Find refuses: then nothing was struck.
func (s *Servitor) Abjure(name string, x Exaltation) ([]string, error) {
	deed := "the abjuring of every sanction"
	if name != "" {
		deed = fmt.Sprintf("the abjuring of the sanction of %q", name)
	}
	store := s.sanctions()
	switch {
	case name != "" && !librarium.ValidName(name):
		return nil, &Unkept{Rite: name}
	case name != "":
		if kept, lament := store.Read(name); kept == nil && lament == nil {
			return nil, &Unkept{Rite: name}
		}
	default:
		if kept, laments := store.List(); len(kept) == 0 && len(laments) == 0 {
			return nil, &Unkept{}
		}
	}
	i, err := Intercession.Find(s.Librarium.Settings.Intercessor)
	if err != nil {
		return nil, fmt.Errorf("%s is refused, and nothing was struck: %w", deed, err)
	}
	words := exalted.Abjuring{Protocol: exalted.Protocol, UID: store.UID, Rite: name, All: name == ""}
	r, err := exalt(deed, i.Name, i.Abjure(name), x,
		func(ch *exalted.Channel) error { return ch.Send(words) })
	switch {
	case err != nil:
		return nil, err
	case r.Verdict == exalted.Fell:
		return r.Abjured, &Fallen{Abjured: r.Abjured, Lament: r.Lament}
	case r.Verdict != exalted.Abjured:
		return nil, &Spurned{Deed: deed, Lament: r.Lament}
	}
	return r.Abjured, nil
}
