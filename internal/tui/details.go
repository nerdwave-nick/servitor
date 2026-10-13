package tui

import (
	"fmt"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
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
			b = append(b, wrap([]string{t.text.Render(fmt.Sprintf("%d:%d: %s", f.Line, f.Column, f.Message))}, w)...)
		}
		return strings.Join(append(b, "", t.dim.Render("o purify in $EDITOR · d excommunicate")), "\n")
	}
	a := r.reading.Augury
	var b []string
	if rite.Purpose != "" {
		b = append(b, t.text.Render(rite.Purpose), "")
	}
	b = append(b, m.aspectsLine(rite, a, label))
	glyph, standing := m.standingGlyph(a.Standing)
	line := label("standing") + glyph + " " + standing
	if a.Desecrated && a.Standing != augury.Desecrated {
		line += t.warn.Render(" · desecrated")
	}
	if a.Desecrated {
		line += t.dim.Render(" (the data-slate recorded " + a.Former + ")")
	}
	b = append(b, line)
	if r.sanction != sanction.None {
		b = append(b, label("sanction")+m.sanctionLine(r))
	}
	b = append(b, label("last rite")+m.lastRite(a.LastRite))
	b = append(b, label("inscriptions")+m.inscriptionLine(rite, a))
	if rite.Edict {
		b = append(b, "", label("liturgy"))
		for _, l := range m.liturgy(rite) {
			b = append(b, " "+fit(l, w-1))
		}
		b = append(b, "", label("vox-cast")+m.voxCasts(rite))
	} else { // a rite of aspects: its subjects, each read by itself
		b = append(b, m.stateRite(rite, a, label, w)...)
	}
	b = append(b, label("recorded in")+t.dim.Render(truncateLeft(shortPath(rite.Path), w-labelW)))
	return strings.Join(b, "\n")
}

// aspectsLine shows the aspects of a rite with the current one marked, or
// names an edict, which bears none.
func (m *model) aspectsLine(rite *librarium.Rite, a augury.Augury, label func(string) string) string {
	t := m.t
	if rite.Edict {
		return label("nature") + t.accent.Bold(true).Render(glyphEdict+" edict") +
			t.dim.Render(" · it bears no aspect and is proclaimed by its name alone")
	}
	var aspects []string
	for _, s := range rite.Aspects {
		if s == a.Aspect {
			aspects = append(aspects, t.accent.Bold(true).Render(t.glyphCurrent+" "+s))
		} else {
			aspects = append(aspects, t.dim.Render(t.glyphOther+" "+s))
		}
	}
	return label("aspects") + strings.Join(aspects, "  ")
}

// sanctionLine tells how the rite stands towards its sanction, and what s
// does about it.
func (m *model) sanctionLine(r *row) string {
	t := m.t
	head := m.sanctionGlyph(r.sanction) + " " + string(r.sanction)
	switch r.sanction {
	case sanction.Sanctioned:
		if r.kept != nil {
			return head + t.dim.Render(" · sigil "+sanction.Short(r.kept.Sigil)+" · "+
				r.kept.SanctionedAt.Local().Format("2006-01-02 15:04"))
		}
		return head
	case sanction.Lapsed:
		since := ""
		if r.kept != nil {
			since = " since sigil " + sanction.Short(r.kept.Sigil)
		}
		return head + t.dim.Render(since+" · s to sanction anew")
	case sanction.Unsanctioned:
		return head + t.dim.Render(" · s to sanction its "+exaltedOf(r.rite(), true))
	}
	return head + t.dim.Render(" · no "+exaltedOf(r.rite(), false)+" — s, then a, to abjure")
}

// exaltedOf names what the rite holds exalted, as many or as one: the
// subjects and observances of a state rite of pattern Mark II, else verses.
func exaltedOf(rite *librarium.Rite, many bool) string {
	switch {
	case rite == nil || !rite.MarkII || rite.Edict:
		if many {
			return "exalted verses"
		}
		return "exalted verse"
	case many:
		return "exalted subjects and observances"
	}
	return "exalted subject or observance"
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

// liturgy is one line per entry of an edict's liturgy: its verse, kind
// glyph and words, an exalted entry marked ✠. An edict reveals no omen, so
// its entries bear no mark.
func (m *model) liturgy(rite *librarium.Rite) []string {
	t := m.t
	lines := make([]string, len(rite.Liturgy))
	for i, step := range rite.Liturgy {
		glyph := kindGlyph(step.Kind())
		if librarium.IsExalted(step) {
			glyph += " ✠"
		}
		lines[i] = t.dim.Render(fmt.Sprintf("%d", i+1)) + " " + t.accent.Render(glyph) + " " +
			t.text.Render(stepWords(step, ""))
	}
	return lines
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
	case kindMachineSpirit:
		return designationGlyph(librarium.DesignationMachineSpirit)
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
