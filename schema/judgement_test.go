package schema

import (
	"fmt"
	"maps"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// scripture is a rite of two aspects, "on" and "off", with the given extra
// keys and liturgy.
func scripture(extra, liturgy string) string {
	return fmt.Sprintf(`{"pattern": "Mark I", "aspects": ["on", "off"]%s, "liturgy": [%s]}`, extra, liturgy)
}

func step(s string) string { return scripture("", s) }

// judgements are scriptures whose purity the schema can tell as well as the
// librarium: both must judge each alike. Heresies only the whole rite can
// reveal — an aspect map naming an undeclared aspect or leaving a declared
// one unserved, unknown placeholders, a ward holding its closing glyph,
// clashing wards — lie beyond any schema and are left to the Inquisition.
var judgements = []struct {
	name, scripture string
	pure            bool
}{
	{"bare", `{"pattern": "Mark I", "aspects": ["on"]}`, true},
	{"schema named", `{"$schema": "https://example.org/r.json", "pattern": "Mark I", "aspects": ["on"]}`, true},
	{"schema no string", `{"$schema": 7, "pattern": "Mark I", "aspects": ["on"]}`, false},
	{"pattern missing", `{"aspects": ["on"]}`, false},
	{"pattern of another Mark", `{"pattern": "Mark II", "aspects": ["on"]}`, false},
	{"pattern malformed", `{"pattern": "Mark 1", "aspects": ["on"]}`, false},
	{"unknown key", `{"pattern": "Mark I", "aspects": ["on"], "states": []}`, false},
	{"no object", `["Mark I"]`, false},
	{"purpose", scripture(`, "purpose": "why"`, ``), true},
	{"purpose no string", scripture(`, "purpose": 1`, ``), false},
	{"aspects missing", `{"pattern": "Mark I"}`, false},
	{"aspects empty", `{"pattern": "Mark I", "aspects": []}`, false},
	{"aspect twice", `{"pattern": "Mark I", "aspects": ["on", "on"]}`, false},
	{"aspect named *", `{"pattern": "Mark I", "aspects": ["*"]}`, false},
	{"aspect with space", `{"pattern": "Mark I", "aspects": ["o n"]}`, false},
	{"aspect no string", `{"pattern": "Mark I", "aspects": [1]}`, false},
	{"tongue", scripture(`, "tongue": "fish"`, ``), true},
	{"tongue empty", scripture(`, "tongue": " "`, ``), false},
	{"liturgy no list", `{"pattern": "Mark I", "aspects": ["on"], "liturgy": {}}`, false},

	{"inscriptions", scripture(`, "inscriptions": {"reason": {"purpose": "why", "mandatory": false}, "mode": {"decrees": "x"},
		"veil": {"mandatory": true, "decrees": {"on": "a", "*": "b"}}, "none": {"decrees": {}}, "bare": {}}`, ``), true},
	{"inscription reserved", scripture(`, "inscriptions": {"foresee": {}}`, ``), false},
	{"inscription misnamed", scripture(`, "inscriptions": {"a b": {}}`, ``), false},
	{"inscription unknown key", scripture(`, "inscriptions": {"a": {"optional": true}}`, ``), false},
	{"inscription no object", scripture(`, "inscriptions": {"a": "why"}`, ``), false},
	{"inscriptions no object", scripture(`, "inscriptions": []`, ``), false},
	{"mandatory no boolean", scripture(`, "inscriptions": {"a": {"mandatory": "yes"}}`, ``), false},
	{"decree no string", scripture(`, "inscriptions": {"a": {"decrees": {"on": 1}}}`, ``), false},

	{"auspex short", scripture(`, "auspex": "cat ~/.cache/x"`, ``), true},
	{"auspex long", scripture(`, "auspex": {"rite": "cat x", "patience": "5s"}`, ``), true},
	{"auspex long without rite", scripture(`, "auspex": {"patience": "5s"}`, ``), false},
	{"auspex unknown key", scripture(`, "auspex": {"rite": "x", "timeout": "5s"}`, ``), false},
	{"auspex empty", scripture(`, "auspex": ""`, ``), false},
	{"auspex number", scripture(`, "auspex": 5`, ``), false},

	{"no kind", step(`{"ward": "w"}`), false},
	{"two kinds", step(`{"sanctum": "v", "tether": "t", "scripture": "", "anchor": "a"}`), false},
	{"step no object", step(`"sanctum"`), false},

	{"sanctum", step(`{"sanctum": "~/v.kdl", "scripture": ""}`), true},
	{"sanctum every key", step(`{"sanctum": "v", "ward": "w", "glyph": "/*", "closing-glyph": "*/", "consecrate": true,
		"scripture": ["a", "b"]}`), true},
	{"sanctum aspect map", step(`{"sanctum": "v", "scripture": {"on": {"tome": "t", "illuminate": true}, "*": []}}`), true},
	{"sanctum tome", step(`{"sanctum": "v", "scripture": {"tome": "t"}}`), true},
	{"sanctum null", step(`{"sanctum": "v", "scripture": null}`), false},
	{"sanctum null for an aspect", step(`{"sanctum": "v", "scripture": {"on": null, "*": ""}}`), false},
	{"sanctum without scripture", step(`{"sanctum": "v"}`), false},
	{"sanctum empty vessel", step(`{"sanctum": "", "scripture": ""}`), false},
	{"sanctum unknown key", step(`{"sanctum": "v", "scripture": "", "zeal": true}`), false},
	{"ward with space", step(`{"sanctum": "v", "ward": "a b", "scripture": ""}`), false},
	{"ward empty", step(`{"sanctum": "v", "ward": "", "scripture": ""}`), false},
	{"glyph empty", step(`{"sanctum": "v", "glyph": " ", "scripture": ""}`), false},
	{"closing glyph of two lines", step(`{"sanctum": "v", "closing-glyph": "a\nb", "scripture": ""}`), false},
	{"closing glyph empty", step(`{"sanctum": "v", "closing-glyph": "", "scripture": ""}`), true},
	{"consecrate no boolean", step(`{"sanctum": "v", "consecrate": 1, "scripture": ""}`), false},
	{"line no string", step(`{"sanctum": "v", "scripture": ["a", 1]}`), false},
	{"tome without tome", step(`{"sanctum": "v", "scripture": {"illuminate": true}}`), false},
	{"tome unknown key", step(`{"sanctum": "v", "scripture": {"tome": "t", "verbatim": true}}`), false},
	{"tome empty", step(`{"sanctum": "v", "scripture": {"tome": ""}}`), false},
	{"empty aspect map", step(`{"sanctum": "v", "scripture": {}}`), false},
	{"aspect map misnamed key", step(`{"sanctum": "v", "scripture": {"o n": "", "*": ""}}`), false},

	{"transcription", step(`{"transcription": "v", "scripture": "x"}`), true},
	{"transcription null", step(`{"transcription": "v", "scripture": null}`), true},
	{"transcription null for an aspect", step(`{"transcription": "v", "scripture": {"on": null, "*": {"tome": "t"}}}`), true},
	{"transcription every key", step(`{"transcription": "v", "scripture": "", "seal": "0755", "zeal": true}`), true},
	{"transcription by alias", step(`{"transcription": "v", "scripture": "", "seal": "644", "force": false}`), true},
	{"transcription zeal and alias", step(`{"transcription": "v", "scripture": "", "zeal": true, "force": true}`), false},
	{"transcription without scripture", step(`{"transcription": "v"}`), false},
	{"seal of nines", step(`{"transcription": "v", "scripture": "", "seal": "999"}`), false},
	{"seal number", step(`{"transcription": "v", "scripture": "", "seal": 644}`), false},
	{"seal too long", step(`{"transcription": "v", "scripture": "", "seal": "07555"}`), false},
	{"zeal no boolean", step(`{"transcription": "v", "scripture": "", "zeal": "yes"}`), false},

	{"tether", step(`{"tether": "t", "anchor": "a"}`), true},
	{"tether null", step(`{"tether": "t", "anchor": null, "zeal": true}`), true},
	{"tether aspect map", step(`{"tether": "t", "anchor": {"on": "a", "off": null}, "force": true}`), true},
	{"tether without anchor", step(`{"tether": "t"}`), false},
	{"tether empty anchor", step(`{"tether": "t", "anchor": ""}`), false},
	{"tether empty name", step(`{"tether": "", "anchor": "a"}`), false},
	{"tether anchor list", step(`{"tether": "t", "anchor": ["a"]}`), false},
	{"tether zeal and alias", step(`{"tether": "t", "anchor": "a", "zeal": false, "force": true}`), false},

	{"incantation", step(`{"incantation": "true"}`), true},
	{"incantation every key", step(`{"incantation": {"on": "a", "*": "b"}, "tongue": "sh", "patience": "1m30s",
		"reversion": {"on": "", "*": "c"}}`), true},
	{"incantation empty", step(`{"incantation": " "}`), false},
	{"incantation empty map", step(`{"incantation": {}}`), false},
	{"incantation list", step(`{"incantation": ["a"]}`), false},
	{"incantation unknown key", step(`{"incantation": "a", "allow_failure": true}`), false},
	{"reversion empty", step(`{"incantation": "a", "reversion": ""}`), true},
	{"reversion number", step(`{"incantation": "a", "reversion": 0}`), false},

	{"litany", step(`{"litany": "s"}`), true},
	{"litany every key", step(`{"litany": {"*": "s"}, "offerings": ["", {"on": "1", "*": "{{aspect}}"}], "tongue": "zsh",
		"patience": "2m", "reversion": "r"}`), true},
	{"offerings empty", step(`{"litany": "s", "offerings": []}`), true},
	{"offerings no list", step(`{"litany": "s", "offerings": "a b"}`), false},
	{"offering number", step(`{"litany": "s", "offerings": [1]}`), false},
	{"offering empty map", step(`{"litany": "s", "offerings": [{}]}`), false},
	{"litany empty", step(`{"litany": ""}`), false},
	{"litany seal", step(`{"litany": "s", "seal": "0755"}`), false},

	{"vox-cast progress", step(`{"vox-cast": "progress"}`), true},
	{"vox-cast success", step(`{"vox-cast": "success"}`), true},
	{"vox-cast failure", step(`{"vox-cast": "failure"}`), false},
	{"vox-cast with patience", step(`{"vox-cast": "progress", "patience": "1s"}`), false},
}

// patiences are spans as the librarium reads them: pure when greater than
// nothing.
var patiences = map[string]bool{
	`"30s"`: true, `"1.5s"`: true, `".5s"`: true, `"1.s"`: true, `"+5s"`: true, `"1h2m3s"`: true, `"500ms"`: true,
	`"2us"`: true, `"2µs"`: true, `"7ns"`: true, `"0s1ms"`: true,
	`"0s"`: false, `"0"`: false, `"0.0m"`: false, `"-5s"`: false, `"5"`: false, `"5 s"`: false, `"soon"`: false,
	`""`: false, `"."`: false, `"s"`: false, `30`: false, `null`: false,
}

func TestRiteSchema_JudgesAsTheLibrariumJudges(t *testing.T) {
	s := compile(t, Rite)
	cases := judgements
	for p, pure := range patiences {
		cases = append(cases, struct {
			name, scripture string
			pure            bool
		}{"patience " + p, step(`{"incantation": "a", "patience": ` + p + `}`), pure})
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, fs := librarium.Parse("r", "/lib/rites/r.json", []byte(c.scripture))
			if fs.Heretical() == c.pure {
				t.Fatalf("the librarium judges %s (pure %v):\n%v", c.scripture, !fs.Heretical(), fs)
			}
			if err := validate(t, s, []byte(c.scripture)); (err == nil) != c.pure {
				t.Fatalf("the schema judges %s (pure %v): %v", c.scripture, err == nil, err)
			}
		})
	}
}

var settingsJudgements = map[string]bool{
	`{"pattern": "Mark I"}`: true,
	`{"$schema": "https://example.org/s.json", "pattern": "Mark I", "tongue": "zsh", "chronicle": "~/c.jsonl",
	  "vox": "notify-send", "patience": "45s"}`: true,
	`{"pattern": "Mark I", "vox": "off"}`:       true,
	`{"tongue": "zsh"}`:                         false,
	`{"pattern": "Mark II"}`:                    false,
	`{"pattern": "Mark I", "shell": "zsh"}`:     false,
	`{"pattern": "Mark I", "vox": "loud"}`:      false,
	`{"pattern": "Mark I", "vox": false}`:       false,
	`{"pattern": "Mark I", "tongue": ""}`:       false,
	`{"pattern": "Mark I", "chronicle": " "}`:   false,
	`{"pattern": "Mark I", "chronicle": ["a"]}`: false,
	`{"pattern": "Mark I", "$schema": null}`:    false,
	`{"pattern": "Mark I", "aspects": ["on"]}`:  false,
	`["Mark I"]`: false,
}

func TestSettingsSchema_JudgesAsTheLibrariumJudges(t *testing.T) {
	s := compile(t, Settings)
	cases := maps.Clone(settingsJudgements)
	for p, pure := range patiences {
		cases[`{"pattern": "Mark I", "patience": `+p+`}`] = pure
	}
	for scripture, pure := range cases {
		t.Run(scripture, func(t *testing.T) {
			_, fs := librarium.ParseSettings("/lib/servitor.json", []byte(scripture))
			if fs.Heretical() == pure {
				t.Fatalf("the librarium judges %s (pure %v):\n%v", scripture, !fs.Heretical(), fs)
			}
			if err := validate(t, s, []byte(scripture)); (err == nil) != pure {
				t.Fatalf("the schema judges %s (pure %v): %v", scripture, err == nil, err)
			}
		})
	}
}
