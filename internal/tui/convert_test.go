package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// TestConvert_TheCogitatorConvertsAndToasts: a cogitator that converts
// writes every scripture of pattern Mark I converted, keeps its original
// in rites/.mark-i/, and tells it in a toast; one that does not converts
// nothing.
func TestConvert_TheCogitatorConvertsAndToasts(t *testing.T) {
	h := newHarness(t)
	scripture := filepath.Join(h.dir, "rites", "mouse.json")
	original, _ := os.ReadFile(scripture)
	h.mustNotShow("is converted")
	if now, _ := os.ReadFile(scripture); string(now) != string(original) {
		t.Fatal("a cogitator that does not convert converted")
	}
	h.renew(librarium.Runes{}, true)
	h.mustShow(`the rite "mouse" is converted to pattern "Mark II"`, "mouse")
	if kept, _ := os.ReadFile(filepath.Join(h.dir, "rites", ".mark-i", "mouse.json")); string(kept) != string(original) {
		t.Error("the original is not kept in rites/.mark-i/")
	}
	if now, _ := os.ReadFile(scripture); !strings.Contains(string(now), `"pattern": "Mark II"`) {
		t.Errorf("the rite is not converted on disk:\n%s", now)
	}
	if h.m.Init() == nil {
		t.Error("the toast of the conversion never fades")
	}
	h.keys("r")
	h.mustShow("mouse")
}
