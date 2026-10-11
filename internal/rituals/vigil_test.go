package rituals

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/augury"
)

// keepVigil keeps the vigil over s and returns every heralding in the
// order it was told.
func keepVigil(ctx context.Context, s *Servitor, forgoAuspex bool) []Heralding {
	var heard []Heralding
	s.Vigil(ctx, forgoAuspex, func(h Heralding) { heard = append(heard, h) })
	return heard
}

// TestVigil_TellsEveryRiteThenEveryAuspexAsItSettles: the vigil tells every
// rite by name as it stands without its auspex, an awaited auspex marked
// so, then every awaited rite once more as its auspex settles, in the order
// they settle; what it told last of each rite is the census.
func TestVigil_TellsEveryRiteThenEveryAuspexAsItSettles(t *testing.T) {
	fx := newFixture(t)
	fx.vessel("util.kdl", "")
	fx.rite("mouse", mouse)
	fx.rite("capture", capture)
	fx.rite("broken", `{"pattern": "Mark I", "aspects": []}`)
	fx.rite("fallen", `{"pattern": "Mark I", "aspects": ["on", "off"], "auspex": "exit 1", "liturgy": []}`)
	fx.rite("nonsense", `{"pattern": "Mark I", "aspects": ["on", "off"], "auspex": "echo porpl", "liturgy": []}`)
	fx.rite("sluggard", `{"pattern": "Mark I", "aspects": ["on", "off"],
	  "auspex": {"rite": "sleep 5; echo on", "patience": "0.3s"}, "liturgy": []}`)
	for i, nap := range []string{"1.2", "0.9", "0.6"} {
		fx.rite(fmt.Sprintf("slow%d", i),
			`{"pattern": "Mark I", "aspects": ["on", "off"], "auspex": "sleep `+nap+`; echo on", "liturgy": []}`)
	}
	s := fx.servitor()
	for _, rite := range []string{"fallen", "broken", "slow1"} {
		fx.write(fx.state, filepath.Join("servitor", augury.SlatesDir, rite+".json"), "{")
	}

	heard := keepVigil(context.Background(), s, false)

	names := s.Librarium.Names()
	var first, settled []string
	laments := map[string]int{}
	last := map[string]Entry{}
	for _, h := range heard {
		if h.Lament != nil {
			if h.Entry != nil || !strings.Contains(h.Lament.Error(), "data-slate") {
				t.Errorf("lament %+v", h)
			}
			laments[h.Rite]++
			continue
		}
		rite := h.Entry.Rite
		if _, told := last[rite]; !told {
			first = append(first, rite)
		} else {
			settled = append(settled, rite)
		}
		if len(settled) > 0 && len(first) < len(names) {
			t.Fatalf("%s settled before every rite was told once", rite)
		}
		last[rite] = *h.Entry
	}
	if !slices.Equal(first, names) {
		t.Errorf("first told %v, want every rite by name %v", first, names)
	}
	want := []string{"fallen", "nonsense", "sluggard", "slow2", "slow1", "slow0"}
	if len(settled) != len(want) || !slices.Equal(settled[2:], want[2:]) || !slices.Equal(slices.Sorted(slices.Values(settled[:2])), want[:2]) {
		t.Errorf("settled %v, want %v as they settle", settled, want)
	}
	for _, h := range heard[:len(names)+len(laments)] {
		if h.Entry == nil {
			continue
		}
		awaited := slices.Contains(want, h.Entry.Rite)
		if got := h.Entry.Auspex; (got == augury.Awaited) != awaited {
			t.Errorf("%s first told with auspex %q", h.Entry.Rite, got)
		}
	}
	if fmt.Sprint(laments) != "map[broken:1 fallen:1 slow1:1]" {
		t.Errorf("laments by rite %v", laments)
	}
	census, _ := s.Census(false)
	var told []Entry
	for _, name := range names {
		told = append(told, last[name])
	}
	if got, want := marshal(t, told), marshal(t, census); got != want {
		t.Errorf("last told\n%s\nwant the census\n%s", got, want)
	}
}

// TestVigil_ForgoneTellsEveryRiteOnce: forgoing the auspex, the vigil tells
// every rite once, by name, as the census forgoing it reads them.
func TestVigil_ForgoneTellsEveryRiteOnce(t *testing.T) {
	fx := newFixture(t)
	mark := filepath.Join(fx.state, "awakened")
	fx.rite("mouse", mouse)
	fx.rite("capture", capture)
	fx.rite("sloth", `{"pattern": "Mark I", "aspects": ["on", "off"], "auspex": "touch `+mark+`; sleep 5", "liturgy": []}`)
	s := fx.servitor()

	start := time.Now()
	heard := keepVigil(context.Background(), s, true)

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("the forgone vigil took %v", elapsed)
	}
	var told []Entry
	for _, h := range heard {
		told = append(told, *h.Entry)
	}
	census, _ := s.Census(true)
	if got, want := marshal(t, told), marshal(t, census); got != want {
		t.Errorf("told\n%s\nwant\n%s", got, want)
	}
	if exists(mark) {
		t.Error("a forgone auspex was awakened")
	}
}

// TestVigil_AnEndedVigilSlaysItsAuspexes: once its context ends, the vigil
// tells nothing more, slays the process group of every auspex still awake
// and returns.
func TestVigil_AnEndedVigilSlaysItsAuspexes(t *testing.T) {
	fx := newFixture(t)
	pids := filepath.Join(fx.state, "pids")
	for i := range 3 {
		fx.rite(fmt.Sprintf("sloth%d", i), `{"pattern": "Mark I", "aspects": ["on", "off"],
		  "auspex": {"rite": "sleep 30 & echo $$ $! >> `+pids+`; wait", "patience": "60s"}, "liturgy": []}`)
	}
	s := fx.servitor()
	ctx, end := context.WithCancel(context.Background())
	done := make(chan []Heralding, 1)
	go func() { done <- keepVigil(ctx, s, false) }()
	var awake []int
	for deadline := time.Now().Add(10 * time.Second); len(awake) < 6; time.Sleep(10 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("the auspexes never awoke: %v", awake)
		}
		b, _ := os.ReadFile(pids)
		awake = awake[:0]
		for _, f := range strings.Fields(string(b)) {
			pid, _ := strconv.Atoi(f)
			awake = append(awake, pid)
		}
	}

	end()
	var heard []Heralding
	select {
	case heard = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the ended vigil did not return")
	}
	if len(heard) != 3 {
		t.Errorf("an ended vigil told %d heraldings, want only the 3 of the first telling", len(heard))
	}
	for _, pid := range awake {
		for deadline := time.Now().Add(2 * time.Second); !ended(pid); time.Sleep(10 * time.Millisecond) {
			if time.Now().After(deadline) {
				t.Fatalf("process %d of an auspex outlived the vigil", pid)
			}
		}
	}
}

// ended tells whether the process pid has ended (a zombie has ended too).
func ended(pid int) bool {
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
