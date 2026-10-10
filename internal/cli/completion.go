package cli

import (
	"io"
	"strings"

	"github.com/spf13/cobra"
)

// completionArgs returns the arguments after cobra's hidden completion
// ritual, and whether args request completion at all. Only the Librarium
// rune may precede the ritual.
func completionArgs(args []string) ([]string, bool) {
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == cobra.ShellCompRequestCmd || arg == cobra.ShellCompNoDescRequestCmd:
			return args[i+1:], true
		case arg == "--librarium" || arg == "-l":
			i++
		case strings.HasPrefix(arg, "-"):
		default:
			return nil, false
		}
	}
	return nil, false
}

func isCompletionRequest(args []string) bool {
	_, ok := completionArgs(args)
	return ok
}

// concealsHelp reports whether the completion requested by args lists the
// rituals of the servitor itself, among which cobra always names its help
// ritual, even when hidden.
func concealsHelp(root *cobra.Command, args []string) bool {
	rest, _ := completionArgs(args)
	if len(rest) == 0 {
		return false
	}
	cmd, _, err := root.Find(rest[:len(rest)-1])
	if err != nil || cmd == nil {
		return false
	}
	return cmd == root || (cmd.Name() == "help" && cmd.Parent() == root)
}

// writeCompletion forwards cobra's completion answer, leaving out the hidden
// help ritual when conceal is set.
func writeCompletion(w io.Writer, answer string, conceal bool) {
	var b strings.Builder
	for line := range strings.Lines(answer) {
		name, _, _ := strings.Cut(strings.TrimRight(line, "\n"), "\t")
		if conceal && name == "help" {
			continue
		}
		b.WriteString(line)
	}
	_, _ = io.WriteString(w, b.String())
}
