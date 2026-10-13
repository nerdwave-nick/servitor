package tui

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMarkII_TheCogitatorStandsFirmAndPerformsNothing: rites of pattern
// Mark II are listed and shown beside those of Mark I, and every key upon
// them answers without the cogitator falling; none is performed, and the
// wizard, which knows not yet their form, refuses to amend or replicate
// them rather than write them anew as Mark I.
func TestMarkII_TheCogitatorStandsFirmAndPerformsNothing(t *testing.T) {
	h := newHarness(t)
	gaming := h.writeRite("gaming", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "m", "designation": "machine-spirit", "auspex": {"scan": "cat $DATA/m"}, "enact": {"*": "echo {{omen}} > $DATA/m"}}]}`)
	h.writeRite("capture", `{"pattern": "Mark II", "edict": true, "liturgy": [
	  {"designation": "incantation", "words": "touch $DATA/captured"}]}`)
	scripture, _ := os.ReadFile(gaming)
	h.keys("r", "g")
	h.mustShow("Rites (3)", "capture", "gaming", "mouse")
	for _, row := range []string{"edict capture", "rite gaming"} {
		for _, key := range []string{"e", "c"} {
			h.keys(key)
			h.mustShow(`pattern "Mark II"`, row)
			h.mustNotShow("Amendment of", "Consecrate a rite")
			h.keys("esc")
		}
		h.keys("j")
	}
	refused := func(keys ...string) {
		t.Helper()
		h.keys("r")
		h.mustNotShow("yet the rite of performance")
		h.keys(keys...)
		h.await()
		h.mustShow(`is written in pattern "Mark II"`, "yet the rite of performance")
	}
	refused("g", "space", "enter")
	h.mustShow(`the edict "capture"`)
	refused("esc", "g", "j", "space")
	h.mustShow(`the rite "gaming"`)
	refused("esc", "g", "j", "enter", "enter")
	h.mustShow(`the rite "gaming"`)
	h.keys("esc", "h", "esc", "i", "esc", "?", "esc", "j", "k")
	for _, mark := range []string{"m", "captured"} {
		if _, err := os.Stat(filepath.Join(h.dataDir, mark)); err == nil {
			t.Fatalf("a rite of Mark II was performed: %s", mark)
		}
	}
	if after, _ := os.ReadFile(gaming); string(after) != string(scripture) {
		t.Fatalf("the scripture of Mark II was written anew:\n%s", after)
	}
	h.desktopSilent()
}
