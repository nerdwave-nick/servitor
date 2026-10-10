// Package codex holds the servitor's book of instruction: one verbose
// passage, with at least one example, for every ritual, rune, step and key
// the servitor knows, and for the further lore beneath them. Passages are
// embedded from passages/<topic>.txt; the first line of each is its
// epigraph, the rest its body.
//
// The chapters of the steps and keys are drawn from the librarium itself,
// so a key the librarium learns appears in the index at once (and fails the
// tests until its passage is written). The rituals and runes are named here;
// the command line's tests hold them against its tree of rituals.
//
// The deliberate aliases (help, json, log and the librarium's key aliases)
// recite the passage they stand for but are never listed.
package codex

import (
	"embed"
	"io/fs"
	"maps"
	"slices"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

//go:embed passages/*.txt
var scrolls embed.FS

// Passage is one passage of the codex.
type Passage struct {
	Topic    string // the name it is recited by
	Epigraph string // one line that sums it up
	Body     string // the lore itself, examples included
}

// Recital is the passage as the servitor recites it.
func (p Passage) Recital() string {
	return "+++ " + p.Topic + " · as written in the codex +++\n\n" + p.Epigraph + "\n\n" + p.Body + "\n"
}

// Titles of the chapters of the index.
const (
	RitualsChapter  = "Rituals"
	RunesChapter    = "Runes"
	StepsChapter    = "Steps of the liturgy"
	RiteChapter     = "Keys of a rite's scripture"
	SettingsChapter = "Keys of the settings"
	FurtherChapter  = "Further lore"
)

var (
	// Rituals are the servitor's rituals, in the order of the codex.
	Rituals = []string{"invoke", "augury", "census", "inquisition", "cogitator", "expound", "completion"}
	// Runes are the servitor's own runes (without their dashes), in the
	// order of the codex. A rite's inscription runes are its own.
	Runes = []string{"librarium", "chronicle", "foresee", "silence", "is", "binharic", "spare-vessels", "version"}
)

// hidden maps every deliberate alias to the topic it stands for.
var hidden = func() map[string]string {
	m := map[string]string{"help": "expound", "json": "binharic", "log": "chronicle"}
	for alias, key := range librarium.Aliases {
		m[alias] = key
	}
	return m
}()

var passages = func() map[string]Passage {
	m := map[string]Passage{}
	entries, _ := fs.ReadDir(scrolls, "passages")
	for _, e := range entries {
		data, err := scrolls.ReadFile("passages/" + e.Name())
		if err != nil {
			continue
		}
		topic := strings.TrimSuffix(e.Name(), ".txt")
		epigraph, body, _ := strings.Cut(string(data), "\n")
		m[topic] = Passage{Topic: topic, Epigraph: strings.TrimSpace(epigraph), Body: strings.Trim(body, "\n")}
	}
	return m
}()

// Lookup finds the passage recited by name; a hidden alias finds the
// passage of the name it stands for.
func Lookup(name string) (Passage, bool) {
	if topic, ok := hidden[name]; ok {
		name = topic
	}
	p, ok := passages[name]
	return p, ok
}

// Chapter is one chapter of the index.
type Chapter struct {
	Title    string
	Passages []Passage
}

// Index returns the chapters of the codex. A topic may stand in several
// chapters (patience is a key of rites and of the settings); every passage
// stands in at least one.
func Index() []Chapter {
	var kinds []string
	for _, k := range librarium.Kinds {
		kinds = append(kinds, k.Key())
	}
	named := [][2]any{
		{RitualsChapter, Rituals},
		{RunesChapter, Runes},
		{StepsChapter, kinds},
		{RiteChapter, riteKeys(kinds)},
		{SettingsChapter, librarium.SettingsKeys},
	}
	var chapters []Chapter
	seen := map[string]bool{}
	for _, n := range named {
		ch := Chapter{Title: n[0].(string)}
		for _, name := range n[1].([]string) {
			if p, ok := passages[name]; ok {
				ch.Passages = append(ch.Passages, p)
				seen[name] = true
			}
		}
		chapters = append(chapters, ch)
	}
	further := Chapter{Title: FurtherChapter}
	for _, name := range slices.Sorted(maps.Keys(passages)) {
		if !seen[name] {
			further.Passages = append(further.Passages, passages[name])
		}
	}
	return append(chapters, further)
}

// riteKeys are the keys a rite's scripture may hold beneath and beside its
// steps' own keys, in the order of the codex, without aliases.
func riteKeys(kinds []string) []string {
	keys := slices.Concat(librarium.RiteKeys, librarium.InscriptionKeys, librarium.AuspexKeys, librarium.TomeKeys)
	for _, k := range librarium.Kinds {
		keys = append(keys, librarium.StepKeys(k)...)
	}
	var out []string
	for _, k := range keys {
		_, alias := hidden[k]
		if !alias && !slices.Contains(kinds, k) && !slices.Contains(out, k) {
			out = append(out, k)
		}
	}
	return out
}

// Topics lists every topic of the index once, in the order of the index.
func Topics() []string {
	var out []string
	for _, ch := range Index() {
		for _, p := range ch.Passages {
			if !slices.Contains(out, p.Topic) {
				out = append(out, p.Topic)
			}
		}
	}
	return out
}
