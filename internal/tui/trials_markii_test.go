package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMarkII_TheCogitatorReadsByTheScanBesideTrials: the cogitator reads a
// machine-spirit whose auspex holds a scan and trials by its scan alone;
// the trials only prove what an invocation set.
func TestMarkII_TheCogitatorReadsByTheScanBesideTrials(t *testing.T) {
	h := newHarness(t)
	h.writeRite("fan", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "fan", "designation": "machine-spirit", "omens": {"on": "loud", "*": "quiet"},
	   "auspex": {"scan": "echo scan >> $DATA/spoken; cat $DATA/fan",
	              "trials": {"loud": "echo trial >> $DATA/spoken; true"}},
	   "enact": {"*": "echo {{omen}} > $DATA/fan"}}]}`)
	h.writeData("fan", "quiet\n")

	a := h.reading("fan")

	if a.Aspect != "off" {
		t.Errorf("the cogitator reads the fan %q, want off", a.Aspect)
	}
	spoken, _ := os.ReadFile(filepath.Join(h.dataDir, "spoken"))
	if words := strings.Fields(string(spoken)); len(words) == 0 || strings.Contains(string(spoken), "trial") {
		t.Errorf("the cogitator spoke %q, want the scan alone", words)
	}
}
