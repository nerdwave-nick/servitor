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
	cmd := &cobra.Command{
		Use:   "census",
		Short: "Take a census of all rites and the aspects they stand in",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			entries := a.listEntries()
			if asJSON {
				return writeJSON(cmd.OutOrStdout(), entries)
			}
			if len(entries) == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "The Librarium at %s holds no rites. Consecrate one in the cogitator (servitor).\n", a.set.Dir)
				return nil
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "RITE\tASPECT\tASPECTS\tPURPOSE")
			for _, e := range entries {
				state := e.State
				if state == "" {
					state = "(" + statusLabel(e.Status) + ")"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", e.Name, state, strings.Join(e.States, "|"), e.Description)
			}
			return tw.Flush()
		},
	}
	binharicRunes(cmd, &asJSON, "render the census in binharic for lesser machines")
	return cmd
}

func statusLabel(status string) string {
	switch status {
	case "applied":
		return lexicon.Performed
	case "not-applied":
		return lexicon.Dormant
	case "inconsistent":
		return lexicon.Corrupted
	default:
		return lexicon.Heretical
	}
}

// binharicRunes registers --binharic and its hidden alias --json on cmd.
func binharicRunes(cmd *cobra.Command, v *bool, usage string) {
	cmd.Flags().BoolVar(v, "binharic", false, usage)
	cmd.Flags().BoolVar(v, "json", false, usage)
	_ = cmd.Flags().MarkHidden("json")
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
			Description: "tainted by heresy, summon the Inquisition"})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries
}
