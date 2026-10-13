package librarium

import (
	"strings"
	"testing"
)

// markII wraps subjects, observances and extra rite keys into a rite of
// pattern Mark II with the aspects "on" and "off" and the inscription
// "reason"; observances "" writes none.
func markII(extra, subjects, observances string) string {
	obs := ""
	if observances != "" {
		obs = `,
  "observances": [` + observances + `]`
	}
	return `{
  "pattern": "Mark II",
  "aspects": ["on", "off"],
  "inscriptions": {"reason": {"purpose": "why"}},` + extra + `
  "subjects": [` + subjects + `]` + obs + `
}`
}

// edictII wraps a liturgy and extra keys into an edict of pattern Mark II.
func edictII(extra, liturgy string) string {
	return `{
  "pattern": "Mark II",
  "edict": true,
  "inscriptions": {"region": {"decrees": "chosen"}},` + extra + `
  "liturgy": [` + liturgy + `]
}`
}

// Subjects and observances that are pure on their own.
const (
	tetherT = `{"name": "t", "designation": "tether", "tether": "t", "omens": {"on": "a", "*": null}}`
	spiritM = `{"name": "m", "designation": "machine-spirit", "omens": {"on": "1", "off": "0"},
     "auspex": {"scan": "cat m"}, "enact": {"*": "echo {{omen}} > m"}}`
	spiritX = `{"name": "x", "designation": "machine-spirit", "auspex": {"scan": "cat x"}, "enact": {"*": "x"}`
)

var markIIHeresyCases = []heresyCase{
	// The rite.
	{"rite of aspects with liturgy", markII(`"liturgy": [],`, tetherT, ``), Heresy,
		`a rite of aspects of pattern "Mark II" bears no "liturgy"`, `"liturgy"`, 0},
	{"rite with auspex", markII(`"auspex": "cat x",`, tetherT, ``), Heresy,
		`bears no "auspex" of its own`, `"auspex"`, 0},
	{"edict with auspex", edictII(`"auspex": "cat x",`, ``), Heresy, `bears no "auspex" of its own`, `"auspex"`, 0},
	{"edict with subjects", edictII(`"subjects": [],`, ``), Heresy, `an edict bears no "subjects"`, `"subjects"`, 0},
	{"edict with observances", edictII(`"observances": [],`, ``), Heresy, `an edict bears no "observances"`, `"observances"`, 0},
	{"edict with aspects", edictII(`"aspects": ["on"],`, ``), Heresy, `bears both "aspects" and "edict": true`, `true`, 0},
	{"empty subjects", markII(``, ``, ``), Heresy, `the rite declares no subjects`, `[]`, 0},
	{"subjects missing", `{"pattern": "Mark II", "aspects": ["on"]}`, Heresy, `the rite declares no "subjects"`, `{`, 0},
	{"subjects no list", `{"pattern": "Mark II", "aspects": ["on"], "subjects": {}}`, Heresy,
		`the "subjects" of a rite must be written as a list of subjects`, `{}`, 0},
	{"unknown rite key", markII(`"steps": [],`, tetherT, ``), Heresy,
		`the key "steps" is not written in the codex for a rite; strike it, or write one of the keys the codex knows there: "pattern", "purpose", "aspects", "edict", "inscriptions", "tongue", "vox-cast", "subjects", "observances", "liturgy", "$schema"`,
		`"steps"`, 0},
	{"vox-cast unknown", markII(`"vox-cast": ["failure"],`, tetherT, ``), Heresy, `"failure" is no tiding the codex knows`, `"failure"`, 0},
	{"vox-cast twice", markII(`"vox-cast": ["success", "progress", "success"],`, tetherT, ``), Heresy,
		`the vox-cast "success" is written twice`, `"success"]`, 0},
	{"vox-cast no list", edictII(`"vox-cast": "success",`, ``), Heresy,
		`the "vox-cast" of a rite must be written as a list of "progress" and "success"`, `"success"`, 0},
	{"plain decree", `{"pattern": "Mark II", "aspects": ["on"], "inscriptions": {"m": {"decrees": "x"}}, "subjects": [` + tetherT + `]}`,
		Heresy, `the "decrees" of the inscription "m" must be written as an object keyed by aspect`, `"x"`, 0},
	{"decree undeclared aspect", `{"pattern": "Mark II", "aspects": ["on"], "inscriptions": {"m": {"decrees": {"of": "x"}}}, "subjects": [` + tetherT + `]}`,
		Heresy, `names the aspect "of", which the rite never declared`, `"of"`, 0},
	{"edict decree map", `{"pattern": "Mark II", "edict": true, "inscriptions": {"m": {"decrees": {"*": "x"}}}}`, Heresy,
		`yet an edict bears no aspect to key it by`, `{"*": "x"}`, 0},

	// Subjects.
	{"designation missing", markII(``, `{"name": "x"}`, ``), Heresy,
		`the subject "x" bears no "designation"; it must name the shape it takes, one of "sanctum", "transcription", "tether", "machine-spirit"`,
		`{"name": "x"}`, 0},
	{"designation unknown", markII(``, `{"name": "x", "designation": "daemon"}`, ``), Heresy,
		`"daemon" is no designation the codex knows for the subject "x"`, `"daemon"`, 0},
	{"designation of speech", markII(``, `{"name": "x", "designation": "incantation", "words": "a"}`, ``), Heresy,
		`"incantation" is no designation the codex knows for the subject "x"`, `"incantation"`, 0},
	{"name missing", markII(``, `{"designation": "tether", "tether": "t", "omens": {"*": null}}`, ``), Heresy,
		`subject 1 of the rite bears no "name"`, `{"designation"`, 0},
	{"name unfit", markII(``, `{"name": "a b", "designation": "tether", "tether": "t", "omens": {"*": null}}`, ``), Heresy,
		`"a b" is no fit name for a subject`, `"a b"`, 0},
	{"name twice", markII(``, tetherT+`, `+strings.Replace(tetherT, `"tether": "t"`, `"tether": "u"`, 1), ``), Heresy,
		`the subject "t" is named twice`, `"t", "designation": "tether", "tether": "u"`, 0},
	{"spirit key on a tether", markII(``, `{"name": "t", "designation": "tether", "tether": "t", "omens": {"*": null}, "enact": {"*": "a"}}`, ``),
		Heresy, `the subject "t" is a tether, which the servitor reads and sets by itself; it bears no "enact"`, `"enact"`, 0},
	{"sanctum key on a tether", markII(``, `{"name": "t", "designation": "tether", "tether": "t", "omens": {"*": null}, "ward": "w"}`, ``),
		Heresy, `the key "ward" belongs to a sanctum, yet the subject "t" is a tether`, `"ward"`, 0},
	{"vessel on a spirit", markII(``, `{"name": "x", "designation": "machine-spirit", "vessel": "v", "enact": {"*": "a"}}`, ``),
		Heresy, `the key "vessel" belongs to a sanctum, yet the subject "x" is a machine-spirit`, `"vessel"`, 0},
	{"unknown subject key", markII(``, `{"name": "t", "designation": "tether", "tether": "t", "omens": {"*": null}, "anchor": "a"}`, ``),
		Heresy, `the key "anchor" is not written in the codex for the subject "t"; strike it, or write one of the keys the codex knows there: "name", "designation", "purpose", "omens", "exalted", "introit", "recessional", "tether", "zeal"`,
		`"anchor"`, 0},
	{"omens missing", markII(``, `{"name": "t", "designation": "tether", "tether": "t"}`, ``), Heresy,
		`the subject "t" holds no "omens"`, `{"name": "t"`, 0},
	{"omens of one value", markII(``, `{"name": "t", "designation": "tether", "tether": "t", "omens": "a"}`, ``), Heresy,
		`the "omens" of the subject "t" must be written as an object keyed by aspect, with "*" serving every aspect it does not name, yet here stands a string`,
		`"a"}`, 0},
	{"omens undeclared aspect", markII(``, `{"name": "t", "designation": "tether", "tether": "t", "omens": {"of": "a", "*": null}}`, ``),
		Heresy, `the "omens" of the subject "t" names the aspect "of", which the rite never declared`, `"of"`, 0},
	{"omens unserved", markII(``, `{"name": "t", "designation": "tether", "tether": "t", "omens": {"on": "a"}}`, ``),
		Heresy, `the "omens" of the subject "t" gives no value for the aspect "off" and holds no "*"`, `{"on": "a"}`, 0},
	{"null sanctum omen", markII(``, `{"name": "s", "designation": "sanctum", "vessel": "v.kdl", "omens": {"*": null}}`, ``),
		Heresy, "the scripture of a sanctum may not be null", `null`, 0},
	{"vessel missing", markII(``, `{"name": "s", "designation": "sanctum", "omens": {"*": ""}}`, ``), Heresy,
		`the subject "s" holds no "vessel"`, `{"name": "s"`, 0},
	{"tether missing", markII(``, `{"name": "t", "designation": "tether", "omens": {"*": null}}`, ``), Heresy,
		`the subject "t" holds no "tether"`, `{"name": "t"`, 0},
	{"bad seal", markII(``, `{"name": "c", "designation": "transcription", "vessel": "v", "omens": {"*": null}, "seal": "rw"}`, ``),
		Heresy, `"rw" is no seal`, `"rw"`, 0},
	{"spaced ward", markII(``, `{"name": "s", "designation": "sanctum", "vessel": "v.kdl", "ward": "a b", "omens": {"*": ""}}`, ``),
		Heresy, `the ward "a b" is unfit`, `"a b"`, 0},
	{"enact missing", markII(``, `{"name": "x", "designation": "machine-spirit", "auspex": {"scan": "a"}}`, ``), Heresy,
		`the subject "x" holds no "enact"`, `{"name": "x"`, 0},
	{"enact undeclared omen", markII(``, strings.Replace(spiritM, `{"*": "echo {{omen}} > m"}`, `{"2": "a", "*": "b"}`, 1), ``),
		Heresy, `the "enact" of the subject "m" names the omen "2", which the subject "m" never declared (it declares "1", "0")`, `"2"`, 0},
	{"enact unserved", markII(``, strings.Replace(spiritM, `{"*": "echo {{omen}} > m"}`, `{"1": "a"}`, 1), ``),
		Heresy, `the "enact" of the subject "m" gives no value for the omen "0" and holds no "*"`, `{"1": "a"}`, 0},
	{"enact named by aspects", markII(``, `{"name": "x", "designation": "machine-spirit", "enact": {"of": "a", "*": "b"}}`, ``),
		Heresy, `names the omen "of", which the subject "x" never declared (it declares "on", "off")`, `"of"`, 0},
	{"enact of one value", markII(``, `{"name": "x", "designation": "machine-spirit", "enact": "a"}`, ``),
		Heresy, `the "enact" of the subject "x" must be written as an object keyed by omen`, `"a"}`, 0},
	{"omen empty", markII(``, `{"name": "x", "designation": "machine-spirit", "omens": {"*": " "}, "enact": {"*": "a"}}`, ``),
		Heresy, `may not be empty`, `" "`, 0},
	{"trials undeclared omen", markII(``, strings.Replace(spiritM, `{"scan": "cat m"}`, `{"trials": {"2": "true"}}`, 1), ``),
		Heresy, `the "trials" of the "auspex" of the subject "m" names the omen "2"`, `"2"`, 0},
	{"auspex of neither", markII(``, strings.Replace(spiritM, `{"scan": "cat m"}`, `{"patience": "1s"}`, 1), ``),
		Heresy, `the "auspex" of the subject "m" holds neither "scan" nor "trials"`, `{"patience"`, 0},
	{"auspex unknown key", markII(``, strings.Replace(spiritM, `{"scan": "cat m"}`, `{"rite": "cat m"}`, 1), ``),
		Heresy, `the key "rite" is not written in the codex for the "auspex" of the subject "m"; strike it, or write one of the keys the codex knows there: "scan", "trials", "patience", "exalted"`,
		`"rite"`, 0},
	{"auspex of words", markII(``, strings.Replace(spiritM, `{"scan": "cat m"}`, `"cat m"`, 1), ``),
		Heresy, `the "auspex" of the subject "m" must be written as an object`, `"cat m"`, 0},
	{"auspex bad patience", markII(``, strings.Replace(spiritM, `{"scan": "cat m"}`, `{"scan": "cat m", "patience": "soon"}`, 1), ``),
		Heresy, `"soon" is no measure of patience`, `"soon"`, 0},
	{"privileged auspex of a common subject", markII(``, strings.Replace(spiritM, `{"scan": "cat m"}`, `{"scan": "cat m", "exalted": true}`, 1), ``),
		Heresy, `the "auspex" of the subject "m" is exalted, yet the subject "m" is not`, `"exalted"`, 0},
	{"relics of a common spirit", markII(``, strings.Replace(spiritM, `"enact"`, `"relics": ["r"], "enact"`, 1), ``),
		Heresy, `the subject "m" names "relics", yet it is not exalted`, `"relics"`, 0},
	{"exalted aspect map", markII(``, `{"name": "x", "designation": "machine-spirit", "exalted": {"*": true}, "enact": {"*": "a"}}`, ``),
		Heresy, `a subject is exalted for every aspect or for none`, `{"*": true}`, 0},
	{"exalted no boolean", markII(``, `{"name": "x", "designation": "machine-spirit", "exalted": "yes", "enact": {"*": "a"}}`, ``),
		Heresy, `the "exalted" of the subject "x" must be written as true or false`, `"yes"`, 0},
	{"spirit bad tongue", markII(``, `{"name": "x", "designation": "machine-spirit", "tongue": "", "enact": {"*": "a"}}`, ``),
		Heresy, `the "tongue" of the subject "x" may not be empty`, `""`, 0},
	{"introit no list", markII(``, strings.Replace(tetherT, `"omens"`, `"introit": "o", "omens"`, 1), ``),
		Heresy, `the "introit" of the subject "t" must be written as a list of the names of observances`, `"o"`, 0},

	// Placeholders.
	{"omen in a scan", markII(``, strings.Replace(spiritM, `"cat m"`, `"cat {{omen}}"`, 1), ``), Heresy,
		`{{omen}} names the omen a machine-spirit is enacted into or tried for`, `{{omen}}"`, 0},
	{"omen in a vessel", markII(``, `{"name": "c", "designation": "transcription", "vessel": "{{omen}}", "omens": {"*": null}}`, ``),
		Heresy, `{{omen}} names the omen`, `{{omen}}`, 0},
	{"omen in omens", markII(``, `{"name": "x", "designation": "machine-spirit", "omens": {"*": "{{omen}}"}, "enact": {"*": "a"}}`, ``),
		Heresy, `{{omen}} names the omen`, `{{omen}}`, 0},
	{"omen in an observance", markII(``, strings.Replace(tetherT, `"omens"`, `"introit": ["o"], "omens"`, 1),
		`{"name": "o", "designation": "incantation", "words": "echo {{omen}}"}`), Heresy, `{{omen}} names the omen`, `{{omen}}`, 0},
	{"omen in an edict", edictII(``, `{"designation": "incantation", "words": "echo {{omen}}"}`), Heresy,
		`{{omen}} names the omen`, `{{omen}}`, 0},
	{"unknown placeholder in an anchor", markII(``, `{"name": "t", "designation": "tether", "tether": "t", "omens": {"*": "{{stat}}"}}`, ``),
		Heresy, "{{stat}} is no placeholder known", `{{stat}}`, 0},
	{"undeclared inscription in enact", markII(``, strings.Replace(spiritM, `> m"`, `{{inscription.mood}}"`, 1), ``),
		Heresy, `inscription "mood", which this rite never declared`, `{{inscription.mood}}`, 0},
	{"aspect in an edict", edictII(``, `{"designation": "litany", "scroll": "s", "offerings": ["{{aspect}}"]}`), Heresy,
		"yet an edict bears no aspect", `{{aspect}}`, 0},
	{"former in exalted enact", markII(``, `{"name": "x", "designation": "machine-spirit", "exalted": true,
     "auspex": {"scan": "cat /x"}, "enact": {"*": "echo {{former}} > /x"}}`, ``), Heresy, `"$SERVITOR_FORMER_ASPECT"`, `{{former}}`, 0},
	{"fleeting inscription in exalted omens", `{"pattern": "Mark II", "aspects": ["on", "off"],
  "inscriptions": {"m": {"mandatory": true, "decrees": {"on": "1"}}},
  "subjects": [{"name": "t", "designation": "transcription", "exalted": true, "vessel": "/etc/x",
    "omens": {"*": "{{inscription.m}}"}}]}`, Heresy, `"$SERVITOR_INSCRIPTION_M"`, `{{inscription.m}}`, 0},
	{"former in a privileged scan", markII(``, `{"name": "x", "designation": "machine-spirit", "exalted": true,
     "auspex": {"scan": "cat {{former}}", "exalted": true}, "enact": {"*": "a"}}`, ``), Heresy, `"$SERVITOR_FORMER_ASPECT"`, `{{former}}`, 0},
	{"exalted tether not rooted", markII(``, `{"name": "t", "designation": "tether", "exalted": true, "tether": "t", "omens": {"*": null}}`, ``),
		Heresy, `the bound name t of the exalted tether "t" does not begin at the root`, `"t", "omens"`, 0},
	{"exalted vessel of the kernel", markII(``, `{"name": "c", "designation": "transcription", "exalted": true,
     "vessel": "/sys/x", "omens": {"*": "1"}}`, ``), Heresy, `/sys/x lies beneath /sys`, `"/sys/x"`, 0},

	// Observances and their references.
	{"unknown introit", markII(``, strings.Replace(tetherT, `"omens"`, `"introit": ["nope"], "omens"`, 1), ``), Heresy,
		`the subject "t" names the observance "nope" in its "introit", yet the rite declares no observance of that name (it declares none)`,
		`"nope"`, 0},
	{"unknown recessional", markII(``, strings.Replace(tetherT, `"omens"`, `"recessional": ["nope"], "omens"`, 1),
		`{"name": "o", "designation": "incantation", "words": "a"}`), Heresy,
		`names the observance "nope" in its "recessional", yet the rite declares no observance of that name (it declares "o")`, `"nope"`, 0},
	{"introit and recessional", markII(``, strings.Replace(tetherT, `"omens"`, `"introit": ["o"], "omens"`, 1)+`, `+
		strings.Replace(spiritX, `"enact"`, `"recessional": ["o"], "enact"`, 1)+`}`,
		`{"name": "o", "designation": "incantation", "words": "a"}`), Heresy,
		`the observance "o" is named as an introit and as a recessional`, `"o"], "enact"`, 0},
	{"observance designation missing", markII(``, tetherT, `{"name": "o", "words": "a"}`), Heresy,
		`the observance "o" bears no "designation"; it must name the shape it takes, one of "incantation", "litany"`, `{"name": "o"`, 0},
	{"observance designation of a subject", markII(``, tetherT, `{"name": "o", "designation": "tether"}`), Heresy,
		`"tether" is no designation the codex knows for the observance "o"`, `"tether"}`, 0},
	{"observance name missing", markII(``, tetherT, `{"designation": "incantation", "words": "a"}`), Heresy,
		`observance 1 of the rite bears no "name"`, `{"designation"`, 0},
	{"observance named twice", markII(``, tetherT, `{"name": "o", "designation": "incantation", "words": "a"},
    {"name": "o", "designation": "incantation", "words": "b"}`), Heresy, `the observance "o" is named twice`, `"o", "designation": "incantation", "words": "b"`, 0},
	{"observance words map", markII(``, tetherT, `{"name": "o", "designation": "incantation", "words": {"on": "w", "*": "b"}}`), Heresy,
		`the "words" of the observance "o" is written as an aspect map, yet the words of an observance are one for every aspect`, `{"on": "w"`, 0},
	{"observance reversion map", markII(``, tetherT, `{"name": "o", "designation": "incantation", "words": "a", "reversion": {"*": ""}}`), Heresy,
		`the "reversion" of the observance "o" is written as an aspect map`, `{"*": ""}`, 0},
	{"observance offering map", markII(``, tetherT, `{"name": "o", "designation": "litany", "scroll": "s", "offerings": [{"*": "a"}]}`), Heresy,
		`offering 1 of the observance "o" is written as an aspect map`, `{"*": "a"}`, 0},
	{"words missing", markII(``, tetherT, `{"name": "o", "designation": "incantation"}`), Heresy,
		`the observance "o" holds no "words"`, `{"name": "o"`, 0},
	{"scroll missing", markII(``, tetherT, `{"name": "o", "designation": "litany"}`), Heresy,
		`the observance "o" holds no "scroll"`, `{"name": "o"`, 0},
	{"litany key on an incantation", markII(``, tetherT, `{"name": "o", "designation": "incantation", "words": "a", "scroll": "s"}`), Heresy,
		`the key "scroll" belongs to a litany, yet the observance "o" is an incantation; strike it, or write the keys the codex knows for an incantation: "name", "designation", "to", "from", "words", "tongue", "patience", "reversion", "exalted", "relics"`,
		`"scroll"`, 0},
	{"to undeclared aspect", markII(``, tetherT, `{"name": "o", "designation": "incantation", "words": "a", "to": ["of"]}`), Heresy,
		`the "to" of the observance "o" names the aspect "of", which the rite never declared`, `"of"`, 0},
	{"from undeclared aspect", markII(``, tetherT, `{"name": "o", "designation": "incantation", "words": "a", "from": ["on", "of"]}`), Heresy,
		`the "from" of the observance "o" names the aspect "of"`, `"of"`, 0},
	{"to no list", markII(``, tetherT, `{"name": "o", "designation": "incantation", "words": "a", "to": "on"}`), Heresy,
		`the "to" of the observance "o" must be written as a list of declared aspects`, `"on"}`, 0},
	{"observance relics not exalted", markII(``, tetherT, `{"name": "o", "designation": "litany", "scroll": "s", "relics": ["r"]}`), Heresy,
		`the observance "o" names "relics", yet it is not exalted`, `"relics"`, 0},
	{"exalted observance speaks former", markII(``, tetherT, `{"name": "o", "designation": "incantation", "exalted": true, "words": "echo {{former}}"}`),
		Heresy, `"$SERVITOR_FORMER_ASPECT"`, `{{former}}`, 0},

	// An edict's entries.
	{"entry designation missing", edictII(``, `{"words": "a"}`), Heresy,
		`verse 1 of the liturgy bears no "designation"`, `{"words"`, 0},
	{"entry named", edictII(``, `{"designation": "incantation", "words": "a", "name": "x"}`), Heresy,
		`the key "name" is not written in the codex for the incantation of verse 1`, `"name"`, 0},
	{"entry guarded", edictII(``, `{"designation": "incantation", "words": "a", "to": ["on"]}`), Heresy,
		`the key "to" is not written in the codex for the incantation of verse 1`, `"to"`, 0},
	{"entry words map", edictII(``, `{"designation": "incantation", "words": {"*": "a"}}`), Heresy,
		`the "words" of the incantation of verse 1 is written as an aspect map, yet an edict bears no aspect`, `{"*": "a"}`, 0},
	{"entry no object", edictII(``, `"echo"`), Heresy, `verse 1 of the liturgy must be written as an object`, `"echo"`, 0},
	{"entry vox-cast", edictII(``, `{"designation": "vox-cast"}`), Heresy,
		`"vox-cast" is no designation the codex knows for verse 1 of the liturgy`, `"vox-cast"}`, 0},
	{"liturgy no list", edictII(``, ``)[:0] + `{"pattern": "Mark II", "edict": true, "liturgy": {}}`, Heresy,
		`the "liturgy" of an edict must be written as a list of incantations and litanies`, `{}`, 0},

	// Impurities.
	{"observance never named", markII(``, tetherT, `{"name": "o", "designation": "incantation", "words": "a"}`), Impurity,
		`the observance "o" is named by no subject`, `"o"`, 0},
	{"spirit without auspex", markII(``, `{"name": "x", "designation": "machine-spirit", "enact": {"*": "a"}}`, ``), Impurity,
		`the machine-spirit "x" bears no "auspex": it cannot be read`, `"x"`, 0},
	{"exalted spirit without auspex", markII(``, `{"name": "x", "designation": "machine-spirit", "exalted": true, "enact": {"*": "a"}}`, ``),
		Impurity, `give it an auspex marked "exalted": true`, `"x"`, 0},
	{"omens alike", markII(``, `{"name": "t", "designation": "tether", "tether": "t", "omens": {"on": "a", "*": "a"}}`, ``),
		Impurity, `the omens of the subject "t" are alike for every aspect`, `{"on": "a", "*": "a"}`, 0},
	{"spirit omens alike", markII(``, `{"name": "x", "designation": "machine-spirit", "omens": {"*": "1"},
     "auspex": {"scan": "cat x"}, "enact": {"*": "a"}}`, ``), Impurity, `the omens of the subject "x" are alike`, `{"*": "1"}`, 0},
	{"intercessor in common enact", markII(``, `{"name": "x", "designation": "machine-spirit", "auspex": {"scan": "cat x"},
     "enact": {"on": "a", "*": "sudo b"}}`, ``), Impurity,
		`the subject "x" calls upon the intercessor "sudo" within words that are not exalted: such words ask for the master's proof by themselves, fall where no terminal attends, and leave that proof lingering for whatever follows — strike "sudo" and write "exalted": true upon the subject`,
		`"sudo b"`, 0},
	{"guarded call beside exalted", markII(``, strings.Replace(tetherT, `"omens"`, `"introit": ["o"], "omens"`, 1)+`, `+
		`{"name": "x", "designation": "machine-spirit", "exalted": true, "auspex": {"scan": "cat /x"}, "enact": {"*": "a"}}`,
		`{"name": "o", "designation": "incantation", "words": "systemctl restart x"}`), Impurity,
		`the observance "o" calls "systemctl", which the machine guards by an intercessor of its own`, `"systemctl restart x"`, 0},
	{"exalted observance reaches home", markII(``, strings.Replace(tetherT, `"omens"`, `"introit": ["o"], "omens"`, 1),
		`{"name": "o", "designation": "incantation", "exalted": true, "words": "cp ~/x /etc/x", "reversion": "rm /etc/x"}`), Impurity,
		`the exalted speech of the observance "o" reaches into the home of its invoker`, `"cp ~/x`, 0},
	{"exalted entry reaches beside", edictII(``, `{"designation": "incantation", "exalted": true, "words": "a", "reversion": "source ./undo"}`),
		Impurity, `the exalted speech of verse 1 reaches for undo, which is not among its relics`, `"source ./undo"`, 0},
}

func TestParse_MarkIIDenouncesWithPosition(t *testing.T) {
	for _, c := range markIIHeresyCases {
		t.Run(c.name, func(t *testing.T) {
			_, fs := Parse("rite", "/lib/rites/rite.json", []byte(c.scripture))
			want := posOf(t, c.scripture, c.at, c.delta)
			for _, f := range fs {
				if strings.Contains(f.Message, c.message) {
					if f.Severity != c.sev || f.Position != want || f.Scripture != "/lib/rites/rite.json" || f.Rite != "rite" {
						t.Fatalf("finding %v; want %s at %v", f, c.sev, want)
					}
					return
				}
			}
			t.Fatalf("no finding containing %q in\n%v", c.message, fs)
		})
	}
}

func TestParse_MarkIIDenunciationsSpeakGrimdark(t *testing.T) {
	for _, c := range markIIHeresyCases {
		_, fs := Parse("rite", "/lib/rites/rite.json", []byte(c.scripture))
		for _, f := range fs {
			if m := plainGlosses.FindString(f.Message); m != "" {
				t.Errorf("%s: %q speaks the plain word %q", c.name, f.Message, m)
			}
		}
	}
}

// TestParse_MarkIIImpuritiesAloneLeaveTheRiteFitToInvoke: an impurity is
// noted, yet the rite is still read whole.
func TestParse_MarkIIImpuritiesAloneLeaveTheRiteFitToInvoke(t *testing.T) {
	for _, c := range markIIHeresyCases {
		if c.sev != Impurity {
			continue
		}
		if r, fs := Parse("rite", "/lib/rites/rite.json", []byte(c.scripture)); r == nil || fs.Heretical() {
			t.Errorf("%s: rite %v, findings:\n%v", c.name, r, fs)
		}
	}
}

func TestParseSettings_MarkIIIsRead(t *testing.T) {
	s, fs := ParseSettings(settingsPath, []byte(`{"pattern": "Mark II", "tongue": "fish", "vox": "off"}`))
	if len(fs) != 0 {
		t.Fatalf("findings:\n%v", fs)
	}
	if *s != (Settings{Path: settingsPath, Tongue: "fish", Vox: VoxOff, MarkII: true}) {
		t.Fatalf("settings %+v", *s)
	}
	if s, _ := ParseSettings(settingsPath, []byte(`{"pattern": "Mark I"}`)); s.MarkII {
		t.Fatal("settings of Mark I are told Mark II")
	}
}
