package cli

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nerdwave-nick/servitor/internal/block"
	"github.com/nerdwave-nick/servitor/internal/config"
	"github.com/nerdwave-nick/servitor/internal/engine"
)

func (a *app) newMetaCmd() *cobra.Command {
	var is string
	var perFile bool
	cmd := &cobra.Command{
		Use:   "augury <rite> [key]",
		Short: "Perform an augury: read the inscriptions of a rite",
		Long:  metaLongHelp,
		Example: `  servitor augury mouse-autohide-toggle
  servitor augury mouse-autohide-toggle reason
  if servitor augury mouse-autohide-toggle --is on; then echo "the cursor is veiled"; fi
  servitor augury mouse-autohide-toggle --per-file`,
		Args:              cobra.RangeArgs(1, 2),
		ValidArgsFunction: a.completeMetaArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			sw, err := a.lookup(args[0])
			if err != nil {
				return &ExitError{Code: 2, Err: err}
			}
			if is != "" && !sw.HasState(is) {
				return &ExitError{Code: 2, Err: unknownState(sw, is)}
			}
			st := engine.ReadStatus(sw)
			out := cmd.OutOrStdout()
			switch {
			case perFile:
				return writeJSON(out, st.Files)
			case is != "":
				if st.Err == nil && st.State() == is {
					return nil
				}
				return &ExitError{Code: 1}
			case st.Err != nil:
				return &ExitError{Code: 2, Err: statusError(sw, st.Err)}
			case len(args) == 2:
				key := args[1]
				if _, ok := sw.MetaKeys()[key]; !ok && key != block.StateKey {
					keys := strings.Join(metaKeyNames(sw), ", ")
					return &ExitError{Code: 2, Err: fmt.Errorf("the rite %q bears no inscription %q (inscriptions: %s)", sw.Name, key, keys)}
				}
				fmt.Fprintln(out, st.Meta[key])
				return nil
			default:
				return writeOrderedMeta(out, st.Meta)
			}
		},
	}
	cmd.Flags().StringVar(&is, "is", "", "exit 0 if the rite stands in this aspect, 1 otherwise")
	cmd.Flags().BoolVar(&perFile, "per-file", false, "augur every vessel separately, as JSON")
	cmd.MarkFlagsMutuallyExclusive("is", "per-file")
	_ = cmd.RegisterFlagCompletionFunc("is", func(cmd *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
		if len(args) > 0 {
			if sw := a.set.Switches[args[0]]; sw != nil {
				return stateCompletions(sw), cobra.ShellCompDirectiveNoFileComp
			}
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func statusError(sw *config.Switch, err error) error {
	if errors.Is(err, engine.ErrNotApplied) {
		return fmt.Errorf("the rite %q lies dormant, never yet performed (%w)", sw.Name, err)
	}
	return fmt.Errorf("the augury of %q is clouded, its vessels are corrupted (%w)", sw.Name, err)
}

// lookup returns the usable switch called name or a descriptive error.
func (a *app) lookup(name string) (*config.Switch, error) {
	if sw := a.set.Switches[name]; sw != nil {
		return sw, nil
	}
	return nil, a.unknownSwitch(name)
}

func metaKeyNames(sw *config.Switch) []string {
	return append([]string{block.StateKey}, sortedSpecKeys(sw.MetaKeys())...)
}

// writeOrderedMeta prints meta as a JSON object with "state" first.
func writeOrderedMeta(w io.Writer, meta map[string]string) error {
	var b strings.Builder
	b.WriteString("{")
	for i, k := range block.SortedKeys(meta) {
		if i > 0 {
			b.WriteString(",")
		}
		kj, _ := json.Marshal(k)
		vj, _ := json.Marshal(meta[k])
		fmt.Fprintf(&b, "\n  %s: %s", kj, vj)
	}
	b.WriteString("\n}\n")
	_, err := io.WriteString(w, b.String())
	return err
}

func writeJSON(w io.Writer, v any) error {
	if err := json.MarshalWrite(w, v, jsontext.WithIndent("  "), json.Deterministic(true)); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func (a *app) completeMetaArgs(cmd *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return a.switchCompletions(), cobra.ShellCompDirectiveNoFileComp
	case 1:
		sw := a.set.Switches[args[0]]
		if sw == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var out []cobra.Completion
		keys := sw.MetaKeys()
		for _, k := range metaKeyNames(sw) {
			desc := keys[k].Description
			if k == block.StateKey {
				desc = "current aspect (" + strings.Join(sw.States, "|") + ")"
			}
			out = append(out, cobra.CompletionWithDesc(k, desc))
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

// switchCompletions lists all usable rites with their purpose.
func (a *app) switchCompletions() []cobra.Completion {
	var out []cobra.Completion
	for _, name := range a.set.Names() {
		sw := a.set.Switches[name]
		desc := sw.Description
		if desc == "" {
			desc = strings.Join(sw.States, "|")
		}
		out = append(out, cobra.CompletionWithDesc(name, desc))
	}
	return out
}

// stateCompletions lists the aspects of sw, marking the current one.
func stateCompletions(sw *config.Switch) []cobra.Completion {
	current := engine.ReadStatus(sw).State()
	out := make([]cobra.Completion, 0, len(sw.States))
	for _, s := range sw.States {
		desc := ""
		if s == current {
			desc = "current aspect"
		}
		out = append(out, cobra.CompletionWithDesc(s, desc))
	}
	return out
}
