package tui

import (
	"testing"

	"github.com/nerdwave-nick/servitor/internal/augury"
)

// rebootEdictII is rebootEdict written in pattern Mark II, its two entries
// counted as two verses and its vox-casts a key of the rite.
const rebootEdictII = `{
  "pattern": "Mark II",
  "purpose": "Reboot the machine",
  "edict": true,
  "vox-cast": ["progress", "success"],
  "liturgy": [
    {"designation": "incantation", "words": "echo proclaimed >> $DATA/runs; echo the machine stirs"},
    {"designation": "incantation", "words": "true"}
  ]
}`

// TestProclaimII_AnEdictOfMarkIIIsProclaimedAsAnyOther: space asks first and
// performs nothing; p foresees every verse; enter then enter proclaims the
// edict of Mark II once, and the cogitator shows it proclaimed without an
// aspect.
func TestProclaimII_AnEdictOfMarkIIIsProclaimedAsAnyOther(t *testing.T) {
	h := newHarness(t)
	h.writeRite("reboot", rebootEdictII)
	h.keys("r", "j")
	h.mustShow("Rites (2)", "Reboot the machine", "edict")

	h.keys("space")
	h.mustShow("Proclaim the edict reboot", "every verse of its liturgy")
	h.nothingPerformed()
	h.keys("p")
	h.mustShow("Foresight of the edict reboot", "verse 1 · incantation", "verse 2 · incantation")
	h.mustNotShow("vox-cast", "verse 3")
	h.nothingPerformed()
	h.keys("esc", "esc")
	h.mustNotShow("Proclaim the edict reboot")
	h.nothingPerformed()

	h.keys("enter", "enter")
	h.mustShow("Proclamation of the edict reboot", "step 0 / 2")
	h.mustNotShow("→")
	h.await()
	if n := h.runs(); n != 1 {
		t.Fatalf("the edict was performed %d times, not once", n)
	}
	h.mustShow("The edict reboot is proclaimed")
	h.mustNotShow("Proclamation of", "→")
	if a := h.reading("reboot"); a.Standing != augury.Proclaimed || a.LastRite == nil {
		t.Fatalf("augury %+v", a)
	}
	h.mustShow("❢ proclaimed", "◉ proclaimed", "✔ triumph")
	h.desktopSilent()
}
