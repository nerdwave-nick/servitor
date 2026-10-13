package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// elderRite is a rite of pattern Mark I: a sanctum, then a vox-cast of
// success. $DATA is the place of its vessel.
const elderRite = `{
  "pattern": "Mark I",
  "aspects": ["on", "off"],
  "liturgy": [
    {"sanctum": "$DATA/elder.kdl", "scripture": {"on": "a", "off": ""}},
    {"vox-cast": "success"}
  ]
}`

// TestConvert_TheCogitatorConvertsAndToasts: the cogitator converts every
// scripture of pattern Mark I at its first reading, writes it converted,
// keeps its original in rites/.mark-i/, and tells it in a toast that
// fades; a later reading finds nothing left to convert.
func TestConvert_TheCogitatorConvertsAndToasts(t *testing.T) {
	h := newHarness(t)
	scripture := h.writeRite("elder", elderRite)
	original, _ := os.ReadFile(scripture)
	h.renew(librarium.Runes{})
	h.mustShow(`the rite "elder" is converted to pattern "Mark II"`, "elder")
	if kept, _ := os.ReadFile(filepath.Join(h.dir, "rites", ".mark-i", "elder.json")); string(kept) != string(original) {
		t.Error("the original is not kept in rites/.mark-i/")
	}
	if now, _ := os.ReadFile(scripture); !strings.Contains(string(now), `"pattern": "Mark II"`) {
		t.Errorf("the rite is not converted on disk:\n%s", now)
	}
	if h.m.Init() == nil {
		t.Error("the toast of the conversion never fades")
	}
	h.keys("r")
	h.mustShow("elder")
	h.mustNotShow("is converted")
}
