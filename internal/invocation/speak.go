package invocation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// Speaking a command. Each command runs in a process group of its own, its
// standard input empty and its standard output and error captured together
// (never printed: the caller decides what to show of a step's Output).
//
// When its patience runs out the whole group is slain with SIGKILL and the
// step falls. When it ends in triumph, children it sent into the background
// live on: should they still hold its output open, the servitor waits only
// linger before it stops listening, so the invocation never hangs on them.
// A child that keeps speaking into the closed output afterwards is struck
// by SIGPIPE; children meant to outlive their step should send their words
// elsewhere (e.g. `swaybg … >/dev/null 2>&1 &`).

// linger is how long the servitor keeps listening after a command ended
// while its children still hold its output open (exec.Cmd.WaitDelay).
const linger = 500 * time.Millisecond

// maxEcho is how much a command's output is kept: its last words.
const maxEcho = 64 << 10

// voice is one command made ready to be spoken.
type voice struct {
	argv     []string
	dir      string   // where it is spoken: the rite's directory
	env      []string // the whole environment it is spoken in
	patience time.Duration
}

// speak runs the command to its end and returns what it uttered.
func (v voice) speak(halt context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(halt, v.patience)
	defer cancel()
	cmd := exec.CommandContext(ctx, v.argv[0], v.argv[1:]...)
	cmd.Dir, cmd.Env = v.dir, v.env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = linger
	var out echo
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return string(out.b), v.judge(ctx, err)
}

// judge speaks the grimdark words for how a command ended; nil is triumph.
func (v voice) judge(ctx context.Context, err error) error {
	var exit *exec.ExitError
	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		return errHalted
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return fmt.Errorf("its patience of %v ran out, and the servitor slew it together with every process "+
			"it had summoned", v.patience)
	case err == nil, errors.Is(err, exec.ErrWaitDelay):
		return nil
	case errors.As(err, &exit) && exit.Exited():
		return fmt.Errorf("it ended bearing the death-mark %d, a sign that its work was not done", exit.ExitCode())
	case errors.As(err, &exit):
		if ws, ok := exit.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return fmt.Errorf("it was struck down by the signal %q before it could end", ws.Signal())
		}
	}
	return fmt.Errorf("it could not be awakened: %w", lament(v.argv[0], unwrapStart(err)))
}

// errHalted is the heresy of a step the master halted, or would have begun
// after the halt.
var errHalted = errors.New("the invocation was halted at its master's command; the servitor slew whatever the " +
	"step had summoned and began no further step")

// unwrapStart lays bare the cause of a failed start, beneath the plain
// words exec wraps it in.
func unwrapStart(err error) error {
	var pe *os.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	var ee *exec.Error
	if errors.As(err, &ee) {
		return ee.Err
	}
	return err
}

// echo keeps the last maxEcho bytes a command uttered.
type echo struct{ b []byte }

func (e *echo) Write(p []byte) (int, error) {
	e.b = append(e.b, p...)
	if over := len(e.b) - maxEcho; over > 0 {
		e.b = append(e.b[:0], e.b[over:]...)
	}
	return len(p), nil
}

// Variables of the environment every incantation and litany is spoken in.
const (
	envAspect            = "SERVITOR_ASPECT"
	envFormerAspect      = "SERVITOR_FORMER_ASPECT"
	envRite              = "SERVITOR_RITE"
	envInscriptionPrefix = "SERVITOR_INSCRIPTION_"
)

// environ is the environment of the servitor with the invocation's own
// variables set: the aspect, the former aspect, the rite and every declared
// inscription (empty when unset). Inscriptions inherited from an enclosing
// invocation are cast out, so a command sees only its own rite's.
func environ(dir string, v placeholder.Values, keys []string) []string {
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, envInscriptionPrefix) {
			env = append(env, kv)
		}
	}
	env = append(env, "PWD="+dir, envAspect+"="+v.Aspect, envFormerAspect+"="+v.Former, envRite+"="+v.Rite)
	for _, k := range keys {
		env = append(env, inscriptionVariable(k)+"="+v.Inscriptions[k])
	}
	return env
}

// inscriptionVariable names the variable of the environment that carries
// the inscription key: SERVITOR_INSCRIPTION_ and the key in capitals, every
// "-" and "." made "_".
func inscriptionVariable(key string) string {
	return envInscriptionPrefix + strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(key))
}
