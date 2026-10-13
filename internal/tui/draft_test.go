package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// everyKind is a rite of aspects holding a subject of every designation the
// wizard writes, observances of both designations with their guards, every
// further rite and omens of every shape the wizard must keep.
const everyKind = `{
  "pattern": "Mark II",
  "purpose": "The visage of the machine",
  "aspects": ["dark", "finii", "porpl"],
  "inscriptions": {
    "reason": {"purpose": "why"},
    "mode": {"mandatory": true, "decrees": {"dark": "dim", "*": "bright"}},
    "hue": {"decrees": {"*": "red"}}
  },
  "tongue": "sh",
  "vox-cast": ["progress", "success"],
  "subjects": [
    {"name": "x", "designation": "sanctum", "purpose": "the visage", "vessel": "~/.config/x.kdl", "ward": "visage",
     "glyph": "/*", "closing-glyph": "*/", "consecrate": true,
     "omens": {"dark": ["a", "b"], "porpl": {"tome": "tomes/porpl", "illuminate": true}, "*": ""},
     "introit": ["fade"]},
    {"name": "whole", "designation": "transcription", "vessel": "/tmp/whole.conf",
     "omens": {"dark": null, "*": "whole {{aspect}}"}, "seal": "0600", "zeal": true},
    {"name": "current", "designation": "tether", "tether": "/tmp/current",
     "omens": {"finii": null, "*": "/tmp/themes/{{aspect}}"}, "zeal": true, "recessional": ["hooks"]},
    {"name": "lamp", "designation": "machine-spirit", "omens": {"dark": "off", "*": "on"},
     "auspex": {"scan": "cat /tmp/lamp", "trials": {"on": "test -e /tmp/lamp"}, "patience": "1s"},
     "enact": {"on": "touch /tmp/lamp", "*": "rm -f /tmp/lamp"}, "tongue": "zsh", "patience": "2m"}
  ],
  "observances": [
    {"name": "fade", "designation": "incantation", "to": ["dark"], "words": "echo dark", "tongue": "zsh",
     "patience": "2m", "reversion": "echo {{former}}"},
    {"name": "hooks", "designation": "litany", "from": ["porpl"], "scroll": "scripts/hooks",
     "offerings": ["--to", "{{aspect}}"], "reversion": "undo"}
  ]
}`

// TestDraft_EveryFormRoundTrips: a rite of every form the wizard writes
// reads back, through its draft, as the very rite it was.
func TestDraft_EveryFormRoundTrips(t *testing.T) {
	want, found := librarium.Parse("visage", "/lib/rites/visage.json", []byte(everyKind))
	if found.Heretical() {
		t.Fatalf("the fixture is heretical: %v", found)
	}
	d := riteDraftFrom(want)
	if d.name != "visage" || d.aspects != "dark, finii, porpl" || len(d.subjects) != 4 || len(d.observances) != 2 ||
		len(d.steps) != 0 {
		t.Fatalf("draft = %+v", d)
	}
	if !d.subjects[1].step.further || d.subjects[0].step.further || !d.observances[0].step.further {
		t.Fatalf("further rites unveiled wrongly: %v %v %v",
			d.subjects[0].step.further, d.subjects[1].step.further, d.observances[0].step.further)
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

// edictScripture is an edict holding every kind of verse an edict may bear
// and every further rite of them, each value written plainly.
const edictScripture = `{
  "pattern": "Mark II",
  "purpose": "Capture the visage",
  "edict": true,
  "inscriptions": {
    "region": {"purpose": "the slurp geometry", "decrees": "full"},
    "name": {"mandatory": true}
  },
  "tongue": "sh",
  "vox-cast": ["progress", "success"],
  "liturgy": [
    {"designation": "incantation", "words": "grim {{inscription.region}}", "tongue": "zsh", "patience": "2m",
     "reversion": "rm -f shot"},
    {"designation": "litany", "scroll": "scripts/shot", "offerings": ["--to", "~/{{inscription.name}}"],
     "patience": "5s"}
  ]
}`

func TestDraft_AnEdictRoundTrips(t *testing.T) {
	want, found := librarium.Parse("capture", "/lib/rites/capture.json", []byte(edictScripture))
	if found.Heretical() || !want.Edict {
		t.Fatalf("the fixture is no pure edict: %v", found)
	}
	d := riteDraftFrom(want)
	if !d.edict || len(d.aspectList()) != 0 {
		t.Fatalf("the draft of an edict speaks of aspects: %+v", d)
	}
	got := d.toRite("/lib/rites/capture.json")
	data, err := librarium.Marshal(got)
	if err != nil || !got.Equal(want) {
		t.Fatalf("the edict changed on its way through the wizard (%v):\n%s", err, data)
	}
	if !strings.Contains(string(data), `"edict": true`) || strings.Contains(string(data), `"aspects"`) {
		t.Fatalf("the edict is written with aspects:\n%s", data)
	}
}

// TestDraft_NewEdictStepsHoldOneValue: the verses of a new edict are
// written in the designation form, each value once, its vox-casts a list
// of the rite.
func TestDraft_NewEdictStepsHoldOneValue(t *testing.T) {
	d := newRiteDraft()
	d.name, d.edict, d.voxCasts = "capture", true, []string{librarium.VoxProgress}
	d.declare([]string{"region"})
	d.inscriptions[0].decrees[librarium.Fallback] = "full"
	inc := newStepDraft(librarium.KindIncantation)
	inc.command.setOnce("grim {{inscription.region}}")
	inc.reversion.setOnce("")
	lit := newStepDraft(librarium.KindLitany)
	lit.command.setOnce("scripts/shot")
	lit.offerings.setOnce([]string{"--to", "~/shots"})
	lit.fixed["patience"] = "5s"
	d.steps = []stepDraft{inc, lit}
	r := d.toRite("/lib/rites/capture.json")
	want := &librarium.Rite{Name: "capture", Edict: true, VoxCasts: []string{librarium.VoxProgress},
		Inscriptions: []librarium.Inscription{{Key: "region", Decrees: librarium.Uniform("full")}},
		Liturgy: []librarium.Step{
			&librarium.Incantation{Command: librarium.Uniform("grim {{inscription.region}}")},
			&librarium.Litany{Scroll: librarium.Uniform("scripts/shot"),
				Offerings: []librarium.AspectMap[string]{librarium.Uniform("--to"), librarium.Uniform("~/shots")},
				Utterance: librarium.Utterance{Patience: 5 * time.Second}},
		}}
	if !r.Equal(want) {
		data, _ := librarium.Marshal(r)
		t.Fatalf("edict =\n%s", data)
	}
	data, err := librarium.Marshal(r)
	if err != nil || !strings.Contains(string(data), `"pattern": "Mark II"`) ||
		!strings.Contains(string(data), `"designation": "incantation"`) {
		t.Fatalf("the edict is not written in pattern Mark II (%v):\n%s", err, data)
	}
}
