package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/nerdwave-nick/servitor/internal/sanction"
)

func (a *app) newSanctionCmd() *cobra.Command {
	var binharic bool
	cmd := &cobra.Command{
		Use:   "sanction [rite]",
		Short: "List the sanctions of your rites' exalted verses, and how each stands",
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
				return refused(fmt.Errorf("%s is refused, and nothing was written: this servitor cannot yet raise "+
					"the exalted servitor that lays a sanction where only the machine's own authority writes — "+
					"`servitor sanction` alone lists the sanctions already laid", deed))
			}
			return a.listSanctions(cmd, binharic)
		},
	}
	binharicRunes(cmd, &binharic, "list the sanctions in binharic for lesser machines")
	return cmd
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
