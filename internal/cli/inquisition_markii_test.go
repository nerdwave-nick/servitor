package cli

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"
)

// TestInquisitionII_KeepsItsBinharicAndDeathMarks: the inquisition of rites
// of pattern Mark II speaks its findings in the shape it speaks those of
// Mark I — the same keys in binharic, file:line:column in the plain form —
// and ends as before: in triumph over impurities alone, with death-mark 1
// over heresy. --spare-vessels spares the machine of the subjects.
func TestInquisitionII_KeepsItsBinharicAndDeathMarks(t *testing.T) {
	e := newEnv(t)
	gaming := e.addRite("rites/gaming.json", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
  {"name": "smt", "designation": "machine-spirit", "tongue": "no-such-tongue-of-spirits",
   "omens": {"on": "off", "*": "on"}, "auspex": {"scan": "cat $TARGET.smt"}, "enact": {"*": "echo {{omen}}"}}]}`)

	out, errOut, code := e.run("inquisition", "gaming")
	if code != 0 || !strings.Contains(errOut, "1 rite") || !strings.Contains(errOut, "0 heresies, 2 impurities") {
		t.Fatalf("impurities alone: exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	for _, want := range []string{
		gaming + `:2:62: impurity: the tongue "no-such-tongue-of-spirits" of the subject "smt"`,
		gaming + `:3:49: impurity: the tongue "no-such-tongue-of-spirits" of the auspex of the subject "smt"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the inquisition does not speak %q:\n%s", want, out)
		}
	}
	grimdark(t, out)

	out, _, code = e.run("inquisition", "gaming", "--binharic")
	var found []map[string]any
	if err := json.Unmarshal([]byte(out), &found); err != nil || code != 0 || len(found) != 2 {
		t.Fatalf("binharic: exit %d, %v\n%s", code, err, out)
	}
	keys := []string{"column", "denunciation", "judgement", "line", "rite", "scripture"}
	for _, f := range found {
		if got := slices.Sorted(maps.Keys(f)); !slices.Equal(got, keys) || f["rite"] != "gaming" ||
			f["judgement"] != "impurity" || f["scripture"] != gaming || f["line"] == 0.0 || f["column"] == 0.0 {
			t.Errorf("binharic judgement %v", f)
		}
	}

	if out, _, code := e.run("inquisition", "gaming", "--spare-vessels"); code != 0 || out != "" {
		t.Errorf("--spare-vessels: exit %d\n%s", code, out)
	}

	e.addRite("rites/heretic.json", `{"pattern": "Mark II", "aspects": ["on"], "subjects": [{"name": "x"}]}`)
	if out, _, code := e.run("inquisition", "heretic", "gaming", "--binharic"); code != 1 ||
		!strings.Contains(out, `"judgement": "heresy"`) {
		t.Errorf("heresy: exit %d\n%s", code, out)
	}
}
