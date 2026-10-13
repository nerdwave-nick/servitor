package rituals

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/dominion"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// TestConvert_WritesEveryConversion: the servitor that may write keeps
// each scripture of pattern Mark I in rites/.mark-i/ and writes its
// conversion in place, telling each once; the vessels of its sanctums are
// untouched, and a second reading converts nothing.
func TestConvert_WritesEveryConversion(t *testing.T) {
	fx := newFixture(t)
	vessel := fx.vessel("util.kdl", "input {}\n")
	path := fx.rite("mouse", mouse)
	original := fx.read(path)
	settings := fx.write(fx.lib, librarium.SettingsName, "{\n  // kept\n  \"pattern\": \"Mark I\"\n}\n")
	s := OpenConverting(fx.lib, librarium.Runes{})
	if r := s.Librarium.Rites["mouse"]; r == nil || !r.MarkII {
		t.Fatalf("the rite is not read converted: %v", s.Librarium.Findings)
	}
	told := s.Convert()
	if len(told.Laments) != 0 || len(told.Tidings) != 2 {
		t.Fatalf("tidings %q, laments %v", told.Tidings, told.Laments)
	}
	if !strings.Contains(told.Tidings[1], `the rite "mouse" is converted`) ||
		!strings.Contains(told.Tidings[1], filepath.Join(fx.lib, "rites", ".mark-i", "mouse.json")) {
		t.Errorf("the tidings do not name the rite and its copy: %q", told.Tidings[1])
	}
	for _, tiding := range told.Tidings {
		grimdark(t, tiding)
	}
	if fx.read(filepath.Join(fx.lib, "rites", ".mark-i", "mouse.json")) != original ||
		!strings.Contains(fx.read(path), `"pattern": "Mark II"`) {
		t.Error("the rite is not kept and converted on disk")
	}
	if fx.read(settings) != "{\n  // kept\n  \"pattern\": \"Mark II\"\n}\n" {
		t.Errorf("the settings are rewritten as %q", fx.read(settings))
	}
	if fx.read(vessel) != "input {}\n" {
		t.Error("the conversion touched a vessel")
	}
	if again := s.Convert(); len(again.Tidings)+len(again.Laments) != 0 {
		t.Errorf("a conversion is told twice: %+v", again)
	}
	if again := OpenConverting(fx.lib, librarium.Runes{}); len(again.Librarium.Conversions) != 0 {
		t.Errorf("the Librarium converted is converted anew")
	}
}

// TestConvert_AwaitsNoDominion: a rite whose dominion another invocation
// holds is not written meanwhile, nor awaited: it stays converted in
// memory alone, to be written at a later reading.
func TestConvert_AwaitsNoDominion(t *testing.T) {
	fx := newFixture(t)
	path := fx.rite("mouse", mouse)
	original := fx.read(path)
	held, err := dominion.Seize(context.Background(), []string{dominion.Rite(os.LookupEnv, "mouse")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Relinquish()
	s := OpenConverting(fx.lib, librarium.Runes{})
	if told := s.Convert(); len(told.Tidings)+len(told.Laments) != 0 {
		t.Errorf("a rite under another's dominion was converted: %+v", told)
	}
	if fx.read(path) != original || !s.Librarium.Rites["mouse"].MarkII {
		t.Error("the rite under another's dominion was written, or is not converted in memory")
	}
	held.Relinquish()
	if told := s.Convert(); len(told.Tidings) != 1 {
		t.Errorf("the rite is not converted once its dominion is free: %+v", told)
	}
}

// TestConvert_LamentsACopyThatStands: a conversion that cannot be written
// is lamented in grimdark, and the scripture stays as it was.
func TestConvert_LamentsACopyThatStands(t *testing.T) {
	fx := newFixture(t)
	path := fx.rite("mouse", mouse)
	original := fx.read(path)
	fx.write(fx.lib, "rites/.mark-i/mouse.json", "elder")
	told := OpenConverting(fx.lib, librarium.Runes{}).Convert()
	if len(told.Tidings) != 0 || len(told.Laments) != 1 || fx.read(path) != original {
		t.Fatalf("tidings %q, laments %v", told.Tidings, told.Laments)
	}
	grimdark(t, told.Laments[0].Error())
}
