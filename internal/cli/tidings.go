package cli

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/rituals"
)

// verseName names a verse as "verse 2 · tether <target>".
func verseName(v invocation.Verse) string {
	return strings.TrimSpace(fmt.Sprintf("verse %d · %s %s", v.Number, v.Kind.Key(), v.Target))
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

// printSummary tells what a triumphant invocation did.
func printSummary(w io.Writer, res rituals.Result) {
	fmt.Fprintf(w, "+++ The rite %s is performed: %s → %s +++\n", res.Rite.Name, former(res), res.Options.Aspect)
	for _, d := range res.Outcome.Deeds {
		if d.Kind.Key() != "vox-cast" {
			fmt.Fprintln(w, "  "+verseName(d.Verse))
		}
	}
	fmt.Fprintln(w, "The Omnissiah is pleased.")
}

// printForesight tells what an invocation would do.
func printForesight(w io.Writer, res rituals.Result) {
	fmt.Fprintf(w, "+++ Foreseen: the rite %s, %s → %s +++\n", res.Rite.Name, former(res), res.Options.Aspect)
	for _, f := range res.Foresight {
		fmt.Fprintln(w, verseName(f.Verse))
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

func foreseeVessel(w io.Writer, c invocation.VesselChange) {
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

func foreseeTether(w io.Writer, c invocation.TetherChange) {
	from, to := c.From, c.To
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
func forbidden(rite, aspect string, hs invocation.Heresies) error {
	var b strings.Builder
	fmt.Fprintf(&b, "the pre-flight forbids the invocation of %q into %q; nothing was touched:", rite, aspect)
	for _, h := range hs {
		b.WriteString("\n  " + h.Error())
	}
	return errors.New(b.String())
}

// fallen is the lament of an invocation whose step fell: why, its last
// words, and how every reversion went.
func fallen(rite string, out invocation.Outcome) error {
	var b strings.Builder
	f := out.Fell
	fmt.Fprintf(&b, "the rite %q fell at %s: %v", rite, verseName(f.Verse), f.Heresy)
	words(&b, "its last words", f.Output)
	for _, r := range out.Reversions {
		if r.Heresy == nil {
			fmt.Fprintf(&b, "\n  %s is undone", verseName(r.Verse))
		} else {
			fmt.Fprintf(&b, "\n  %s could not be undone: %v", verseName(r.Verse), r.Heresy)
		}
		words(&b, "the words of its reversion", r.Output)
	}
	if out.Verdict == invocation.Faltered {
		b.WriteString("\nThe reversion faltered; the rite lies corrupted. Summon the Inquisition, and mend by hand what remains.")
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
