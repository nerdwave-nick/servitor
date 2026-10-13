package sanction

import (
	"fmt"
	"slices"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// state draws the exalted subjects of a state rite of pattern Mark II into
// plan, each by its place among the rite's subjects, and the exalted
// observances the plan's aspect admits by "to", each with its "from" and
// the places of the subjects that name it. A sanctum, transcription or
// tether is drawn as its verse of Mark I would be, an observance as its
// incantation or litany; a machine-spirit by the omen of the aspect and
// its enact words for every omen it declares.
func (d *drawer) state(plan *Plan) error {
	r := d.rite
	d.verse = 0
	for i, s := range r.Subjects {
		if !s.Exalted() {
			continue
		}
		d.what = fmt.Sprintf("the subject %q", s.Name)
		var (
			v   Verse
			err error
		)
		if ms, ok := s.Matter.(*librarium.MachineSpirit); ok {
			v, err = d.spirit(ms)
		} else {
			v, err = d.draw(s.Matter.(librarium.Step))
		}
		if err != nil {
			return err
		}
		v.Subject, v.Place = s.Name, i
		plan.Subjects = append(plan.Subjects, v)
	}
	for _, o := range r.Observances {
		if !o.Exalted() || (o.To != nil && !slices.Contains(o.To, d.aspect)) {
			continue
		}
		d.what = fmt.Sprintf("the observance %q", o.Name)
		v, err := d.draw(o.Speech)
		if err != nil {
			return err
		}
		v.Observance, v.From = o.Name, slices.Clone(o.From)
		v.Introit, v.Recessional = referrers(r, o.Name)
		plan.Observances = append(plan.Observances, v)
	}
	return nil
}

// referrers are the places of the subjects of r that name the observance
// as their introit, and as their recessional.
func referrers(r *librarium.Rite, observance string) (introit, recessional []int) {
	introit, recessional = []int{}, []int{}
	for i, s := range r.Subjects {
		if slices.Contains(s.Introit, observance) {
			introit = append(introit, i)
		}
		if slices.Contains(s.Recessional, observance) {
			recessional = append(recessional, i)
		}
	}
	return introit, recessional
}

// spirit draws an exalted machine-spirit: the omen of the plan's aspect,
// the enact words of every omen it declares (each hearing it as {{omen}}),
// and the tongue, patience and relics of its speech.
func (d *drawer) spirit(ms *librarium.MachineSpirit) (Verse, error) {
	v := Verse{Enact: map[string]string{}}
	var err error
	if v.Omen, err = d.omen(ms, d.aspect); err != nil {
		return v, err
	}
	enact := d.scope
	enact.Omen = true
	for _, aspect := range d.rite.Aspects {
		omen, err := d.omen(ms, aspect)
		if err != nil {
			return v, err
		}
		if _, drawn := v.Enact[omen]; drawn {
			continue // an omen several aspects share
		}
		text, ok := ms.Enact.For(omen)
		if !ok {
			return v, d.lament("no enact words are written for the omen %q, and no \"*\" serves in their stead", omen)
		}
		values := d.values
		values.Omen = omen
		if v.Enact[omen], err = d.renderIn(enact, text, values); err != nil {
			return v, err
		}
	}
	return v, d.speech(&v, librarium.Utterance{Tongue: ms.Tongue, Patience: ms.Patience, Relics: ms.Relics})
}

// omen is the omen aspect demands of the machine-spirit, rendered for that
// aspect with its decrees: the aspect's own name when it declares no omens.
func (d *drawer) omen(ms *librarium.MachineSpirit, aspect string) (string, error) {
	if ms.Omens.IsZero() {
		return aspect, nil
	}
	text, ok := ms.Omens.For(aspect)
	if !ok {
		return "", d.lament("no omen is written for the aspect %q, and no \"*\" serves in its stead", aspect)
	}
	omen, err := d.renderIn(d.scope, text, decreed(d.rite, aspect))
	return strings.TrimSpace(omen), err
}
