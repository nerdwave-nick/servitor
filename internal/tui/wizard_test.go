package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// loadRite reads the rite name back from the Librarium of the harness.
func (h *harness) loadRite(name string) *librarium.Rite {
	h.t.Helper()
	r, found := librarium.LoadFile(filepath.Join(h.dir, "rites", name+".json"))
	if r == nil || found.Heretical() {
		h.t.Fatalf("the rite %s does not load: %v", name, found)
	}
	return r
}

// TestWizard_AmendARite: the wizard reads a rite of pattern Mark II, every
// subject and omen intact, and an amendment of its purpose alone seals a
// rite equal to the old one but for its purpose; the lost comments are
// foretold.
func TestWizard_AmendARite(t *testing.T) {
	h := newHarness(t)
	h.writeRite("mouse", "// the cursor hides in the dark\n"+markRite)
	h.keys("r")
	before := *h.m.s.Librarium.Rites["mouse"]

	h.keys("e")
	h.mustShow("Amendment of the rite mouse", "Hide the cursor", "reason, mode")
	h.keys("tab", "ctrl+u")
	h.typeText("Hide the cursor from sight")
	h.keys("ctrl+s")
	h.mustShow(`Inscription "reason"`, "why")
	h.keys("ctrl+s")
	h.mustShow(`Inscription "mode"`, "hide", "show")
	h.keys("ctrl+s")
	h.mustShow("The subjects of the rite mouse", "1 § cursor", "2 ⚙ chant")

	h.keys("tab")
	h.mustShow("The observances of the rite mouse")
	h.keys("tab")
	h.mustShow("The rite is pure", "comments of the amended scripture will not survive",
		`"purpose": "Hide the cursor from sight"`)
	h.keys("enter")
	h.mustShow("The rite mouse is amended")
	want := before
	want.Purpose = "Hide the cursor from sight"
	if got := h.loadRite("mouse"); !got.Equal(&want) {
		data, _ := librarium.Marshal(got)
		t.Fatalf("the amended rite differs:\n%s", data)
	}
}

// TestWizard_RenameStrikesTheOldScripture: an amendment under a new name
// leaves the rite recorded once.
func TestWizard_RenameStrikesTheOldScripture(t *testing.T) {
	h := newHarness(t)
	h.keys("e", "ctrl+u")
	h.typeText("cursor")
	h.keys("ctrl+s", "ctrl+s", "ctrl+s", "tab", "tab")
	h.mustShow("rites/cursor.json")
	h.keys("enter")
	h.mustShow("The rite cursor is amended")
	if _, err := os.Stat(filepath.Join(h.dir, "rites", "mouse.json")); !os.IsNotExist(err) {
		t.Fatal("the old scripture still stands after the renaming")
	}
	h.loadRite("cursor")
}

// TestWizard_HeresyBlocksTheSeal: a replica keeping the same ward in the
// same vessel is heresy, and the wizard refuses to seal it until its
// sanctum is struck; a rite without subjects is refused as well.
func TestWizard_HeresyBlocksTheSeal(t *testing.T) {
	h := newHarness(t)
	h.keys("c")
	h.mustShow("Consecration of a new rite", "mouse-copy")
	h.keys("ctrl+s", "ctrl+s", "ctrl+s", "tab", "tab")
	h.mustShow("Heresy detected", `the ward "cursor"`)
	h.keys("enter")
	h.mustShow("Heresy remains. The rite cannot be sealed.")
	if _, err := os.Stat(filepath.Join(h.dir, "rites", "mouse-copy.json")); !os.IsNotExist(err) {
		t.Fatal("a heretical rite was sealed")
	}
	h.keys("esc", "esc", "d") // strike the sanctum
	h.mustShow("1 ⚙ chant")
	h.keys("tab", "tab")
	h.mustShow("The rite is pure")
	h.keys("esc", "esc", "d", "tab")
	h.mustShow("A rite without subjects has nothing to read or set")
	h.keys("esc", "esc")
	h.mustShow("The consecration is abandoned.")
}

// TestWizard_HereticalRitesAreAmendedByHand: the wizard cannot read what
// heresy has corrupted.
func TestWizard_HereticalRitesAreAmendedByHand(t *testing.T) {
	h := newHarness(t)
	h.writeRite("mouse", `{"pattern": "Mark II", "aspects": [}`)
	h.keys("r", "e")
	h.mustShow("purify it with o")
	h.keys("c")
	h.mustShow("purify it with o")
}

// TestWizard_TheCatalogueNamesTheWizardKeys: the keys of the liturgy are
// told in the catalogue of the cogitator.
func TestWizard_TheCatalogueNamesTheWizardKeys(t *testing.T) {
	h := newHarness(t)
	h.keys("?")
	h.mustShow("a add a step", "J/K reorder", "tab onward to the seal")
}

// TestWizard_ALongLiturgyFollowsTheCursor: a liturgy longer than the
// viewscreen shows the entries around the chosen one.
func TestWizard_ALongLiturgyFollowsTheCursor(t *testing.T) {
	h := newHarness(t)
	var entries []string
	for i := 1; i <= 40; i++ {
		entries = append(entries, fmt.Sprintf(`{"designation": "incantation", "words": "echo verse%d"}`, i))
	}
	h.writeRite("long", `{"pattern": "Mark II", "edict": true, "liturgy": [`+strings.Join(entries, ",")+`]}`)
	h.keys("r", "g", "e", "ctrl+s")
	h.mustShow("1 » echo verse1", "The liturgy of the edict long")
	h.mustNotShow("40 » echo verse40")
	h.keys("G")
	h.mustShow("40 » echo verse40")
	h.mustNotShow("1 » echo verse1 ")
}
