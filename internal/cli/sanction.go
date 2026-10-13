package cli

import (
	"errors"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/nerdwave-nick/servitor/internal/rituals"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

func (a *app) newSanctionCmd() *cobra.Command {
	var binharic bool
	cmd := &cobra.Command{
		Use:   "sanction [rite]",
		Short: "Sanction the exalted verses of a rite with one proof, or list the sanctions and how each stands",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 1 {
				return &ExitError{Code: 2, Err: fmt.Errorf("the sanction names at most one rite, yet %d words were "+
					"offered", len(args))}
			}
			return nil
		},
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return a.riteCompletions(), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			deed := "the reading of the sanctions"
			if len(args) == 1 {
				deed = fmt.Sprintf("the sanctioning of the rite %q", args[0])
			}
			if err := unexalted(deed); err != nil {
				return err
			}
			if len(args) == 1 {
				if binharic {
					return refused(fmt.Errorf("%s is refused: its master beholds what they sanction, and the "+
						"sanctioning knows no binharic; speak it without --binharic", deed))
				}
				return a.sanctify(cmd, args[0])
			}
			return a.listSanctions(cmd, binharic)
		},
	}
	binharicRunes(cmd, &binharic, "list the sanctions in binharic for lesser machines")
	return cmd
}

// sanctify sanctions the exalted verses of the rite name: it shows its
// master the unit, or its difference from the sanction that stands, naming
// the sigil, and asks the intercessor for the one proof that lays it. An
// unchanged unit is told and asks no proof; a sanctioning no terminal
// attends is refused.
func (a *app) sanctify(cmd *cobra.Command, name string) error {
	sv := a.servitor(cmd)
	o, err := sv.Offer(name)
	if err != nil {
		return refused(err)
	}
	out := cmd.OutOrStdout()
	if o.Lament != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "servitor ✠ "+o.Lament.Error())
	}
	if o.Unchanged() {
		fmt.Fprintf(out, "✠ The %s of %q stand sanctioned already under the sigil %s; nothing is "+
			"sanctioned anew.\n", o.Rite.ExaltedNoun(), name, sanction.Short(o.Sigil()))
		return nil
	}
	x := rituals.Exaltation{Attended: rituals.Intercession.Attended()}
	if !x.Attended {
		return refused(&rituals.Unattended{Rite: name})
	}
	if o.Kept == nil {
		fmt.Fprintf(out, "✠ The %s of %q, offered for sanction under the sigil %s:\n\n", o.Rite.ExaltedNoun(), name,
			o.Sigil())
	} else {
		fmt.Fprintf(out, "✠ The %s of %q changed since they were sanctioned under the sigil %s; offered "+
			"anew under the sigil %s:\n\n", o.Rite.ExaltedNoun(), name, sanction.Short(o.Kept.Sigil), o.Sigil())
	}
	for _, line := range o.Shown() {
		fmt.Fprintln(out, line)
	}
	fmt.Fprintf(out, "\n⋯ The intercessor asks the master's proof; the proof is the sanction.\n")
	if err := sv.Sanction(o, x); err != nil {
		return proven(err)
	}
	fmt.Fprintf(out, "✠ The %s of %q stand sanctioned under the sigil %s.\n", o.Rite.ExaltedNoun(), name,
		sanction.Short(o.Sigil()))
	return nil
}

// proven ends a ritual that asked the intercessor for the master's proof:
// a proof dismissed or failed, or an abjuring that fell midway, bears the
// death-mark 1; every other lament is a refusal before anything was
// written, the death-mark 2.
func proven(err error) error {
	var dismissed *rituals.Dismissed
	var fallen *rituals.Fallen
	if errors.As(err, &dismissed) || errors.As(err, &fallen) {
		return &ExitError{Code: 1, Err: err}
	}
	return refused(err)
}

// listSanctions speaks every sanction of the master: rite, standing, sigil
// and the hour of its sanctioning.
func (a *app) listSanctions(cmd *cobra.Command, binharic bool) error {
	list, laments := a.servitor(cmd).ListSanctions()
	for _, l := range laments {
		fmt.Fprintln(cmd.ErrOrStderr(), "servitor ✠ "+l.Error())
	}
	if binharic {
		return writeJSON(cmd.OutOrStdout(), list)
	}
	if len(list) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "No sanction is laid for any rite. Write \"exalted\": true upon a verse, "+
			"then sanction it: servitor sanction <rite>")
		return nil
	}
	tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "RITE\tSTANDING\tSIGIL\tSANCTIONED AT")
	for _, s := range list {
		fmt.Fprintf(tw, "%s\t%s %s\t%s\t%s\n", s.Rite, sanctionGlyph(s.Standing), s.Standing, sanction.Short(s.Sigil),
			s.SanctionedAt.Local().Format("2006-01-02 15:04"))
	}
	return tw.Flush()
}

// sanctionGlyph marks a sanction standing in the tables of the census and
// the sanctions; a rite without exalted verses bears none.
func sanctionGlyph(s sanction.Standing) string {
	switch s {
	case sanction.Sanctioned:
		return "✠"
	case sanction.Lapsed:
		return "☓"
	case sanction.Unsanctioned:
		return "○"
	case sanction.Forsaken:
		return "†"
	}
	return ""
}
