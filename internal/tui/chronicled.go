package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/chronicle"
	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// chronicled renders one entry of the chronicle whole: when and how the
// rite was invoked, its inscriptions, and for a fallen invocation the verse
// it fell at, its heresy and the verdict upon every reversion.
func (m *model) chronicled(e chronicle.Entry) string {
	t := m.t
	label := func(s string) string { return t.label.Render(padRight(s, labelW)) }
	b := []string{
		t.bold.Render(fmt.Sprintf("The rite %s, %s: %s", e.Rite, turn(e), e.Verdict)), "",
		label("INVOKED") + t.text.Render(e.At.Local().Format("2006-01-02 15:04:05 MST")),
		label("VERDICT") + m.verdictMark(e.Verdict),
		label("INSCRIPTIONS") + m.inscribed(e.Inscriptions),
	}
	if f := e.FellAt; f != nil {
		at := strings.TrimSpace(fmt.Sprintf("verse %d · %s %s", f.Number, f.Key, shortPath(f.Target)))
		b = append(b, "", t.bold.Render("It fell at "+at))
		if e.Heresy != "" {
			b = append(b, t.danger.Render(e.Heresy))
		}
	}
	if len(e.Reversions) > 0 {
		b = append(b, "", t.label.Render("REVERSIONS"))
	}
	for _, r := range e.Reversions {
		if r.Verdict == invocation.Triumph {
			b = append(b, t.ok.Render("✔ ")+t.text.Render(fmt.Sprintf("verse %d is undone", r.Verse)))
		} else {
			b = append(b, t.danger.Render("✖ ")+t.text.Render(fmt.Sprintf("verse %d could not be undone", r.Verse)))
		}
	}
	b = append(b, "")
	switch e.Verdict {
	case invocation.Triumph:
		b = append(b, t.ok.Render("The rite was performed in triumph."))
	case invocation.Reverted:
		b = append(b, t.ok.Render("Every deed is undone; the machine stood as it stood before."))
	default:
		b = append(b, t.danger.Render("The reversion faltered; the rite was left corrupted."))
	}
	return strings.Join(b, "\n")
}

// inscribed renders the inscriptions an invocation carried, by key.
func (m *model) inscribed(ins map[string]string) string {
	if len(ins) == 0 {
		return m.t.dim.Render("none inscribed")
	}
	keys := make([]string, 0, len(ins))
	for k := range ins {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = m.t.dim.Render(k+"=") + m.t.text.Render(ins[k])
	}
	return strings.Join(parts, "  ")
}
