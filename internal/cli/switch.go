package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/nerdwave-nick/servitor/internal/config"
	"github.com/nerdwave-nick/servitor/internal/engine"
)

func (a *app) newSwitchCmd() *cobra.Command {
	var foresee, silence bool
	cmd := &cobra.Command{
		Use:   "invoke <rite> <aspect> [--<key> <value>...]",
		Short: "Invoke an aspect of a rite upon its vessels",
		Long:  switchLongHelp,
		Example: `  servitor invoke mouse-autohide-toggle on --reason "gaming remnant"
  servitor invoke mouse-autohide-toggle off --foresee`,
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
	cmd.PersistentFlags().BoolVarP(&foresee, "foresee", "f", false,
		"divine the outcome without touching any vessel")
	cmd.PersistentFlags().BoolVarP(&silence, "silence", "s", false,
		"perform the rite in reverent silence")
	a.switchCmd = cmd
	for _, name := range a.set.Names() {
		cmd.AddCommand(a.newSwitchSubCmd(a.set.Switches[name], &foresee, &silence))
	}
	return cmd
}

// unknownSwitch explains why name is not a usable switch.
func (a *app) unknownSwitch(name string) error {
	if a.set.Broken[name] {
		var b strings.Builder
		fmt.Fprintf(&b, "the rite %q is tainted by heresy:", name)
		for _, d := range a.set.Diags {
			if d.Switch == name && d.Severity == config.SevError {
				b.WriteString("\n  " + formatDiag(d))
			}
		}
		b.WriteString("\nSummon the Inquisition for the full verdict: servitor inquisition")
		return fmt.Errorf("%s", b.String())
	}
	msg := fmt.Sprintf("no rite named %q is recorded in the Librarium (%s)", name, a.set.Dir)
	if s := a.switchCmd.SuggestionsFor(name); len(s) > 0 {
		msg += "\n\nPerhaps you sought:\n\t" + strings.Join(s, "\n\t")
	}
	return fmt.Errorf("%s", msg)
}

func (a *app) newSwitchSubCmd(sw *config.Switch, foresee, silence *bool) *cobra.Command {
	short := sw.Description
	if short == "" {
		short = "Invoke one of the aspects " + strings.Join(sw.States, "|")
	}
	cmd := &cobra.Command{
		Use:     sw.Name + " <" + strings.Join(sw.States, "|") + ">",
		Short:   short,
		Long:    switchLong(sw),
		Example: switchExample(sw),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("the rite demands exactly one aspect (%s), yet %d were offered", strings.Join(sw.States, ", "), len(args))
			}
			if !sw.HasState(args[0]) {
				return unknownState(sw, args[0])
			}
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return stateCompletions(sw), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			overrides := map[string]string{}
			cmd.Flags().Visit(func(f *pflag.Flag) {
				if _, ok := sw.MetaKeys()[f.Name]; ok {
					overrides[f.Name] = f.Value.String()
				}
			})
			res, err := engine.Apply(sw, args[0], overrides, engine.Options{DryRun: *foresee})
			if err != nil {
				return err
			}
			switch {
			case *foresee:
				printPlan(cmd, res.Changes)
			case !*silence:
				printSummary(cmd, sw, args[0], res)
			}
			return nil
		},
	}
	keys := sw.MetaKeys()
	for _, key := range sortedSpecKeys(keys) {
		spec := keys[key]
		usage := spec.Description
		if usage == "" {
			usage = "inscription for " + key
		}
		if spec.Required() {
			usage += " (mandatory unless the aspect prescribes it)"
		}
		cmd.Flags().String(key, "", usage)
		_ = cmd.RegisterFlagCompletionFunc(key, func(*cobra.Command, []string, string) ([]cobra.Completion, cobra.ShellCompDirective) {
			return sw.MetaValues(key), cobra.ShellCompDirectiveNoFileComp
		})
	}
	return cmd
}

func unknownState(sw *config.Switch, state string) error {
	valid := strings.Join(sw.States, ", ")
	return fmt.Errorf("the rite %q knows no aspect %q (known aspects: %s)", sw.Name, state, valid)
}

func printSummary(cmd *cobra.Command, sw *config.Switch, state string, res engine.Result) {
	prev := res.Previous
	if prev == "" {
		prev = "(dormant)"
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "+++ Rite %s performed: %s → %s +++\n", sw.Name, prev, state)
	for _, c := range res.Changes {
		verb := "sanctified"
		switch {
		case c.Created:
			verb = "consecrated"
		case !c.Changed():
			verb = "undisturbed"
		}
		fmt.Fprintf(out, "  %-11s %s\n", verb, c.Path)
	}
	fmt.Fprintln(out, "The Omnissiah is pleased.")
}

func printPlan(cmd *cobra.Command, changes []engine.Change) {
	out := cmd.OutOrStdout()
	for _, c := range changes {
		switch {
		case !c.Changed():
			fmt.Fprintf(out, "%s %s\n", "The augury foresees no change to", c.Path)
			continue
		case c.Created:
			fmt.Fprintf(out, "%s %s\n", "The augury foresees the consecration of", c.Path)
		default:
			fmt.Fprintf(out, "%s %s\n", "The augury foresees changes to", c.Path)
		}
		for _, line := range engine.DiffLines(c.Before, c.After) {
			fmt.Fprintln(out, "  "+line)
		}
	}
}

func switchLong(sw *config.Switch) string {
	var b strings.Builder
	if sw.Description != "" {
		b.WriteString(sw.Description + "\n\n")
	}
	fmt.Fprintf(&b, "Aspects: %s\nRecorded in: %s\n\nVessels:\n", strings.Join(sw.States, ", "), sw.Path)
	for i := range sw.Files {
		f := &sw.Files[i]
		fmt.Fprintf(&b, "  %s (ward %q, glyph %q)\n", sw.Target(f), f.Guard, strings.TrimSpace(f.Comment+" "+f.CommentEnd))
	}
	return strings.TrimRight(b.String(), "\n")
}

func switchExample(sw *config.Switch) string {
	ex := "  servitor invoke " + sw.Name + " " + sw.States[0]
	if keys := sortedSpecKeys(sw.MetaKeys()); len(keys) > 0 {
		ex += " --" + keys[0] + " <value>"
	}
	return ex + "\n  servitor augury " + sw.Name
}

func sortedSpecKeys(m map[string]config.MetaSpec) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
