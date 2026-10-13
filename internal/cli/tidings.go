package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// verseName names a verse of r as "verse 2 · tether <target>", or a
// subject of a state rite as "subject smt · machine-spirit", a built-in one
// with its target: the home of the faithful is spoken as "~", and a
// litany's scroll beside the rite as it is written.
func verseName(r *librarium.Rite, v invocation.Verse) string {
	if v.Subject != "" {
		sub, _ := r.Subject(v.Subject)
		return strings.TrimSpace(fmt.Sprintf("subject %s · %s %s", v.Subject, sub.Designation(), homeward(v.Target)))
	}
	target := homeward(v.Target)
	if hall := r.Dir() + string(filepath.Separator); v.Kind == librarium.KindLitany && strings.HasPrefix(v.Target, hall) {
		target = strings.TrimPrefix(v.Target, hall)
	}
	return strings.TrimSpace(fmt.Sprintf("verse %d · %s %s", v.Number, v.Kind.Key(), target))
}

// former names the aspect a rite stood in before, or, when none could be
// read, how it stood: dormant, corrupted and the like.
func former(res rituals.Result) string {
	switch {
	case res.Options.Former != "":
		return res.Options.Former
	case res.Former == "":
		return "(" + string(augury.Dormant) + ")"
	}
	return "(" + string(res.Former) + ")"
}

// printForesight tells what an invocation would do.
func printForesight(w io.Writer, res rituals.Result) {
	if res.Rite.Edict {
		fmt.Fprintf(w, "+++ Foreseen: the edict %s +++\n", res.Rite.Name)
	} else {
		fmt.Fprintf(w, "+++ Foreseen: the rite %s, %s → %s +++\n", res.Rite.Name, former(res), res.Options.Aspect)
	}
	if told := foreseenSanction(res); told != "" {
		fmt.Fprintln(w, told)
	}
	for _, f := range res.Foresight {
		name := verseName(res.Rite, f.Verse)
		if f.Exalted {
			name += " ✠"
		}
		if f.Plan != nil {
			sub, _ := res.Rite.Subject(f.Subject)
			name += ": " + foreseenPlan(*f.Plan, sub.Designation() == librarium.DesignationMachineSpirit)
		}
		fmt.Fprintln(w, name)
		switch {
		case f.Vessel != nil:
			foreseeVessel(w, *f.Vessel)
		case f.Tether != nil:
			foreseeTether(w, *f.Tether)
		case f.Speech != nil:
			foreseeSpeech(w, *f.Speech)
		}
	}
	fmt.Fprintln(w, "Nothing was performed.")
}

// foreseenPlan tells what a subject was read to show and whether it
// changes: a machine-spirit also the omen read and the omen commanded,
// «"on" → "off", changing».
func foreseenPlan(p invocation.SubjectPlan, spirit bool) string {
	told := "unchanged"
	if p.Changing {
		told = "changing"
	}
	if !spirit {
		return told
	}
	read := strconv.Quote(p.Read)
	switch {
	case p.Unread:
		read = "(no auspex)"
	case p.Mute:
		read = "(mute)"
	}
	return fmt.Sprintf("%s → %q, %s", read, p.Target, told)
}

// foreseenSanction tells how the exalted verses of the foreseen plan stand
// towards their sanction; "" for a rite without exalted verses.
func foreseenSanction(res rituals.Result) string {
	name := res.Rite.Name
	switch res.Sanction {
	case sanction.Sanctioned:
		return fmt.Sprintf("✠ The exalted verses stand sanctioned under the sigil %s; the exalted servitor performs "+
			"them, after one proof of the master.", sanction.Short(res.Sigil))
	case sanction.Lapsed:
		return fmt.Sprintf("✠ The sanction of the exalted verses has lapsed: they are no longer those their master "+
			"sanctioned, and the invocation would be refused — sanction them anew: servitor sanction %s", name)
	case sanction.Unsanctioned:
		return fmt.Sprintf("✠ The exalted verses stand unsanctioned, and the invocation would be refused — sanction "+
			"them: servitor sanction %s", name)
	}
	return ""
}

func foreseeVessel(w io.Writer, c invocation.VesselChange) {
	if c.Hidden {
		for _, line := range hiddenVessel(c) {
			fmt.Fprintln(w, line)
		}
		return
	}
	switch {
	case !c.Changed():
		fmt.Fprintln(w, "  the vessel stands undisturbed")
		return
	case !c.Exists:
		fmt.Fprintln(w, "  the vessel is struck from the machine")
	case !c.Existed:
		fmt.Fprintf(w, "  the vessel is consecrated, sealed %04o\n", c.AfterSeal.Perm())
	case c.BeforeSeal != c.AfterSeal:
		fmt.Fprintf(w, "  the seal changes from %04o to %04o\n", c.BeforeSeal.Perm(), c.AfterSeal.Perm())
	}
	for _, line := range c.Diff() {
		fmt.Fprintln(w, "  "+line)
	}
}

// hiddenVessel tells what is known of a vessel hidden from the invoker.
func hiddenVessel(c invocation.VesselChange) []string {
	owner, size := c.Owner, "of unknown size"
	if owner == "" {
		owner = "unknown"
	}
	if c.Size >= 0 {
		size = fmt.Sprintf("%d bytes", c.Size)
	}
	out := []string{"  contents hidden — change unknown"}
	if c.Existed {
		out = append(out, fmt.Sprintf("  owner %s, sealed %04o, %s", owner, c.BeforeSeal.Perm(), size))
	}
	switch {
	case !c.Exists:
		out = append(out, "  the vessel is struck from the machine")
	case c.Digest != "":
		out = append(out, fmt.Sprintf("  it shall hold the scripture of SHA-256 %s, sealed %04o", c.Digest,
			c.AfterSeal.Perm()))
	}
	return out
}

func foreseeTether(w io.Writer, c invocation.TetherChange) {
	if c.Hidden {
		fmt.Fprintln(w, "  what stands at the name is hidden — change unknown")
		if c.To != "" {
			fmt.Fprintf(w, "  the tether is bound to %s\n", homeward(c.To))
		}
		return
	}
	from, to := homeward(c.From), homeward(c.To)
	if from == "" {
		from = "nothing"
	}
	switch {
	case !c.Changed():
		fmt.Fprintf(w, "  the tether stays bound to %s\n", to)
		return
	case to == "":
		fmt.Fprintf(w, "  the tether is unbound from %s\n", from)
	default:
		fmt.Fprintf(w, "  the tether is bound to %s (it led to %s)\n", to, from)
	}
	if c.Displaced {
		fmt.Fprintln(w, "  the vessel standing in its place is cast down by zeal")
	}
}

func foreseeSpeech(w io.Writer, sp invocation.Speech) {
	how := "spoken in the tongue " + sp.Tongue
	if sp.Shebang {
		how = "recited by its own shebang"
	}
	fmt.Fprintf(w, "  %s, with the patience of %v: %s\n", how, sp.Patience, quoteArgv(sp.Argv))
	if sp.Reversion != "" {
		fmt.Fprintf(w, "  should the rite fall, its reversion: %s\n", sp.Reversion)
	}
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

// forbidden is the lament of an invocation the pre-flight refused.
func forbidden(r *librarium.Rite, aspect string, hs invocation.Heresies) error {
	var b strings.Builder
	if r.Edict {
		fmt.Fprintf(&b, "the pre-flight forbids the proclaiming of the edict %q; nothing was touched:", r.Name)
	} else {
		fmt.Fprintf(&b, "the pre-flight forbids the invocation of %q into %q; nothing was touched:", r.Name, aspect)
	}
	for _, h := range hs {
		b.WriteString("\n  " + h.Error())
	}
	return errors.New(b.String())
}

// fallen is the lament of an invocation whose step fell: why, its last
// words, and how every reversion went.
func fallen(r *librarium.Rite, out invocation.Outcome) error {
	var b strings.Builder
	f := out.Fell
	kind := "rite"
	if r.Edict {
		kind = "edict"
	}
	fmt.Fprintf(&b, "the %s %q fell at %s: %v", kind, r.Name, verseName(r, f.Verse), f.Heresy)
	words(&b, "its last words", f.Output)
	for _, rv := range out.Reversions {
		if rv.Heresy == nil {
			fmt.Fprintf(&b, "\n  %s is undone", verseName(r, rv.Verse))
		} else {
			fmt.Fprintf(&b, "\n  %s could not be undone: %v", verseName(r, rv.Verse), rv.Heresy)
		}
		words(&b, "the words of its reversion", rv.Output)
	}
	if out.Verdict == invocation.Faltered {
		b.WriteString("\nThe reversion faltered; the " + kind + " lies corrupted. Summon the Inquisition, and mend by hand " +
			"what remains.")
	} else {
		b.WriteString("\nEvery deed is undone; the machine stands as it stood before.")
	}
	return errors.New(b.String())
}

func words(b *strings.Builder, what, output string) {
	if output = strings.TrimRight(output, "\n"); output == "" {
		return
	}
	fmt.Fprintf(b, "\n  %s:", what)
	for _, line := range strings.Split(output, "\n") {
		b.WriteString("\n    " + line)
	}
}
