package rituals

import (
	"errors"
	"fmt"

	"github.com/nerdwave-nick/servitor/internal/exalted"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// Offering is the exalted unit of a rite as it is offered to its master
// for sanction, beside the sanction that stands for it now.
type Offering struct {
	Rite *librarium.Rite
	Unit *sanction.Unit
	// Kept is the sanction that stands for the rite; nil when none does, or
	// when it cannot be read (Lament tells why), and the unit is offered
	// whole.
	Kept   *sanction.Sanction
	Lament error
}

// Sigil is the sigil of the unit offered.
func (o *Offering) Sigil() string { return o.Unit.Sigil() }

// Unchanged tells that the unit is the one sanctioned already: there is
// nothing to sanction anew.
func (o *Offering) Unchanged() bool { return o.Kept != nil && o.Kept.Sigil == o.Sigil() }

// Shown is what the master beholds before the proof: the whole unit, or
// its difference from the sanction that stands.
func (o *Offering) Shown() []string {
	if o.Kept == nil {
		return sanction.Show(o.Unit)
	}
	return sanction.Difference(&o.Kept.Unit, o.Unit)
}

// Unsanctionable is the lament for a rite that offers nothing to sanction:
// unrecorded, or holding no exalted verse. A sanction still kept under its
// name is forsaken, and the lament says how it is abjured.
type Unsanctionable struct {
	Rite     string
	Err      error // *Unrecorded, or sanction.ErrUnexalted
	Forsaken bool
}

func (u *Unsanctionable) Error() string {
	why := u.Err.Error()
	if errors.Is(u.Err, sanction.ErrUnexalted) {
		why = fmt.Sprintf("the rite %q holds nothing exalted, and there is nothing to sanction", u.Rite)
	}
	if u.Forsaken {
		why += fmt.Sprintf("; the sanction kept under its name is forsaken and serves nothing — abjure it: "+
			"servitor abjure %s", u.Rite)
	}
	return why
}

func (u *Unsanctionable) Unwrap() error { return u.Err }

// Offer offers the exalted unit of the rite name to its master for
// sanction. The Inquisition comes first: a rite holding any heresy is
// *Heretical, an unrecorded rite or one without exalted verses is
// *Unsanctionable. A unit the exalted servitor would never hear whole is
// refused as well.
func (s *Servitor) Offer(name string) (*Offering, error) {
	r, err := s.Rite(name)
	if errors.As(err, new(*Unrecorded)) {
		return nil, s.unsanctionable(name, err)
	}
	if err != nil {
		return nil, err
	}
	findings, err := s.Inquire([]string{name}, false)
	if err != nil {
		return nil, err
	}
	h := &Heretical{Name: name}
	for _, f := range findings {
		if f.Severity == librarium.Heresy {
			h.Findings = append(h.Findings, f)
		}
	}
	if len(h.Findings) > 0 {
		return nil, h
	}
	if !r.Exalted() {
		return nil, s.unsanctionable(name, sanction.ErrUnexalted)
	}
	u, err := s.Unit(r)
	if err != nil {
		return nil, fmt.Errorf("the %s of %q cannot be drawn into one unit: %w", r.ExaltedNoun(), name, err)
	}
	if size := len(u.Canonical()); size > exalted.UnitCap {
		return nil, fmt.Errorf("the %s of %q hold %d bytes, more than the %d a sanction may hold", r.ExaltedNoun(),
			name, size, exalted.UnitCap)
	}
	kept, lament := s.sanctions().Read(name)
	return &Offering{Rite: r, Unit: u, Kept: kept, Lament: lament}, nil
}

// unsanctionable is the lament for the rite name, which offers nothing to
// sanction for why.
func (s *Servitor) unsanctionable(name string, why error) error {
	kept, lament := s.sanctions().Read(name)
	return &Unsanctionable{Rite: name, Err: why, Forsaken: kept != nil || lament != nil}
}

// Unattended is the lament for a sanctioning no terminal attends: its
// master could not behold what they sanction.
type Unattended struct{ Rite string }

func (u *Unattended) Error() string {
	return fmt.Sprintf("the sanctioning of the rite %q is refused, and nothing was written: no terminal attends, and "+
		"the master must behold what they sanction — speak it at a terminal, or in the cogitator", u.Rite)
}

// Sanction lays the offering in the store as its master's sanction,
// through the exalted servitor and one proof the intercessor demands: the
// proof is the master's assent. An unchanged unit is not sanctioned anew,
// and no proof is asked. A sanctioning no terminal attends is
// *Unattended; a proof the master dismissed or failed is *Dismissed; an
// intercessor not found refuses as Intercession.Find does. Whatever is
// refused, nothing was written.
func (s *Servitor) Sanction(o *Offering, x Exaltation) error {
	if o.Unchanged() {
		return nil
	}
	if !x.Attended {
		return &Unattended{Rite: o.Unit.Rite}
	}
	deed := fmt.Sprintf("the sanctioning of the rite %q", o.Unit.Rite)
	i, err := Intercession.Find(s.Librarium.Settings.Intercessor)
	if err != nil {
		return fmt.Errorf("%s is refused, and nothing was written: %w", deed, err)
	}
	uid := s.sanctions().UID
	r, err := exalt(deed, i.Name, i.Sanction(o.Unit.Rite, o.Sigil()), x,
		func(ch *exalted.Channel) error { return ch.Offer(uid, o.Unit) })
	if err != nil {
		return err
	}
	if r.Verdict != exalted.Sanctioned {
		return &Spurned{Deed: deed, Lament: r.Lament}
	}
	return nil
}
