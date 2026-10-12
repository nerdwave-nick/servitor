package rituals

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

// TestSummon_KeepsTheStderrLaidUponIt: the intercessor speaks upon the
// stderr its caller laid upon it (the cogitator captures its words, for the
// screen is its own), and upon the servitor's own only when none was laid;
// the channel lies upon its stdin and stdout.
func TestSummon_KeepsTheStderrLaidUponIt(t *testing.T) {
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	must(t, err)
	ours, theirs := os.NewFile(uintptr(fds[0]), "servitor"), os.NewFile(uintptr(fds[1]), "exalted servitor")
	defer func() { _ = ours.Close() }()
	var words strings.Builder
	cmd := exec.Command("/bin/sh", "-c", "echo lament >&2; echo answer")
	cmd.Stderr = &words
	wait, err := summon(cmd, theirs)
	_ = theirs.Close()
	must(t, err)
	must(t, wait())
	heard := make([]byte, 64)
	n, _ := ours.Read(heard)
	if words.String() != "lament\n" || string(heard[:n]) != "answer\n" {
		t.Errorf("the intercessor spoke %q upon stderr and %q upon the channel", words.String(), heard[:n])
	}
}
