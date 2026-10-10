package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// TestWizard_ConsecrateAThemeRite: a rite like the nfluff theme rite —
// an incantation, a tether, a litany with offerings and a vox-cast — is
// consecrated through every station by key presses alone, sealed, loaded
// back and invoked.
func TestWizard_ConsecrateAThemeRite(t *testing.T) {
	h := newHarness(t)
	d := h.dataDir
	h.writeData("themes/dark/colors", "dark\n")
	h.writeData("themes/porpl/colors", "porpl\n")
	hooks := h.writeData("hooks.sh", "#!/bin/sh\necho \"$@\" > "+filepath.Join(d, "offered")+"\n")
	if err := os.Chmod(hooks, 0o755); err != nil {
		t.Fatal(err)
	}

	h.keys("n")
	h.mustShow("Consecration of a new rite", "1 Rite", "Liturgy", "Seal", "Name of the rite")
	h.keys("enter")
	h.mustShow("a rite's name must")
	h.typeText("theme")
	h.keys("tab")
	h.typeText("The visage of the machine")
	h.keys("tab", "ctrl+u")
	h.typeText("dark, porpl")
	h.keys("tab")
	h.typeText("reason")
	h.keys("ctrl+s")
	h.mustShow(`Inscription "reason"`, "Purpose", "Mandatory", `Decree for aspect "dark"`, `Decree for aspect "porpl"`)
	h.typeText("why the visage changes")
	h.keys("tab", "tab")
	h.typeText("dusk")
	h.keys("ctrl+s")
	h.mustShow("The liturgy of the rite theme", "No step is written yet")

	h.keys("a")
	h.mustShow("§ sanctum", "¶ transcription", "↣ tether", "» incantation", "≡ litany", "✉ vox-cast")
	h.keys("4")
	h.mustShow("» incantation", "Further rites")
	h.keys("enter")
	h.mustShow(`Command for aspect "dark"`, "Same command for every aspect")
	h.typeText("echo {{aspect}} >> " + filepath.Join(d, "transitions"))
	h.keys("ctrl+s")
	h.mustShow("1 » echo {{aspect}}")

	h.keys("a", "3")
	h.mustShow("↣ tether", "Bound name")
	h.typeText(filepath.Join(d, "current-theme"))
	h.keys("ctrl+s")
	h.mustShow(`Anchor for aspect "dark"`)
	h.typeText(filepath.Join(d, "themes", "{{aspect}}"))
	h.keys("ctrl+s")
	h.mustShow("2 ↣ ")

	h.keys("a", "5", "space", "ctrl+s") // unveil the further rites of the litany
	h.mustShow("Tongue", "Patience")
	h.keys("tab")
	h.typeText("5s")
	h.keys("ctrl+s")
	h.mustShow(`Scroll for aspect "dark"`, "Offerings", "Reversion")
	h.typeText(hooks)
	h.keys("tab", "tab")
	h.typeText(filepath.Join(d, "themes", "{{aspect}}"))
	h.keys("ctrl+s")
	h.mustShow("3 ≡ ")

	h.keys("a", "6")
	h.mustShow("✉ vox-cast", "progress")
	h.keys("space", "ctrl+s")
	h.mustShow("4 ✉ success")
	h.keys("K")
	h.mustShow("3 ✉ success", "4 ≡ ")
	h.keys("J")
	h.mustShow("3 ≡ ", "4 ✉ success")

	h.keys("tab")
	h.mustShow("The rite is pure", `"pattern": "Mark I"`, "rites/theme.json")
	h.keys("G")
	h.mustShow(`"litany"`, `"offerings"`, `"patience": "5s"`, `"vox-cast": "success"`)
	h.keys("enter")
	h.mustShow("The rite theme is consecrated")

	want := &librarium.Rite{Name: "theme", Purpose: "The visage of the machine", Aspects: []string{"dark", "porpl"},
		Inscriptions: []librarium.Inscription{{Key: "reason", Purpose: "why the visage changes",
			Decrees: librarium.PerAspect(librarium.Entry[string]{Aspect: "dark", Value: "dusk"})}},
		Liturgy: []librarium.Step{
			&librarium.Incantation{Command: librarium.Uniform("echo {{aspect}} >> " + filepath.Join(d, "transitions"))},
			&librarium.Tether{Name: filepath.Join(d, "current-theme"),
				Anchor: librarium.Uniform(librarium.Anchor{Path: filepath.Join(d, "themes", "{{aspect}}")})},
			&librarium.Litany{Scroll: librarium.Uniform(hooks),
				Offerings: []librarium.AspectMap[string]{librarium.Uniform(filepath.Join(d, "themes", "{{aspect}}"))},
				Utterance: librarium.Utterance{Patience: 5 * time.Second}},
			&librarium.VoxCast{Tidings: librarium.VoxSuccess},
		}}
	if got := h.loadRite("theme"); !got.Equal(want) {
		data, _ := librarium.Marshal(got)
		t.Fatalf("the sealed rite differs:\n%s", data)
	}

	h.keys("j", "space") // theme sorts after mouse; it is invoked into dark
	h.await()
	h.mustShow("The rite theme is performed: dormant → dark")
	if anchor, err := os.Readlink(filepath.Join(d, "current-theme")); err != nil || anchor != filepath.Join(d, "themes", "dark") {
		t.Fatalf("the tether leads to %q (%v)", anchor, err)
	}
	if got := h.readData("offered"); got != filepath.Join(d, "themes", "dark")+"\n" {
		t.Fatalf("the litany was offered %q", got)
	}
}

// TestWizard_AmendARite: the wizard reads a rite of pattern Mark I, every
// step and aspect map intact, and an amendment of its purpose alone seals
// a rite equal to the old one but for its purpose; the lost comments are
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
	h.mustShow("1 ✉ progress", "2 § ", "3 » echo chanting {{aspect}}", "4 ✉ success")

	h.keys("j", "enter") // the sanctum, aspect by aspect
	h.mustShow("§ sanctum", "cursor")
	h.keys("ctrl+s")
	h.mustShow(`Scripture for aspect "on"`, "a", "b")
	h.keys("ctrl+s")
	h.mustShow(`Scripture for aspect "off"`)
	h.keys("ctrl+s")
	h.mustShow("The liturgy of the rite mouse")

	h.keys("j", "enter") // the incantation, its further rites unveiled
	h.mustShow("» incantation", "Further rites")
	h.keys("ctrl+s")
	h.mustShow("Tongue", "Patience")
	h.keys("ctrl+s")
	h.mustShow(`Command for aspect "on"`, "Reversion", "rm -f")
	h.keys("ctrl+s")

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
	h.keys("ctrl+s", "ctrl+s", "ctrl+s", "tab")
	h.mustShow("rites/cursor.json")
	h.keys("enter")
	h.mustShow("The rite cursor is amended")
	if _, err := os.Stat(filepath.Join(h.dir, "rites", "mouse.json")); !os.IsNotExist(err) {
		t.Fatal("the old scripture still stands after the renaming")
	}
	h.loadRite("cursor")
}

// TestWizard_SameForEveryAspect: a value spoken once for every aspect is
// written under "*", while an aspect may keep a value of its own.
func TestWizard_SameForEveryAspect(t *testing.T) {
	h := newHarness(t)
	three := strings.Replace(markRite, `"aspects": ["on", "off"]`, `"aspects": ["on", "off", "auto"]`, 1)
	h.writeRite("mouse", strings.Replace(three, `{"on": "a\nb", "off": ""}`, `{"auto": "a\nb", "*": ""}`, 1))
	h.keys("r", "e", "ctrl+s", "ctrl+s", "ctrl+s")

	// on and off share, but auto speaks its own: its page is not skipped
	h.keys("j", "enter", "ctrl+s")
	h.mustShow(`Scripture for aspect "on"`)
	h.keys("ctrl+s")
	h.mustShow(`Scripture for aspect "off"`)
	h.keys("ctrl+s")
	h.mustShow(`Scripture for aspect "auto"`)
	h.keys("ctrl+s")
	h.mustShow("The liturgy of the rite mouse")

	h.keys("G")
	h.keys("a", "1")
	h.typeText(filepath.Join(h.dataDir, "y.conf"))
	h.keys("ctrl+s")
	h.mustShow(`Scripture for aspect "on"`, "Same scripture for every aspect")
	h.typeText("own")
	h.keys("ctrl+s")
	h.mustShow(`Scripture for aspect "off"`)
	h.typeText("shared")
	h.keys("tab", "tab", "tab", "space", "ctrl+s") // scripture, tome, illuminate, same
	h.mustShow("The liturgy of the rite mouse", "5 § ")
	h.keys("tab", "G")
	h.mustShow(`"on": "own"`, `"*": "shared"`)
	h.keys("enter")
	st := h.loadRite("mouse").Liturgy[4].(*librarium.Sanctum)
	if e := st.Scripture.Entries(); len(e) != 2 || e[1].Aspect != librarium.Fallback || e[1].Value.Text != "shared" {
		t.Fatalf("scripture = %+v", e)
	}
}

// TestWizard_HeresyBlocksTheSeal: a replica keeping the same ward in the
// same vessel is heresy, and the wizard refuses to seal it.
func TestWizard_HeresyBlocksTheSeal(t *testing.T) {
	h := newHarness(t)
	h.keys("c")
	h.mustShow("Consecration of a new rite", "mouse-copy")
	h.keys("ctrl+s", "ctrl+s", "ctrl+s", "tab")
	h.mustShow("Heresy detected", `the ward "cursor"`)
	h.keys("enter")
	h.mustShow("Heresy remains. The rite cannot be sealed.")
	if _, err := os.Stat(filepath.Join(h.dir, "rites", "mouse-copy.json")); !os.IsNotExist(err) {
		t.Fatal("a heretical rite was sealed")
	}
	h.keys("esc", "j", "d") // cast out the progress vox-cast, then the sanctum
	h.mustShow("1 ✉ progress", "2 » ")
	h.keys("tab")
	h.mustShow("The rite is pure")
	h.keys("esc", "d", "d", "d", "tab")
	h.mustShow("A rite without a liturgy is an empty prayer")
	h.keys("esc", "esc")
	h.mustShow("The consecration is abandoned.")
}

// TestWizard_HereticalRitesAreAmendedByHand: the wizard cannot read what
// heresy has corrupted.
func TestWizard_HereticalRitesAreAmendedByHand(t *testing.T) {
	h := newHarness(t)
	h.writeRite("mouse", `{"pattern": "Mark I", "aspects": [}`)
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
// viewscreen shows the steps around the chosen one.
func TestWizard_ALongLiturgyFollowsTheCursor(t *testing.T) {
	h := newHarness(t)
	var steps []string
	for i := 1; i <= 40; i++ {
		steps = append(steps, fmt.Sprintf(`{"incantation": "echo verse%d"}`, i))
	}
	h.writeRite("long", `{"pattern": "Mark I", "aspects": ["on"], "liturgy": [`+strings.Join(steps, ",")+`]}`)
	h.keys("r", "g", "e", "ctrl+s")
	h.mustShow("1 » echo verse1", "The liturgy of the rite long")
	h.mustNotShow("40 » echo verse40")
	h.keys("G")
	h.mustShow("40 » echo verse40")
	h.mustNotShow("1 » echo verse1 ")
}
