package cli

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// spiritRite is a rite of one machine-spirit "spirit", whose omens are the
// names of the aspects on and off, read by the scan words.
func spiritRite(scan string) string { return patientSpiritRite(scan, "") }

// patientSpiritRite is spiritRite whose auspex labours for patience ("":
// as long as the codex grants).
func patientSpiritRite(scan, patience string) string {
	if patience != "" {
		patience = `, "patience": "` + patience + `"`
	}
	return `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [{"name": "spirit",
  "designation": "machine-spirit", "auspex": {"scan": ` + strconv.Quote(scan) + patience + `}, "enact": {"*": "true"}}]}`
}

// sloth writes the rite "sloth", whose auspex shows the target during an
// invocation, yet awakened outside one marks that it was, then labours long
// and falls; it returns the path of the mark.
func (e *env) sloth() string {
	e.t.Helper()
	mark := filepath.Join(e.t.TempDir(), "awakened")
	e.addRite("rites/sloth.json", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [{"name": "spirit",
  "designation": "machine-spirit", "enact": {"*": "true"},
  "auspex": {"scan": "if [ -n \"$SERVITOR_ASPECT\" ]; then echo \"$SERVITOR_ASPECT\"; exit 0; fi; touch `+mark+`; sleep 5; exit 1",
             "patience": "10s"}}]}`)
	return mark
}

// census reads `census --binharic` with the further runes, by rite.
func (e *env) census(runes ...string) map[string]map[string]any {
	e.t.Helper()
	var entries []map[string]any
	if err := json.Unmarshal([]byte(e.mustRun(append([]string{"census", "--binharic"}, runes...)...)), &entries); err != nil {
		e.t.Fatal(err)
	}
	byRite := map[string]map[string]any{}
	for _, entry := range entries {
		byRite[entry["rite"].(string)] = entry
	}
	return byRite
}

// TestCensus_EveryEntryMarksWhatTheAuspexGave: every entry of the binharic
// census bears "auspex": heeded, mute, or "" for a rite whose subjects bear
// no auspex, an edict and a heretical rite.
func TestCensus_EveryEntryMarksWhatTheAuspexGave(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", edictJSON)
	e.addRite("rites/broken.json", `{`)
	e.addRite("rites/dnd.json", spiritRite("echo off"))
	e.addRite("rites/fallen.json", spiritRite("exit 1"))

	entries := e.census()

	want := map[string]string{"capture": "", "broken": "", "dnd": "heeded", "fallen": "mute", "mouse-autohide-toggle": ""}
	for rite, marker := range want {
		got, ok := entries[rite]["auspex"]
		if !ok || got != marker {
			t.Errorf("%s: auspex %v (present %v), want %q", rite, got, ok, marker)
		}
	}
	if entries["dnd"]["aspect"] != "off" || entries["fallen"]["aspect"] != "" || len(entries) != len(want) {
		t.Errorf("census %v", entries)
	}
}

// TestSpareAuspex_TheCensusAwakensNoAuspex: with --spare-auspex the census
// answers at once from the other omens, the omens recorded on the
// data-slate and the data-slate, marking forgone every rite whose subject
// keeps an auspex; its table is spoken as ever, a recorded omen marked ~.
func TestSpareAuspex_TheCensusAwakensNoAuspex(t *testing.T) {
	e := newEnv(t)
	mark := e.sloth()
	e.addRite("rites/capture.json", edictJSON)
	e.mustRun("invoke", "sloth", "on", "-s")

	start := time.Now()
	entries := e.census("--spare-auspex")
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("the census waited %v for a spared auspex", elapsed)
	}
	if _, err := os.Stat(mark); err == nil {
		t.Fatal("the census awakened a spared auspex")
	}
	if got := entries["sloth"]; got["auspex"] != "forgone" || got["aspect"] != "on" || got["standing"] != "performed" {
		t.Errorf("sloth %v, want forgone, on and performed from its data-slate", got)
	}
	for _, rite := range []string{"capture", "mouse-autohide-toggle"} {
		if got := entries[rite]["auspex"]; got != "" {
			t.Errorf("%s: auspex %v, want \"\"", rite, got)
		}
	}
	out := e.mustRun("census", "--spare-auspex")
	if !regexp.MustCompile(`(?m)^RITE\s+ASPECT\s+STANDING\s+ASPECTS\s+LAST RITE\s+PURPOSE$`).MatchString(out) ||
		!regexp.MustCompile(`(?m)^sloth\s+~on\s+performed\s+on\|off\s+triumph `).MatchString(out) ||
		strings.Contains(out, "forgone") {
		t.Errorf("census --spare-auspex:\n%s", out)
	}
	if _, err := os.Stat(mark); err == nil {
		t.Fatal("the spoken census awakened a spared auspex")
	}
}

// TestSpareAuspex_TheAuguryAwakensNoAuspex: with --spare-auspex the augury
// marks the auspex forgone and never awakens it, in every manner of reading;
// the omen recorded at the invocation stands in its stead, stale.
func TestSpareAuspex_TheAuguryAwakensNoAuspex(t *testing.T) {
	e := newEnv(t)
	mark := e.sloth()
	e.mustRun("invoke", "sloth", "off", "-s")

	start := time.Now()
	var reading map[string]any
	if err := json.Unmarshal([]byte(e.mustRun("augury", "sloth", "--spare-auspex")), &reading); err != nil {
		t.Fatal(err)
	}
	omens := reading["omens"].([]any)
	if reading["auspex"] != "forgone" || reading["aspect"] != "off" || len(omens) != 1 ||
		omens[0].(map[string]any)["stale"] != true {
		t.Errorf("augury --spare-auspex: %v", reading)
	}
	if got := e.mustRun("augury", "sloth", "aspect", "--spare-auspex"); got != "off\n" {
		t.Errorf("augury sloth aspect --spare-auspex = %q", got)
	}
	if _, errOut, code := e.run("augury", "sloth", "--is", "off", "--spare-auspex"); code != 0 {
		t.Errorf("augury sloth --is off --spare-auspex: exit %d, %s", code, errOut)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("the augury waited %v for a spared auspex", elapsed)
	}
	if _, err := os.Stat(mark); err == nil {
		t.Fatal("the augury awakened a spared auspex")
	}
	if out := e.mustRun("augury", "mouse-autohide-toggle", "--spare-auspex"); !strings.Contains(out, `"auspex": "",`) {
		t.Errorf("a rite without auspex, spared:\n%s", out)
	}
}

// TestAugury_MarksAHeededAndAMuteAuspex: without the rune the augury awakens
// the auspex and marks what it gave.
func TestAugury_MarksAHeededAndAMuteAuspex(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/dnd.json", spiritRite("echo on"))
	e.addRite("rites/fallen.json", spiritRite("echo porpl"))
	for rite, want := range map[string]string{"dnd": `"auspex": "heeded",`, "fallen": `"auspex": "mute",`} {
		if out := e.mustRun("augury", rite); !strings.Contains(out, want) {
			t.Errorf("augury %s lacks %s:\n%s", rite, want, out)
		}
	}
}
