package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// everyForm is a state rite of pattern Mark II bearing every designation
// of subject, every form of auspex — scan, trials, both, none and
// privileged — exalted subjects and observances of either designation,
// with every guard.
const everyForm = `{
  "pattern": "Mark II",
  "purpose": "Bear every form",
  "aspects": ["on", "off", "idle"],
  "inscriptions": {"mood": {"purpose": "the mood", "decrees": {"*": "grim"}}},
  "tongue": "bash",
  "subjects": [
    {"name": "niri", "designation": "sanctum", "vessel": "$DATA/niri.kdl", "ward": "every", "glyph": "//",
     "consecrate": true, "omens": {"*": "mode {{aspect}}"}, "introit": ["veil"]},
    {"name": "conf", "designation": "transcription", "vessel": "$DATA/conf", "omens": {"on": "lit", "*": null},
     "recessional": ["hooks"]},
    {"name": "theme", "designation": "tether", "tether": "$DATA/theme", "exalted": true,
     "omens": {"on": "$DATA/themes/on", "*": null}},
    {"name": "smt", "designation": "machine-spirit", "omens": {"on": "off", "*": "on"},
     "auspex": {"scan": "cat $DATA/smt", "trials": {"off": "grep -qx off $DATA/smt"}, "patience": "1s"},
     "enact": {"*": "echo {{omen}} > $DATA/smt"}, "tongue": "sh", "patience": "5s"},
    {"name": "gov", "designation": "machine-spirit", "omens": {"on": "performance", "off": "powersave", "*": "powersave"},
     "auspex": {"trials": {"performance": "grep -qx performance $DATA/gov"}}, "enact": {"*": "echo {{omen}} > $DATA/gov"}},
    {"name": "nft", "designation": "machine-spirit", "exalted": true, "auspex": {"scan": "nft list ruleset", "exalted": true},
     "enact": {"*": "true"}},
    {"name": "blind", "designation": "machine-spirit", "enact": {"*": "true"}}
  ],
  "observances": [
    {"name": "veil", "designation": "incantation", "to": ["on", "idle"], "from": [], "words": "true",
     "reversion": "true", "patience": "2s"},
    {"name": "hooks", "designation": "litany", "scroll": "hooks.sh", "offerings": ["{{aspect}}", "~/x"],
     "exalted": true, "relics": ["hooks.sh"]},
    {"name": "idle", "designation": "incantation", "words": "true"}
  ]
}`

// ownerRites are the master's own rites, of pattern Mark I, read
// converting.
var ownerRites = map[string]string{
	"dnd": `{"pattern": "Mark I", "purpose": "turn on/off dnd", "aspects": ["on", "off"],
	  "auspex": {"rite": "makoctl mode | grep \"dnd\" &>/dev/null && echo \"on\" || echo \"off\"", "patience": "200ms"},
	  "tongue": "bash", "liturgy": [
	    {"incantation": {"on": "notify-send \"Initiating Vow of Silence\" -t 1000 && sleep 1", "*": ":"}, "patience": "2s"},
	    {"incantation": {"on": "makoctl mode -a dnd", "*": "makoctl mode -r dnd"}},
	    {"vox-cast": "success"}]}`,
	"gaming": `{"pattern": "Mark I", "purpose": "enable or disable gaming optimization", "aspects": ["on", "off"],
	  "liturgy": [
	    {"incantation": {"on": "sysctl -wq kernel.split_lock_mitigate=0", "*": "sysctl -wq kernel.split_lock_mitigate=1"}},
	    {"vox-cast": "progress"},
	    {"incantation": {"on": "scx_cosmos -p 0 -c 0 &> /dev/null", "*": "killall scx_cosmos &> /dev/null || true"}},
	    {"vox-cast": "progress"},
	    {"incantation": {"on": "echo on | tee /sys/devices/system/cpu/smt/control &> /dev/null",
	      "*": "echo off | tee /sys/devices/system/cpu/smt/control &> /dev/null"}},
	    {"vox-cast": "progress"}]}`,
	"mouse-autohide-toggle": `{"pattern": "Mark I", "purpose": "Hide the cursor of the machine automatically - or don't",
	  "aspects": ["on", "off"],
	  "inscriptions": {"reason": {"purpose": "why the rite was invoked", "decrees": {"off": "some stupid reason probably"}},
	    "time_ms": {"purpose": "the actual autohide timeout", "decrees": {"on": "400"}}},
	  "auspex": {"rite": "grep -Fq \"hide-after-inactive-ms\" ~/.config/niri/util.kdl && echo \"on\" || echo \"off\"",
	    "patience": "300ms"},
	  "tongue": "bash", "liturgy": [
	    {"sanctum": "~/.config/niri/util.kdl", "ward": "mouse-autohide-toggle", "glyph": "//", "consecrate": true,
	     "scripture": {"on": ["cursor {", "  hide-after-inactive-ms {{inscription.time_ms}}", "}"], "*": ["cursor {", "}"]}},
	    {"vox-cast": "success"}]}`,
}

// TestMarkII_NoFormFellsTheCogitator: every form of state rite, the
// master's own rites once converted, an edict and a heretical rite of
// pattern Mark II are drawn at every size and foreseen without the
// cogitator falling; nothing is performed. The harness renders after
// every key, so a panic fails the trial.
func TestMarkII_NoFormFellsTheCogitator(t *testing.T) {
	h := newHarness(t)
	t.Setenv("HOME", t.TempDir())
	h.writeRite("every", everyForm)
	h.writeRite("reboot", rebootEdictII)
	h.writeRite("broken", `{"pattern": "Mark II", "aspects": ["on"], "subjects": [{"name": "x", "designation": "ghost"}]}`)
	for name, body := range ownerRites {
		h.writeRite(name, body)
	}
	h.writeData("smt", "off\n")
	h.renew(librarium.Runes{})
	h.keys("r", "g")
	h.mustShow("Rites (7)", "every", "gaming", "dnd")
	for range len(h.m.rows) {
		for _, size := range []tea.WindowSizeMsg{{Width: 82, Height: 24}, {Width: 220, Height: 70}, {Width: 120, Height: 36}} {
			h.send(size)
		}
		h.keys("enter", "p", "esc", "esc", "esc", "j")
	}
	h.mustShow("Rites (7)")
	h.mustShow("every") // the rite of every form is drawn
	h.keys("g", "j", "j")
	h.mustShow("Bear every form", "⚙ ✠ nft · unread", "↣ ✠ theme", "≡ ✠ hooks · hooks.sh {{aspect}} ~/x",
		"» veil · to on, idle · from no aspect · true", "» idle · true", "VOX-CAST     none")
	h.desktopSilent()
}
