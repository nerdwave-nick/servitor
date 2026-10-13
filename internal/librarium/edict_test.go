package librarium

import (
	"strings"
	"testing"
	"time"
)

// edict wraps a liturgy and extra rite keys into the scripture of an edict
// bearing the inscription "reason".
func edict(extra, liturgy string) string {
	return `{
  "pattern": "Mark I",
  "edict": true,
  "inscriptions": {"reason": {"purpose": "why"}},` + extra + `
  "liturgy": [` + liturgy + `]
}`
}

// visage is the edict of the codex: one plain value everywhere, every key
// an edict may bear.
const visage = `{
  "pattern": "Mark I",
  "purpose": "Capture the visage of the machine",
  "edict": true,
  "inscriptions": {
    "region": {"purpose": "the slurp geometry", "mandatory": true, "decrees": "full"},
    "reason": {},
  },
  "tongue": "zsh",
  "liturgy": [
    {"incantation": "grim -g \"$(slurp)\" ~/pictures/{{inscription.region}}.png", "patience": "2m", "tongue": "bash",
     "reversion": "rm -f ~/pictures/{{inscription.region}}.png"},
    {"vox-cast": "progress"},
    {"litany": "scripts/{{rite.name}}", "offerings": ["~/pictures", "{{inscription.reason}}"], "reversion": ""},
    {"vox-cast": "success"},
  ],
}`

var edictHeresyCases = []heresyCase{
	{"edict and aspects", `{"pattern": "Mark I", "aspects": ["on"], "edict": true}`, Heresy,
		`bears both "aspects" and "edict": true`, `true}`, 0},
	{"edict false without aspects", `{"pattern": "Mark I", "edict": false}`, Heresy,
		`declares neither "aspects" nor "edict": true`, `{`, 0},
	{"edict no boolean", `{"pattern": "Mark I", "edict": "yes"}`, Heresy,
		`the "edict" of a rite must be written as true or false, yet here stands a string`, `"yes"`, 0},
	{"edict keeps a sanctum", edict(``, `{"sanctum": "v.kdl", "scripture": ""}`), Heresy,
		`an edict may not keep a sanctum`, `"sanctum"`, 0},
	{"edict transcribes", edict(``, `{"vox-cast": "progress"}, {"transcription": "v", "scripture": null}`), Heresy,
		`an edict may not perform a transcription`, `"transcription"`, 0},
	{"edict tethers", edict(``, `{"tether": "t", "anchor": "a"}`), Heresy,
		`an edict may not bind a tether`, `"tether"`, 0},
	{"edict scans", edict(`"auspex": "cat x",`, ``), Heresy,
		`an edict bears no "auspex"`, `"cat x"`, 0},
	{"edict scans long", edict(`"auspex": {"rite": "cat x"},`, ``), Heresy,
		`an edict bears no "auspex"`, `{"rite"`, 0},
	{"edict incantation map", edict(``, `{"incantation": {"*": "a"}}`), Heresy,
		`the incantation of verse 1 is written as an aspect map, yet an edict bears no aspect`, `{"*"`, 0},
	{"edict incantation keyed map", edict(``, `{"incantation": {"on": "a", "off": "b"}}`), Heresy,
		`the incantation of verse 1 is written as an aspect map`, `{"on"`, 0},
	{"edict scroll map", edict(``, `{"litany": {"*": "s"}}`), Heresy,
		`the litany of verse 1's scroll is written as an aspect map`, `{"*"`, 0},
	{"edict offering map", edict(``, `{"litany": "s", "offerings": ["a", {"*": "b"}]}`), Heresy,
		`offering 2 of the litany of verse 1 is written as an aspect map`, `{"*": "b"}`, 0},
	{"edict reversion map", edict(``, `{"incantation": "a", "reversion": {"*": ""}}`), Heresy,
		`the "reversion" of the incantation of verse 1 is written as an aspect map`, `{"*": ""}`, 0},
	{"edict decrees map", `{"pattern": "Mark I", "edict": true, "inscriptions": {"m": {"decrees": {"*": "x"}}}}`, Heresy,
		`the "decrees" of the inscription "m" is written as an aspect map`, `{"*": "x"}`, 0},
	{"edict empty map", edict(``, `{"incantation": "a", "reversion": {}}`), Heresy,
		`is written as an aspect map, yet an edict bears no aspect`, `{}`, 0},
	{"edict patience map", edict(``, `{"incantation": "a", "patience": {"*": "1s"}}`), Heresy,
		`the "patience" of the incantation of verse 1 must be written as a string, yet here stands an object`, `{"*": "1s"}`, 0},
	{"edict step tongue map", edict(``, `{"litany": "s", "tongue": {"*": "zsh"}}`), Heresy,
		`the "tongue" of the litany of verse 1 must be written as a string, yet here stands an object`, `{"*": "zsh"}`, 0},
	{"edict tongue map", edict(`"tongue": {"*": "zsh"},`, ``), Heresy,
		`the "tongue" of a rite must be written as a string, yet here stands an object`, `{"*": "zsh"}`, 0},
	{"edict speaks aspect", edict(``, `{"incantation": "echo {{aspect}}"}`), Heresy,
		`{{aspect}} calls upon the aspect being invoked, yet an edict bears no aspect`, `{{aspect}}`, 0},
	{"edict speaks former", edict(``, `{"incantation": "a", "reversion": "echo {{former}}"}`), Heresy,
		`{{former}} calls upon the former aspect, yet an edict bears no aspect`, `{{former}}`, 0},
	{"edict speaks aspect in an offering", edict(``, `{"litany": "s", "offerings": ["{{aspect}}"]}`), Heresy,
		`yet an edict bears no aspect`, `{{aspect}}`, 0},
}

func init() { heresyCases = append(heresyCases, edictHeresyCases...) }

func TestParse_EdictIsARiteWithoutAspects(t *testing.T) {
	r, fs := parse("visage", "/lib/rites/visage.json", []byte(visage))
	if len(fs) != 0 {
		t.Fatalf("unexpected findings:\n%v", fs)
	}
	want := &Rite{
		Name: "visage", Path: "/lib/rites/visage.json", Purpose: "Capture the visage of the machine", Edict: true, markI: true,
		Inscriptions: []Inscription{
			{Key: "region", Purpose: "the slurp geometry", Mandatory: true, Decrees: Uniform("full")},
			{Key: "reason"},
		},
		Tongue: "zsh",
		Liturgy: []Step{
			&Incantation{Command: Uniform(`grim -g "$(slurp)" ~/pictures/{{inscription.region}}.png`),
				Utterance: Utterance{Tongue: "bash", Patience: 2 * time.Minute, Reversion: Uniform("rm -f ~/pictures/{{inscription.region}}.png")}},
			&voxCast{Tidings: VoxProgress},
			&Litany{Scroll: Uniform("scripts/{{rite.name}}"),
				Offerings: []AspectMap[string]{Uniform("~/pictures"), Uniform("{{inscription.reason}}")},
				Utterance: Utterance{Reversion: Uniform("")}},
			&voxCast{Tidings: VoxSuccess},
		},
	}
	if !r.Equal(want) {
		t.Fatalf("got  %#v\nwant %#v", r, want)
	}
	if r.Aspects != nil || !r.Scope().Edict {
		t.Fatalf("an edict bears aspects %v or speaks them (scope %+v)", r.Aspects, r.Scope())
	}
}

func TestParse_EdictMarkerFalseIsUnwritten(t *testing.T) {
	cases := map[string]string{
		"edict false with aspects": `{"pattern": "Mark I", "aspects": ["on", "off"], "edict": false,
			"liturgy": [{"incantation": {"on": "{{aspect}}", "*": "{{former}}"}}]}`,
		"edict false and auspex": `{"pattern": "Mark I", "edict": false, "aspects": ["on"], "auspex": "cat x"}`,
	}
	for name, scripture := range cases {
		t.Run(name, func(t *testing.T) {
			r, fs := parse("rite", "/r.json", []byte(scripture))
			if r == nil || len(fs) > 0 {
				t.Fatalf("rite %v, findings:\n%v", r, fs)
			}
			if r.Edict || len(r.Aspects) == 0 || r.Scope().Edict {
				t.Fatalf("a rite of \"edict\": false is read as an edict: %#v", r)
			}
		})
	}
}

func TestParse_EdictBearsEveryKeyItMay(t *testing.T) {
	cases := map[string]string{
		"bare edict":       `{"pattern": "Mark I", "edict": true}`,
		"empty liturgy":    edict(``, ``),
		"decree one word":  `{"pattern": "Mark I", "edict": true, "inscriptions": {"m": {"mandatory": true, "decrees": "x"}}}`,
		"schema named":     `{"$schema": "https://example.org/r.json", "pattern": "Mark I", "edict": true}`,
		"escaped aspect":   edict(``, `{"incantation": "echo {{{{aspect}}"}`),
		"every edict mark": edict(``, `{"incantation": "{{rite.name}} {{inscription.reason}}", "reversion": "{{rite.name}}"}`),
		"incantation":      edict(``, `{"incantation": "a", "tongue": "sh", "patience": "1s", "reversion": "b"}`),
		"litany":           edict(``, `{"litany": "s", "offerings": [], "tongue": "sh", "patience": "1s", "reversion": "b"}`),
		"vox-casts":        edict(``, `{"vox-cast": "progress"}, {"incantation": "a"}, {"vox-cast": "success"}`),
	}
	for name, scripture := range cases {
		t.Run(name, func(t *testing.T) {
			if r, fs := parse("rite", "/r.json", []byte(scripture)); r == nil || len(fs) > 0 || !r.Edict {
				t.Fatalf("rite %v, findings:\n%v", r, fs)
			}
		})
	}
}

// TestParse_EdictHeresyStandsAlone: a malformed marker, or one beside
// aspects, is denounced once — without a second lament that the rite
// declares neither.
func TestParse_EdictHeresyStandsAlone(t *testing.T) {
	for _, scripture := range []string{
		`{"pattern": "Mark I", "edict": "yes"}`,
		`{"pattern": "Mark I", "edict": null}`,
		`{"pattern": "Mark I", "aspects": ["on"], "edict": true}`,
	} {
		_, fs := parse("rite", "/r.json", []byte(scripture))
		if len(fs) != 1 {
			t.Errorf("%s: want one finding, got\n%v", scripture, fs)
		}
	}
}

// TestParse_EdictAspectMapIsDenouncedOnce: an aspect map in an edict is one
// heresy, never further laments of undeclared or unserved aspects.
func TestParse_EdictAspectMapIsDenouncedOnce(t *testing.T) {
	_, fs := parse("rite", "/r.json", []byte(edict(``, `{"incantation": {"on": "a", "off": "b"}}`)))
	if len(fs) != 1 || !strings.Contains(fs[0].Message, "aspect map") {
		t.Fatalf("findings:\n%v", fs)
	}
}

// TestParse_NeitherNamesBothWays: a rite declaring neither aspects nor the
// edict's marker is told of both ways to amend it.
func TestParse_NeitherNamesBothWays(t *testing.T) {
	_, fs := parse("rite", "/r.json", []byte(`{"pattern": "Mark I", "liturgy": []}`))
	if len(fs) != 1 {
		t.Fatalf("findings:\n%v", fs)
	}
	for _, want := range []string{`"aspects"`, `"edict": true`, "edict"} {
		if !strings.Contains(fs[0].Message, want) {
			t.Errorf("the lament lacks %q: %s", want, fs[0].Message)
		}
	}
}

// TestMarshal_EdictRoundTrips: an edict is written with "edict": true and
// no "aspects", a rite of aspects with no "edict".
func TestMarshal_EdictRoundTrips(t *testing.T) {
	forged := &Rite{Name: "reboot", Edict: true, Liturgy: []Step{&Incantation{Command: Uniform("systemctl reboot")}}}
	if data := string(roundTrip(t, forged)); !strings.Contains(data, `"edict": true`) || strings.Contains(data, `"aspects"`) {
		t.Fatalf("a forged edict is written otherwise:\n%s", data)
	}
	rite := &Rite{Name: "x", Aspects: []string{"on", "off"}, Subjects: []Subject{{Name: "s",
		Matter: &MachineSpirit{Auspex: &SpiritAuspex{Scan: "cat s"}, Enact: PerAspect(Entry[string]{"*", "echo {{omen}} > s"})}}}}
	if data := string(roundTrip(t, rite)); strings.Contains(data, `"edict"`) {
		t.Fatalf("a rite of aspects must bear no \"edict\":\n%s", data)
	}
}
