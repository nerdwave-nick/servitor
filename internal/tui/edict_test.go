package tui

import (
	"os"
	"path/filepath"
	"testing"
)

// edictRite is an edict that leaves a mark among the vessels, were it ever
// performed. It sorts before the rite "mouse".
const edictRite = `{
  "pattern": "Mark I",
  "purpose": "Capture the visage",
  "edict": true,
  "inscriptions": {"region": {"purpose": "the slurp geometry", "decrees": "full"}},
  "liturgy": [
    {"incantation": "echo {{inscription.region}} > $DATA/captured"},
    {"vox-cast": "success"}
  ]
}`

// TestEdict_TheCogitatorStandsFirmBeforeAnEdict: an edict is listed and
// shown, and every key upon it — space, enter, amend, replicate, the
// chronicle, the verdict — answers without the cogitator falling. space and
// enter only ask before proclaiming it and e and c lead it into the wizard as
// an edict; a single key performs nothing of it.
func TestEdict_TheCogitatorStandsFirmBeforeAnEdict(t *testing.T) {
	h := newHarness(t)
	h.writeRite("capture", edictRite)
	h.keys("r", "g")
	h.mustShow("Rites (2)", "capture", "mouse", "Capture the visage")
	for _, key := range []string{"space", "enter", "e", "c"} {
		h.keys(key)
		h.mustShow("capture", "edict")
		h.mustNotShow("Consecrate a rite", "Choose the aspect")
		h.keys("esc")
		h.mustShow("Rites (2)")
		h.mustNotShow("Proclaim the edict")
	}
	h.keys("h", "esc", "i", "esc", "?", "esc", "j", "k")
	if _, err := os.Stat(filepath.Join(h.dataDir, "captured")); err == nil {
		t.Fatal("the edict was performed")
	}
	h.desktopSilent()
}
