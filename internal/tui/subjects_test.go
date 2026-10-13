package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// warRite is a state rite of pattern Mark II: a machine-spirit read by its
// scan, one read by its trials, a tether, a machine-spirit that cannot be
// read, and one whose scan prints no omen it declares; an introit spoken
// into on alone and a recessional spoken out of off alone. $DATA is the
// place of its vessels.
const warRite = `{
  "pattern": "Mark II",
  "purpose": "Ready the machine for war",
  "aspects": ["on", "off"],
  "vox-cast": ["progress", "success"],
  "subjects": [
    {"name": "smt", "designation": "machine-spirit", "omens": {"on": "off", "*": "on"},
     "auspex": {"scan": "cat $DATA/smt"}, "enact": {"*": "echo {{omen}} > $DATA/smt"}, "introit": ["veil"]},
    {"name": "sched", "designation": "machine-spirit", "omens": {"on": "cosmos", "*": "eevdf"},
     "auspex": {"trials": {"cosmos": "grep -qx cosmos $DATA/sched"}}, "enact": {"*": "echo {{omen}} > $DATA/sched"},
     "recessional": ["hooks"]},
    {"name": "theme", "designation": "tether", "tether": "$DATA/theme", "omens": {"*": "$DATA/themes/{{aspect}}"}},
    {"name": "boost", "designation": "machine-spirit", "enact": {"*": "echo {{omen}} > $DATA/boost"}},
    {"name": "loud", "designation": "machine-spirit", "omens": {"on": "roaring", "*": "quiet"},
     "auspex": {"scan": "echo whispers"}, "enact": {"*": "echo summoning {{omen}}"}}
  ],
  "observances": [
    {"name": "veil", "designation": "incantation", "to": ["on"], "words": "echo veiled >> $DATA/said",
     "reversion": "echo unveiled >> $DATA/said"},
    {"name": "hooks", "designation": "incantation", "from": ["off"], "words": "echo hooked >> $DATA/said"}
  ]
}`

// stand sets the subjects of warRite to show smt, sched and the tether's
// aspect.
func (h *harness) stand(smt, sched, theme string) {
	h.t.Helper()
	h.writeData("smt", smt+"\n")
	h.writeData("sched", sched+"\n")
	h.writeData("themes/on", "on\n")
	h.writeData("themes/off", "off\n")
	link := filepath.Join(h.dataDir, "theme")
	_ = os.Remove(link)
	if err := os.Symlink(filepath.Join(h.dataDir, "themes", theme), link); err != nil {
		h.t.Fatal(err)
	}
}

// TestSubjects_TheDetailShowsEverySubjectItsOmenAndMark: the detail panel
// of a performed state rite shows its subjects in place of a liturgy, one
// line each with the mark of its omen, its designation's glyph, its name
// and the omen read; a subject without an omen tells why. Its observances
// follow with their guards, then its vox-casts; no auspex of the rite is
// shown, for it has none.
func TestSubjects_TheDetailShowsEverySubjectItsOmenAndMark(t *testing.T) {
	h := newHarness(t)
	h.writeRite("war", warRite)
	h.stand("off", "cosmos", "on")
	h.send(tea.WindowSizeMsg{Width: 160, Height: 40})
	h.keys("r", "j")
	h.mustShow("Ready the machine for war", "◆ on", "performed", "SUBJECTS",
		`✔ ⚙ smt "off"`, `✔ ⚙ sched "cosmos"`, "✔ ↣ theme "+shortPath(filepath.Join(h.dataDir, "theme")),
		"· ⚙ boost · unread", `· ⚙ loud "whispers" · mute`,
		"OBSERVANCES", "» veil · to on · echo veiled", "» hooks · from off · echo hooked",
		"VOX-CAST", "progress · success")
	h.mustNotShow("LITURGY", "AUSPEX")
}

// TestSubjects_AnEmptyAgreementMarksTheSubjectsThatEmptiedIt: when the
// omens of a state rite agree on no aspect, the rite is corrupted, and the
// subjects that fit no aspect the others agree on are marked as breaking
// the agreement; the others stand marked as agreeing.
func TestSubjects_AnEmptyAgreementMarksTheSubjectsThatEmptiedIt(t *testing.T) {
	h := newHarness(t)
	h.writeRite("war", warRite)
	h.stand("on", "cosmos", "on")
	h.send(tea.WindowSizeMsg{Width: 160, Height: 40})
	h.keys("r", "j")
	h.mustShow("corrupted", `✖ ⚙ smt "on" (off) · breaks the agreement`, `✔ ⚙ sched "cosmos"`, "✔ ↣ theme")
	h.mustNotShow(`sched "cosmos" (`, "theme · breaks")

	// the tether turns to off: sched alone fits on, and breaks it
	h.stand("on", "cosmos", "off")
	h.keys("r")
	h.mustShow(`✔ ⚙ smt "on"`, "✔ ↣ theme", `✖ ⚙ sched "cosmos" (on) · breaks the agreement`)
	h.mustNotShow("smt \"on\" (off)")

	// two subjects alone, each against the other: both break it
	h.writeRite("war", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "smt", "designation": "machine-spirit", "omens": {"on": "off", "*": "on"},
	   "auspex": {"scan": "cat $DATA/smt"}, "enact": {"*": "true"}},
	  {"name": "sched", "designation": "machine-spirit", "omens": {"on": "cosmos", "*": "eevdf"},
	   "auspex": {"trials": {"cosmos": "grep -qx cosmos $DATA/sched"}}, "enact": {"*": "true"}}]}`)
	h.keys("r")
	h.mustShow("corrupted", `✖ ⚙ smt "on" (off) · breaks the agreement`,
		`✖ ⚙ sched "cosmos" (on) · breaks the agreement`)
}

// TestSubjects_MuteUnreadAndForgoneSubjectsTellWhy: a subject that yields
// no omen is marked · with the reason: mute when it was read and fits no
// aspect, unread when it has no auspex, and forgone when its auspex was
// left unawakened. A trials auspex none of whose trials succeeds fits the
// omen left without a trial.
func TestSubjects_MuteUnreadAndForgoneSubjectsTellWhy(t *testing.T) {
	h := newHarness(t)
	h.writeRite("war", warRite)
	h.send(tea.WindowSizeMsg{Width: 160, Height: 40})
	h.keys("r", "j")
	h.mustShow(`· ⚙ smt · mute`, `✔ ⚙ sched "eevdf"`, "· ↣ theme "+shortPath(filepath.Join(h.dataDir, "theme"))+" · mute", "· ⚙ boost · unread",
		`· ⚙ loud "whispers" · mute`)

	h.stand("off", "cosmos", "on")
	spared, err := h.m.s.Augur("war", true)
	if err != nil {
		t.Fatal(err)
	}
	h.m.rows[h.m.cursor].reading = spared
	h.mustShow("· ⚙ smt · forgone", "· ⚙ sched · forgone", "· ⚙ loud · forgone", "· ⚙ boost · unread", "✔ ↣ theme")
}
