package invocation

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const mouseSanctum = `{"sanctum": "$DATA/util.kdl",
  "scripture": {"on": "cursor {\n    hide-after-inactive-ms 400\n}", "off": ""}}`

func TestSanctum_WritesAndRewritesItsSanctum(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, mouseSanctum)
	p := write(t, fx.data, "util.kdl", "input {\n}\n", 0o640)

	invoke(t, r, "on")
	want := "input {\n}\n\n// +++ begin of sanctum mouse -- aspect|on +++\ncursor {\n    hide-after-inactive-ms 400\n}\n" +
		"// +++ end of sanctum mouse +++\n"
	if got := read(t, p); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	if sealOf(t, p) != 0o640 {
		t.Fatalf("seal not kept: %v", sealOf(t, p))
	}

	seen := invoke(t, r, "off")
	want = "input {\n}\n\n// +++ begin of sanctum mouse -- aspect|off +++\n// +++ end of sanctum mouse +++\n"
	if got := read(t, p); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	if c := seen[0].Vessel; c == nil || !c.Changed() || c.Path != p || !slices.Contains(c.Diff(), "- cursor {") {
		t.Fatalf("foresight = %+v", seen[0])
	}

	if seen := invoke(t, r, "off"); seen[0].Vessel.Changed() {
		t.Fatalf("a second invocation of the same aspect changed %+v", seen[0].Vessel)
	}
}

func TestSanctum_Consecration(t *testing.T) {
	fx := newFixture(t)
	p := filepath.Join(fx.data, "util.kdl")

	r := fx.rite(t, mouseSanctum)
	before := tree(t, fx.data)
	refused(t, r, Options{Aspect: "on"}, "consecrate")
	sameTree(t, before, tree(t, fx.data))

	r = fx.rite(t, `{"sanctum": "$DATA/util.kdl", "consecrate": true, "scripture": "x"}`)
	invoke(t, r, "on")
	if got := read(t, p); got != "// +++ begin of sanctum mouse -- aspect|on +++\nx\n// +++ end of sanctum mouse +++\n" {
		t.Fatalf("consecrated vessel holds %q", got)
	}
	if sealOf(t, p) != 0o644 {
		t.Fatalf("consecrated vessel sealed %v", sealOf(t, p))
	}

	r = fx.rite(t, `{"sanctum": "$DATA/absent/util.kdl", "consecrate": true, "scripture": "x"}`)
	before = tree(t, fx.data)
	refused(t, r, Options{Aspect: "on"}, "absent")
	sameTree(t, before, tree(t, fx.data))
}

func TestSanctum_ReadsTomesAfresh(t *testing.T) {
	fx := newFixture(t)
	p := write(t, fx.data, "util.kdl", "", 0o644)
	tomes := filepath.Join(fx.lib, "rites", "tomes")
	write(t, tomes, "on.kdl", "a {{aspect}}\n", 0o644)

	r := fx.rite(t, `{"sanctum": "$DATA/util.kdl", "scripture": {"tome": "tomes/{{aspect}}.kdl"}}`)
	invoke(t, r, "on")
	if got := read(t, p); !strings.Contains(got, "+++\na {{aspect}}\n//") {
		t.Fatalf("a tome not illuminated must be placed verbatim:\n%s", got)
	}

	r = fx.rite(t, `{"sanctum": "$DATA/util.kdl", "scripture": {"tome": "tomes/{{aspect}}.kdl", "illuminate": true}}`)
	write(t, tomes, "on.kdl", "b {{aspect}} {{rite.name}}\n", 0o644)
	invoke(t, r, "on")
	if got := read(t, p); !strings.Contains(got, "+++\nb on mouse\n//") {
		t.Fatalf("the illuminated tome was not read afresh:\n%s", got)
	}

	refused(t, r, Options{Aspect: "off"}, "tome")

	tome := write(t, tomes, "on.kdl", "fine\n  {{nope}}\n", 0o644)
	hs := refused(t, r, Options{Aspect: "on"}, "{{nope}}")
	if hs[0].Scripture != tome || hs[0].Line != 2 || hs[0].Column != 3 || hs[0].Verse != 1 {
		t.Fatalf("heresy within the tome located at %+v", hs[0])
	}
}

func TestSanctum_FollowsBondsAndKeepsThem(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `{"sanctum": "$DATA/util.kdl", "consecrate": true, "scripture": "x"}`)
	real := write(t, fx.data, "dotfiles/util.kdl", "", 0o600)
	name := filepath.Join(fx.data, "util.kdl")
	link(t, "dotfiles/util.kdl", name)

	invoke(t, r, "on")
	if anchorOf(t, name) != "dotfiles/util.kdl" || !strings.Contains(read(t, real), "aspect|on") || sealOf(t, real) != 0o600 {
		t.Fatalf("the bond was not followed: %q", read(t, real))
	}

	if err := os.Remove(real); err != nil {
		t.Fatal(err)
	}
	seen := invoke(t, r, "off")
	if anchorOf(t, name) != "dotfiles/util.kdl" || !strings.Contains(read(t, real), "aspect|off") {
		t.Fatal("a vessel behind a broken bond was not consecrated where the bond leads")
	}
	if seen[0].Vessel.Path != real || seen[0].Target != name {
		t.Fatalf("foresight %+v", seen[0])
	}
}

func TestSanctum_ManyWardsInOneVessel(t *testing.T) {
	fx := newFixture(t)
	p := filepath.Join(fx.data, "util.kdl")
	r := fx.rite(t, `{"sanctum": "$DATA/util.kdl", "ward": "a", "consecrate": true, "scripture": "1"},
	  {"sanctum": "$DATA/util.kdl", "ward": "b", "glyph": "#", "scripture": "2"}`)

	inv, err := Prepare(r, Options{Aspect: "on"})
	if err != nil {
		t.Fatal(err)
	}
	if seen := inv.Foresee(); seen[1].Vessel.Before != seen[0].Vessel.After || !seen[1].Vessel.Existed {
		t.Fatalf("the second sanctum did not see the first: %+v", seen[1].Vessel)
	}
	if exists(p) {
		t.Fatal("the pre-flight consecrated the vessel")
	}
	if _, err := inv.Perform(); err != nil {
		t.Fatal(err)
	}
	want := "// +++ begin of sanctum a -- aspect|on +++\n1\n// +++ end of sanctum a +++\n\n" +
		"# +++ begin of sanctum b -- aspect|on +++\n2\n# +++ end of sanctum b +++\n"
	if got := read(t, p); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestSanctum_InfersGlyphsFromTheVessel(t *testing.T) {
	fx := newFixture(t)
	p := write(t, fx.data, "style.css", "", 0o644)
	invoke(t, fx.rite(t, `{"sanctum": "$DATA/style.css", "ward": "w", "scripture": ""}`), "on")
	if got := read(t, p); got != "/* +++ begin of sanctum w -- aspect|on +++ */\n/* +++ end of sanctum w +++ */\n" {
		t.Fatalf("got %q", got)
	}
}

func TestSanctum_Refusals(t *testing.T) {
	cases := map[string]struct {
		vessel  string
		liturgy string
		want    string
	}{
		"broken sanctum": {"// +++ begin of sanctum mouse -- aspect|on +++\n", mouseSanctum, "never sealed"},
		"own marker in scripture": {"", `{"sanctum": "$DATA/util.kdl",
		  "scripture": "// +++ end of sanctum mouse +++"}`, "marker"},
		"vessel is a hall": {"", `{"sanctum": "$DATA", "scripture": ""}`, "no vessel that can hold"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			fx := newFixture(t)
			write(t, fx.data, "util.kdl", tc.vessel, 0o644)
			before := tree(t, fx.data)
			hs := refused(t, fx.rite(t, tc.liturgy), Options{Aspect: "on"}, tc.want)
			sameTree(t, before, tree(t, fx.data))
			for _, h := range hs {
				grimdark(t, h.Message)
				if h.Line == 0 || h.Scripture == "" {
					t.Errorf("heresy without place: %+v", h)
				}
			}
		})
	}
}

func TestPrepare_AbortsWithNothingChanged(t *testing.T) {
	fx := newFixture(t)
	write(t, fx.data, "util.kdl", "", 0o644)
	r := fx.rite(t, mouseSanctum+`, {"sanctum": "$DATA/missing.kdl", "scripture": ""}`)
	before := tree(t, fx.data)
	hs := refused(t, r, Options{Aspect: "on"}, "missing.kdl")
	if len(hs) != 1 || hs[0].Verse != 2 || hs[0].Line != 6 {
		t.Fatalf("heresies %+v", hs)
	}
	sameTree(t, before, tree(t, fx.data))
}

func TestPrepare_RefusesAnUndeclaredAspect(t *testing.T) {
	fx := newFixture(t)
	hs := refused(t, fx.rite(t, mouseSanctum), Options{Aspect: "sideways"}, `"sideways"`)
	grimdark(t, hs[0].Message)
	if hs[0].Line != 3 {
		t.Fatalf("heresy at %+v", hs[0])
	}
}
