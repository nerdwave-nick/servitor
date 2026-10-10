package sanctum

import (
	"strings"
	"testing"
)

var mouse = Marker{Glyph: "//", Ward: "mouse"}

func TestRender_SpeaksTheMarkIMarkers(t *testing.T) {
	cases := map[string]struct {
		m       Marker
		indent  string
		content string
		want    string
	}{
		"plain": {mouse, "", "cursor {}\n",
			"// +++ begin of sanctum mouse -- aspect|on +++\ncursor {}\n// +++ end of sanctum mouse +++"},
		"closing glyph and indent": {Marker{Glyph: "/*", ClosingGlyph: "*/", Ward: "g"}, "  ", "a {}",
			"  /* +++ begin of sanctum g -- aspect|on +++ */\na {}\n  /* +++ end of sanctum g +++ */"},
		"empty scripture keeps an empty sanctum": {mouse, "", "",
			"// +++ begin of sanctum mouse -- aspect|on +++\n// +++ end of sanctum mouse +++"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := Render(tc.m, tc.indent, "on", tc.content); got != tc.want {
				t.Fatalf("got\n%s\nwant\n%s", got, tc.want)
			}
		})
	}
}

func TestFind_ReadsTheAspectAndIgnoresOtherPairs(t *testing.T) {
	cases := map[string]struct {
		begin string
		want  string
	}{
		"aspect only":        {"// +++ begin of sanctum mouse -- aspect|on +++", "on"},
		"extra pairs before": {`// +++ begin of sanctum mouse -- reason|"gaming remnant" aspect|on +++`, "on"},
		"extra pairs after":  {"// +++ begin of sanctum mouse -- aspect|on by|hand x|\"a +++ b\" +++", "on"},
		"quoted aspect":      {`// +++ begin of sanctum mouse -- aspect|"on" +++`, "on"},
		"stray words":        {"// +++ begin of sanctum mouse -- lonely aspect|off +++", "off"},
		"no pairs at all":    {"// +++ begin of sanctum mouse +++", ""},
		"tight glyph":        {"//+++ begin of sanctum mouse -- aspect|on +++", "on"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			text := "input {}\n  " + tc.begin + "\n  cursor {}\n// +++ end of sanctum mouse +++\n"
			s, found, err := Find(text, mouse)
			if err != nil || !found {
				t.Fatalf("found=%v err=%v", found, err)
			}
			if s.Aspect != tc.want || s.Content != "  cursor {}" || s.Indent != "  " || s.BeginLine != 1 || s.EndLine != 3 {
				t.Fatalf("sanctum = %+v", s)
			}
		})
	}
}

func TestFind_KeepsWardsApart(t *testing.T) {
	text := "# +++ begin of sanctum mousey -- aspect|on +++\n# +++ end of sanctum mousey +++\n"
	if _, found, err := Find(text, Marker{Glyph: "#", Ward: "mouse"}); found || err != nil {
		t.Fatalf("ward mouse must not answer for mousey: found=%v err=%v", found, err)
	}
}

func TestFind_DenouncesBrokenSanctums(t *testing.T) {
	begin := "// +++ begin of sanctum mouse -- aspect|on +++\n"
	end := "// +++ end of sanctum mouse +++\n"
	cases := map[string]string{
		"unsealed":    begin + "x\n",
		"stray end":   end + begin + end,
		"twice":       begin + end + begin + end,
		"nested":      begin + begin + end,
		"end without": "x\n" + end,
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, err := Find(text, mouse); err == nil {
				t.Fatal("expected a denunciation")
			}
		})
	}
}

func TestUpsert_AppendsWhenAbsentAndReplacesInPlace(t *testing.T) {
	got, err := Upsert("input {}", mouse, "on", "cursor {}")
	if err != nil {
		t.Fatal(err)
	}
	want := "input {}\n\n// +++ begin of sanctum mouse -- aspect|on +++\ncursor {}\n// +++ end of sanctum mouse +++\n"
	if got != want {
		t.Fatalf("append: got\n%q\nwant\n%q", got, want)
	}
	if got, _ := Upsert("", mouse, "on", ""); got != "// +++ begin of sanctum mouse -- aspect|on +++\n// +++ end of sanctum mouse +++\n" {
		t.Fatalf("empty vessel: %q", got)
	}
	again, err := Upsert("head\n"+strings.Replace(want, "input {}\n\n", "", 1)+"tail\n", mouse, "off", "")
	if err != nil {
		t.Fatal(err)
	}
	if again != "head\n// +++ begin of sanctum mouse -- aspect|off +++\n// +++ end of sanctum mouse +++\ntail\n" {
		t.Fatalf("replace: %q", again)
	}
	if _, err := Upsert("// +++ end of sanctum mouse +++\n", mouse, "on", ""); err == nil {
		t.Fatal("a broken sanctum must not be rewritten")
	}
}

func TestUpsert_KeepsIndentAndOtherSanctums(t *testing.T) {
	text := "a {\n    // +++ begin of sanctum mouse -- aspect|on +++\n    x\n    // +++ end of sanctum mouse +++\n}\n" +
		"// +++ begin of sanctum other -- aspect|on +++\n// +++ end of sanctum other +++\n"
	got, err := Upsert(text, mouse, "off", "    y")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(text, "aspect|on +++\n    x", "aspect|off +++\n    y", 1)
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestIsMarkerLine(t *testing.T) {
	for line, want := range map[string]bool{
		"// +++ begin of sanctum mouse -- aspect|on +++": true,
		"   // +++ end of sanctum mouse +++":             true,
		"// +++ end of sanctum other +++":                false,
		"cursor {}":                                      false,
	} {
		if got := mouse.IsMarkerLine(line); got != want {
			t.Errorf("IsMarkerLine(%q) = %v", line, got)
		}
	}
}
