package tui

import (
	"slices"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// kindMachineSpirit is the kind of the step draft that writes a
// machine-spirit, which no liturgy bears: its tongue, patience, exaltation
// and relics stand in the step's fixed values and relics, as for speech.
const kindMachineSpirit librarium.Kind = 0

// subjectDraft is the editable form of one subject of a rite of pattern
// Mark II. A sanctum, transcription or tether is written as the step of
// its kind: its own keys in the step's fixed values, its omens as the
// step's scripture or anchor, varied per aspect. A machine-spirit keeps
// its omens, auspex and enact words in spirit.
type subjectDraft struct {
	name, purpose        string
	introit, recessional []string // the observances it names; nil when unwritten
	step                 stepDraft
	spirit               spiritDraft
}

// subjectDesignations are the designations of the subjects the wizard
// writes, as the kinds of step that write them, in the order of the codex.
var subjectDesignations = []librarium.Kind{librarium.KindSanctum, librarium.KindTranscription, librarium.KindTether,
	kindMachineSpirit}

// kindKey names the kind k as the scripture does: its step key, or the
// designation machine-spirit.
func kindKey(k librarium.Kind) string {
	if k == kindMachineSpirit {
		return librarium.DesignationMachineSpirit
	}
	return k.Key()
}

// subjectFrom is the subject s as the wizard edits it.
func subjectFrom(s librarium.Subject) subjectDraft {
	sd := subjectDraft{name: s.Name, purpose: s.Purpose, introit: s.Introit, recessional: s.Recessional}
	switch m := s.Matter.(type) {
	case *librarium.MachineSpirit:
		sd.step, sd.spirit = spiritFrom(m)
	case librarium.Step:
		sd.step = stepFrom(m)
	}
	return sd
}

// newSubjectDraft is a new subject of designation k.
func newSubjectDraft(k librarium.Kind) subjectDraft {
	return subjectDraft{step: newStepDraft(k), spirit: newSpiritDraft()}
}

// clone is the subject with maps and lists of its own, so that amending it
// leaves the draft it came from as it was.
func (sd subjectDraft) clone() subjectDraft {
	sd.introit, sd.recessional = slices.Clone(sd.introit), slices.Clone(sd.recessional)
	sd.spirit.omens.own = copyMap(sd.spirit.omens.own)
	sd.spirit.trials.own = copyMap(sd.spirit.trials.own)
	sd.spirit.enact.own = copyMap(sd.spirit.enact.own)
	return sd
}

// toSubject writes the subject for the rite of aspects the draft d holds,
// its maps as objects, as pattern Mark II writes every map.
func (sd subjectDraft) toSubject(d riteDraft) librarium.Subject {
	aspects := d.aspectList()
	s := librarium.Subject{Name: sd.name, Purpose: sd.purpose, Introit: sd.introit, Recessional: sd.recessional}
	if sd.step.kind == kindMachineSpirit {
		s.Matter = sd.spirit.toSpirit(sd.step, aspects, d.declaredOmens(sd.spirit))
		return s
	}
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

// subjectWords is what the list of subjects tells of sd: the vessel or
// bound name of a built-in subject, or the scan of a machine-spirit, else
// its first enact words.
func subjectWords(sd subjectDraft, aspects []string) string {
	if sd.step.kind != kindMachineSpirit {
		return stepWords(sd.step.toStep(aspects), "")
	}
	if sd.spirit.scan != "" {
		return sd.spirit.scan
	}
	if e := sd.spirit.enact.toMap(nil).Entries(); len(e) > 0 {
		return e[0].Value
	}
	return ""
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
