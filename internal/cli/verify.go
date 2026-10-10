package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nerdwave-nick/servitor/internal/config"
	"github.com/nerdwave-nick/servitor/internal/engine"
	"github.com/nerdwave-nick/servitor/internal/lexicon"
)

func (a *app) newVerifyCmd() *cobra.Command {
	var asJSON, spareVessels bool
	cmd := &cobra.Command{
		Use:   "inquisition [rite...]",
		Short: "Summon the Inquisition to purge heresy from rites and vessels",
		Long:  verifyLongHelp,
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
			var out []cobra.Completion
			for _, c := range a.switchCompletions() {
				if name, _, _ := strings.Cut(c, "\t"); !slices.Contains(args, name) {
					out = append(out, c)
				}
			}
			return out, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, n := range args {
				if a.set.Switches[n] == nil && !a.set.Broken[n] {
					return a.unknownSwitch(n)
				}
			}
			diags := a.verify(args, !spareVessels)
			if asJSON {
				if err := writeJSON(cmd.OutOrStdout(), diags); err != nil {
					return err
				}
			} else {
				for _, d := range diags {
					fmt.Fprintln(cmd.OutOrStdout(), formatDiag(d))
				}
				fmt.Fprintln(cmd.ErrOrStderr(), verifySummary(a.set, args, diags))
			}
			if diags.HasErrors() {
				return &ExitError{Code: 1}
			}
			return nil
		},
	}
	binharicRunes(cmd, &asJSON, "record the verdict in binharic")
	cmd.Flags().BoolVar(&spareVessels, "spare-vessels", false, "judge only the rites, not their vessels")
	return cmd
}

func (a *app) verify(names []string, checkFiles bool) config.Diagnostics {
	want := func(name string) bool { return len(names) == 0 || slices.Contains(names, name) }
	diags := config.Diagnostics{}
	for _, d := range a.set.Diags {
		if d.Switch == "" || want(d.Switch) {
			diags = append(diags, d)
		}
	}
	if checkFiles {
		for _, name := range a.set.Names() {
			if want(name) {
				diags = append(diags, engine.CheckTargets(a.set.Switches[name])...)
			}
		}
	}
	diags.Sort()
	return diags
}

// formatDiag renders a diagnostic as a heresy or an impurity.
func formatDiag(d config.Diagnostic) string {
	if d.Severity == config.SevError {
		d.Severity = lexicon.Heresy
	} else {
		d.Severity = lexicon.Impurity
	}
	return d.String()
}

func verifySummary(set *config.Set, names []string, diags config.Diagnostics) string {
	errs, warns := 0, 0
	for _, d := range diags {
		if d.Severity == config.SevError {
			errs++
		} else {
			warns++
		}
	}
	n := len(names)
	if n == 0 {
		n = len(set.Switches) + len(set.Broken)
	}
	verdict := "The Emperor protects."
	if errs > 0 {
		verdict = "Purge the heretical, then summon the Inquisition anew."
	}
	return fmt.Sprintf("+++ The Inquisition examined %d rite(s) in %s: %d heres(y/ies), %d impurit(y/ies). %s +++", n, set.Dir, errs, warns, verdict)
}
