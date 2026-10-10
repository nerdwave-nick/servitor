package tui

import (
	"fmt"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

const labelW = 13

func (m *model) detailView(w int) string {
	r := m.current()
	if r == nil {
		return ""
	}
	t := m.t
	label := func(s string) string { return t.label.Render(padRight(strings.ToUpper(s), labelW)) }
	rite := r.rite()
	if rite == nil {
		b := []string{t.danger.Render("✠ This rite is tainted by heresy:"), ""}
		for _, f := range r.heresies {
			b = append(b, t.text.Render(truncate(fmt.Sprintf("%d:%d: %s", f.Line, f.Column, f.Message), w)))
		}
		return strings.Join(append(b, "", t.dim.Render("o purify in $EDITOR · d excommunicate")), "\n")
	}
	a := r.reading.Augury
	var b []string
	if rite.Purpose != "" {
		b = append(b, t.text.Render(rite.Purpose), "")
	}
	var aspects []string
	for _, s := range rite.Aspects {
		if s == a.Aspect {
			aspects = append(aspects, t.accent.Bold(true).Render(t.glyphCurrent+" "+s))
		} else {
			aspects = append(aspects, t.dim.Render(t.glyphOther+" "+s))
		}
	}
	b = append(b, label("aspects")+strings.Join(aspects, "  "))
	glyph, standing := m.standingGlyph(a.Standing)
	line := label("standing") + glyph + " " + standing
	if a.Desecrated && a.Standing != augury.Desecrated {
		line += t.warn.Render(" · desecrated")
	}
	if a.Desecrated {
		line += t.dim.Render(" (the data-slate recorded " + a.Former + ")")
	}
	b = append(b, line, label("last rite")+m.lastRite(a.LastRite))
	b = append(b, label("inscriptions")+m.inscriptionLine(rite, a))
	b = append(b, "", label("liturgy"))
	for _, l := range m.liturgy(rite, a) {
		b = append(b, " "+fit(l, w-1))
	}
	b = append(b, "", label("auspex")+m.auspexLine(rite, a))
	b = append(b, label("recorded in")+t.dim.Render(truncateLeft(shortPath(rite.Path), w-labelW)))
	return strings.Join(b, "\n")
}

func (m *model) lastRite(lr *augury.LastRite) string {
	t := m.t
	if lr == nil {
		return t.dim.Render("never invoked by the servitor")
	}
	at := t.dim.Render(" · " + lr.At.Local().Format("2006-01-02 15:04"))
	if lr.Verdict == invocation.Triumph {
		return t.ok.Render("✔ "+string(lr.Verdict)) + at
	}
	return t.danger.Render("✖ "+string(lr.Verdict)) + at
}

func (m *model) inscriptionLine(rite *librarium.Rite, a augury.Augury) string {
	t := m.t
	if len(rite.Inscriptions) == 0 {
		return t.dim.Render("none declared")
	}
	var parts []string
	for _, k := range rite.InscriptionKeys() {
		v := a.Inscriptions[k]
		if v == "" {
			v = "—"
		}
		parts = append(parts, t.dim.Render(k+"=")+t.text.Render(v))
	}
	return strings.Join(parts, "  ")
}

// omenMark is ✔ for an omen naming the rite's aspect, ✖ (with the aspect
// it names) for one naming another, and · for no omen.
func (m *model) omenMark(o *augury.Omen, aspect string) string {
	t := m.t
	switch {
	case o == nil:
		return t.dim.Render("·")
	case o.Aspect == aspect:
		return t.ok.Render("✔")
	}
	return t.danger.Render("✖")
}

// liturgy is one line per step: its omen, verse, kind glyph and what its
// own key holds.
func (m *model) liturgy(rite *librarium.Rite, a augury.Augury) []string {
	t := m.t
	omens, tainted := map[int]*augury.Omen{}, map[int]bool{}
	for i := range a.Omens {
		omens[a.Omens[i].Number] = &a.Omens[i]
	}
	for _, tt := range a.Taint {
		tainted[tt.Number] = true
	}
	lines := make([]string, len(rite.Liturgy))
	for i, step := range rite.Liturgy {
		o := omens[i+1]
		line := m.omenMark(o, a.Aspect) + " " + t.dim.Render(fmt.Sprintf("%d", i+1)) + " " +
			t.accent.Render(kindGlyph(step.Kind())) + " " + t.text.Render(stepWords(step, a.Aspect))
		if o != nil && o.Aspect != a.Aspect {
			line += t.dim.Render(" (" + o.Aspect + ")")
		}
		if tainted[i+1] {
			line += t.warn.Render(" tainted")
		}
		lines[i] = line
	}
	return lines
}

func (m *model) auspexLine(rite *librarium.Rite, a augury.Augury) string {
	t := m.t
	if rite.Auspex == nil {
		return t.dim.Render("none")
	}
	for i := range a.Omens {
		if o := &a.Omens[i]; o.Auspex {
			return m.omenMark(o, a.Aspect) + " " + t.text.Render(o.Aspect)
		}
	}
	return t.dim.Render("· silent")
}

// kindGlyph is the glyph of a step's kind.
func kindGlyph(k librarium.Kind) string {
	switch k {
	case librarium.KindSanctum:
		return "§"
	case librarium.KindTranscription:
		return "¶"
	case librarium.KindTether:
		return "↣"
	case librarium.KindIncantation:
		return "»"
	case librarium.KindLitany:
		return "≡"
	}
	return "✉"
}

// stepWords is what a step's own key holds as written: the vessel, the
// tether and its anchor, the command, the scroll and its offerings, or the
// tidings. A value written per aspect is shown for aspect.
func stepWords(step librarium.Step, aspect string) string {
	switch s := step.(type) {
	case *librarium.Sanctum:
		return shortPath(s.Vessel)
	case *librarium.Transcription:
		return shortPath(s.Vessel)
	case *librarium.Tether:
		anchor := "(per aspect)"
		if a, ok := forAspect(s.Anchor, aspect); ok {
			anchor = shortPath(a.Path)
			if a.Null {
				anchor = "(unbound)"
			}
		}
		return shortPath(s.Name) + " → " + anchor
	case *librarium.Incantation:
		return written(s.Command, aspect)
	case *librarium.Litany:
		words := shortPath(written(s.Scroll, aspect))
		for _, o := range s.Offerings {
			words += " " + written(o, aspect)
		}
		return words
	case *librarium.VoxCast:
		return s.Tidings
	}
	return ""
}

func written(m librarium.AspectMap[string], aspect string) string {
	if v, ok := forAspect(m, aspect); ok {
		return v
	}
	return "(per aspect)"
}

// forAspect is the value of m written for every aspect, or else for aspect.
func forAspect[T any](m librarium.AspectMap[T], aspect string) (T, bool) {
	if m.IsUniform() || aspect != "" {
		return m.For(aspect)
	}
	var zero T
	return zero, false
}
