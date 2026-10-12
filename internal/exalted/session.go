package exalted

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/nerdwave-nick/servitor/internal/dominion"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// session is one session of the exalted servitor.
type session struct {
	cfg   Config
	out   *Channel
	lines <-chan heard
	uid   int
	argv  argv
	hs    Handshake
	sn    *sanction.Sanction
	plan  sanction.Plan

	next      int  // the index in the plan of the verse to perform next
	fallen    bool // a verse fell; nothing more is performed
	reverting bool // a verse was reverted; nothing more is performed
	performed []deed
	speaker   Speaker
	dominion  *dominion.Dominion
}

// deed is one verse performed; undo is nil when it changed nothing.
type deed struct {
	verse sanction.Verse
	undo  Undo
}

// heard is one line of the channel, or why none came.
type heard struct {
	line []byte
	err  error
}

var (
	errIdle   = errors.New("no word was heard for too long")
	errHalted = errors.New("the exalted servitor was halted")
)

// Serve is the exalted servitor roused by argv (after Mode), hearing its
// invoker on in and answering on out, in the world cfg; it returns the
// death-mark: 0 finished, 1 reverted, 2 refused before anything was
// touched, 3 faltered.
func Serve(args []string, in *os.File, out io.Writer, cfg Config) int {
	if cfg.Idle <= 0 {
		cfg.Idle = Machine().Idle
	}
	s := &session{cfg: cfg, out: NewChannel(nil, out)}
	if cfg.Exalted == nil || !cfg.Exalted() {
		return s.refuse(errors.New("the exalted servitor is roused only with the machine's own authority, by " +
			"an intercessor, and never by hand; nothing was touched"))
	}
	a, err := parseArgv(args)
	if err != nil {
		return s.refuse(err)
	}
	s.argv = a
	if s.uid, err = identify(in, cfg.Getenv); err != nil {
		return s.refuse(err)
	}
	done := make(chan struct{})
	defer close(done)
	s.lines = hear(in, done)
	defer s.close()
	if err := s.accept(); err != nil {
		return s.refuse(err)
	}
	verses := []int{}
	for _, v := range s.plan.Liturgy {
		verses = append(verses, v.Verse)
	}
	if err := s.reply(Reply{To: "handshake", Verdict: Accepted, Protocol: Protocol, Verses: verses}); err != nil {
		return s.fall(err)
	}
	s.record(Entry{Deed: "accept", Verdict: Accepted})
	return s.serve()
}

// hear hears the lines of in, one after another, until the channel ends
// or done closes.
func hear(in io.Reader, done <-chan struct{}) <-chan heard {
	lines := make(chan heard)
	go func() {
		c := NewChannel(in, nil)
		for {
			line, err := c.line()
			select {
			case lines <- heard{line, err}:
			case <-done:
				return
			}
			if err != nil {
				return
			}
		}
	}()
	return lines
}

// await awaits the next line, at most Idle.
func (s *session) await() ([]byte, error) {
	idle := time.NewTimer(s.cfg.Idle)
	defer idle.Stop()
	select {
	case h := <-s.lines:
		return h.line, h.err
	case <-idle.C:
		return nil, errIdle
	case <-s.cfg.Halt:
		return nil, errHalted
	}
}

// serve hears the requests of an accepted session until it ends.
func (s *session) serve() int {
	for {
		line, err := s.await()
		switch {
		case errors.Is(err, errHalted):
			return s.abort()
		case errors.Is(err, errOverlong):
			return s.violated("", err)
		case err != nil:
			return s.fall(err)
		}
		var r Request
		if err := Decode(line, &r); err != nil {
			return s.violated("", err)
		}
		switch {
		case r.Perform != nil && r.Revert == nil && r.Finish == nil && r.Abort == nil:
			if err := s.perform(*r.Perform); err != nil {
				return s.fall(err)
			}
		case r.Revert != nil && r.Perform == nil && r.Finish == nil && r.Abort == nil:
			if err := s.revert(*r.Revert); err != nil {
				return s.fall(err)
			}
		case r.Finish != nil && *r.Finish && r.Perform == nil && r.Revert == nil && r.Abort == nil:
			s.record(Entry{Deed: "finish", Verdict: Finished})
			_ = s.reply(Reply{To: "finish", Verdict: Finished})
			return markFinished
		case r.Abort != nil && *r.Abort && r.Perform == nil && r.Revert == nil && r.Finish == nil:
			return s.abort()
		default:
			return s.violated("", errors.New("a request names exactly one of perform, revert, finish and abort"))
		}
	}
}

// violation is a request out of its order, to the request named; the
// session ends upon it.
type violation struct {
	to string
	error
}

// perform performs verse n, which must be the next of the plan.
func (s *session) perform(n int) error {
	switch {
	case s.fallen || s.reverting:
		return violation{"perform", fmt.Errorf("verse %d is not performed: the session reverts what it performed", n)}
	case s.next >= len(s.plan.Liturgy):
		return violation{"perform", fmt.Errorf("verse %d is not performed: every exalted verse of the plan was", n)}
	case s.plan.Liturgy[s.next].Verse != n:
		return violation{"perform", fmt.Errorf("verse %d is not performed: the next exalted verse is %d", n,
			s.plan.Liturgy[s.next].Verse)}
	}
	v := s.plan.Liturgy[s.next]
	s.next++
	ctx, cancel := s.halting()
	defer cancel()
	r, d := s.act(ctx, v)
	if d != nil {
		s.performed = append(s.performed, *d)
	}
	s.fallen = r.Verdict == Fell
	s.record(Entry{Verse: n, Deed: "perform", Verdict: r.Verdict, Lament: r.Lament})
	return s.reply(r)
}

// revert reverts verse n, which must be the last performed and not yet
// reverted.
func (s *session) revert(n int) error {
	if len(s.performed) == 0 || s.performed[len(s.performed)-1].verse.Verse != n {
		return violation{"revert", fmt.Errorf("verse %d is not reverted: it is not the last verse performed and not yet "+
			"reverted", n)}
	}
	s.reverting = true
	return s.reply(s.undo())
}

// reply speaks r to the invoker.
func (s *session) reply(r Reply) error { return s.out.Send(r) }

// record journals e as an act of the session.
func (s *session) record(e Entry) {
	if s.cfg.Journal == nil {
		return
	}
	e.UID, e.Rite, e.Aspect, e.Sigil = s.uid, s.argv.rite, s.argv.aspect, s.argv.sigil
	if err := s.cfg.Journal.Record(e); err != nil && s.cfg.Lament != nil {
		fmt.Fprintln(s.cfg.Lament, "servitor ✠ the journal of the machine refuses an act: "+e.String())
	}
}

// halting is a context ended by a halt.
func (s *session) halting() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-s.cfg.Halt:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}

// close ends what the session raised.
func (s *session) close() {
	if s.speaker != nil {
		if err := s.speaker.Close(); err != nil && s.cfg.Lament != nil {
			fmt.Fprintln(s.cfg.Lament, "servitor ✠ "+err.Error())
		}
	}
	if s.dominion != nil {
		s.dominion.Relinquish()
	}
}
