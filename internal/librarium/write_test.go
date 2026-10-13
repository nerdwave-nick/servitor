package librarium

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"
)

// roundTrip writes r, reads it back and demands an equal, pure rite whose
// scripture is written identically a second time.
func roundTrip(t *testing.T, r *Rite) []byte {
	t.Helper()
	data, err := Marshal(r)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	back, fs := parse(r.Name, "/lib/rites/"+r.Name+".json", data)
	if len(fs) != 0 {
		t.Fatalf("written scripture is impure:\n%v\n%s", fs, data)
	}
	if !back.Equal(r) {
		t.Fatalf("read back a different rite\nwritten:\n%s\n got: %#v\nwant: %#v", data, back, r)
	}
	again, err := Marshal(back)
	if err != nil || !bytes.Equal(again, data) {
		t.Fatalf("second writing differs (%v):\n%s\n%s", err, data, again)
	}
	return data
}

// TestMarshal_WritesNoScriptureOfMarkI: a rite read as pattern Mark I is
// never written; only its conversion is.
func TestMarshal_WritesNoScriptureOfMarkI(t *testing.T) {
	r, fs := readRite(t, theme)
	if len(fs) != 0 || !r.markI {
		t.Fatalf("rite %+v, findings: %v", r, fs)
	}
	if data, err := Marshal(r); err == nil || !strings.Contains(err.Error(), `converted to pattern "Mark II" first`) {
		t.Fatalf("a rite of Mark I was written (%v):\n%s", err, data)
	}
}

// alienStep is a step of no kind the codex knows.
type alienStep struct{}

func (alienStep) Kind() Kind { return 0 }

func TestMarshal_RefusesAlienSteps(t *testing.T) {
	if _, err := Marshal(&Rite{Name: "x", Edict: true, Liturgy: []Step{alienStep{}}}); err == nil {
		t.Fatal("an alien step must not be written")
	}
}

// TestMarshal_ShortScalarMembersShareOneLine: an array or object of scalars
// is written on one line when that line, indent and key with it, fits in
// eighty columns; scripture written as lines keeps one line per line.
func TestMarshal_ShortScalarMembersShareOneLine(t *testing.T) {
	r, fs := parse("visage", "/lib/rites/visage.json", []byte(visageII))
	if len(fs) != 0 {
		t.Fatalf("findings: %v", fs)
	}
	data := string(roundTrip(t, r))
	for _, want := range []string{
		"\n  \"aspects\": [\"default\", \"porpl\"],\n",
		"\n      \"recessional\": [\"hooks\", \"recite\"]\n",
		"\"to\": [\"porpl\", \"default\"]",
	} {
		if !strings.Contains(data, want) {
			t.Errorf("scripture lacks %q:\n%s", want, data)
		}
	}
	for _, line := range strings.Split(data, "\n") {
		if n := utf8.RuneCountInString(line); n > 80 && (strings.HasSuffix(line, "]") || strings.HasSuffix(line, "],") ||
			strings.HasSuffix(line, "}") || strings.HasSuffix(line, "},")) && strings.Count(line, `"`) > 2 {
			t.Errorf("a line of %d columns was joined: %s", n, line)
		}
	}
}

func TestMarshal_JoinsWhatFitsAndNothingElse(t *testing.T) {
	long := strings.Repeat("x", 60)
	r := &Rite{Name: "x", Aspects: []string{"on", "off"}, Subjects: []Subject{
		{Name: "v", Matter: &Sanctum{Vessel: "/v", Scripture: PerAspect(Entry[Scripture]{"on", Inline("")},
			Entry[Scripture]{"*", Inline("a\nb")})}},
		{Name: "w", Matter: &Sanctum{Vessel: "/w", Scripture: PerAspect(Entry[Scripture]{"on", Inline("c\nd")},
			Entry[Scripture]{"*", Inline("e")})}, Recessional: []string{"short", "bare", "long"}},
	}, Observances: []Observance{
		{Name: "short", Speech: &Litany{Scroll: Uniform("s"), Offerings: []AspectMap[string]{Uniform("~/x/{{aspect}}")}}},
		{Name: "bare", Speech: &Litany{Scroll: Uniform("s"), Offerings: []AspectMap[string]{}}},
		{Name: "long", Speech: &Litany{Scroll: Uniform("s"), Offerings: []AspectMap[string]{Uniform(long), Uniform(long)}}},
	}}
	data := string(roundTrip(t, r))
	for _, want := range []string{
		"\n      \"offerings\": [\"~/x/{{aspect}}\"]\n",
		"\"offerings\": []",
		"\n      \"offerings\": [\n        \"" + long + "\",\n        \"" + long + "\"\n      ]\n",
		"\n        \"*\": [\n          \"a\",\n          \"b\"\n        ]\n",
		"\n        \"on\": [\n          \"c\",\n          \"d\"\n        ],\n        \"*\": \"e\"\n",
	} {
		if !strings.Contains(data, want) {
			t.Errorf("scripture lacks %q:\n%s", want, data)
		}
	}
}
