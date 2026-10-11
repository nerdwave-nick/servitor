package cli

import (
	"strings"
	"syscall"
	"testing"
)

// TestInvoke_ASlainHolderLeavesNoDominion: a servitor holding dominion
// mid-liturgy is slain by SIGKILL; the next invocation of the rite neither
// waits nor tells of an awaiting.
func TestInvoke_ASlainHolderLeavesNoDominion(t *testing.T) {
	for _, edict := range []bool{false, true} {
		e := newEnv(t)
		e.terminal(80)
		marker, release := e.lingering("lingering", edict)
		args := []string{"invoke", "lingering", "on"}
		if edict {
			args = args[:2]
		}
		holder := e.servitorProcess(args...)
		if err := holder.Start(); err != nil {
			t.Fatal(err)
		}
		await(t, marker, "the holder never reached the gate")
		// Its session is its own: slay it with every process it summoned.
		if err := syscall.Kill(-holder.Process.Pid, syscall.SIGKILL); err != nil {
			t.Fatal(err)
		}
		_ = holder.Wait()

		touch(t, release)
		s := e.start(args...)
		s.ended(t)
		if s.code != 0 || strings.Contains(s.stdout.String(), "⋯") {
			t.Fatalf("edict %v: exit %d\nstdout: %s\nstderr: %s", edict, s.code, s.stdout.String(), s.stderr.String())
		}
	}
}
