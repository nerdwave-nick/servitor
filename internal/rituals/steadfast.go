package rituals

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Steadfast keeps the servitor at its labour should the one who summoned it
// depart, until the returned relent is called: a hangup is not heeded, and
// a write upon a stdout or stderr whose reader is gone fails (EPIPE) rather
// than striking the servitor down with SIGPIPE. An interrupt and SIGTERM
// are heeded as ever. It may be held several times at once; each relent
// lets go of its own hold, and calling it again does nothing.
//
// The signals are caught rather than ignored, so the processes an
// invocation summons are born heeding both as they would otherwise. A
// hangup already ignored when the servitor awoke stays ignored.
func Steadfast() (relent func()) {
	signals := []os.Signal{syscall.SIGPIPE}
	if !signal.Ignored(syscall.SIGHUP) {
		signals = append(signals, syscall.SIGHUP)
	}
	heard := make(chan os.Signal, 1) // never read: what it hears is let pass
	signal.Notify(heard, signals...)
	return sync.OnceFunc(func() { signal.Stop(heard) })
}
