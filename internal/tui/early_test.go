package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// slowRite is a state rite of pattern Mark II whose machine-spirits smt and
// gov are read by scans that wait at a gate until $DATA/release exists,
// each scan's shell telling its pid in $DATA/pids.
const slowRite = `{
  "pattern": "Mark II",
  "aspects": ["on", "off"],
  "subjects": [
    {"name": "smt", "designation": "machine-spirit", "omens": {"on": "off", "*": "on"},
     "auspex": {"scan": "echo $$ >> $DATA/pids; while [ ! -e $DATA/release ]; do sleep 0.01; done; cat $DATA/smt",
                "patience": "5s"},
     "enact": {"*": "echo {{omen}} > $DATA/smt"}},
    {"name": "gov", "designation": "machine-spirit", "omens": {"on": "performance", "*": "powersave"},
     "auspex": {"scan": "echo $$ >> $DATA/pids; while [ ! -e $DATA/release ]; do sleep 0.01; done; cat $DATA/gov",
                "patience": "5s"},
     "enact": {"*": "echo {{omen}} > $DATA/gov"}}
  ]
}`

// noon is when the omens of slowRite were recorded.
var noon = time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)

// slow writes slowRite, whose subjects show smt and gov once the gate is
// open, and its data-slate recording the aspect off and smt's omen "on",
// read at noon in this boot.
func (h *harness) slow(smt, gov string) {
	h.t.Helper()
	h.writeRite("slow", slowRite)
	h.writeData("smt", smt+"\n")
	h.writeData("gov", gov+"\n")
	if err := augury.WriteSlate(os.LookupEnv, "slow", augury.Slate{Aspect: "off",
		LastRite: &augury.LastRite{Verdict: invocation.Triumph, At: noon}, Boot: augury.Boot(),
		Omens: map[string]augury.Recorded{"smt": {Omen: "on", At: noon}}}); err != nil {
		h.t.Fatal(err)
	}
	h.t.Cleanup(h.open) // no scan outlives its test
}

// open opens the gate of slowRite's scans.
func (h *harness) open() { _ = os.WriteFile(filepath.Join(h.dataDir, "release"), nil, 0o644) }

// scans waits until n scans of slowRite have awoken, and returns their pids
// in the order they awoke.
func (h *harness) scans(n int) []int {
	h.t.Helper()
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(10 * time.Millisecond) {
		b, _ := os.ReadFile(filepath.Join(h.dataDir, "pids"))
		var pids []int
		for _, f := range strings.Fields(string(b)) {
			pid, _ := strconv.Atoi(f)
			pids = append(pids, pid)
		}
		if len(pids) >= n {
			return pids
		}
		if time.Now().After(deadline) {
			h.t.Fatalf("%d scans awoke, want %d", len(pids), n)
		}
	}
}

// slain tells whether the process pid has ended (a zombie has ended too).
func slain(pid int) bool {
	if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
		return true
	}
	stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return true
	}
	_, after, _ := strings.Cut(string(stat), ") ")
	return strings.HasPrefix(after, "Z")
}

// TestEarly_TheFirstFrameShowsTheStaleOmenThenTheLiveOne: the cogitator
// draws its rites at once, while their auspexes still labour: a subject
// from the omen its data-slate recorded, dimmed stale with its age, one
// without a record as awaited, a rite of Mark I its auspex awaited. Once
// the auspexes settle, it draws the live omens.
func TestEarly_TheFirstFrameShowsTheStaleOmenThenTheLiveOne(t *testing.T) {
	h := newHarness(t)
	h.hasty = true
	h.slow("off", "performance")
	h.m.now = func() time.Time { return noon.Add(2*time.Hour + 7*time.Minute) }
	h.send(tea.WindowSizeMsg{Width: 160, Height: 40})

	h.keys("r")
	h.mustShow("mouse", "AUSPEX", "· awaited")
	h.keys("j")
	h.mustShow("◆ off", "~off", `✔ ⚙ smt "on" stale · read 2h ago`, "· ⚙ gov · awaited")
	h.scans(2) // the frame was drawn while both scans laboured

	h.open()
	h.settle()
	h.mustShow("◆ on", `✔ ⚙ smt "off"`, `✔ ⚙ gov "performance"`)
	h.mustNotShow("stale", "awaited", "~off")
	h.keys("k")
	h.mustShow("AUSPEX", "· silent")
}

// TestEarly_AnExaltedAuspexStaysStale: a subject read by an exalted
// auspex, which is spoken at invocation alone, is drawn from its recorded
// omen and stays stale once every other auspex has settled.
func TestEarly_AnExaltedAuspexStaysStale(t *testing.T) {
	h := newHarness(t)
	h.writeRite("firewall", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "nft", "designation": "machine-spirit", "exalted": true, "omens": {"on": "on", "*": "off"},
	   "auspex": {"scan": "nft list ruleset", "exalted": true}, "enact": {"*": "nft -f /etc/{{omen}}.nft"}}]}`)
	if err := augury.WriteSlate(os.LookupEnv, "firewall", augury.Slate{Aspect: "on",
		LastRite: &augury.LastRite{Verdict: invocation.Triumph, At: noon}, Boot: augury.Boot(),
		Omens: map[string]augury.Recorded{"nft": {Omen: "on", At: noon, Exalted: true}}}); err != nil {
		t.Fatal(err)
	}
	h.m.now = func() time.Time { return noon.Add(3 * 24 * time.Hour) }
	h.send(tea.WindowSizeMsg{Width: 160, Height: 40})
	h.keys("r", "g") // the harness awaits every auspex that may be heard
	h.mustShow("firewall", "~on")
	h.mustShow(`✔ ⚙ ✠ nft "on" stale · read 3d ago`)
}

// TestEarly_EveryAgeIsToldInItsGreatestUnit: the age of a stale omen is
// told in whole seconds, minutes, hours or days.
func TestEarly_EveryAgeIsToldInItsGreatestUnit(t *testing.T) {
	for d, want := range map[time.Duration]string{
		-time.Minute: "0s", 0: "0s", 59 * time.Second: "59s", time.Minute: "1m", 59*time.Minute + 59*time.Second: "59m",
		2*time.Hour + 7*time.Minute: "2h", 47 * time.Hour: "47h", 48 * time.Hour: "2d", 400 * time.Hour: "16d",
	} {
		if got := age(d); got != want {
			t.Errorf("age(%v) = %q, want %q", d, got, want)
		}
	}
}

// TestEarly_AReloadSlaysTheVigilBeforeItAndHeedsItNot: reading the
// Librarium anew slays the auspexes still labouring for the reading before,
// and what a vigil of an older reading tells never overwrites a newer one.
func TestEarly_AReloadSlaysTheVigilBeforeItAndHeedsItNot(t *testing.T) {
	h := newHarness(t)
	h.hasty = true
	h.slow("off", "performance")
	h.send(tea.WindowSizeMsg{Width: 160, Height: 40})
	h.keys("r", "j")
	first := h.scans(2)
	h.keys("r")
	gen := h.m.vigil.gen
	h.scans(4)
	for _, pid := range first[:2] { // their patience is far longer
		for deadline := time.Now().Add(2 * time.Second); !slain(pid); time.Sleep(10 * time.Millisecond) {
			if time.Now().After(deadline) {
				t.Fatalf("the scan %d of the reading before outlived it", pid)
			}
		}
	}

	h.open()
	var older []tea.Msg // what the vigil of this reading tells of slow, held back
	for ended := false; !ended; {
		select {
		case msg := <-h.vigil:
			switch msg := msg.(type) {
			case settledMsg:
				if msg.gen == gen && msg.reading.Augury.Rite == "slow" {
					older = append(older, msg)
				}
			case attendedMsg:
				ended = msg.gen == gen
			}
		case <-time.After(10 * time.Second):
			t.Fatal("the vigil never ended")
		}
	}
	if len(older) != 1 || older[0].(settledMsg).reading.Aspect != "on" {
		t.Fatalf("the vigil told %+v", older)
	}
	h.writeData("smt", "on\n")
	h.writeData("gov", "powersave\n")
	h.keys("r")
	h.settle()
	h.mustShow("◆ off", `✔ ⚙ smt "on"`, `✔ ⚙ gov "powersave"`)
	h.send(older...)
	h.mustShow("◆ off", `✔ ⚙ smt "on"`, `✔ ⚙ gov "powersave"`)
}

// TestRun_LeavingSlaysTheLabouringAuspexes: the cogitator retreats while an
// auspex still labours; it slays it, and returns only once it is slain.
func TestRun_LeavingSlaysTheLabouringAuspexes(t *testing.T) {
	h := newHarness(t)
	h.slow("off", "performance")
	keys, ended := h.awaken()
	defer func() { _ = keys.Close() }()
	pids := h.scans(2)
	if _, err := keys.Write([]byte("q")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ended:
	case <-time.After(10 * time.Second):
		t.Fatal("the cogitator did not retreat")
	}
	for _, pid := range pids {
		if !slain(pid) {
			t.Errorf("the scan %d outlived the cogitator", pid)
		}
	}
}
