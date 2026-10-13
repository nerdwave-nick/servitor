package vox

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// TestExaltationTemplates_NameTheIntercessorAndTheSigil: the tidings that
// the exalted verses await the master's proof are drawn from lists of
// their own beside the dominion's, a rite's and an edict's apart; each
// names the rite, the intercessor and the sigil, speaks no aspect, and is
// grimdark.
func TestExaltationTemplates_NameTheIntercessorAndTheSigil(t *testing.T) {
	marks := regexp.MustCompile(`\{\{[^}]*\}\}`)
	for edict, list := range map[bool][]string{false: templates[invocation.Exaltation],
		true: edictTemplates[invocation.Exaltation]} {
		if len(list) < 3 {
			t.Errorf("edict %v: %d templates", edict, len(list))
		}
		for _, tmpl := range list {
			grimdark(t, tmpl)
			got := marks.FindAllString(tmpl, -1)
			for _, m := range []string{"{{rite.name}}", "{{intercessor}}", "{{sigil}}"} {
				if !slices.Contains(got, m) {
					t.Errorf("template %q does not speak %s", tmpl, m)
				}
			}
			for _, m := range got {
				if m != "{{rite.name}}" && m != "{{intercessor}}" && m != "{{sigil}}" {
					t.Errorf("template %q speaks %s", tmpl, m)
				}
			}
			if edict && !strings.Contains(strings.ToLower(tmpl), "the edict {{rite.name}}") {
				t.Errorf("edict template %q names not the edict", tmpl)
			}
		}
	}
}

func TestCompose_ExaltationNamesTheIntercessorAndTheSigil(t *testing.T) {
	for edict, list := range map[bool][]string{false: templates[invocation.Exaltation],
		true: edictTemplates[invocation.Exaltation]} {
		for i := range list {
			p := invocation.Proclamation{Tidings: invocation.Exaltation, Edict: edict, Intercessor: "sudo",
				Sigil: "4f0c2a9e1b7d", Values: placeholder.Values{Rite: "split-lock", Aspect: "on"}}
			m := Compose(p, func(int) int { return i })
			if m.Tidings != invocation.Exaltation || m.Body != m.Flavour || !strings.Contains(m.Flavour, "split-lock") ||
				!strings.Contains(m.Flavour, "sudo") || !strings.Contains(m.Flavour, "4f0c2a9e1b7d") ||
				strings.ContainsAny(m.Flavour, "«»{") {
				t.Errorf("edict %v, template %d composed %+v", edict, i, m)
			}
		}
	}
	p := invocation.Proclamation{Tidings: invocation.Exaltation, Intercessor: "{{former}}", Sigil: "x",
		Values: placeholder.Values{Rite: "r", Aspect: "on", Former: "off"}}
	if m := Compose(p, func(int) int { return 0 }); !strings.Contains(m.Flavour, "{{former}}") {
		t.Errorf("the intercessor was illuminated: %q", m.Flavour)
	}
}

// TestExaltationTemplates_OfARiteOfAspectsSpeakNoVerse: a rite of aspects
// bears subjects and observances, never verses, and may hold only one of
// them exalted: its tidings of exaltation speak of neither by name.
func TestExaltationTemplates_OfARiteOfAspectsSpeakNoVerse(t *testing.T) {
	for _, tmpl := range templates[invocation.Exaltation] {
		for _, word := range []string{"verse", "subject", "observance"} {
			if strings.Contains(strings.ToLower(tmpl), word) {
				t.Errorf("template %q speaks of %ss", tmpl, word)
			}
		}
	}
}
