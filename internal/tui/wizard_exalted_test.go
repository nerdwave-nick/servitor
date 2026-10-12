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

// TestWizard_ConsecrateAnExaltedIncantation: an incantation exalted on its
// essence page, naming two relics, is consecrated by key presses alone and
// sealed exactly; the vox-cast beside it is offered no exaltation. The
// cogitator still refuses to invoke what it consecrated.
func TestWizard_ConsecrateAnExaltedIncantation(t *testing.T) {
	h := newHarness(t)
	h.writeHall("tuning.list", "kernel.split_lock_mitigate = 0\n", 0o644)
	h.writeHall("scrolls/tune.conf", "smt off\n", 0o644)

	h.keys("n")
	h.typeText("tune")
	h.keys("ctrl+s") // aspects on and off, as the wizard proposes
	h.mustShow("The liturgy of the rite tune")

	h.keys("a", "4")
	h.mustShow("» incantation", "Exalted verse", "[ ] no", "Further rites")
	h.mustNotShow("Relics, one per line")
	h.keys("space")
	h.mustShow("Exalted verse", "[✔] yes", "Relics, one per line", "Further rites")
	h.keys("tab")
	h.typeText("tuning.list")
	h.keys("enter")
	h.typeText("scrolls/tune.conf")
	h.keys("ctrl+s")
	h.mustShow(`Command for aspect "on"`)
	h.typeText("sysctl -p ./tuning.list")
	h.keys("ctrl+s")
	h.mustShow("1 » sysctl -p ./tuning.list")

	h.keys("a", "6")
	h.mustShow("✉ vox-cast", "Tidings")
	h.mustNotShow("Exalted verse", "Relics, one per line")
	h.keys("space", "ctrl+s")
	h.mustShow("2 ✉ success")

	h.keys("tab")
	h.mustShow("The rite is pure", `"exalted": true`, `"relics"`)
	h.keys("enter")
	h.mustShow("The rite tune is consecrated")

	want := `{
  "pattern": "Mark I",
  "aspects": ["on", "off"],
  "liturgy": [
    {
      "incantation": "sysctl -p ./tuning.list",
      "exalted": true,
      "relics": ["tuning.list", "scrolls/tune.conf"]
    },
    {"vox-cast": "success"}
  ]
}
`
	data, err := os.ReadFile(filepath.Join(h.dir, "rites", "tune.json"))
	if err != nil || string(data) != want {
		t.Fatalf("the scripture sealed (%v):\n%s\nwant:\n%s", err, data, want)
	}
	inc := h.loadRite("tune").Liturgy[0].(*librarium.Incantation)
	if !inc.Exalted || strings.Join(inc.Relics, ",") != "tuning.list,scrolls/tune.conf" {
		t.Fatalf("the incantation loads as %+v", inc.Utterance)
	}

	h.keys("j", "space") // tune sorts after mouse
	h.await()
	h.mustShow("The pre-flight forbids the invocation", "nothing was touched", "exalted:")
}

// TestWizard_ExaltationIsOfferedToEveryKindButTheVoxCast: every step a rite
// or edict may bear is offered exaltation on its essence page, save the
// vox-cast; relics are offered to exalted incantations and litanies alone.
func TestWizard_ExaltationIsOfferedToEveryKindButTheVoxCast(t *testing.T) {
	for _, nature := range []string{natureAspects, natureEdict} {
		t.Run(nature, func(t *testing.T) {
			h := newHarness(t)
			h.keys("n")
			h.typeText("x")
			if nature == natureEdict {
				h.keys("tab", "tab", "space")
			}
			h.keys("ctrl+s")
			h.mustShow("The liturgy of the")
			kinds := librarium.Kinds
			if nature == natureEdict {
				kinds = []librarium.Kind{librarium.KindIncantation, librarium.KindLitany, librarium.KindVoxCast}
			}
			for i, k := range kinds {
				h.keys("a", string(rune('1'+i)))
				h.mustShow(kindGlyph(k) + " " + k.Key())
				if k == librarium.KindVoxCast {
					h.mustNotShow("Exalted verse", "Relics, one per line")
					h.keys("esc")
					continue
				}
				h.mustShow("Exalted verse")
				h.mustNotShow("Relics, one per line")
				h.focusOn("exalted")
				h.keys("space")
				if k == librarium.KindIncantation || k == librarium.KindLitany {
					h.mustShow("Relics, one per line")
				} else {
					h.mustNotShow("Relics, one per line")
				}
				h.keys("esc")
			}
		})
	}
}

// exaltedRite holds an exalted step of every kind that may be exalted,
// each beside the vessels, scrolls and relics it names.
const exaltedRite = `{
  "pattern": "Mark I",
  "aspects": ["on", "off"],
  "liturgy": [
    {"sanctum": "$DATA/y.conf", "consecrate": true, "scripture": {"on": "a", "*": ""}, "exalted": true},
    {"transcription": "$DATA/whole.conf", "scripture": "whole {{aspect}}", "seal": "0600", "exalted": true},
    {"tether": "$DATA/current", "anchor": {"on": null, "*": "$DATA/themes"}, "exalted": true},
    {"incantation": "sysctl -w x=1", "reversion": "sysctl -w x=0", "exalted": true, "relics": []},
    {"litany": "scrolls/tune", "offerings": ["{{aspect}}"], "exalted": true,
     "relics": ["scrolls/tune.conf", "tuning.list"]},
    {"vox-cast": "success"}
  ]
}`

const exaltedEdict = `{"pattern": "Mark I", "edict": true, "liturgy": [
  {"incantation": "systemctl reboot", "exalted": true},
  {"litany": "scrolls/tune", "exalted": true, "relics": ["tuning.list"]},
  {"vox-cast": "success"}
]}`

// TestWizard_AmendAndReplicateKeepExaltation: a rite and an edict amended
// through every page of every step, and replicated, keep what is exalted
// and the relics of their exalted speech.
func TestWizard_AmendAndReplicateKeepExaltation(t *testing.T) {
	for name, scripture := range map[string]string{"tune": exaltedRite, "reboot": exaltedEdict} {
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
			h.keys("j", "e") // both sort after mouse
			for range before.Inscriptions {
				h.keys("ctrl+s")
			}
			h.keys("ctrl+s")
			h.mustShow("The liturgy of the")
			for i, s := range before.Liturgy {
				h.keys("enter")
				if s.Kind() != librarium.KindVoxCast {
					h.mustShow("Exalted verse", "[✔] yes")
				}
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
// sealing: an exalted vessel that does not begin at the root, {{former}}
// in exalted words, and relics outside the hall of the rite or nowhere to
// be found beside it.
func TestWizard_TheSealJudgesExaltation(t *testing.T) {
	h := newHarness(t)
	h.keys("n")
	h.typeText("tune")
	h.keys("ctrl+s")

	h.keys("a", "2") // a transcription whose vessel lies beside the scripture
	h.typeText("whole.conf")
	h.keys("tab", "space", "ctrl+s")
	h.typeText("whole")
	h.keys("tab", "tab", "tab", "tab", "space", "ctrl+s")
	h.mustShow("1 ¶ ")

	h.keys("a", "4", "space", "tab")
	h.typeText("../beyond.list")
	h.keys("enter")
	h.typeText("missing.list")
	h.keys("ctrl+s")
	h.typeText("echo {{former}}")
	h.keys("ctrl+s")
	h.mustShow("2 » echo {{former}}")

	h.keys("tab")
	h.mustShow("Heresy detected", "does not begin at the root", "lies outside the hall of the rite",
		"SERVITOR_FORMER_ASPECT")
	h.keys("enter")
	h.mustShow("Heresy remains. The rite cannot be sealed.")

	// The Inquisition seeks the relics of scripture that is otherwise pure.
	h.keys("esc", "d", "d", "a", "4", "space", "tab")
	h.typeText("missing.list")
	h.keys("ctrl+s")
	h.typeText("true")
	h.keys("ctrl+s", "tab")
	h.mustShow("Heresy detected", "missing.list", "nowhere to be found beside the rite")
	h.keys("enter")
	h.mustShow("Heresy remains. The rite cannot be sealed.")
	if _, err := os.Stat(filepath.Join(h.dir, "rites", "tune.json")); !os.IsNotExist(err) {
		t.Fatal("a heretical rite was sealed")
	}
}
