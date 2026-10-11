package cli

import (
	"os"
	"slices"
	"strings"
	"testing"
)

// edictJSON is an edict that leaves a mark beside the vessel, were it ever
// performed.
const edictJSON = `{"pattern": "Mark I", "purpose": "Capture the visage of the machine", "edict": true,
  "inscriptions": {"region": {"purpose": "the slurp geometry", "decrees": "full"}},
  "liturgy": [{"incantation": "echo {{inscription.region}} > $TARGET.edict"}, {"vox-cast": "success"}]}`

// TestEdict_NoRitualFallsBeforeIt: a Librarium holding an edict is read,
// listed, examined, expounded and completed by every ritual without one of
// them falling.
func TestEdict_NoRitualFallsBeforeIt(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", edictJSON)
	for _, args := range [][]string{
		{"census"}, {"census", "--binharic"}, {"inquisition"}, {"inquisition", "capture", "--binharic"},
		{"completion", "bash"}, {"completion", "zsh"}, {"invoke", "--help"}, {"invoke", "capture", "--help"},
	} {
		if out, errOut, code := e.run(args...); code != 0 {
			t.Errorf("servitor %v: exit %d\nstdout: %s\nstderr: %s", args, code, out, errOut)
		}
	}
	if out := e.mustRun("census"); !strings.Contains(out, "capture") {
		t.Errorf("the census does not list the edict:\n%s", out)
	}
	if out := e.mustRun("expound", "edict"); !strings.Contains(out, "+++ edict · as written in the codex +++") {
		t.Errorf("expound edict recites otherwise:\n%s", out)
	}
	help := e.mustRun("invoke", "capture", "--help")
	for _, want := range []string{"servitor invoke capture --region", "Edict: proclaimed whole, bearing no aspect",
		"verse 1 · incantation echo {{inscription.region}} > " + e.target + ".edict", "verse 2 · vox-cast success"} {
		if !strings.Contains(help, want) {
			t.Errorf("the edict's own help lacks %q:\n%s", want, help)
		}
	}
	if strings.Contains(help, "<>") || strings.Contains(help, "Aspects:") || strings.Contains(help, "per aspect") {
		t.Errorf("the edict's own help speaks of aspects:\n%s", help)
	}
	if rites, _ := e.complete("invoke", ""); !slices.Contains(rites, "capture") {
		t.Errorf("invoke completes %v without the edict", rites)
	}
	if aspects, directive := e.complete("invoke", "capture", ""); len(aspects) != 0 || directive != noFileComp {
		t.Errorf("an edict completes aspects %v (%s)", aspects, directive)
	}
	if words, _ := e.complete("invoke", "capture", "--region", ""); !slices.Contains(words, "full") {
		t.Errorf("the edict's decree does not complete: %v", words)
	}
}

// TestEdict_IsRefusedAndNeverPerformed: until the servitor learns to
// proclaim an edict, invoking one is refused with a lament, and nothing of
// its liturgy is performed.
func TestEdict_IsRefusedAndNeverPerformed(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", edictJSON)
	for _, args := range [][]string{
		{"invoke", "capture"}, {"invoke", "capture", "on"}, {"invoke", "capture", "--region", "x"},
		{"invoke", "capture", "--foresee"},
	} {
		_, errOut, code := e.run(args...)
		if code == 0 || code == 2 && strings.Contains(errOut, "panic") || !strings.Contains(errOut, `the edict "capture"`) {
			t.Errorf("servitor %v: exit %d\nstderr: %s", args, code, errOut)
		}
	}
	if _, err := os.Stat(e.target + ".edict"); err == nil || e.chronicleLines() != nil || e.notified() != "" {
		t.Fatal("a refused edict touched the machine, the chronicle or the desktop")
	}
}

// TestInvoke_HelpNamesThePlainWordsOfAVerse: a verse written once for every
// aspect is named by its words; only a verse of an aspect map is named as
// varying per aspect.
func TestInvoke_HelpNamesThePlainWordsOfAVerse(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/dnd.json", `{"pattern": "Mark I", "aspects": ["on", "off"], "liturgy": [
	  {"incantation": "makoctl mode -t dnd"}, {"litany": {"on": "a", "off": "b"}}, {"litany": {"*": "c"}}]}`)
	help := e.mustRun("invoke", "dnd", "--help")
	for _, want := range []string{"verse 1 · incantation makoctl mode -t dnd", "verse 2 · litany (per aspect)",
		"verse 3 · litany (per aspect)"} {
		if !strings.Contains(help, want) {
			t.Errorf("the help lacks %q:\n%s", want, help)
		}
	}
}
