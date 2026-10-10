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

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
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
	chronicle string // --chronicle and its hidden alias --log; read through runesOf
	lib       *librarium.Librarium
	invokeCmd *cobra.Command // parent of the per-rite rituals
}

// servitor makes the Librarium ready for the ritual cmd, swayed by its runes.
func (a *app) servitor(cmd *cobra.Command) *rituals.Servitor {
	return &rituals.Servitor{Librarium: a.lib, Runes: runesOf(cmd)}
}

// runTUI awakens the cogitator; replaced in tests.
var runTUI = func(opt tui.Options) error {
	return tui.Run(opt)
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
	a.lib = librarium.Load(a.configDir)

	root := &cobra.Command{
		Use:           "servitor",
		Short:         "A thrall of the Adeptus Mechanicus that performs rites upon your config files",
		Version:       version(),
		Args:          unknownRitual,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !isTerminal() {
				return cmd.Help()
			}
			return a.cogitator(cmd)
		},
	}
	root.SetOut(stdout) // before flavor: the completion command captures its writer
	root.SetErr(stderr)
	root.SetVersionTemplate("servitor, pattern {{.Version}} — blessed be the Omnissiah\n")
	root.PersistentFlags().StringVarP(&a.configDir, "librarium", "l", a.configDir,
		"path to the Librarium where rites are kept (env "+EnvLibrarium+")")
	_ = root.MarkPersistentFlagDirname("librarium")
	chronicleRunes(root, &a.chronicle)
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return fmt.Errorf("%s\n%s", runeError(err), liturgyHint(cmd))
	})
	root.AddCommand(a.newInvokeCmd(), a.newAuguryCmd(), a.newCensusCmd(), a.newInquisitionCmd(), a.newTUICmd(),
		a.newExpoundCmd())
	root.SetHelpCommand(newHelpCmd())
	root.SetHelpFunc(a.lore(root.HelpFunc()))
	flavor(root)
	return root
}

func (a *app) newTUICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cogitator",
		Short: "Awaken the cogitator, the interactive shrine of rites",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return a.cogitator(cmd) },
	}
}

// cogitator awakens the cogitator upon the Librarium, swayed by the runes
// spoken to the ritual cmd.
func (a *app) cogitator(cmd *cobra.Command) error {
	return runTUI(tui.Options{Dir: a.lib.Dir, Runes: runesOf(cmd)})
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
		return defaultDir()
	}
	return expandDir(dir)
}

// defaultDir is the Librarium when none is named: $XDG_CONFIG_HOME/servitor,
// else ~/.config/servitor.
func defaultDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "servitor")
}

// expandDir expands a leading "~" and $VARS in p and makes it absolute.
func expandDir(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			p = home + p[1:]
		}
	}
	p = os.ExpandEnv(p)
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return filepath.Clean(p)
}

func version() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

// chronicleRunes registers --chronicle and its hidden alias --log for every
// ritual under root.
func chronicleRunes(root *cobra.Command, v *string) {
	const usage = "path of the chronicle in which every invocation is recorded (env " + librarium.EnvChronicle + ")"
	root.PersistentFlags().StringVar(v, "chronicle", "", usage)
	root.PersistentFlags().StringVar(v, "log", "", usage)
	_ = root.PersistentFlags().MarkHidden("log")
	_ = root.MarkPersistentFlagFilename("chronicle")
}

// runesOf returns the runes spoken to the ritual cmd that sway the settings,
// to be resolved by (*librarium.Settings).Resolve.
func runesOf(cmd *cobra.Command) librarium.Runes {
	var r librarium.Runes
	if f := cmd.Root().PersistentFlags().Lookup("chronicle"); f != nil {
		r.Chronicle = f.Value.String()
	}
	return r
}
