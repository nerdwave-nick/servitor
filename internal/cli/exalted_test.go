package cli

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// splitLockJSON keeps a common sanctum in the target and speaks one
// exalted observance after it changes.
const splitLockJSON = `{
  "pattern": "Mark II",
  "aspects": ["on", "off"],
  "vox-cast": ["success"],
  "subjects": [
    {"name": "veil", "designation": "sanctum", "vessel": "$TARGET", "omens": {"on": "split-lock 0", "off": ""},
     "recessional": ["exalt"]}
  ],
  "observances": [
    {"name": "exalt", "designation": "incantation", "words": "echo {{aspect}} >> $TARGET.exalted", "exalted": true}
  ]
}`

// TestInvoke_UnsanctionedExaltedVersesAreRefused: invoking a rite or
// proclaiming an edict whose exalted observances or verses were never
// sanctioned exits 2 before anything is touched — no proof, no vessel, no
// speech, no dominion, no chronicle — and the binharic forbiddance names
// the exalted observance; --foresee still reveals it.
func TestInvoke_UnsanctionedExaltedVersesAreRefused(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/split-lock.json", splitLockJSON)
	e.addRite("rites/reboot.json", `{"pattern": "Mark II", "edict": true, "liturgy": [
	  {"designation": "incantation", "words": "echo proclaimed >> $TARGET.exalted", "exalted": true}]}`)
	untouched := func(args []string) {
		t.Helper()
		if got := e.targetContent(); got != "input {}\n" {
			t.Fatalf("%v touched the vessel:\n%s", args, got)
		}
		for _, p := range []string{e.target + ".exalted", filepath.Join(e.state, "servitor")} {
			if _, err := os.Stat(p); err == nil {
				t.Fatalf("%v touched %s", args, p)
			}
		}
	}
	for _, args := range [][]string{{"invoke", "split-lock", "on"}, {"invoke", "reboot"}} {
		out, errOut, code := e.run(args...)
		if code != 2 || out != "" || !strings.Contains(errOut, "is exalted") || !strings.Contains(errOut, "sanction") ||
			!strings.Contains(errOut, "nothing was touched") {
			t.Fatalf("%v: exit %d\nstdout: %s\nstderr: %s", args, code, out, errOut)
		}
		untouched(args)
	}

	out, errOut, code := e.run("invoke", "split-lock", "off", "--binharic")
	var refusal struct {
		Rite      string   `json:"rite"`
		Aspect    string   `json:"aspect"`
		Forbidden []string `json:"forbidden"`
	}
	if err := json.Unmarshal([]byte(out), &refusal); err != nil || code != 2 {
		t.Fatalf("exit %d, %v\nstdout: %s\nstderr: %s", code, err, out, errOut)
	}
	if refusal.Rite != "split-lock" || refusal.Aspect != "off" || len(refusal.Forbidden) != 1 ||
		!strings.Contains(refusal.Forbidden[0], "split-lock.json:10:") ||
		!strings.Contains(refusal.Forbidden[0], `the observance "exalt" is exalted`) {
		t.Fatalf("refusal %+v", refusal)
	}
	untouched([]string{"--binharic"})

	for _, args := range [][]string{{"invoke", "split-lock", "on", "--foresee"}, {"invoke", "reboot", "-f"}} {
		out := e.mustRun(args...)
		if !strings.Contains(out, "incantation") || !strings.Contains(out, ".exalted") {
			t.Fatalf("%v foresees:\n%s", args, out)
		}
		untouched(args)
	}
}
