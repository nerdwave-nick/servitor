package cli

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
)

func (a *app) newInvokeCmd() *cobra.Command {
	var foresee, silence bool
	cmd := &cobra.Command{
		Use:   "invoke <rite> <aspect> [--<inscription> <value>...]",
		Short: "Invoke an aspect of a rite: perform its liturgy",
		Args:  cobra.ArbitraryArgs,
		// Rites complete as rituals of their own; never fall back to files.
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return a.unfitRite(args[0])
		},
	}
	cmd.PersistentFlags().BoolVarP(&foresee, "foresee", "f", false,
		"divine the outcome without touching anything, not even the auspex")
	cmd.PersistentFlags().BoolVarP(&silence, "silence", "s", false,
		"perform the rite in reverent silence")
	a.invokeCmd = cmd
	for _, name := range a.lib.Names() {
		if r := a.lib.Rites[name]; r != nil {
			cmd.AddCommand(a.newRiteCmd(r, &foresee, &silence))
		}
	}
	return cmd
}

// unfitRite explains why name cannot be invoked.
func (a *app) unfitRite(name string) error {
	_, err := (&rituals.Servitor{Librarium: a.lib}).Rite(name)
	var unrecorded *rituals.Unrecorded
	if errors.As(err, &unrecorded) {
		if s := a.invokeCmd.SuggestionsFor(name); len(s) > 0 {
			return fmt.Errorf("%w\n\nPerhaps you sought:\n\t%s", err, strings.Join(s, "\n\t"))
		}
	}
	return err
}

func (a *app) newRiteCmd(r *librarium.Rite, foresee, silence *bool) *cobra.Command {
	short := r.Purpose
	if short == "" {
		short = "Invoke one of the aspects " + strings.Join(r.Aspects, "|")
	}
	cmd := &cobra.Command{
		Use:     r.Name + " <" + strings.Join(r.Aspects, "|") + ">",
		Short:   short,
		Long:    riteLong(r),
		Example: riteExample(r),
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("the rite demands exactly one aspect (%s), yet %d were offered",
					strings.Join(r.Aspects, ", "), len(args))
			}
			if !r.HasAspect(args[0]) {
				return unknownAspect(r, args[0])
			}
			return nil
		},
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]cobra.Completion, cobra.ShellCompDirective) {
			return a.aspectCompletions(r), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.invoke(cmd, r, args[0], *foresee, *silence)
		},
	}
	for _, in := range r.Inscriptions {
		usage := in.Purpose
		if usage == "" {
			usage = "inscription " + in.Key
		}
		if in.Mandatory {
			usage += " (mandatory, unless the aspect decrees it)"
		}
		cmd.Flags().Var(wordRune(new(string), "", "word"), in.Key, usage)
		_ = cmd.RegisterFlagCompletionFunc(in.Key, func(*cobra.Command, []string, string) ([]cobra.Completion, cobra.ShellCompDirective) {
			return decreeCompletions(in), cobra.ShellCompDirectiveNoFileComp
		})
	}
	return cmd
}

// invoke performs (or foresees) the rite r into aspect, heeding an
// interrupt from the terminal by halting and reverting the invocation.
func (a *app) invoke(cmd *cobra.Command, r *librarium.Rite, aspect string, foresee, silence bool) error {
	runes := map[string]string{}
	for _, in := range r.Inscriptions {
		if f := cmd.Flags().Lookup(in.Key); f != nil && f.Changed {
			runes[in.Key] = f.Value.String()
		}
	}
	s := a.servitor(cmd)
	p := rituals.Petition{Rite: r.Name, Aspect: aspect, Runes: runes, Foresee: foresee}
	var told *narration
	if !foresee {
		told = newNarration(cmd.OutOrStdout(), s.Orders().Vox, silence)
		p.Herald, p.Commence = told, told.commence
	}
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	res, err := s.Invoke(ctx, p)
	stop()
	if told != nil {
		told.end(res.Outcome)
	}
	for _, l := range res.Laments {
		fmt.Fprintln(cmd.ErrOrStderr(), "servitor ✠ "+l.Error())
	}
	if foresee && res.Rite != nil {
		printForesight(cmd.OutOrStdout(), res)
	}
	var hs invocation.Heresies
	switch {
	case errors.As(err, &hs):
		return forbidden(r.Name, aspect, hs)
	case err != nil:
		return err
	case foresee:
		return nil
	case res.Outcome.Verdict != invocation.Triumph:
		return &ExitError{Code: 1, Err: fallen(r, *res.Outcome)}
	}
	return nil
}

func unknownAspect(r *librarium.Rite, aspect string) error {
	return fmt.Errorf("the rite %q knows no aspect %q (known aspects: %s)", r.Name, aspect, strings.Join(r.Aspects, ", "))
}

func riteLong(r *librarium.Rite) string {
	var b strings.Builder
	if r.Purpose != "" {
		b.WriteString(r.Purpose + "\n\n")
	}
	fmt.Fprintf(&b, "Aspects: %s\nRecorded in: %s\n\nLiturgy:\n", strings.Join(r.Aspects, ", "), r.Path)
	for i, step := range r.Liturgy {
		fmt.Fprintf(&b, "  verse %d · %s %s\n", i+1, step.Kind().Key(), stepWords(step))
	}
	return strings.TrimRight(b.String(), "\n")
}

// stepWords is what a step's own key holds, as written.
func stepWords(step librarium.Step) string {
	written := func(m librarium.AspectMap[string]) string {
		if v, ok := m.For(librarium.Fallback); ok && m.IsUniform() {
			return v
		}
		return "(per aspect)"
	}
	switch s := step.(type) {
	case *librarium.Sanctum:
		return s.Vessel
	case *librarium.Transcription:
		return s.Vessel
	case *librarium.Tether:
		return s.Name
	case *librarium.Incantation:
		return written(s.Command)
	case *librarium.Litany:
		return written(s.Scroll)
	case *librarium.VoxCast:
		return s.Tidings
	}
	return ""
}

func riteExample(r *librarium.Rite) string {
	ex := "  servitor invoke " + r.Name + " " + r.Aspects[0]
	if keys := r.InscriptionKeys(); len(keys) > 0 {
		ex += " --" + keys[0] + " <value>"
	}
	return ex + "\n  servitor augury " + r.Name
}
