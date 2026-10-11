package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// openTerminal opens a new pseudo-terminal: its master and its servant.
func openTerminal() (master, servant *os.File, err error) {
	master, err = os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}
	var n uint32
	var unlock int32
	for _, c := range []struct {
		req uintptr
		arg unsafe.Pointer
	}{{syscall.TIOCGPTN, unsafe.Pointer(&n)}, {syscall.TIOCSPTLCK, unsafe.Pointer(&unlock)}} {
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), c.req, uintptr(c.arg)); errno != 0 {
			_ = master.Close()
			return nil, nil, errno
		}
	}
	servant, err = os.OpenFile("/dev/pts/"+strconv.Itoa(int(n)), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		_ = master.Close()
	}
	return master, servant, err
}

// beCaller is a caller as a picker or a shell is one: it invokes the
// servitor upon a terminal it holds (args: where to write the servitor's
// pid, where to keep what it hears, then the servitor's own arguments),
// listens to its stdout and stderr, and lives until it is slain. Its death
// closes the terminal, which hangs up, and both readers.
func beCaller(args []string) int {
	pidFile, heardFile := args[0], args[1]
	master, servant, err := openTerminal()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	r, w, err := os.Pipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	cmd := exec.Command(os.Args[0], args[2:]...)
	cmd.Env = append(os.Environ(), awakenAs+"=servitor")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = servant, w, w
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_, _ = w.Close(), servant.Close()
	heard, err := os.Create(heardFile)
	if err == nil {
		err = os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_, _ = io.Copy(heard, r)
	time.Sleep(time.Minute)
	runtime.KeepAlive(master) // the terminal stays open until the caller dies
	return 0
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

// within waits until ok holds.
func within(t *testing.T, ok func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatal(what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestInvoke_SurvivesTheDeathOfItsCaller: the caller who invoked the
// servitor upon its terminal is slain mid-liturgy; its terminal hangs up
// and its readers are gone, yet the servitor, orphaned, performs the rite
// or edict whole, tells it to no one and chronicles its triumph.
func TestInvoke_SurvivesTheDeathOfItsCaller(t *testing.T) {
	for _, edict := range []bool{false, true} {
		e := newEnv(t)
		marker, release := e.lingering("lingering", edict)
		pidFile, heardFile := filepath.Join(e.state, "pid"), filepath.Join(e.state, "heard")
		args := []string{pidFile, heardFile, "--librarium", e.cfgDir, "invoke", "lingering"}
		if !edict {
			args = append(args, "on")
		}
		caller := exec.Command(os.Args[0], args...)
		caller.Env = append(os.Environ(), awakenAs+"=caller")
		if err := caller.Start(); err != nil {
			t.Fatal(err)
		}
		await(t, marker, "the gate was never reached")
		heard := func() string { b, _ := os.ReadFile(heardFile); return string(b) }
		within(t, func() bool { return strings.Contains(heard(), "✔ verse 1") },
			"the servitor did not narrate upon its terminal")
		pid, err := strconv.Atoi(func() string { b, _ := os.ReadFile(pidFile); return string(b) }())
		if err != nil {
			t.Fatal(err)
		}
		_ = caller.Process.Kill()
		_ = caller.Wait()
		time.Sleep(50 * time.Millisecond) // a heeded hangup would end the servitor here
		if ended(pid) {
			t.Fatalf("edict=%v: the servitor died with its caller; it told:\n%s", edict, heard())
		}
		touch(t, release)
		within(t, func() bool { return ended(pid) }, "the servitor never ended")
		if lines := e.chronicleLines(); len(lines) != 1 || !strings.Contains(lines[0], `"verdict":"triumph"`) {
			t.Fatalf("edict=%v: chronicle %q", edict, lines)
		}
		wholeOrNothing(t, e, edict, false)
	}
}
