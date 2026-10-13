package rituals

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/intercessor"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// wordCap is the most a word given at invocation may hold to reach exalted
// speech: the exalted servitor refuses more.
const wordCap = 4 << 10

// exaltedPlan is what an invocation of a rite with exalted verses knows of
// them before it performs: the unit drawn for its master, the sigil of the
// plan for the aspect invoked, and the intercessor that is to vouch for the
// master.
type exaltedPlan struct {
	unit        *sanction.Unit
	sigil       string
	intercessor *intercessor.Intercessor
}

// exaltation judges what the exalted verses of r demand of the petition
// before anything is touched, and tells res the sanction standing of the
// plan of the aspect invoked. A rune given for an inscription spliced into
// exalted words is heresy, foreseen or not. To be performed, the plan must
// stand sanctioned, the petition must lend an Exaltation, and an
// intercessor must be found to vouch for the master beside the servitor
// installed where the root alone may write; else the invocation is refused
// before any proof is asked. A rite without exalted verses demands none of
// it (nil).
func (s *Servitor) exaltation(r *librarium.Rite, p Petition, res *Result) (*exaltedPlan, error) {
	if !r.Exalted() {
		return nil, nil
	}
	u, err := s.Unit(r)
	if err != nil {
		if p.Foresee {
			return nil, nil // the pre-flight tells what cannot be read
		}
		return nil, fmt.Errorf("%s is refused, and nothing was touched: its %s cannot be drawn into "+
			"the unit their sanction is judged by: %w", invocationOf(r, p.Aspect), r.ExaltedNoun(), err)
	}
	if hs := splicedRunes(r, u, p.Runes); len(hs) > 0 {
		return nil, hs
	}
	sigil, ok := u.SigilOf(p.Aspect)
	if !ok {
		return nil, nil // no aspect of the rite: the pre-flight denounces it
	}
	kept, lament := s.sanctions().Read(r.Name)
	res.Sigil, res.Sanction = sigil, standingOf(sigil, p.Aspect, kept, lament)
	if p.Foresee {
		return nil, nil
	}
	if res.Sanction != sanction.Sanctioned {
		return nil, unsanctioned(r, p.Aspect, res.Sanction)
	}
	if p.Exaltation == nil {
		return nil, unexalted(r, fmt.Sprintf("this invocation was lent no intercessor to raise the exalted servitor "+
			"— invoke it at a terminal: servitor invoke %s", strings.TrimSpace(r.Name+
			" "+p.Aspect)))
	}
	i, err := Intercession.Find(s.Librarium.Settings.Intercessor)
	if err != nil {
		return nil, fmt.Errorf("%s is refused, and nothing was touched: %w", invocationOf(r, p.Aspect), err)
	}
	res.Intercessor = i.Name
	return &exaltedPlan{unit: u, sigil: sigil, intercessor: i}, nil
}

// standingOf is how the plan of aspect, bearing sigil, stands towards the
// sanction kept for its rite (nil when none is; lament when it cannot be
// read, which lapses it).
func standingOf(sigil, aspect string, kept *sanction.Sanction, lament error) sanction.Standing {
	switch {
	case kept == nil && lament == nil:
		return sanction.Unsanctioned
	case lament != nil:
		return sanction.Lapsed
	}
	if was, ok := kept.SigilOf(aspect); ok && was == sigil {
		return sanction.Sanctioned
	}
	return sanction.Lapsed
}

// invocationOf names the invocation of r into aspect, or the proclaiming
// of the edict r.
func invocationOf(r *librarium.Rite, aspect string) string {
	if r.Edict {
		return fmt.Sprintf("the proclaiming of the edict %q", r.Name)
	}
	return fmt.Sprintf("the invocation of the rite %q into %q", r.Name, aspect)
}

// unsanctioned refuses the plan of aspect that stands unsanctioned or
// lapsed, naming the ritual that sanctions it.
func unsanctioned(r *librarium.Rite, aspect string, st sanction.Standing) error {
	of := fmt.Sprintf("for %q", aspect)
	if r.Edict {
		of = "of the edict"
	}
	why := fmt.Sprintf("the rite %q stands unsanctioned, for its master never sanctioned its %s — "+
		"sanction them: servitor sanction %s", r.Name, r.ExaltedNoun(), r.Name)
	if st == sanction.Lapsed {
		why = fmt.Sprintf("the sanction of the rite %q has lapsed, for its %s %s are no longer those its "+
			"master sanctioned — sanction them anew: servitor sanction %s", r.Name, r.ExaltedNoun(), of, r.Name)
	}
	return unexalted(r, why)
}

// unexalted refuses the invocation of r, with a heresy for each exalted
// verse — or exalted subject and observance of a state rite — that tells
// why: nothing exalted is ever performed with its invoker's authority.
func unexalted(r *librarium.Rite, why string) error {
	noun := "verse"
	if r.MarkII && !r.Edict {
		noun = "subject or observance"
	}
	var hs invocation.Heresies
	for _, item := range r.ExaltedItems() {
		hs = append(hs, invocation.Heresy{Verse: item.Verse, Finding: librarium.Finding{
			Severity: librarium.Heresy, Scripture: r.Path, Rite: r.Name, Position: r.Locate(item.Pointer),
			Message: fmt.Sprintf("%s is exalted: it is performed with the machine's own authority alone, by "+
				"the exalted servitor from the sanction of its master; %s — not one %s of %q was performed",
				item, why, noun, r.Name),
		}})
	}
	return hs
}

// splicedRunes denounces every rune given for an inscription spliced into
// the exalted words of the unit u of r: such an inscription takes its
// decree alone, fixed when its master sanctions them.
func splicedRunes(r *librarium.Rite, u *sanction.Unit, runes map[string]string) invocation.Heresies {
	spliced := map[string]bool{}
	for _, plan := range u.Aspects {
		for key := range plan.Inscriptions {
			spliced[key] = true
		}
	}
	var hs invocation.Heresies
	for _, key := range slices.Sorted(maps.Keys(runes)) {
		if !spliced[key] {
			continue
		}
		hs = append(hs, invocation.Heresy{Finding: librarium.Finding{
			Severity: librarium.Heresy, Scripture: r.Path, Rite: r.Name, Position: r.Locate("/inscriptions/" + key),
			Message: fmt.Sprintf("the inscription %q is spliced into the %s of %q, whose words are fixed "+
				"when their master sanctions them: it takes its decree alone, and no rune --%s may be given to "+
				"their invocation — strike the rune, or let exalted speech hear the word as \"$%s\" instead of "+
				"splicing it", key, r.ExaltedNoun(), r.Name, key, placeholder.InscriptionVariable(key)),
		}})
	}
	return hs
}

// unspeakable denounces every word given at invocation that the exalted
// servitor would refuse to let exalted speech hear: longer than wordCap,
// or bearing a NUL or a newline.
func unspeakable(r *librarium.Rite, inscriptions map[string]string) invocation.Heresies {
	var hs invocation.Heresies
	for _, key := range slices.Sorted(maps.Keys(inscriptions)) {
		v := inscriptions[key]
		if len(v) <= wordCap && !strings.ContainsAny(v, "\x00\n") {
			continue
		}
		hs = append(hs, invocation.Heresy{Finding: librarium.Finding{
			Severity: librarium.Heresy, Scripture: r.Path, Rite: r.Name, Position: r.Locate("/inscriptions/" + key),
			Message: fmt.Sprintf("the word of the inscription %q reaches exalted speech as $%s, which bears no NUL "+
				"and no newline and holds at most %d bytes — the exalted servitor would refuse it", key,
				placeholder.InscriptionVariable(key), wordCap),
		}})
	}
	return hs
}

// errLost is a channel to the exalted servitor that failed mid-liturgy.
var errLost = errors.New("the channel to the exalted servitor was lost")
