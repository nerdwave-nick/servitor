package invocation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// Exalter performs the exalted verses of one invocation — or the exalted
// subjects and observances of a rite of aspects that change:
// the exalted servitor, raised for it alone with the machine's own
// authority, which performs each from the sanction of its master. The
// invoker's servitor never performs an exalted verse itself, nor reverts
// one.
type Exalter interface {
	// Perform performs the exalted verse v, the next of the plan. performed
	// tells that it stands performed, to be reverted should the invocation
	// fall: it triumphed, or it fell having changed the machine all the
	// same. heard are the words its speech uttered.
	Perform(v Verse) (heard string, performed bool, err error)
	// Revert reverts the exalted verse v, the last performed and not yet
	// reverted; changed is false when there was nothing to undo.
	Revert(v Verse) (heard string, changed bool, err error)
	// Enact sets the exalted machine-spirit v of a rite of aspects
	// by the enact words of the omen of the aspect invoked, as
	// Perform performs a verse; restore is the omen it is restored to
	// should the exalted servitor revert it by itself, "" when none is
	// known.
	Enact(v Verse, restore string) (heard string, performed bool, err error)
	// Restore reverts the exalted machine-spirit v, the last performed and
	// not yet reverted, by the enact words of omen; "" when none is known,
	// and it cannot be restored.
	Restore(v Verse, omen string) (heard string, changed bool, err error)
	// Finish ends the work of the exalted servitor, keeping what it
	// performed. When it is not heard, the exalted servitor reverts every
	// verse it performed by itself.
	Finish() error
}

// exaltable is a step the exalted servitor performs: it is lent the
// exalted servitor before the performance.
type exaltable interface {
	lend(x Exalter)
	// exaltedAt is the JSON pointer of the step's "exalted"; "" when the
	// step is common after all.
	exaltedAt() string
}

// wrapper is a step that performs another, which it wraps.
type wrapper interface{ wrapped() performer }

// exaltedOf is the exalted step that step is or wraps; nil when it is
// common.
func exaltedOf(step performer) exaltable {
	for step != nil {
		if e, ok := step.(exaltable); ok && e.exaltedAt() != "" {
			return e
		}
		w, ok := step.(wrapper)
		if !ok {
			return nil
		}
		step = w.wrapped()
	}
	return nil
}

// exaltedStep is an exalted step made ready by the pre-flight. It is
// planned as any other, so that it can be foreseen and the steps after it
// are planned upon what it will have done, yet the invoker's servitor
// never performs what it planned: the exalted servitor alone performs and
// reverts the verse, from the sanction.
type exaltedStep struct {
	planned      performer // the step as the pre-flight planned it, never performed
	at           string    // the JSON pointer of its "exalted"
	x            Exalter
	performed    bool
	said, unsaid string
}

func (s *exaltedStep) verse() Verse       { return s.planned.verse() }
func (s *exaltedStep) foresee() Foresight { return s.planned.foresee() }
func (s *exaltedStep) lend(x Exalter)     { s.x = x }
func (s *exaltedStep) exaltedAt() string  { return s.at }

func (s *exaltedStep) perform(context.Context) error {
	if s.x == nil {
		return errNoExalter
	}
	heard, performed, err := s.x.Perform(s.verse())
	s.said, s.performed = heard, performed || err == nil
	return err
}

func (s *exaltedStep) revert() (bool, error) {
	if !s.performed {
		return false, nil
	}
	heard, changed, err := s.x.Revert(s.verse())
	s.unsaid = heard
	return changed || err != nil, err
}

func (s *exaltedStep) uttered() (said, unsaid string) { return s.said, s.unsaid }

// errNoExalter is an exalted verse reached without an exalted servitor;
// PerformExalted refuses before it can be.
var errNoExalter = fmt.Errorf("no exalted servitor was raised to perform it")

// PerformExalted is PerformContext with the exalted servitor x, which
// performs and reverts every exalted verse in its place in written order,
// and whose work is finished once the performance ends, however it ends.
// When the finish is not heard after a triumph, the exalted servitor
// reverts its verses by itself, so every other verse is reverted too and
// the invocation falls at its last exalted verse. Without an exalted
// servitor an invocation holding exalted verses is refused, and nothing is
// performed: the invoker's servitor never performs one itself.
func (inv *Invocation) PerformExalted(ctx context.Context, x Exalter) (Outcome, error) {
	if len(inv.heresies) > 0 {
		return Outcome{}, inv.heresies
	}
	if inv.performed {
		return Outcome{}, ErrPerformed
	}
	if hs := inv.unexalted(x); len(hs) > 0 {
		return Outcome{}, hs
	}
	inv.performed = true
	var finish func() error
	if x != nil {
		finish = x.Finish
		for _, step := range inv.steps {
			if e := exaltedOf(step); e != nil {
				e.lend(x)
			}
		}
	}
	c := newCrier(inv.opts.Herald, inv.rite, values(inv.rite, inv.opts, inv.opts.Inscriptions), inv.steps)
	out := performAll(ctx, inv.steps, c, finish)
	if !inv.rite.Edict {
		out.Changed, out.Known = changed(out.Deeds), inv.known()
	}
	return out, nil
}

// unexalted denounces every exalted verse, subject or observance an
// invocation would perform when it has no exalted servitor x to perform
// them.
func (inv *Invocation) unexalted(x Exalter) Heresies {
	if x != nil {
		return nil
	}
	var hs Heresies
	for _, step := range inv.steps {
		e := exaltedOf(step)
		if e == nil {
			continue
		}
		v := step.verse()
		named := librarium.ExaltedItem{Verse: v.Number, Subject: v.Subject, Observance: v.Observance}.String()
		hs = append(hs, Heresy{Verse: v.Number, Finding: librarium.Finding{
			Severity: librarium.Heresy, Scripture: inv.rite.Path, Rite: inv.rite.Name,
			Position: inv.rite.Locate(e.exaltedAt()),
			Message: fmt.Sprintf("%s is exalted: it is performed with the machine's own authority alone, by "+
				"the exalted servitor from the sanction of its master, never by the invoker's servitor — and no "+
				"exalted servitor was raised for this invocation, so not one %s of %q was performed", named,
				inv.noun(), inv.rite.Name),
		}})
	}
	return hs
}

// noun is what the invocation performs: the subjects and observances of a
// rite of aspects, or the verses of an edict.
func (inv *Invocation) noun() string {
	if !inv.rite.Edict {
		return "subject or observance"
	}
	return "verse"
}

// Exalts reports whether the invocation performs anything exalted: an
// exalted verse, or an exalted subject or observance of a rite of aspects that
// changes. Only then is the exalted servitor raised.
func (inv *Invocation) Exalts() bool { return lastExalted(inv.steps) >= 0 }

// lastExalted is the index of the last exalted step; -1 when none is.
func lastExalted(steps []performer) int {
	for k := len(steps) - 1; k >= 0; k-- {
		if exaltedOf(steps[k]) != nil {
			return k
		}
	}
	return -1
}

// hiddenChange is the change an exalted step plans upon a vessel its
// invoker may not read (e, at real): its scripture, and so the change, are
// unknown. planned is the scripture to be written, when it is known before
// the vessel is read; seal the seal decreed, nil when none is.
func hiddenChange(real string, e entry, planned *string, seal *fs.FileMode, strike bool) VesselChange {
	c := VesselChange{Path: real, Hidden: true, Owner: e.owner, Size: e.size, Existed: e.form == scroll,
		Exists: !strike, BeforeSeal: e.mode, AfterSeal: e.mode}
	if e.form == veiled {
		c.Size = -1
	}
	if seal != nil && !strike {
		c.AfterSeal = *seal
	}
	if planned != nil && !strike {
		sum := sha256.Sum256([]byte(*planned))
		c.Digest = hex.EncodeToString(sum[:])
	}
	return c
}

// hidden reports whether the invoker may not read what stands as e.
func (e entry) hidden() bool { return e.veil || e.form == veiled }

// exaltedTongue denounces a tongue of exalted speech that is none of the
// tongues exalted speech knows, by the name of the program it truly is. A
// tongue spoken nowhere is denounced by speakable.
func (p *preflight) exaltedTongue(tongue string) bool {
	program, err := exec.LookPath(tongue)
	if err == nil {
		program, err = filepath.EvalSymlinks(program)
	}
	if err != nil {
		return true
	}
	if _, known := librarium.ExaltedTongue(program); !known {
		p.denounce("exalted", "the tongue %s of %s is %s, which exalted speech is never spoken in: it knows only %s, "+
			"each bidden to read no scripture of its own as it awakens — name one of them as its \"tongue\"", tongue,
			p.exaltedItem(), filepath.Base(program), librarium.ExaltedTongueNames)
		return false
	}
	return true
}

// exaltedItem names the exalted subject, observance or verse being
// examined, as laments name it.
func (p *preflight) exaltedItem() string {
	switch {
	case p.subject != "":
		return fmt.Sprintf("the exalted subject %q", p.subject)
	case p.observance != "":
		return fmt.Sprintf("the exalted observance %q", p.observance)
	}
	return fmt.Sprintf("the exalted verse %d", p.index+1)
}
