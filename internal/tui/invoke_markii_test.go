package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestSubjects_ThePreviewForeseesEverySubjectsPlan: p upon the invoke
// screen of a state rite tells, per subject, the omen read and the omen
// commanded and whether it changes, the words that would set it, and the
// observances that would be spoken; nothing is performed.
func TestSubjects_ThePreviewForeseesEverySubjectsPlan(t *testing.T) {
	h := newHarness(t)
	h.writeRite("war", warRite)
	h.stand("on", "cosmos", "off")
	h.send(tea.WindowSizeMsg{Width: 160, Height: 70})
	h.keys("r", "j", "enter")
	h.mustShow("Invoke war", "Choose the aspect to invoke")
	h.keys("p")
	h.mustShow("Foresight of aspect on",
		`subject smt · machine-spirit: "on" → "off", changing`, "echo off > ",
		`subject sched · machine-spirit: "cosmos" → "cosmos", unchanged`,
		"subject theme · tether: changing", "the tether is bound to",
		`subject boost · machine-spirit: (no auspex) → "on", changing`,
		`subject loud · machine-spirit: (mute) → "roaring", changing`,
		"observance veil · incantation", "echo veiled",
		"Nothing was performed.")
	h.mustNotShow("observance hooks") // the former aspect is not off
	if got := h.readData("smt"); got != "on\n" {
		t.Fatalf("the foresight set a subject: %q", got)
	}
	if _, err := os.Stat(filepath.Join(h.dataDir, "said")); err == nil {
		t.Fatal("the foresight spoke an observance")
	}
}

// TestSubjects_TheRunningPanelCountsSubjectsAndTheFallNamesTheOmens: while
// a state rite is invoked, the running panel counts "subject x / n" of the
// changing subjects alone, names the subject being set and the omen it is
// set to, and tells the observances spoken. A fall names the subject, the
// omen commanded and the omen found, its last words, and every reversion.
func TestSubjects_TheRunningPanelCountsSubjectsAndTheFallNamesTheOmens(t *testing.T) {
	h := newHarness(t)
	h.writeRite("war", warRite)
	h.stand("on", "cosmos", "off")
	h.send(tea.WindowSizeMsg{Width: 160, Height: 70})
	h.keys("r", "j", "enter", "enter")
	h.mustShow("Invocation of war → on", "the subjects await their setting")

	h.next() // the plan is drawn
	h.mustShow("subject 0 / 4")
	h.next() // the introit
	h.mustShow("observances spoken: veil")
	h.next() // smt
	h.mustShow("subject 1 / 4", `▸ subject smt · machine-spirit: "on" → "off"`)
	h.next() // its progress
	h.next() // the tether
	h.mustShow("subject 2 / 4", "▸ subject theme · tether")
	h.mustNotShow("subject sched") // unchanged: never set
	h.await()

	h.mustShow("The rite war has fallen", "It fell at subject loud · machine-spirit",
		`the subject "loud" shows the omen "whispers"; "roaring" was commanded`,
		`the omen commanded: "roaring" · the omen read before: "whispers" (mute)`,
		"its last words:", "summoning roaring",
		"REVERSIONS", "subject loud could not be undone", "subject boost could not be undone",
		"subject theme is undone", "subject smt is undone", "observance veil is undone",
		"The reversion faltered")
	if got := h.readData("smt"); got != "on\n" {
		t.Fatalf("smt was not restored: %q", got)
	}
	if got := h.readData("said"); got != "veiled\nunveiled\n" {
		t.Fatalf("the introit was spoken and undone as %q", got)
	}
	h.desktopSilent()
}
