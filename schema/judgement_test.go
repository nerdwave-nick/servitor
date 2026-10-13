package schema

import (
	"maps"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// judgements are scriptures of no pattern this servitor reads, or of a
// malformed one, whose heresy the schema tells as well as the librarium;
// the judgements of pattern Mark II are markIIJudgements. Scripture of
// pattern Mark I is judged by no schema: the librarium converts it.
var judgements = []struct {
	name, scripture string
	pure            bool
}{
	{"pattern missing", `{"aspects": ["on"], "subjects": [` + spirit + `]}`, false},
	{"pattern malformed", `{"pattern": "Mark 1", "aspects": ["on"], "subjects": [` + spirit + `]}`, false},
	{"pattern of an unknown Mark", `{"pattern": "Mark III", "aspects": ["on"], "subjects": [` + spirit + `]}`, false},
	{"pattern no string", `{"pattern": 2, "aspects": ["on"], "subjects": [` + spirit + `]}`, false},
	{"no object", `["Mark II"]`, false},
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
		}{"patience " + p, subject(`{"name": "x", "designation": "machine-spirit", "auspex": {"scan": "cat x"},
			"enact": {"*": "a"}, "patience": ` + p + `}`), pure})
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
	`{"pattern": "Mark II"}`: true,
	`{"$schema": "https://example.org/s.json", "pattern": "Mark II", "tongue": "zsh", "chronicle": "~/c.jsonl",
	  "vox": "notify-send", "patience": "45s"}`: true,
	`{"pattern": "Mark II", "vox": "off"}`:                                                            true,
	`{"pattern": "Mark II", "vox": "auto"}`:                                                           true,
	`{"pattern": "Mark II", "intercessor": "auto"}`:                                                   true,
	`{"pattern": "Mark II", "intercessor": "sudo"}`:                                                   true,
	`{"pattern": "Mark II", "intercessor": "run0"}`:                                                   true,
	`{"pattern": "Mark II", "intercessor": "pkexec"}`:                                                 true,
	`{"pattern": "Mark II", "intercessor": "doas"}`:                                                   false,
	`{"pattern": "Mark II", "intercessor": ""}`:                                                       false,
	`{"pattern": "Mark II", "intercessor": true}`:                                                     false,
	`{"tongue": "zsh"}`:                                                                               false,
	`{"pattern": "Mark II", "tongue": "fish", "vox": "off", "intercessor": "sudo", "patience": "1m"}`: true,
	`{"pattern": "Mark III"}`:                                                                         false,
	`{"pattern": "Mark II", "shell": "zsh"}`:                                                          false,
	`{"pattern": "Mark II", "vox": "loud"}`:                                                           false,
	`{"pattern": "Mark II", "vox": false}`:                                                            false,
	`{"pattern": "Mark II", "tongue": ""}`:                                                            false,
	`{"pattern": "Mark II", "chronicle": " "}`:                                                        false,
	`{"pattern": "Mark II", "chronicle": ["a"]}`:                                                      false,
	`{"pattern": "Mark II", "$schema": null}`:                                                         false,
	`{"pattern": "Mark II", "aspects": ["on"]}`:                                                       false,
	`["Mark II"]`: false,
}

func TestSettingsSchema_JudgesAsTheLibrariumJudges(t *testing.T) {
	s := compile(t, Settings)
	cases := maps.Clone(settingsJudgements)
	for p, pure := range patiences {
		cases[`{"pattern": "Mark II", "patience": `+p+`}`] = pure
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
