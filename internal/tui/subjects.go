package tui

import (
	"slices"
	"strconv"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// stateRite is the detail of a state rite of pattern Mark II below its
// inscriptions: its subjects in place of a liturgy, its observances, and
// its vox-casts. It bears no auspex of its own: each subject bears its own.
func (m *model) stateRite(rite *librarium.Rite, a augury.Augury, label func(string) string, w int) []string {
	b := []string{"", label("subjects")}
	for _, l := range m.subjectLines(rite, a) {
		b = append(b, " "+fit(l, w-1))
	}
	if len(rite.Observances) > 0 {
		b = append(b, "", label("observances"))
		for _, l := range m.observanceLines(rite) {
			b = append(b, " "+fit(l, w-1))
		}
	}
	return append(b, "", label("vox-cast")+m.voxCasts(rite))
}

// subjectLines are one line per subject: the mark of its omen, its
// designation's glyph (✠ when exalted), its name, and what it was read to
// show — the omen a machine-spirit printed, or a built-in's vessel or bound
// name — with the aspects it fits when they are not the rite's alone.
//
// The mark is ✔ when its omen fits the rite's aspect, ✖ when it does not,
// and · when it yields none, with the reason: mute, unread or forgone. A
// rite that stands in no aspect marks its subjects by the aspects they
// agree on; when they agree on none, those fitting no aspect that the most
// subjects fit are marked ✖ as breaking the agreement.
func (m *model) subjectLines(rite *librarium.Rite, a augury.Augury) []string {
	t := m.t
	omens, tainted := map[string]*augury.Omen{}, map[string]bool{}
	for i := range a.Omens {
		omens[a.Omens[i].Subject] = &a.Omens[i]
	}
	for _, tt := range a.Taint {
		tainted[tt.Subject] = true
	}
	agreed, broken := agreement(a)
	lines := make([]string, len(rite.Subjects))
	for i, sub := range rite.Subjects {
		glyph := designationGlyph(sub.Designation())
		if sub.Exalted() {
			glyph += " ✠"
		}
		o := omens[sub.Name]
		mark, fits := t.dim.Render("·"), o != nil && containsAll(o.Aspects, agreed)
		switch {
		case o == nil:
		case fits:
			mark = t.ok.Render("✔")
		default:
			mark = t.danger.Render("✖")
		}
		line := mark + " " + t.accent.Render(glyph) + " " + t.text.Render(sub.Name) + m.omenRead(sub, o, a, i)
		switch {
		case o == nil:
			line += t.dim.Render(" · " + silence(sub, a))
		case !fits || len(o.Aspects) > 1:
			line += t.dim.Render(" (" + strings.Join(o.Aspects, ", ") + ")")
		}
		if o != nil && o.Stale {
			line += t.dim.Render(" stale")
		}
		if o != nil && !fits && broken {
			line += t.warn.Render(" · breaks the agreement")
		}
		if tainted[sub.Name] {
			line += t.warn.Render(" tainted")
		}
		lines[i] = line
	}
	return lines
}

// omenRead is what the subject at index i was read to show: the omen a
// machine-spirit printed, quoted, even one that fits no aspect; a built-in
// subject's vessel or bound name; "" when nothing is to be told.
func (m *model) omenRead(sub librarium.Subject, o *augury.Omen, a augury.Augury, i int) string {
	t := m.t
	switch s := sub.Matter.(type) {
	case *librarium.Sanctum:
		return " " + t.dim.Render(shortPath(s.Vessel))
	case *librarium.Transcription:
		return " " + t.dim.Render(shortPath(s.Vessel))
	case *librarium.Tether:
		return " " + t.dim.Render(shortPath(s.Name))
	}
	read := ""
	switch {
	case o != nil:
		read = o.Read
	case i < len(a.Readings):
		read = a.Readings[i].Omen
	}
	if read == "" {
		return ""
	}
	return " " + t.text.Render(strconv.Quote(read))
}

// silence tells why the subject yields no omen: unread when it has no
// auspex to be heard outside an invocation, forgone when its auspex was
// left unawakened, else mute — read, yet fitting no aspect.
func silence(sub librarium.Subject, a augury.Augury) string {
	ms, spirit := sub.Matter.(*librarium.MachineSpirit)
	switch {
	case !spirit:
		return "mute"
	case ms.Auspex == nil || ms.Auspex.Exalted:
		return "unread"
	case a.Auspex == augury.Forgone:
		return "forgone"
	}
	return "mute"
}

// agreement names the aspects a subject's omen must fit to be marked as
// agreeing: the rite's aspect, or the aspects its omens agree on. When
// they agree on none (broken), it names those the most subjects fit.
func agreement(a augury.Augury) (agreed []string, broken bool) {
	switch {
	case a.Aspect != "":
		return []string{a.Aspect}, false
	case len(a.Fitting) > 0 || len(a.Omens) < 2:
		return a.Fitting, false
	}
	counts, most := map[string]int{}, 0
	for _, o := range a.Omens {
		for _, aspect := range o.Aspects {
			counts[aspect]++
			most = max(most, counts[aspect])
		}
	}
	for _, o := range a.Omens {
		for _, aspect := range o.Aspects {
			if counts[aspect] == most && !slices.Contains(agreed, aspect) {
				agreed = append(agreed, aspect)
			}
		}
	}
	return agreed, true
}

// containsAll reports whether every one of want stands among have.
func containsAll(have, want []string) bool {
	for _, w := range want {
		if !slices.Contains(have, w) {
			return false
		}
	}
	return true
}

// observanceLines are one line per observance: its designation's glyph (✠
// when exalted), its name, its guards and its words.
func (m *model) observanceLines(rite *librarium.Rite) []string {
	t := m.t
	lines := make([]string, len(rite.Observances))
	for i, o := range rite.Observances {
		glyph := kindGlyph(o.Speech.Kind())
		if o.Exalted() {
			glyph += " ✠"
		}
		lines[i] = t.accent.Render(glyph) + " " + t.text.Render(o.Name) +
			t.dim.Render(guard("to", o.To)+guard("from", o.From)+" · "+stepWords(o.Speech, ""))
	}
	return lines
}

// guard tells the guard named key admitting aspects: "" when unwritten,
// for then it admits every aspect.
func guard(key string, aspects []string) string {
	switch {
	case aspects == nil:
		return ""
	case len(aspects) == 0:
		return " · " + key + " no aspect"
	}
	return " · " + key + " " + strings.Join(aspects, ", ")
}

// voxCasts tells the vox-casts a rite of pattern Mark II bears as its own
// key.
func (m *model) voxCasts(rite *librarium.Rite) string {
	if len(rite.VoxCasts) == 0 {
		return m.t.dim.Render("none")
	}
	return m.t.text.Render(strings.Join(rite.VoxCasts, " · "))
}

// designationGlyph is the glyph of a subject's designation.
func designationGlyph(d string) string {
	switch d {
	case librarium.DesignationSanctum:
		return kindGlyph(librarium.KindSanctum)
	case librarium.DesignationTranscription:
		return kindGlyph(librarium.KindTranscription)
	case librarium.DesignationTether:
		return kindGlyph(librarium.KindTether)
	}
	return "⚙"
}
