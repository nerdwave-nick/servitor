package rituals

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanctum"
)

// machine examines what the rite's steps act upon: vessels, anchors and
// tongues.
func (e *examiner) machine() {
	for i, step := range e.rite.Liturgy {
		switch st := step.(type) {
		case *librarium.Sanctum:
			e.vessel(field(i, "sanctum"), "the sanctum", st)
		case *librarium.Tether:
			e.anchors(field(i, "anchor"), "", st)
		case *librarium.Incantation:
			e.tongue(i, st.Tongue)
		case *librarium.Litany:
			if e.rite.Edict {
				e.scroll(i, st)
			}
			if e.litanyNeedsTongue(st) {
				e.tongue(i, st.Tongue)
			}
		}
	}
	if a := e.rite.Auspex; a != nil {
		if t := e.tongueOf(""); !speakable(t) {
			e.note(librarium.Impurity, "/auspex", "the tongue %q of the auspex is spoken nowhere on this "+
				"machine: no program of that name lies along its PATH, so the auspex could never be heard and "+
				"the rite loses its omen — name another \"tongue\" for the rite or the settings", t)
		}
	}
	e.subjectsMachine()
}

// vessel examines for every aspect the vessel of the sanctum s, written
// at ptr; who names it.
func (e *examiner) vessel(ptr, who string, s *librarium.Sanctum) {
	glyph, closing := s.Glyphs()
	m := sanctum.Marker{Glyph: glyph, ClosingGlyph: closing, Ward: s.WardFor(e.rite.Name)}
	for _, aspect := range e.aspects() {
		p, ok := e.path(aspect, s.Vessel)
		if !ok || !e.once(ptr+" "+p) {
			continue
		}
		data, err := os.ReadFile(p)
		switch {
		case errors.Is(err, fs.ErrNotExist) && !s.Consecrate:
			e.note(librarium.Impurity, ptr, "no vessel stands at %s, and %s may not consecrate one: every "+
				"invocation will be refused until the vessel is raised, or until %s is granted \"consecrate\": "+
				"true", p, who, who)
		case err != nil:
		default:
			if _, _, err := sanctum.Find(string(data), m); err != nil {
				e.note(librarium.Heresy, ptr, "the vessel %s is defiled at %v — mend its "+
					"markers by hand, for no invocation can rewrite a sanctum whose bounds are broken", p, err)
			}
		}
	}
}

// anchors examines for every aspect the anchor of the tether t, whose
// anchors are written at base; subject names a tether subject, "" a
// tether verse.
func (e *examiner) anchors(base, subject string, t *librarium.Tether) {
	who := "the tether"
	if subject != "" {
		who = fmt.Sprintf("the tether subject %q", subject)
	}
	for _, aspect := range e.aspects() {
		anchor, ok := t.Anchor.For(aspect)
		if !ok || anchor.Null {
			continue
		}
		p, okAnchor := e.path(aspect, anchor.Path)
		name, okName := e.path(aspect, t.Name)
		if !okAnchor || !okName || !e.once(base+" "+p) {
			continue
		}
		ptr := mapPointer(base, t.Anchor, aspect)
		if _, err := os.Stat(p); errors.Is(err, fs.ErrNotExist) {
			e.note(librarium.Impurity, ptr, "the anchor %s, to which %s binds %s in the aspect %q, does not "+
				"exist: invoked into that aspect, the tether would lead into the void", p, who, name, aspect)
		}
		if hall := openHall(p); t.Exalted && hall != "" {
			exalted := "the exalted tether " + name
			if subject != "" {
				exalted = fmt.Sprintf("the exalted tether subject %q, binding %s,", subject, name)
			}
			e.note(librarium.Heresy, ptr, "%s would lead in the aspect %q to %s, beneath the hall %s its "+
				"invoker may write: whatever stands there would be read with the machine's own authority — "+
				"transcribe the scripture by an exalted transcription instead", exalted, aspect, p, hall)
		}
	}
}

// openHall is the first hall above the place path truly leads to (its
// bonds followed as far as they stand) that the invoker may write; "" when
// none is.
func openHall(path string) string {
	real := filepath.Clean(path)
	for p := real; ; p = filepath.Dir(p) {
		if r, err := filepath.EvalSymlinks(p); err == nil {
			real = filepath.Join(r, strings.TrimPrefix(real, p))
			break
		}
		if p == filepath.Dir(p) {
			break
		}
	}
	for hall := filepath.Dir(real); ; hall = filepath.Dir(hall) {
		if unix.Access(hall, unix.W_OK) == nil {
			return hall
		}
		if hall == filepath.Dir(hall) {
			return ""
		}
	}
}

// tongueOf is the tongue a step naming tongue (or "") is spoken in:
// step > rite > settings.
func (e *examiner) tongueOf(tongue string) string {
	for _, t := range []string{tongue, e.rite.Tongue, e.orders.Tongue} {
		if t != "" {
			return t
		}
	}
	return librarium.DefaultTongue
}

// tongue denounces the tongue of the step at index i when this machine
// does not speak it.
func (e *examiner) tongue(i int, tongue string) {
	e.speaks(field(i, "tongue"), tongue, fmt.Sprintf("verse %d", i+1), "step", "uttered")
}

// speaks denounces the tongue (or "") a speaker names at ptr when this
// machine does not speak it. place names the speaker, holder tells what it
// is (a step, a subject, an observance) and deed what it could then never
// be (uttered, set).
func (e *examiner) speaks(ptr, tongue, place, holder, deed string) {
	if tongue == "" && e.rite.Tongue != "" {
		ptr = "/tongue" // the rite's own tongue speaks for the speaker
	}
	if t := e.tongueOf(tongue); !speakable(t) {
		e.note(librarium.Impurity, ptr, "the tongue %q of %s is spoken nowhere on this machine: no program of "+
			"that name lies along its PATH, so the %s could never be %s — name another \"tongue\" for the %s, "+
			"the rite or the settings", t, place, holder, deed, holder)
	}
}

// litanyNeedsTongue reports whether a litany speaks through its tongue: for
// its reversion, or for a scroll that stands and may not be executed.
func (e *examiner) litanyNeedsTongue(l *librarium.Litany) bool {
	if !l.Reversion.IsZero() {
		return true
	}
	for _, aspect := range e.aspects() {
		text, ok := l.Scroll.For(aspect)
		if !ok {
			continue
		}
		if p, ok := e.path(aspect, text); ok {
			if info, err := os.Stat(p); err == nil && info.Mode()&0o111 == 0 {
				return true
			}
		}
	}
	return false
}

// scroll denounces the scroll of an edict's litany that does not stand: no
// step of an edict writes one, so every proclaiming would be refused.
func (e *examiner) scroll(i int, l *librarium.Litany) {
	text, ok := l.Scroll.For("")
	if !ok {
		return
	}
	ptr := field(i, "litany")
	if e.rite.MarkII {
		ptr = field(i, "scroll") // an entry of the designation form names its scroll so
	}
	p, ok := e.path("", text)
	if !ok || !e.once(ptr+" "+p) {
		return
	}
	if _, err := os.Stat(p); errors.Is(err, fs.ErrNotExist) {
		e.note(librarium.Impurity, ptr, "no scroll stands at %s for the litany of verse %d to "+
			"recite, and no step of an edict writes one: every proclaiming of the edict will be refused until "+
			"the scroll is written, or the litany names one that stands", p, i+1)
	}
}

func speakable(tongue string) bool {
	_, err := exec.LookPath(tongue)
	return err == nil
}

// omens reads the rite's augury, auspex and all, and denounces its taint
// and its desecration.
func (e *examiner) omens(s *Servitor) {
	a, lament := augury.Augur(e.rite, augury.Options{Orders: e.orders, Env: s.env()})
	if lament != nil {
		e.found = append(e.found, librarium.Finding{Severity: librarium.Impurity,
			Scripture: augury.SlatePath(s.env(), e.rite.Name), Rite: e.rite.Name,
			Position: librarium.Position{Line: 1, Column: 1}, Message: lament.Error()})
	}
	for _, t := range a.Taint {
		if t.Subject != "" {
			e.note(librarium.Impurity, subjectField(e.rite, t.Subject), "the sanctum subject %q in the vessel %s is "+
				"tainted: other hands altered its scripture, which no longer matches the aspect its marker records "+
				"— the next invocation will overwrite their work", t.Subject, e.rite.ResolvePath(t.Target))
			continue
		}
		e.note(librarium.Impurity, field(t.Number-1, "sanctum"), "the sanctum of verse %d in the vessel %s is "+
			"tainted: other hands altered its scripture, which no longer matches the aspect its marker records "+
			"— the next invocation will overwrite their work", t.Number, e.rite.ResolvePath(t.Target))
	}
	if a.Desecrated {
		e.note(librarium.Impurity, "/aspects", "the rite is desecrated: its omens agree that it stands in the "+
			"aspect %q, yet its data-slate records %q — other hands than the servitor's changed the machine; "+
			"invoke the rite anew to make them one", a.Aspect, a.Former)
	}
}

// subjectField is the JSON pointer of the subject name of the rite r.
func subjectField(r *librarium.Rite, name string) string {
	for i, s := range r.Subjects {
		if s.Name == name {
			return fmt.Sprintf("/subjects/%d", i)
		}
	}
	return "/subjects"
}
