package invocation

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// inscribedRite is a rite of aspects on and off whose inscription "mode" is
// mandatory with a decree only for on, beside the free "reason" and
// "colour" (decreed "*": "grey").
func inscribedRite(t *testing.T) *librarium.Rite {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rites", "mouse.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	scripture := `{
  "pattern": "Mark I",
  "aspects": ["on", "off"],
  "inscriptions": {
    "reason": {"purpose": "why"},
    "mode":   {"mandatory": true, "decrees": {"on": "auto-hide"}},
    "colour": {"decrees": {"*": "grey"}}
  },
  "liturgy": [{"vox-cast": "success"}]
}`
	if err := os.WriteFile(path, []byte(scripture), 0o644); err != nil {
		t.Fatal(err)
	}
	r, findings := librarium.LoadFile(path)
	if findings.Heretical() {
		t.Fatalf("rite is heretical:\n%v", findings)
	}
	return r
}

func TestResolveInscriptions_RuneOverDecree(t *testing.T) {
	r := inscribedRite(t)
	for _, tc := range []struct {
		name   string
		aspect string
		runes  map[string]string
		want   map[string]string
	}{
		{"decrees serve where no rune is given", "on", nil,
			map[string]string{"mode": "auto-hide", "colour": "grey"}},
		{"a rune outranks the decree", "on", map[string]string{"mode": "manual", "colour": "red"},
			map[string]string{"mode": "manual", "colour": "red"}},
		{"a rune without decree is kept", "off", map[string]string{"mode": "x", "reason": "gaming remnant"},
			map[string]string{"mode": "x", "reason": "gaming remnant", "colour": "grey"}},
		{"an empty rune clears the decree", "on", map[string]string{"colour": ""},
			map[string]string{"mode": "auto-hide"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveInscriptions(r, tc.aspect, tc.runes)
			if err != nil {
				t.Fatalf("unexpected heresy: %v", err)
			}
			if !maps.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolveInscriptions_MandatoryWithoutValueIsHeresy(t *testing.T) {
	r := inscribedRite(t)
	for _, tc := range []struct {
		name   string
		aspect string
		runes  map[string]string
	}{
		{"no decree for the aspect and no rune", "off", nil},
		{"the decree cleared by an empty rune", "on", map[string]string{"mode": ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ResolveInscriptions(r, tc.aspect, tc.runes)
			var hs Heresies
			if !errors.As(err, &hs) || len(hs) != 1 {
				t.Fatalf("expected one heresy, got %v", err)
			}
			if !strings.Contains(hs[0].Message, `"mode"`) || hs[0].Line != 6 {
				t.Errorf("heresy names the wrong inscription or place: %v", hs[0])
			}
			grimdark(t, hs[0].Message)
		})
	}
}

func TestResolveInscriptions_UndeclaredRuneIsHeresy(t *testing.T) {
	r := inscribedRite(t)
	_, err := ResolveInscriptions(r, "on", map[string]string{"colour": "red", "hue": "blue"})
	var hs Heresies
	if !errors.As(err, &hs) || len(hs) != 1 || !strings.Contains(hs[0].Message, `"hue"`) {
		t.Fatalf("expected one heresy naming \"hue\", got %v", err)
	}
	grimdark(t, hs[0].Message)
}

func TestPrepare_RefusesAMandatoryInscriptionWithoutValue(t *testing.T) {
	r := inscribedRite(t)
	hs := refused(t, r, Options{Aspect: "off", Inscriptions: map[string]string{"reason": "x"}}, `"mode"`)
	if hs[0].Verse != 0 {
		t.Errorf("the heresy belongs to the invocation as a whole, got verse %d", hs[0].Verse)
	}
	if _, err := Prepare(r, Options{Aspect: "off", Inscriptions: map[string]string{"mode": "x"}}); err != nil {
		t.Errorf("a mandatory inscription with value was refused: %v", err)
	}
}
