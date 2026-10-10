package tui

import (
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/codex"
)

// entries are the passages of the codex screen, chapter by chapter.
func entries() []codex.Passage {
	var out []codex.Passage
	for _, ch := range codex.Index() {
		out = append(out, ch.Passages...)
	}
	return out
}

// TestCodex_QuestionMarkExpounds: ? opens the codex beneath the catalogue of
// keys; enter recites the chosen passage, esc withdraws step by step.
func TestCodex_QuestionMarkExpounds(t *testing.T) {
	h := newHarness(t)
	h.keys("?")
	h.mustShow("The Codex", "Catalogue of Sacred Keys", "excommunicate", "Passages of the Codex", "Rituals", "invoke")

	at := -1
	for i, p := range entries() {
		if p.Topic == "tether" {
			at = i
			break
		}
	}
	if at < 0 {
		t.Fatal("the codex holds no tether")
	}
	for range at {
		h.keys("j")
	}
	h.keys("enter")
	h.mustShow("tether · as written in the codex", "anchor")
	h.keys("space", "space")
	h.mustShow("Exempla:", "current-theme")
	h.keys("esc")
	h.mustShow("Passages of the Codex", "tether")
	h.keys("esc")
	if strings.Contains(h.screen(), "Passages of the Codex") {
		t.Fatalf("esc did not withdraw from the codex:\n%s", h.screen())
	}
	h.mustShow("Rites (1)")
}

// TestCodex_TheChosenPassageStaysInSight: the index scrolls with the choice,
// so the last passage is in sight once chosen, and the first again after g.
func TestCodex_TheChosenPassageStaysInSight(t *testing.T) {
	h := newHarness(t)
	all := entries()
	first, last := all[0], all[len(all)-1]
	h.keys("?", "G")
	h.mustShow(last.Topic + " ")
	if strings.Contains(h.screen(), "Catalogue of Sacred Keys") {
		t.Fatalf("the index did not scroll to its end:\n%s", h.screen())
	}
	h.keys("enter")
	h.mustShow(last.Topic + " · as written in the codex")
	h.keys("esc", "g")
	h.mustShow("Catalogue of Sacred Keys", first.Topic+" ")
	h.keys("k", "q")
	h.mustShow("Rites (1)")
}
