package cli

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nerdwave-nick/servitor/internal/codex"
	"github.com/nerdwave-nick/servitor/schema"
)

func (a *app) newExpoundCmd() *cobra.Command {
	var forms bool
	cmd := &cobra.Command{
		Use:   "expound [topic]",
		Short: "Recite a passage of the codex: the lore of every ritual, rune, step and key",
		Args:  oneTopic,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
			if forms {
				return completeSchemas(args, toComplete)
			}
			return completeTopics(cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case forms:
				return reciteSchema(cmd.OutOrStdout(), args)
			case len(args) == 0:
				return recite(cmd.OutOrStdout(), codex.IndexText())
			}
			return expound(cmd, args[0])
		},
	}
	cmd.Flags().BoolVar(&forms, "schema", false,
		"recite the schema of a rite's scripture, or with \"settings\" the schema of the settings")
	return cmd
}

// reciteSchema recites the schema named by args — a rite's when none is
// named — exactly as the repository keeps it.
func reciteSchema(w io.Writer, args []string) error {
	name := schema.Rite
	if len(args) > 0 {
		name = args[0]
	}
	data, ok := schema.For(name)
	if !ok {
		msg := fmt.Sprintf("the codex holds no schema of %q; it bears only the schemas of %s", name, quoted(schema.Names))
		return errors.New(msg + "\n\nRecite 'servitor expound schema' for their lore.")
	}
	_, err := w.Write(data)
	return err
}

// completeSchemas offers the names of the schemas.
func completeSchemas(args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var out []cobra.Completion
	for _, name := range schema.Names {
		if strings.HasPrefix(name, toComplete) {
			out = append(out, name)
		}
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func quoted(names []string) string {
	q := make([]string, len(names))
	for i, n := range names {
		q[i] = strconv.Quote(n)
	}
	return strings.Join(q, " and ")
}

// newHelpCmd is the hidden help ritual: it recites the codex like expound,
// and also follows a path of rituals (help completion fish).
func newHelpCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "help [topic]",
		Hidden:            true,
		ValidArgsFunction: completeTopics,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return recite(cmd.OutOrStdout(), codex.IndexText())
			}
			if _, ok := codex.Lookup(args[0]); ok && len(args) == 1 {
				return expound(cmd, args[0])
			}
			if c, rest, err := cmd.Root().Find(args); err == nil && len(rest) == 0 && c != cmd.Root() {
				c.HelpFunc()(c, nil)
				return nil
			}
			return unknownTopic(strings.Join(args, " "))
		},
	}
}

func oneTopic(_ *cobra.Command, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("the codex recites one passage at a time, yet %d topics were named", len(args))
	}
	return nil
}

// completeTopics offers every listed topic of the codex with its epigraph;
// the hidden names are never offered.
func completeTopics(_ *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var out []cobra.Completion
	for _, topic := range codex.Topics() {
		if strings.HasPrefix(topic, toComplete) {
			p, _ := codex.Lookup(topic)
			out = append(out, cobra.CompletionWithDesc(topic, p.Epigraph))
		}
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// expound recites the passage on topic. A ritual's passage is followed by
// its invocation and runes, as the tree of rituals knows them.
func expound(cmd *cobra.Command, topic string) error {
	p, ok := codex.Lookup(topic)
	if !ok {
		return unknownTopic(topic)
	}
	text := p.Recital()
	if r := ritual(cmd.Root(), p.Topic); r != nil {
		text += "\n" + r.UsageString()
	}
	return recite(cmd.OutOrStdout(), text)
}

func recite(w io.Writer, text string) error {
	_, err := io.WriteString(w, text)
	return err
}

// ritual finds the ritual of the servitor named name.
func ritual(root *cobra.Command, name string) *cobra.Command {
	if !slices.Contains(codex.Rituals, name) {
		return nil
	}
	for _, c := range root.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func unknownTopic(topic string) error {
	msg := fmt.Sprintf("the codex holds no passage on %q", topic)
	if s := codex.Suggest(topic); len(s) > 0 {
		msg += "\n\nPerhaps you sought:\n\t" + strings.Join(s, "\n\t")
	}
	return errors.New(msg + "\n\nRecite 'servitor expound' for the index of every passage.")
}

// lore answers the hidden help runes and cmd.Help(): the passage of the
// ritual that cmd belongs to, or the index for the servitor itself. A rite's
// own page is drawn from its scripture instead, by page.
func (a *app) lore(page func(*cobra.Command, []string)) func(*cobra.Command, []string) {
	return func(c *cobra.Command, args []string) {
		if c.HasParent() && c.Parent() == a.invokeCmd {
			page(c, args)
			return
		}
		var err error
		if topic := loreTopic(c); topic != "" {
			err = expound(c, topic)
		} else {
			err = recite(c.OutOrStdout(), codex.IndexText())
		}
		if err != nil {
			c.PrintErrln("servitor ✠ " + err.Error())
		}
	}
}

// loreTopic is the ritual whose passage tells of c: c itself or the ritual
// it lies beneath; "" for the servitor itself.
func loreTopic(c *cobra.Command) string {
	for ; c.HasParent(); c = c.Parent() {
		if !c.Parent().HasParent() && slices.Contains(codex.Rituals, c.Name()) {
			return c.Name()
		}
	}
	return ""
}

// liturgyHint tells where the proper liturgy of cmd is written.
func liturgyHint(cmd *cobra.Command) string {
	recital := "servitor expound"
	if topic := loreTopic(cmd); topic != "" {
		recital += " " + topic
	}
	return "Recite '" + recital + "' for the proper liturgy"
}
