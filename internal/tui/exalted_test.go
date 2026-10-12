package tui

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// TestInvoke_TheCogitatorRefusesExaltedVerses: the cogitator performs no
// exalted verse with its invoker's authority. Its invocation of a rite
// never sanctioned is refused before anything is touched and the refusal
// is shown, while foresight still reveals what the rite would do.
func TestInvoke_TheCogitatorRefusesExaltedVerses(t *testing.T) {
	h := newHarness(t)
	h.writeRite("mouse", strings.Replace(markRite, `"reversion": "rm -f $DATA/mode"}`,
		`"reversion": "rm -f $DATA/mode", "exalted": true}`, 1))
	h.keys("r", "enter", "p")
	h.mustShow("Foresight of aspect on", "verse 3 · incantation", "Nothing was performed.")
	h.keys("esc", "1", "enter", "enter")
	h.await()
	h.mustShow("The pre-flight forbids the invocation", "nothing was touched", "exalted:", "sanction")
	if got := h.readData("x.kdl"); got != "input {}\n" {
		t.Fatalf("the vessel was touched: %q", got)
	}
	if _, err := os.Stat(filepath.Join(h.dataDir, "mode")); err == nil {
		t.Fatal("the exalted verse was spoken")
	}
	h.desktopSilent()
}

// TestInvoke_TheCogitatorRaisesNoExaltedServitorYet: the cogitator lends no
// intercessor yet, so even a sanctioned rite is refused before any proof is
// asked, and the refusal says to invoke it at a terminal. Nothing is
// touched; the exalted verse is never spoken by the cogitator itself.
func TestInvoke_TheCogitatorRaisesNoExaltedServitorYet(t *testing.T) {
	h := newHarness(t)
	h.writeRite("mouse", strings.Replace(markRite, `"reversion": "rm -f $DATA/mode"}`,
		`"reversion": "rm -f $DATA/mode", "exalted": true}`, 1))
	store := sanction.Store{Root: t.TempDir(), UID: os.Getuid()}
	former := rituals.SanctionRoot
	t.Cleanup(func() { rituals.SanctionRoot = former })
	rituals.SanctionRoot = store.Root
	lib := librarium.Load(h.dir)
	u, err := (&rituals.Servitor{Librarium: lib, Sanctions: &store}).Unit(lib.Rites["mouse"])
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(sanction.New(u, time.Now()))
	if err == nil {
		err = os.MkdirAll(filepath.Dir(store.Path("mouse")), 0o750)
	}
	if err == nil {
		err = os.WriteFile(store.Path("mouse"), data, 0o640)
	}
	if err != nil {
		t.Fatal(err)
	}
	h.m = newModel(h.dir, librarium.Runes{})
	h.m.send = func(msg tea.Msg) { h.inbox <- msg }
	h.send(tea.WindowSizeMsg{Width: 160, Height: 36})
	h.keys("r", "enter", "1", "enter", "enter")
	h.await()
	h.mustShow("The pre-flight forbids the invocation", "lent no intercessor", "servitor invoke mouse on")
	if got := h.readData("x.kdl"); got != "input {}\n" {
		t.Fatalf("the vessel was touched: %q", got)
	}
	if _, err := os.Stat(filepath.Join(h.dataDir, "mode")); err == nil {
		t.Fatal("the exalted verse was spoken")
	}
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

// exaltedKinds holds an exalted step of every kind that may be exalted.
const exaltedKinds = `{
  "pattern": "Mark I",
  "aspects": ["on", "off"],
  "liturgy": [
    {"sanctum": "/etc/x.conf", "scripture": {"on": "a", "*": ""}, "exalted": true},
    {"transcription": "/etc/whole.conf", "scripture": "whole {{aspect}}", "seal": "0600", "exalted": true},
    {"tether": "/etc/current", "anchor": {"on": null, "*": "/usr/share/x"}, "exalted": true},
    {"incantation": "sysctl -w x=1", "reversion": "sysctl -w x=0", "exalted": true, "relics": []},
    {"litany": "scrolls/tune", "offerings": ["{{aspect}}"], "exalted": true, "relics": ["scrolls/tune.conf", "a"]},
    {"incantation": "true"}
  ]
}`

// TestDraft_ExaltedVersesRoundTrip: the wizard keeps what is exalted, and
// the relics of exalted speech, when it amends a rite or an edict.
func TestDraft_ExaltedVersesRoundTrip(t *testing.T) {
	for name, scripture := range map[string]string{"rite": exaltedKinds, "edict": `{"pattern": "Mark I", "edict": true,
		"liturgy": [{"incantation": "systemctl reboot", "exalted": true}, {"litany": "s", "exalted": true, "relics": ["s.list"]}]}`} {
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
