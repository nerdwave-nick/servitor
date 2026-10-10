package cli

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/nerdwave-nick/servitor/internal/codex"
)

func TestExpound_ListsEveryTopic(t *testing.T) {
	e := newEnv(t)
	out := e.mustRun("expound")
	if out != codex.IndexText() {
		t.Fatalf("expound did not recite the index of the codex:\n%s", out)
	}
	for _, topic := range []string{"invoke", "foresee", "tether", "zeal", "vox", "placeholders", "aspect-maps",
		"omens", "standing", "verdicts", "chronicle", "vox-cast"} {
		if !strings.Contains(out, "\n  "+topic+" ") {
			t.Errorf("the index lacks %q:\n%s", topic, out)
		}
	}
}

func TestExpound_RecitesOnePassage(t *testing.T) {
	e := newEnv(t)
	out := e.mustRun("expound", "tether")
	p, _ := codex.Lookup("tether")
	if out != p.Recital() {
		t.Fatalf("expound tether:\n%s", out)
	}
	for _, want := range []string{"as written in the codex", "anchor", "Exempla:"} {
		if !strings.Contains(out, want) {
			t.Errorf("expound tether lacks %q", want)
		}
	}
	// A ritual's passage is followed by its invocation and runes, as the
	// tree of rituals knows them — for invoke, with the rites of the Librarium.
	out = e.mustRun("expound", "invoke")
	for _, want := range []string{"+++ invoke · as written in the codex +++", "Invocation:", "-f, --foresee",
		"-s, --silence", "mouse-autohide-toggle", "Universal Runes:", "--chronicle", "servitor expound <topic>"} {
		if !strings.Contains(out, want) {
			t.Errorf("expound invoke lacks %q:\n%s", want, out)
		}
	}
}

func TestExpound_HiddenNamesReciteWhatTheyStandFor(t *testing.T) {
	e := newEnv(t)
	for alias, topic := range map[string]string{"force": "zeal", "json": "binharic", "log": "chronicle", "help": "expound"} {
		if got, want := e.mustRun("expound", alias), e.mustRun("expound", topic); got != want {
			t.Errorf("expound %s does not recite the passage of %s:\n%s", alias, topic, got)
		}
	}
}

func TestExpound_UnknownTopic(t *testing.T) {
	e := newEnv(t)
	out, errOut, code := e.run("expound", "teher")
	if code != 1 || out != "" {
		t.Fatalf("expound teher: code %d, stdout %q", code, out)
	}
	for _, want := range []string{`no passage on "teher"`, "Perhaps you sought:", "tether", "servitor expound"} {
		if !strings.Contains(errOut, want) {
			t.Errorf("stderr lacks %q:\n%s", want, errOut)
		}
	}
	if _, errOut, code := e.run("expound", "tether", "anchor"); code != 1 || !strings.Contains(errOut, "one passage at a time") {
		t.Errorf("two topics: code %d, stderr %q", code, errOut)
	}
}

func TestExpound_TopicsComplete(t *testing.T) {
	e := newEnv(t)
	got, directive := e.complete("expound", "")
	if !slices.Equal(got, codex.Topics()) || directive != noFileComp {
		t.Fatalf("expound completes %v (%s), want every topic %v", got, directive, codex.Topics())
	}
	for _, hidden := range []string{"help", "json", "log", "force"} {
		if slices.Contains(got, hidden) {
			t.Errorf("completion offers the hidden name %q", hidden)
		}
	}
	if got, _ := e.complete("expound", "te"); !slices.Equal(got, []string{"tether"}) {
		t.Errorf("expound te completes %v", got)
	}
	if got, _ := e.complete("expound", "tether", ""); len(got) != 0 {
		t.Errorf("a second topic completes %v", got)
	}
	if got, _ := e.complete("help", "teth"); !slices.Equal(got, []string{"tether"}) {
		t.Errorf("help teth completes %v", got)
	}
}

// TestHelpAliases_LeadToExpound: help, -h and --help recite the codex; only a
// rite's own page is drawn from its scripture rather than from the codex.
func TestHelpAliases_LeadToExpound(t *testing.T) {
	e := newEnv(t)
	index := e.mustRun("expound")
	for _, args := range [][]string{{"--help"}, {"-h"}, {"help"}, {"expound", "--help"}} {
		want := index
		if args[0] == "expound" {
			want = e.mustRun("expound", "expound")
		}
		if got := e.mustRun(args...); got != want {
			t.Errorf("%v does not recite the codex:\n%s", args, got)
		}
	}
	for _, c := range []struct {
		args  []string
		topic string
	}{
		{[]string{"help", "tether"}, "tether"},
		{[]string{"help", "census"}, "census"},
		{[]string{"census", "--help"}, "census"},
		{[]string{"augury", "-h"}, "augury"},
		{[]string{"invoke", "--help"}, "invoke"},
		{[]string{"invoke"}, "invoke"},
		{[]string{"completion", "bash", "--help"}, "completion"},
		{[]string{"help", "completion", "fish"}, "completion"},
	} {
		if got, want := e.mustRun(c.args...), e.mustRun("expound", c.topic); got != want {
			t.Errorf("%v does not recite the passage of %s:\n%s", c.args, c.topic, got)
		}
	}
	page := e.mustRun("invoke", "mouse-autohide-toggle", "--help")
	if got := e.mustRun("help", "invoke", "mouse-autohide-toggle"); got != page || !strings.Contains(page, "Aspects: on, off") {
		t.Errorf("a rite's page:\n%s\nby help:\n%s", page, got)
	}
	if _, errOut, code := e.run("help", "nonesuch"); code != 1 || !strings.Contains(errOut, `no passage on "nonesuch"`) {
		t.Errorf("help nonesuch: %d %q", code, errOut)
	}
}

// TestRuneErrors_PointToTheCodex: a malformed rune names the passage to
// recite, never a help rune.
func TestRuneErrors_PointToTheCodex(t *testing.T) {
	e := newEnv(t)
	for _, c := range []struct {
		args []string
		code int
		hint string
	}{
		{[]string{"--bogus"}, 1, "'servitor expound'"},
		{[]string{"census", "--bogus"}, 1, "'servitor expound census'"},
		{[]string{"augury", "mouse-autohide-toggle", "--bogus"}, 2, "'servitor expound augury'"},
		{[]string{"invoke", "mouse-autohide-toggle", "on", "--bogus"}, 1, "'servitor expound invoke'"},
	} {
		_, errOut, code := e.run(c.args...)
		if code != c.code || !strings.Contains(errOut, "unknown rune") || !strings.Contains(errOut, c.hint) || strings.Contains(errOut, "--help") {
			t.Errorf("%v: code %d, stderr %q", c.args, code, errOut)
		}
	}
}

// TestCodex_EveryRitualAndRuneHasAPassage: the tree of rituals drives the
// codex. Every ritual and every rune the servitor offers has a passage under
// its own name and stands in the matching chapter of the index; the hidden
// aliases are offered nowhere. A rite's own runes are its inscriptions,
// expounded together under "inscriptions".
func TestCodex_EveryRitualAndRuneHasAPassage(t *testing.T) {
	e := newEnv(t)
	root := NewRootCmd([]string{"--librarium", e.cfgDir}, io.Discard, io.Discard)
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()
	var rituals, runes []string
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		rite := c.Parent() != nil && c.Parent().Name() == "invoke"
		if c.HasParent() && c.Parent() == root && c.IsAvailableCommand() {
			rituals = append(rituals, c.Name())
		}
		flags := func(f *pflag.Flag) {
			if !f.Hidden && !slices.Contains(runes, f.Name) {
				runes = append(runes, f.Name)
			}
		}
		if !rite {
			c.LocalFlags().VisitAll(flags)
		}
		c.InheritedFlags().VisitAll(flags)
		for _, sub := range c.Commands() {
			if sub.IsAvailableCommand() {
				walk(sub)
			}
		}
	}
	walk(root)
	for _, name := range slices.Concat(rituals, runes) {
		if p, ok := codex.Lookup(name); !ok || p.Topic != name {
			t.Errorf("%q has no passage of its own in the codex", name)
		}
	}
	slices.Sort(rituals)
	slices.Sort(runes)
	if want := slices.Sorted(slices.Values(codex.Rituals)); !slices.Equal(rituals, want) {
		t.Errorf("the tree offers the rituals %v, the codex names %v", rituals, want)
	}
	if want := slices.Sorted(slices.Values(codex.Runes)); !slices.Equal(runes, want) {
		t.Errorf("the tree offers the runes %v, the codex names %v", runes, want)
	}
	for _, alias := range []string{"help", "json", "log"} {
		if slices.Contains(rituals, alias) || slices.Contains(runes, alias) {
			t.Errorf("the hidden %q is offered", alias)
		}
	}
}

// TestExpound_SchemaRecitesTheSchemaAsWritten: expound --schema recites the
// schema of a rite or of the settings exactly as the repository keeps it.
func TestExpound_SchemaRecitesTheSchemaAsWritten(t *testing.T) {
	e := newEnv(t)
	for _, c := range []struct {
		args []string
		file string
	}{
		{[]string{"expound", "--schema"}, "rite.schema.json"},
		{[]string{"expound", "--schema", "rite"}, "rite.schema.json"},
		{[]string{"expound", "rite", "--schema"}, "rite.schema.json"},
		{[]string{"expound", "--schema", "settings"}, "settings.schema.json"},
	} {
		want, err := os.ReadFile(filepath.Join("..", "..", "schema", c.file))
		if err != nil {
			t.Fatal(err)
		}
		if got := e.mustRun(c.args...); got != string(want) {
			t.Errorf("%v does not recite %s as written:\n%s", c.args, c.file, got)
		}
	}
	out, errOut, code := e.run("expound", "--schema", "tether")
	if code != 1 || out != "" {
		t.Fatalf("expound --schema tether: code %d, stdout %q", code, out)
	}
	for _, want := range []string{`no schema of "tether"`, `"rite"`, `"settings"`} {
		if !strings.Contains(errOut, want) {
			t.Errorf("stderr lacks %q:\n%s", want, errOut)
		}
	}
	if got, directive := e.complete("expound", "--schema", ""); !slices.Equal(got, []string{"rite", "settings"}) || directive != noFileComp {
		t.Errorf("expound --schema completes %v (%s)", got, directive)
	}
	if got := e.mustRun("expound", "schema"); !strings.Contains(got, "+++ schema · as written in the codex +++") {
		t.Errorf("expound schema:\n%s", got)
	}
}
