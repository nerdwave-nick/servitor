package tui

import (
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// everyKind is a rite of pattern Mark I holding every kind of step, every
// further rite and aspect maps of every shape the wizard must keep.
const everyKind = `{
  "pattern": "Mark I",
  "purpose": "The visage of the machine",
  "aspects": ["dark", "finii", "porpl"],
  "inscriptions": {
    "reason": {"purpose": "why"},
    "mode": {"mandatory": true, "decrees": {"dark": "dim", "*": "bright"}},
    "hue": {"decrees": "red"}
  },
  "auspex": {"rite": "cat /tmp/aspect", "patience": "5s"},
  "tongue": "sh",
  "liturgy": [
    {"vox-cast": "progress"},
    {"sanctum": "~/.config/x.kdl", "ward": "visage", "glyph": "/*", "closing-glyph": "*/", "consecrate": true,
     "scripture": {"dark": ["a", "b"], "porpl": {"tome": "tomes/porpl", "illuminate": true}, "*": ""}},
    {"transcription": "/tmp/whole.conf", "scripture": {"dark": null, "*": "whole {{aspect}}"}, "seal": "0600", "zeal": true},
    {"tether": "/tmp/current", "anchor": {"finii": null, "*": "/tmp/themes/{{aspect}}"}, "zeal": true},
    {"incantation": {"dark": "echo dark", "*": "echo light"}, "tongue": "zsh", "patience": "2m", "reversion": "echo {{former}}"},
    {"litany": "scripts/hooks", "offerings": ["--to", {"porpl": "purple", "*": "{{aspect}}"}], "reversion": {"dark": "", "*": "undo"}},
    {"incantation": "true"},
    {"vox-cast": "success"}
  ]
}`

func TestDraft_EveryKindRoundTrips(t *testing.T) {
	want, found := librarium.Parse("visage", "/lib/rites/visage.json", []byte(everyKind))
	if found.Heretical() {
		t.Fatalf("the fixture is heretical: %v", found)
	}
	d := riteDraftFrom(want)
	if d.name != "visage" || d.aspects != "dark, finii, porpl" || len(d.steps) != 8 {
		t.Fatalf("draft = %+v", d)
	}
	if !d.steps[4].further || d.steps[6].further || !d.steps[2].further || d.steps[1].further {
		t.Fatalf("further rites unveiled wrongly: %v %v %v %v",
			d.steps[1].further, d.steps[2].further, d.steps[4].further, d.steps[6].further)
	}
	got := d.toRite("/lib/rites/visage.json")
	if !got.Equal(want) {
		data, _ := librarium.Marshal(got)
		t.Fatalf("the rite changed on its way through the wizard:\n%s", data)
	}
}

func TestDraft_SameForEveryAspectWritesTheFallback(t *testing.T) {
	aspects := []string{"dark", "finii", "porpl"}
	v := newVaried[string](true)
	if _, same := v.value("dark"); same {
		t.Fatal("a value set apart begins as each aspect's own")
	}
	v.set("dark", "x", false)
	v.set("finii", "y", true)
	if v.ownAfter(aspects, 1) {
		t.Fatal("porpl speaks no value of its own")
	}
	m := v.toMap(aspects)
	if e := m.Entries(); len(e) != 2 || e[0] != (librarium.Entry[string]{Aspect: "dark", Value: "x"}) ||
		e[1] != (librarium.Entry[string]{Aspect: librarium.Fallback, Value: "y"}) {
		t.Fatalf("entries = %+v", e)
	}
	if got, same := v.value("porpl"); got != "y" || !same {
		t.Fatalf("porpl = %q, same %v", got, same)
	}
	v.set("dark", "y", true)
	if m := v.toMap(aspects); !m.IsUniform() {
		t.Fatalf("one value for every aspect is written once: %+v", m.Entries())
	}
}

func TestDraft_NewStepsAndUnwrittenRites(t *testing.T) {
	d := newRiteDraft()
	d.name, d.aspects = "theme", "dark, porpl"
	inc := newStepDraft(librarium.KindIncantation)
	inc.command.set("dark", "echo {{aspect}}", true)
	lit := newStepDraft(librarium.KindLitany)
	lit.command.set("dark", "scripts/hooks", true)
	lit.offerings.set("dark", []string{"~/themes/{{aspect}}"}, true)
	lit.reversion.set("dark", "", true)
	lit.fixed["patience"] = "5s"
	d.steps = []stepDraft{inc, lit}
	r := d.toRite("/lib/rites/theme.json")
	want := &librarium.Rite{Name: "theme", Aspects: []string{"dark", "porpl"}, Liturgy: []librarium.Step{
		&librarium.Incantation{Command: librarium.Uniform("echo {{aspect}}")},
		&librarium.Litany{Scroll: librarium.Uniform("scripts/hooks"),
			Offerings: []librarium.AspectMap[string]{librarium.Uniform("~/themes/{{aspect}}")},
			Utterance: librarium.Utterance{Patience: 5 * time.Second}},
	}}
	if !r.Equal(want) {
		data, _ := librarium.Marshal(r)
		t.Fatalf("rite =\n%s", data)
	}
}

func TestParseAspectsAndInscriptionKeys(t *testing.T) {
	if s, err := parseAspects(" on,off  auto "); err != nil || len(s) != 3 {
		t.Fatalf("%v %v", s, err)
	}
	for _, bad := range []string{"", "on, on", "b@d", "*"} {
		if _, err := parseAspects(bad); err == nil {
			t.Errorf("parseAspects(%q) should fail", bad)
		}
	}
	if k, err := parseKeys("reason, mode"); err != nil || len(k) != 2 {
		t.Fatalf("%v %v", k, err)
	}
	for _, bad := range []string{"foresee", "a, a", "b@d"} {
		if _, err := parseKeys(bad); err == nil {
			t.Errorf("parseKeys(%q) should fail", bad)
		}
	}
}
