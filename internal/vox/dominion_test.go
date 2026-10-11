package vox

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// TestDominionTemplates_AsSpecified: the awaiting of dominion is told in
// the words decreed for it, a rite's and an edict's apart, illuminated by
// {{rite.name}} alone, for the same notice serves every dominion awaited.
func TestDominionTemplates_AsSpecified(t *testing.T) {
	rites := []string{
		"Another invocation holds dominion over what {{rite.name}} must touch. The servitor waits its turn.",
		"Another hand holds dominion where {{rite.name}} must labour; the servitor kneels and waits for it to be relinquished.",
		"{{rite.name}} must wait: dominion lies with another invocation. Stillness is the first devotion.",
	}
	edicts := []string{
		"The edict {{rite.name}} is already being proclaimed; the servitor waits for its dominion to be relinquished.",
		"Another invocation holds dominion over the edict {{rite.name}}. The servitor waits its turn.",
		"Another hand proclaims the edict {{rite.name}}; the servitor kneels and waits for its end.",
	}
	if got := templates[invocation.Dominion]; !slices.Equal(got, rites) {
		t.Errorf("the rite's dominion templates are %q", got)
	}
	if got := edictTemplates[invocation.Dominion]; !slices.Equal(got, edicts) {
		t.Errorf("the edict's dominion templates are %q", got)
	}
	marks := regexp.MustCompile(`\{\{[^}]*\}\}`)
	for _, list := range [][]string{templates[invocation.Dominion], edictTemplates[invocation.Dominion]} {
		for _, tmpl := range list {
			grimdark(t, tmpl)
			for _, m := range marks.FindAllString(tmpl, -1) {
				if m != "{{rite.name}}" {
					t.Errorf("dominion template %q speaks %s", tmpl, m)
				}
			}
			if hs := (placeholder.Scope{Mode: placeholder.VoxCastMode, Edict: true}).Check(tmpl); len(hs) > 0 {
				t.Errorf("dominion template %q is heretical: %v", tmpl, hs)
			}
		}
	}
	for _, tmpl := range edictTemplates[invocation.Dominion] {
		if !strings.Contains(strings.ToLower(tmpl), "the edict {{rite.name}}") {
			t.Errorf("edict dominion template %q names not the edict", tmpl)
		}
	}
}

func TestCompose_DominionNamesTheRiteAlone(t *testing.T) {
	for i := range templates[invocation.Dominion] {
		v := placeholder.Values{Rite: "theme", Aspect: "porpl"}
		m := Compose(invocation.Proclamation{Tidings: invocation.Dominion, Values: v}, func(int) int { return i })
		if m.Tidings != invocation.Dominion || m.Summary != "theme → porpl" || m.Body != m.Flavour ||
			!strings.Contains(m.Flavour, "theme") || strings.ContainsAny(m.Flavour, "«»{") {
			t.Errorf("template %d composed %+v", i, m)
		}
	}
	for i := range edictTemplates[invocation.Dominion] {
		v := placeholder.Values{Rite: "screenshot"}
		m := Compose(invocation.Proclamation{Tidings: invocation.Dominion, Edict: true, Values: v}, func(int) int { return i })
		if m.Summary != "screenshot" || m.Body != m.Flavour || !strings.Contains(strings.ToLower(m.Flavour), "the edict screenshot") {
			t.Errorf("edict template %d composed %+v", i, m)
		}
	}
}

// TestHerald_DominionLingersUntilReplaced: the awaiting never expires while
// the invocation waits; what follows replaces it in place, and an
// invocation that ends without a vox-cast lets it expire.
func TestHerald_DominionLingersUntilReplaced(t *testing.T) {
	awaiting := invocation.Proclamation{Tidings: invocation.Dominion, Values: placeholder.Values{Rite: "theme", Aspect: "porpl"}}
	t.Run("replaced", func(t *testing.T) {
		s := newStub(t)
		h := quiet(Config{})
		h.Proclaim(awaiting)
		h.Proclaim(success())
		_ = h.Close()
		calls := s.calls(t)
		if len(calls) != 2 {
			t.Fatalf("calls %q", calls)
		}
		if v, _ := flag(calls[0], "-t"); v != "0" {
			t.Errorf("the awaiting may expire while it waits: %q", calls[0])
		}
		if u, _ := flag(calls[0], "-u"); u != "normal" {
			t.Errorf("the awaiting is spoken with urgency %q", u)
		}
		if _, ok := flag(calls[0], "-h"); ok {
			t.Errorf("the awaiting hints progress: %q", calls[0])
		}
		if id, _ := flag(calls[1], "-r"); id != "42" {
			t.Errorf("the triumph replaces not the awaiting: %q", calls[1])
		}
	})
	t.Run("resting", func(t *testing.T) {
		s := newStub(t)
		h := quiet(Config{Vox: librarium.VoxAuto})
		h.Proclaim(awaiting)
		_ = h.Close()
		calls := s.calls(t)
		if len(calls) != 2 {
			t.Fatalf("calls %q", calls)
		}
		if _, ok := flag(calls[1], "-t"); ok {
			t.Errorf("the resting awaiting never expires: %q", calls[1])
		}
	})
}
