// Package cli implements the servitor command line interface.
package cli

import (
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
	"github.com/nerdwave-nick/servitor/internal/lexicon"
	"github.com/nerdwave-nick/servitor/internal/tui"
)

// EnvConfig is the environment variable overriding the config directory.
const EnvConfig = "SERVITOR_CONFIG"

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
	noGrim    bool
	set       *config.Set
	lex       *lexicon.Lexicon
	switchCmd *cobra.Command // parent of the per-switch subcommands
}

// runTUI starts the interactive interface; replaced in tests.
var runTUI = func(dir string, lex *lexicon.Lexicon) error {
	return tui.Run(tui.Options{Dir: dir, Lex: lex})
}

// isTerminal reports whether the TUI can run; replaced in tests.
var isTerminal = func() bool {
	return term.IsTerminal(os.Stdin.Fd()) && term.IsTerminal(os.Stdout.Fd())
}

// NewRootCmd builds the command tree. args are the command line arguments
// (without the program name); they are inspected up front so that switches
// from a --config directory become subcommands and the vocabulary is known.
func NewRootCmd(args []string, stdout, stderr io.Writer) *cobra.Command {
	a := &app{configDir: resolveConfigDir(args), lex: lexicon.Get(lexicon.Detect(args))}
	a.set = config.Load(a.configDir)
	l := a.lex

	root := &cobra.Command{
		Use: "servitor",
		Short: l.P("A thrall of the Adeptus Mechanicus that performs rites upon your config files",
			"Switch managed blocks in files between configured states"),
		Long:          rootLong(l),
		Example:       rootExample(l),
		Version:       version(),
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !isTerminal() {
				return cmd.Help()
			}
			return runTUI(a.set.Dir, a.lex)
		},
	}
	root.SetOut(stdout) // before flavor: the completion command captures its writer
	root.SetErr(stderr)
	root.SetVersionTemplate(l.P("servitor, pattern {{.Version}} — blessed be the Omnissiah\n", "servitor version {{.Version}}\n"))
	root.PersistentFlags().StringVarP(&a.configDir, "config", "c", a.configDir,
		l.P("path to the Librarium where rites are kept", "configuration directory")+" (env "+EnvConfig+")")
	_ = root.MarkPersistentFlagDirname("config")
	root.PersistentFlags().BoolVar(&a.noGrim, lexicon.FlagNoGrimdark, !l.Grimdark,
		l.P("forsake the liturgy and speak like a heretic adept (env "+lexicon.EnvNoGrimdark+")",
			"use plain vocabulary instead of the Warhammer 40k theme (env "+lexicon.EnvNoGrimdark+")"))
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return fmt.Errorf("%w\n%s", err, l.P("Consult '"+cmd.CommandPath()+" --help' for the proper liturgy",
			"Run '"+cmd.CommandPath()+" --help' for usage"))
	})
	root.AddCommand(a.newSwitchCmd(), a.newMetaCmd(), a.newListCmd(), a.newVerifyCmd(), a.newTUICmd())
	a.flavor(root)
	return root
}

func (a *app) newTUICmd() *cobra.Command {
	l := a.lex
	return &cobra.Command{
		Use:     l.Cmd.TUI,
		Aliases: aliases(l, "cogitator", "tui", "ui"),
		Short:   l.P("Awaken the cogitator, the interactive shrine of rites", "Open the interactive TUI"),
		Long: l.P(`Awaken the cogitator: survey every rite of the Librarium, invoke aspects,
consecrate new rites, amend or excommunicate old ones. Press ? within for the
full catalogue of keys. Also awakened by invoking servitor without a command.`,
			`Open the interactive TUI: an overview of all switches with their state, and
guided menus to apply, create, edit and delete them. Press ? inside for all
key bindings. Also started by running servitor without a command.`),
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error { return runTUI(a.set.Dir, a.lex) },
	}
}

// aliases returns all names of a command except the primary one, which is
// the first name in grimdark mode and the second one otherwise.
func aliases(l *lexicon.Lexicon, names ...string) []string {
	primary := names[0]
	if !l.Grimdark {
		primary = names[1]
	}
	var out []string
	for _, n := range names {
		if n != primary {
			out = append(out, n)
		}
	}
	return out
}

// flavor applies the vocabulary to cobra's built-in texts.
func (a *app) flavor(root *cobra.Command) {
	l := a.lex
	root.SetUsageTemplate(flavorUsage(l, root.UsageTemplate()))
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()
	root.InitDefaultVersionFlag()
	if f := root.Flags().Lookup("version"); f != nil {
		f.Usage = l.P("reveal the pattern of this servitor", "version for servitor")
	}
	for _, c := range root.Commands() {
		switch c.Name() {
		case "help":
			c.Short = l.P("Consult the lore of any ritual", c.Short)
		case "completion":
			c.Short = l.P("Engrave completion litanies into your shell", c.Short)
			for _, sub := range c.Commands() {
				sub.Short = l.P("Engrave the completion litany for "+sub.Name(), sub.Short)
			}
		}
	}
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		c.InitDefaultHelpFlag()
		if f := c.Flags().Lookup("help"); f != nil {
			f.Usage = l.P("reveal the lore of "+c.Name(), "help for "+c.Name())
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
}

// Execute runs the CLI and returns the process exit code.
func Execute(args []string, stdout, stderr io.Writer) int {
	root := NewRootCmd(args, stdout, stderr)
	root.SetArgs(args)
	err := root.Execute()
	if err == nil {
		return 0
	}
	prefix := lexicon.Get(lexicon.Detect(args)).P("servitor ✠ ", "servitor: ")
	var ee *ExitError
	if errors.As(err, &ee) {
		if ee.Err != nil {
			fmt.Fprintln(stderr, prefix+ee.Err.Error())
		}
		return ee.Code
	}
	fmt.Fprintln(stderr, prefix+err.Error())
	return 1
}

// resolveConfigDir finds the config directory from the last --config/-c in
// args (matching pflag semantics), then $SERVITOR_CONFIG, then the XDG default.
func resolveConfigDir(args []string) string {
	dir := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			i = len(args)
		case arg == "--config" || arg == "-c":
			if i+1 < len(args) {
				i++
				dir = args[i]
			}
		case strings.HasPrefix(arg, "--config="):
			dir = strings.TrimPrefix(arg, "--config=")
		case strings.HasPrefix(arg, "-c") && !strings.HasPrefix(arg, "--") && len(arg) > 2:
			dir = strings.TrimPrefix(arg[2:], "=")
		}
	}
	if dir == "" {
		dir = os.Getenv(EnvConfig)
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
