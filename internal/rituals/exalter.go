package rituals

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/nerdwave-nick/servitor/internal/exalted"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// exalter is the exalted servitor raised for one invocation, through the
// intercessor and one proof of the master, as the invocation's Exalter: it
// bids it perform and revert each exalted verse in its turn over their
// channel, and finish.
type exalter struct {
	deed        string
	intercessor string
	c           *conduit
	// lost tells why the channel failed mid-liturgy: the exalted servitor
	// hears no more, and reverts by itself what it performed.
	lost    error
	reverts error // how the exalted servitor ended reverting by itself
	settled bool
	laments []error
	// proofs are the omens the exalted servitor told a privileged subject
	// showed when it last set or restored it, by subject, until the
	// invocation reads them (see reader).
	proofs map[string][]string
}

// raise raises the exalted servitor for the invocation of r prepared in
// res, through the intercessor of x, and hears it accept the session: the
// handshake names the master, the rite, the aspect, the sigil of its plan,
// the former aspect and the words of the inscriptions. A proof the master
// dismissed or failed is *Dismissed, an exalted servitor of another
// forging *exalted.Mismatch, one that refused *Spurned; each is a refusal
// before anything was touched. The exalted servitor is summoned as ex
// summons it.
func (s *Servitor) raise(r *librarium.Rite, res *Result, x *exaltedPlan, ex Exaltation) (*exalter, error) {
	aspect := res.Options.Aspect
	e := &exalter{deed: invocationOf(r, aspect), intercessor: x.intercessor.Name}
	c, err := raiseConduit(e.deed, e.intercessor, x.intercessor.Command(r.Name, x.sigil, aspect), ex)
	if err != nil {
		return nil, err
	}
	e.c = c
	inscriptions := map[string]string{}
	maps.Copy(inscriptions, res.Options.Inscriptions)
	_ = c.ch.Send(exalted.Handshake{Protocol: exalted.Protocol, UID: s.sanctions().UID, Rite: r.Name, Aspect: aspect,
		Sigil: x.sigil, Former: res.Options.Former, Inscriptions: inscriptions}) // an exalted servitor never raised answers nothing
	reply, err := c.answer()
	if err != nil {
		c.silence()
		return nil, unanswered(e.deed, e.intercessor, err, c.end())
	}
	if reply.Verdict != exalted.Accepted {
		c.silence()
		_ = c.end()
		return nil, &Spurned{Deed: e.deed, Lament: reply.Lament}
	}
	if why := unlike(r, x, aspect, reply); why != "" {
		e.abandon()
		return nil, &Spurned{Deed: e.deed, Lament: why}
	}
	return e, nil
}

// unlike tells how the plan the exalted servitor accepted, as its reply
// names it, differs from the plan of the aspect the rite holds: its
// exalted verses, or the exalted subjects and observances of a state rite;
// "" when they are one.
func unlike(r *librarium.Rite, x *exaltedPlan, aspect string, reply exalted.Reply) string {
	if !r.MarkII || r.Edict {
		if want := r.ExaltedVerses(); !slices.Equal(reply.Verses, want) {
			return fmt.Sprintf("it would perform the verses %v of the sanction, yet the rite holds the exalted verses %v",
				reply.Verses, want)
		}
		return ""
	}
	var subjects, observances []string
	if plan, ok := x.unit.Narrow(aspect); ok {
		for _, v := range plan.Aspects[0].Subjects {
			subjects = append(subjects, v.Subject)
		}
		for _, v := range plan.Aspects[0].Observances {
			observances = append(observances, v.Observance)
		}
	}
	if !slices.Equal(reply.Subjects, subjects) || !slices.Equal(reply.Observances, observances) {
		return fmt.Sprintf("it would perform the subjects %q and observances %q of the sanction, yet the rite holds "+
			"the exalted subjects %q and observances %q", reply.Subjects, reply.Observances, subjects, observances)
	}
	return ""
}

// abandon ends an exalted servitor that performed nothing: it hears the
// end of its channel.
func (e *exalter) abandon() {
	e.c.silence()
	_ = e.c.end()
}

// itemOf names the verse, or the subject or observance of a state rite, v
// is.
func itemOf(v invocation.Verse) exalted.Item {
	return exalted.Item{Verse: v.Number, Subject: v.Subject, Observance: v.Observance}
}

// ask speaks the request to the exalted servitor and hears its reply, which
// must answer it, of the item it names; a channel that fails, or a reply
// that answers another word, loses the exalted servitor.
func (e *exalter) ask(req exalted.Request, to string, it exalted.Item) (exalted.Reply, error) {
	if e.lost != nil {
		return exalted.Reply{}, e.lost
	}
	var r exalted.Reply
	err := e.c.ch.Send(req)
	if err == nil {
		err = e.c.ch.Receive(&r)
	}
	switch {
	case err != nil:
		e.lose(fmt.Errorf("%w: %v", errLost, err))
	case r.Verdict == exalted.Refused:
		e.lose(fmt.Errorf("%w: the exalted servitor refused the word to %s: %s", errLost, to, r.Lament))
	case r.To != to || r.Verse != it.Verse || r.Subject != it.Subject || r.Observance != it.Observance:
		e.lose(fmt.Errorf("%w: the exalted servitor answered another word", errLost))
	}
	return r, e.lost
}

// lose marks the exalted servitor lost for why.
func (e *exalter) lose(why error) {
	if e.lost == nil {
		e.lost = why
	}
}

// Perform bids the exalted servitor perform the exalted verse, subject or
// observance v.
func (e *exalter) Perform(v invocation.Verse) (string, bool, error) { return e.perform(itemOf(v)) }

// Enact bids the exalted servitor set the exalted machine-spirit v, naming
// the omen it is restored to.
func (e *exalter) Enact(v invocation.Verse, restore string) (string, bool, error) {
	it := itemOf(v)
	it.Omen = restore
	return e.perform(it)
}

func (e *exalter) perform(it exalted.Item) (string, bool, error) {
	r, err := e.ask(exalted.Request{Perform: &it}, "perform", it)
	e.prove(it, r, err == nil && r.Verdict == exalted.Triumph)
	switch {
	case err != nil:
		return "", false, err
	case r.Verdict == exalted.Triumph:
		return r.Heard, true, nil
	case r.Verdict == exalted.Fell:
		return r.Heard, r.Changed, fmt.Errorf("the exalted servitor: %s", r.Lament)
	}
	e.lose(fmt.Errorf("%w: the exalted servitor answered %s with %q", errLost, it.Named(), r.Verdict))
	return "", false, e.lost
}

// prove keeps the omens the reply r told the subject it names showed, when
// it proved it (proven); every earlier proof of it is struck.
func (e *exalter) prove(it exalted.Item, r exalted.Reply, proven bool) {
	if it.Subject == "" {
		return
	}
	delete(e.proofs, it.Subject)
	if proven && r.Omens != nil {
		if e.proofs == nil {
			e.proofs = map[string][]string{}
		}
		e.proofs[it.Subject] = r.Omens
	}
}

// Revert bids the exalted servitor revert the exalted verse, subject or
// observance v. When it is lost, it reverts every item it performed by
// itself, and how it ended tells whether every reversion triumphed.
func (e *exalter) Revert(v invocation.Verse) (string, bool, error) { return e.revert(itemOf(v)) }

// Restore bids the exalted servitor restore the exalted machine-spirit v to
// omen.
func (e *exalter) Restore(v invocation.Verse, omen string) (string, bool, error) {
	it := itemOf(v)
	it.Omen = omen
	return e.revert(it)
}

func (e *exalter) revert(it exalted.Item) (string, bool, error) {
	r, err := e.ask(exalted.Request{Revert: &it}, "revert", exalted.Item{Verse: it.Verse, Subject: it.Subject,
		Observance: it.Observance})
	e.prove(it, r, err == nil && r.Verdict == exalted.Reverted)
	switch {
	case err != nil:
		return "", true, e.revertedAlone()
	case r.Verdict == exalted.Reverted:
		return r.Heard, r.Changed, nil
	case r.Verdict == exalted.Faltered:
		return r.Heard, true, errors.New(r.Lament) // the exalted servitor tells that the reversion fell
	}
	e.lose(fmt.Errorf("%w: the exalted servitor answered the reversion of %s with %q", errLost, it.Named(), r.Verdict))
	return "", true, e.revertedAlone()
}

// revertedAlone awaits the lost exalted servitor, which reverts by itself
// every verse it performed: its death-mark 1 tells that every reversion
// triumphed; any other end leaves the reversion unknown.
func (e *exalter) revertedAlone() error {
	if !e.settled {
		e.settled = true
		e.c.silence()
		ended := e.c.end()
		var exit interface{ ExitCode() int } // as *exec.ExitError tells it
		if !errors.As(ended, &exit) || exit.ExitCode() != 1 {
			e.reverts = fmt.Errorf("%v; it reverts by itself what it performed, yet how it ended (%v) does not "+
				"tell that every reversion triumphed — behold the journal of the machine: journalctl -t servitor",
				e.lost, ended)
		}
	}
	return e.reverts
}

// Finish bids the exalted servitor finish, keeping what it performed, and
// awaits its end.
func (e *exalter) Finish() error {
	if e.lost != nil {
		_ = e.revertedAlone()
		return e.lost
	}
	finish := true
	r, err := e.ask(exalted.Request{Finish: &finish}, "finish", exalted.Item{})
	if err == nil && r.Verdict != exalted.Finished {
		e.lose(fmt.Errorf("%w: the exalted servitor answered the finish with %q", errLost, r.Verdict))
		err = e.lost
	}
	if err != nil {
		_ = e.revertedAlone()
		return err
	}
	if ended := e.c.end(); ended != nil {
		e.laments = append(e.laments, fmt.Errorf("the intercessor %s ended amiss after the exalted servitor "+
			"finished: %w", e.intercessor, ended))
	}
	return nil
}

// short is the sigil as the chronicle names it.
func short(sigil string) string { return sanction.Short(sigil) }
