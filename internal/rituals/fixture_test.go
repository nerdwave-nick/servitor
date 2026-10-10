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

// mouse is a rite with a sanctum, an incantation and a decreed inscription.
const mouse = `{
  "pattern": "Mark I",
  "purpose": "Hide the cursor after inactivity",
  "aspects": ["on", "off"],
  "inscriptions": {
    "reason": {"purpose": "why the rite was invoked", "decrees": {"off": "restored"}}
  },
  "liturgy": [
    {"sanctum": "$DATA/util.kdl", "scripture": {"on": "hide-after-inactive-ms 400", "off": ""}},
    {"incantation": "echo {{aspect}}-$SERVITOR_INSCRIPTION_REASON >> $DATA/said"},
    {"vox-cast": "success"}
  ]
}`

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
