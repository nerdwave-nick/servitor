package librarium

import (
	"fmt"

	"github.com/tailscale/hujson"
)

// decodeSubjects reads the subjects of a rite of aspects: a list that holds
// at least one.
func (d *decoder) decodeSubjects(v *hujson.Value) {
	arr, ok := v.Value.(*hujson.Array)
	if !ok {
		d.wrongForm(v, `the "subjects" of a rite`, "a list of subjects")
		return
	}
	if len(arr.Elements) == 0 {
		d.heresy(v, "the rite declares no subjects; a rite of aspects reads and sets its subjects, and a rite with "+
			"nothing to read or set can bring the machine into no aspect — declare at least one, or mark it an "+
			"edict with \"edict\": true should it only ever be proclaimed whole")
	}
	for i := range arr.Elements {
		if s := d.decodeSubject(&arr.Elements[i], i); s != nil {
			d.rite.Subjects = append(d.rite.Subjects, *s)
			d.rite.subjectAt = append(d.rite.subjectAt, i)
		}
	}
}

// decodeSubject reads the subject at index i; nil when its designation
// cannot be told.
func (d *decoder) decodeSubject(v *hujson.Value, i int) *Subject {
	what := subjectAt(i)
	all, ok := d.object(v, what, nil)
	if !ok {
		return nil
	}
	s := &Subject{}
	if s.Name = d.name(v, all, what, "subject", &d.markII.subjects); s.Name != "" {
		what = fmt.Sprintf("the subject %q", s.Name)
	}
	designation := d.designation(v, all, what, SubjectDesignations)
	if designation == "" {
		return nil
	}
	fs := d.designated(all, what, designation, SubjectKeys, SubjectDesignations)
	sd := stepDecoder{d: d, v: v, fs: fs, what: what, name: s.Name, designation: designation}
	sd.exalted = sd.exaltation()
	if m, ok := fs.get("purpose"); ok {
		s.Purpose, _ = d.str(m.val, sd.field("purpose"))
	}
	s.Introit = sd.references("introit", false)
	s.Recessional = sd.references("recessional", true)
	if sd.exalted {
		// The words that set an exalted subject are fixed at its sanction.
		defer d.within(d.rite.ExaltedScope())()
	}
	switch designation {
	case DesignationSanctum:
		s.Matter = sd.sanctumSubject()
	case DesignationTranscription:
		s.Matter = sd.transcriptionSubject()
	case DesignationTether:
		s.Matter = sd.tetherSubject()
	default:
		s.Matter = sd.machineSpirit()
	}
	return s
}

// references reads the observances the subject names in its introit or
// recessional, kept to be judged once every observance is read.
func (s stepDecoder) references(key string, recessional bool) []string {
	m, ok := s.fs.get(key)
	if !ok {
		return nil
	}
	arr, ok := m.val.Value.(*hujson.Array)
	if !ok {
		s.d.wrongForm(m.val, s.field(key), "a list of the names of observances")
		return nil
	}
	names := []string{}
	for i := range arr.Elements {
		e := &arr.Elements[i]
		if n, ok := s.d.str(e, fmt.Sprintf("each observance named in %s", s.field(key))); ok {
			names = append(names, n)
			s.d.markII.references = append(s.d.markII.references,
				reference{subject: s.name, observance: n, recessional: recessional, at: e})
		}
	}
	return names
}

// omens reads the omens of a subject of a built-in designation, which it
// must declare for every aspect.
func omens[T any](s stepDecoder, why string, one func(*hujson.Value, string) (T, bool)) AspectMap[T] {
	m, ok := s.required("omens", why)
	if !ok {
		return AspectMap[T]{}
	}
	return keyedMap(s.d, m.val, s.field("omens"), s.d.aspectKeying(), true, one)
}

// vessel reads the vessel of a sanctum or transcription.
func (s stepDecoder) vessel(why string) string {
	m, ok := s.required("vessel", why)
	if !ok {
		return ""
	}
	p, _ := s.d.templated(m.val, s.field("vessel"), false)
	if s.exalted {
		s.rootedPath(p, "vessel", m.val)
	}
	return p
}

func (s stepDecoder) sanctumSubject() *Sanctum {
	st := &Sanctum{Exalted: s.exalted}
	st.Vessel = s.vessel("a sanctum must name the vessel that keeps its warded region")
	s.sanctumMarks(st)
	st.Scripture = omens(s, "a sanctum must declare the scripture it keeps for each aspect",
		func(v *hujson.Value, what string) (Scripture, bool) { return s.d.oneScripture(v, what, false) })
	return st
}

func (s stepDecoder) transcriptionSubject() *Transcription {
	st := &Transcription{Zeal: s.zeal(), Exalted: s.exalted}
	st.Vessel = s.vessel("a transcription must name the vessel it writes whole")
	st.Scripture = omens(s, "a transcription must declare the scripture of its vessel for each aspect, or null "+
		"where the vessel shall not be", func(v *hujson.Value, what string) (Scripture, bool) {
		return s.d.oneScripture(v, what, true)
	})
	if m, ok := s.fs.get("seal"); ok {
		st.Seal = s.seal(m.val)
	}
	return st
}

func (s stepDecoder) tetherSubject() *Tether {
	st := &Tether{Zeal: s.zeal(), Exalted: s.exalted}
	if m, ok := s.required("tether", "a tether must name the name it binds"); ok {
		st.Name, _ = s.d.templated(m.val, s.field("tether"), false)
		if s.exalted {
			s.rootedPath(st.Name, "bound name", m.val)
		}
	}
	st.Anchor = omens(s, "a tether must declare where it leads for each aspect, or null where it shall be "+
		"unbound", s.d.anchor)
	return st
}

// machineSpirit reads a subject read by its auspex and set by its enact.
func (s stepDecoder) machineSpirit() *MachineSpirit {
	d := s.d
	ms := &MachineSpirit{Exalted: s.exalted, Relics: s.relics()}
	declared := d.declaredOmens(AspectMap[string]{})
	if m, ok := s.fs.get("omens"); ok {
		ms.Omens = keyedMap(d, m.val, s.field("omens"), d.aspectKeying(), true, s.text(false))
		declared = nil
		if !ms.Omens.IsZero() {
			declared = d.declaredOmens(ms.Omens)
		}
	}
	k := omenKeying(s.what, declared)
	if m, ok := s.fs.get("auspex"); ok {
		ms.Auspex = s.spiritAuspex(m.val, k)
	}
	if m, ok := s.required("enact", "a machine-spirit that cannot be set belongs in no rite — write under "+
		"\"enact\" the words that set it into each omen"); ok {
		sc := d.scope
		sc.Omen = true
		restore := d.within(sc)
		ms.Enact = keyedMap(d, m.val, s.field("enact"), k, true, s.text(false))
		restore()
	}
	if m, ok := s.fs.get("tongue"); ok {
		ms.Tongue, _ = d.word(m.val, s.field("tongue"))
	}
	if m, ok := s.fs.get("patience"); ok {
		ms.Patience = d.patience(m.val, s.field("patience"))
	}
	return ms
}

// spiritAuspex reads the auspex of a machine-spirit: its scan, its trials
// keyed by omen, its patience, and whether it is privileged.
func (s stepDecoder) spiritAuspex(v *hujson.Value, k keying) *SpiritAuspex {
	d := s.d
	what := s.field("auspex")
	fs, ok := d.object(v, what, SpiritAuspexKeys)
	if !ok {
		return nil
	}
	a := &SpiritAuspex{}
	if m, ok := fs.get("exalted"); ok {
		a.Exalted = d.boolean(m.val, `the "exalted" of `+what)
		if a.Exalted && !s.exalted {
			d.heresy(m.key, "%s is exalted, yet %s is not: an auspex is spoken with the machine's own authority "+
				"only to read a subject that is set with it — write \"exalted\": true upon the subject, or strike "+
				"it from the auspex", what, s.what)
		}
	}
	// An auspex is spoken with the invoker's authority unless it is itself
	// exalted; then its words are fixed at the subject's sanction.
	sc := d.rite.Scope()
	if a.Exalted {
		sc = d.rite.ExaltedScope()
	}
	defer d.within(sc)()
	scan, hasScan := fs.get("scan")
	if hasScan {
		a.Scan, _ = d.templated(scan.val, `the "scan" of `+what, false)
	}
	trials, hasTrials := fs.get("trials")
	if hasTrials {
		omen := sc
		omen.Omen = true
		restore := d.within(omen)
		a.Trials = keyedMap(d, trials.val, `the "trials" of `+what, k, false, s.text(false))
		restore()
	}
	if !hasScan && !hasTrials {
		d.heresy(v, "%s holds neither \"scan\" nor \"trials\"; an auspex reads its subject by the words of its "+
			"scan, which print the omen it shows, or by its trials, one for each omen it may show — write at "+
			"least one", what)
	}
	if m, ok := fs.get("patience"); ok {
		a.Patience = d.patience(m.val, `the "patience" of `+what)
	}
	return a
}
