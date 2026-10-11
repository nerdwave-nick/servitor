package dominion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// patience is how long a test waits for what must come to pass.
const patience = 10 * time.Second

func seized(t *testing.T, places ...string) *Dominion {
	t.Helper()
	d, err := Seize(context.Background(), places, func() { t.Errorf("free dominion over %v was awaited", places) })
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// seizing seizes places in a goroutine of its own; told hears every
// awaiting, done the outcome.
func seizing(ctx context.Context, places ...string) (told *atomic.Int32, awaited chan struct{}, done chan error, d **Dominion) {
	told, awaited, done, d = new(atomic.Int32), make(chan struct{}, 8), make(chan error, 1), new(*Dominion)
	go func() {
		held, err := Seize(ctx, places, func() { told.Add(1); awaited <- struct{}{} })
		*d = held
		done <- err
	}()
	return told, awaited, done, d
}

func within[T any](t *testing.T, c <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-c:
		return v
	case <-time.After(patience):
		t.Fatalf("%s never came to pass", what)
	}
	var zero T
	return zero
}

func TestRite_LiesAmongTheDominionsOfTheStateHome(t *testing.T) {
	env := func(k string) (string, bool) {
		if k == "XDG_STATE_HOME" {
			return "/state", true
		}
		return "", false
	}
	if got := Rite(env, "theme"); got != "/state/servitor/dominions/rites/theme" {
		t.Fatalf("the dominion over theme lies at %s", got)
	}
}

func TestSeize_FreeDominionIsHeldAtOnceAndRaisedOnDemand(t *testing.T) {
	place := filepath.Join(t.TempDir(), "dominions", "rites", "theme")
	d := seized(t, place)
	info, err := os.Stat(place)
	if err != nil || info.Size() != 0 {
		t.Fatalf("the place of dominion was not raised empty: %v %v", info, err)
	}
	d.Relinquish()
	d.Relinquish() // a second relinquishing does nothing
	if _, err := os.Stat(place); err != nil {
		t.Fatalf("the place of dominion was struck: %v", err)
	}
	seized(t, place).Relinquish()
}

func TestSeize_HeldDominionIsAwaitedAndToldOnce(t *testing.T) {
	dir := t.TempDir()
	rite, vessel := filepath.Join(dir, "rite"), filepath.Join(dir, "vessel")
	first, second := seized(t, rite), seized(t, vessel)
	told, awaited, done, _ := seizing(context.Background(), rite, vessel)
	within(t, awaited, "the awaiting")
	first.Relinquish()
	select {
	case err := <-done:
		t.Fatalf("dominion was seized while another still held it: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	second.Relinquish()
	if err := within(t, done, "the seizing"); err != nil {
		t.Fatal(err)
	}
	if n := told.Load(); n != 1 {
		t.Fatalf("the awaiting was told %d times, not once", n)
	}
}

func TestSeize_HaltedWhileAwaitingHoldsNothing(t *testing.T) {
	place := filepath.Join(t.TempDir(), "rite")
	holder := seized(t, place)
	ctx, halt := context.WithCancel(context.Background())
	_, awaited, done, d := seizing(ctx, place)
	within(t, awaited, "the awaiting")
	halt()
	if err := within(t, done, "the halting"); !errors.Is(err, context.Canceled) {
		t.Fatalf("a halted seizing ended with %v", err)
	}
	if *d != nil {
		t.Fatal("a halted seizing holds dominion")
	}
	holder.Relinquish()
	_, _, done, d = seizing(context.Background(), place)
	if err := within(t, done, "the seizing after the halt"); err != nil {
		t.Fatal(err)
	}
	(*d).Relinquish()
}

func TestSeize_AnUnraisablePlaceIsAnError(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "stone")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(t.TempDir(), "rite")
	d, err := Seize(context.Background(), []string{first, filepath.Join(blocker, "rite")}, nil)
	if err == nil || d != nil {
		t.Fatalf("seized %v beneath a stone: %v", d, err)
	}
	seized(t, first).Relinquish() // what was seized before is relinquished
}
