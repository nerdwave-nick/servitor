package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// writeHall writes rel beside the scriptures of the rites, in their hall,
// where relics and scrolls are reckoned from.
func (h *harness) writeHall(rel, content string, mode os.FileMode) {
	h.t.Helper()
	p := write(h.t, filepath.Join(h.dir, "rites", rel), content)
	if err := os.Chmod(p, mode); err != nil {
		h.t.Fatal(err)
	}
}

// throughPages seals page after page of the step being written until the
// liturgy is shown again; a page that will not be left fails the test.
func (h *harness) throughPages() {
	h.t.Helper()
	for range 8 {
		h.keys("ctrl+s")
		if strings.Contains(h.screen(), "The liturgy of the") {
			return
		}
	}
	h.t.Fatalf("the pages of the step would not be left:\n%s", h.screen())
}

// focusOn tabs through the wizard's page until the field key is focused.
func (h *harness) focusOn(key string) {
	h.t.Helper()
	w, ok := h.m.screen.(*wizard)
	if !ok || w.form == nil {
		h.t.Fatalf("no page of the wizard is shown:\n%s", h.screen())
	}
	for range w.form.fields {
		if w.form.fields[w.form.focus].key == key {
			return
		}
		h.keys("tab")
	}
	h.t.Fatalf("the page bears no field %q:\n%s", key, h.screen())
}

// TestWizard_ExaltationIsOfferedToEveryVerseAndSubject: every verse an
// edict may bear and every subject a rite of aspects may hold is offered
// exaltation on its essence page; relics are offered once it is exalted,
// to speech and a machine-spirit's enact words alone.
func TestWizard_ExaltationIsOfferedToEveryVerseAndSubject(t *testing.T) {
	for _, nature := range []string{natureAspects, natureEdict} {
		t.Run(nature, func(t *testing.T) {
			h := newHarness(t)
			h.keys("n")
			h.typeText("x")
			noun, kinds := "Exalted subject", subjectDesignations
			if nature == natureEdict {
				h.keys("tab", "tab", "space")
				noun, kinds = "Exalted verse", []librarium.Kind{librarium.KindIncantation, librarium.KindLitany}
			}
			h.keys("ctrl+s")
			for i, k := range kinds {
				h.keys("a", string(rune('1'+i)))
				h.mustShow(kindGlyph(k)+" "+kindKey(k), noun)
				h.mustNotShow("Relics, one per line")
				h.focusOn("exalted")
				h.keys("space")
				if bearsRelics(k) {
					h.mustShow("Relics, one per line")
				} else {
					h.mustNotShow("Relics, one per line")
				}
				h.keys("esc")
			}
		})
	}
}

// exaltedEdict is an edict of exalted speech, its litany bearing a relic.
const exaltedEdict = `{"pattern": "Mark II", "edict": true, "vox-cast": ["success"], "liturgy": [
  {"designation": "incantation", "words": "systemctl reboot", "exalted": true},
  {"designation": "litany", "scroll": "scrolls/tune", "exalted": true, "relics": ["tuning.list"]}
]}`

// TestWizard_AmendAndReplicateKeepExaltation: an edict amended through
// every page of every entry, and replicated, keeps what is exalted and the
// relics of its exalted speech. (The exaltation of subjects and
// observances is kept by TestWizard_AmendEveryFormOfMarkII.)
func TestWizard_AmendAndReplicateKeepExaltation(t *testing.T) {
	for name, scripture := range map[string]string{"reboot": exaltedEdict} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			h.writeData("themes/.keep", "")
			h.writeHall("scrolls/tune", "#!/bin/sh\nsysctl -p ./tuning.list\n", 0o755)
			h.writeHall("scrolls/tune.conf", "smt off\n", 0o644)
			h.writeHall("tuning.list", "x = 1\n", 0o644)
			h.writeRite(name, scripture)
			h.keys("r")
			before := *h.m.s.Librarium.Rites[name]
			if !before.Exalted() {
				t.Fatalf("the fixture is not exalted: %v", h.m.s.Librarium.Findings)
			}
			h.keys("j", "e") // it sorts after mouse
			for range before.Inscriptions {
				h.keys("ctrl+s")
			}
			h.keys("ctrl+s")
			h.mustShow("The liturgy of the")
			for i, s := range before.Liturgy {
				h.keys("enter")
				h.mustShow("Exalted verse", "[✔] yes")
				if s.Kind() == librarium.KindLitany {
					h.mustShow("Relics, one per line", "tuning.list")
				}
				h.throughPages()
				if i < len(before.Liturgy)-1 {
					h.keys("j")
				}
			}
			h.keys("tab")
			h.mustShow("is pure", `"exalted": true`)
			h.keys("enter")
			h.mustShow("is amended")
			if got := h.loadRite(name); !got.Equal(&before) {
				data, _ := librarium.Marshal(got)
				t.Fatalf("the amended scripture differs:\n%s", data)
			}

			h.keys("c", "ctrl+s")
			for range before.Inscriptions {
				h.keys("ctrl+s")
			}
			h.keys("tab")
			h.mustShow("is pure")
			h.keys("enter")
			h.mustShow(name + "-copy is consecrated")
			want := before
			want.Name = name + "-copy"
			if got := h.loadRite(name + "-copy"); !got.Equal(&want) {
				data, _ := librarium.Marshal(got)
				t.Fatalf("the replica differs:\n%s", data)
			}
		})
	}
}

// TestWizard_AVerseUnexaltedLaysDownItsRelics: an exalted litany whose
// exaltation is struck sheds its relics with it, for relics are borne by
// exalted speech alone; struck and restored before sealing, they return.
func TestWizard_AVerseUnexaltedLaysDownItsRelics(t *testing.T) {
	h := newHarness(t)
	h.writeHall("scrolls/tune", "#!/bin/sh\necho tuned\n", 0o755)
	h.writeHall("tuning.list", "x = 1\n", 0o644)
	h.writeRite("reboot", exaltedEdict)
	h.keys("r", "j", "e", "ctrl+s", "j", "enter") // reboot sorts after mouse
	h.mustShow("≡ litany", "[✔] yes", "Relics, one per line", "tuning.list")
	h.keys("space")
	h.mustShow("[ ] no")
	h.mustNotShow("Relics, one per line", "tuning.list")
	h.keys("space")
	h.mustShow("[✔] yes", "Relics, one per line", "tuning.list")
	h.keys("space")
	h.throughPages()
	h.keys("tab")
	h.mustShow("The edict is pure")
	h.keys("enter")
	h.mustShow("The edict reboot is amended")
	lit := h.loadRite("reboot").Liturgy[1].(*librarium.Litany)
	if lit.Exalted || lit.Relics != nil {
		t.Fatalf("the litany stayed exalted: %+v", lit.Utterance)
	}
}

// TestWizard_TheSealJudgesExaltation: the heresies of exaltation forbid the
// sealing of an edict: relics outside the hall of the rite, or nowhere to
// be found beside it.
func TestWizard_TheSealJudgesExaltation(t *testing.T) {
	h := newHarness(t)
	h.keys("n")
	h.typeText("tune")
	h.keys("tab", "tab", "space", "ctrl+s")
	h.mustShow("The liturgy of the edict tune")

	h.keys("a", "1", "space", "tab")
	h.typeText("../beyond.list")
	h.keys("ctrl+s")
	h.typeText("true")
	h.keys("ctrl+s")
	h.mustShow("1 » true")

	h.keys("tab")
	h.mustShow("Heresy detected", "lies outside the hall of the rite")
	h.keys("enter")
	h.mustShow("Heresy remains. The edict cannot be sealed.")

	// The Inquisition seeks the relics of scripture that is otherwise pure.
	h.keys("esc", "d", "a", "1", "space", "tab")
	h.typeText("missing.list")
	h.keys("ctrl+s")
	h.typeText("true")
	h.keys("ctrl+s", "tab")
	h.mustShow("Heresy detected", "missing.list", "nowhere to be found beside the rite")
	h.keys("enter")
	h.mustShow("Heresy remains. The edict cannot be sealed.")
	if _, err := os.Stat(filepath.Join(h.dir, "rites", "tune.json")); !os.IsNotExist(err) {
		t.Fatal("a heretical rite was sealed")
	}
}
