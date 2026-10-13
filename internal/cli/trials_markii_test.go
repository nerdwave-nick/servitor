package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fanJSON is a state rite whose machine-spirit holds a scan and a trial,
// each leaving its word in $DATA/spoken: "scan", or "trial-<omen>".
const fanJSON = `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
  {"name": "fan", "designation": "machine-spirit", "omens": {"on": "loud", "*": "quiet"},
   "auspex": {"scan": "echo scan >> $DATA/spoken; cat $DATA/fan",
              "trials": {"loud": "echo trial-$SERVITOR_OMEN >> $DATA/spoken; test -e $DATA/loud-holds"}},
   "enact": {"*": "echo {{omen}} > $DATA/fan"}}]}`

// TestMarkII_TheTrialsBesideAScanOnlyProve: every ritual that reads a
// machine-spirit whose auspex holds a scan and trials reads it by the scan
// alone, and completion and the spared census by neither; only an
// invocation speaks a trial, to prove what it set.
func TestMarkII_TheTrialsBesideAScanOnlyProve(t *testing.T) {
	e := newEnv(t)
	data := t.TempDir()
	e.addRite("rites/fan.json", strings.ReplaceAll(fanJSON, "$DATA", data))
	e.shows(data, "fan", "quiet\n")
	e.shows(data, "loud-holds", "")
	spoken := func() string {
		b, _ := os.ReadFile(filepath.Join(data, "spoken"))
		_ = os.Remove(filepath.Join(data, "spoken"))
		return strings.Join(strings.Fields(string(b)), " ")
	}
	for _, args := range [][]string{
		{"augury", "fan"}, {"augury", "fan", "aspect"}, {"census"}, {"census", "--binharic"},
		{"inquisition", "fan"}, {"invoke", "fan", "on", "--foresee"},
	} {
		e.mustRun(args...)
		if got := spoken(); got != "scan" {
			t.Errorf("servitor %v spoke %q, want the scan alone", args, got)
		}
	}
	e.vigil()
	if got := spoken(); got != "scan" {
		t.Errorf("the vigil spoke %q, want the scan alone", got)
	}
	for _, args := range [][]string{{"__complete", "invoke", "fan", ""}, {"census", "--spare-auspex"}} {
		e.mustRun(args...)
		if got := spoken(); got != "" {
			t.Errorf("servitor %v spoke %q, want nothing", args, got)
		}
	}
	e.mustRun("invoke", "fan", "on")
	if got := spoken(); got != "scan trial-loud" {
		t.Errorf("the invocation spoke %q, want the scan to read and the trial to prove", got)
	}
}
