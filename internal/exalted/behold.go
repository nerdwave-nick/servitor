package exalted

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
	sanctums "github.com/nerdwave-nick/servitor/internal/sanctum"
)

// A subject only the root can read is read by the exalted servitor in its
// invoker's stead. Right after the handshake, before any item, the invoker
// names them all in one read: every machine-spirit whose auspex is
// privileged, and every sanctum, transcription or tether hidden from it.
// The exalted servitor speaks their auspexes at once and matches what each
// printed against the omens its sanction declares; it answers with the
// omens that fit, or none, and of a built-in subject only whether it
// stands as the plan demands: never the words an auspex printed, never the
// scripture of a vessel. So a sanctioned auspex cannot serve as an oracle
// for other words only the root may read. It proves a privileged subject
// it set or restored in the same way.

// readSubjects reads the subjects named, once, before any item of a state
// rite is performed, and tells what each shows.
func (s *session) readSubjects(names []string) error {
	read, err := s.readable(names)
	if err != nil {
		return violation{"read", err}
	}
	s.beheld = true
	ctx, cancel := s.halting()
	defer cancel()
	sights := make([]Sight, len(read))
	var wg sync.WaitGroup
	for i, v := range read {
		wg.Go(func() { sights[i] = s.sight(ctx, v) })
	}
	wg.Wait()
	for _, v := range read {
		s.record(Entry{Subject: v.Subject, Deed: "read", Verdict: Beheld})
	}
	return s.reply(Reply{To: "read", Verdict: Beheld, Sights: sights})
}

// readable finds the subjects named in the plan: a read is heard once,
// before any item is performed, from the invoker of a state rite, and names
// at least one privileged or built-in subject of the plan, each once.
func (s *session) readable(names []string) ([]sanction.Verse, error) {
	switch {
	case !s.sn.State():
		return nil, errors.New("a read is heard only for a rite of aspects, never for an edict")
	case s.beheld:
		return nil, errors.New("the subjects are read once, and they were read already")
	case s.fallen || s.reverting || len(s.performed) > 0 || s.cursor != (position{place: -1, phase: afterSubject}):
		return nil, errors.New("the subjects are read before any item is performed, and one was")
	case len(names) == 0:
		return nil, errors.New("a read names at least one subject")
	}
	var read []sanction.Verse
	for _, name := range names {
		v, ok := find(s.plan.Subjects, Item{Subject: name})
		switch {
		case !ok:
			return nil, unnamed(Item{Subject: name})
		case slices.ContainsFunc(read, func(r sanction.Verse) bool { return r.Subject == name }):
			return nil, fmt.Errorf("a read names %s twice", v.Named())
		case v.Spirit() && v.Auspex == nil:
			return nil, fmt.Errorf("%s is never read by the exalted servitor: its auspex is not exalted, and the "+
				"invoker reads it with its own authority", v.Named())
		}
		read = append(read, v)
	}
	return read, nil
}

// sight is what the subject v shows: the declared omens a machine-spirit
// fits, or whether a built-in subject stands as the plan demands.
func (s *session) sight(ctx context.Context, v sanction.Verse) Sight {
	if v.Spirit() {
		return Sight{Subject: v.Subject, Omens: s.behold(ctx, v, false)}
	}
	return Sight{Subject: v.Subject, Unchanged: s.settled(v)}
}

// behold speaks the privileged auspex of the machine-spirit v and returns
// the declared omens it fits, sorted; none when it is mute. Its scan reads,
// when it has one: it prints one omen, or, of a spirit that declares no
// omens, the names of aspects. Its trials prove what was set, when it has
// them (proving), and read when it has no scan: every trial is spoken at
// once, and the omens whose trial ends with the death-mark 0 fit — the
// omens bearing no trial when none does. Words that fall or are silent,
// print nothing or an omen never declared, or one silent trial make it
// mute.
func (s *session) behold(ctx context.Context, v sanction.Verse, proving bool) []string {
	a, declared := v.Auspex, slices.Sorted(maps.Keys(v.Enact))
	if a.Scan != "" && (!proving || len(a.Trials) == 0) {
		printed, spoke, err := s.speaker.Hear(ctx, v, a.Scan, "")
		read := strings.TrimSpace(printed)
		if err != nil || !spoke || read == "" {
			return nil
		}
		shown := []string{read}
		if a.NamesAspects {
			shown = strings.Fields(read)
		}
		for _, omen := range shown {
			if !slices.Contains(declared, omen) {
				return nil
			}
		}
		slices.Sort(shown)
		return slices.Compact(shown)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var (
		mu       sync.Mutex
		shown    []string
		silenced bool
		wg       sync.WaitGroup
	)
	for omen, words := range a.Trials {
		wg.Go(func() {
			_, spoke, err := s.speaker.Hear(ctx, v, words, omen)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err != nil:
				silenced = true
				cancel()
			case spoke:
				shown = append(shown, omen)
			}
		})
	}
	wg.Wait()
	switch {
	case silenced:
		return nil
	case len(shown) == 0:
		shown = slices.DeleteFunc(declared, func(omen string) bool { _, tried := a.Trials[omen]; return tried })
	}
	slices.Sort(shown)
	return shown
}

// prove reads the privileged machine-spirit v anew once it was set to
// omen: it must show it. It returns the omens it fits; nil, with no
// error, for a spirit whose auspex is common, which its invoker proves.
func (s *session) prove(ctx context.Context, v sanction.Verse, omen string) ([]string, error) {
	if v.Auspex == nil {
		return nil, nil
	}
	shown := s.behold(ctx, v, true)
	if !slices.Contains(shown, omen) {
		return nil, fmt.Errorf("%s shows %s; %q was commanded", v.Named(), omensNamed(shown), omen)
	}
	return shown, nil
}

// omensNamed names the omens shown: "no omen", the omen "x", or the omens
// "x", "y".
func omensNamed(shown []string) string {
	quoted := make([]string, len(shown))
	for i, omen := range shown {
		quoted[i] = strconv.Quote(omen)
	}
	switch len(shown) {
	case 0:
		return "no omen"
	case 1:
		return "the omen " + quoted[0]
	}
	return "the omens " + strings.Join(quoted, ", ")
}

// settled reports whether the sanctum, transcription or tether v already
// stands as the plan demands, so that performing it would change nothing:
// what cannot be read never does.
func (s *session) settled(v sanction.Verse) bool {
	switch v.Kind {
	case librarium.KindSanctum:
		before, absent, err := s.read(v.Target)
		if err != nil || absent {
			return false
		}
		m := sanctums.Marker{Glyph: v.Glyph, ClosingGlyph: v.ClosingGlyph, Ward: v.Ward}
		after, err := sanctums.Upsert(string(before), m, s.plan.Aspect, string(v.Scripture))
		return err == nil && after == string(before)
	case librarium.KindTranscription:
		if v.Null {
			_, absent, err := s.read(v.Target)
			return err == nil && absent
		}
		stands, err := s.cfg.Ground.Stands(v.Target, v.Scripture, v.Seal)
		return err == nil && stands
	case librarium.KindTether:
		anchor, bond, absent, err := s.cfg.Ground.Bonded(v.Target)
		return err == nil && (v.Null && absent || !v.Null && bond && anchor == v.Anchor)
	}
	return false
}
