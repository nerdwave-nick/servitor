package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/nerdwave-nick/servitor/internal/chronicle"
)

// TestMarkII_TheCogitatorStandsFirmAndPerformsNothing: rites of pattern
// Mark II are listed and shown beside those of Mark I, and every key upon
// them answers without the cogitator falling; a state rite whose subject
// names an observance is not performed (its edicts are proclaimed, see
// edict_markii_test.go, and other state rites invoked, see
// TestMarkII_TheCogitatorInvokesAStateRite). The wizard opens upon an
// edict, and refuses to amend or replicate a rite of machine-spirits, whose
// form it knows not yet, rather than lose it.
func TestMarkII_TheCogitatorStandsFirmAndPerformsNothing(t *testing.T) {
	h := newHarness(t)
	gaming := h.writeRite("gaming", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "m", "designation": "machine-spirit", "auspex": {"scan": "cat $DATA/m"}, "enact": {"*": "echo {{omen}} > $DATA/m"},
	   "introit": ["herald"]}],
	  "observances": [{"name": "herald", "designation": "incantation", "words": "touch $DATA/herald"}]}`)
	h.writeRite("capture", `{"pattern": "Mark II", "edict": true, "liturgy": [
	  {"designation": "incantation", "words": "touch $DATA/captured"}]}`)
	scripture, _ := os.ReadFile(gaming)
	h.keys("r", "g")
	h.mustShow("Rites (3)", "capture", "gaming", "mouse")
	h.keys("e")
	h.mustShow("Amendment of the edict capture", `pattern Mark II`)
	h.keys("esc", "c")
	h.mustShow("Consecration of a new edict", "capture-copy")
	h.keys("esc", "j")
	for _, key := range []string{"e", "c"} {
		h.keys(key)
		h.mustShow(`machine-spirits and observances of pattern "Mark II"`, "rite gaming")
		h.mustNotShow("Amendment of", "Consecration of")
		h.keys("esc")
	}
	refused := func(keys ...string) {
		t.Helper()
		h.keys("r")
		h.mustNotShow("yet the rite of performance")
		h.keys(keys...)
		h.await()
		h.mustShow(`is written in pattern "Mark II"`, "yet the rite of performance")
	}
	refused("g", "j", "space")
	h.mustShow(`the rite "gaming"`)
	refused("esc", "g", "j", "enter", "enter")
	h.mustShow(`the rite "gaming"`)
	h.keys("esc", "h", "esc", "i", "esc", "?", "esc", "j", "k")
	if _, err := os.Stat(filepath.Join(h.dataDir, "m")); err == nil {
		t.Fatal("a rite of aspects of Mark II was performed")
	}
	if _, err := os.Stat(filepath.Join(h.dataDir, "herald")); err == nil {
		t.Fatal("an observance was spoken")
	}
	if after, _ := os.ReadFile(gaming); string(after) != string(scripture) {
		t.Fatalf("the scripture of Mark II was written anew:\n%s", after)
	}
	h.desktopSilent()
}

// TestMarkII_TheCogitatorInvokesAStateRite: a state rite of pattern Mark II
// is invoked from the cogitator as every rite is, its subjects set.
func TestMarkII_TheCogitatorInvokesAStateRite(t *testing.T) {
	h := newHarness(t)
	h.writeRite("gaming", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "m", "designation": "machine-spirit", "auspex": {"scan": "cat $DATA/m"}, "enact": {"*": "echo {{omen}} > $DATA/m"}}]}`)
	h.keys("r", "g", "enter", "enter")
	h.await()
	h.mustShow("The rite gaming is performed", "→ on")
	if got, _ := os.ReadFile(filepath.Join(h.dataDir, "m")); string(got) != "on\n" {
		t.Fatalf("the subject holds %q", got)
	}
}

// TestMarkII_TheFallOfAStateRiteNamesItsSubjects: the fall of a state rite
// and its entry in the chronicle name the subject that fell and every
// subject reverted, never a verse.
func TestMarkII_TheFallOfAStateRiteNamesItsSubjects(t *testing.T) {
	h := newHarness(t)
	h.writeRite("gaming", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "m", "designation": "machine-spirit", "auspex": {"scan": "cat $DATA/m"}, "enact": {"*": "echo {{omen}} > $DATA/m"}},
	  {"name": "z", "designation": "machine-spirit", "enact": {"*": "exit 3"}}]}`)
	h.keys("r", "g", "enter", "enter")
	h.await()
	h.mustShow("It fell at subject z", "subject z could not be undone", "subject m could not be undone")
	h.mustNotShow("verse 0")
	entries, err := chronicle.Read(filepath.Join(os.Getenv("XDG_STATE_HOME"), "servitor", "chronicle.jsonl"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("chronicle %v, %v", entries, err)
	}
	whole := ansi.Strip(h.m.chronicled(entries[0]))
	for _, want := range []string{"It fell at subject z", "subject z could not be undone", "subject m could not be undone"} {
		if !strings.Contains(whole, want) {
			t.Errorf("the entry lacks %q:\n%s", want, whole)
		}
	}
	if strings.Contains(whole, "verse") {
		t.Errorf("the entry names a verse:\n%s", whole)
	}
}
