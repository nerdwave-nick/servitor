package tui

import (
	"context"
	"os"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/dominion"
	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// TestInvoke_TheCogitatorAwaitsDominionAndSaysSo: while another invocation
// holds dominion over the rite, the running panel tells the awaiting and
// nothing is performed; once it is relinquished, the rite is performed.
func TestInvoke_TheCogitatorAwaitsDominionAndSaysSo(t *testing.T) {
	h := newHarness(t)
	held, err := dominion.Seize(context.Background(), []string{dominion.Rite(os.LookupEnv, "mouse")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Relinquish()
	h.keys("r", "space")
	told, ok := h.next().(voxMsg)
	if !ok || told.Tidings != invocation.Dominion {
		t.Fatalf("the cogitator first heard %+v", told)
	}
	h.mustShow("Invocation of mouse", string([]rune(told.Body)[:30]))
	if got := h.readData("x.kdl"); got != "input {}\n" {
		t.Fatalf("the vessel was touched while dominion was awaited: %q", got)
	}
	held.Relinquish()
	h.await()
	h.mustShow("The rite mouse is performed")
	h.desktopSilent()
}
