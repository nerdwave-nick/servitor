package exalted

import (
	"errors"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

// The limits the exalted servitor sets upon itself, whatever its invoker
// left it: no core is ever dumped (it holds what only the root may read),
// a scroll may grow as the machine allows, and the descriptors are as many
// as the machine grants by custom.
const (
	openSoft = 1024
	openHard = 524288
)

// ignored are the signals of the terminal the exalted servitor ignores: an
// interrupt, a hang-up or a stop of its invoker's terminal never cuts a
// verse in half. SIGTERM alone ends it, as an abort.
var ignored = []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTSTP, syscall.SIGTTIN,
	syscall.SIGTTOU, syscall.SIGPIPE}

// harden makes the exalted servitor's own process its own, whatever its
// invoker and intercessor left it: it dwells at the root of the machine with
// umask 022, dumps no core, bears limits of its own, lets no descriptor
// above 2 pass to what it summons, and ignores the signals of the terminal.
func harden() error {
	if err := os.Chdir("/"); err != nil {
		return err
	}
	unix.Umask(0o022)
	if err := unix.Setrlimit(unix.RLIMIT_CORE, &unix.Rlimit{}); err != nil {
		return err
	}
	if err := limit(unix.RLIMIT_FSIZE, unix.RLIM_INFINITY, unix.RLIM_INFINITY); err != nil {
		return err
	}
	if err := limit(unix.RLIMIT_NOFILE, openSoft, openHard); err != nil {
		return err
	}
	if err := sealDescriptors(); err != nil {
		return err
	}
	signal.Ignore(ignored...)
	return nil
}

// limit sets the limit of resource to soft and hard; when the hard limit
// cannot be raised (only the root may), the hard limit stays and the soft
// limit is held beneath it.
func limit(resource int, soft, hard uint64) error {
	// syscall.Setrlimit, so that the runtime knows the limit of descriptors
	// it hands to what it summons.
	err := syscall.Setrlimit(resource, &syscall.Rlimit{Cur: soft, Max: hard})
	if !errors.Is(err, unix.EPERM) {
		return err
	}
	var now syscall.Rlimit
	if err := syscall.Getrlimit(resource, &now); err != nil {
		return err
	}
	return syscall.Setrlimit(resource, &syscall.Rlimit{Cur: min(soft, now.Max), Max: now.Max})
}

// sealDescriptors makes every descriptor above 2 close upon exec.
func sealDescriptors() error {
	err := unix.CloseRange(3, ^uint(0), unix.CLOSE_RANGE_CLOEXEC)
	if !errors.Is(err, unix.ENOSYS) && !errors.Is(err, unix.EINVAL) {
		return err
	}
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if fd, err := strconv.Atoi(e.Name()); err == nil && fd > 2 {
			unix.CloseOnExec(fd)
		}
	}
	return nil
}
