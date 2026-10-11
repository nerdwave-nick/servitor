package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// TestWizard_ConsecrateAnEdict: an edict of one inscription and one
// incantation is consecrated by key presses alone. The Rite page asks no
// aspects and no auspex once the edict is chosen, the inscription takes
// one plain decree, the liturgy offers only what an edict may speak, and
// the command is spoken once; the scripture sealed is the edict exactly.
func TestWizard_ConsecrateAnEdict(t *testing.T) {
	h := newHarness(t)
	h.keys("n")
	h.typeText("capture")
	h.keys("tab")
	h.typeText("Capture the visage")
	h.keys("tab")
	h.mustShow("◆ aspects", "◇ edict", "Aspects", "Auspex")
	h.keys("space")
	h.mustShow("Consecration of a new edict", "◆ edict", "capture", "Capture the visage", "Inscriptions", "Tongue")
	h.mustNotShow("Aspects", "Auspex", "Patience of the auspex")
	h.keys("tab")
	h.typeText("region")
	h.keys("ctrl+s")
	h.mustShow(`Inscription "region"`, "Purpose", "Mandatory", "Decree")
	h.mustNotShow("for aspect", "every other aspect")
	h.typeText("the slurp geometry")
	h.keys("tab", "tab")
	h.typeText("full")
	h.keys("ctrl+s")
	h.mustShow("The liturgy of the edict capture", "No step is written yet")

	h.keys("a")
	h.mustShow("1 » incantation", "2 ≡ litany", "3 ✉ vox-cast")
	h.mustNotShow("sanctum", "transcription", "tether")
	h.keys("4") // no fourth kind for an edict
	h.mustShow("1 » incantation")
	h.keys("1")
	h.mustShow("» incantation", "Further rites", "Its command is spoken on the page that follows")
	h.keys("ctrl+s")
	h.mustShow("Command", "2/2")
	h.mustNotShow("for aspect", "Same command", "{{aspect}}")
	h.typeText("grim -g {{inscription.region}} ~/visage.png")
	h.keys("ctrl+s")
	h.mustShow("1 » grim -g {{inscription.region}}")

	h.keys("tab")
	h.mustShow("The edict is pure", `"edict": true`, "rites/capture.json")
	h.mustNotShow(`"aspects"`)
	h.keys("enter")
	h.mustShow("The edict capture is consecrated")

	want := `{
  "pattern": "Mark I",
  "purpose": "Capture the visage",
  "edict": true,
  "inscriptions": {
    "region": {"purpose": "the slurp geometry", "decrees": "full"}
  },
  "liturgy": [
    {"incantation": "grim -g {{inscription.region}} ~/visage.png"}
  ]
}
`
	data, err := os.ReadFile(filepath.Join(h.dir, "rites", "capture.json"))
	if err != nil || string(data) != want {
		t.Fatalf("the scripture sealed (%v):\n%s\nwant:\n%s", err, data, want)
	}
	if r := h.loadRite("capture"); !r.Edict || r.Aspects != nil || r.Auspex != nil {
		t.Fatalf("the scripture does not load as an edict: %+v", r)
	}
}

// fullEdict is an edict whose steps bear every further rite an edict may.
const fullEdict = `{
  "pattern": "Mark I",
  "purpose": "Capture the visage",
  "edict": true,
  "inscriptions": {"region": {"purpose": "the slurp geometry", "decrees": "full"}},
  "tongue": "sh",
  "liturgy": [
    {"incantation": "echo {{inscription.region}} > $DATA/captured", "patience": "2m", "reversion": "rm -f $DATA/captured"},
    {"litany": "$DATA/shot.sh", "offerings": ["--to", "~/{{inscription.region}}"], "tongue": "bash"},
    {"vox-cast": "success"}
  ]
}`

// TestWizard_AmendAndReplicateAnEdict: an edict amended or replicated
// through every page stays an edict, each value spoken once.
func TestWizard_AmendAndReplicateAnEdict(t *testing.T) {
	h := newHarness(t)
	h.writeRite("capture", fullEdict)
	h.keys("r", "g")
	before := *h.m.s.Librarium.Rites["capture"]

	h.keys("e")
	h.mustShow("Amendment of the edict capture", "◆ edict", "Capture the visage", "region")
	h.mustNotShow("Auspex", "Aspects")
	h.keys("ctrl+s")
	h.mustShow(`Inscription "region"`, "Decree", "full")
	h.mustNotShow("for aspect")
	h.keys("ctrl+s")
	h.mustShow("1 » echo {{inscription.region}}", "2 ≡ ", "3 ✉ success")

	h.keys("enter") // the incantation, its further rites unveiled
	h.mustShow("» incantation", "Further rites", "1/3")
	h.keys("ctrl+s")
	h.mustShow("Tongue", "Patience", "2/3")
	h.keys("ctrl+s")
	h.mustShow("Command", "Reversion", "rm -f", "3/3")
	h.mustNotShow("for aspect", "Same command", "Same reversion")
	h.keys("ctrl+s")

	h.keys("j", "enter") // the litany and its offerings
	h.keys("ctrl+s", "ctrl+s")
	h.mustShow("Scroll", "Offerings", "--to", "~/{{inscription.region}}")
	h.mustNotShow("for aspect", "Same offerings")
	h.keys("ctrl+s")
	h.mustShow("The liturgy of the edict capture")

	h.keys("tab")
	h.mustShow("The edict is pure", `"edict": true`)
	h.keys("enter")
	h.mustShow("The edict capture is amended")
	if got := h.loadRite("capture"); !got.Equal(&before) {
		data, _ := librarium.Marshal(got)
		t.Fatalf("the amended edict differs:\n%s", data)
	}

	h.keys("g", "c")
	h.mustShow("Consecration of a new edict", "capture-copy", "◆ edict")
	h.keys("ctrl+s", "ctrl+s", "tab")
	h.mustShow("The edict is pure")
	h.keys("enter")
	h.mustShow("The edict capture-copy is consecrated")
	want := before
	want.Name = "capture-copy"
	if got := h.loadRite("capture-copy"); !got.Equal(&want) {
		data, _ := librarium.Marshal(got)
		t.Fatalf("the replica differs:\n%s", data)
	}
}

// TestWizard_ARiteTurnedEdictMustPassTheInquisition: a rite declared an
// edict keeps its sanctum and its {{aspect}} until they are cast out, and
// heresy forbids the sealing until then; the wizard does not write the
// sanctum of an edict.
func TestWizard_ARiteTurnedEdictMustPassTheInquisition(t *testing.T) {
	h := newHarness(t)
	h.keys("e", "tab", "tab")
	h.mustShow("◆ aspects", "Auspex")
	h.keys("space")
	h.mustShow("◆ edict", "mouse", "Hide the cursor")
	h.mustNotShow("Auspex")
	h.keys("ctrl+s")
	h.mustShow(`Inscription "reason"`)
	h.keys("ctrl+s")
	h.mustShow(`Inscription "mode"`, "Decree")
	h.mustNotShow("for aspect")
	h.keys("ctrl+s")
	h.mustShow("The liturgy of the edict mouse", "2 § ")

	h.keys("j", "enter")
	h.mustShow("An edict bears no sanctum", "The liturgy of the edict mouse")
	h.keys("tab")
	h.mustShow("Heresy detected", "sanctum")
	h.keys("enter")
	h.mustShow("Heresy remains")
	if r := h.loadRite("mouse"); r.Edict {
		t.Fatal("a heretical edict was sealed")
	}

	h.keys("esc", "d", "d") // the sanctum, then the incantation speaking {{aspect}}
	h.mustShow("1 ✉ progress", "2 ✉ success")
	h.keys("tab")
	h.mustShow("The edict is pure")
	h.mustNotShow(`"aspects"`, `"auspex"`)
	h.keys("enter")
	h.mustShow("The edict mouse is amended")
	if r := h.loadRite("mouse"); !r.Edict || r.Aspects != nil || r.Auspex != nil {
		t.Fatalf("the rite was not turned edict: %+v", r)
	}
}

// TestWizard_AnEdictTurnedRiteAsksItsAspects: an edict declared a rite of
// aspects again must name them, and keeps its plain values for each.
func TestWizard_AnEdictTurnedRiteAsksItsAspects(t *testing.T) {
	h := newHarness(t)
	h.writeRite("capture", fullEdict)
	h.keys("r", "g", "e", "tab", "tab", "space")
	h.mustShow("Amendment of the rite capture", "◆ aspects", "Aspects", "Auspex")
	h.keys("ctrl+s")
	h.mustShow("a rite without aspects can never be invoked")
	h.typeText("on")
	h.keys("ctrl+s")
	h.mustShow(`Inscription "region"`, `Decree for aspect "on"`, "full")
	h.keys("ctrl+s", "tab")
	h.mustShow("The rite is pure", `"aspects": ["on"]`)
	h.keys("enter")
	h.mustShow("The rite capture is amended")
	r := h.loadRite("capture")
	if r.Edict || len(r.Aspects) != 1 || len(r.Liturgy) != 3 {
		t.Fatalf("the edict was not turned rite: %+v", r)
	}
	if lit := r.Liturgy[1].(*librarium.Litany); len(lit.Offerings) != 2 || !lit.Offerings[1].IsUniform() {
		t.Fatalf("the offerings changed: %+v", lit.Offerings)
	}
	if inc := r.Liturgy[0].(*librarium.Incantation); inc.Patience != 2*time.Minute {
		t.Fatalf("the patience changed: %v", inc.Patience)
	}
}

// TestWizard_TheNatureKeepsWhatWasWritten: turning the Rite page to an
// edict and back keeps the aspects and auspex once written.
func TestWizard_TheNatureKeepsWhatWasWritten(t *testing.T) {
	h := newHarness(t)
	h.keys("n")
	h.typeText("theme")
	h.keys("tab", "tab", "tab", "ctrl+u")
	h.typeText("dark, porpl")
	h.keys("up", "space")
	h.mustNotShow("dark, porpl")
	h.keys("space")
	h.mustShow("◆ aspects", "theme", "dark, porpl")
}
