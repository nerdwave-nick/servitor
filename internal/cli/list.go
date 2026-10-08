package cli

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/nerdwave-nick/servitor/internal/engine"
	"github.com/nerdwave-nick/servitor/internal/lexicon"
)

type listEntry struct {
	Name        string   `json:"name"`
	State       string   `json:"state,omitempty"`
	Status      string   `json:"status"` // applied, not-applied, inconsistent, invalid
	States      []string `json:"states,omitempty"`
	Description string   `json:"description,omitempty"`
	Path        string   `json:"path,omitempty"`
}

func (a *app) newListCmd() *cobra.Command {
	var asJSON bool
	l := a.lex
	cmd := &cobra.Command{
		Use:     l.Cmd.List,
		Aliases: aliases(l, "census", "list", "ls"),
		Short:   l.P("Take a census of all rites and the aspects they stand in", "List configured switches and their current state"),
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			entries := a.listEntries()
			if asJSON {
				return writeJSON(cmd.OutOrStdout(), entries)
			}
			if len(entries) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), l.P(
					fmt.Sprintf("The Librarium at %s holds no rites. Consecrate one in the cogitator (servitor).", a.set.Dir),
					fmt.Sprintf("no switches found in %s (create one in the TUI: servitor)", a.set.Dir)))
				return nil
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, strings.ToUpper(strings.Join([]string{l.Switch, l.State, l.States, l.Description}, "\t")))
			for _, e := range entries {
				state := e.State
				if state == "" {
					state = "(" + statusLabel(l, e.Status) + ")"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", e.Name, state, strings.Join(e.States, "|"), e.Description)
			}
			return tw.Flush()
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, l.P("render the census as JSON for lesser machines", "print as JSON"))
	return cmd
}

func statusLabel(l *lexicon.Lexicon, status string) string {
	switch status {
	case "applied":
		return l.Applied
	case "not-applied":
		return l.NotApplied
	case "inconsistent":
		return l.Inconsistent
	default:
		return l.Broken
	}
}

func (a *app) listEntries() []listEntry {
	entries := []listEntry{}
	for _, name := range a.set.Names() {
		sw := a.set.Switches[name]
		e := listEntry{Name: name, States: sw.States, Description: sw.Description, Path: sw.Path}
		st := engine.ReadStatus(sw)
		switch {
		case st.Err == nil:
			e.State, e.Status = st.State(), "applied"
		case errors.Is(st.Err, engine.ErrNotApplied):
			e.Status = "not-applied"
		default:
			e.Status = "inconsistent"
		}
		entries = append(entries, e)
	}
	for name := range a.set.Broken {
		entries = append(entries, listEntry{Name: name, Status: "invalid",
			Description: a.lex.P("tainted by heresy, summon the Inquisition", "configuration errors, run 'servitor verify'")})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries
}
