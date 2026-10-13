// Package augury reads which aspect a rite stands in, and how it stands,
// and keeps each rite's data-slate.
//
// A rite of aspects is read by its subjects (readSubjects): a sanctum yields
// the aspect its marker records, a transcription every aspect whose
// scripture its vessel holds (none when absent and no aspect writes null),
// a tether every aspect whose rendered anchor its bound name leads to, and
// a machine-spirit every aspect it fits by the scan or the trials of its
// auspex. Omens are rendered for every aspect with the inscriptions the
// data-slate recorded for its aspect, or else the decrees of the aspect.
// The omens agree by intersection (intersect): none corrupts the rite, one
// names its aspect, and a data-slate that recorded another desecrates it.
// Without omen the data-slate names the aspect, and without either the rite
// lies dormant. A sanctum whose scripture other hands altered taints the
// rite, and a last invocation that faltered corrupts it.
//
// An exalted auspex is never spoken here. What each subject showed is kept
// for the invocation that plans by it (Augury.Readings), and ReadSubject
// reads one subject anew to prove it once set: by the trials of its auspex
// when it holds them, though a scan beside them reads. Where the live omen
// of a machine-spirit is not heard — its auspex exalted or forgone — the
// omen its data-slate recorded at the last invocation serves, marked stale
// (remembered), unless recorded in another boot or no longer declared; a
// stale omen alone never desecrates the rite. An invocation never trusts
// one.
//
// An edict bears no aspect and yields no omen: it lies dormant until first
// proclaimed, stands proclaimed after a last invocation that ended in
// triumph or was reverted, and corrupted after one that faltered.
package augury

import (
	"context"
	"os"
	"slices"
	"time"

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
	Corrupted  Standing = "corrupted"  // its omens fit no aspect together, or its last invocation faltered
	Tainted    Standing = "tainted"    // other hands altered a sanctum's scripture
	Desecrated Standing = "desecrated" // its omens agree on an aspect the data-slate did not record
	Performed  Standing = "performed"  // its omens, or else its data-slate, name its aspect
	Dormant    Standing = "dormant"    // neither omen nor data-slate names an aspect
	// Proclaimed is for edicts alone: the last invocation of the edict
	// ended in triumph, or was reverted.
	Proclaimed Standing = "proclaimed"
)

// Heed is what the auspexes of the rite's machine-spirits gave the augury;
// it marshals as the augury's and the census's "auspex".
type Heed string

// The heeds of the auspex.
const (
	NoAuspex Heed = ""        // no subject bears an auspex the augury may speak, or the rite is an edict or heretical
	Forgone  Heed = "forgone" // the auspexes were left unawakened on request
	Heeded   Heed = "heeded"  // every auspex read an omen fitting an aspect
	Mute     Heed = "mute"    // an auspex fell, outlasted its patience, or read no declared omen
	// Awaited is for a census kept as a vigil alone: the rite is told
	// before its auspex settled, and will be told again once it has. No
	// augury reads it; the vigil marks the readings it tells so.
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
	Auspex     Heed   // what the auspexes gave; their omens stand among Omens
	Desecrated bool   // the omens agree on an aspect the data-slate did not record
	Former     string // the data-slate's aspect, when desecrated
	// Inscriptions are those on the data-slate: a rite never invoked
	// through the servitor has none.
	Inscriptions map[string]string
	LastRite     *LastRite // nil when the rite was never invoked
	Omens        []Omen    // one per subject that yielded one, in written order
	Taint        []Taint   // in written order
	// Fitting are the aspects every omen of a rite of aspects fits, in the order
	// of the rite's aspects; empty when they fit none, or when no subject
	// yielded an omen.
	Fitting []string
	// Readings are what every subject of a rite was read to show, in
	// written order, mute ones too: what an invocation plans by. They are
	// never told.
	Readings []invocation.Reading
	// Canon are the inscriptions the rite declares, in written order; a
	// heretical rite declares none.
	Canon []Declaration
}

// Omen is what one subject reveals about the aspect.
type Omen struct {
	// Aspect is the one aspect the subject fits, "" when it fits several.
	Aspect string

	Subject     string   // the subject's name
	Designation string   // the subject's designation
	Read        string   // the omen a machine-spirit was read to show; "" when no single one
	Aspects     []string // every aspect the subject fits, in the order of the rite's
	// Stale marks an omen taken from the data-slate's record, where the
	// live omen of the subject was not heard; ReadAt is when it was read.
	Stale  bool
	ReadAt time.Time
}

// Stale tells whether any omen of the augury was taken from the data-slate's
// record.
func (a Augury) Stale() bool {
	return slices.ContainsFunc(a.Omens, func(o Omen) bool { return o.Stale })
}

// Taint is a sanctum subject whose scripture other hands altered: it no
// longer holds the scripture of the aspect its marker records.
type Taint struct {
	Subject string // the sanctum subject's name
	Vessel  string // its vessel, rendered for that aspect (not resolved)
}

// Options are what an augury is read with.
type Options struct {
	// Target is the aspect being invoked when the augury is read during an
	// invocation; every auspex sees it as {{aspect}} and SERVITOR_ASPECT.
	// Outside an invocation it is "".
	Target string
	// ForgoAuspex leaves every auspex unawakened, for answers that must
	// come at once (shell completion, --spare-auspex); a rite keeping one is
	// then marked Forgone.
	ForgoAuspex bool
	// Orders are the settings resolved; an auspex is spoken in its
	// subject's tongue, else the rite's, else theirs, else bash.
	Orders librarium.Orders
	// Env is the environment that places the data-slates; nil reads the
	// servitor's own.
	Env librarium.Environment
	// Places bound how many processes of auspexes labour at once, shared
	// by every augury read with them; nil bounds none.
	Places Places
	// Inscriptions are those of the invocation into Target: the omens of
	// Target are rendered with them before any other, as they will be set.
	// Outside an invocation they are nil.
	Inscriptions map[string]string
	// Foreseen marks the augury of a foresight into Target, which performs
	// nothing: a privileged auspex, which only the exalted servitor speaks
	// at invocation, shows the omen the data-slate recorded, stale, as it
	// does outside an invocation.
	Foreseen bool
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

// AugurContext is Augur whose auspexes labour only while ctx lasts: when
// ctx ends, their whole process groups are slain and they are mute.
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
	s.invoked(opts)
	a.readSubjects(ctx, s, opts)
	a.intersect(slate)
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
