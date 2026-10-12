// Package exalted is the exalted servitor: the hidden mode in which the
// root-owned servitor, roused through an intercessor with the machine's own
// authority, performs the exalted verses of one invocation from the
// sanction of their master, while the invoker's servitor drives it.
//
// Before it hears a word it judges the scroll it runs from (/proc/self/exe)
// as package hallowed judges the installed servitor. It hears its invoker
// over one channel on fd 0 and 1, a socketpair (or pipes, when sudo records
// what passes), in strict JSON lines (Channel). The first line is the
// Handshake, checked against argv and the channel's credentials; the
// sanction is then read from the root's store, the root-side dominions are
// seized, and the session is accepted. Every answer to the handshake names
// the protocol, which the invoker's servitor hears first (Channel.Answer).
// Thereafter only Requests are heard: perform the next exalted verse,
// revert the last performed, finish, abort. The end of the channel before
// finish, ten minutes without a word, or SIGTERM revert every verse
// performed, in reverse. Vessels and tethers are acted upon through package
// hallowed; exalted speech is lent by a Voice. Every act is journaled as
// the root's.
package exalted

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nerdwave-nick/servitor/internal/hallowed"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// Mode is the hidden word that rouses the exalted servitor:
// servitor __exalt <rite> <sigil12> [<aspect>]. It is never listed.
const Mode = "__exalt"

// The death-marks of the exalted servitor, as an invocation's.
const (
	markFinished = 0 // finished: what was performed is kept
	markReverted = 1 // ended before finish, every verse performed reverted
	markRefused  = 2 // refused before anything was touched
	markFaltered = 3 // ended before finish, and a reversion fell
)

// Config is the world of an exalted servitor. Machine is the machine's
// own; the trials raise worlds of their own in temporary halls.
type Config struct {
	// Exalted tells whether the servitor bears the machine's own authority.
	Exalted func() bool
	// Getenv reads the environment the intercessor left.
	Getenv func(key string) (string, bool)
	// Store is the root of the store of sanctions; Sovereign reports
	// whether a uid is the root's, whom the store must belong to.
	Store     string
	Sovereign func(uid uint32) bool
	// Ground is where vessels and tethers are acted upon.
	Ground hallowed.Ground
	// Self names the scroll the servitor runs from, as /proc/self/exe does;
	// it must stand enshrined in Ground.
	Self string
	// Dominions is where the root-side places of dominion lie.
	Dominions string
	// Journal keeps every act.
	Journal Journal
	// Idle is how long the servitor awaits a word before it reverts and
	// ends.
	Idle time.Duration
	// Halt, when it closes, aborts the session (SIGTERM).
	Halt <-chan struct{}
	// Voice lends the session exalted speech; nil refuses every plan that
	// holds an incantation or a litany.
	Voice Voice
	// Lament hears the words of a refusal that reaches no invoker.
	Lament io.Writer
}

// DominionRoot is where the root-side places of dominion lie.
const DominionRoot = "/run/servitor/dominions"

// Machine is the world of the machine itself.
func Machine() Config {
	return Config{
		Exalted:   func() bool { return os.Geteuid() == 0 },
		Getenv:    os.LookupEnv,
		Store:     sanction.DefaultRoot,
		Sovereign: func(uid uint32) bool { return uid == 0 },
		Ground:    hallowed.Machine,
		Self:      "/proc/self/exe",
		Dominions: DominionRoot,
		Idle:      10 * time.Minute,
		Voice:     MachineSpeech().Voice,
		Lament:    os.Stderr,
	}
}

// Rouse is the exalted servitor roused by argv (after Mode) upon the
// machine itself, its channel on fd 0 and 1; it returns the death-mark. It
// bears the machine's authority by design, so it refuses to begin unless it
// truly does: as anyone else it is never more than a stranger's servitor.
func Rouse(args []string, stderr io.Writer) int {
	if os.Geteuid() != 0 {
		fmt.Fprintln(stderr, "servitor ✠ the exalted servitor is roused only with the machine's own authority, "+
			"by an intercessor, and never by hand; nothing was touched")
		return markRefused
	}
	if err := harden(); err != nil {
		fmt.Fprintln(stderr, "servitor ✠ the exalted servitor cannot make its process its own: "+err.Error())
		return markRefused
	}
	journal, err := SystemJournal()
	if err != nil {
		fmt.Fprintln(stderr, "servitor ✠ "+err.Error())
		return markRefused
	}
	halt := make(chan struct{})
	terms := make(chan os.Signal, 1)
	signal.Notify(terms, syscall.SIGTERM)
	go func() { <-terms; close(halt) }()
	cfg := Machine()
	cfg.Journal, cfg.Halt, cfg.Lament = journal, halt, stderr
	return Serve(args, os.Stdin, os.Stdout, cfg)
}
