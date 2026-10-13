package exalted

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// undo reverts the last verse performed and journals it.
func (s *session) undo() Reply {
	d := s.performed[len(s.performed)-1]
	s.performed = s.performed[:len(s.performed)-1]
	r := Reply{To: "revert", Verse: d.verse.Verse, Subject: d.verse.Subject, Observance: d.verse.Observance,
		Verdict: Reverted}
	undo := d.undo
	if d.restore != nil {
		undo = func(ctx context.Context) (heard string, err error) {
			heard, r.Omens, err = d.restore(ctx, d.omen)
			return heard, err
		}
	}
	if undo != nil {
		heard, err := undo(context.Background())
		r.Changed, r.Heard = true, heard
		if err != nil {
			r.Verdict, r.Lament = Faltered, err.Error()
		}
	}
	s.record(Entry{Verse: r.Verse, Subject: r.Subject, Observance: r.Observance, Deed: "revert", Verdict: r.Verdict,
		Lament: r.Lament})
	return r
}

// undoAll reverts every verse performed, from the last back to the first.
func (s *session) undoAll() (reversions []Reply, faltered bool) {
	for len(s.performed) > 0 {
		r := s.undo()
		faltered = faltered || r.Verdict == Faltered
		reversions = append(reversions, r)
	}
	return reversions, faltered
}

// abort reverts every verse performed and tells it.
func (s *session) abort() int {
	reversions, faltered := s.undoAll()
	r := Reply{To: "abort", Verdict: Reverted, Reversions: reversions}
	if faltered {
		r.Verdict = Faltered
	}
	s.record(Entry{Deed: "abort", Verdict: r.Verdict})
	_ = s.reply(r)
	return mark(faltered)
}

// fall ends the session upon why, an end of the channel, a silence, a
// request out of order or a word that could not be spoken: every verse
// performed is reverted.
func (s *session) fall(why error) int {
	var v violation
	if errors.As(why, &v) {
		return s.violated(v.to, v.error)
	}
	if errors.Is(why, io.EOF) {
		why = errors.New("the channel ended before the session was finished")
	}
	_, faltered := s.undoAll()
	s.record(Entry{Deed: "fall", Verdict: map[bool]Verdict{false: Reverted, true: Faltered}[faltered],
		Lament: why.Error()})
	return mark(faltered)
}

// violated refuses a line of an accepted session, and ends it.
func (s *session) violated(to string, why error) int {
	s.record(Entry{Deed: "refuse", Verdict: Refused, Lament: why.Error()})
	_ = s.reply(Reply{To: to, Verdict: Refused, Lament: why.Error()})
	return s.fall(errors.New("a line was refused"))
}

// refuse refuses the session before anything was touched.
func (s *session) refuse(why error) int {
	if errors.Is(why, io.EOF) {
		why = errors.New("the channel ended before the handshake")
	}
	s.record(Entry{Deed: "accept", Verdict: Refused, Lament: why.Error()})
	if s.cfg.Lament != nil {
		fmt.Fprintln(s.cfg.Lament, "servitor ✠ "+why.Error())
	}
	_ = s.reply(Reply{To: "handshake", Verdict: Refused, Protocol: Protocol, Lament: why.Error()})
	return markRefused
}

func mark(faltered bool) int {
	if faltered {
		return markFaltered
	}
	return markReverted
}
