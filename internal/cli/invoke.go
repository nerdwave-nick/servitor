package cli

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
)

// invokeRunes are the runes every invocation heeds.
type invokeRunes struct{ foresee, silence, binharic bool }

func (a *app) newInvokeCmd() *cobra.Command {
	runes := &invokeRunes{}
	cmd := &cobra.Command{
		Use:   "invoke <rite> <aspect> | <edict> [--<inscription> <value>...]",
		Short: "Invoke an aspect of a rite, or proclaim an edict: perform its liturgy",
		Args:  cobra.ArbitraryArgs,
		// Rites complete as rituals of their own; never fall back to files.
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return refused(a.unfitRite(args[0]))
		},
	}
	cmd.PersistentFlags().BoolVarP(&runes.foresee, "foresee", "f", false,
		"divine the outcome without touching anything, not even the auspex")
	cmd.PersistentFlags().BoolVarP(&runes.silence, "silence", "s", false,
		"perform the rite in reverent silence")
	const binharic = "speak the chronicled verdict in binharic for lesser machines, and narrate upon stderr"
	cmd.PersistentFlags().BoolVar(&runes.binharic, "binharic", false, binharic)
	cmd.PersistentFlags().BoolVar(&runes.binharic, "json", false, binharic)
	_ = cmd.PersistentFlags().MarkHidden("json")
	cmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return refused(fmt.Errorf("%s\n%s", runeError(err), liturgyHint(cmd)))
	})
	a.invokeCmd = cmd
	for _, name := range a.lib.Names() {
		if r := a.lib.Rites[name]; r != nil {
			cmd.AddCommand(a.newRiteCmd(r, runes))
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

func (a *app) newRiteCmd(r *librarium.Rite, runes *invokeRunes) *cobra.Command {
	if r.Edict {
		return a.newEdictCmd(r, runes)
	}
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
				return refused(fmt.Errorf("the rite demands exactly one aspect (%s), yet %d were offered",
					strings.Join(r.Aspects, ", "), len(args)))
			}
			if !r.HasAspect(args[0]) {
				return refused(unknownAspect(r, args[0]))
			}
			return nil
		},
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]cobra.Completion, cobra.ShellCompDirective) {
			return a.aspectCompletions(r), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.invoke(cmd, r, args[0], *runes)
		},
	}
	inscriptionRunes(cmd, r)
	return cmd
}

// newEdictCmd is the ritual of the edict r: it is proclaimed by its name
// alone, and any word after it is refused before anything is performed.
func (a *app) newEdictCmd(r *librarium.Rite, runes *invokeRunes) *cobra.Command {
	short := r.Purpose
	if short == "" {
		short = "Proclaim the edict " + r.Name
	}
	cmd := &cobra.Command{
		Use:     r.Name,
		Short:   short,
		Long:    riteLong(r),
		Example: riteExample(r),
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 {
				return refused(fmt.Errorf("the edict %q bears no aspect and is proclaimed by its name alone, yet %s "+
					"followed its name; not one verse of its liturgy is performed", r.Name, quoteAll(args)))
			}
			return nil
		},
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return a.invoke(cmd, r, "", *runes)
		},
	}
	inscriptionRunes(cmd, r)
	return cmd
}

// quoteAll quotes every word and joins them.
func quoteAll(words []string) string {
	quoted := make([]string, len(words))
	for i, w := range words {
		quoted[i] = strconv.Quote(w)
	}
	return strings.Join(quoted, " ")
}

// inscriptionRunes gives the ritual of r a rune for each of its
// inscriptions, completing the words it decrees.
func inscriptionRunes(cmd *cobra.Command, r *librarium.Rite) {
	for _, in := range r.Inscriptions {
		usage := in.Purpose
		if usage == "" {
			usage = "inscription " + in.Key
		}
		switch {
		case in.Mandatory && r.Edict:
			usage += " (mandatory, unless the edict decrees it)"
		case in.Mandatory:
			usage += " (mandatory, unless the aspect decrees it)"
		}
		cmd.Flags().Var(wordRune(new(string), "", "word"), in.Key, usage)
		_ = cmd.RegisterFlagCompletionFunc(in.Key, func(*cobra.Command, []string, string) ([]cobra.Completion, cobra.ShellCompDirective) {
			return decreeCompletions(in, r.Edict), cobra.ShellCompDirectiveNoFileComp
		})
	}
}

// invoke performs (or foresees) the rite r into aspect, or proclaims the
// edict r (aspect ""), heeding an interrupt from the terminal by halting
// and reverting the invocation. In binharic the verdict (or the
// forbiddance) is spoken upon stdout and the narration upon stderr.
//
// From here until Execute has spoken the lament, the servitor is
// steadfast: neither a hangup nor a reader gone from stdout or stderr cuts
// short the liturgy, its report or its death-mark.
func (a *app) invoke(cmd *cobra.Command, r *librarium.Rite, aspect string, ir invokeRunes) error {
	if ir.foresee && ir.binharic {
		return refused(errForeseenInBinharic)
	}
	if !ir.foresee {
		a.relent = rituals.Steadfast()
	}
	runes := map[string]string{}
	for _, in := range r.Inscriptions {
		if f := cmd.Flags().Lookup(in.Key); f != nil && f.Changed {
			runes[in.Key] = f.Value.String()
		}
	}
	s := a.servitor(cmd)
	p := rituals.Petition{Rite: r.Name, Aspect: aspect, Runes: runes, Foresee: ir.foresee}
	var told *narration
	if !ir.foresee {
		w := cmd.OutOrStdout()
		if ir.binharic {
			w = cmd.ErrOrStderr()
		}
		told = newNarration(w, s.Orders().Vox, ir.silence)
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
	if ir.foresee && res.Rite != nil {
		printForesight(cmd.OutOrStdout(), res)
	}
	var hs invocation.Heresies
	switch {
	case errors.As(err, &hs):
		return forbid(cmd.OutOrStdout(), r, aspect, hs, ir.binharic)
	case err != nil:
		return refused(err)
	case ir.foresee:
		return nil
	}
	return judged(cmd.OutOrStdout(), r, res, ir.binharic)
}

func unknownAspect(r *librarium.Rite, aspect string) error {
	return fmt.Errorf("the rite %q knows no aspect %q (known aspects: %s)", r.Name, aspect, strings.Join(r.Aspects, ", "))
}

func riteLong(r *librarium.Rite) string {
	var b strings.Builder
	if r.Purpose != "" {
		b.WriteString(r.Purpose + "\n\n")
	}
	if r.Edict {
		b.WriteString("Edict: proclaimed whole, bearing no aspect\n")
	} else {
		fmt.Fprintf(&b, "Aspects: %s\n", strings.Join(r.Aspects, ", "))
	}
	fmt.Fprintf(&b, "Recorded in: %s\n\nLiturgy:\n", r.Path)
	for i, step := range r.Liturgy {
		fmt.Fprintf(&b, "  verse %d · %s %s\n", i+1, step.Kind().Key(), stepWords(step))
	}
	return strings.TrimRight(b.String(), "\n")
}

// stepWords is what a step's own key holds, as written.
func stepWords(step librarium.Step) string {
	written := func(m librarium.AspectMap[string]) string {
		if m.IsUniform() {
			return m.Entries()[0].Value
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
	ex := "  servitor invoke " + r.Name
	if len(r.Aspects) > 0 {
		ex += " " + r.Aspects[0]
	}
	if keys := r.InscriptionKeys(); len(keys) > 0 {
		ex += " --" + keys[0] + " <value>"
	}
	return ex + "\n  servitor augury " + r.Name
}
