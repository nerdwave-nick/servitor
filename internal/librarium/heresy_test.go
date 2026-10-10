package librarium

import (
	"regexp"
	"strings"
	"testing"
)

// mark wraps a liturgy and extra rite keys into a scripture with the aspects
// "on" and "off" and the inscription "reason".
func mark(extra, liturgy string) string {
	return `{
  "pattern": "Mark I",
  "aspects": ["on", "off"],
  "inscriptions": {"reason": {"purpose": "why"}},` + extra + `
  "liturgy": [` + liturgy + `]
}`
}

type heresyCase struct {
	name      string
	scripture string
	sev       Severity
	message   string // substring of the finding's message
	at        string // the finding points at the first occurrence of at …
	delta     int    // … plus delta bytes
}

var heresyCases = []heresyCase{
	{"syntax", `{"pattern": "Mark I",, }`, Heresy, "malformed beyond reading", `,, }`, 1},
	{"scripture no object", `["Mark I"]`, Heresy, "must be written as one object", `[`, 0},
	{"pattern malformed", `{"pattern": "Mark 1"}`, Heresy, "numerals of the old tongue", `"Mark 1"`, 0},
	{"pattern no string", `{"pattern": 1}`, Heresy, "numerals of the old tongue", `1}`, 0},
	{"pattern object", `{"pattern": {"mark": 1}}`, Heresy, "numerals of the old tongue", `{"mark"`, 0},
	{"pattern list", `{"pattern": ["Mark I"]}`, Heresy, "numerals of the old tongue", `["Mark I"]`, 0},
	{"pattern bare mark", `{"pattern": "Mark "}`, Heresy, "numerals of the old tongue", `"Mark "`, 0},
	{"pattern unknown mark", `{"pattern": "Mark II"}`, Heresy, `forged to read only "Mark I"`, `"Mark II"`, 0},
	{"unknown key of rite", mark(`"files": [],`, ``), Heresy, `the key "files" is not written in the codex for a rite`, `"files"`, 0},
	{"unknown key of step", mark(``, `{"tether": "t", "anchor": "a", "ward": "w"}`), Heresy,
		`the key "ward" is not written in the codex for the tether of verse 1`, `"ward"`, 0},
	{"unknown key of inscription", mark(``, ``) + "", Heresy, "", "", 0}, // replaced below
	{"key written twice", mark(`"purpose": "a", "purpose": "b",`, ``), Heresy, `the key "purpose" is written twice`, `"purpose": "b"`, 0},
	{"wrong form", mark(`"purpose": 7,`, ``), Heresy, `the "purpose" of a rite must be written as a string, yet here stands a number`, `7,`, 0},
	{"invalid rite name", ``, Heresy, "no fit name for a rite", `{`, 0}, // see TestParse_InvalidRiteName
	{"aspects missing", `{"pattern": "Mark I", "liturgy": []}`, Heresy, `declares no "aspects"`, `{`, 0},
	{"aspects empty", `{"pattern": "Mark I", "aspects": []}`, Heresy, "declares no aspects", `[]`, 0},
	{"aspects no list", `{"pattern": "Mark I", "aspects": "on"}`, Heresy, "must be written as a list of names", `"on"`, 0},
	{"aspect twice", `{"pattern": "Mark I", "aspects": ["on", "on"]}`, Heresy, `the aspect "on" is declared twice`, `"on"]`, 0},
	{"aspect fallback", `{"pattern": "Mark I", "aspects": ["*"]}`, Heresy, `"*" is no fit name for an aspect`, `"*"`, 0},
	{"aspect spaced", `{"pattern": "Mark I", "aspects": ["o n"]}`, Heresy, `"o n" is no fit name for an aspect`, `"o n"`, 0},
	{"map key undeclared", mark(``, `{"incantation": {"on": "a", "of": "b", "*": "c"}}`), Heresy,
		`names the aspect "of", which the rite never declared`, `"of"`, 0},
	{"map without value", mark(``, `{"tether": "t", "anchor": {"on": "a"}}`), Heresy,
		`the "anchor" of the tether of verse 1 gives no value for the aspect "off" and holds no "*"`, `{"on": "a"}`, 0},
	{"empty map", mark(``, `{"incantation": "a", "reversion": {}}`), Heresy, `gives no value for the aspects "on", "off"`, `{}`, 0},
	{"unknown placeholder", mark(``, `{"incantation": "echo {{stat}}"}`), Heresy, "{{stat}} is no placeholder known", `{{stat}}`, 0},
	{"placeholder after escape", mark(``, `{"incantation": "echo \"{{stat}}\""}`), Heresy, "{{stat}} is no placeholder", `{{stat}}`, 0},
	{"placeholder after unicode", mark(``, "{\"incantation\": \"\\u00e9\\ud83d\\ude00 {{stat}}\"}"), Heresy, "{{stat}} is no placeholder", `{{stat}}`, 0},
	{"placeholder in line", mark(``, `{"sanctum": "v.kdl", "scripture": ["a", "b {{stat}}"]}`), Heresy, "{{stat}}", `{{stat}}`, 0},
	{"placeholder in tome", mark(``, `{"sanctum": "v.kdl", "scripture": {"tome": "{{stat}}"}}`), Heresy, "{{stat}}", `{{stat}}`, 0},
	{"placeholder in vessel", mark(``, `{"transcription": "{{stat}}", "scripture": null}`), Heresy, "{{stat}}", `{{stat}}`, 0},
	{"placeholder in anchor", mark(``, `{"tether": "t", "anchor": {"*": "{{stat}}"}}`), Heresy, "{{stat}}", `{{stat}}`, 0},
	{"placeholder in offering", mark(``, `{"litany": "s", "offerings": [{"*": "{{stat}}"}]}`), Heresy, "{{stat}}", `{{stat}}`, 0},
	{"placeholder in reversion", mark(``, `{"incantation": "a", "reversion": "{{stat}}"}`), Heresy, "{{stat}}", `{{stat}}`, 0},
	{"placeholder in auspex", mark(`"auspex": {"rite": "{{stat}}"},`, ``), Heresy, "{{stat}}", `{{stat}}`, 0},
	{"vox mark in rite", mark(``, `{"incantation": "{{heresy}}"}`), Heresy, "only in the tidings of a vox-cast", `{{heresy}}`, 0},
	{"undeclared inscription", mark(``, `{"incantation": "echo {{inscription.mood}}"}`), Heresy,
		`inscription "mood", which this rite never declared`, `{{inscription.mood}}`, 0},
	{"null sanctum scripture", mark(``, `{"sanctum": "v.kdl", "scripture": {"on": null, "off": ""}}`), Heresy,
		"the scripture of a sanctum may not be null", `null`, 0},
	{"invalid vox-cast", mark(``, `{"vox-cast": "failure"}`), Heresy, `"failure" is no tiding the codex knows`, `"failure"`, 0},
	{"step without kind", mark(``, `{"anchor": "a"}`), Heresy, "verse 1 of the liturgy names no kind of step", `{"anchor"`, 0},
	{"step of two kinds", mark(``, `{"tether": "t", "anchor": "a", "litany": "s"}`), Heresy, `names both "tether" and "litany"`, `"litany"`, 0},
	{"step no object", mark(``, `"echo"`), Heresy, "verse 1 of the liturgy must be written as an object", `"echo"`, 0},
	{"step lacks anchor", mark(``, `{"tether": "t"}`), Heresy, `the tether of verse 1 holds no "anchor"`, `{"tether"`, 0},
	{"empty vessel", mark(``, `{"sanctum": " ", "scripture": ""}`), Heresy, "vessel may not be empty", `" "`, 0},
	{"bad patience", mark(``, `{"incantation": "a", "patience": "soon"}`), Heresy, `"soon" is no measure of patience`, `"soon"`, 0},
	{"bad seal", mark(``, `{"transcription": "v", "scripture": "", "seal": "rw"}`), Heresy, `"rw" is no seal`, `"rw"`, 0},
	{"aliases unlisted", mark(``, `{"tether": "t", "anchor": "a", "zel": true}`), Heresy, `the keys the codex knows there: "tether", "anchor", "zeal"`, `"zel"`, 0},
	{"fallback named", mark(``, `{"incantation": {"*": 1}}`), Heresy, `the incantation of verse 1 for every aspect not named otherwise must be written as a string`, `1}`, 0},
	{"zeal and force", mark(``, `{"tether": "t", "anchor": "a", "zeal": true, "force": true}`), Heresy, `"zeal" and "force" are one and the same`, `"force"`, 0},
	{"spaced ward", mark(``, `{"sanctum": "v.kdl", "ward": "a b", "scripture": ""}`), Heresy, `the ward "a b" is unfit`, `"a b"`, 0},
	{"ward holds closing glyph", mark(``, `{"sanctum": "v.css", "ward": "a*/", "scripture": ""}`), Heresy, `holds the closing glyph "*/"`, `"a*/"`, 0},
	{"tome without path", mark(``, `{"sanctum": "v.kdl", "scripture": {"illuminate": true}}`), Heresy, `a tome must name the scripture`, `{"illuminate"`, 0},
	{"inscription reserved", mark(``, ``), Heresy, "", "", 0}, // replaced below
	{"auspex without rite", mark(`"auspex": {"patience": "1s"},`, ``), Heresy, `the long form of the auspex must name its "rite"`, `{"patience"`, 0},
	{"mandatory without decree", `{"pattern": "Mark I", "aspects": ["on", "off"],
  "inscriptions": {"mode": {"mandatory": true, "decrees": {"on": "auto"}}}}`, Impurity,
		`the inscription "mode" is mandatory, yet no decree serves the aspect "off"`, `"mode"`, 0},
	{"success not last", mark(``, `{"vox-cast": "success"}, {"incantation": "a"}`), Impurity,
		`a "success" vox-cast proclaims triumph, yet further steps follow it`, `"success"`, 0},
}

func init() {
	for i, c := range heresyCases {
		switch c.name {
		case "unknown key of inscription":
			heresyCases[i].scripture = `{"pattern": "Mark I", "aspects": ["on"], "inscriptions": {"r": {"optional": true}}}`
			heresyCases[i].message = `the key "optional" is not written in the codex for the inscription "r"`
			heresyCases[i].at = `"optional"`
		case "inscription reserved":
			heresyCases[i].scripture = `{"pattern": "Mark I", "aspects": ["on"], "inscriptions": {"foresee": {}}}`
			heresyCases[i].message = `would be spoken as the rune --foresee`
			heresyCases[i].at = `"foresee"`
		}
	}
}

func TestParse_DenouncesWithPosition(t *testing.T) {
	for _, c := range heresyCases {
		if c.name == "invalid rite name" {
			continue
		}
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

func TestParse_InvalidRiteName(t *testing.T) {
	_, fs := Parse("bad name", "/lib/rites/bad name.json", []byte(mark(``, ``)))
	if len(fs) != 1 || !strings.Contains(fs[0].Message, `"bad name" is no fit name for a rite`) || fs[0].Line != 1 {
		t.Fatalf("findings: %v", fs)
	}
}

func TestParse_PatternHeresyStandsAlone(t *testing.T) {
	r, fs := Parse("v0", "/lib/rites/v0.json", []byte(`{"states": ["on"], "files": []}`))
	if r != nil || len(fs) != 1 || !strings.Contains(fs[0].Message, `bears no "pattern"`) {
		t.Fatalf("rite %v, findings %v", r, fs)
	}
}

func TestParse_ElderScriptureIsNamedAsSuch(t *testing.T) {
	cases := []struct {
		data, want string
		elder      bool
	}{
		{`{"states": ["on"], "files": []}`, `its "states" and "files" are the words of the elder scripture`, true},
		{`{"files": []}`, `its "files" are the words of the elder scripture`, true},
		{`{"aspects": ["on"], "liturgy": []}`, "", false},
	}
	for _, c := range cases {
		_, fs := Parse("v0", "/lib/rites/v0.json", []byte(c.data))
		if len(fs) != 1 || !strings.Contains(fs[0].Message, `bears no "pattern"`) {
			t.Fatalf("%s: findings %v", c.data, fs)
		}
		msg := fs[0].Message
		if got := strings.Contains(msg, "nothing of it will be converted"); got != c.elder || !strings.Contains(msg, c.want) {
			t.Errorf("%s: elder=%v, want %v containing %q:\n%s", c.data, got, c.elder, c.want, msg)
		}
	}
}

func TestParse_ValidScriptureIsPure(t *testing.T) {
	cases := map[string]string{
		"escaped braces":   mark(``, `{"incantation": "echo {{{{aspect}}"}`),
		"every mark":       mark(``, `{"incantation": "{{aspect}} {{former}} {{rite.name}} {{inscription.reason}}"}`),
		"fallback only":    mark(``, `{"tether": "t", "anchor": {"*": null}}`),
		"partial decrees":  `{"pattern": "Mark I", "aspects": ["on", "off"], "inscriptions": {"m": {"decrees": {"on": "x"}}}}`,
		"success last":     mark(``, `{"vox-cast": "progress"}, {"incantation": "a"}, {"vox-cast": "success"}`),
		"empty reversion":  mark(``, `{"incantation": "a", "reversion": ""}`),
		"no liturgy":       `{"pattern": "Mark I", "aspects": ["on"]}`,
		"verbatim tome":    mark(``, `{"sanctum": "v.kdl", "scripture": {"tome": "t.kdl"}}`),
		"empty offering":   mark(``, `{"litany": "s", "offerings": [""]}`),
		"marks not in key": mark(``, `{"sanctum": "v.kdl", "ward": "{{x}}", "glyph": "{{", "scripture": ""}`),
	}
	for name, scripture := range cases {
		t.Run(name, func(t *testing.T) {
			if r, fs := Parse("rite", "/r.json", []byte(scripture)); r == nil || len(fs) > 0 {
				t.Fatalf("rite %v, findings:\n%v", r, fs)
			}
		})
	}
}

// plainGlosses are plain words no denunciation may speak.
var plainGlosses = regexp.MustCompile(`(?i)\b(error|warning|switch|states?|metadata|config|invalid|required|description|file|field|value is|timeout|symlink|command line)\b`)

func TestParse_DenunciationsSpeakGrimdark(t *testing.T) {
	for _, c := range heresyCases {
		_, fs := Parse("rite", "/lib/rites/rite.json", []byte(c.scripture))
		for _, f := range fs {
			if m := plainGlosses.FindString(f.Message); m != "" {
				t.Errorf("%s: %q speaks the plain word %q", c.name, f.Message, m)
			}
		}
	}
}
