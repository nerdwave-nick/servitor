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

// Places bound how many processes of auspexes labour at once: each takes a
// place while it labours and gives it back when it ends, so no augury waits
// upon itself. A nil Places bounds none.
type Places chan struct{}

// take waits for a place; false when ctx ended first. A nil Places always
// has room.
func (p Places) take(ctx context.Context) bool {
	if p == nil {
		return true
	}
	select {
	case p <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (p Places) give() {
	if p != nil {
		<-p
	}
}

// fate is how the words of an auspex ended.
type fate int

const (
	spoke  fate = iota // they ended with the death-mark 0
	fell               // they ended with another death-mark
	silent             // they outlasted their patience, were slain or could not be spoken
)

// speech is one process of an auspex: its words, spoken in tongue from
// hall with env, slain with its whole process group when patience runs out.
type speech struct {
	tongue, words, hall string
	env                 []string
	patience            time.Duration
}

// speak speaks sp once it holds a place among places, and returns what it
// printed on stdout (at most maxAuspexWords, when hear) and how it ended.
// When ctx ends its process group is slain and it is silent.
func (sp speech) speak(ctx context.Context, places Places, hear bool) (string, fate) {
	if !places.take(ctx) {
		return "", silent
	}
	defer places.give()
	ctx, cancel := context.WithTimeout(ctx, sp.patience)
	defer cancel()
	cmd := exec.CommandContext(ctx, sp.tongue, "-c", sp.words)
	cmd.Dir, cmd.Env = sp.hall, sp.env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = auspexLinger
	var heard words
	if hear {
		cmd.Stdout = &heard
	}
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case ctx.Err() != nil:
		return "", silent
	case err == nil, errors.Is(err, exec.ErrWaitDelay):
		return string(heard.b), spoke
	case errors.As(err, &exit) && !exit.Sys().(syscall.WaitStatus).Signaled():
		return string(heard.b), fell
	}
	return "", silent
}

// tongueOf is the first tongue named of tongues, from the most particular
// down, else that of the settings, else bash.
func tongueOf(orders librarium.Orders, tongues ...string) string {
	for _, t := range tongues {
		if t != "" {
			return t
		}
	}
	if orders.Tongue != "" {
		return orders.Tongue
	}
	return librarium.DefaultTongue
}

// auspex awakens the rite's auspex and returns the declared aspect it
// printed. It is spoken in the rite's tongue (else the settings', else
// bash) in its own process group, from the rite's directory, with the
// environment of the rite's commands: the target aspect of opts, no former
// aspect and the data-slate's inscriptions. When its patience runs out the
// whole group is slain. A failure, an impatient auspex and words that name
// no declared aspect give no omen; only its standard output is heard.
// When ctx ends the group is slain as well, and the auspex gives no omen.
func (s *seer) auspex(ctx context.Context, opts Options) (string, bool) {
	v := s.auspexValues(opts)
	command, ok := s.render(s.rite.Auspex.Rite, v)
	if !ok {
		return "", false
	}
	sp := speech{tongue: tongueOf(opts.Orders, s.rite.Tongue), words: command, hall: s.rite.Dir(),
		env: invocation.Environ(s.rite, v), patience: s.rite.Auspex.Wait()}
	heard, f := sp.speak(ctx, opts.Places, true)
	if f != spoke {
		return "", false
	}
	aspect := strings.TrimSpace(heard)
	return aspect, s.rite.HasAspect(aspect)
}

// auspexValues are the values every auspex is rendered with: the target
// aspect of opts, no former aspect and the data-slate's inscriptions.
func (s *seer) auspexValues(opts Options) placeholder.Values {
	v := placeholder.Values{Aspect: opts.Target, Rite: s.rite.Name}
	if s.slate != nil {
		v.Inscriptions = s.slate.Inscriptions
	}
	return v
}

// words keeps the first maxAuspexWords bytes spoken to it.
type words struct{ b []byte }

func (w *words) Write(p []byte) (int, error) {
	if room := maxAuspexWords - len(w.b); room > 0 {
		w.b = append(w.b, p[:min(room, len(p))]...)
	}
	return len(p), nil
}
