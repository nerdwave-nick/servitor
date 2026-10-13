package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// TestInvoke_TheCogitatorRefusesExaltedSubjects: the cogitator sets no
// exalted subject with its invoker's authority. Its invocation of a rite
// never sanctioned is refused before anything is touched and the refusal
// is shown, while foresight still reveals what the rite would do.
func TestInvoke_TheCogitatorRefusesExaltedSubjects(t *testing.T) {
	h := newHarness(t)
	h.writeRite("mouse", strings.Replace(markRite, `{"name": "chant", "designation": "machine-spirit",`,
		`{"name": "chant", "designation": "machine-spirit", "exalted": true,`, 1))
	h.keys("r", "enter", "p")
	h.mustShow("Foresight of aspect on", "subject chant · machine-spirit", "Nothing was performed.")
	h.keys("esc", "1", "enter", "enter")
	h.await()
	h.mustShow("The pre-flight forbids the invocation", "nothing was touched", "exalted:", "sanction")
	if got := h.readData("x.kdl"); got != "input {}\n" {
		t.Fatalf("the vessel was touched: %q", got)
	}
	if _, err := os.Stat(filepath.Join(h.dataDir, "mode")); err == nil {
		t.Fatal("the exalted subject was set")
	}
	h.desktopSilent()
}

// TestProclaim_TheCogitatorRefusesAnExaltedEdict: an edict holding an
// exalted verse is never proclaimed by the cogitator; foresight still is.
func TestProclaim_TheCogitatorRefusesAnExaltedEdict(t *testing.T) {
	h := newHarness(t)
	h.writeRite("reboot", strings.Replace(rebootEdict, `the machine stirs"}`, `the machine stirs", "exalted": true}`, 1))
	h.keys("r", "j", "enter", "p")
	h.mustShow("Foresight of the edict reboot", "echo proclaimed", "Nothing was performed.")
	h.keys("esc", "enter")
	h.await()
	h.mustShow("The pre-flight forbids the invocation", "proclaiming of the edict reboot", "nothing was touched",
		"exalted:", "sanction")
	if n := h.runs(); n != 0 {
		t.Fatalf("the exalted edict was proclaimed %d times", n)
	}
}

// exaltedKinds holds an exalted subject of every designation and an
// exalted observance of both, beside common ones.
const exaltedKinds = `{
  "pattern": "Mark II",
  "aspects": ["on", "off"],
  "subjects": [
    {"name": "x", "designation": "sanctum", "vessel": "/etc/x.conf", "omens": {"on": "a", "*": ""}, "exalted": true,
     "introit": ["sysctl"]},
    {"name": "whole", "designation": "transcription", "vessel": "/etc/whole.conf", "omens": {"*": "whole {{aspect}}"},
     "seal": "0600", "exalted": true},
    {"name": "current", "designation": "tether", "tether": "/etc/current", "omens": {"on": null, "*": "/usr/share/x"},
     "exalted": true, "recessional": ["tune", "plain"]},
    {"name": "knob", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"},
     "auspex": {"scan": "cat /etc/knob", "exalted": true}, "enact": {"*": "echo {{omen}} > /etc/knob"},
     "exalted": true, "relics": ["scrolls/tune.conf"]}
  ],
  "observances": [
    {"name": "sysctl", "designation": "incantation", "words": "sysctl -w x=1", "reversion": "sysctl -w x=0",
     "exalted": true, "relics": []},
    {"name": "tune", "designation": "litany", "scroll": "scrolls/tune", "offerings": ["{{aspect}}"], "exalted": true,
     "relics": ["scrolls/tune.conf", "a"]},
    {"name": "plain", "designation": "incantation", "words": "true"}
  ]
}`

// TestDraft_ExaltedItemsRoundTrip: the wizard keeps what is exalted — and
// the relics of exalted speech and enact words, and a privileged auspex —
// when it amends a rite of aspects or an edict.
func TestDraft_ExaltedItemsRoundTrip(t *testing.T) {
	for name, scripture := range map[string]string{"rite": exaltedKinds, "edict": `{"pattern": "Mark II", "edict": true,
		"liturgy": [{"designation": "incantation", "words": "systemctl reboot", "exalted": true},
		{"designation": "litany", "scroll": "s", "exalted": true, "relics": ["s.list"]}]}`} {
		t.Run(name, func(t *testing.T) {
			want, found := librarium.Parse(name, "/lib/rites/"+name+".json", []byte(scripture))
			if found.Heretical() || !want.Exalted() {
				t.Fatalf("the fixture is heretical: %v", found)
			}
			got := riteDraftFrom(want).toRite("/lib/rites/" + name + ".json")
			if !got.Equal(want) {
				data, _ := librarium.Marshal(got)
				t.Fatalf("the rite changed on its way through the wizard:\n%s", data)
			}
		})
	}
}
