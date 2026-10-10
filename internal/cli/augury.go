package cli

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nerdwave-nick/servitor/internal/rituals"
)

func (a *app) newAuguryCmd() *cobra.Command {
	var is string
	cmd := &cobra.Command{
		Use:   "augury <rite> [key]",
		Short: "Perform an augury: read the aspect, standing and inscriptions of a rite",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) < 1 || len(args) > 2 {
				return &ExitError{Code: 2, Err: fmt.Errorf("the augury demands one rite, and at most one key, "+
					"yet %d words were offered", len(args))}
			}
			return nil
		},
		ValidArgsFunction: a.completeAuguryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := augur(cmd, a.servitor(cmd), args, is); err != nil {
				var ee *ExitError
				if errors.As(err, &ee) {
					return err
				}
				return &ExitError{Code: 2, Err: err}
			}
			return nil
		},
	}
	cmd.Flags().Var(wordRune(&is, "", "aspect"), "is", "exit 0 if the rite stands in this aspect, 1 otherwise")
	_ = cmd.RegisterFlagCompletionFunc("is", func(_ *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
		if len(args) > 0 {
			if r := a.lib.Rites[args[0]]; r != nil {
				return a.aspectCompletions(r), cobra.ShellCompDirectiveNoFileComp
			}
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return &ExitError{Code: 2, Err: fmt.Errorf("%s\n%s", runeError(err), liturgyHint(cmd))}
	})
	return cmd
}

// augur reads the rite args[0] and speaks its JSON, the value of the key
// args[1], or — for is — answers by exit code alone.
func augur(cmd *cobra.Command, s *rituals.Servitor, args []string, is string) error {
	name := args[0]
	if is != "" {
		if len(args) > 1 {
			return fmt.Errorf("the rune --is asks after an aspect alone and admits no key %q", args[1])
		}
		r, err := s.Rite(name)
		if err != nil {
			return err
		}
		if !r.HasAspect(is) {
			return unknownAspect(r, is)
		}
	}
	reading, err := s.Augur(name, false)
	if err != nil {
		return err
	}
	if reading.Lament != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "servitor ✠ "+reading.Lament.Error())
	}
	out := cmd.OutOrStdout()
	switch {
	case is != "":
		if reading.Aspect != is {
			return &ExitError{Code: 1}
		}
		return nil
	case len(args) == 2:
		v, ok := reading.Value(args[1])
		if !ok {
			return fmt.Errorf("the rite %q bears no key %q (keys: %s)", name, args[1], strings.Join(keysOf(reading), ", "))
		}
		fmt.Fprintln(out, v)
		return nil
	}
	return writeJSON(out, reading.Augury)
}

func keysOf(r rituals.Reading) []string {
	if r.Rite != nil {
		return rituals.Keys(r.Rite)
	}
	return rituals.Signs
}

func writeJSON(w io.Writer, v any) error {
	if err := json.MarshalWrite(w, v, jsontext.WithIndent("  "), json.Deterministic(true)); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}
