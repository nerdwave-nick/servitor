package exalted

import (
	"errors"
	"fmt"
	"slices"

	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// The items of a state rite of pattern Mark II are performed in the order
// of its subjects: each changing subject in its place, its introits before
// it and its recessionals after it. Which subjects change is known only to
// the invoker, which reads them, and the plan holds only the exalted ones;
// so the exalted servitor skips forward but never back. A subject lies at
// its place; an observance at the place of a subject that names it — just
// before it as its introit, just after it as its recessional — the first
// such place not behind what was performed. Every item must lie after
// every item performed before it, and an observance is performed once.

// acceptance is the answer that accepts the session, naming its plan.
func (s *session) acceptance() Reply {
	r := Reply{To: "handshake", Verdict: Accepted, Protocol: Protocol}
	if s.sn.State() {
		r.Subjects, r.Observances = []string{}, []string{}
		for _, v := range s.plan.Subjects {
			r.Subjects = append(r.Subjects, v.Subject)
		}
		for _, v := range s.plan.Observances {
			r.Observances = append(r.Observances, v.Observance)
		}
		return r
	}
	r.Verses = []int{}
	for _, v := range s.plan.Liturgy {
		r.Verses = append(r.Verses, v.Verse)
	}
	return r
}

// performItem performs the item named: a verse, or a subject or observance
// of a state rite.
func (s *session) performItem(it Item) error {
	switch {
	case s.fallen || s.reverting:
		return violation{"perform", fmt.Errorf("%s is not performed: the session reverts what it performed",
			it.Named())}
	case s.sn.State():
		v, err := s.ordained(it)
		if err != nil {
			return violation{"perform", err}
		}
		return s.perform(v, it.Omen)
	case it.Subject != "" || it.Observance != "":
		return violation{"perform", fmt.Errorf("%s is not performed: the plan holds verses alone", it.Named())}
	}
	return s.performVerse(it.Verse)
}

// The phases of a subject's place: its introits, the subject, its
// recessionals.
const (
	beforeSubject = iota
	atSubject
	afterSubject
)

// position is where an item lies: at the place of a subject, in a phase.
type position struct{ place, phase int }

// after tells that p lies after q.
func (p position) after(q position) bool {
	return p.place > q.place || p.place == q.place && p.phase > q.phase
}

// ordained finds the item named in the plan of a state rite and moves the
// cursor to where it lies, which must lie after every item performed; an
// observance must be admitted by its "from" and not yet performed, and
// only a machine-spirit names an omen, which it must declare.
func (s *session) ordained(it Item) (sanction.Verse, error) {
	var (
		v     sanction.Verse
		found bool
		at    position
	)
	switch {
	case it.Verse != 0:
		return v, fmt.Errorf("%s is not performed: a state rite is performed by its subjects and observances, never "+
			"by verses", it.Named())
	case it.Subject != "" && it.Observance != "":
		return v, errors.New("an item names a subject or an observance, never both")
	case it.Subject != "":
		if v, found = find(s.plan.Subjects, it); !found {
			return v, unnamed(it)
		}
		at = position{place: v.Place, phase: atSubject}
		if it.Omen != "" {
			if err := restorable(v, it.Omen); err != nil {
				return v, err
			}
		}
		if !at.after(s.cursor) {
			return v, backwards(it)
		}
	default:
		if v, found = find(s.plan.Observances, it); !found {
			return v, unnamed(it)
		}
		if err := s.admitted(v, it); err != nil {
			return v, err
		}
		var ok bool
		if at, ok = s.placeOf(v); !ok {
			return v, backwards(it)
		}
		s.spoken[v.Observance] = true
	}
	s.cursor = at
	return v, nil
}

// admitted checks that the observance v may be performed: never before,
// admitted by its "from", and named without an omen.
func (s *session) admitted(v sanction.Verse, it Item) error {
	switch {
	case it.Omen != "":
		return fmt.Errorf("%s names the omen %q, yet only a machine-spirit is restored to an omen", it.Named(), it.Omen)
	case s.spoken[v.Observance]:
		return fmt.Errorf("%s is not performed: it was performed already, and an observance is spoken once",
			it.Named())
	case v.From != nil && (s.hs.Former == "" || !slices.Contains(v.From, s.hs.Former)):
		return fmt.Errorf("%s is not performed: its \"from\" admits no invocation out of the former aspect %q",
			it.Named(), s.hs.Former)
	}
	return nil
}

// placeOf is the first place of the observance v not behind the cursor:
// before a subject naming it as introit, after one naming it as
// recessional.
func (s *session) placeOf(v sanction.Verse) (position, bool) {
	var places []position
	for _, p := range v.Introit {
		places = append(places, position{place: p, phase: beforeSubject})
	}
	for _, p := range v.Recessional {
		places = append(places, position{place: p, phase: afterSubject})
	}
	var first *position
	for _, p := range places {
		if !s.cursor.after(p) && (first == nil || first.after(p)) {
			first = &p
		}
	}
	if first == nil {
		return position{}, false
	}
	return *first, true
}

// backwards is the lament for an item that lies behind what was performed.
func backwards(it Item) error {
	return fmt.Errorf("%s is not performed: it lies behind what was performed, and the exalted servitor skips "+
		"forward, never back", it.Named())
}

// find finds the item named among items.
func find(items []sanction.Verse, it Item) (sanction.Verse, bool) {
	for _, v := range items {
		if it.names(v) {
			return v, true
		}
	}
	return sanction.Verse{}, false
}

// restorable checks that omen is a declared omen of the machine-spirit v,
// to which it may be restored.
func restorable(v sanction.Verse, omen string) error {
	if !v.Spirit() {
		return fmt.Errorf("%s names the omen %q, yet only a machine-spirit is restored to an omen", v.Named(), omen)
	}
	if _, ok := v.Enact[omen]; !ok {
		return fmt.Errorf("%s is never restored to the omen %q: its sanction declares no such omen, and the exalted "+
			"servitor speaks no words but those sanctioned", v.Named(), omen)
	}
	return nil
}
