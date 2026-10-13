package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// gamingIIJSON is a rite of pattern Mark II whose machine-spirit would leave
// its omen beside the target, whose sanctum would keep the target, whose
// exalted subject bears a privileged auspex that would leave its mark
// beside it, and whose exalted observance would leave its mark beside it.
const gamingIIJSON = `{"pattern": "Mark II", "purpose": "enable or disable gaming optimization",
  "aspects": ["on", "off"], "vox-cast": ["success"],
  "subjects": [
    {"name": "smt", "designation": "machine-spirit", "omens": {"on": "off", "*": "on"},
     "auspex": {"scan": "cat $TARGET.smt"}, "enact": {"*": "echo {{omen}} > $TARGET.smt"}, "introit": ["herald"]},
    {"name": "util", "designation": "sanctum", "vessel": "$TARGET", "omens": {"on": "a", "*": ""}},
    {"name": "nft", "designation": "machine-spirit", "exalted": true,
     "auspex": {"scan": "echo on > $TARGET.nft; echo on", "exalted": true}, "enact": {"*": "echo {{omen}} > $TARGET.nft"}}],
  "observances": [{"name": "herald", "designation": "incantation", "exalted": true,
    "words": "echo heard >> $TARGET.herald"}]}`

// captureIIJSON is an edict of pattern Mark II.
const captureIIJSON = `{"pattern": "Mark II", "edict": true, "vox-cast": ["success"],
  "liturgy": [{"designation": "incantation", "words": "echo proclaimed >> $TARGET.edict"}]}`

// TestMarkII_IsReadByEveryRitualYetNeverInvokedUnsanctioned: rites of
// pattern Mark II stand in the Librarium; every reading ritual heeds them. Invoking a rite of aspects of that pattern
// whose exalted subject bears a privileged auspex is refused before
// anything is read or touched while it stands unsanctioned, naming the
// ritual that sanctions it; foreseeing it never speaks the privileged
// auspex, which is read at invocation. Its edicts are proclaimed (see
// edict_markii_test.go), and other rites of aspects invoked (see
// invoke_markii_test.go and exalted_markii_test.go).
func TestMarkII_IsReadByEveryRitualYetNeverInvokedUnsanctioned(t *testing.T) {
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
	for _, args := range [][]string{{"invoke", "gaming", "on"}, {"invoke", "gaming", "on", "--binharic"}} {
		out, errOut, code := e.run(args...)
		if code != markRefused {
			t.Fatalf("servitor %v: exit %d, want %d\nstdout: %s\nstderr: %s", args, code, markRefused, out, errOut)
		}
		for _, want := range []string{`the subject "nft" is exalted`, "stands unsanctioned", "servitor sanction gaming"} {
			if !strings.Contains(errOut, want) {
				t.Errorf("servitor %v: the refusal lacks %q:\n%s", args, want, errOut)
			}
		}
		grimdark(t, errOut)
	}
	out := e.mustRun("invoke", "gaming", "off", "--foresee")
	if !strings.Contains(out, `subject nft · machine-spirit ✠: (read at invocation) → "off", changing`) {
		t.Errorf("the foresight does not tell that nft is read at invocation:\n%s", out)
	}
	for _, mark := range []string{".smt", ".herald", ".nft"} {
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

// TestMarkII_TheHelpOfARiteNamesItsSubjectsAndObservances: the help of a
// rite of aspects names each subject by its designation and what it reads
// or sets, then each observance by its words, the exalted marked ✠.
func TestMarkII_TheHelpOfARiteNamesItsSubjectsAndObservances(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/gaming.json", gamingIIJSON)
	help := e.mustRun("invoke", "gaming", "--help")
	want := "Subjects:\n" +
		"  smt · machine-spirit cat " + e.target + ".smt\n" +
		"  util · sanctum " + e.target + "\n" +
		"  nft · machine-spirit ✠ echo on > " + e.target + ".nft; echo on\n" +
		"Observances:\n" +
		"  herald · incantation ✠ echo heard >> " + e.target + ".herald\n"
	if !strings.Contains(help, want) {
		t.Errorf("the rite's help lacks\n%s\nin\n%s", want, help)
	}
}
