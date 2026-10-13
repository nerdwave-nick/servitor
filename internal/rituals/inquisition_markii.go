package rituals

import (
	"fmt"
	"slices"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// The subjects and observances of a state rite of pattern Mark II are
// examined as the verses of Mark I are: their tomes, the relics, scrolls
// and tongues of their exalted speech, and the machine they act upon. A
// rite free of heresy holds every subject and observance at its written
// place, so each is found by its index.

// subjectAt is the JSON pointer of the field f of the subject at index i.
func subjectAt(i int, f string) string { return fmt.Sprintf("/subjects/%d/%s", i, f) }

// observanceAt is the JSON pointer of the field f of the observance at
// index i.
func observanceAt(i int, f string) string { return fmt.Sprintf("/observances/%d/%s", i, f) }

// subjectTomes opens every tome a sanctum or transcription subject draws an
// omen from.
func (e *examiner) subjectTomes() {
	for i, s := range e.rite.Subjects {
		var m librarium.AspectMap[librarium.Scripture]
		switch matter := s.Matter.(type) {
		case *librarium.Sanctum:
			m = matter.Scripture
		case *librarium.Transcription:
			m = matter.Scripture
		default:
			continue
		}
		for _, aspect := range e.aspects() {
			sc, ok := m.For(aspect)
			if !ok || !sc.IsTome() {
				continue
			}
			p, ok := e.path(aspect, sc.Tome)
			if ok && e.once("tome "+p) {
				e.tome(mapPointer(subjectAt(i, "omens"), m, aspect)+"/tome", p, sc.Illuminate,
					e.rite.ScopeOf(s.Matter.(librarium.Step)))
			}
		}
	}
}

// subjectsExaltation examines the relics and tongues of the exalted
// machine-spirits and observances, and the words the scrolls of the
// observances speak.
func (e *examiner) subjectsExaltation() {
	r := e.rite
	amid := slices.ContainsFunc(r.Subjects, librarium.Subject.Exalted) ||
		slices.ContainsFunc(r.Observances, librarium.Observance.Exalted)
	for i, s := range r.Subjects {
		ms, ok := s.Matter.(*librarium.MachineSpirit)
		if !ok {
			continue
		}
		place := fmt.Sprintf("the subject %q", s.Name)
		for j, relic := range ms.Relics {
			e.relic(subjectAt(i, fmt.Sprintf("relics/%d", j)), fmt.Sprintf("relic %d of %s, %s,", j+1, place, relic), relic)
		}
		if ms.Exalted {
			e.exaltedTongue(fmt.Sprintf("/subjects/%d", i), ms.Tongue, place, "subject")
		}
	}
	for i, o := range r.Observances {
		place := fmt.Sprintf("the observance %q", o.Name)
		var u librarium.Utterance
		switch st := o.Speech.(type) {
		case *librarium.Incantation:
			u = st.Utterance
		case *librarium.Litany:
			u = st.Utterance
			sp := librarium.Speech{Place: place, Exalted: u.Exalted, Relics: u.Relics, AmidExalted: amid}
			for _, aspect := range e.aspects() {
				text, ok := st.Scroll.For(aspect)
				if !ok {
					continue
				}
				if p, ok := e.path(aspect, text); ok && e.once("scroll "+p) {
					e.speech(p, sp)
				}
			}
		}
		for j, relic := range u.Relics {
			e.relic(observanceAt(i, fmt.Sprintf("relics/%d", j)), fmt.Sprintf("relic %d of %s, %s,", j+1, place, relic), relic)
		}
		if u.Exalted {
			e.exaltedTongue(fmt.Sprintf("/observances/%d", i), u.Tongue, place, "observance")
		}
	}
}

// subjectsMachine examines what the subjects and observances act upon:
// the vessels of sanctum subjects, the anchors of tether subjects, and the
// tongues of machine-spirits, of their auspexes and of observances.
func (e *examiner) subjectsMachine() {
	for i, s := range e.rite.Subjects {
		switch matter := s.Matter.(type) {
		case *librarium.Sanctum:
			e.vessel(subjectAt(i, "vessel"), fmt.Sprintf("the sanctum subject %q", s.Name), matter)
		case *librarium.Tether:
			e.anchors(subjectAt(i, "omens"), s.Name, matter)
		case *librarium.MachineSpirit:
			e.spiritTongues(i, s.Name, matter)
		}
	}
	for i, o := range e.rite.Observances {
		place := fmt.Sprintf("the observance %q", o.Name)
		switch st := o.Speech.(type) {
		case *librarium.Incantation:
			e.speaks(observanceAt(i, "tongue"), st.Tongue, place, "observance", "uttered")
		case *librarium.Litany:
			if e.litanyNeedsTongue(st) {
				e.speaks(observanceAt(i, "tongue"), st.Tongue, place, "observance", "uttered")
			}
		}
	}
}

// spiritTongues denounces the tongue of the machine-spirit name at index i
// when this machine does not speak it: for its enact words, and — noted
// apart, at the auspex — for its auspex, which speaks the same tongue.
func (e *examiner) spiritTongues(i int, name string, ms *librarium.MachineSpirit) {
	place := fmt.Sprintf("the subject %q", name)
	e.speaks(subjectAt(i, "tongue"), ms.Tongue, place, "subject", "set")
	if ms.Auspex == nil {
		return
	}
	if t := e.tongueOf(ms.Tongue); !speakable(t) {
		e.note(librarium.Impurity, subjectAt(i, "auspex"), "the tongue %q of the auspex of %s is spoken "+
			"nowhere on this machine: no program of that name lies along its PATH, so the auspex could never be "+
			"heard and the subject loses its omen — name another \"tongue\" for the subject, the rite or the "+
			"settings", t, place)
	}
}
