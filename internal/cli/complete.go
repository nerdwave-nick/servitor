package cli

import (
	"hash/fnv"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
)

// Completion answers at once: it reads omens but never awakens an auspex.

// riteCompletions lists every rite of the Librarium but those given, with
// its purpose.
func (a *app) riteCompletions(given ...string) []cobra.Completion {
	var out []cobra.Completion
	for _, name := range a.lib.Names() {
		if slices.Contains(given, name) {
			continue
		}
		desc := "heretical"
		if r := a.lib.Rites[name]; r != nil {
			desc = r.Purpose
			switch {
			case desc != "":
			case r.Edict:
				desc = edictLore(name)
			default:
				desc = strings.Join(r.Aspects, "|")
			}
		}
		out = append(out, cobra.CompletionWithDesc(name, desc))
	}
	return out
}

// lore is told of an edict that writes no purpose, in place of nothing.
var lore = []string{
	"an edict of unwritten purpose, proclaimed whole",
	"the Omnissiah alone knows its purpose",
	"its purpose is sealed; proclaim it and behold",
	"no purpose was inscribed, yet the edict endures",
	"a command of the Machine God, its reason unspoken",
}

// edictLore is the lore told of the edict name: the same words at every
// Tab, though edicts differ among themselves.
func edictLore(name string) string {
	h := fnv.New32a()
	h.Write([]byte(name))
	return lore[h.Sum32()%uint32(len(lore))]
}

// aspectCompletions lists the aspects of r, marking the one it stands in.
func (a *app) aspectCompletions(r *librarium.Rite) []cobra.Completion {
	reading, _ := (&rituals.Servitor{Librarium: a.lib}).Augur(r.Name, true)
	out := make([]cobra.Completion, 0, len(r.Aspects))
	for _, aspect := range r.Aspects {
		desc := ""
		if aspect == reading.Aspect {
			desc = "current aspect"
		}
		out = append(out, cobra.CompletionWithDesc(aspect, desc))
	}
	return out
}

// decreeCompletions lists the values decreed for an inscription, each with
// the aspects decreeing it; an edict decrees one value for itself alone.
func decreeCompletions(in librarium.Inscription, edict bool) []cobra.Completion {
	var values []string
	aspects := map[string][]string{}
	for _, e := range in.Decrees.Entries() {
		if e.Value == "" {
			continue
		}
		if aspects[e.Value] == nil {
			values = append(values, e.Value)
		}
		aspects[e.Value] = append(aspects[e.Value], e.Aspect)
	}
	out := make([]cobra.Completion, len(values))
	for i, v := range values {
		desc := "decreed for " + strings.Join(aspects[v], ", ")
		if edict {
			desc = "decreed by the edict"
		}
		out[i] = cobra.CompletionWithDesc(v, desc)
	}
	return out
}

// signs describe the keys an augury answers; an edict answers its
// standing alone.
var signs = map[string]string{
	"aspect":     "the aspect the rite stands in",
	"standing":   "how the rite stands",
	"desecrated": "whether other hands changed what the rite keeps",
	"former":     "the aspect its data-slate records, when desecrated",
}

func (a *app) completeAuguryArgs(_ *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return a.riteCompletions(), cobra.ShellCompDirectiveNoFileComp
	case 1:
		if _, known := a.lib.Scriptures[args[0]]; !known {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		r := a.lib.Rites[args[0]]
		var out []cobra.Completion
		for _, k := range rituals.SignsOf(r) {
			out = append(out, cobra.CompletionWithDesc(k, signs[k]))
		}
		if r != nil {
			for _, in := range r.Inscriptions {
				out = append(out, cobra.CompletionWithDesc(in.Key, in.Purpose))
			}
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}
