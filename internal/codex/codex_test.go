package codex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/schema"
)

// keysOfTheLibrarium are every key a rite or the settings may hold and every
// step kind, as the librarium itself knows them.
func keysOfTheLibrarium() []string {
	keys := slices.Concat(librarium.RiteKeys, librarium.InscriptionKeys, librarium.AuspexKeys,
		librarium.TomeKeys, librarium.SettingsKeys)
	for _, k := range librarium.Kinds {
		keys = append(keys, librarium.StepKeys(k)...)
	}
	return keys
}

func listed() []string {
	var names []string
	for _, ch := range Index() {
		for _, p := range ch.Passages {
			names = append(names, p.Topic)
		}
	}
	return names
}

// TestEveryKeyAndStepHasAPassage: the codex is driven by the librarium; a key
// or step kind without its passage fails here.
func TestEveryKeyAndStepHasAPassage(t *testing.T) {
	for _, key := range keysOfTheLibrarium() {
		p, ok := Lookup(key)
		if !ok {
			t.Errorf("the key %q has no passage in the codex", key)
			continue
		}
		if _, alias := librarium.Aliases[key]; alias {
			continue
		}
		topic := key
		if t, ok := keyTopics[key]; ok {
			topic = t
		}
		if p.Topic != topic {
			t.Errorf("the key %q recites the passage of %q", key, p.Topic)
		}
		if !slices.Contains(listed(), topic) {
			t.Errorf("the key %q is missing from the index of the codex", key)
		}
	}
	for _, k := range librarium.Kinds {
		if !slices.Contains(chapter(t, StepsChapter), k.Key()) {
			t.Errorf("the step %q is missing from the chapter of steps", k.Key())
		}
	}
	for _, k := range librarium.SettingsKeys {
		if t, ok := keyTopics[k]; ok {
			k = t
		}
		if !slices.Contains(chapter(t, SettingsChapter), k) {
			t.Errorf("the settings key %q is missing from the chapter of settings", k)
		}
	}
}

func chapter(t *testing.T, title string) []string {
	t.Helper()
	for _, ch := range Index() {
		if ch.Title == title {
			var names []string
			for _, p := range ch.Passages {
				names = append(names, p.Topic)
			}
			return names
		}
	}
	t.Fatalf("the codex has no chapter %q", title)
	return nil
}

// TestEveryListedNameHasAPassage: no chapter names a topic the codex cannot
// recite, and every passage is listed somewhere.
func TestEveryListedNameHasAPassage(t *testing.T) {
	for _, name := range slices.Concat(Rituals, Runes) {
		if _, ok := passages[name]; !ok {
			t.Errorf("%q is named by the codex but has no passage", name)
		}
	}
	all := listed()
	for name := range passages {
		if !slices.Contains(all, name) {
			t.Errorf("the passage %q is listed in no chapter", name)
		}
	}
	if got := Topics(); len(got) != len(passages) {
		t.Errorf("Topics lists %d names for %d passages: %v", len(got), len(passages), got)
	}
}

// TestSchemaKey_IsExpoundedAsSchema: "$schema" is expounded under the name
// of its rune, which every terminal can speak, and stands in the chapters
// of the keys of a rite and of the settings as that.
func TestSchemaKey_IsExpoundedAsSchema(t *testing.T) {
	p, ok := Lookup(librarium.SchemaKey)
	if !ok || p.Topic != "schema" {
		t.Fatalf("%q recites %q (found %v), want schema", librarium.SchemaKey, p.Topic, ok)
	}
	for _, title := range []string{RunesChapter, RiteChapter, SettingsChapter} {
		if !slices.Contains(chapter(t, title), "schema") {
			t.Errorf("the chapter %q lacks schema", title)
		}
	}
	if slices.Contains(listed(), librarium.SchemaKey) || slices.Contains(Topics(), librarium.SchemaKey) {
		t.Errorf("the codex lists %q under its own name", librarium.SchemaKey)
	}
	for _, want := range []string{`"$schema"`, "--schema", schema.URL(schema.Rite), schema.URL(schema.Settings)} {
		if !strings.Contains(p.Body, want) {
			t.Errorf("the passage on the schema lacks %q", want)
		}
	}
}

// TestSchemas_SpeakNoPlainGloss: the lore the schemas lend the faithful's
// editors is grimdark, like the codex.
func TestSchemas_SpeakNoPlainGloss(t *testing.T) {
	for _, name := range schema.Names {
		data, _ := schema.For(name)
		var tree any
		if err := json.Unmarshal(data, &tree); err != nil {
			t.Fatalf("schema %s: %v", name, err)
		}
		texts := lore(tree)
		if len(texts) < 10 {
			t.Fatalf("schema %s holds too little lore: %v", name, texts)
		}
		for _, text := range texts {
			if w := plainWord(text); w != "" {
				t.Errorf("schema %s speaks the plain word %q: %s", name, w, text)
			}
		}
	}
}

// lore gathers every title and description beneath v.
func lore(v any) []string {
	var out []string
	switch v := v.(type) {
	case map[string]any:
		for k, child := range v {
			if s, ok := child.(string); ok && (k == "description" || k == "title") {
				out = append(out, s)
			}
			out = append(out, lore(child)...)
		}
	case []any:
		for _, child := range v {
			out = append(out, lore(child)...)
		}
	}
	return out
}

// TestHiddenNames_ReciteButAreNeverListed: the deliberate aliases lead to the
// passage they stand for, yet no index or topic list names them.
func TestHiddenNames_ReciteButAreNeverListed(t *testing.T) {
	for alias, topic := range map[string]string{"help": "expound", "json": "binharic", "log": "chronicle", "force": "zeal"} {
		p, ok := Lookup(alias)
		if !ok || p.Topic != topic {
			t.Errorf("%q recites %q (found %v), want %q", alias, p.Topic, ok, topic)
		}
		if slices.Contains(listed(), alias) || slices.Contains(Topics(), alias) {
			t.Errorf("the hidden name %q is listed by the codex", alias)
		}
		if strings.Contains(IndexText(), " "+alias+" ") {
			t.Errorf("the index text names the hidden %q", alias)
		}
	}
}

// plainWords are glosses from the plain column of the lookup in AGENTS.md
// that the codex must never speak. Ordinary English that the plain column
// merely shares ("list", "file", "step", "command") is not denied.
var plainWords = regexp.MustCompile(`(?i)\b(` + strings.Join([]string{
	`configs?`, `configuration`, `switch(es)?`, `profiles?`, `states?`, `descriptions?`,
	`meta(data)?`, `managed`, `blocks?`, `symlinks?`, `shells?`, `scripts?`, `timeouts?`,
	`rollbacks?`, `roll(ed)? back`, `notifications?`, `status(es)?`, `drift`, `validat(e|ion)`,
	`tui`, `documentation`, `dry[ -]run`, `quiet`, `timestamps?`, `file mode`, `target file`,
	`step index`, `comment (prefix|suffix)`, `file content`, `force`, `flags?`, `subcommands?`,
	`arguments?`, `errors?`, `warnings?`, `verify`, `logs?`, `results?`, `help`,
}, "|") + `)\b`)

// names are the machine's own words — paths (~/.config/niri/config.kdl) and
// hyphenated command names (load-config-file) — not prose; compounds that
// are themselves glosses are denied before names are set aside.
var (
	names          = regexp.MustCompile(`\S*/\S*|\w+(?:-\w+)+`)
	plainCompounds = regexp.MustCompile(`(?i)\bdry-run\b|\broll-?back\b|--json\b`)
)

func plainWord(line string) string {
	if w := plainCompounds.FindString(line); w != "" {
		return w
	}
	return plainWords.FindString(names.ReplaceAllString(line, " "))
}

// TestPassages_SpeakNoPlainGloss: the codex explains in grimdark alone.
func TestPassages_SpeakNoPlainGloss(t *testing.T) {
	for name, p := range passages {
		for i, line := range strings.Split(p.Epigraph+"\n"+p.Body, "\n") {
			if w := plainWord(line); w != "" {
				t.Errorf("passage %q, line %d speaks the plain word %q: %s", name, i+1, w, line)
			}
		}
	}
	if w := plainWord(IndexText()); w != "" {
		t.Errorf("the index speaks the plain word %q", w)
	}
}

// TestScripturesOfTheFaithful_SpeakNoPlainGloss: the README, the example
// Librarium the faithful read first and the litanies of the vox-casts are
// grimdark, like the codex.
func TestScripturesOfTheFaithful_SpeakNoPlainGloss(t *testing.T) {
	paths := []string{"../../README.md", "../../examples/rites/scripts/recite-hooks",
		"../vox/success.txt", "../vox/progress.txt", "../vox/failure.txt"}
	rites, _ := filepath.Glob("../../examples/rites/*.json")
	if len(rites) < 2 {
		t.Fatalf("too few example rites: %v", rites)
	}
	for _, path := range append(paths, rites...) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			if w := plainWord(line); w != "" {
				t.Errorf("%s:%d speaks the plain word %q: %s", path, i+1, w, line)
			}
		}
	}
}

func TestPlainWords_DenyGlossesButNotOrdinaryWords(t *testing.T) {
	for _, plain := range []string{"the config", "a symlink", "dry run", "Shell", "status", "the --json rune", "force", "a dry-run"} {
		if plainWord(plain) == "" {
			t.Errorf("%q passes the guard", plain)
		}
	}
	for _, fine := range []string{"scripture", "statement", "a list of lines", "the step", "SERVITOR_STATE", "$XDG_STATE_HOME", "binharic",
		"~/.config/niri/config.kdl", "$XDG_CONFIG_HOME", "niri msg action load-config-file"} {
		if plainWord(fine) != "" {
			t.Errorf("%q is denied, yet ordinary", fine)
		}
	}
}

var exemplum = regexp.MustCompile(`(?m)^Exempl(um|a):\n( {2,}\S.*\n?)+`)

// TestPassages_AreVerboseAndBearAnExample: every passage has its epigraph,
// several lines of lore and at least one indented example.
func TestPassages_AreVerboseAndBearAnExample(t *testing.T) {
	if len(passages) == 0 {
		t.Fatal("the codex holds no passage")
	}
	for name, p := range passages {
		if p.Epigraph == "" || strings.HasSuffix(p.Epigraph, ".") {
			t.Errorf("passage %q: the epigraph must be one line without a closing period: %q", name, p.Epigraph)
		}
		if n := strings.Count(p.Body, "\n"); n < 6 {
			t.Errorf("passage %q is not verbose: %d lines", name, n)
		}
		if !exemplum.MatchString(p.Body) {
			t.Errorf("passage %q bears no example", name)
		}
	}
}

func TestRecital_AsWrittenInTheCodex(t *testing.T) {
	p, ok := Lookup("tether")
	if !ok {
		t.Fatal("no passage on the tether")
	}
	got := p.Recital()
	for _, want := range []string{"tether", "as written in the codex", p.Epigraph, "anchor", "Exempl"} {
		if !strings.Contains(got, want) {
			t.Errorf("the recital lacks %q:\n%s", want, got)
		}
	}
}

func TestIndexText_NamesEveryTopic(t *testing.T) {
	text := IndexText()
	for _, name := range Topics() {
		if !regexp.MustCompile(`(?m)^  ` + regexp.QuoteMeta(name) + ` `).MatchString(text) {
			t.Errorf("the index text lacks %q", name)
		}
	}
	if !strings.Contains(text, "servitor expound <topic>") {
		t.Errorf("the index does not teach how to recite a passage:\n%s", text)
	}
}

func TestSuggest_NamesNearTopics(t *testing.T) {
	for in, want := range map[string]string{"teher": "tether", "litny": "litany", "insc": "inscriptions", "forsee": "foresee"} {
		if got := Suggest(in); !slices.Contains(got, want) {
			t.Errorf("Suggest(%q) = %v, want %q among them", in, got, want)
		}
	}
	if got := Suggest("zzzzzz"); len(got) != 0 {
		t.Errorf("Suggest(zzzzzz) = %v", got)
	}
	if got := Suggest("forc"); slices.Contains(got, "force") {
		t.Errorf("Suggest offers a hidden name: %v", got)
	}
}
