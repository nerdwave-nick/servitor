package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/config"
	"github.com/nerdwave-nick/servitor/internal/engine"
	"github.com/nerdwave-nick/servitor/internal/lexicon"
)

func (m *model) detailView(w int) string {
	r := m.current()
	if r == nil {
		return ""
	}
	t := m.t
	label := func(s string) string { return t.label.Render(padRight(strings.ToUpper(s), 13)) }
	var b []string
	if r.sw == nil {
		b = append(b, t.danger.Render("✠ This rite is tainted by heresy:"), "")
		for _, d := range r.diags {
			b = append(b, t.text.Render(truncate(formatLoc(d)+": "+d.Message, w)))
		}
		return strings.Join(append(b, "", t.dim.Render("o purify in $EDITOR · d excommunicate")), "\n")
	}
	sw := r.sw
	if sw.Description != "" {
		b = append(b, t.text.Render(sw.Description), "")
	}
	var aspects []string
	for _, s := range sw.States {
		if s == r.status.State() {
			aspects = append(aspects, t.accent.Bold(true).Render(t.glyphCurrent+" "+s))
		} else {
			aspects = append(aspects, t.dim.Render(t.glyphOther+" "+s))
		}
	}
	b = append(b, label("aspects")+strings.Join(aspects, "  "))
	glyph, stLabel := m.statusGlyph(*r)
	if r.status.Err == nil {
		stLabel = t.ok.Render(lexicon.Performed)
	}
	b = append(b, label("status")+glyph+" "+stLabel)
	if r.status.Err != nil && !errors.Is(r.status.Err, engine.ErrNotApplied) {
		b = append(b, strings.Repeat(" ", 13)+t.warn.Render(truncate(r.status.Err.Error(), w-13)))
	}
	b = append(b, "", label("inscriptions")+m.metaLine(r))
	b = append(b, "", label("vessels"))
	for _, fs := range r.status.Files {
		mark := t.dim.Render(t.glyphDormant)
		switch {
		case fs.Error != "":
			mark = t.danger.Render("✖")
		case fs.Drift:
			mark = t.warn.Render("◐")
		case fs.Present:
			mark = t.ok.Render("✔")
		}
		b = append(b, " "+mark+" "+t.text.Render(truncateLeft(shortPath(fs.File), w-4)))
		b = append(b, "   "+t.dim.Render("ward "+fs.Guard))
	}
	b = append(b, "", label("recorded in")+t.dim.Render(truncateLeft(shortPath(sw.Path), w-13)))
	return strings.Join(b, "\n")
}

func (m *model) metaLine(r *row) string {
	t := m.t
	keys := r.sw.MetaKeys()
	if len(keys) == 0 {
		return t.dim.Render("none declared")
	}
	var parts []string
	for _, k := range sortedKeys(keys) {
		v := r.status.Meta[k]
		if v == "" {
			v = "—"
		}
		parts = append(parts, t.dim.Render(k+"=")+t.text.Render(v))
	}
	return strings.Join(parts, "  ")
}

func formatLoc(d config.Diagnostic) string {
	if d.Line > 0 {
		return fmt.Sprintf("%d:%d", d.Line, d.Col)
	}
	return shortPath(d.File)
}
