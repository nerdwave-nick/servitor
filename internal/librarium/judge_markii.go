package librarium

import (
	"fmt"
	"slices"
	"strings"
)

// judgeObservances judges what the subjects name in their introits and
// recessionals: every name must be an observance of the rite, an
// observance is named as an introit or as a recessional but never both, and
// an observance no subject names is never spoken.
func (d *decoder) judgeObservances() {
	r := d.rite
	var introits, recessionals []string
	for _, ref := range d.markII.references {
		if ref.recessional {
			recessionals = append(recessionals, ref.observance)
		} else {
			introits = append(introits, ref.observance)
		}
	}
	for _, ref := range d.markII.references {
		key := "introit"
		if ref.recessional {
			key = "recessional"
		}
		if _, ok := r.Observance(ref.observance); !ok {
			d.heresy(ref.at, "the subject %q names the observance %q in its %q, yet the rite declares no "+
				"observance of that name (%s) — declare it under \"observances\", or strike it", ref.subject,
				ref.observance, key, declares(r.Observances))
			continue
		}
		if ref.recessional && slices.Contains(introits, ref.observance) {
			d.heresy(ref.at, "the observance %q is named as an introit and as a recessional; an observance is "+
				"spoken once in an invocation, before the first subject that names it changes or after the last, "+
				"never both — name it in one place alone, or declare a second observance under another name",
				ref.observance)
		}
	}
	for _, o := range d.markII.observances {
		if _, ok := r.Observance(o.name); ok && !slices.Contains(introits, o.name) && !slices.Contains(recessionals, o.name) {
			d.impurity(o.at, "the observance %q is named by no subject in its \"introit\" or \"recessional\", so it "+
				"is never spoken — name it where it belongs, or strike it", o.name)
		}
	}
}

// findingAt notes a finding at the value of the JSON pointer ptr, or its
// closest written parent.
func (d *decoder) findingAt(sev Severity, ptr, format string, args ...any) {
	d.findings = append(d.findings, Finding{Severity: sev, Scripture: d.src.path, Rite: d.rite.Name,
		Position: d.src.locate(ptr), Message: fmt.Sprintf(format, args...)})
}

// declares tells which observances the rite declares.
func declares(os []Observance) string {
	if len(os) == 0 {
		return "it declares none"
	}
	names := make([]string, len(os))
	for i, o := range os {
		names[i] = o.Name
	}
	return "it declares " + quoteAll(names)
}

// judgeSubjects notes the impurities of the subjects: a machine-spirit that
// cannot be read, and omens alike for every aspect.
func (d *decoder) judgeSubjects() {
	r := d.rite
	for i, s := range r.Subjects {
		ptr := r.subjectPointer(i)
		if ms, ok := s.Matter.(*MachineSpirit); ok && ms.Auspex == nil {
			sharper := ""
			if ms.Exalted {
				sharper = "; being exalted, it calls for the master's proof on every invocation as well — should " +
					"only the machine's own authority read it, give it an auspex marked \"exalted\": true"
			}
			d.findingAt(Impurity, ptr+"/name", "the machine-spirit %q bears no "+
				"\"auspex\": it cannot be read, so it is enacted anew on every invocation and never found already "+
				"in its omen, nor proven after it is set%s", s.Name, sharper)
		}
		if d.alike(s) {
			d.findingAt(Impurity, ptr+"/omens", "the omens of the subject %q are alike "+
				"for every aspect: it bears the same mark whichever aspect the machine stands in, so it never "+
				"moves between them nor reveals which one stands — give each aspect its own omen, or strike the "+
				"subject", s.Name)
		}
	}
}

// alike reports whether the subject declares the same omen for every
// aspect, rendered with the decrees; never for a rite of fewer than two
// aspects, a machine-spirit named by the aspects themselves, or omens that
// cannot be rendered or are drawn from illuminated tomes.
func (d *decoder) alike(s Subject) bool {
	if len(d.aspects) < 2 {
		return false
	}
	var marks []string
	for _, a := range d.aspects {
		mark, ok := d.omenOf(s.Matter, a)
		if !ok {
			return false
		}
		marks = append(marks, mark)
	}
	return !slices.ContainsFunc(marks, func(m string) bool { return m != marks[0] })
}

// omenOf renders the omen of matter for aspect as one comparable word.
func (d *decoder) omenOf(matter Matter, aspect string) (string, bool) {
	render := func(w string) (string, bool) {
		out, err := d.scope.Render(w, d.values(aspect))
		return out, err == nil
	}
	scripture := func(m AspectMap[Scripture]) (string, bool) {
		sc, ok := m.For(aspect)
		switch {
		case !ok || sc.Illuminate:
			return "", false
		case sc.Null:
			return "null", true
		case sc.IsTome():
			p, ok := render(sc.Tome)
			return "tome " + p, ok
		}
		t, ok := render(sc.Text)
		return "text " + t, ok
	}
	switch m := matter.(type) {
	case *Sanctum:
		return scripture(m.Scripture)
	case *Transcription:
		return scripture(m.Scripture)
	case *Tether:
		a, ok := m.Anchor.For(aspect)
		if !ok || a.Null {
			return "null", ok
		}
		p, ok := render(a.Path)
		return "anchor " + p, ok
	case *MachineSpirit:
		if m.Omens.IsZero() {
			return aspect, true
		}
		w, ok := m.Omens.For(aspect)
		if !ok {
			return "", false
		}
		out, ok := render(w)
		return strings.TrimSpace(out), ok
	}
	return "", false
}

// judgeSpeechII notes the impurities of the words a rite of aspects of
// pattern Mark II speaks: its machine-spirits' enact words, and the words
// and reversions of its observances.
func (d *decoder) judgeSpeechII() {
	r := d.rite
	amid := slices.ContainsFunc(r.Subjects, Subject.Exalted) || slices.ContainsFunc(r.Observances, Observance.Exalted)
	judge := func(sp Speech, ptr string, words AspectMap[string]) {
		for _, e := range words.Entries() {
			p := ptr
			if !words.IsUniform() {
				p += "/" + e.Aspect
			}
			for _, msg := range sp.Impurities(e.Value) {
				d.findings = append(d.findings, Finding{Severity: Impurity, Scripture: d.src.path, Rite: r.Name,
					Position: d.src.locate(p), Message: msg})
			}
		}
	}
	for i, s := range r.Subjects {
		ms, ok := s.Matter.(*MachineSpirit)
		if !ok {
			continue
		}
		sp := Speech{Place: fmt.Sprintf("the subject %q", s.Name), Exalted: ms.Exalted, Relics: ms.Relics, AmidExalted: amid}
		judge(sp, r.subjectPointer(i)+"/enact", ms.Enact)
	}
	for i, o := range r.Observances {
		ptr := fmt.Sprintf("/observances/%d", d.markII.observanceAt[i])
		var u Utterance
		switch st := o.Speech.(type) {
		case *Incantation:
			u = st.Utterance
			judge(Speech{Place: fmt.Sprintf("the observance %q", o.Name), Exalted: u.Exalted, Relics: u.Relics,
				AmidExalted: amid}, ptr+"/words", st.Command)
		case *Litany:
			u = st.Utterance
		}
		judge(Speech{Place: fmt.Sprintf("the observance %q", o.Name), Exalted: u.Exalted, Relics: u.Relics,
			AmidExalted: amid}, ptr+"/reversion", u.Reversion)
	}
}
