// Package augury reads which aspect a rite of pattern Mark I stands in, and
// how it stands, and keeps each rite's data-slate.
//
// Every step that can be observed yields an omen: a sanctum the aspect its
// marker records, a tether the aspect whose rendered anchor it leads to
// (null when unbound), a transcription the aspect whose scripture its
// vessel holds (null when absent), and the rite's auspex the aspect it
// prints. Incantations, litanies, vox-casts and vessels that cannot be read
// yield none, and so does a step whose observation fits several aspects.
// Observed fields are rendered for every aspect with the inscriptions the
// data-slate recorded for its aspect, or else the decrees of the aspect.
//
// All omens agreeing name the aspect (performed); any disagreement leaves it
// unknown (corrupted). Without omen the data-slate names it, and without
// either the rite lies dormant. Omens agreeing on an aspect the data-slate
// did not record mark the rite desecrated, a sanctum whose scripture other
// hands altered taints it, and a last invocation that faltered corrupts it.
package augury

import (
	"os"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// Standing is how a rite stands.
type Standing string

// The standings, from the gravest down. A rite holds the gravest that
// applies: heretical, corrupted, tainted, desecrated, then performed or
// dormant.
const (
	Heretical  Standing = "heretical"  // its scripture is heresy
	Corrupted  Standing = "corrupted"  // its omens disagree, or its last invocation faltered
	Tainted    Standing = "tainted"    // other hands altered a sanctum's scripture
	Desecrated Standing = "desecrated" // its omens agree on an aspect the data-slate did not record
	Performed  Standing = "performed"  // its omens, or else its data-slate, name its aspect
	Dormant    Standing = "dormant"    // neither omen nor data-slate names an aspect
)

// Augury is what the servitor reads of one rite; it marshals as the JSON of
// `servitor augury`.
type Augury struct {
	Rite       string
	Aspect     string // "" when unknown
	Standing   Standing
	Desecrated bool   // the omens agree on an aspect the data-slate did not record
	Former     string // the data-slate's aspect, when desecrated
	// Inscriptions are those on the data-slate: a rite never invoked
	// through the servitor has none.
	Inscriptions map[string]string
	LastRite     *LastRite // nil when the rite was never invoked
	Omens        []Omen    // in written order, the auspex's last
	Taint        []Taint   // in written order
}

// Omen is what one step, or the auspex, reveals about the aspect.
type Omen struct {
	// Verse names the step: its verse, its own key, and what that key holds
	// rendered for Aspect (not resolved). Zero for the auspex.
	invocation.Verse
	Auspex bool // the omen of the rite's auspex
	Aspect string
}

// Taint is a sanctum whose scripture other hands altered: it no longer
// holds the scripture of the aspect its marker records.
type Taint struct {
	invocation.Verse
}

// Options are what an augury is read with.
type Options struct {
	// Target is the aspect being invoked when the augury is read during an
	// invocation; the auspex sees it as {{aspect}} and SERVITOR_ASPECT.
	// Outside an invocation it is "".
	Target string
	// ForgoAuspex leaves the auspex unawakened, for answers that must come
	// at once (shell completion).
	ForgoAuspex bool
	// Orders are the settings resolved; the auspex is spoken in the rite's
	// tongue, else theirs, else bash.
	Orders librarium.Orders
	// Env is the environment that places the data-slates; nil reads the
	// servitor's own.
	Env librarium.Environment
}

func (o Options) env() librarium.Environment {
	if o.Env == nil {
		return os.LookupEnv
	}
	return o.Env
}

// Augur reads the augury of rite r, which must not be heretical. The error
// laments a data-slate that cannot be read; the augury is then read as if
// the rite had none.
func Augur(r *librarium.Rite, opts Options) (Augury, error) {
	slate, err := ReadSlate(opts.env(), r.Name)
	a := fromSlate(r.Name, slate)
	s := newSeer(r, slate)
	for i, step := range r.Liturgy {
		omen, taint := s.observe(i, step)
		if omen != nil {
			a.Omens = append(a.Omens, *omen)
		}
		if taint != nil {
			a.Taint = append(a.Taint, *taint)
		}
	}
	if r.Auspex != nil && !opts.ForgoAuspex {
		if aspect, ok := s.auspex(opts); ok {
			a.Omens = append(a.Omens, Omen{Auspex: true, Aspect: aspect})
		}
	}
	a.judge(slate)
	return a, err
}

// Heretic is the augury of the heretical rite named rite: its omens cannot
// be read, only its data-slate. The error laments a data-slate that cannot
// be read.
func Heretic(rite string, opts Options) (Augury, error) {
	slate, err := ReadSlate(opts.env(), rite)
	a := fromSlate(rite, slate)
	a.Standing = Heretical
	return a, err
}

func fromSlate(rite string, slate *Slate) Augury {
	a := Augury{Rite: rite}
	if slate != nil {
		a.Inscriptions, a.LastRite = slate.Inscriptions, slate.LastRite
	}
	return a
}

// judge names the aspect and standing from the omens, the taint and the
// data-slate.
func (a *Augury) judge(slate *Slate) {
	recorded := ""
	if slate != nil {
		recorded = slate.Aspect
	}
	agreed, agree := agreement(a.Omens)
	switch {
	case len(a.Omens) == 0:
		a.Aspect, a.Standing = recorded, Performed
		if recorded == "" {
			a.Standing = Dormant
		}
	case !agree:
		a.Standing = Corrupted
	default:
		a.Aspect, a.Standing = agreed, Performed
		if recorded != "" && recorded != agreed {
			a.Desecrated, a.Former, a.Standing = true, recorded, Desecrated
		}
	}
	if len(a.Taint) > 0 && a.Standing != Corrupted {
		a.Standing = Tainted
	}
	if slate != nil && slate.LastRite != nil && slate.LastRite.Verdict == invocation.Faltered {
		a.Standing = Corrupted
	}
}

// agreement returns the aspect every omen names, and whether they agree.
func agreement(omens []Omen) (string, bool) {
	for _, o := range omens {
		if o.Aspect != omens[0].Aspect {
			return "", false
		}
	}
	if len(omens) == 0 {
		return "", false
	}
	return omens[0].Aspect, true
}
