package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/rituals"
)

func (a *app) newCensusCmd() *cobra.Command {
	var binharic bool
	cmd := &cobra.Command{
		Use:   "census",
		Short: "Take a census of all rites and how they stand",
		Long:  censusLongHelp,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			entries, laments := a.servitor(cmd).Census(false)
			for _, l := range laments {
				fmt.Fprintln(cmd.ErrOrStderr(), "servitor ✠ "+l.Error())
			}
			if binharic {
				return writeJSON(cmd.OutOrStdout(), entries)
			}
			if len(entries) == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "The Librarium at %s holds no rites. Consecrate one in the cogitator (servitor).\n", a.lib.Dir)
				return nil
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "RITE\tASPECT\tSTANDING\tASPECTS\tLAST RITE\tPURPOSE")
			for _, e := range entries {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", e.Rite, orDot(e.Aspect), standing(e),
					orDot(strings.Join(e.Aspects, "|")), lastRite(e.LastRite), e.Purpose)
			}
			return tw.Flush()
		},
	}
	binharicRunes(cmd, &binharic, "render the census in binharic for lesser machines")
	return cmd
}

func orDot(s string) string {
	if s == "" {
		return "·"
	}
	return s
}

// standing names how the rite stands, and its desecration when a graver
// standing hides it.
func standing(e rituals.Entry) string {
	if e.Desecrated && e.Standing != augury.Desecrated {
		return string(e.Standing) + ", desecrated"
	}
	return string(e.Standing)
}

func lastRite(l *augury.LastRite) string {
	if l == nil {
		return "·"
	}
	return string(l.Verdict) + " " + l.At.Local().Format("2006-01-02 15:04")
}

// binharicRunes registers --binharic and its hidden alias --json on cmd.
func binharicRunes(cmd *cobra.Command, v *bool, usage string) {
	cmd.Flags().BoolVar(v, "binharic", false, usage)
	cmd.Flags().BoolVar(v, "json", false, usage)
	_ = cmd.Flags().MarkHidden("json")
}
