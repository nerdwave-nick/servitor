package rituals

import (
	"errors"
	"strconv"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// Reading is the augury of one rite.
type Reading struct {
	augury.Augury
	Rite   *librarium.Rite // nil when the rite is heretical
	Lament error           // a data-slate that could not be read; the augury stands
}

// Augur reads the augury of the rite name. A heretical rite is read from
// its data-slate alone, standing heretical. forgoAuspex leaves the auspex
// unawakened. The error is *Unrecorded.
func (s *Servitor) Augur(name string, forgoAuspex bool) (Reading, error) {
	r, err := s.Rite(name)
	opts := augury.Options{ForgoAuspex: forgoAuspex, Orders: s.Orders(), Env: s.env()}
	var heretical *Heretical
	switch {
	case errors.As(err, &heretical):
		a, lament := augury.Heretic(name, opts)
		return Reading{Augury: a, Lament: lament}, nil
	case err != nil:
		return Reading{}, err
	}
	a, lament := augury.Augur(r, opts)
	return Reading{Augury: a, Rite: r, Lament: lament}, nil
}

// Signs are the keys an augury answers besides the rite's inscriptions;
// an inscription bearing the name of a sign is read through the binharic
// augury alone.
var Signs = []string{"aspect", "standing", "desecrated", "former"}

// Keys are the keys the augury of r answers: Signs, then every inscription.
func Keys(r *librarium.Rite) []string {
	return append(append([]string{}, Signs...), r.InscriptionKeys()...)
}

// Value is what the reading holds under key — a sign, else an inscription
// ("" when not inscribed); ok is false for a key the rite knows not. A
// heretical rite answers the inscriptions its data-slate holds.
func (r Reading) Value(key string) (value string, ok bool) {
	switch key {
	case "aspect":
		return r.Aspect, true
	case "standing":
		return string(r.Standing), true
	case "desecrated":
		return strconv.FormatBool(r.Desecrated), true
	case "former":
		return r.Former, true
	}
	if r.Rite != nil {
		if _, declared := r.Rite.Inscription(key); !declared {
			return "", false
		}
		return r.Inscriptions[key], true
	}
	value, ok = r.Inscriptions[key]
	return value, ok
}
