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
	var asJSON, noFiles bool
	l := a.lex
	cmd := &cobra.Command{
		Use:     l.Cmd.Verify + " [" + l.Switch + "...]",
		Aliases: aliases(l, "inquisition", "verify", "check", "validate"),
		Short:   l.P("Summon the Inquisition to purge heresy from rites and vessels", "Check switch definitions and managed files for errors"),
		Long:    verifyLongHelp(l),
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
			diags := a.verify(args, !noFiles)
			if asJSON {
				if err := writeJSON(cmd.OutOrStdout(), diags); err != nil {
					return err
				}
			} else {
				for _, d := range diags {
					fmt.Fprintln(cmd.OutOrStdout(), formatDiag(l, d))
				}
				fmt.Fprintln(cmd.ErrOrStderr(), verifySummary(l, a.set, args, diags))
			}
			if diags.HasErrors() {
				return &ExitError{Code: 1}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, l.P("record the verdict as JSON", "print diagnostics as JSON"))
	cmd.Flags().BoolVar(&noFiles, "no-files", false, l.P("judge only the rites, not their vessels", "only check the configuration, not the target files"))
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
				diags = append(diags, engine.CheckTargets(a.lex, a.set.Switches[name])...)
			}
		}
	}
	diags.Sort()
	return diags
}

// formatDiag renders a diagnostic with the severity in the active vocabulary.
func formatDiag(l *lexicon.Lexicon, d config.Diagnostic) string {
	if d.Severity == config.SevError {
		d.Severity = config.Severity(l.Error)
	} else {
		d.Severity = config.Severity(l.Warning)
	}
	return d.String()
}

func verifySummary(l *lexicon.Lexicon, set *config.Set, names []string, diags config.Diagnostics) string {
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
	if !l.Grimdark {
		return fmt.Sprintf("checked %d switch(es) in %s: %d error(s), %d warning(s)", n, set.Dir, errs, warns)
	}
	verdict := "The Emperor protects."
	if errs > 0 {
		verdict = "Purge the heretical, then summon the Inquisition anew."
	}
	return fmt.Sprintf("+++ The Inquisition examined %d rite(s) in %s: %d heres(y/ies), %d impurit(y/ies). %s +++", n, set.Dir, errs, warns, verdict)
}
