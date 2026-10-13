package rituals

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/chronicle"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// fixture is a temporary Librarium, a place for vessels and a state home.
type fixture struct {
	t                *testing.T
	lib, data, state string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	fx := &fixture{t: t, lib: t.TempDir(), data: t.TempDir(), state: t.TempDir()}
	t.Setenv("XDG_STATE_HOME", fx.state)
	t.Setenv(librarium.EnvChronicle, "")
	return fx
}

// rite writes the scripture of the rite name; $DATA is the vessels' place.
func (fx *fixture) rite(name, scripture string) string {
	fx.t.Helper()
	return fx.write(fx.lib, "rites/"+name+".json", strings.ReplaceAll(scripture, "$DATA", fx.data))
}

// vessel writes content to rel among the vessels.
func (fx *fixture) vessel(rel, content string) string {
	fx.t.Helper()
	return fx.write(fx.data, rel, content)
}

func (fx *fixture) write(dir, rel, content string) string {
	fx.t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		fx.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		fx.t.Fatal(err)
	}
	return p
}

func (fx *fixture) read(p string) string {
	fx.t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		fx.t.Fatal(err)
	}
	return string(b)
}

// servitor opens the Librarium, ending invocations at a fixed moment.
func (fx *fixture) servitor() *Servitor {
	s := Open(fx.lib, librarium.Runes{})
	s.Now = func() time.Time { return time.Date(2026, 10, 10, 13, 30, 0, 0, time.UTC) }
	return s
}

func (fx *fixture) chronicled() []chronicle.Entry {
	fx.t.Helper()
	entries, err := chronicle.Read(filepath.Join(fx.state, "servitor", "chronicle.jsonl"))
	if err != nil {
		fx.t.Fatal(err)
	}
	return entries
}

func (fx *fixture) slate(rite string) *augury.Slate {
	fx.t.Helper()
	s, err := augury.ReadSlate(os.LookupEnv, rite)
	if err != nil {
		fx.t.Fatal(err)
	}
	return s
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

// mouse is a rite with a sanctum subject, an observance spoken after it
// changes and a decreed inscription.
const mouse = `{
  "pattern": "Mark II",
  "purpose": "Hide the cursor after inactivity",
  "aspects": ["on", "off"],
  "inscriptions": {
    "reason": {"purpose": "why the rite was invoked", "decrees": {"off": "restored"}}
  },
  "vox-cast": ["success"],
  "subjects": [
    {"name": "util", "designation": "sanctum", "vessel": "$DATA/util.kdl",
     "omens": {"on": "hide-after-inactive-ms 400", "off": ""}, "recessional": ["said"]}
  ],
  "observances": [
    {"name": "said", "designation": "incantation", "words": "echo {{aspect}}-$SERVITOR_INSCRIPTION_REASON >> $DATA/said"}
  ]
}`

// falling is mouse whose last observance falls with the death-mark 3 after
// telling "dying".
var falling = strings.Replace(strings.Replace(mouse, `"recessional": ["said"]`, `"recessional": ["said", "fall"]`, 1),
	`"observances": [`, `"observances": [
    {"name": "fall", "designation": "incantation", "words": "echo dying; exit 3"},`, 1)

// spirit is a rite of the aspects on and off whose one machine-spirit is
// read by auspex, the JSON of its auspex ("" for none), and set by words
// that change nothing.
func spirit(auspex string) string {
	if auspex != "" {
		auspex = `"auspex": ` + auspex + `, `
	}
	return `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
  {"name": "spirit", "designation": "machine-spirit", ` + auspex + `"enact": {"*": "true"}}]}`
}

// broken is a heretical rite: it declares no aspect.
const broken = `{"pattern": "Mark II", "aspects": [], "subjects": []}`

// herald records every proclamation.
type herald struct{ heard []invocation.Tidings }

func (h *herald) Proclaim(p invocation.Proclamation) { h.heard = append(h.heard, p.Tidings) }

// plainGlosses are plain words no denunciation may speak.
var plainGlosses = regexp.MustCompile(`(?i)\b(error|warning|switch|states?|metadata|config|invalid|required|` +
	`description|file|field|timeout|symlink|directory|fail(ed|s|ure)?|status)\b`)

func grimdark(t *testing.T, words string) {
	t.Helper()
	if m := plainGlosses.FindString(words); m != "" {
		t.Errorf("%q speaks the plain word %q", words, m)
	}
}
