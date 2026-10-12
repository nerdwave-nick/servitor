package exalted

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// hardenTrial marks the process of the trial below that hardens itself.
const hardenTrial = "SERVITOR_TRIAL_HARDEN"

// hardened is what the hardened process tells of itself.
type hardened struct {
	Dir         string
	Umask       string
	Core        uint64
	OpenSoft    uint64
	FileSize    uint64
	InheritedOn bool // close-on-exec of the descriptor it inherited
	Ignored     string
	// Weathered are the signals of the terminal it outlived, raised upon
	// itself; Summoned are those what it summons ignores.
	Weathered, Summoned string
}

// TestHarden_MakesTheProcessItsOwn: whatever its invoker left it — a hall,
// a umask, limits, an inherited descriptor — the exalted servitor dwells at
// the root with umask 022, dumps no core, bears limits of its own, lets no
// descriptor above 2 pass to what it summons, and outlives the signals of
// the terminal — yet what it summons ignores none of them, and SIGTERM is
// left to the servitor's own heed. It is seen in a process of its own.
func TestHarden_MakesTheProcessItsOwn(t *testing.T) {
	if os.Getenv(hardenTrial) == "1" {
		hardenAndTell()
		return
	}
	inherited, err := os.CreateTemp(t.TempDir(), "inherited")
	must(t, err)
	defer func() { _ = inherited.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), patience)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestHarden_MakesTheProcessItsOwn$", "-test.count=1")
	cmd.Env = append(os.Environ(), hardenTrial+"=1")
	cmd.Dir = t.TempDir()
	cmd.ExtraFiles = []*os.File{inherited}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the hardened process fell: %v\n%s", err, out)
	}
	var got hardened
	found := false
	for line := range strings.Lines(string(out)) {
		if report, ok := strings.CutPrefix(line, "HARDENED "); ok {
			must(t, json.Unmarshal([]byte(report), &got))
			found = true
		}
	}
	if !found {
		t.Fatalf("the hardened process told nothing:\n%s", out)
	}
	var hard unix.Rlimit
	must(t, unix.Getrlimit(unix.RLIMIT_NOFILE, &hard))
	want := hardened{Dir: "/", Umask: "0022", Core: 0, OpenSoft: min(openSoft, hard.Max), FileSize: got.FileSize,
		InheritedOn: true, Ignored: "", Weathered: "HUP INT QUIT PIPE TSTP TTIN TTOU", Summoned: ""}
	if got != want {
		t.Errorf("hardened as %+v\nwant          %+v", got, want)
	}
	var fsize unix.Rlimit
	must(t, unix.Getrlimit(unix.RLIMIT_FSIZE, &fsize))
	if got.FileSize != fsize.Max {
		t.Errorf("the limit of a scroll's size is %d, want %d", got.FileSize, fsize.Max)
	}
}

// hardenAndTell is the hardened process: it is left a hall, a umask and
// limits of its invoker's choosing, hardens itself and tells what it is.
func hardenAndTell() {
	unix.Umask(0o077)
	_ = unix.Setrlimit(unix.RLIMIT_NOFILE, &unix.Rlimit{Cur: 64, Max: rlimitMax(unix.RLIMIT_NOFILE)})
	_ = unix.Setrlimit(unix.RLIMIT_FSIZE, &unix.Rlimit{Cur: 1 << 20, Max: rlimitMax(unix.RLIMIT_FSIZE)})
	_ = unix.Setrlimit(unix.RLIMIT_CORE, &unix.Rlimit{Cur: rlimitMax(unix.RLIMIT_CORE), Max: rlimitMax(unix.RLIMIT_CORE)})
	if err := harden(); err != nil {
		fmt.Println("the process could not harden itself:", err)
		os.Exit(1)
	}
	var h hardened
	h.Dir, _ = os.Getwd()
	var core, open, size unix.Rlimit
	_ = unix.Getrlimit(unix.RLIMIT_CORE, &core)
	_ = unix.Getrlimit(unix.RLIMIT_NOFILE, &open)
	_ = unix.Getrlimit(unix.RLIMIT_FSIZE, &size)
	h.Core, h.OpenSoft, h.FileSize = core.Cur, open.Cur, size.Cur
	flags, _ := unix.FcntlInt(3, unix.F_GETFD, 0)
	h.InheritedOn = flags&unix.FD_CLOEXEC != 0
	status, _ := os.ReadFile("/proc/self/status")
	h.Umask, h.Ignored = told(string(status))
	var weathered []string
	for _, s := range terminalSignals {
		_ = syscall.Kill(os.Getpid(), s.sig)
		time.Sleep(10 * time.Millisecond) // a signal not heeded ends or stops the process ere long
		weathered = append(weathered, s.name)
	}
	h.Weathered = strings.Join(weathered, " ")
	child, err := exec.Command("/bin/sh", "-c", "cat /proc/self/status").Output()
	if err != nil {
		fmt.Println("the process could not summon:", err)
		os.Exit(1)
	}
	_, h.Summoned = told(string(child))
	data, _ := json.Marshal(h)
	fmt.Println("HARDENED " + string(data))
}

func rlimitMax(resource int) uint64 {
	var r unix.Rlimit
	_ = unix.Getrlimit(resource, &r)
	return r.Max
}

// terminalSignal is a signal of the terminal and its name.
type terminalSignal struct {
	sig  syscall.Signal
	name string
}

// terminalSignals are the signals of the terminal the exalted servitor
// outlives.
var terminalSignals = []terminalSignal{{syscall.SIGHUP, "HUP"}, {syscall.SIGINT, "INT"}, {syscall.SIGQUIT, "QUIT"}, {syscall.SIGPIPE, "PIPE"},
	{syscall.SIGTSTP, "TSTP"}, {syscall.SIGTTIN, "TTIN"}, {syscall.SIGTTOU, "TTOU"}}

// told is what /proc/<pid>/status tells: the umask, and which of the
// signals of the terminal and SIGTERM are ignored.
func told(status string) (umask, ignored string) {
	watched := append(slices.Clone(terminalSignals), terminalSignal{syscall.SIGTERM, "TERM"})
	for line := range strings.Lines(status) {
		if v, ok := strings.CutPrefix(line, "Umask:"); ok {
			umask = strings.TrimSpace(v)
		}
		if v, ok := strings.CutPrefix(line, "SigIgn:"); ok {
			mask, _ := strconv.ParseUint(strings.TrimSpace(v), 16, 64)
			var names []string
			for _, s := range watched {
				if mask&(1<<(uint(s.sig)-1)) != 0 {
					names = append(names, s.name)
				}
			}
			ignored = strings.Join(names, " ")
		}
	}
	return umask, ignored
}
