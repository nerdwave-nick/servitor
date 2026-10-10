package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// judgement is one finding of the Inquisition in binharic.
type judgement struct {
	Scripture    string `json:"scripture"`
	Line         int    `json:"line"`
	Column       int    `json:"column"`
	Rite         string `json:"rite,omitempty"` // "" for the settings and the Librarium itself
	Judgement    string `json:"judgement"`      // heresy or impurity
	Denunciation string `json:"denunciation"`
}

func (a *app) newInquisitionCmd() *cobra.Command {
	var binharic, spareVessels bool
	cmd := &cobra.Command{
		Use:   "inquisition [rite...]",
		Short: "Summon the Inquisition to purge heresy from rites, settings and vessels",
		Long:  inquisitionLongHelp,
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
			return a.riteCompletions(args...), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			found, err := a.servitor(cmd).Inquire(args, spareVessels)
			if err != nil {
				return err
			}
			if binharic {
				out := make([]judgement, len(found))
				for i, f := range found {
					out[i] = judgement{Scripture: f.Scripture, Line: f.Line, Column: f.Column, Rite: f.Rite,
						Judgement: f.Severity.String(), Denunciation: f.Message}
				}
				if err := writeJSON(cmd.OutOrStdout(), out); err != nil {
					return err
				}
			} else {
				for _, f := range found {
					fmt.Fprintln(cmd.OutOrStdout(), f.String())
				}
				fmt.Fprintln(cmd.ErrOrStderr(), a.verdict(args, found))
			}
			if found.Heretical() {
				return &ExitError{Code: 1}
			}
			return nil
		},
	}
	binharicRunes(cmd, &binharic, "record the verdict in binharic")
	cmd.Flags().BoolVar(&spareVessels, "spare-vessels", false, "judge only the scriptures, not the machine they act upon")
	return cmd
}

// verdict sums up the findings of the Inquisition.
func (a *app) verdict(names []string, found librarium.Findings) string {
	heresies, impurities := 0, 0
	for _, f := range found {
		if f.Severity == librarium.Heresy {
			heresies++
		} else {
			impurities++
		}
	}
	n := len(names)
	if n == 0 {
		n = len(a.lib.Scriptures)
	}
	verdict := "The Emperor protects."
	if heresies > 0 {
		verdict = "Purge the heretical, then summon the Inquisition anew."
	}
	return fmt.Sprintf("+++ The Inquisition examined %d rite(s) in %s: %d heres(y/ies), %d impurit(y/ies). %s +++",
		n, a.lib.Dir, heresies, impurities, verdict)
}
