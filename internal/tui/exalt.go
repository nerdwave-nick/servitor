package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/intercessor"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
)

// intercession lends one exalted act of the cogitator — an invocation, a
// sanctioning or an abjuring — what its intercessor needs to ask for the
// master's proof. sudo asks at the terminal, so the cogitator lends it the
// terminal (tea.Exec) until the exalted servitor first answers, or the
// channel ends unanswered, and then takes it back; the channel to the
// exalted servitor stays upon its socketpair throughout. run0 and pkexec
// are summoned in a session of their own, with no terminal to ask at, so
// that the graphical agent asks. What the intercessor and the exalted
// servitor speak upon stderr is kept, never printed upon the cogitator's
// screen, to be shown in its lament. After an act that called upon run0,
// run0 is bidden forget the master's proof, so that the next act asks anew.
//
// The exalted servitor alone ever performs an exalted verse: the cogitator
// only summons it, as the command line does.
type intercession struct {
	send     func(tea.Msg)
	gone     <-chan struct{} // closed once the cogitator has ended
	answered chan struct{}   // closed once the intercessor asks no more
	told     sync.Once
	words    words
	forget   *exec.Cmd // run0 -k, once run0 was called upon
}

// intercede readies the intercession of one exalted act.
func (m *model) intercede() *intercession {
	return &intercession{send: m.send, gone: m.gone, answered: make(chan struct{})}
}

// exaltation is how the act asks for the master's proof: the cogitator
// attends, so that the master beholds what they sanction.
func (c *intercession) exaltation() *rituals.Exaltation {
	return &rituals.Exaltation{Attended: true, Summon: c.summon, Answered: c.answer}
}

// answer tells that the intercessor asks the master no more.
func (c *intercession) answer() { c.told.Do(func() { close(c.answered) }) }

// summon raises the exalted servitor through the intercessor's cmd, its
// channel laid upon end, as rituals.Summon raises it: sudo upon the
// terminal the cogitator lends it, run0 and pkexec in a session of their
// own.
func (c *intercession) summon(cmd *exec.Cmd, end *os.File) (func() error, error) {
	cmd.Stderr = &c.words
	c.forget = intercessor.Forgetting(cmd)
	if intercessor.Named(cmd) != librarium.IntercessorSudo {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		return rituals.Summon(cmd, end)
	}
	l := &lent{cmd: cmd, end: end, words: &c.words, answered: c.answered, started: make(chan begun, 1)}
	c.send(lendMsg{l})
	select {
	case b := <-l.started:
		return b.wait, b.err
	case <-c.gone:
		return nil, errors.New("the cogitator ended before it could lend sudo its terminal")
	}
}

// forgotten bids run0 forget the master's proof, when the act called upon
// it; the lament tells that it could not be bidden.
func (c *intercession) forgotten() error {
	if c.forget == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.forget.Path)
	cmd.Args = c.forget.Args
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("run0 could not be bidden forget the master's proof (%s): %v %s — the next invocation "+
			"may not ask it anew", strings.Join(c.forget.Args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// lendMsg asks the cogitator to lend sudo its terminal; reclaimedMsg tells
// that it took it back.
type (
	lendMsg      struct{ l *lent }
	reclaimedMsg struct {
		l   *lent
		err error
	}
)

// lend lends the terminal to sudo, as m.exec lends it.
func (m *model) lend(l *lent) tea.Cmd {
	return m.exec(l, func(err error) tea.Msg { return reclaimedMsg{l: l, err: err} })
}

// lent is sudo upon the terminal the cogitator lends it, as tea.Exec runs
// it: it is started upon the channel, and the terminal is taken back once
// the intercessor asks the master no more, while sudo and the exalted
// servitor it raised go on.
type lent struct {
	cmd      *exec.Cmd
	end      *os.File
	words    *words
	answered <-chan struct{}
	terminal io.Writer
	once     sync.Once
	started  chan begun
}

// begun is how the summoning of sudo began.
type begun struct {
	wait func() error
	err  error
}

// The channel is sudo's stdin and stdout, never the terminal; sudo asks
// upon the terminal itself, and its laments are shown upon it while it is
// lent.
func (l *lent) SetStdin(io.Reader)    {}
func (l *lent) SetStdout(io.Writer)   {}
func (l *lent) SetStderr(w io.Writer) { l.terminal = w }

// Run starts sudo and holds the terminal until it asks no more.
func (l *lent) Run() error {
	l.once.Do(func() {
		l.words.echo(l.terminal)
		defer l.words.echo(nil)
		wait, err := rituals.Summon(l.cmd, l.end)
		l.started <- begun{wait: wait, err: err}
		if err == nil {
			<-l.answered
		}
	})
	return nil
}

// forgo tells the summoning that the terminal could never be lent.
func (l *lent) forgo(err error) {
	l.once.Do(func() {
		if err == nil {
			err = errors.New("the terminal could not be lent")
		}
		l.started <- begun{err: fmt.Errorf("the cogitator could not lend sudo its terminal: %w", err)}
	})
}

// wordsCap is the most the cogitator keeps of what an intercessor and its
// exalted servitor speak upon stderr.
const wordsCap = 64 << 10

// words keeps what is spoken upon stderr, and shows it upon the terminal
// as well while that is lent.
type words struct {
	mu   sync.Mutex
	kept []byte
	to   io.Writer
}

func (w *words) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if room := wordsCap - len(w.kept); room > 0 {
		w.kept = append(w.kept, p[:min(room, len(p))]...)
	}
	if w.to != nil {
		_, _ = w.to.Write(p)
	}
	return len(p), nil
}

// echo shows what is spoken upon to as well; nil shows it nowhere.
func (w *words) echo(to io.Writer) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.to = to
}

func (w *words) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return string(w.kept)
}

// spoken renders what the intercessor and the exalted servitor spoke upon
// stderr; nothing when they were silent.
func (m *model) spoken(words string) []string {
	lines := uttered("the words of the intercessor and the exalted servitor", words)
	if lines == nil {
		return nil
	}
	for i, l := range lines {
		lines[i] = m.t.dim.Render(l)
	}
	return append([]string{""}, lines...)
}
