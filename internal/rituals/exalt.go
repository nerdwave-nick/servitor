package rituals

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"golang.org/x/sys/unix"

	"github.com/nerdwave-nick/servitor/internal/exalted"
)

// Summoner raises the exalted servitor through the intercessor's cmd, its
// channel laid upon end as its descriptors 0 and 1, and returns what awaits
// the intercessor's end. It keeps end only until it returns: the caller
// closes it then.
type Summoner func(cmd *exec.Cmd, end *os.File) (wait func() error, err error)

// Summon is how the servitor raises the exalted servitor when it is given
// no Summoner of its own; the trials raise it within their own process.
var Summon Summoner = summon

// summon starts the intercessor upon the machine. Before it does, no other
// process of the master may trace the servitor or read its memory any more
// (PR_SET_DUMPABLE): the channel is the master's word to the root. The
// intercessor speaks upon the servitor's own stderr, unless the caller
// laid another upon cmd.
func summon(cmd *exec.Cmd, end *os.File) (func() error, error) {
	if err := unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0); err != nil {
		return nil, err
	}
	cmd.Stdin, cmd.Stdout = end, end
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd.Wait, nil
}

// Exaltation is how the master is asked for the proof the intercessor
// demands: whether a terminal or the cogitator attends, so that the master
// beholds what they sanction, and how the exalted servitor is raised (nil
// is Summon). The cogitator raises it in a Summoner of its own, lending
// sudo its terminal. An invocation needs no terminal attending: the
// intercessor asks as it may.
type Exaltation struct {
	Attended bool
	Summon   Summoner
	// Answered, when not nil, is told once the exalted servitor raised gave
	// its first answer, or its channel ended before any: the intercessor
	// asks the master for no proof any more. The cogitator then takes back
	// the terminal it lent sudo. It is never told when no intercessor was
	// called upon.
	Answered func()
}

func (x Exaltation) summoner() Summoner {
	if x.Summon == nil {
		return Summon
	}
	return x.Summon
}

// Dismissed is the lament of a proof the intercessor did not accept: the
// master dismissed it, or failed it. Nothing was written.
type Dismissed struct {
	Deed        string // e.g. `the sanctioning of the rite "x"`
	Intercessor string
	Err         error // how the intercessor ended
}

func (d *Dismissed) Error() string {
	return fmt.Sprintf("%s is forgone: the intercessor %s did not vouch for the master — the proof was dismissed or "+
		"refused — and the exalted servitor touched nothing", d.Deed, d.Intercessor)
}

func (d *Dismissed) Unwrap() error { return d.Err }

// Spurned is the lament of an exalted servitor that refused what it was
// bidden; it touched nothing.
type Spurned struct {
	Deed   string
	Lament string
}

func (s *Spurned) Error() string {
	return fmt.Sprintf("%s is refused by the exalted servitor, which touched nothing: %s", s.Deed, s.Lament)
}

// conduit is the channel of the invoker's servitor to an exalted servitor
// raised through the intercessor, and what awaits the intercessor's end.
type conduit struct {
	ours     *os.File
	ch       *exalted.Channel
	wait     func() error
	answered func() // told once the intercessor asks no more; may be nil
	told     sync.Once
	once     sync.Once
	ended    error // how the intercessor ended, once it did
}

// raiseConduit lays a socketpair and raises the exalted servitor through
// cmd upon its other end, as x summons it.
func raiseConduit(deed, intercessor string, cmd *exec.Cmd, x Exaltation) (*conduit, error) {
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("%s is refused: no channel to the exalted servitor can be laid: %w", deed, err)
	}
	ours, theirs := os.NewFile(uintptr(fds[0]), "servitor"), os.NewFile(uintptr(fds[1]), "exalted servitor")
	wait, err := x.summoner()(cmd, theirs)
	_ = theirs.Close()
	if err != nil {
		_ = ours.Close()
		return nil, fmt.Errorf("%s is refused: the intercessor %s cannot be called upon: %w", deed, intercessor, err)
	}
	return &conduit{ours: ours, ch: exalted.NewChannel(ours, ours), wait: wait, answered: x.Answered}, nil
}

// answer hears the first answer of the exalted servitor; once it is heard,
// or the channel ended before it, the intercessor asks the master no more.
func (c *conduit) answer() (exalted.Reply, error) {
	r, err := c.ch.Answer()
	c.tell()
	return r, err
}

// tell tells, once, that the intercessor asks the master no more.
func (c *conduit) tell() {
	c.told.Do(func() {
		if c.answered != nil {
			c.answered()
		}
	})
}

// silence lets the exalted servitor hear no more words, and the invoker's
// servitor hear none of its: an exalted servitor that awaits more words
// hears the end of its channel.
func (c *conduit) silence() { _ = unix.Shutdown(int(c.ours.Fd()), unix.SHUT_RDWR) }

// end awaits the end of the intercessor, once, and closes the channel; an
// intercessor that ended asks the master no more.
func (c *conduit) end() error {
	c.once.Do(func() {
		c.ended = c.wait()
		_ = c.ours.Close()
		c.tell()
	})
	return c.ended
}

// exalt raises the exalted servitor through cmd as x summons it, speaks to
// it with speak, and hears its one answer. The end of the channel before
// any answer is *Dismissed when the intercessor ended amiss; an exalted
// servitor of another protocol is *exalted.Mismatch.
func exalt(deed, intercessor string, cmd *exec.Cmd, x Exaltation, speak func(*exalted.Channel) error) (
	exalted.Reply, error) {
	c, err := raiseConduit(deed, intercessor, cmd, x)
	if err != nil {
		return exalted.Reply{}, err
	}
	spoken := make(chan error, 1)
	go func() { spoken <- speak(c.ch) }()
	reply, err := c.answer()
	if err != nil {
		c.silence()
	}
	ended := c.end()
	<-spoken // what it could not hear, its answer told
	if err != nil {
		return exalted.Reply{}, unanswered(deed, intercessor, err, ended)
	}
	return reply, nil
}

// unanswered is the lament of an exalted servitor that gave no answer (err)
// to deed; the intercessor ended so (ended).
func unanswered(deed, intercessor string, err, ended error) error {
	switch {
	case errors.Is(err, io.EOF) && ended != nil:
		return &Dismissed{Deed: deed, Intercessor: intercessor, Err: ended}
	case errors.Is(err, io.EOF):
		return fmt.Errorf("%s is refused: the exalted servitor ended without a word, and touched nothing", deed)
	}
	return fmt.Errorf("%s is refused: %w", deed, err)
}
