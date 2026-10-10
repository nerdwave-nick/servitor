package augury

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// auspexLinger is how long the servitor keeps listening after the auspex
// ended while children it sent away still hold its output open.
const auspexLinger = 100 * time.Millisecond

// maxAuspexWords is how much the servitor hears of an auspex: an aspect's
// name is short, and longer words name no aspect.
const maxAuspexWords = 4 << 10

// auspex awakens the rite's auspex and returns the declared aspect it
// printed. It is spoken in the rite's tongue (else the settings', else
// bash) in its own process group, from the rite's directory, with the
// environment of the rite's commands: the target aspect of opts, no former
// aspect and the data-slate's inscriptions. When its patience runs out the
// whole group is slain. A failure, an impatient auspex and words that name
// no declared aspect give no omen; only its standard output is heard.
func (s *seer) auspex(opts Options) (string, bool) {
	v := placeholder.Values{Aspect: opts.Target, Rite: s.rite.Name}
	if s.slate != nil {
		v.Inscriptions = s.slate.Inscriptions
	}
	command, ok := s.render(s.rite.Auspex.Rite, v)
	if !ok {
		return "", false
	}
	tongue := librarium.DefaultTongue
	for _, t := range []string{opts.Orders.Tongue, s.rite.Tongue} {
		if t != "" {
			tongue = t
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.rite.Auspex.Wait())
	defer cancel()
	cmd := exec.CommandContext(ctx, tongue, "-c", command)
	cmd.Dir, cmd.Env = s.rite.Dir(), invocation.Environ(s.rite, v)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = auspexLinger
	var heard words
	cmd.Stdout = &heard
	err := cmd.Run()
	if ctx.Err() != nil || (err != nil && !errors.Is(err, exec.ErrWaitDelay)) {
		return "", false
	}
	aspect := strings.TrimSpace(string(heard.b))
	return aspect, s.rite.HasAspect(aspect)
}

// words keeps the first maxAuspexWords bytes spoken to it.
type words struct{ b []byte }

func (w *words) Write(p []byte) (int, error) {
	if room := maxAuspexWords - len(w.b); room > 0 {
		w.b = append(w.b, p[:min(room, len(p))]...)
	}
	return len(p), nil
}
