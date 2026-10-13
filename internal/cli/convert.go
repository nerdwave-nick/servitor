package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// convert writes the conversions of the Librarium before the ritual cmd is
// performed, and tells each in one line upon stderr — even in binharic —
// unless --silence mutes it; a conversion that cannot be written is
// lamented. Completion, foresight and a servitor bearing the machine's own
// authority write nothing: their Librarium stays converted in memory
// alone. The cogitator converts and tells by itself.
func (a *app) convert(cmd *cobra.Command, _ []string) error {
	if len(a.lib.Conversions) == 0 || a.completing || bearsMachinesAuthority() || runeHeld(cmd, "foresee") ||
		cmd == cmd.Root() || cmd.Name() == "cogitator" {
		return nil
	}
	told := a.servitor(cmd).Convert()
	for _, l := range told.Laments {
		fmt.Fprintln(cmd.ErrOrStderr(), "servitor ✠ "+l.Error())
	}
	if !runeHeld(cmd, "silence") {
		for _, t := range told.Tidings {
			fmt.Fprintln(cmd.ErrOrStderr(), "⋯ "+t)
		}
	}
	return nil
}

// runeHeld reports whether the boolean rune name was spoken to cmd.
func runeHeld(cmd *cobra.Command, name string) bool {
	f := cmd.Flag(name)
	return f != nil && f.Value.String() == "true"
}
