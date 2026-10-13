package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// sealedScripture reads the scripture the wizard sealed for the rite name
// and holds it to want, $DATA being the vessels' place.
func (h *harness) sealedScripture(name, want string) {
	h.t.Helper()
	want = strings.ReplaceAll(want, "$DATA", h.dataDir)
	data, err := os.ReadFile(filepath.Join(h.dir, "rites", name+".json"))
	if err != nil || string(data) != want {
		h.t.Fatalf("the scripture sealed (%v):\n%s\nwant:\n%s", err, data, want)
	}
}

// choosePatternMarkII turns the Rite page of a new rite, its name focused,
// to pattern Mark II, and returns to the name.
func (h *harness) choosePatternMarkII() {
	h.t.Helper()
	h.keys("up")
	h.mustShow("Pattern", "◆ Mark I", "◇ Mark II", "Auspex")
	h.keys("space")
	h.mustShow("◆ Mark II", "Vox-cast of progress", "Vox-cast of success")
	h.mustNotShow("Auspex", "Patience of the auspex")
	h.keys("down")
}

// TestWizard_ConsecrateASanctumRiteOfMarkII: a rite of pattern Mark II
// whose one subject is a sanctum is consecrated by key presses alone. The
// Rite page asks the vox-cast list and no auspex, the inscription's decree
// is written as an aspect map, the subject is chosen by its designation,
// and its omens are written on one grid, a row for each aspect.
func TestWizard_ConsecrateASanctumRiteOfMarkII(t *testing.T) {
	h := newHarness(t)
	h.keys("n")
	h.typeText("cursor")
	h.choosePatternMarkII()
	h.keys("down")
	h.typeText("Hide the cursor")
	h.keys("down", "down", "down")
	h.typeText("mode")
	h.keys("down", "down", "space", "ctrl+s")
	h.mustShow(`Inscription "mode"`, `Decree for aspect "on"`, "Decree for every other aspect (*)")
	h.keys("tab", "tab", "tab", "tab")
	h.typeText("plain")
	h.keys("ctrl+s")
	h.mustShow("✔ Rite", "2 Subjects", "3 Seal", "The subjects of the rite cursor", "No subject is declared yet")
	h.mustNotShow("Liturgy")

	h.keys("a")
	h.mustShow("1 § sanctum", "2 ¶ transcription", "3 ↣ tether")
	h.mustNotShow("incantation", "litany", "vox-cast")
	h.keys("1")
	h.mustShow("New subject", "§ sanctum", "Name of the subject", "Vessel", "Ward", "Exalted subject")
	h.keys("ctrl+s")
	h.mustShow("a subject must be named")
	h.typeText("niri")
	h.keys("tab", "tab")
	h.typeText(filepath.Join(h.dataDir, "x.kdl"))
	h.keys("tab")
	h.typeText("hidden")
	h.keys("ctrl+s")
	h.mustShow("omens", `Aspect "on" bears the omen of every other aspect`, `Omen of aspect "on"`, `Omen of aspect "off"`)
	h.mustNotShow("Omen of every other aspect")
	h.keys("tab")
	h.typeText("a")
	h.keys("enter")
	h.typeText("b")
	h.keys("ctrl+s")
	h.mustShow("1 § niri")

	h.keys("tab")
	h.mustShow("The rite is pure", `"pattern": "Mark II"`, "rites/cursor.json")
	h.keys("enter")
	h.mustShow("The rite cursor is consecrated")
	h.sealedScripture("cursor", `{
  "pattern": "Mark II",
  "purpose": "Hide the cursor",
  "aspects": ["on", "off"],
  "inscriptions": {
    "mode": {
      "decrees": {"*": "plain"}
    }
  },
  "vox-cast": ["success"],
  "subjects": [
    {
      "name": "niri",
      "designation": "sanctum",
      "vessel": "$DATA/x.kdl",
      "ward": "hidden",
      "omens": {
        "on": [
          "a",
          "b"
        ],
        "off": ""
      }
    }
  ]
}
`)
	if r := h.loadRite("cursor"); !r.MarkII || len(r.Subjects) != 1 || r.Subjects[0].Designation() != librarium.DesignationSanctum {
		t.Fatalf("the scripture does not load as a rite of Mark II: %+v", r)
	}
}

// TestWizard_ConsecrateATetherRiteOfMarkII: a rite of pattern Mark II with
// a tether whose every aspect bears the omen written under "*", and a
// transcription whose one aspect bears its own omen and every other the
// vessel struck; a subject is reordered with K and J, and one is struck
// with d before the sealing.
func TestWizard_ConsecrateATetherRiteOfMarkII(t *testing.T) {
	h := newHarness(t)
	h.keys("n")
	h.typeText("theme")
	h.choosePatternMarkII()
	h.keys("down", "down", "down", "ctrl+u")
	h.typeText("dark, porpl")
	h.keys("ctrl+s")
	h.mustShow("The subjects of the rite theme")

	h.keys("a", "3")
	h.mustShow("New subject", "↣ tether", "Bound name", "Exalted subject", "Further rites")
	h.typeText("current")
	h.keys("tab", "tab")
	h.typeText("current-theme")
	h.keys("ctrl+s")
	h.mustShow(`Aspect "dark" bears the omen of every other aspect`, "[✔] yes",
		"Omen of every other aspect (*): its anchor", "Unbind the tether in every other aspect (*)")
	h.mustNotShow(`Omen of aspect "dark"`, `Omen of aspect "porpl"`)
	h.keys("tab", "tab")
	h.typeText("themes/{{aspect}}")
	h.keys("ctrl+s")
	h.mustShow("1 ↣ current")

	h.keys("a", "2")
	h.mustShow("¶ transcription", "Vessel")
	h.typeText("palette")
	h.keys("tab", "tab")
	h.typeText("palette.conf")
	h.keys("tab", "tab", "space", "ctrl+s")
	h.mustShow("further rites", "Seal", "Zeal")
	h.typeText("0600")
	h.keys("ctrl+s")
	h.mustShow("omens", `Omen of aspect "dark": its scripture`, `Aspect "porpl" bears the omen`)
	h.mustNotShow("Omen of every other aspect")
	h.keys("tab")
	h.typeText("dusk")
	h.keys("tab", "tab", "tab", "tab", "space") // porpl bears the omen of every other aspect
	h.mustNotShow(`Omen of aspect "porpl"`)
	h.keys("down")
	h.mustShow("Omen of every other aspect (*): its scripture")
	h.keys("tab", "tab", "tab", "space")
	h.mustShow("Strike the vessel in every other aspect (*)")
	h.keys("ctrl+s")
	h.mustShow("1 ↣ current", "2 ¶ palette")

	h.keys("K")
	h.mustShow("1 ¶ palette", "2 ↣ current")
	h.keys("J")
	h.mustShow("1 ↣ current", "2 ¶ palette")
	h.keys("a", "1")
	h.typeText("doomed")
	h.keys("tab", "tab")
	h.typeText("doomed.kdl")
	h.keys("ctrl+s", "ctrl+s")
	h.mustShow("3 § doomed")
	h.keys("d")
	h.mustNotShow("doomed")

	h.keys("tab")
	h.mustShow("The rite is pure")
	h.keys("enter")
	h.mustShow("The rite theme is consecrated")
	h.sealedScripture("theme", `{
  "pattern": "Mark II",
  "aspects": ["dark", "porpl"],
  "subjects": [
    {
      "name": "current",
      "designation": "tether",
      "tether": "current-theme",
      "omens": {"*": "themes/{{aspect}}"}
    },
    {
      "name": "palette",
      "designation": "transcription",
      "vessel": "palette.conf",
      "omens": {"dark": "dusk", "*": null},
      "seal": "0600"
    }
  ]
}
`)
}

// TestWizard_ConsecrateAnEdictOfMarkII: an edict of pattern Mark II bears
// its vox-casts as a list of the rite, and its liturgy only incantations
// and litanies, each written in the designation form.
func TestWizard_ConsecrateAnEdictOfMarkII(t *testing.T) {
	h := newHarness(t)
	h.keys("n")
	h.typeText("capture")
	h.choosePatternMarkII()
	h.keys("down")
	h.typeText("Capture the visage")
	h.keys("down", "space")
	h.mustShow("Consecration of a new edict", "◆ edict", "◆ Mark II", "Vox-cast of success")
	h.mustNotShow("Aspects", "Auspex")
	h.keys("down")
	h.typeText("region")
	h.keys("down", "down", "space", "ctrl+s")
	h.mustShow(`Inscription "region"`, "Decree")
	h.mustNotShow("for aspect")
	h.typeText("the slurp geometry")
	h.keys("tab", "tab")
	h.typeText("chosen")
	h.keys("ctrl+s")
	h.mustShow("✔ Rite", "2 Liturgy", "The liturgy of the edict capture")

	h.keys("a")
	h.mustShow("1 » incantation", "2 ≡ litany")
	h.mustNotShow("3 ✉", "sanctum")
	h.keys("3") // no third kind for an edict of Mark II
	h.keys("1", "tab", "space", "ctrl+s")
	h.mustShow("Tongue", "Patience")
	h.keys("tab")
	h.typeText("2m")
	h.keys("ctrl+s")
	h.mustShow("Command", "Reversion")
	h.typeText("grim -g {{inscription.region}} ~/visage.png")
	h.keys("tab")
	h.typeText("rm -f ~/visage.png")
	h.keys("ctrl+s", "a", "1", "ctrl+s")
	h.typeText("wl-copy < ~/visage.png")
	h.keys("ctrl+s")
	h.mustShow("1 » grim", "2 » wl-copy")

	h.keys("tab")
	h.mustShow("The edict is pure", `"pattern": "Mark II"`)
	h.keys("enter")
	h.mustShow("The edict capture is consecrated")
	h.sealedScripture("capture", `{
  "pattern": "Mark II",
  "purpose": "Capture the visage",
  "edict": true,
  "inscriptions": {
    "region": {"purpose": "the slurp geometry", "decrees": "chosen"}
  },
  "vox-cast": ["success"],
  "liturgy": [
    {
      "designation": "incantation",
      "words": "grim -g {{inscription.region}} ~/visage.png",
      "patience": "2m0s",
      "reversion": "rm -f ~/visage.png"
    },
    {"designation": "incantation", "words": "wl-copy < ~/visage.png"}
  ]
}
`)
}

// Scriptures of pattern Mark II that the wizard amends and replicates: a
// rite of sanctums, a rite of a tether and a transcription, and an edict,
// each bearing every key the wizard writes for its form. $DATA is the
// place of their vessels.
const (
	sanctumsII = `{
  "$schema": "https://example.org/rite.schema.json",
  "pattern": "Mark II",
  "purpose": "Hide the cursor",
  "aspects": ["on", "off"],
  "inscriptions": {
    "reason": {"purpose": "why"},
    "mode": {"mandatory": true, "decrees": {"on": "hide", "*": "show"}}
  },
  "tongue": "sh",
  "vox-cast": ["success", "progress"],
  "subjects": [
    {"name": "niri", "designation": "sanctum", "purpose": "the cursor of niri", "vessel": "$DATA/niri.kdl",
     "ward": "cursor-ii", "glyph": "//", "consecrate": true,
     "omens": {"on": "hide-cursor", "off": {"tome": "tomes/{{aspect}}.kdl", "illuminate": true}}},
    {"name": "css", "designation": "sanctum", "exalted": true, "vessel": "$DATA/x.css", "closing-glyph": "*/",
     "omens": {"*": ["a", "b"]}}
  ]
}`
	tetherII = `{
  "pattern": "Mark II",
  "aspects": ["dark", "porpl", "dusk"],
  "subjects": [
    {"name": "current", "designation": "tether", "tether": "current-theme", "zeal": true,
     "omens": {"dusk": null, "*": "themes/{{aspect}}"}},
    {"name": "palette", "designation": "transcription", "vessel": "palette.conf", "seal": "0640", "zeal": true,
     "omens": {"dark": "dusk", "porpl": {"tome": "p.conf"}, "*": null}}
  ]
}`
	edictII = `{
  "pattern": "Mark II",
  "purpose": "Capture the visage",
  "edict": true,
  "inscriptions": {"region": {"purpose": "the slurp geometry", "decrees": "chosen"}},
  "tongue": "sh",
  "vox-cast": ["progress", "success"],
  "liturgy": [
    {"designation": "incantation", "patience": "2m", "words": "echo {{inscription.region}} > $DATA/captured",
     "reversion": "rm -f $DATA/captured"},
    {"designation": "litany", "scroll": "$DATA/shot.sh", "offerings": ["--to", "~/{{inscription.region}}"],
     "tongue": "bash"}
  ]
}`
)

// TestWizard_AmendAndReplicateMarkII: a rite of sanctums, a rite of a
// tether and a transcription, and an edict, all of pattern Mark II, are
// amended through every page and replicated, and each reads back as the
// very rite it was.
func TestWizard_AmendAndReplicateMarkII(t *testing.T) {
	cases := []struct {
		name, scripture string
		amend           []string // the keys from the Rite page to the Seal
		shown           []string // shown upon the pages amended
	}{
		{"cursor", sanctumsII, []string{"ctrl+s", "ctrl+s", "ctrl+s", "enter", "ctrl+s", "ctrl+s", "j", "enter",
			"ctrl+s", "ctrl+s", "tab"}, []string{"1 § niri", "2 § css"}},
		{"theme", tetherII, []string{"ctrl+s", "enter", "ctrl+s", "ctrl+s", "ctrl+s", "j", "enter", "ctrl+s",
			"ctrl+s", "ctrl+s", "tab"}, []string{"1 ↣ current", "2 ¶ palette"}},
		{"capture", edictII, []string{"ctrl+s", "ctrl+s", "enter", "ctrl+s", "ctrl+s", "ctrl+s", "j", "enter",
			"ctrl+s", "ctrl+s", "ctrl+s", "tab"}, []string{"1 » echo", "2 ≡ "}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness(t)
			write(t, filepath.Join(h.dir, "rites", "tomes", "off.kdl"), "show-cursor\n")
			// The tome and the anchors of the theme stand, so that the
			// Inquisition at the Seal finds the machine pure.
			write(t, filepath.Join(h.dir, "rites", "p.conf"), "porpl\n")
			for _, anchor := range []string{"dark", "porpl"} {
				write(t, filepath.Join(h.dir, "rites", "themes", anchor), anchor+"\n")
			}
			h.writeRite(c.name, c.scripture)
			before := *h.loadRite(c.name)
			h.keys("r", "/")
			h.typeText(c.name)
			h.keys("enter", "e")
			h.mustShow("Amendment of the", c.name, "pattern Mark II")
			h.mustNotShow("Pattern", "Auspex")
			h.keys(c.amend[:len(c.amend)-1]...)
			h.mustShow(c.shown...)
			h.keys(c.amend[len(c.amend)-1])
			h.mustShow("is pure", `"pattern": "Mark II"`)
			h.keys("enter")
			h.mustShow("is amended")
			if got := h.loadRite(c.name); !got.Equal(&before) {
				data, _ := librarium.Marshal(got)
				t.Fatalf("the amended rite differs:\n%s", data)
			}

			h.keys("c")
			h.mustShow("Consecration of a new", c.name+"-copy")
			h.mustNotShow("Pattern")
			// A replica claims the wards of its original, which is struck so
			// that the Inquisition finds no clash.
			if err := os.Remove(filepath.Join(h.dir, "rites", c.name+".json")); err != nil {
				t.Fatal(err)
			}
			h.keys(c.amend...)
			h.mustShow("is pure")
			h.keys("enter")
			h.mustShow(c.name + "-copy is consecrated")
			want := before
			want.Name = c.name + "-copy"
			if got := h.loadRite(c.name + "-copy"); !got.Equal(&want) {
				data, _ := librarium.Marshal(got)
				t.Fatalf("the replica differs:\n%s", data)
			}
		})
	}
}
