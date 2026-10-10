package librarium

import (
	"io/fs"
	"strings"
	"testing"
	"time"
)

// theme is a rite using every key of pattern Mark I, with comments,
// trailing commas and both forms of every aspect-varying field.
const theme = `{
  "pattern": "Mark I", // the Mark of its form
  "purpose": "The visage of the machine",
  "aspects": ["default", "finii", "porpl"],
  "inscriptions": {
    "reason": {"purpose": "why the rite was invoked"},
    "mode": {"mandatory": true, "decrees": {"default": "plain", "*": "ornate"}},
  },
  "auspex": {"rite": "cat ~/.cache/theme", "patience": "5s"},
  "tongue": "zsh",
  "liturgy": [
    {"incantation": "niri msg action do-screen-transition -d 200"},
    {"sanctum": "~/.config/niri/util.kdl", "ward": "visage", "glyph": "/*", "closing-glyph": "*/",
     "consecrate": true, "scripture": {"default": "", "finii": ["a {", "  b", "}"], "*": {"tome": "snippets/{{aspect}}.kdl", "illuminate": true}}},
    {"transcription": "~/.config/x/{{rite.name}}.conf", "scripture": {"porpl": null, "*": "visage {{aspect}}"},
     "seal": "0600", "force": true},
    {"tether": "~/.local/share/nfluff/current-theme", "anchor": {"default": null, "*": "~/.config/nfluff/themes/{{aspect}}"}, "zeal": true},
    {"incantation": {"default": "true", "*": "source ~/.config/nfluff/themes/{{aspect}}/hooks"}, "tongue": "bash",
     "patience": "1m30s", "reversion": "source ~/.config/nfluff/themes/{{former}}/hooks"},
    {"vox-cast": "progress"},
    {"litany": "scripts/change-theme", "offerings": ["~/.config/nfluff/themes/{{aspect}}", {"default": "-q", "*": "{{inscription.reason}}"}],
     "reversion": {"*": ""}},
    {"vox-cast": "success"},
  ],
}`

func parse(t *testing.T, scripture string) (*Rite, Findings) {
	t.Helper()
	return Parse("theme", "/lib/rites/theme.json", []byte(scripture))
}

func TestParse_ThemeYieldsEveryField(t *testing.T) {
	r, fs := parse(t, theme)
	if len(fs) != 0 {
		t.Fatalf("unexpected findings:\n%v", fs)
	}
	seal := fs0600
	want := &Rite{
		Name: "theme", Path: "/lib/rites/theme.json",
		Purpose: "The visage of the machine",
		Aspects: []string{"default", "finii", "porpl"},
		Inscriptions: []Inscription{
			{Key: "reason", Purpose: "why the rite was invoked"},
			{Key: "mode", Mandatory: true, Decrees: PerAspect(Entry[string]{"default", "plain"}, Entry[string]{"*", "ornate"})},
		},
		Auspex: &Auspex{Rite: "cat ~/.cache/theme", Patience: 5 * time.Second},
		Tongue: "zsh",
		Liturgy: []Step{
			&Incantation{Command: Uniform("niri msg action do-screen-transition -d 200")},
			&Sanctum{Vessel: "~/.config/niri/util.kdl", Ward: "visage", Glyph: "/*", ClosingGlyph: "*/", Consecrate: true,
				Scripture: PerAspect(
					Entry[Scripture]{"default", Inline("")},
					Entry[Scripture]{"finii", Inline("a {\n  b\n}")},
					Entry[Scripture]{"*", Tome("snippets/{{aspect}}.kdl", true)})},
			&Transcription{Vessel: "~/.config/x/{{rite.name}}.conf",
				Scripture: PerAspect(Entry[Scripture]{"porpl", Scripture{Null: true}}, Entry[Scripture]{"*", Inline("visage {{aspect}}")}),
				Seal:      &seal, Zeal: true},
			&Tether{Name: "~/.local/share/nfluff/current-theme",
				Anchor: PerAspect(Entry[Anchor]{"default", Anchor{Null: true}}, Entry[Anchor]{"*", Anchor{Path: "~/.config/nfluff/themes/{{aspect}}"}}),
				Zeal:   true},
			&Incantation{Command: PerAspect(Entry[string]{"default", "true"}, Entry[string]{"*", "source ~/.config/nfluff/themes/{{aspect}}/hooks"}),
				Utterance: Utterance{Tongue: "bash", Patience: 90 * time.Second, Reversion: Uniform("source ~/.config/nfluff/themes/{{former}}/hooks")}},
			&VoxCast{Tidings: VoxProgress},
			&Litany{Scroll: Uniform("scripts/change-theme"),
				Offerings: []AspectMap[string]{
					Uniform("~/.config/nfluff/themes/{{aspect}}"),
					PerAspect(Entry[string]{"default", "-q"}, Entry[string]{"*", "{{inscription.reason}}"}),
				},
				Utterance: Utterance{Reversion: PerAspect(Entry[string]{"*", ""})}},
			&VoxCast{Tidings: VoxSuccess},
		},
	}
	if r == nil {
		t.Fatal("Parse returned no rite")
	}
	if !r.Equal(want) || r.Path != want.Path {
		t.Fatalf("Parse mismatch\n got: %#v\nwant: %#v", r, want)
	}
}

const fs0600 = fs.FileMode(0o600)

func TestRite_Equal(t *testing.T) {
	a, _ := parse(t, theme)
	b, _ := Parse("theme", "/elsewhere/theme.jsonc", []byte(theme))
	if !a.Equal(b) || a.Equal(nil) || !(*Rite)(nil).Equal(nil) {
		t.Fatal("rites read from different places must be equal")
	}
	b.Liturgy[3].(*Tether).Zeal = false
	if a.Equal(b) {
		t.Fatal("a changed step must make rites differ")
	}
}

func TestAspectMap_For(t *testing.T) {
	m := PerAspect(Entry[string]{"on", "a"}, Entry[string]{"*", "b"})
	for aspect, want := range map[string]string{"on": "a", "off": "b"} {
		if got, ok := m.For(aspect); !ok || got != want {
			t.Errorf("For(%q) = %q, %v; want %q", aspect, got, ok, want)
		}
	}
	partial := PerAspect(Entry[string]{"on", "a"})
	if _, ok := partial.For("off"); ok {
		t.Error("For(off) of a map without fallback must yield nothing")
	}
	if got, ok := Uniform("x").For("anything"); !ok || got != "x" {
		t.Errorf("Uniform.For = %q, %v", got, ok)
	}
	var zero AspectMap[string]
	if !zero.IsZero() || Uniform("").IsZero() {
		t.Error("IsZero must hold only for a map never written")
	}
}

func TestSanctum_Defaults(t *testing.T) {
	cases := []struct {
		s                  Sanctum
		ward, glyph, close string
	}{
		{Sanctum{Vessel: "a.css"}, "rite", "/*", "*/"},
		{Sanctum{Vessel: "a.kdl", Ward: "w"}, "w", "//", ""},
		{Sanctum{Vessel: "a.css", Glyph: "#"}, "rite", "#", ""},
		{Sanctum{Vessel: "a.html", ClosingGlyph: "!>"}, "rite", "<!--", "!>"},
		{Sanctum{Vessel: "config"}, "rite", "#", ""},
	}
	for _, c := range cases {
		g, cl := c.s.Glyphs()
		if w := c.s.WardFor("rite"); w != c.ward || g != c.glyph || cl != c.close {
			t.Errorf("%+v: ward %q glyphs %q %q; want %q %q %q", c.s, w, g, cl, c.ward, c.glyph, c.close)
		}
	}
}

// posOf returns the position of the first needle in text, plus delta bytes.
func posOf(t *testing.T, text, needle string, delta int) Position {
	t.Helper()
	off := strings.Index(text, needle)
	if off < 0 {
		t.Fatalf("fixture lacks %q", needle)
	}
	off += delta
	line := strings.Count(text[:off], "\n") + 1
	return Position{Line: line, Column: off - strings.LastIndex(text[:off], "\n")}
}

func TestRite_Locate(t *testing.T) {
	r, _ := parse(t, theme)
	if r == nil {
		t.Fatal("Parse returned no rite")
	}
	anchor := `"anchor": {"default": null`
	cases := map[string]Position{
		"/liturgy/3/anchor/default": posOf(t, theme, anchor, len(anchor)-len("null")),
		"/liturgy/3/anchor/porpl":   posOf(t, theme, anchor, len(`"anchor": `)), // closest parent: the map
		"":                          {Line: 1, Column: 1},
	}
	for ptr, want := range cases {
		if got := r.Locate(ptr); got != want {
			t.Errorf("Locate(%q) = %v, want %v", ptr, got, want)
		}
	}
	if (&Rite{}).Locate("/aspects") != (Position{}) {
		t.Error("a rite never read from scripture must locate nothing")
	}
}
