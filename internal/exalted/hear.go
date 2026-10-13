package exalted

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"syscall"
	"time"

	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// auspexLinger is how long the exalted servitor keeps listening after a
// word of an auspex ended while what it summoned still holds its output
// open, as the invoker's servitor does.
const auspexLinger = 100 * time.Millisecond

// auspexCap is how much the exalted servitor hears of what an auspex
// printed: an omen is short, and longer words name none.
const auspexCap = 4 << 10

// Hear speaks words of the privileged auspex of the machine-spirit v in
// the working hall, in its tongue and under the patience of its auspex, in
// a session of its own whose processes are slain when the session of the
// exalted servitor ends; a trial hears its omen as SERVITOR_OMEN. Only its
// standard output is heard, at most auspexCap bytes of it.
func (sp *speaker) Hear(ctx context.Context, v sanction.Verse, words, omen string) (string, bool, error) {
	if v.Auspex == nil {
		return "", false, fmt.Errorf("%s bears no exalted auspex", v.Named())
	}
	tongue, err := sp.spoken(v)
	if err != nil {
		return "", false, err
	}
	env := sp.env
	if omen != "" {
		env = append(slices.Clone(env), sanction.OmenVariable+"="+omen)
	}
	ctx, cancel := context.WithTimeout(ctx, v.Auspex.Patience)
	defer cancel()
	cmd := exec.CommandContext(ctx, tongue[0], append(slices.Clone(tongue[1:]), "-c", words)...)
	cmd.Dir, cmd.Env = sp.hall, env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = auspexLinger
	var printed capped
	cmd.Stdout = &printed
	if err := cmd.Start(); err != nil {
		return "", false, err
	}
	sp.summoned(cmd.Process.Pid)
	err = cmd.Wait()
	var exit *exec.ExitError
	switch {
	case ctx.Err() != nil:
		return "", false, fmt.Errorf("its patience of %v ran out", v.Auspex.Patience)
	case err == nil, errors.Is(err, exec.ErrWaitDelay):
		return string(printed.b), true, nil
	case errors.As(err, &exit) && !exit.Sys().(syscall.WaitStatus).Signaled():
		return string(printed.b), false, nil
	}
	return "", false, fmt.Errorf("it was slain: %v", err)
}

// capped keeps the first auspexCap bytes spoken to it.
type capped struct{ b []byte }

func (c *capped) Write(p []byte) (int, error) {
	if room := auspexCap - len(c.b); room > 0 {
		c.b = append(c.b, p[:min(room, len(p))]...)
	}
	return len(p), nil
}
