package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// gamingIIJSON is a rite of pattern Mark II whose machine-spirit would leave
// its omen beside the target, and whose sanctum would keep the target.
const gamingIIJSON = `{"pattern": "Mark II", "purpose": "enable or disable gaming optimization",
  "aspects": ["on", "off"], "vox-cast": ["success"],
  "subjects": [
    {"name": "smt", "designation": "machine-spirit", "omens": {"on": "off", "*": "on"},
     "auspex": {"scan": "cat $TARGET.smt"}, "enact": {"*": "echo {{omen}} > $TARGET.smt"}, "introit": ["herald"]},
    {"name": "util", "designation": "sanctum", "vessel": "$TARGET", "omens": {"on": "a", "*": ""}}],
  "observances": [{"name": "herald", "designation": "incantation", "words": "echo heard >> $TARGET.herald"}]}`

// captureIIJSON is an edict of pattern Mark II.
const captureIIJSON = `{"pattern": "Mark II", "edict": true, "vox-cast": ["success"],
  "liturgy": [{"designation": "incantation", "words": "echo proclaimed >> $TARGET.edict"}]}`

// TestMarkII_IsReadByEveryRitualYetNeverInvoked: rites of pattern Mark II
// stand in the Librarium beside those of Mark I; every reading ritual
// heeds them, yet invoking or foreseeing a rite of aspects of that pattern
// is refused before anything is touched, naming the rite of performance
// not yet forged. Its edicts are proclaimed (see edict_markii_test.go).
func TestMarkII_IsReadByEveryRitualYetNeverInvoked(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/gaming.json", gamingIIJSON)
	e.addRite("rites/capture.json", captureIIJSON)
	for _, args := range [][]string{
		{"census"}, {"census", "--binharic"}, {"inquisition"}, {"inquisition", "gaming", "capture", "--binharic"},
		{"augury", "gaming"}, {"augury", "capture"}, {"completion", "bash"}, {"invoke", "--help"},
		{"invoke", "gaming", "--help"}, {"invoke", "capture", "--help"},
	} {
		if out, errOut, code := e.run(args...); code != 0 {
			t.Errorf("servitor %v: exit %d\nstdout: %s\nstderr: %s", args, code, out, errOut)
		}
	}
	if out := e.mustRun("census"); !strings.Contains(out, "gaming") || !strings.Contains(out, "capture") {
		t.Errorf("the census does not list the rites of Mark II:\n%s", out)
	}
	for _, args := range [][]string{{"invoke", "gaming", "on"}, {"invoke", "gaming", "off", "--foresee"},
		{"invoke", "gaming", "on", "--binharic"}} {
		out, errOut, code := e.run(args...)
		if code != markRefused {
			t.Fatalf("servitor %v: exit %d, want %d\nstdout: %s\nstderr: %s", args, code, markRefused, out, errOut)
		}
		for _, want := range []string{`pattern "Mark II"`, "rite of performance", "not yet forged"} {
			if !strings.Contains(errOut, want) {
				t.Errorf("servitor %v: the refusal lacks %q:\n%s", args, want, errOut)
			}
		}
		grimdark(t, errOut)
	}
	for _, mark := range []string{".smt", ".herald"} {
		if _, err := os.Stat(e.target + mark); err == nil {
			t.Errorf("a refused invocation left %s", mark)
		}
	}
	if got := e.targetContent(); got != "input {}\n" {
		t.Errorf("a refused invocation kept a sanctum: %q", got)
	}
	if e.chronicleLines() != nil || e.notified() != "" {
		t.Errorf("a refused invocation was chronicled or proclaimed: %v %q", e.chronicleLines(), e.notified())
	}
	if _, err := os.Stat(filepath.Join(e.state, "servitor", "data-slates")); err == nil {
		t.Error("a refused invocation kept a data-slate")
	}
}
