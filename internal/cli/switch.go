package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/nerdwave-nick/servitor/internal/config"
	"github.com/nerdwave-nick/servitor/internal/engine"
	"github.com/nerdwave-nick/servitor/internal/lexicon"
)

func (a *app) newSwitchCmd() *cobra.Command {
	var dryRun, quiet bool
	l := a.lex
	cmd := &cobra.Command{
		Use:     l.Cmd.Switch + " <" + l.Switch + "> <" + l.State + "> [--<key> <value>...]",
		Aliases: aliases(l, "invoke", "switch", "profile", "sw"),
		Short:   l.P("Invoke an aspect of a rite upon its vessels", "Apply a state of a switch to its files"),
		Long:    switchLongHelp(l),
		Example: "  servitor " + l.Cmd.Switch + ` mouse-autohide-toggle on --reason "gaming remnant"
  servitor ` + l.Cmd.Switch + " mouse-autohide-toggle off --dry-run",
		Args: cobra.ArbitraryArgs,
		// Switch names complete as subcommands; never fall back to files.
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return a.unknownSwitch(args[0])
		},
	}
	cmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "n", false,
		l.P("divine the outcome without touching any vessel", "show what would change without writing files"))
	cmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false,
		l.P("perform the rite in reverent silence", "do not print a summary"))
	a.switchCmd = cmd
	for _, name := range a.set.Names() {
		cmd.AddCommand(a.newSwitchSubCmd(a.set.Switches[name], &dryRun, &quiet))
	}
	return cmd
}

// unknownSwitch explains why name is not a usable switch.
func (a *app) unknownSwitch(name string) error {
	l := a.lex
	if a.set.Broken[name] {
		var b strings.Builder
		b.WriteString(l.P(fmt.Sprintf("the rite %q is tainted by heresy:", name),
			fmt.Sprintf("switch %q has configuration errors:", name)))
		for _, d := range a.set.Diags {
			if d.Switch == name && d.Severity == config.SevError {
				b.WriteString("\n  " + formatDiag(l, d))
			}
		}
		b.WriteString("\n" + l.P("Summon the Inquisition for the full verdict: servitor "+l.Cmd.Verify,
			"Run 'servitor "+l.Cmd.Verify+"' for details"))
		return fmt.Errorf("%s", b.String())
	}
	msg := l.P(fmt.Sprintf("no rite named %q is recorded in the Librarium (%s)", name, a.set.Dir),
		fmt.Sprintf("unknown switch %q (config: %s)", name, a.set.Dir))
	if s := a.switchCmd.SuggestionsFor(name); len(s) > 0 {
		msg += "\n\n" + l.P("Perhaps you sought:", "Did you mean this?") + "\n\t" + strings.Join(s, "\n\t")
	}
	return fmt.Errorf("%s", msg)
}

func (a *app) newSwitchSubCmd(sw *config.Switch, dryRun, quiet *bool) *cobra.Command {
	l := a.lex
	short := sw.Description
	if short == "" {
		short = l.P("Invoke one of the aspects ", "Switch files between ") + strings.Join(sw.States, "|")
	}
	cmd := &cobra.Command{
		Use:     sw.Name + " <" + strings.Join(sw.States, "|") + ">",
		Short:   short,
		Long:    switchLong(l, sw),
		Example: switchExample(l, sw),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("%s", l.P(
					fmt.Sprintf("the rite demands exactly one aspect (%s), yet %d were offered", strings.Join(sw.States, ", "), len(args)),
					fmt.Sprintf("expected exactly one state (%s), got %d arguments", strings.Join(sw.States, ", "), len(args))))
			}
			if !sw.HasState(args[0]) {
				return unknownState(l, sw, args[0])
			}
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return stateCompletions(l, sw), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			overrides := map[string]string{}
			cmd.Flags().Visit(func(f *pflag.Flag) {
				if _, ok := sw.MetaKeys()[f.Name]; ok {
					overrides[f.Name] = f.Value.String()
				}
			})
			res, err := engine.Apply(sw, args[0], overrides, engine.Options{DryRun: *dryRun})
			if err != nil {
				return err
			}
			switch {
			case *dryRun:
				printPlan(cmd, l, res.Changes)
			case !*quiet:
				printSummary(cmd, l, sw, args[0], res)
			}
			return nil
		},
	}
	keys := sw.MetaKeys()
	for _, key := range sortedSpecKeys(keys) {
		spec := keys[key]
		usage := spec.Description
		if usage == "" {
			usage = l.P("inscription for ", "metadata value for ") + key
		}
		if spec.Required() {
			usage += l.P(" (mandatory unless the aspect prescribes it)", " (required unless configured for the state)")
		}
		cmd.Flags().String(key, "", usage)
		_ = cmd.RegisterFlagCompletionFunc(key, func(*cobra.Command, []string, string) ([]cobra.Completion, cobra.ShellCompDirective) {
			return sw.MetaValues(key), cobra.ShellCompDirectiveNoFileComp
		})
	}
	return cmd
}

func unknownState(l *lexicon.Lexicon, sw *config.Switch, state string) error {
	valid := strings.Join(sw.States, ", ")
	return fmt.Errorf("%s", l.P(
		fmt.Sprintf("the rite %q knows no aspect %q (known aspects: %s)", sw.Name, state, valid),
		fmt.Sprintf("unknown state %q for switch %q (valid: %s)", state, sw.Name, valid)))
}

func printSummary(cmd *cobra.Command, l *lexicon.Lexicon, sw *config.Switch, state string, res engine.Result) {
	prev := res.Previous
	if prev == "" {
		prev = l.P("(dormant)", "(unset)")
	}
	out := cmd.OutOrStdout()
	fmt.Fprint(out, l.P(fmt.Sprintf("+++ Rite %s performed: %s → %s +++\n", sw.Name, prev, state),
		fmt.Sprintf("%s: %s -> %s\n", sw.Name, prev, state)))
	for _, c := range res.Changes {
		verb := l.P("sanctified", "updated")
		switch {
		case c.Created:
			verb = l.P("consecrated", "created")
		case !c.Changed():
			verb = l.P("undisturbed", "unchanged")
		}
		fmt.Fprintf(out, "  %-11s %s\n", verb, c.Path)
	}
	if l.Grimdark {
		fmt.Fprintln(out, "The Omnissiah is pleased.")
	}
}

func printPlan(cmd *cobra.Command, l *lexicon.Lexicon, changes []engine.Change) {
	out := cmd.OutOrStdout()
	for _, c := range changes {
		switch {
		case !c.Changed():
			fmt.Fprintf(out, "%s %s\n", l.P("The augury foresees no change to", "unchanged:"), c.Path)
			continue
		case c.Created:
			fmt.Fprintf(out, "%s %s\n", l.P("The augury foresees the consecration of", "would create:"), c.Path)
		default:
			fmt.Fprintf(out, "%s %s\n", l.P("The augury foresees changes to", "would update:"), c.Path)
		}
		for _, line := range engine.DiffLines(c.Before, c.After) {
			fmt.Fprintln(out, "  "+line)
		}
	}
}

func switchLong(l *lexicon.Lexicon, sw *config.Switch) string {
	var b strings.Builder
	if sw.Description != "" {
		b.WriteString(sw.Description + "\n\n")
	}
	fmt.Fprintf(&b, "%s: %s\n%s: %s\n\n%s:\n", lexicon.Title(l.States), strings.Join(sw.States, ", "),
		l.P("Recorded in", "Defined in"), sw.Path, lexicon.Title(l.Files))
	for i := range sw.Files {
		f := &sw.Files[i]
		fmt.Fprintf(&b, "  %s (%s %q, comment %q)\n", sw.Target(f), l.Guard, f.Guard, strings.TrimSpace(f.Comment+" "+f.CommentEnd))
	}
	return strings.TrimRight(b.String(), "\n")
}

func switchExample(l *lexicon.Lexicon, sw *config.Switch) string {
	ex := "  servitor " + l.Cmd.Switch + " " + sw.Name + " " + sw.States[0]
	if keys := sortedSpecKeys(sw.MetaKeys()); len(keys) > 0 {
		ex += " --" + keys[0] + " <value>"
	}
	return ex + "\n  servitor " + l.Cmd.Meta + " " + sw.Name
}

func sortedSpecKeys(m map[string]config.MetaSpec) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
