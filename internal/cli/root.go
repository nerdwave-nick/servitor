// Package cli implements the servitor command line interface.
package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/nerdwave-nick/servitor/internal/config"
	"github.com/nerdwave-nick/servitor/internal/tui"
)

// EnvLibrarium is the environment variable naming the Librarium.
const EnvLibrarium = "SERVITOR_LIBRARIUM"

// ExitError carries a specific process exit code. A nil Err exits silently.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit status %d", e.Code)
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error { return e.Err }

type app struct {
	configDir string
	set       *config.Set
	switchCmd *cobra.Command // parent of the per-rite subcommands
}

// runTUI awakens the cogitator; replaced in tests.
var runTUI = func(dir string) error {
	return tui.Run(tui.Options{Dir: dir})
}

// isTerminal reports whether the cogitator can run; replaced in tests.
var isTerminal = func() bool {
	return term.IsTerminal(os.Stdin.Fd()) && term.IsTerminal(os.Stdout.Fd())
}

// NewRootCmd builds the command tree. args are the command line arguments
// (without the program name); they are inspected up front so that the rites
// of a --librarium directory become subcommands.
func NewRootCmd(args []string, stdout, stderr io.Writer) *cobra.Command {
	a := &app{configDir: resolveConfigDir(args)}
	a.set = config.Load(a.configDir)

	root := &cobra.Command{
		Use:           "servitor",
		Short:         "A thrall of the Adeptus Mechanicus that performs rites upon your config files",
		Long:          rootLong,
		Example:       rootExample,
		Version:       version(),
		Args:          unknownRitual,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !isTerminal() {
				return cmd.Help()
			}
			return runTUI(a.set.Dir)
		},
	}
	root.SetOut(stdout) // before flavor: the completion command captures its writer
	root.SetErr(stderr)
	root.SetVersionTemplate("servitor, pattern {{.Version}} — blessed be the Omnissiah\n")
	root.PersistentFlags().StringVarP(&a.configDir, "librarium", "l", a.configDir,
		"path to the Librarium where rites are kept (env "+EnvLibrarium+")")
	_ = root.MarkPersistentFlagDirname("librarium")
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return fmt.Errorf("%s\nConsult '%s --help' for the proper liturgy", runeError(err), cmd.CommandPath())
	})
	root.AddCommand(a.newSwitchCmd(), a.newMetaCmd(), a.newListCmd(), a.newVerifyCmd(), a.newTUICmd())
	flavor(root)
	return root
}

func (a *app) newTUICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cogitator",
		Short: "Awaken the cogitator, the interactive shrine of rites",
		Long: `Awaken the cogitator: survey every rite of the Librarium, invoke aspects,
consecrate new rites, amend or excommunicate old ones. Press ? within for the
full catalogue of keys. Also awakened by invoking servitor without a ritual.`,
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error { return runTUI(a.set.Dir) },
	}
}

// unknownRitual rejects arguments to the servitor itself: whatever was
// spoken is no ritual it knows.
func unknownRitual(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	msg := fmt.Sprintf("unknown ritual %q for %q", args[0], cmd.CommandPath())
	if s := cmd.SuggestionsFor(args[0]); len(s) > 0 {
		msg += "\n\nPerhaps you sought:\n\t" + strings.Join(s, "\n\t")
	}
	return errors.New(msg)
}

// runeError speaks the complaints of the rune parser in the liturgy.
func runeError(err error) string {
	return strings.NewReplacer(
		"unknown shorthand flag", "unknown rune",
		"unknown flag", "unknown rune",
		"flag needs an argument", "the rune demands a value",
		"bad flag syntax", "malformed rune",
		"invalid argument", "unworthy value",
		`" flag:`, `" rune:`,
	).Replace(err.Error())
}

// Execute runs the CLI and returns the process exit code.
func Execute(args []string, stdout, stderr io.Writer) int {
	var completion bytes.Buffer
	out := stdout
	if isCompletionRequest(args) {
		out = &completion
	}
	root := NewRootCmd(args, out, stderr)
	root.SetArgs(args)
	err := root.Execute()
	if out == &completion {
		writeCompletion(stdout, completion.String(), concealsHelp(root, args))
	}
	if err == nil {
		return 0
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		if ee.Err != nil {
			fmt.Fprintln(stderr, "servitor ✠ "+ee.Err.Error())
		}
		return ee.Code
	}
	fmt.Fprintln(stderr, "servitor ✠ "+err.Error())
	return 1
}

// resolveConfigDir finds the Librarium from the last --librarium/-l in args
// (matching pflag semantics), then $SERVITOR_LIBRARIUM, then the XDG default.
func resolveConfigDir(args []string) string {
	dir := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			i = len(args)
		case arg == "--librarium" || arg == "-l":
			if i+1 < len(args) {
				i++
				dir = args[i]
			}
		case strings.HasPrefix(arg, "--librarium="):
			dir = strings.TrimPrefix(arg, "--librarium=")
		case strings.HasPrefix(arg, "-l") && !strings.HasPrefix(arg, "--") && len(arg) > 2:
			dir = strings.TrimPrefix(arg[2:], "=")
		}
	}
	if dir == "" {
		dir = os.Getenv(EnvLibrarium)
	}
	if dir == "" {
		return config.DefaultDir()
	}
	return expandDir(dir)
}

func expandDir(p string) string {
	p = config.ExpandPath(p, ".")
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}

func version() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
