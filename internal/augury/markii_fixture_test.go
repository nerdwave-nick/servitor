package augury

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// stateRite writes the state rite "gaming" of pattern Mark II with the
// given aspects, further keys of the rite (e.g. `"tongue": "sh",`) and
// subjects ($DATA is the place of the machine's state files) into the
// Librarium and loads it.
func (fx fixture) stateRite(t *testing.T, aspects, keys, subjects string) *librarium.Rite {
	t.Helper()
	scripture := fmt.Sprintf(`{
  "pattern": "Mark II",
  "aspects": [%s],%s
  "inscriptions": {"place": {"purpose": "where"}},
  "subjects": [%s]
}`, aspects, keys, subjects)
	return fx.load(t, "gaming", scripture)
}

// load writes scripture as the rite name into the Librarium and loads it.
func (fx fixture) load(t *testing.T, name, scripture string) *librarium.Rite {
	t.Helper()
	path := filepath.Join(fx.lib, "rites", name+".json")
	write(t, path, strings.ReplaceAll(scripture, "$DATA", fx.data))
	r, findings := librarium.LoadFile(path)
	if findings.Heretical() {
		t.Fatalf("rite is heretical:\n%v", findings)
	}
	return r
}

// spirit is a machine-spirit subject named name, with its omens (empty for
// none) and auspex, set by words that never matter to an augury.
func spirit(name, omens, auspex string) string {
	if omens != "" {
		omens = `"omens": ` + omens + `, `
	}
	return fmt.Sprintf(`{"name": %q, "designation": "machine-spirit", %s"auspex": %s, "enact": {"*": "true"}}`,
		name, omens, auspex)
}

// shows sets what the machine shows of the state file name.
func (fx fixture) shows(t *testing.T, name, content string) {
	t.Helper()
	write(t, filepath.Join(fx.data, name), content)
}

// readings renders the omens of a state rite compactly, in written order:
// "<subject>=<aspect>|<aspect>…", with "(<omen>)" when an omen was read.
func readings(a Augury) string {
	var parts []string
	for _, o := range a.Omens {
		part := o.Subject + "=" + strings.Join(o.Aspects, "|")
		if o.Read != "" {
			part += "(" + o.Read + ")"
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, " ")
}

// judged expects aspect, standing and fitting aspects of a.
func judged(t *testing.T, a Augury, aspect string, s Standing, fitting ...string) {
	t.Helper()
	if fitting == nil {
		fitting = []string{}
	}
	if a.Aspect != aspect || a.Standing != s || !slices.Equal(a.Fitting, fitting) {
		t.Errorf("aspect %q standing %q fitting %q, want %q %q %q (omens %s)", a.Aspect, a.Standing, a.Fitting,
			aspect, s, fitting, readings(a))
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
