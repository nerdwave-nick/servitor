package rituals

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

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
// (PR_SET_DUMPABLE): the channel is the master's word to the root.
func summon(cmd *exec.Cmd, end *os.File) (func() error, error) {
	if err := unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0); err != nil {
		return nil, err
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = end, end, os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd.Wait, nil
}

// Exaltation is how the master is asked for the proof the intercessor
// demands: whether a terminal or the cogitator attends, so that the master
// beholds what they sanction, and how the exalted servitor is raised (nil
// is Summon). The cogitator raises it in a Summoner of its own, lending
// sudo its terminal.
type Exaltation struct {
	Attended bool
	Summon   Summoner
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

// exalt raises the exalted servitor through cmd, speaks to it with speak,
// and hears its one answer. The end of the channel before any answer is
// *Dismissed when the intercessor ended amiss; an exalted servitor of
// another protocol is *exalted.Mismatch.
func exalt(deed, intercessor string, cmd *exec.Cmd, summoner Summoner, speak func(*exalted.Channel) error) (
	exalted.Reply, error) {
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return exalted.Reply{}, fmt.Errorf("%s is refused: no channel to the exalted servitor can be laid: %w", deed, err)
	}
	ours, theirs := os.NewFile(uintptr(fds[0]), "servitor"), os.NewFile(uintptr(fds[1]), "exalted servitor")
	defer func() { _ = ours.Close() }()
	wait, err := summoner(cmd, theirs)
	_ = theirs.Close()
	if err != nil {
		return exalted.Reply{}, fmt.Errorf("%s is refused: the intercessor %s cannot be called upon: %w", deed,
			intercessor, err)
	}
	ch := exalted.NewChannel(ours, ours)
	spoken := make(chan error, 1)
	go func() { spoken <- speak(ch) }()
	reply, err := ch.Answer()
	if err != nil {
		_ = unix.Shutdown(int(ours.Fd()), unix.SHUT_RDWR) // an exalted servitor that awaits more words hears none
	}
	ended := wait()
	<-spoken // what it could not hear, its answer told
	switch {
	case errors.Is(err, io.EOF) && ended != nil:
		return exalted.Reply{}, &Dismissed{Deed: deed, Intercessor: intercessor, Err: ended}
	case errors.Is(err, io.EOF):
		return exalted.Reply{}, fmt.Errorf("%s is refused: the exalted servitor ended without a word, and touched "+
			"nothing", deed)
	case err != nil:
		return exalted.Reply{}, fmt.Errorf("%s is refused: %w", deed, err)
	}
	return reply, nil
}
