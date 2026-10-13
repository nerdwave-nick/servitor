package tui

import "testing"

// TestWizard_ARiteTurnedEdictMustPassTheInquisition: a rite declared an
// edict leaves its subjects behind and must be given a liturgy; a verse
// speaking {{aspect}}, which an edict bears none of, is heresy that
// forbids the sealing until it is cast out.
func TestWizard_ARiteTurnedEdictMustPassTheInquisition(t *testing.T) {
	h := newHarness(t)
	h.keys("e", "tab", "tab")
	h.mustShow("◆ aspects", "Aspects")
	h.keys("space")
	h.mustShow("◆ edict", "mouse", "Hide the cursor", "2 Liturgy", "3 Seal")
	h.mustNotShow("Aspects", "Subjects")
	h.keys("ctrl+s")
	h.mustShow(`Inscription "reason"`)
	h.keys("ctrl+s")
	h.mustShow(`Inscription "mode"`, "Decree")
	h.mustNotShow("for aspect")
	h.keys("ctrl+s")
	h.mustShow("The liturgy of the edict mouse", "No step is written yet")
	h.keys("tab")
	h.mustShow("A rite without a liturgy is an empty prayer")

	h.keys("a", "1", "ctrl+s")
	h.typeText("echo {{aspect}}")
	h.keys("ctrl+s")
	h.mustShow("1 » echo {{aspect}}")
	h.keys("tab")
	h.mustShow("Heresy detected", "{{aspect}}")
	h.keys("enter")
	h.mustShow("Heresy remains")
	if r := h.loadRite("mouse"); r.Edict {
		t.Fatal("a heretical edict was sealed")
	}

	h.keys("esc", "d", "a", "1", "ctrl+s")
	h.typeText("true")
	h.keys("ctrl+s", "tab")
	h.mustShow("The edict is pure")
	h.mustNotShow(`"aspects"`, `"subjects"`)
	h.keys("enter")
	h.mustShow("The edict mouse is amended")
	if r := h.loadRite("mouse"); !r.Edict || r.Aspects != nil || r.Subjects != nil || len(r.Liturgy) != 1 {
		t.Fatalf("the rite was not turned edict: %+v", r)
	}
}

// TestWizard_AnEdictTurnedRiteAsksItsAspects: an edict declared a rite of
// aspects again must name them; its plain decree serves every aspect under
// "*", and its liturgy, which a rite of aspects bears not, gives way to
// subjects, of which it must declare one.
func TestWizard_AnEdictTurnedRiteAsksItsAspects(t *testing.T) {
	h := newHarness(t)
	h.writeRite("capture", edictII)
	h.keys("r", "g", "e", "tab", "tab", "space")
	h.mustShow("Amendment of the rite capture", "◆ aspects", "Aspects", "2 Subjects", "3 Observances")
	h.keys("ctrl+s")
	h.mustShow("a rite without aspects can never be invoked")
	h.typeText("on")
	h.keys("ctrl+s")
	h.mustShow(`Inscription "region"`, `Decree for aspect "on"`, "Decree for every other aspect (*)", "chosen")
	h.keys("ctrl+s")
	h.mustShow("The subjects of the rite capture", "No subject is declared yet")
	h.keys("tab")
	h.mustShow("A rite without subjects has nothing to read or set")
	h.keys("esc", "esc", "esc")
	h.mustShow("is abandoned.")
	if r := h.loadRite("capture"); !r.Edict {
		t.Fatalf("the abandoned amendment was sealed: %+v", r)
	}
}

// TestWizard_TheNatureKeepsWhatWasWritten: turning the Rite page to an
// edict and back keeps the aspects and auspex once written.
func TestWizard_TheNatureKeepsWhatWasWritten(t *testing.T) {
	h := newHarness(t)
	h.keys("n")
	h.typeText("theme")
	h.keys("tab", "tab", "tab", "ctrl+u")
	h.typeText("dark, porpl")
	h.keys("up", "space")
	h.mustNotShow("dark, porpl")
	h.keys("space")
	h.mustShow("◆ aspects", "theme", "dark, porpl")
}
