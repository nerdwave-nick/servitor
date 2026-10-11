package cli

import (
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// readerHeed is how often the servitor looks whether the reader of a vigil
// has departed.
const readerHeed = 50 * time.Millisecond

// watchReader calls departed once the reader of w departs, while the
// returned stop is not yet called. Only a pipe or socket knows its reader:
// for any other w it watches nothing.
func watchReader(w io.Writer, departed func()) (stop func()) {
	f, ok := w.(*os.File)
	if !ok {
		return func() {}
	}
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		fds := []unix.PollFd{{Fd: int32(f.Fd())}} // no event sought: a departure is told unasked
		for {
			select {
			case <-done:
				return
			default:
			}
			n, err := unix.Poll(fds, int(readerHeed/time.Millisecond))
			if err != nil && err != unix.EINTR {
				return
			}
			switch {
			case n == 0:
			case fds[0].Revents&(unix.POLLERR|unix.POLLHUP) != 0:
				departed()
				return
			default: // no longer a stream to watch
				return
			}
		}
	})
	return func() { close(done); wg.Wait() }
}
