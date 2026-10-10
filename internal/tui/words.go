package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// verseName names a verse as "verse 2 · tether <target>".
func verseName(v invocation.Verse) string {
	return strings.TrimSpace(fmt.Sprintf("verse %d · %s %s", v.Number, v.Kind.Key(), shortPath(v.Target)))
}

// former names the aspect a rite stood in before, or its dormancy.
func former(aspect string) string {
	if aspect == "" {
		return "dormant"
	}
	return aspect
}

// foreseen tells what one step would do, a line each, indented.
func foreseen(f invocation.Foresight) []string {
	switch {
	case f.Vessel != nil:
		return foreseenVessel(*f.Vessel)
	case f.Tether != nil:
		return foreseenTether(*f.Tether)
	case f.Speech != nil:
		return foreseenSpeech(*f.Speech)
	}
	return nil
}

func foreseenVessel(c invocation.VesselChange) []string {
	var out []string
	switch {
	case !c.Changed():
		return []string{"  the vessel stands undisturbed"}
	case !c.Exists:
		out = append(out, "  the vessel is struck from the machine")
	case !c.Existed:
		out = append(out, fmt.Sprintf("  the vessel is consecrated, sealed %04o", c.AfterSeal.Perm()))
	case c.BeforeSeal != c.AfterSeal:
		out = append(out, fmt.Sprintf("  the seal changes from %04o to %04o", c.BeforeSeal.Perm(), c.AfterSeal.Perm()))
	}
	for _, line := range c.Diff() {
		out = append(out, "  "+line)
	}
	return out
}

func foreseenTether(c invocation.TetherChange) []string {
	from, to := c.From, c.To
	if from == "" {
		from = "nothing"
	}
	var out []string
	switch {
	case !c.Changed():
		return []string{"  the tether stays bound to " + to}
	case to == "":
		out = append(out, "  the tether is unbound from "+from)
	default:
		out = append(out, "  the tether is bound to "+to+" (it led to "+from+")")
	}
	if c.Displaced {
		out = append(out, "  the vessel standing in its place is cast down by zeal")
	}
	return out
}

func foreseenSpeech(sp invocation.Speech) []string {
	how := "spoken in the tongue " + sp.Tongue
	if sp.Shebang {
		how = "recited by its own shebang"
	}
	out := []string{fmt.Sprintf("  %s, with the patience of %v: %s", how, sp.Patience, quoteArgv(sp.Argv))}
	if sp.Reversion != "" {
		out = append(out, "  should the rite fall, its reversion: "+sp.Reversion)
	}
	return out
}

// quoteArgv renders argv as a reader would type it.
func quoteArgv(argv []string) string {
	out := make([]string, len(argv))
	for i, a := range argv {
		out[i] = a
		if a == "" || strings.ContainsAny(a, " \t\n'\"\\$`;&|<>()*?[]#~") {
			out[i] = strconv.Quote(a)
		}
	}
	return strings.Join(out, " ")
}

// uttered renders the words a command uttered, indented; "" when silent.
func uttered(what, output string) []string {
	if output = strings.TrimRight(output, "\n"); output == "" {
		return nil
	}
	out := []string{"  " + what + ":"}
	for _, line := range strings.Split(output, "\n") {
		out = append(out, "    "+line)
	}
	return out
}

// fallen tells why the rite fell, its last words and how every reversion
// went.
func (m *model) fallen(out invocation.Outcome) string {
	t := m.t
	f := out.Fell
	b := []string{t.bold.Render("It fell at " + verseName(f.Verse)), t.danger.Render(f.Heresy.Error())}
	for _, l := range uttered("its last words", f.Output) {
		b = append(b, t.text.Render(l))
	}
	if len(out.Reversions) > 0 {
		b = append(b, "", t.label.Render("REVERSIONS"))
	}
	for _, r := range out.Reversions {
		if r.Heresy == nil {
			b = append(b, t.ok.Render("✔ ")+t.text.Render(verseName(r.Verse)+" is undone"))
		} else {
			b = append(b, t.danger.Render("✖ ")+t.text.Render(verseName(r.Verse)+" could not be undone: "+r.Heresy.Error()))
		}
		for _, l := range uttered("the words of its reversion", r.Output) {
			b = append(b, t.dim.Render(l))
		}
	}
	b = append(b, "")
	if out.Verdict == invocation.Faltered {
		b = append(b, t.danger.Render("The reversion faltered; the rite lies corrupted. Summon the Inquisition, "+
			"and mend by hand what remains."))
	} else {
		b = append(b, t.ok.Render("Every deed is undone; the machine stands as it stood before."))
	}
	return strings.Join(b, "\n")
}

// showWords shows every word the last invocation uttered.
func (m *model) showWords() tea.Cmd {
	if m.last == nil || m.last.Outcome == nil {
		return m.notify(toastInfo, "No rite has yet been invoked in this cogitator; it holds no words.")
	}
	t := m.t
	out := m.last.Outcome
	b := []string{t.bold.Render(fmt.Sprintf("The rite %s, %s → %s: %s", m.last.Rite.Name,
		former(m.last.Options.Former), m.last.Options.Aspect, out.Verdict)), ""}
	for _, d := range out.Deeds {
		if d.Kind == librarium.KindVoxCast {
			continue
		}
		b = append(b, t.accent.Render(verseName(d.Verse)))
		b = append(b, m.words(d.Output)...)
	}
	if len(out.Reversions) > 0 {
		b = append(b, "", t.label.Render("REVERSIONS"))
	}
	for _, r := range out.Reversions {
		b = append(b, t.accent.Render(verseName(r.Verse)))
		b = append(b, m.words(r.Output)...)
	}
	m.screen = newTextScreen("Words of the last invocation", strings.Join(b, "\n"))
	return nil
}

func (m *model) words(output string) []string {
	lines := uttered("its words", output)
	if lines == nil {
		return []string{m.t.dim.Render("  silent")}
	}
	for i, l := range lines {
		lines[i] = m.t.text.Render(l)
	}
	return lines
}
