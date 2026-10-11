package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/chronicle"
	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// rebootEdict is an edict without inscriptions; every proclaiming adds a
// line to $DATA/runs. It sorts after the rite "mouse".
const rebootEdict = `{
  "pattern": "Mark I",
  "purpose": "Reboot the machine",
  "edict": true,
  "liturgy": [
    {"incantation": "echo proclaimed >> $DATA/runs; echo the machine stirs"},
    {"vox-cast": "success"}
  ]
}`

// withReboot is a harness whose Librarium also holds the edict reboot,
// chosen in the list.
func withReboot(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t)
	h.writeRite("reboot", rebootEdict)
	h.keys("r", "j")
	h.mustShow("Rites (2)", "Reboot the machine")
	return h
}

// runs is how often the edict reboot was performed.
func (h *harness) runs() int {
	h.t.Helper()
	data, err := os.ReadFile(filepath.Join(h.dataDir, "runs"))
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		h.t.Fatal(err)
	}
	return strings.Count(string(data), "proclaimed\n")
}

// nothingPerformed fails when an invocation was begun.
func (h *harness) nothingPerformed() {
	h.t.Helper()
	if h.m.isRunning() || len(h.inbox) > 0 {
		h.t.Fatalf("an invocation was begun:\n%s", h.screen())
	}
	if n := h.runs(); n != 0 {
		h.t.Fatalf("the edict was performed %d times", n)
	}
}

// TestProclaim_SpaceAsksFirstAndPerformsNothing: space on an edict opens
// its confirmation and performs nothing; neither does space upon the
// confirmation; esc retreats with nothing performed.
func TestProclaim_SpaceAsksFirstAndPerformsNothing(t *testing.T) {
	h := withReboot(t)
	h.keys("space")
	h.mustShow("Proclaim the edict reboot", "every verse of its liturgy", "enter", "proclaim")
	h.mustNotShow("Choose the aspect", "→")
	h.nothingPerformed()
	h.keys("space", "space")
	h.mustShow("Proclaim the edict reboot")
	h.nothingPerformed()
	h.keys("esc")
	h.mustNotShow("Proclaim the edict reboot")
	h.mustShow("Rites (2)")
	h.nothingPerformed()
	h.desktopSilent()
}

// TestProclaim_EnterThenEnterProclaimsOnce: enter asks, a second enter
// proclaims the edict exactly once; the running panel, the toast, the
// words of the invocation and the overview speak of no aspect.
func TestProclaim_EnterThenEnterProclaimsOnce(t *testing.T) {
	h := withReboot(t)
	h.keys("enter")
	h.mustShow("Proclaim the edict reboot")
	h.nothingPerformed()
	h.keys("enter")
	h.mustShow("Proclamation of the edict reboot", "step 0 / 1")
	h.mustNotShow("→")
	h.await()
	if n := h.runs(); n != 1 {
		t.Fatalf("the edict was performed %d times, not once", n)
	}
	h.mustShow("The edict reboot is proclaimed")
	h.mustNotShow("Proclamation of", "→", "dormant →")
	if a := h.reading("reboot"); a.Standing != augury.Proclaimed || a.LastRite == nil {
		t.Fatalf("augury %+v", a)
	}
	h.mustShow("❢ proclaimed", "◉ proclaimed", "✔ triumph")
	h.desktopSilent()

	h.keys("O")
	h.mustShow("Words of the last invocation", "The edict reboot: triumph", "the machine stirs")
	h.mustNotShow("→")
	h.keys("esc")
	if n := h.runs(); n != 1 {
		t.Fatalf("the edict was performed %d times, not once", n)
	}
}

// TestProclaim_PForeseesAndEscRetreats: p upon the confirmation foresees
// the edict; esc returns to the confirmation, and esc again retreats.
func TestProclaim_PForeseesAndEscRetreats(t *testing.T) {
	h := withReboot(t)
	h.keys("space", "p")
	h.mustShow("Foresight of the edict reboot", "verse 1 · incantation", "echo proclaimed",
		"Nothing was performed.")
	h.mustNotShow("Foresight of aspect", "for aspect")
	h.nothingPerformed()
	h.keys("esc")
	h.mustShow("Proclaim the edict reboot")
	h.keys("esc")
	h.mustShow("Rites (2)")
	h.mustNotShow("Proclaim the edict")
	h.nothingPerformed()
}

// TestProclaim_TheInscriptionFormComesFirst: an edict with inscriptions
// asks for them first, prefilled with its plain decree; the value written
// reaches the command once the confirmation is answered with enter.
func TestProclaim_TheInscriptionFormComesFirst(t *testing.T) {
	h := newHarness(t)
	h.writeRite("capture", edictRite)
	h.keys("r", "g", "space")
	h.mustShow("Inscriptions for the edict capture", "region", "full")
	h.mustNotShow("Choose the aspect", "for aspect")
	h.keys("esc") // the form retreats to the overview
	h.mustShow("Rites (2)")
	h.mustNotShow("Inscriptions for the edict")

	h.keys("enter")
	h.mustShow("Inscriptions for the edict capture")
	h.keys("ctrl+p")
	h.mustShow("Foresight of the edict capture", "echo full")
	h.keys("esc")
	h.mustShow("Inscriptions for the edict capture")
	h.keys("ctrl+u")
	h.typeText("window")
	h.keys("enter")
	h.mustShow("Proclaim the edict capture", "region=window")
	if _, err := os.Stat(filepath.Join(h.dataDir, "captured")); err == nil {
		t.Fatal("the edict was performed before it was confirmed")
	}
	h.keys("esc") // back to the form, its value kept
	h.mustShow("Inscriptions for the edict capture", "window")
	h.keys("enter", "enter")
	h.await()
	if got := h.readData("captured"); got != "window\n" {
		t.Fatalf("the inscription did not reach the command: %q", got)
	}
	h.mustShow("The edict capture is proclaimed")
	if a := h.reading("capture"); a.Inscriptions["region"] != "window" {
		t.Fatalf("augury %+v", a)
	}
}

// TestProclaim_AFallNamesTheEdictWithoutAnAspect: the modal of a fallen
// edict and of one the pre-flight forbids name the edict alone.
func TestProclaim_AFallNamesTheEdictWithoutAnAspect(t *testing.T) {
	h := newHarness(t)
	h.writeRite("reboot", strings.Replace(rebootEdict, `{"vox-cast": "success"}`,
		`{"incantation": "echo dying words; exit 3", "reversion": "echo the deed undone"}`, 1))
	h.keys("r", "j", "enter", "enter")
	h.await()
	h.mustShow("The edict reboot has fallen", "It fell at verse 2 · incantation", "dying words",
		"Every deed is undone")
	h.mustNotShow("→", "The rite reboot")
	h.keys("esc")
	h.mustShow("◉ proclaimed", "✖ reverted")

	h.writeRite("reboot", strings.Replace(rebootEdict, `"edict": true,`,
		`"edict": true, "inscriptions": {"why": {"mandatory": true}},`, 1))
	h.keys("r", "enter", "enter", "enter")
	h.await()
	h.mustShow("The pre-flight forbids the proclaiming", "edict reboot", "nothing was touched")
	h.mustNotShow("→", " into ")
}

// TestProclaim_TheDetailOfAnEdict: the detail panel names the edict in
// place of its aspects, lists its liturgy without omen marks and shows no
// auspex; its row shows its standing beside the edict's glyph.
func TestProclaim_TheDetailOfAnEdict(t *testing.T) {
	h := withReboot(t)
	h.mustShow("❢ dormant", "NATURE", "❢ edict", "STANDING", "○ dormant", "LAST RITE", "never invoked",
		"INSCRIPTIONS", "none declared", "LITURGY", " 1 » echo proclaimed", " 2 ✉ success", "RECORDED IN",
		"rites/reboot.json")
	h.mustNotShow("ASPECTS", "AUSPEX", "· 1 »", "· 2 ✉", "«", "→")
}

// TestProclaim_HintsTellThatAnEdictAsksFirst: the footer offers to proclaim
// an edict, never to cycle it.
func TestProclaim_HintsTellThatAnEdictAsksFirst(t *testing.T) {
	h := withReboot(t)
	h.mustShow("enter proclaim", "space proclaim")
	h.mustNotShow("cycle aspect")
	h.keys("k")
	h.mustShow("enter invoke", "space cycle aspect")
}

var proclaimedReboot = chronicle.Entry{
	At: time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC), Rite: "reboot", Edict: true,
	Inscriptions: map[string]string{"why": "updates"}, Verdict: invocation.Triumph,
}

// TestProclaim_TheChronicleMarksAnEdict: an edict's entry bears the edict's
// mark in place of "former → aspect", is found by the filter "edict" but
// not by "dormant", and is read whole without an aspect.
func TestProclaim_TheChronicleMarksAnEdict(t *testing.T) {
	h := seeded(t)
	seedChronicle(t, h.m.s.Orders().Chronicle, fallenTheme, falteredMouse, proclaimedReboot)
	h.writeRite("reboot", rebootEdict)
	h.keys("r", "j", "h")
	h.mustShow("Chronicle (1)", "/ reboot")
	h.inOrder(when(proclaimedReboot) + " reboot ❢ edict ✔ triumph")
	h.mustNotShow("dormant →", "→ ")

	h.keys("/", "esc", "/")
	h.typeText("edict")
	h.keys("enter")
	h.mustShow("Chronicle (1)")
	h.keys("/", "esc", "/")
	h.typeText("dormant")
	h.keys("enter")
	h.mustShow("Chronicle (1)") // the mouse, dormant → on; never the edict
	h.mustNotShow("reboot")

	h.keys("/", "esc", "enter")
	h.mustShow("The edict reboot: triumph", "why=updates", "The edict was proclaimed in triumph")
	h.mustNotShow("The rite reboot")
	if whole := ansi.Strip(h.m.chronicled(proclaimedReboot)); strings.ContainsAny(whole, "→«»") {
		t.Fatalf("the edict's entry speaks of aspects:\n%s", whole)
	}
}
