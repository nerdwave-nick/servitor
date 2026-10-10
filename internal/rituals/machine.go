package rituals

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"

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
			e.sanctum(i, st)
		case *librarium.Tether:
			e.tether(i, st)
		case *librarium.Incantation:
			e.tongue(i, st.Tongue)
		case *librarium.Litany:
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
}

// sanctum examines the vessel of a sanctum for every aspect.
func (e *examiner) sanctum(i int, s *librarium.Sanctum) {
	glyph, closing := s.Glyphs()
	m := sanctum.Marker{Glyph: glyph, ClosingGlyph: closing, Ward: s.WardFor(e.rite.Name)}
	for _, aspect := range e.rite.Aspects {
		p, ok := e.path(aspect, s.Vessel)
		if !ok || !e.once(field(i, "sanctum")+" "+p) {
			continue
		}
		data, err := os.ReadFile(p)
		switch {
		case errors.Is(err, fs.ErrNotExist) && !s.Consecrate:
			e.note(librarium.Impurity, field(i, "sanctum"), "no vessel stands at %s, and the sanctum may not "+
				"consecrate one: every invocation will be refused until the vessel is raised, or until the "+
				"sanctum is granted \"consecrate\": true", p)
		case err != nil:
		default:
			if _, _, err := sanctum.Find(string(data), m); err != nil {
				e.note(librarium.Heresy, field(i, "sanctum"), "the vessel %s is defiled at %v — mend its "+
					"markers by hand, for no invocation can rewrite a sanctum whose bounds are broken", p, err)
			}
		}
	}
}

// tether examines the anchor of a tether for every aspect.
func (e *examiner) tether(i int, t *librarium.Tether) {
	for _, aspect := range e.rite.Aspects {
		anchor, ok := t.Anchor.For(aspect)
		if !ok || anchor.Null {
			continue
		}
		p, okAnchor := e.path(aspect, anchor.Path)
		name, okName := e.path(aspect, t.Name)
		if !okAnchor || !okName || !e.once(field(i, "anchor")+" "+p) {
			continue
		}
		if _, err := os.Stat(p); errors.Is(err, fs.ErrNotExist) {
			e.note(librarium.Impurity, mapField(i, "anchor", t.Anchor, aspect), "the anchor %s, to which the "+
				"tether binds %s in the aspect %q, does not exist: invoked into that aspect, the tether would "+
				"lead into the void", p, name, aspect)
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
	ptr := field(i, "tongue")
	if tongue == "" && e.rite.Tongue != "" {
		ptr = "/tongue" // the rite's own tongue speaks for the step
	}
	if t := e.tongueOf(tongue); !speakable(t) {
		e.note(librarium.Impurity, ptr, "the tongue %q of verse %d is spoken nowhere on this "+
			"machine: no program of that name lies along its PATH, so the step could never be uttered — name "+
			"another \"tongue\" for the step, the rite or the settings", t, i+1)
	}
}

// litanyNeedsTongue reports whether a litany speaks through its tongue: for
// its reversion, or for a scroll that stands and may not be executed.
func (e *examiner) litanyNeedsTongue(l *librarium.Litany) bool {
	if !l.Reversion.IsZero() {
		return true
	}
	for _, aspect := range e.rite.Aspects {
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
