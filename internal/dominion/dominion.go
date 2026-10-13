// Package dominion keeps the dominion of invocations: the hold an
// invocation keeps over its rite and over every vessel and tethered name it
// writes, from before its pre-flight to its verdict, so that no other
// invocation performs upon them meanwhile.
//
// A dominion is an exclusive flock upon an empty place beneath the state
// home of the servitor ($XDG_STATE_HOME/servitor/dominions/), raised on
// demand and never struck. The machine relinquishes a flock when its holder
// dies, however it dies, so no dominion outlives its invocation.
package dominion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// Dir is the directory of the servitor's state directory holding the places
// of dominion.
const Dir = "dominions"

// Rite is the place of the dominion over the rite named rite:
// $XDG_STATE_HOME/servitor/dominions/rites/<rite>.
func Rite(env librarium.Environment, rite string) string {
	return filepath.Join(librarium.StateDir(env), Dir, "rites", rite)
}

// Dominion is held over places until it is relinquished.
type Dominion struct {
	held []*os.File
	once sync.Once
}

// Seize takes dominion over every place, one after another in the order
// given; the caller orders them so that no two invocations seize in
// opposing orders. Each is first tried at once; when another holds it,
// awaiting (when not nil) is told — once for the whole seizing, before its
// first wait — and the seizing waits, without end, until it is
// relinquished. A halted ctx ends the waiting with ctx's error. On any
// error nothing stays held.
func Seize(ctx context.Context, places []string, awaiting func()) (*Dominion, error) {
	d := &Dominion{}
	told := false
	await := func() {
		if !told && awaiting != nil {
			awaiting()
		}
		told = true
	}
	for _, place := range places {
		f, err := seize(ctx, place, await)
		if err != nil {
			d.Relinquish()
			return nil, err
		}
		d.held = append(d.held, f)
	}
	return d, nil
}

// Relinquish lets go of every place, in the reverse order of their
// seizing. Relinquishing again does nothing.
func (d *Dominion) Relinquish() {
	d.once.Do(func() {
		for i := len(d.held) - 1; i >= 0; i-- {
			_ = d.held[i].Close() // closing the last descriptor releases the flock
		}
	})
}

// seize takes dominion over one place, raising it when it does not stand.
func seize(ctx context.Context, place string, await func()) (*os.File, error) {
	f, err := open(place)
	if err != nil {
		return nil, err
	}
	err = flock(f, syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		return f, nil
	}
	if !errors.Is(err, syscall.EWOULDBLOCK) {
		_ = f.Close()
		return nil, err
	}
	await()
	won := make(chan error, 1)
	go func() { won <- flock(f, syscall.LOCK_EX) }()
	select {
	case err := <-won:
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		return f, nil
	case <-ctx.Done():
		// The flock cannot be called back: once it is granted, it is
		// relinquished at once.
		go func() { <-won; _ = f.Close() }()
		return nil, ctx.Err()
	}
}

// open opens a place of dominion, raising it when it does not stand.
func open(place string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(place), 0o755); err != nil {
		return nil, err
	}
	// os.OpenFile opens upon close-on-exec: no process the invocation
	// summons inherits its dominion.
	return os.OpenFile(place, os.O_RDONLY|os.O_CREATE, 0o644)
}

// flock is syscall.Flock, begun anew when a signal interrupts it.
func flock(f *os.File, how int) error {
	for {
		err := syscall.Flock(int(f.Fd()), how)
		if !errors.Is(err, syscall.EINTR) {
			return err
		}
	}
}

// Try takes dominion over every place, as Seize does, yet never waits:
// when another holds any of them, nothing stays held and ok is false. On
// any error nothing stays held.
func Try(places []string) (d *Dominion, ok bool, err error) {
	d = &Dominion{}
	for _, place := range places {
		f, err := open(place)
		if err == nil {
			err = flock(f, syscall.LOCK_EX|syscall.LOCK_NB)
			if err != nil {
				_ = f.Close()
			}
		}
		if err != nil {
			d.Relinquish()
			if errors.Is(err, syscall.EWOULDBLOCK) {
				return nil, false, nil
			}
			return nil, false, err
		}
		d.held = append(d.held, f)
	}
	return d, true, nil
}
