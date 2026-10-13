package cli

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// edictJSON is an edict that leaves a mark beside the vessel each time it
// is proclaimed.
const edictJSON = `{"pattern": "Mark II", "purpose": "Capture the visage of the machine", "edict": true,
  "inscriptions": {"region": {"purpose": "the slurp geometry", "decrees": "full"}},
  "vox-cast": ["progress", "success"],
  "liturgy": [{"designation": "incantation", "words": "echo {{inscription.region}} >> $TARGET.edict"},
    {"designation": "incantation", "words": "true"}]}`

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
		"verse 1 · incantation echo {{inscription.region}} >> " + e.target + ".edict", "verse 2 · incantation true"} {
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
	if out := e.mustRun("__complete", "invoke", "capture", "--region", ""); !strings.Contains(out, "full\tdecreed by the edict\n") ||
		strings.Contains(out, "*") {
		t.Errorf("the edict's decree completes as:\n%s", out)
	}
}

// aspectless matches what an edict's report, vox-casts and laments never
// speak: an aspect, «», or "former → aspect".
var aspectless = regexp.MustCompile(`(?i)«|»|→|\baspects?\b`)

// plainGlosses are plain words no lament may speak.
var plainGlosses = regexp.MustCompile(`(?i)\b(error|warning|switch|states?|metadata|config|invalid|required|` +
	`description|file|field|timeout|symlink|argument|directory|fail(ed|s|ure)?|status)\b`)

func grimdark(t *testing.T, words string) {
	t.Helper()
	if m := plainGlosses.FindString(words); m != "" {
		t.Errorf("%q speaks the plain word %q", words, m)
	}
}

func TestEdict_RefusesEveryWordAfterItsName(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", edictJSON)
	for _, tc := range []struct {
		args  []string
		words string
	}{
		{[]string{"invoke", "capture", "on"}, `"on"`},
		{[]string{"invoke", "capture", "taken", "again"}, `"taken" "again"`},
		{[]string{"invoke", "capture", "on", "-f"}, `"on"`},
	} {
		_, errOut, code := e.run(tc.args...)
		if code != 2 || !strings.Contains(errOut, `the edict "capture" bears no aspect`) ||
			!strings.Contains(errOut, "yet "+tc.words+" followed its name") {
			t.Errorf("servitor %v: exit %d\nstderr: %s", tc.args, code, errOut)
		}
		grimdark(t, errOut)
	}
	if _, err := os.Stat(e.target + ".edict"); err == nil || e.chronicleLines() != nil || e.notified() != "" {
		t.Fatal("a refused edict touched the machine, the chronicle or the desktop")
	}
}

func TestEdict_ForeseeingTellsAndPerformsNothing(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", edictJSON)

	out := e.mustRun("invoke", "capture", "-f", "--region", "window")

	got := lines(out)
	if len(got) < 5 || got[0] != "+++ Foreseen: the edict capture +++" || got[1] != "verse 1 · incantation echo window >> "+homeward(e.target)+".edict" ||
		!strings.Contains(got[2], "spoken in the tongue bash") || got[len(got)-1] != "Nothing was performed." || aspectless.MatchString(out) {
		t.Fatalf("foresight:\n%s", out)
	}
	if _, err := os.Stat(e.target + ".edict"); err == nil || e.chronicleLines() != nil || e.notified() != "" {
		t.Fatal("a foreseen edict touched the machine, the chronicle or the desktop")
	}
}

func TestEdict_UnmetInscriptionIsForbiddenByThePreflight(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", `{"pattern": "Mark II", "edict": true,
	  "inscriptions": {"region": {"mandatory": true}},
	  "liturgy": [{"designation": "incantation", "words": "echo {{inscription.region}} >> $TARGET.edict"}]}`)

	_, errOut, code := e.run("invoke", "capture")

	if code != 2 || !strings.Contains(errOut, `the pre-flight forbids the proclaiming of the edict "capture"`) ||
		!strings.Contains(errOut, "--region") || aspectless.MatchString(errOut) {
		t.Fatalf("exit %d\nstderr: %s", code, errOut)
	}
	if _, err := os.Stat(e.target + ".edict"); err == nil || e.chronicleLines() != nil {
		t.Fatal("a forbidden edict touched the machine or the chronicle")
	}
}

// edictCanon is the canon of edictJSON as the augury speaks it.
const edictCanon = "  \"canon\": [\n    {\n      \"key\": \"region\",\n      \"purpose\": \"the slurp geometry\",\n" +
	"      \"mandatory\": false,\n      \"decrees\": {},\n      \"decree\": \"full\"\n    }\n  ],\n"

// TestEdict_AuguryBearsNoAspect: the augury of an edict speaks rite, edict,
// standing, inscriptions and last rite alone, answers its standing and
// inscriptions, and refuses every sign of an aspect, and --is, with a
// heresy (exit 2).
func TestEdict_AuguryBearsNoAspect(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", edictJSON)
	dormant := "{\n  \"rite\": \"capture\",\n  \"edict\": true,\n  \"standing\": \"dormant\",\n  \"sanction\": \"\",\n" +
		"  \"auspex\": \"\",\n  \"inscriptions\": {},\n" + edictCanon + "  \"last_rite\": null\n}\n"
	if out := e.mustRun("augury", "capture"); out != dormant {
		t.Errorf("augury of a dormant edict:\n%s\nwant\n%s", out, dormant)
	}
	e.mustRun("invoke", "capture", "-s", "--region", "window")
	proclaimed := regexp.MustCompile(`^\{\n  "rite": "capture",\n  "edict": true,\n  "standing": "proclaimed",\n  "sanction": "",\n` +
		`  "auspex": "",\n  "inscriptions": \{\n    "region": "window"\n  \},\n` + regexp.QuoteMeta(edictCanon) +
		`  "last_rite": \{\n    "verdict": "triumph",\n    "at": "[0-9T:Z-]+"\n  \}\n\}\n$`)
	if out := e.mustRun("augury", "capture"); !proclaimed.MatchString(out) {
		t.Errorf("augury of a proclaimed edict:\n%s", out)
	}
	for key, want := range map[string]string{"standing": "proclaimed\n", "region": "window\n"} {
		if out := e.mustRun("augury", "capture", key); out != want {
			t.Errorf("augury capture %s = %q, want %q", key, out, want)
		}
	}
	for _, args := range [][]string{
		{"augury", "capture", "aspect"}, {"augury", "capture", "desecrated"}, {"augury", "capture", "former"},
		{"augury", "capture", "--is", "x"}, {"augury", "capture", "--is", "taken"},
	} {
		out, errOut, code := e.run(args...)
		if code != 2 || out != "" || !strings.Contains(errOut, `the edict "capture" bears no aspect`) {
			t.Errorf("servitor %v: exit %d\nstdout: %s\nstderr: %s", args, code, out, errOut)
		}
		grimdark(t, errOut)
	}
	if out, errOut, code := e.run("augury", "capture", "nope"); code != 2 || !strings.Contains(errOut, "(keys: standing, sanction, region)") {
		t.Errorf("an unknown key: exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
}

// TestEdict_AuguryCompletesNoSignOfAnAspect: completion offers an edict's
// standing and inscriptions as keys, and nothing for --is.
func TestEdict_AuguryCompletesNoSignOfAnAspect(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", edictJSON)
	if keys, _ := e.complete("augury", "capture", ""); !slices.Equal(keys, []string{"standing", "sanction", "region"}) {
		t.Errorf("the augury of an edict completes %v", keys)
	}
	if aspects, directive := e.complete("augury", "capture", "--is", ""); len(aspects) != 0 || directive != noFileComp {
		t.Errorf("--is completes %v (%s) for an edict", aspects, directive)
	}
	if keys, _ := e.complete("augury", "mouse-autohide-toggle", ""); !slices.Equal(keys[:4], []string{"aspect", "standing", "desecrated", "former"}) {
		t.Errorf("the augury of a rite completes %v", keys)
	}
}

// TestEdict_InTheCensus: the census shows an edict with · for aspect and
// aspects, its standing and last rite; its binharic marks every rite an
// edict or not, and gives an edict "aspects": [].
func TestEdict_InTheCensus(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", edictJSON)
	if out := e.mustRun("census"); !regexp.MustCompile(`(?m)^capture\s+·\s+dormant\s+·\s+·\s+Capture the visage of the machine$`).MatchString(out) {
		t.Errorf("census of a dormant edict:\n%s", out)
	}
	e.mustRun("invoke", "capture", "-s")
	out := e.mustRun("census")
	if !regexp.MustCompile(`(?m)^capture\s+·\s+proclaimed\s+·\s+triumph \d{4}-\d\d-\d\d \d\d:\d\d\s+Capture the visage of the machine$`).MatchString(out) {
		t.Errorf("census of a proclaimed edict:\n%s", out)
	}
	if !regexp.MustCompile(`(?m)^mouse-autohide-toggle\s+·\s+dormant\s+on\|off\s+·\s+Hide the cursor`).MatchString(out) {
		t.Errorf("census of a rite beside an edict:\n%s", out)
	}
	raw := e.mustRun("census", "--binharic")
	if !strings.Contains(raw, "\"rite\": \"capture\",\n    \"edict\": true,\n") || !strings.Contains(raw, "\"aspects\": [],\n") ||
		!strings.Contains(raw, "\"rite\": \"mouse-autohide-toggle\",\n    \"edict\": false,\n") {
		t.Errorf("census --binharic:\n%s", raw)
	}
}

// TestAugury_ShortSpeaksOfEdictsToo: the augury's own words do not promise
// an aspect of every rite, for an edict bears none.
func TestAugury_ShortSpeaksOfEdictsToo(t *testing.T) {
	e := newEnv(t)
	out := e.mustRun("__complete", "")
	line := ""
	for l := range strings.Lines(out) {
		if strings.HasPrefix(l, "augury\t") {
			line = strings.TrimSpace(strings.TrimPrefix(l, "augury\t"))
		}
	}
	if !strings.Contains(line, "edict") {
		t.Errorf("the augury is described as %q, which tells nothing of edicts", line)
	}
	grimdark(t, line)
}

// TestEdict_WithoutPurposeIsCompletedWithFlavour: an edict that writes no
// purpose is offered with a few words of lore rather than with nothing,
// the same words at every Tab, and never with an aspect.
func TestEdict_WithoutPurposeIsCompletedWithFlavour(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/slumber.json", `{"pattern": "Mark II", "edict": true, "liturgy": [{"designation": "incantation", "words": "true"}]}`)
	for _, ritual := range []string{"invoke", "augury"} {
		first := ""
		for range 3 {
			out := e.mustRun("__complete", ritual, "")
			desc, found := "", false
			for l := range strings.Lines(out) {
				if name, d, ok := strings.Cut(strings.TrimSuffix(l, "\n"), "\t"); ok && name == "slumber" {
					desc, found = d, true
				}
			}
			if !found || strings.TrimSpace(desc) == "" {
				t.Fatalf("__complete %s offers the edict without lore:\n%s", ritual, out)
			}
			if aspectless.MatchString(desc) {
				t.Errorf("the edict's lore %q speaks of an aspect", desc)
			}
			grimdark(t, desc)
			if first == "" {
				first = desc
			} else if desc != first {
				t.Errorf("the edict's lore changes between Tabs: %q, then %q", first, desc)
			}
		}
	}
}
