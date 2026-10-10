package rituals

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// theme is a rite with two sanctums (one vessel per aspect), a
// transcription and a tether.
const theme = `{
  "pattern": "Mark I",
  "aspects": ["dark", "light"],
  "liturgy": [
    {"sanctum": "$DATA/{{aspect}}.conf", "scripture": "scheme = {{aspect}}"},
    {"sanctum": "$DATA/link.conf", "ward": "accent", "scripture": "accent = {{aspect}}"},
    {"transcription": "$DATA/whole.conf", "scripture": "whole {{aspect}}"},
    {"tether": "$DATA/current", "anchor": "$DATA/themes/{{aspect}}"}
  ]
}`

type themeVessels struct {
	dark, light, real, link, whole, current string
}

// themeFixture writes the rite theme with its vessels and anchors: link.conf
// is a symbolic link to real.conf.
func themeFixture(t *testing.T) (*fixture, themeVessels) {
	fx := newFixture(t)
	v := themeVessels{
		dark:    fx.vessel("dark.conf", "head\n"),
		light:   fx.vessel("light.conf", "top\n"),
		real:    fx.vessel("real.conf", "real\n"),
		link:    filepath.Join(fx.data, "link.conf"),
		whole:   filepath.Join(fx.data, "whole.conf"),
		current: filepath.Join(fx.data, "current"),
	}
	fx.vessel("themes/dark/x", "")
	fx.vessel("themes/light/x", "")
	if err := os.Symlink(v.real, v.link); err != nil {
		t.Fatal(err)
	}
	fx.rite("theme", theme)
	return fx, v
}

func TestExcommunicate_StrikesTheRiteAndLeavesTheMachine(t *testing.T) {
	fx, v := themeFixture(t)
	s := fx.servitor()
	if _, err := s.Invoke(context.Background(), Petition{Rite: "theme", Aspect: "dark"}); err != nil {
		t.Fatal(err)
	}
	before := fx.read(v.dark)

	ex, err := fx.servitor().Excommunicate("theme", false)

	if err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(fx.lib, "rites", "theme.json")) {
		t.Fatal("the scripture still stands")
	}
	if len(ex.Struck) != 1 || len(ex.Purged) != 0 {
		t.Fatalf("excommunication %+v", ex)
	}
	if fx.read(v.dark) != before || !strings.Contains(before, "begin of sanctum theme") {
		t.Fatalf("the sanctum was touched:\n%s", fx.read(v.dark))
	}
}

func TestExcommunicate_PurgeRemovesOnlyTheSanctums(t *testing.T) {
	fx, v := themeFixture(t)
	if _, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "theme", Aspect: "dark"}); err != nil {
		t.Fatal(err)
	}
	// a sanctum left by an earlier invocation into the other aspect
	fx.vessel("light.conf", "top\n\n# +++ begin of sanctum theme -- aspect|light +++\nscheme = light\n"+
		"# +++ end of sanctum theme +++\n")
	whole, anchor := fx.read(v.whole), readlink(t, v.current)

	ex, err := fx.servitor().Excommunicate("theme", true)

	if err != nil {
		t.Fatal(err)
	}
	for p, want := range map[string]string{v.dark: "head\n", v.light: "top\n", v.real: "real\n"} {
		if got := fx.read(p); got != want {
			t.Errorf("%s holds %q, want %q", p, got, want)
		}
	}
	if info, err := os.Lstat(v.link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Error("the bond to the vessel was cast down")
	}
	if fx.read(v.whole) != whole || readlink(t, v.current) != anchor {
		t.Error("the transcription or the tether was touched")
	}
	if exists(filepath.Join(fx.lib, "rites", "theme.json")) {
		t.Fatal("the scripture still stands")
	}
	if len(ex.Purged) != 3 {
		t.Fatalf("purged %v", ex.Purged)
	}
}

func TestExcommunicate_ADefiledVesselStopsThePurge(t *testing.T) {
	fx, v := themeFixture(t)
	// the vessel of the first aspect would yield; that of the second would not
	fx.vessel("dark.conf", "top\n\n# +++ begin of sanctum theme -- aspect|dark +++\nx\n# +++ end of sanctum theme +++\n")
	fx.vessel("light.conf", "# +++ begin of sanctum theme -- aspect|light +++\nnever sealed\n")
	dark := fx.read(v.dark)

	_, err := fx.servitor().Excommunicate("theme", true)

	if err == nil {
		t.Fatal("a defiled vessel was purged")
	}
	grimdark(t, err.Error())
	if !exists(filepath.Join(fx.lib, "rites", "theme.json")) || fx.read(v.dark) != dark {
		t.Fatal("something was touched although the purge was refused")
	}
}

func TestExcommunicate_HereticalAndUnrecordedRites(t *testing.T) {
	fx := newFixture(t)
	p := fx.rite("broken", `{"pattern": "Mark I", "aspects": []}`)
	s := fx.servitor()

	var heretical *Heretical
	if _, err := s.Excommunicate("broken", true); !errors.As(err, &heretical) || !exists(p) {
		t.Fatalf("a heretical rite cannot be purged: %v", err)
	}
	if _, err := s.Excommunicate("broken", false); err != nil || exists(p) {
		t.Fatalf("a heretical rite may still be struck: %v", err)
	}
	var unrecorded *Unrecorded
	if _, err := s.Excommunicate("nowhere", false); !errors.As(err, &unrecorded) {
		t.Fatalf("err = %v", err)
	}
}

func readlink(t *testing.T, p string) string {
	t.Helper()
	l, err := os.Readlink(p)
	if err != nil {
		t.Fatal(err)
	}
	return l
}
