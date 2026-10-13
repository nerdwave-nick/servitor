package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestInvoke_PickForeseeInscribeAndWatchTheRiteRun: the aspect is chosen,
// foreseen subject by subject without touching a vessel, inscribed, and
// the invocation watched as it sets each changing subject; its words are
// kept for O.
func TestInvoke_PickForeseeInscribeAndWatchTheRiteRun(t *testing.T) {
	h := newHarness(t)
	h.keys("enter")
	h.mustShow("Invoke mouse", "Choose the aspect to invoke", "on", "off")
	h.keys("p")
	h.mustShow("Foresight of aspect on", "subject cursor · sanctum: changing",
		"+ // +++ begin of sanctum cursor -- aspect|on +++", "+ a",
		`subject chant · machine-spirit: (mute) → "on", changing`, "spoken in the tongue bash", "echo chanting on",
		"Nothing was performed.")
	if got := h.readData("x.kdl"); got != "input {}\n" {
		t.Fatalf("the foresight touched the vessel: %q", got)
	}
	h.keys("esc")
	h.mustShow("Choose the aspect to invoke")
	h.keys("1")
	h.mustShow("Inscriptions for aspect on", "mode *", "hide", "reason")
	h.typeText("gaming remnant") // the first inscription, reason
	h.keys("ctrl+p")
	h.mustShow("Foresight of aspect on")
	h.keys("esc")
	h.mustShow("Inscriptions for aspect on")
	h.keys("enter", "enter") // past mode, its decree kept, to perform
	h.mustShow("Invocation of mouse → on", "the subjects await their setting")

	h.next() // the plan is drawn
	h.mustShow("subject 0 / 2")
	h.next() // the sanctum is set
	h.mustShow("subject 1 / 2", "▸ subject cursor · sanctum")
	h.next() // its progress, which names it
	h.mustShow("cursor")
	h.next() // the machine-spirit is set
	h.mustShow("subject 2 / 2", `▸ subject chant · machine-spirit: (mute) → "on"`)
	h.await()

	h.mustNotShow("Invocation of mouse")
	h.mustShow("The rite mouse is performed: dormant → on", "reason=gaming remnant", "mode=hide")
	if a := h.reading("mouse"); a.Aspect != "on" || a.Inscriptions["reason"] != "gaming remnant" {
		t.Fatalf("augury %+v", a)
	}
	h.desktopSilent()

	h.keys("O")
	h.mustShow("Words of the last invocation", "subject chant", "chanting on")
	h.keys("esc")
	h.mustNotShow("Words of the last invocation")
}

// TestInvoke_AFallShowsTheHeresyTheReversionsAndTheLastWords: a subject
// whose enact words fall is named with its death-mark and last words, and
// everything set before it is restored and told.
func TestInvoke_AFallShowsTheHeresyTheReversionsAndTheLastWords(t *testing.T) {
	h := newHarness(t)
	h.writeRite("mouse", strings.Replace(markRite, `"enact": {"*": "echo chanting {{aspect}}; echo {{omen}} > $DATA/mode"}`,
		`"enact": {"on": "echo dying words; exit 3", "*": "echo the deed undone; echo {{omen}} > $DATA/mode"}`, 1))
	h.writeData("mode", "off\n")
	h.keys("r", "space")
	h.await()
	h.mustShow("The rite mouse has fallen", "It fell at subject chant · machine-spirit",
		"death-mark 3", "its last words:", "dying words", "subject chant is undone",
		"subject cursor is undone", "Every deed is undone")
	if got := h.readData("x.kdl"); got != "input {}\n" {
		t.Fatalf("the vessel was not restored: %q", got)
	}
	if got := h.readData("mode"); got != "off\n" {
		t.Fatalf("the machine-spirit was not restored: %q", got)
	}
	h.keys("esc")
	h.mustNotShow("has fallen")
	h.keys("O")
	h.mustShow("Words of the last invocation", "dying words", "the deed undone")
	h.desktopSilent()
}

func TestInvoke_TheLastWordsAwaitAnInvocation(t *testing.T) {
	h := newHarness(t)
	h.keys("O")
	h.mustShow("No rite has yet been invoked")
}

func TestInvoke_ThePreflightRefusalIsShown(t *testing.T) {
	h := newHarness(t)
	h.keys("enter", "1", "tab", "ctrl+u", "enter") // mode emptied
	h.await()
	h.mustShow("The pre-flight forbids the invocation", "nothing was touched", "mode")
	if got := h.readData("x.kdl"); got != "input {}\n" {
		t.Fatalf("the vessel was touched: %q", got)
	}
}

// TestInvoke_CtrlCHaltsTheRiteRatherThanTheCogitator: ctrl+c while a rite
// runs halts the rite, which is reverted, and leaves the cogitator be.
func TestInvoke_CtrlCHaltsTheRiteRatherThanTheCogitator(t *testing.T) {
	h := newHarness(t)
	h.writeRite("mouse", strings.Replace(markRite, `"enact": {"*": "echo chanting {{aspect}}; echo {{omen}} > $DATA/mode"}`,
		`"enact": {"on": "echo {{omen}} > $DATA/mode; sleep 30", "*": "echo {{omen}} > $DATA/mode"}`, 1))
	h.writeData("mode", "off\n")
	h.keys("r", "space")
	h.next() // the plan is drawn
	_, cmd := h.m.Update(keyMsg("ctrl+c"))
	if cmd != nil {
		if msg, ok := runQuick(cmd); ok {
			if _, quit := msg.(tea.QuitMsg); quit {
				t.Fatal("ctrl+c abandoned the cogitator while a rite was running")
			}
		}
	}
	h.mustShow("Halting the rite")
	h.await()
	h.mustShow("The rite mouse has fallen", "halted at its master's command", "Every deed is undone")
	h.mustShow("further step") // a long heresy is wrapped, not cut short
	if got := h.readData("mode"); got != "off\n" {
		t.Fatalf("the machine-spirit was not restored: %q", got)
	}
}
