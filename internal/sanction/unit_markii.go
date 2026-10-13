package sanction

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strconv"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// An exalted subject or observance of a state rite is
// written by its name and designation, as the scripture writes it, with
// the keys of that designation: a subject also by its place among the
// rite's subjects, an observance by the places of the subjects that name
// it as introit or recessional and by its "from".

// Misshapen tells why the plans of the unit do not hold what its pattern
// holds; "" when they do. A plan of a rite of aspects holds subjects and
// observances alone; the plan of an edict exalted verses alone, each an
// incantation or a litany. A unit of PatternMarkI, read from a sanction of
// old, holds verses of every kind.
func (u *Unit) Misshapen() string {
	for _, p := range u.Aspects {
		switch {
		case u.State() && len(p.Liturgy) > 0:
			return fmt.Sprintf("the plan of %q holds verses, yet a rite of aspects of pattern %s holds subjects "+
				"and observances", p.Aspect, Pattern)
		case !u.State() && len(p.Subjects)+len(p.Observances) > 0:
			return fmt.Sprintf("the plan of %q holds subjects or observances, which only a rite of aspects of "+
				"pattern %s holds", p.Aspect, Pattern)
		}
		for _, v := range p.Liturgy {
			switch {
			case v.Verse < 1 || v.Subject != "" || v.Observance != "":
				return fmt.Sprintf("the plan of %q holds %s among its verses", p.Aspect, v.Named())
			case u.Pattern == Pattern && v.Kind != librarium.KindIncantation && v.Kind != librarium.KindLitany:
				return fmt.Sprintf("the plan of %q holds %s, a %s, yet an edict of pattern %s speaks only "+
					"incantations and litanies", p.Aspect, v.Named(), v.Kind.Key(), Pattern)
			}
		}
		for _, v := range p.Subjects {
			if v.Subject == "" {
				return fmt.Sprintf("the plan of %q holds %s among its subjects", p.Aspect, v.Named())
			}
		}
		for _, v := range p.Observances {
			if v.Observance == "" {
				return fmt.Sprintf("the plan of %q holds %s among its observances", p.Aspect, v.Named())
			}
		}
	}
	return ""
}

// Spirit tells that the item is a machine-spirit subject.
func (v Verse) Spirit() bool { return v.Subject != "" && v.Kind == 0 }

// Speaks tells that the item is exalted speech: an incantation, a litany,
// or a machine-spirit's enact words.
func (v Verse) Speaks() bool {
	return v.Kind == librarium.KindIncantation || v.Kind == librarium.KindLitany || v.Spirit()
}

// Named names the item as laments name it: "verse 2", or the subject or
// observance by its name.
func (v Verse) Named() string {
	return librarium.ExaltedItem{Verse: v.Verse, Subject: v.Subject, Observance: v.Observance}.String()
}

// Designation is the designation of a subject or observance.
func (v Verse) Designation() string {
	if v.Spirit() {
		return librarium.DesignationMachineSpirit
	}
	return v.Kind.Key()
}

// The writings of each designation of an item.
type (
	sanctumSubject struct {
		Name         string `json:"name"`
		Place        int    `json:"place"`
		Designation  string `json:"designation"`
		Vessel       string `json:"vessel"`
		Ward         string `json:"ward"`
		Glyph        string `json:"glyph"`
		ClosingGlyph string `json:"closing-glyph"`
		Consecrate   bool   `json:"consecrate"`
		Scripture    []byte `json:"scripture"`
	}
	transcriptionSubject struct {
		Name        string  `json:"name"`
		Place       int     `json:"place"`
		Designation string  `json:"designation"`
		Vessel      string  `json:"vessel"`
		Scripture   *[]byte `json:"scripture"` // null removes the vessel
		Seal        *string `json:"seal"`      // octal, e.g. "0644"
		Zeal        bool    `json:"zeal"`
	}
	tetherSubject struct {
		Name        string  `json:"name"`
		Place       int     `json:"place"`
		Designation string  `json:"designation"`
		Tether      string  `json:"tether"`
		Anchor      *string `json:"anchor"` // null unbinds it
		Zeal        bool    `json:"zeal"`
	}
	spiritSubject struct {
		Name        string            `json:"name"`
		Place       int               `json:"place"`
		Designation string            `json:"designation"`
		Omen        string            `json:"omen"`
		Enact       map[string]string `json:"enact"`
		Tongue      string            `json:"tongue"`
		Patience    string            `json:"patience"`
		Relics      []Relic           `json:"relics"`
		Auspex      *auspexItem       `json:"auspex,omitzero"` // a privileged auspex alone
	}
	auspexItem struct {
		Scan         string            `json:"scan"`
		Trials       map[string]string `json:"trials"`
		Patience     string            `json:"patience"`
		NamesAspects bool              `json:"names-aspects"`
	}
	observanceItem struct {
		Name        string    `json:"name"`
		Designation string    `json:"designation"`
		Words       string    `json:"words,omitzero"`     // an incantation's
		Scroll      string    `json:"scroll,omitzero"`    // a litany's, as written
		Scripture   *[]byte   `json:"scripture,omitzero"` // the bytes of a litany's scroll
		Offerings   *[]string `json:"offerings,omitzero"` // a litany's
		Tongue      string    `json:"tongue"`
		Patience    string    `json:"patience"`
		Reversion   string    `json:"reversion"`
		Relics      []Relic   `json:"relics"`
		From        *[]string `json:"from"` // null admits every former aspect
		Introit     []int     `json:"introit"`
		Recessional []int     `json:"recessional"`
	}
)

// marshalItem writes a subject or observance.
func (v Verse) marshalItem() ([]byte, error) {
	var w any
	switch {
	case v.Observance != "" && (v.Kind == librarium.KindIncantation || v.Kind == librarium.KindLitany):
		o := observanceItem{Name: v.Observance, Designation: v.Kind.Key(), Tongue: v.Tongue,
			Patience: v.Patience.String(), Reversion: v.Reversion, Relics: orEmpty(v.Relics),
			Introit: orEmpty(v.Introit), Recessional: orEmpty(v.Recessional)}
		if v.From != nil {
			from := slices.Clone(v.From)
			o.From = &from
		}
		if v.Kind == librarium.KindIncantation {
			o.Words = v.Target
		} else {
			scroll, offerings := orEmpty(v.Scroll), orEmpty(v.Offerings)
			o.Scroll, o.Scripture, o.Offerings = v.Target, &scroll, &offerings
		}
		w = o
	case v.Observance != "" || v.Verse != 0:
		return nil, fmt.Errorf("%s is no exalted item of a state rite", v.Named())
	case v.Spirit():
		sp := spiritSubject{v.Subject, v.Place, librarium.DesignationMachineSpirit, v.Omen, enactOf(v.Enact), v.Tongue,
			v.Patience.String(), orEmpty(v.Relics), nil}
		if a := v.Auspex; a != nil {
			sp.Auspex = &auspexItem{a.Scan, enactOf(a.Trials), a.Patience.String(), a.NamesAspects}
		}
		w = sp
	case v.Kind == librarium.KindSanctum:
		w = sanctumSubject{v.Subject, v.Place, v.Kind.Key(), v.Target, v.Ward, v.Glyph, v.ClosingGlyph, v.Consecrate,
			orEmpty(v.Scripture)}
	case v.Kind == librarium.KindTranscription:
		t := transcriptionSubject{Name: v.Subject, Place: v.Place, Designation: v.Kind.Key(), Vessel: v.Target,
			Zeal: v.Zeal}
		if !v.Null {
			sc := orEmpty(v.Scripture)
			t.Scripture = &sc
		}
		if v.Seal != nil {
			s := fmt.Sprintf("%04o", uint32(*v.Seal))
			t.Seal = &s
		}
		w = t
	case v.Kind == librarium.KindTether:
		t := tetherSubject{Name: v.Subject, Place: v.Place, Designation: v.Kind.Key(), Tether: v.Target, Zeal: v.Zeal}
		if !v.Null {
			t.Anchor = &v.Anchor
		}
		w = t
	default:
		return nil, fmt.Errorf("%s is of no designation a subject bears", v.Named())
	}
	return json.Marshal(w, json.Deterministic(true))
}

func enactOf(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}

// unmarshalItem reads a subject or observance strictly, by its designation.
func (v *Verse) unmarshalItem(data []byte, keys map[string]jsontext.Value) error {
	var designation string
	if err := json.Unmarshal(keys["designation"], &designation); err != nil {
		return err
	}
	strict := json.RejectUnknownMembers(true)
	*v = Verse{}
	var err error
	switch designation {
	case librarium.DesignationSanctum:
		var w sanctumSubject
		err = json.Unmarshal(data, &w, strict)
		*v = Verse{Subject: w.Name, Place: w.Place, Kind: librarium.KindSanctum, Target: w.Vessel, Ward: w.Ward,
			Glyph: w.Glyph, ClosingGlyph: w.ClosingGlyph, Consecrate: w.Consecrate, Scripture: w.Scripture}
	case librarium.DesignationTranscription:
		var w transcriptionSubject
		if err = json.Unmarshal(data, &w, strict); err == nil {
			*v = Verse{Subject: w.Name, Place: w.Place, Kind: librarium.KindTranscription, Target: w.Vessel,
				Zeal: w.Zeal, Null: w.Scripture == nil}
			if w.Scripture != nil {
				v.Scripture = *w.Scripture
			}
			if w.Seal != nil {
				seal, perr := strconv.ParseUint(*w.Seal, 8, 32)
				m := fs.FileMode(seal)
				v.Seal, err = &m, perr
			}
		}
	case librarium.DesignationTether:
		var w tetherSubject
		if err = json.Unmarshal(data, &w, strict); err == nil {
			*v = Verse{Subject: w.Name, Place: w.Place, Kind: librarium.KindTether, Target: w.Tether, Zeal: w.Zeal,
				Null: w.Anchor == nil}
			if w.Anchor != nil {
				v.Anchor = *w.Anchor
			}
		}
	case librarium.DesignationMachineSpirit:
		var w spiritSubject
		if err = json.Unmarshal(data, &w, strict); err == nil {
			*v = Verse{Subject: w.Name, Place: w.Place, Omen: w.Omen, Enact: maps.Clone(w.Enact), Tongue: w.Tongue,
				Relics: w.Relics}
			v.Patience, err = time.ParseDuration(w.Patience)
			if a := w.Auspex; a != nil && err == nil {
				v.Auspex = &Auspex{Scan: a.Scan, Trials: maps.Clone(a.Trials), NamesAspects: a.NamesAspects}
				v.Auspex.Patience, err = time.ParseDuration(a.Patience)
			}
		}
	case librarium.DesignationIncantation, librarium.DesignationLitany:
		err = v.unmarshalObservance(data, designation)
	default:
		return fmt.Errorf("an item of a sanction bears the designation %q, which none bears", designation)
	}
	if err == nil && v.Subject == "" && v.Observance == "" {
		err = fmt.Errorf("an item of a sanction of the designation %q bears no name", designation)
	}
	return err
}

// unmarshalObservance reads an observance of the designation strictly.
func (v *Verse) unmarshalObservance(data []byte, designation string) error {
	var w observanceItem
	if err := json.Unmarshal(data, &w, json.RejectUnknownMembers(true)); err != nil {
		return err
	}
	*v = Verse{Observance: w.Name, Kind: librarium.KindIncantation, Target: w.Words, Tongue: w.Tongue,
		Reversion: w.Reversion, Relics: w.Relics, Introit: w.Introit, Recessional: w.Recessional}
	if w.From != nil {
		v.From = *w.From
	}
	litany := designation == librarium.DesignationLitany
	switch {
	case litany && (w.Words != "" || w.Scripture == nil || w.Offerings == nil):
		return fmt.Errorf("the litany %q of a sanction bears no scroll and offerings, or the words of an incantation",
			w.Name)
	case !litany && (w.Scroll != "" || w.Scripture != nil || w.Offerings != nil):
		return fmt.Errorf("the incantation %q of a sanction bears the scroll of a litany", w.Name)
	case litany:
		v.Kind, v.Target, v.Scroll, v.Offerings = librarium.KindLitany, w.Scroll, *w.Scripture, *w.Offerings
	}
	var err error
	v.Patience, err = time.ParseDuration(w.Patience)
	return err
}
