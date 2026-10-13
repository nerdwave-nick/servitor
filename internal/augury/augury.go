// Package augury reads which aspect a rite stands in, and how it stands,
// and keeps each rite's data-slate.
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
//
// A state rite of pattern Mark II reads its subjects instead (readSubjects):
// each yields every aspect it fits, a machine-spirit through the scan or
// the trials of its auspex, and the omens agree by intersection
// (intersect). An exalted auspex is never spoken here.
//
// An edict bears no aspect and yields no omen: it lies dormant until first
// proclaimed, stands proclaimed after a last invocation that ended in
// triumph or was reverted, and corrupted after one that faltered.
package augury

import (
	"context"
	"os"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// Standing is how a rite stands.
type Standing string

// The standings, from the gravest down. A rite holds the gravest that
// applies: heretical, corrupted, tainted, desecrated, then performed or
// dormant. An edict is heretical, corrupted, proclaimed or dormant.
const (
	Heretical  Standing = "heretical"  // its scripture is heresy
	Corrupted  Standing = "corrupted"  // its omens disagree, or its last invocation faltered
	Tainted    Standing = "tainted"    // other hands altered a sanctum's scripture
	Desecrated Standing = "desecrated" // its omens agree on an aspect the data-slate did not record
	Performed  Standing = "performed"  // its omens, or else its data-slate, name its aspect
	Dormant    Standing = "dormant"    // neither omen nor data-slate names an aspect
	// Proclaimed is for edicts alone: the last invocation of the edict
	// ended in triumph, or was reverted.
	Proclaimed Standing = "proclaimed"
)

// Heed is what the rite's auspex gave the augury; it marshals as the
// augury's and the census's "auspex".
type Heed string

// The heeds of the auspex.
const (
	NoAuspex Heed = ""        // the rite keeps no auspex, is an edict, or is heretical
	Forgone  Heed = "forgone" // the auspex was left unawakened on request
	Heeded   Heed = "heeded"  // the auspex named one of the rite's aspects
	Mute     Heed = "mute"    // the auspex fell, outlasted its patience, or named no aspect
	// Awaited is for a census kept as a vigil alone: the rite is told
	// before its auspex settled, and will be told again once it has. No
	// augury bears it.
	Awaited Heed = "awaited"
)

// Augury is what the servitor reads of one rite; it marshals as the JSON of
// `servitor augury`.
type Augury struct {
	Rite     string
	Edict    bool   // the rite is an edict, which bears no aspect
	Aspect   string // "" when unknown
	Standing Standing
	// Sanction is how the rite stands towards its sanction; the augury
	// leaves it None, for its reader to judge.
	Sanction   sanction.Standing
	Auspex     Heed   // what the auspex gave; its omen, when heeded, stands among Omens
	Desecrated bool   // the omens agree on an aspect the data-slate did not record
	Former     string // the data-slate's aspect, when desecrated
	// Inscriptions are those on the data-slate: a rite never invoked
	// through the servitor has none.
	Inscriptions map[string]string
	LastRite     *LastRite // nil when the rite was never invoked
	Omens        []Omen    // in written order, the auspex's last
	Taint        []Taint   // in written order
	// MarkII marks the augury of a state rite of pattern Mark II, whose
	// omens are named by subject and agree by intersection.
	MarkII bool
	// Fitting are the aspects every omen of a state rite fits, in the order
	// of the rite's aspects; empty when they fit none, or when no subject
	// yielded an omen.
	Fitting []string
	// Canon are the inscriptions the rite declares, in written order; a
	// heretical rite declares none.
	Canon []Declaration
}

// Omen is what one step, or the auspex, reveals about the aspect; or, for
// a state rite of pattern Mark II, what one subject reveals.
type Omen struct {
	// Verse names the step: its verse, its own key, and what that key holds
	// rendered for Aspect (not resolved). Zero for the auspex, and for a
	// subject.
	invocation.Verse
	Auspex bool // the omen of the rite's auspex
	// Aspect is the aspect the omen names; for a subject, the one aspect it
	// fits, "" when it fits several.
	Aspect string

	Subject     string   // the subject's name; "" for a step or the auspex
	Designation string   // the subject's designation
	Read        string   // the omen a machine-spirit was read to show; "" when no single one
	Aspects     []string // every aspect the subject fits, in the order of the rite's
}

// Taint is a sanctum whose scripture other hands altered: it no longer
// holds the scripture of the aspect its marker records.
type Taint struct {
	invocation.Verse
	Subject string // the sanctum subject's name; "" for a step
}

// Options are what an augury is read with.
type Options struct {
	// Target is the aspect being invoked when the augury is read during an
	// invocation; the auspex sees it as {{aspect}} and SERVITOR_ASPECT.
	// Outside an invocation it is "".
	Target string
	// ForgoAuspex leaves the auspex unawakened, for answers that must come
	// at once (shell completion, --spare-auspex); a rite keeping one is
	// then marked Forgone.
	ForgoAuspex bool
	// Orders are the settings resolved; the auspex is spoken in the rite's
	// tongue, else theirs, else bash.
	Orders librarium.Orders
	// Env is the environment that places the data-slates; nil reads the
	// servitor's own.
	Env librarium.Environment
	// Places bound how many processes of auspexes labour at once, shared
	// by every augury read with them; nil bounds none.
	Places Places
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
	return AugurContext(context.Background(), r, opts)
}

// AugurContext is Augur whose auspex labours only while ctx lasts: when ctx
// ends, its whole process group is slain and it is mute.
func AugurContext(ctx context.Context, r *librarium.Rite, opts Options) (Augury, error) {
	slate, err := ReadSlate(opts.env(), r.Name)
	a := fromSlate(r.Name, slate)
	a.Canon = CanonOf(r)
	if r.Edict {
		a.Edict = true
		a.proclamation(slate)
		return a, err
	}
	s := newSeer(r, slate)
	if r.MarkII {
		a.MarkII = true
		a.readSubjects(ctx, s, opts)
		a.intersect(slate)
		return a, err
	}
	for i, step := range r.Liturgy {
		omen, taint := s.observe(i, step)
		if omen != nil {
			a.Omens = append(a.Omens, *omen)
		}
		if taint != nil {
			a.Taint = append(a.Taint, *taint)
		}
	}
	switch {
	case r.Auspex == nil:
	case opts.ForgoAuspex:
		a.Auspex = Forgone
	default:
		a.Auspex = Mute
		if aspect, ok := s.auspex(ctx, opts); ok {
			a.Auspex = Heeded
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
	a.weigh(slate)
}

// weigh lays the graver standings upon the one the omens name: taint,
// unless the omens disagree, and a last invocation that faltered.
func (a *Augury) weigh(slate *Slate) {
	if len(a.Taint) > 0 && a.Standing != Corrupted {
		a.Standing = Tainted
	}
	if slate != nil && slate.LastRite != nil && slate.LastRite.Verdict == invocation.Faltered {
		a.Standing = Corrupted
	}
}

// proclamation judges the standing of an edict from its data-slate alone:
// it seeks no omen and awakens no auspex, for an edict brings the machine
// into no aspect.
func (a *Augury) proclamation(slate *Slate) {
	switch {
	case slate == nil || slate.LastRite == nil:
		a.Standing = Dormant
	case slate.LastRite.Verdict == invocation.Faltered:
		a.Standing = Corrupted
	default:
		a.Standing = Proclaimed
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
