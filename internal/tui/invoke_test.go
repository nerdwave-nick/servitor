package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestInvoke_PickForeseeInscribeAndWatchTheRiteRun(t *testing.T) {
	h := newHarness(t)
	h.keys("enter")
	h.mustShow("Invoke mouse", "Choose the aspect to invoke", "on", "off")
	h.keys("p")
	h.mustShow("Foresight of aspect on", "verse 2 · sanctum", "+ // +++ begin of sanctum cursor -- aspect|on +++",
		"+ a", "spoken in the tongue bash", "echo chanting on", "should the rite fall, its reversion",
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
	h.mustShow("Invocation of mouse → on", "step 0 / 2")

	h.next() // the progress vox-cast before any real step
	h.mustShow("The machine spirit stirs; the liturgy of mouse begins.", "step 0 / 2")
	h.next() // the sanctum begins
	h.mustShow("step 1 / 2", "verse 2 · sanctum")
	h.next() // the incantation begins
	h.mustShow("step 2 / 2", "verse 3 · incantation echo chanting on")
	h.await()

	h.mustNotShow("Invocation of mouse")
	h.mustShow("The rite mouse is performed: dormant → on", "reason=gaming remnant", "mode=hide")
	if a := h.reading("mouse"); a.Aspect != "on" || a.Inscriptions["reason"] != "gaming remnant" {
		t.Fatalf("augury %+v", a)
	}
	h.desktopSilent()

	h.keys("O")
	h.mustShow("Words of the last invocation", "verse 3 · incantation", "chanting on")
	h.keys("esc")
	h.mustNotShow("Words of the last invocation")
}

func TestInvoke_AFallShowsTheHeresyTheReversionsAndTheLastWords(t *testing.T) {
	h := newHarness(t)
	h.writeRite("mouse", strings.Replace(markRite, `{"vox-cast": "success"}`,
		`{"incantation": "echo dying words; exit 3", "reversion": "echo the deed undone"}`, 1))
	h.keys("r", "space")
	h.await()
	h.mustShow("The rite mouse has fallen", "It fell at verse 4 · incantation echo dying words; exit 3",
		"death-mark 3", "its last words:", "dying words", "verse 4 · incantation", "is undone",
		"the deed undone", "verse 2 · sanctum", "Every deed is undone")
	if got := h.readData("x.kdl"); got != "input {}\n" {
		t.Fatalf("the vessel was not restored: %q", got)
	}
	h.keys("esc")
	h.mustNotShow("has fallen")
	h.mustShow("dormant")
	h.keys("O")
	h.mustShow("Words of the last invocation", "chanting on", "dying words", "the deed undone")
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

func TestInvoke_CtrlCHaltsTheRiteRatherThanTheCogitator(t *testing.T) {
	h := newHarness(t)
	h.writeRite("mouse", strings.Replace(markRite, `{"vox-cast": "success"}`, `{"incantation": "sleep 30"}`, 1))
	h.keys("r", "space")
	h.next() // the progress vox-cast
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
	if _, err := os.Stat(filepath.Join(h.dataDir, "mode")); !os.IsNotExist(err) {
		t.Fatal("the reversion of the incantation was not spoken")
	}
}
