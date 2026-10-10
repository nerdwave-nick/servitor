package librarium

import (
	"bytes"
	"io/fs"
	"strings"
	"testing"
	"time"
)

// roundTrip writes r, reads it back and demands an equal, pure rite whose
// scripture is written identically a second time.
func roundTrip(t *testing.T, r *Rite) []byte {
	t.Helper()
	data, err := Marshal(r)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	back, fs := Parse(r.Name, "/lib/rites/"+r.Name+".json", data)
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

func TestMarshal_ThemeRoundTrips(t *testing.T) {
	r, fs := parse(t, theme)
	if len(fs) != 0 {
		t.Fatalf("findings: %v", fs)
	}
	data := string(roundTrip(t, r))
	for _, want := range []string{
		"{\n  \"pattern\": \"Mark I\",\n  \"purpose\": ",
		`"zeal": true`, `"seal": "0600"`, `"patience": "1m30s"`, `"patience": "5s"`,
		"\"finii\": [\n", `"tome": "snippets/{{aspect}}.kdl"`, `"porpl": null`,
	} {
		if !strings.Contains(data, want) {
			t.Errorf("scripture lacks %q:\n%s", want, data)
		}
	}
	if strings.Contains(data, `"force"`) || !strings.HasSuffix(data, "}\n") {
		t.Errorf("scripture must write zeal and end in one newline:\n%s", data)
	}
}

func TestMarshal_ForgedRitesRoundTrip(t *testing.T) {
	seal := fs.FileMode(0o4755)
	cases := map[string]*Rite{
		"bare": {Name: "bare", Aspects: []string{"on"}},
		"short auspex": {Name: "dnd", Aspects: []string{"on", "off"}, Auspex: &Auspex{Rite: "makoctl mode"},
			Liturgy: []Step{&Incantation{Command: PerAspect(Entry[string]{"on", "makoctl mode -a dnd"}, Entry[string]{"off", "makoctl mode -r dnd"})}}},
		"long auspex": {Name: "x", Aspects: []string{"on"}, Auspex: &Auspex{Rite: "probe", Patience: 1500 * time.Millisecond}},
		"inscriptions": {Name: "x", Aspects: []string{"on", "off"}, Inscriptions: []Inscription{
			{Key: "z"}, {Key: "a", Purpose: "why", Mandatory: true, Decrees: Uniform("all")},
			{Key: "m", Decrees: PerAspect(Entry[string]{"on", "x"})}, {Key: "e", Decrees: PerAspect[string]()},
		}},
		"every step": {Name: "x", Purpose: "p", Aspects: []string{"on", "off"}, Tongue: "fish", Liturgy: []Step{
			&Sanctum{Vessel: "/v.css", Scripture: Uniform(Inline("a\n\nb\n"))},
			&Sanctum{Vessel: "/w.kdl", Ward: "w", ClosingGlyph: "!", Scripture: Uniform(Tome("t", false))},
			&Transcription{Vessel: "/t", Scripture: Uniform(Scripture{Null: true}), Seal: &seal},
			&Transcription{Vessel: "/u", Scripture: PerAspect(Entry[Scripture]{"*", Tome("{{aspect}}", true)})},
			&Tether{Name: "/l", Anchor: Uniform(Anchor{Null: true})},
			&Tether{Name: "/m", Anchor: PerAspect(Entry[Anchor]{"on", Anchor{Path: "/a"}}, Entry[Anchor]{"off", Anchor{Null: true}}), Zeal: true},
			&Litany{Scroll: Uniform("s"), Offerings: []AspectMap[string]{}},
			&Litany{Scroll: PerAspect(Entry[string]{"*", "s"}), Offerings: []AspectMap[string]{Uniform(""), PerAspect(Entry[string]{"on", "1"}, Entry[string]{"*", "2"})},
				Utterance: Utterance{Tongue: "sh", Patience: time.Minute, Reversion: Uniform("")}},
			&VoxCast{Tidings: VoxProgress},
			&VoxCast{Tidings: VoxSuccess},
		}},
	}
	for name, r := range cases {
		t.Run(name, func(t *testing.T) { roundTrip(t, r) })
	}
}

type alienStep struct{}

func (alienStep) Kind() Kind { return 0 }

func TestMarshal_RefusesAlienSteps(t *testing.T) {
	if _, err := Marshal(&Rite{Name: "x", Aspects: []string{"on"}, Liturgy: []Step{alienStep{}}}); err == nil {
		t.Fatal("an alien step must not be written")
	}
}
