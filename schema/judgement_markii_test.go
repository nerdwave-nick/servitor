package schema

import (
	"fmt"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// markII is a rite of pattern Mark II of two aspects, "on" and "off", with
// the given extra keys and subjects.
func markII(extra, subjects string) string {
	return fmt.Sprintf(`{"pattern": "Mark II", "aspects": ["on", "off"]%s, "subjects": [%s]}`, extra, subjects)
}

func subject(s string) string { return markII("", s) }

// observed is a rite of one spirit naming the observance "o" as its introit,
// and the given observances.
func observed(observances string) string {
	return markII(`, "observances": [`+observances+`]`, `{"name": "x", "designation": "machine-spirit",
		"auspex": {"scan": "cat x"}, "enact": {"*": "a"}, "introit": ["o"]}`)
}

// edictII is an edict of pattern Mark II with the given extra keys and
// liturgy.
func edictII(extra, liturgy string) string {
	return fmt.Sprintf(`{"pattern": "Mark II", "edict": true%s, "liturgy": [%s]}`, extra, liturgy)
}

const spirit = `{"name": "x", "designation": "machine-spirit", "auspex": {"scan": "cat x"}, "enact": {"*": "a"}}`

// markIIJudgements are scriptures of pattern Mark II whose purity the
// schema tells as well as the librarium.
var markIIJudgements = []struct {
	name, scripture string
	pure            bool
}{
	{"bare", subject(spirit), true},
	{"every rite key", markII(`, "$schema": "https://example.org/r.json", "purpose": "p", "tongue": "zsh", "edict": false,
		"vox-cast": ["success", "progress"], "inscriptions": {"m": {"purpose": "p", "mandatory": true, "decrees": {"*": "a"}},
		"e": {"decrees": {}}, "b": {}}`, spirit), true},
	{"no vox-cast", markII(`, "vox-cast": []`, spirit), true},
	{"subjects missing", `{"pattern": "Mark II", "aspects": ["on"]}`, false},
	{"subjects empty", markII(``, ``), false},
	{"subjects no list", `{"pattern": "Mark II", "aspects": ["on"], "subjects": {}}`, false},
	{"aspects missing", `{"pattern": "Mark II", "subjects": [` + spirit + `]}`, false},
	{"aspects empty", `{"pattern": "Mark II", "aspects": [], "subjects": [` + spirit + `]}`, false},
	{"aspect named *", `{"pattern": "Mark II", "aspects": ["*"], "subjects": [` + spirit + `]}`, false},
	{"liturgy beside aspects", markII(`, "liturgy": []`, spirit), false},
	{"auspex of the rite", markII(`, "auspex": "cat x"`, spirit), false},
	{"unknown rite key", markII(`, "states": []`, spirit), false},
	{"vox-cast failure", markII(`, "vox-cast": ["failure"]`, spirit), false},
	{"vox-cast twice", markII(`, "vox-cast": ["success", "success"]`, spirit), false},
	{"vox-cast plain", markII(`, "vox-cast": "success"`, spirit), false},
	{"decree plain", markII(`, "inscriptions": {"m": {"decrees": "a"}}`, spirit), false},
	{"decree no string", markII(`, "inscriptions": {"m": {"decrees": {"on": 1}}}`, spirit), false},
	{"inscription reserved", markII(`, "inscriptions": {"foresee": {}}`, spirit), false},
	{"observances no list", markII(`, "observances": {}`, spirit), false},
	{"tongue empty", markII(`, "tongue": " "`, spirit), false},

	{"sanctum every key", subject(`{"name": "s", "designation": "sanctum", "purpose": "p", "vessel": "~/v.kdl", "ward": "w",
		"glyph": "/*", "closing-glyph": "*/", "consecrate": true, "omens": {"on": ["a", "b"], "*": {"tome": "t", "illuminate": true}},
		"exalted": false, "introit": [], "recessional": []}`), true},
	{"sanctum null omen", subject(`{"name": "s", "designation": "sanctum", "vessel": "v", "omens": {"*": null}}`), false},
	{"sanctum omens plain", subject(`{"name": "s", "designation": "sanctum", "vessel": "v", "omens": ""}`), false},
	{"sanctum omens a tome", subject(`{"name": "s", "designation": "sanctum", "vessel": "v", "omens": {"tome": "t"}}`), false},
	{"sanctum omens missing", subject(`{"name": "s", "designation": "sanctum", "vessel": "v"}`), false},
	{"sanctum vessel missing", subject(`{"name": "s", "designation": "sanctum", "omens": {"*": ""}}`), false},
	{"sanctum vessel empty", subject(`{"name": "s", "designation": "sanctum", "vessel": "", "omens": {"*": ""}}`), false},
	{"sanctum ward spaced", subject(`{"name": "s", "designation": "sanctum", "vessel": "v", "ward": "a b", "omens": {"*": ""}}`), false},
	{"sanctum zeal", subject(`{"name": "s", "designation": "sanctum", "vessel": "v", "zeal": true, "omens": {"*": ""}}`), false},
	{"transcription every key", subject(`{"name": "c", "designation": "transcription", "vessel": "v", "seal": "0644", "zeal": true,
		"omens": {"on": null, "*": "x"}}`), true},
	{"transcription by alias", subject(`{"name": "c", "designation": "transcription", "vessel": "v", "force": true, "omens": {"*": null}}`), true},
	{"transcription zeal and alias", subject(`{"name": "c", "designation": "transcription", "vessel": "v", "zeal": true, "force": true,
		"omens": {"*": null}}`), false},
	{"transcription bad seal", subject(`{"name": "c", "designation": "transcription", "vessel": "v", "seal": "rw", "omens": {"*": null}}`), false},
	{"tether", subject(`{"name": "t", "designation": "tether", "tether": "t", "omens": {"on": "a", "*": null}, "zeal": true}`), true},
	{"tether empty anchor", subject(`{"name": "t", "designation": "tether", "tether": "t", "omens": {"*": ""}}`), false},
	{"tether anchor list", subject(`{"name": "t", "designation": "tether", "tether": "t", "omens": {"*": ["a"]}}`), false},
	{"tether bound name missing", subject(`{"name": "t", "designation": "tether", "omens": {"*": null}}`), false},
	{"tether with a vessel", subject(`{"name": "t", "designation": "tether", "tether": "t", "vessel": "v", "omens": {"*": null}}`), false},
	{"tether with enact", subject(`{"name": "t", "designation": "tether", "tether": "t", "omens": {"*": null}, "enact": {"*": "a"}}`), false},
	{"tether with relics", subject(`{"name": "t", "designation": "tether", "tether": "/t", "exalted": true, "omens": {"*": null},
		"relics": []}`), false},
	{"spirit every key", subject(`{"name": "m", "designation": "machine-spirit", "purpose": "p", "exalted": true,
		"omens": {"on": "1", "*": "0"}, "auspex": {"scan": "cat /m", "trials": {"1": "true"}, "patience": "1s", "exalted": true},
		"enact": {"*": "echo {{omen}} > /m"}, "tongue": "sh", "patience": "5s", "relics": ["r"]}`), true},
	{"spirit by trials alone", subject(`{"name": "m", "designation": "machine-spirit", "auspex": {"trials": {}}, "enact": {"on": "a", "off": "b"}}`), true},
	{"spirit without auspex", subject(`{"name": "m", "designation": "machine-spirit", "enact": {"*": "a"}}`), true},
	{"spirit auspex of neither", subject(`{"name": "m", "designation": "machine-spirit", "auspex": {"patience": "1s"}, "enact": {"*": "a"}}`), false},
	{"spirit auspex unknown key", subject(`{"name": "m", "designation": "machine-spirit", "auspex": {"rite": "x"}, "enact": {"*": "a"}}`), false},
	{"spirit auspex plain", subject(`{"name": "m", "designation": "machine-spirit", "auspex": "cat m", "enact": {"*": "a"}}`), false},
	{"spirit auspex empty scan", subject(`{"name": "m", "designation": "machine-spirit", "auspex": {"scan": " "}, "enact": {"*": "a"}}`), false},
	{"spirit auspex exalted no boolean", subject(`{"name": "m", "designation": "machine-spirit", "exalted": true,
		"auspex": {"scan": "a", "exalted": 1}, "enact": {"*": "a"}}`), false},
	{"spirit privileged auspex of a common subject", subject(`{"name": "m", "designation": "machine-spirit",
		"auspex": {"scan": "a", "exalted": true}, "enact": {"*": "a"}}`), false},
	{"spirit privileged auspex of a subject exalted false", subject(`{"name": "m", "designation": "machine-spirit", "exalted": false,
		"auspex": {"scan": "a", "exalted": true}, "enact": {"*": "a"}}`), false},
	{"spirit enact missing", subject(`{"name": "m", "designation": "machine-spirit", "auspex": {"scan": "a"}}`), false},
	{"spirit enact plain", subject(`{"name": "m", "designation": "machine-spirit", "enact": "a"}`), false},
	{"spirit enact empty map", subject(`{"name": "m", "designation": "machine-spirit", "enact": {}}`), false},
	{"spirit enact empty words", subject(`{"name": "m", "designation": "machine-spirit", "enact": {"*": " "}}`), false},
	{"spirit omen empty", subject(`{"name": "m", "designation": "machine-spirit", "omens": {"*": ""}, "enact": {"*": "a"}}`), false},
	{"spirit omens plain", subject(`{"name": "m", "designation": "machine-spirit", "omens": "a", "enact": {"*": "a"}}`), false},
	{"spirit with a vessel", subject(`{"name": "m", "designation": "machine-spirit", "vessel": "v", "enact": {"*": "a"}}`), false},
	{"spirit relics not exalted", subject(`{"name": "m", "designation": "machine-spirit", "relics": [], "enact": {"*": "a"}}`), false},
	{"spirit exalted map", subject(`{"name": "m", "designation": "machine-spirit", "exalted": {"*": true}, "enact": {"*": "a"}}`), false},
	{"spirit bad patience", subject(`{"name": "m", "designation": "machine-spirit", "patience": "soon", "enact": {"*": "a"}}`), false},
	{"designation missing", subject(`{"name": "m", "enact": {"*": "a"}}`), false},
	{"designation unknown", subject(`{"name": "m", "designation": "daemon", "enact": {"*": "a"}}`), false},
	{"designation of speech", subject(`{"name": "m", "designation": "incantation", "words": "a"}`), false},
	{"name missing", subject(`{"designation": "machine-spirit", "enact": {"*": "a"}}`), false},
	{"name unfit", subject(`{"name": "a b", "designation": "machine-spirit", "enact": {"*": "a"}}`), false},
	{"name no string", subject(`{"name": 1, "designation": "machine-spirit", "enact": {"*": "a"}}`), false},
	{"purpose no string", subject(`{"name": "m", "designation": "machine-spirit", "purpose": 1, "enact": {"*": "a"}}`), false},
	{"unknown subject key", subject(`{"name": "m", "designation": "machine-spirit", "states": [], "enact": {"*": "a"}}`), false},
	{"introit no list", subject(`{"name": "m", "designation": "machine-spirit", "introit": "o", "enact": {"*": "a"}}`), false},
	{"introit number", subject(`{"name": "m", "designation": "machine-spirit", "introit": [1], "enact": {"*": "a"}}`), false},
	{"subject no object", subject(`"m"`), false},

	{"incantation every key", observed(`{"name": "o", "designation": "incantation", "to": ["on"], "from": [], "words": "a",
		"tongue": "sh", "patience": "1s", "reversion": "", "exalted": true, "relics": []}`), true},
	{"litany every key", observed(`{"name": "o", "designation": "litany", "scroll": "s", "offerings": ["", "{{aspect}}"],
		"tongue": "sh", "patience": "1s", "reversion": "r", "exalted": false}`), true},
	{"observance unnamed by any", markII(`, "observances": [{"name": "o", "designation": "incantation", "words": "a"}]`, spirit), true},
	{"words missing", observed(`{"name": "o", "designation": "incantation"}`), false},
	{"words empty", observed(`{"name": "o", "designation": "incantation", "words": " "}`), false},
	{"words map", observed(`{"name": "o", "designation": "incantation", "words": {"*": "a"}}`), false},
	{"scroll missing", observed(`{"name": "o", "designation": "litany"}`), false},
	{"scroll map", observed(`{"name": "o", "designation": "litany", "scroll": {"*": "s"}}`), false},
	{"reversion map", observed(`{"name": "o", "designation": "incantation", "words": "a", "reversion": {"*": ""}}`), false},
	{"offering map", observed(`{"name": "o", "designation": "litany", "scroll": "s", "offerings": [{"*": "a"}]}`), false},
	{"offerings no list", observed(`{"name": "o", "designation": "litany", "scroll": "s", "offerings": "a"}`), false},
	{"to no list", observed(`{"name": "o", "designation": "incantation", "words": "a", "to": "on"}`), false},
	{"to number", observed(`{"name": "o", "designation": "incantation", "words": "a", "to": [1]}`), false},
	{"observance name missing", observed(`{"designation": "incantation", "words": "a"}`), false},
	{"observance of a subject's designation", observed(`{"name": "o", "designation": "tether", "tether": "t"}`), false},
	{"scroll on an incantation", observed(`{"name": "o", "designation": "incantation", "words": "a", "scroll": "s"}`), false},
	{"observance unknown key", observed(`{"name": "o", "designation": "incantation", "words": "a", "seal": "0644"}`), false},
	{"observance relics not exalted", observed(`{"name": "o", "designation": "litany", "scroll": "s", "relics": []}`), false},
	{"observance exalted map", observed(`{"name": "o", "designation": "litany", "scroll": "s", "exalted": {"*": true}}`), false},

	{"edict bare", `{"pattern": "Mark II", "edict": true}`, true},
	{"edict every key", edictII(`, "$schema": "https://example.org/r.json", "purpose": "p", "tongue": "sh", "vox-cast": ["progress"],
		"inscriptions": {"r": {"mandatory": true, "decrees": "x"}, "b": {}}`, `{"designation": "incantation", "words": "a",
		"tongue": "sh", "patience": "2m", "reversion": "b", "exalted": true, "relics": ["r"]},
		{"designation": "litany", "scroll": "s", "offerings": ["{{rite.name}}"], "reversion": ""}`), true},
	{"edict false alone", `{"pattern": "Mark II", "edict": false}`, false},
	{"edict decree map", edictII(`, "inscriptions": {"r": {"decrees": {"*": "x"}}}`, ``), false},
	{"edict subjects", edictII(`, "subjects": []`, ``), false},
	{"edict observances", edictII(`, "observances": []`, ``), false},
	{"edict aspects", edictII(`, "aspects": ["on"]`, ``), false},
	{"edict auspex", edictII(`, "auspex": "cat x"`, ``), false},
	{"edict liturgy no list", `{"pattern": "Mark II", "edict": true, "liturgy": {}}`, false},
	{"entry named", edictII(``, `{"designation": "incantation", "words": "a", "name": "n"}`), false},
	{"entry guarded", edictII(``, `{"designation": "incantation", "words": "a", "to": []}`), false},
	{"entry designation missing", edictII(``, `{"words": "a"}`), false},
	{"entry vox-cast", edictII(``, `{"designation": "vox-cast"}`), false},
	{"entry words map", edictII(``, `{"designation": "incantation", "words": {"*": "a"}}`), false},
	{"entry words missing", edictII(``, `{"designation": "incantation"}`), false},
	{"entry Mark I step", edictII(``, `{"incantation": "a"}`), false},
	{"entry relics not exalted", edictII(``, `{"designation": "litany", "scroll": "s", "relics": []}`), false},
}

// markIIBeyondTheSchema are heresies of pattern Mark II that only the
// librarium can tell, each with the reason no schema can.
var markIIBeyondTheSchema = []struct{ name, scripture, why string }{
	{"omens of an undeclared aspect", subject(`{"name": "t", "designation": "tether", "tether": "t", "omens": {"of": "a", "*": null}}`),
		"whether a key is a declared aspect is told by the aspects, beside the map"},
	{"omens leaving an aspect unserved", subject(`{"name": "t", "designation": "tether", "tether": "t", "omens": {"on": "a"}}`),
		"whether every aspect is served is told by the aspects, beside the map"},
	{"enact of an undeclared omen", subject(`{"name": "m", "designation": "machine-spirit", "omens": {"on": "1", "off": "0"},
		"enact": {"2": "a", "*": "b"}}`), "an omen is declared by the subject's omens, rendered for each aspect, beside the map"},
	{"enact leaving an omen unserved", subject(`{"name": "m", "designation": "machine-spirit", "omens": {"on": "1", "off": "0"},
		"enact": {"1": "a"}}`), "an omen is declared by the subject's omens, rendered for each aspect, beside the map"},
	{"trials of an undeclared omen", subject(`{"name": "m", "designation": "machine-spirit", "auspex": {"trials": {"2": "true"}},
		"enact": {"*": "a"}}`), "an omen is declared by the subject's omens, or by the aspects, beside the map"},
	{"subject named twice", markII(``, spirit+`, `+spirit), "a name is judged against every other subject's"},
	{"observance named twice", observed(`{"name": "o", "designation": "incantation", "words": "a"},
		{"name": "o", "designation": "incantation", "words": "b"}`), "a name is judged against every other observance's"},
	{"introit of no observance", subject(`{"name": "m", "designation": "machine-spirit", "enact": {"*": "a"}, "introit": ["nope"]}`),
		"an introit names an observance declared elsewhere in the rite"},
	{"introit and recessional", markII(`, "observances": [{"name": "o", "designation": "incantation", "words": "a"}]`,
		`{"name": "x", "designation": "machine-spirit", "enact": {"*": "a"}, "introit": ["o"]},
		{"name": "y", "designation": "machine-spirit", "enact": {"*": "a"}, "recessional": ["o"]}`),
		"where else an observance is named is told by the other subjects"},
	{"guard of an undeclared aspect", observed(`{"name": "o", "designation": "incantation", "words": "a", "to": ["of"]}`),
		"whether a guard names a declared aspect is told by the aspects, beside it"},
	{"omen spoken in a scan", subject(`{"name": "m", "designation": "machine-spirit", "auspex": {"scan": "cat {{omen}}"}, "enact": {"*": "a"}}`),
		"placeholders are words within a string, which a schema does not read"},
	{"former in exalted enact", subject(`{"name": "m", "designation": "machine-spirit", "exalted": true, "enact": {"*": "echo {{former}}"}}`),
		"placeholders are words within a string, which a schema does not read"},
	{"aspect in an edict's words", edictII(``, `{"designation": "incantation", "words": "echo {{aspect}}"}`),
		"placeholders are words within a string, which a schema does not read"},
	{"exalted vessel not rooted", subject(`{"name": "c", "designation": "transcription", "exalted": true, "vessel": "{{aspect}}/x",
		"omens": {"*": null}}`), "a vessel is judged as it is rendered for each aspect, which a schema cannot do"},
	{"exalted vessel of the kernel", subject(`{"name": "c", "designation": "transcription", "exalted": true, "vessel": "/etc/../sys/x",
		"omens": {"*": "1"}}`), "a vessel is judged as it is rendered and its path made plain, which a schema cannot do"},
	{"relic above the hall", subject(`{"name": "m", "designation": "machine-spirit", "exalted": true, "relics": ["a/../../r"],
		"enact": {"*": "a"}}`), "a relic is judged once its path is made plain, which a schema cannot do"},
}

func TestRiteSchema_JudgesMarkIIAsTheLibrariumJudges(t *testing.T) {
	s := compile(t, Rite)
	for _, c := range markIIJudgements {
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

func TestRiteSchema_LeavesBeyondItWhatOnlyTheLibrariumTellsOfMarkII(t *testing.T) {
	s := compile(t, Rite)
	for _, c := range markIIBeyondTheSchema {
		t.Run(c.name, func(t *testing.T) {
			if c.why == "" {
				t.Fatal("a heresy beyond the schema must say why")
			}
			if _, fs := librarium.Parse("r", "/lib/rites/r.json", []byte(c.scripture)); !fs.Heretical() {
				t.Fatalf("the librarium finds %s pure", c.scripture)
			}
			if err := validate(t, s, []byte(c.scripture)); err != nil {
				t.Fatalf("the schema judges %s after all (%v); judge it among the judgements", c.scripture, err)
			}
		})
	}
}
