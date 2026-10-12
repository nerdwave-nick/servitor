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
}

// raise raises the exalted servitor for the invocation of r prepared in
// res, through the intercessor of x, and hears it accept the session: the
// handshake names the master, the rite, the aspect, the sigil of its plan,
// the former aspect and the words of the inscriptions. A proof the master
// dismissed or failed is *Dismissed, an exalted servitor of another
// forging *exalted.Mismatch, one that refused *Spurned; each is a refusal
// before anything was touched.
func (s *Servitor) raise(r *librarium.Rite, res *Result, x *exaltedPlan, summoner Summoner) (*exalter, error) {
	aspect := res.Options.Aspect
	e := &exalter{deed: invocationOf(r, aspect), intercessor: x.intercessor.Name}
	c, err := raiseConduit(e.deed, e.intercessor, x.intercessor.Command(r.Name, x.sigil, aspect), summoner)
	if err != nil {
		return nil, err
	}
	e.c = c
	inscriptions := map[string]string{}
	maps.Copy(inscriptions, res.Options.Inscriptions)
	_ = c.ch.Send(exalted.Handshake{Protocol: exalted.Protocol, UID: s.sanctions().UID, Rite: r.Name, Aspect: aspect,
		Sigil: x.sigil, Former: res.Options.Former, Inscriptions: inscriptions}) // an exalted servitor never raised answers nothing
	reply, err := c.ch.Answer()
	if err != nil {
		c.silence()
		return nil, unanswered(e.deed, e.intercessor, err, c.end())
	}
	if reply.Verdict != exalted.Accepted {
		c.silence()
		_ = c.end()
		return nil, &Spurned{Deed: e.deed, Lament: reply.Lament}
	}
	if want := r.ExaltedVerses(); !slices.Equal(reply.Verses, want) {
		e.abandon()
		return nil, &Spurned{Deed: e.deed, Lament: fmt.Sprintf("it would perform the verses %v of the sanction, yet "+
			"the rite holds the exalted verses %v", reply.Verses, want)}
	}
	return e, nil
}

// abandon ends an exalted servitor that performed nothing: it hears the
// end of its channel.
func (e *exalter) abandon() {
	e.c.silence()
	_ = e.c.end()
}

// ask speaks the request to the exalted servitor and hears its reply, which
// must answer it; a channel that fails, or a reply that answers another
// word, loses the exalted servitor.
func (e *exalter) ask(req exalted.Request, to string, verse int) (exalted.Reply, error) {
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
	case r.To != to || r.Verse != verse:
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

// Perform bids the exalted servitor perform the exalted verse v.
func (e *exalter) Perform(v invocation.Verse) (string, bool, error) {
	n := v.Number
	r, err := e.ask(exalted.Request{Perform: &n}, "perform", n)
	switch {
	case err != nil:
		return "", false, err
	case r.Verdict == exalted.Triumph:
		return r.Heard, true, nil
	case r.Verdict == exalted.Fell:
		return r.Heard, r.Changed, fmt.Errorf("the exalted servitor: %s", r.Lament)
	}
	e.lose(fmt.Errorf("%w: the exalted servitor answered the verse %d with %q", errLost, n, r.Verdict))
	return "", false, e.lost
}

// Revert bids the exalted servitor revert the exalted verse v. When it is
// lost, it reverts every verse it performed by itself, and how it ended
// tells whether every reversion triumphed.
func (e *exalter) Revert(v invocation.Verse) (string, bool, error) {
	n := v.Number
	r, err := e.ask(exalted.Request{Revert: &n}, "revert", n)
	switch {
	case err != nil:
		return "", true, e.revertedAlone()
	case r.Verdict == exalted.Reverted:
		return r.Heard, r.Changed, nil
	case r.Verdict == exalted.Faltered:
		return r.Heard, true, errors.New(r.Lament) // the exalted servitor tells that the reversion fell
	}
	e.lose(fmt.Errorf("%w: the exalted servitor answered the reversion of verse %d with %q", errLost, n, r.Verdict))
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
	r, err := e.ask(exalted.Request{Finish: &finish}, "finish", 0)
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
