package tui

import (
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// subjectDraft is the editable form of one subject of a rite of pattern
// Mark II. A sanctum, transcription or tether is written as the step of
// its kind: its own keys in the step's fixed values, its omens as the
// step's scripture or anchor, varied per aspect.
type subjectDraft struct {
	name, purpose string
	// introit and recessional are kept as read, until the wizard knows
	// the observances they name.
	introit, recessional []string
	step                 stepDraft
}

// subjectDesignations are the designations of the subjects the wizard
// writes, as the kinds of step that write them, in the order of the codex.
var subjectDesignations = []librarium.Kind{librarium.KindSanctum, librarium.KindTranscription, librarium.KindTether}

// wizardWrites reports whether the consecration wizard writes every form
// the rite holds: of pattern Mark II, it knows edicts and the rites whose
// subjects are sanctums, transcriptions and tethers, without observances.
func wizardWrites(r *librarium.Rite) bool {
	if !r.MarkII || r.Edict {
		return true
	}
	if len(r.Observances) > 0 {
		return false
	}
	for _, s := range r.Subjects {
		switch s.Matter.(type) {
		case *librarium.Sanctum, *librarium.Transcription, *librarium.Tether:
		default:
			return false
		}
	}
	return true
}

// subjectFrom is the subject s as the wizard edits it.
func subjectFrom(s librarium.Subject) subjectDraft {
	sd := subjectDraft{name: s.Name, purpose: s.Purpose, introit: s.Introit, recessional: s.Recessional}
	if step, ok := s.Matter.(librarium.Step); ok {
		sd.step = stepFrom(step)
	}
	return sd
}

// newSubjectDraft is a new subject of designation k.
func newSubjectDraft(k librarium.Kind) subjectDraft { return subjectDraft{step: newStepDraft(k)} }

// toSubject writes the subject for a rite of aspects, its omens as an
// object, as pattern Mark II writes every map.
func (sd subjectDraft) toSubject(aspects []string) librarium.Subject {
	s := librarium.Subject{Name: sd.name, Purpose: sd.purpose, Introit: sd.introit, Recessional: sd.recessional}
	switch st := sd.step.toStep(aspects).(type) {
	case *librarium.Sanctum:
		st.Scripture = omensObject(st.Scripture)
		s.Matter = st
	case *librarium.Transcription:
		st.Scripture = omensObject(st.Scripture)
		s.Matter = st
	case *librarium.Tether:
		st.Anchor = omensObject(st.Anchor)
		s.Matter = st
	}
	return s
}

// object is the aspect map m written as an object, even when it holds one
// value for every aspect; unwritten, it stays so.
func object[T any](m librarium.AspectMap[T]) librarium.AspectMap[T] {
	if m.IsZero() {
		return m
	}
	return librarium.PerAspect(m.Entries()...)
}

// omensObject is the omens m as an object, written even when empty.
func omensObject[T any](m librarium.AspectMap[T]) librarium.AspectMap[T] {
	return librarium.PerAspect(m.Entries()...)
}
