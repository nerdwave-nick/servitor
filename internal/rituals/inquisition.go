package rituals

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// Inquire examines the rites named (every rite when none is named), the
// settings and — unless spareVessels — the machine their liturgies act
// upon, and returns every heresy and impurity found, sorted. The error is
// *Unrecorded for a name the Librarium does not hold.
//
// Beyond the findings of the Librarium itself, the tomes of every rite are
// opened (a missing tome, or a placeholder an illuminated tome cannot
// fill, is heresy). The machine yields impurities — a tongue spoken
// nowhere, a vessel missing that its sanctum may not consecrate, an anchor
// missing for some aspect, a tainted sanctum, a desecrated rite — and
// heresy for a vessel whose sanctum markers are broken.
func (s *Servitor) Inquire(names []string, spareVessels bool) (librarium.Findings, error) {
	lib := s.Librarium
	for _, n := range names {
		if _, ok := lib.Scriptures[n]; !ok {
			return nil, &Unrecorded{Name: n, Librarium: lib.Dir}
		}
	}
	wanted := func(name string) bool { return len(names) == 0 || slices.Contains(names, name) }
	found := librarium.Findings{}
	for _, f := range lib.Findings {
		if f.Rite == "" || wanted(f.Rite) {
			found = append(found, f)
		}
	}
	orders := s.Orders()
	for _, name := range lib.Names() {
		r := lib.Rites[name]
		if r == nil || !wanted(name) {
			continue
		}
		e := &examiner{rite: r, orders: orders, seen: map[string]bool{}}
		e.tomes()
		if !spareVessels {
			e.machine()
			e.omens(s)
		}
		found = append(found, e.found...)
	}
	found.Sort()
	return found, nil
}

// examiner gathers the findings of one rite that is free of heresy, whose
// steps therefore stand at their written places.
type examiner struct {
	rite   *librarium.Rite
	orders librarium.Orders
	found  librarium.Findings
	seen   map[string]bool
}

// note records a finding at the JSON pointer ptr of the rite's scripture.
func (e *examiner) note(sev librarium.Severity, ptr, format string, args ...any) {
	e.found = append(e.found, librarium.Finding{Severity: sev, Scripture: e.rite.Path, Rite: e.rite.Name,
		Position: e.rite.Locate(ptr), Message: fmt.Sprintf(format, args...)})
}

// once reports whether key is examined for the first time.
func (e *examiner) once(key string) bool {
	if e.seen[key] {
		return false
	}
	e.seen[key] = true
	return true
}

// path renders text for aspect, with the decrees of that aspect, and
// resolves it against the rite's directory; ok is false when it cannot be
// rendered (the Librarium denounced that already).
func (e *examiner) path(aspect, text string) (string, bool) {
	inscriptions := map[string]string{}
	for _, in := range e.rite.Inscriptions {
		if v, ok := in.Decrees.For(aspect); ok && v != "" {
			inscriptions[in.Key] = v
		}
	}
	out, err := e.rite.Scope().Render(text, placeholder.Values{Aspect: aspect, Rite: e.rite.Name, Inscriptions: inscriptions})
	if err != nil {
		return "", false
	}
	return e.rite.ResolvePath(out), true
}

// field is the JSON pointer of field within the step at index i.
func field(i int, f string) string { return fmt.Sprintf("/liturgy/%d/%s", i, f) }

// mapField is the JSON pointer of the entry of m that answers for aspect.
func mapField[T any](i int, f string, m librarium.AspectMap[T], aspect string) string {
	if key, ok := m.Key(aspect); ok && !m.IsUniform() {
		return field(i, f+"/"+key)
	}
	return field(i, f)
}

// tomes opens every tome of the rite's sanctums and transcriptions.
func (e *examiner) tomes() {
	for i, step := range e.rite.Liturgy {
		var m librarium.AspectMap[librarium.Scripture]
		switch st := step.(type) {
		case *librarium.Sanctum:
			m = st.Scripture
		case *librarium.Transcription:
			m = st.Scripture
		default:
			continue
		}
		for _, aspect := range e.rite.Aspects {
			sc, ok := m.For(aspect)
			if !ok || !sc.IsTome() {
				continue
			}
			p, ok := e.path(aspect, sc.Tome)
			if ok && e.once("tome "+p) {
				e.tome(mapField(i, "scripture", m, aspect)+"/tome", p, sc.Illuminate)
			}
		}
	}
}

// tome opens the tome at p, written at ptr, and examines its placeholders
// when it is illuminated.
func (e *examiner) tome(ptr, p string, illuminate bool) {
	data, err := os.ReadFile(p)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		e.note(librarium.Heresy, ptr, "the tome %s is nowhere to be found: no scripture can be drawn from it, and "+
			"every invocation that needs it would be refused — write the tome, or name one that stands", p)
		return
	case errors.Is(err, fs.ErrPermission):
		e.note(librarium.Heresy, ptr, "the machine spirit denies the servitor access to the tome %s: no scripture "+
			"can be drawn from it, and every invocation that needs it would be refused", p)
		return
	case err != nil:
		e.note(librarium.Heresy, ptr, "the tome %s resists every attempt to read it: no scripture can be drawn "+
			"from it, and every invocation that needs it would be refused", p)
		return
	}
	if !illuminate {
		return
	}
	for _, h := range e.rite.Scope().Check(string(data)) {
		e.found = append(e.found, librarium.Finding{Severity: librarium.Heresy, Scripture: p, Rite: e.rite.Name,
			Position: librarium.Position{Line: h.Pos.Line, Column: h.Pos.Column},
			Message:  "within the illuminated tome: " + h.Message()})
	}
}
