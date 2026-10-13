package cli

import (
	"fmt"
	"slices"

	"github.com/spf13/cobra"

	"github.com/nerdwave-nick/servitor/internal/rituals"
)

func (a *app) newAbjureCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "abjure <rite> | --all",
		Short: "Withdraw the sanction of what a rite holds exalted, or every sanction, with one proof",
		Args: func(_ *cobra.Command, args []string) error {
			if (all && len(args) == 0) || (!all && len(args) == 1) {
				return nil
			}
			return &ExitError{Code: 2, Err: fmt.Errorf("the abjuring names one rite, or --all alone, yet %d words "+
				"were offered%s", len(args), map[bool]string{true: " beside --all", false: ""}[all])}
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
			if len(args) > 0 || all {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			list, _ := a.servitor(cmd).ListSanctions()
			var out []cobra.Completion
			for _, s := range list {
				out = append(out, cobra.CompletionWithDesc(s.Rite, string(s.Standing)))
			}
			return out, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name, deed := "", "the abjuring of every sanction"
			if len(args) == 1 {
				name, deed = args[0], fmt.Sprintf("the abjuring of the sanction of %q", args[0])
			}
			if err := unexalted(deed); err != nil {
				return err
			}
			abjured, err := a.servitor(cmd).Abjure(name, rituals.Exaltation{})
			for _, rite := range slices.Sorted(slices.Values(abjured)) {
				fmt.Fprintf(cmd.OutOrStdout(), "✠ The sanction of %q is abjured.\n", rite)
			}
			if err != nil {
				return proven(err)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "abjure every sanction laid for you")
	return cmd
}
