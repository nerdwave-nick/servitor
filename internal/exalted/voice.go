package exalted

import (
	"context"

	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// Voice lends exalted speech to an accepted session whose plan holds an
// incantation or a litany: it raises the session's Speaker, or refuses the
// session before anything is touched.
type Voice func(Sitting) (Speaker, error)

// Sitting is an accepted session, as exalted speech knows it.
type Sitting struct {
	UID      int
	Sanction *sanction.Sanction
	Plan     sanction.Plan // the plan of the aspect performed
	Sigil    string        // the whole sigil of the plan
	// Former and Inscriptions are the words given at invocation, which
	// speech hears from its environment alone.
	Former       string
	Inscriptions map[string]string
}

// Speaker performs the exalted speech of one session.
type Speaker interface {
	// Speak performs the incantation or litany v, returning the words it
	// uttered and how it is reverted (nil when it reverts nothing).
	Speak(ctx context.Context, v sanction.Verse) (heard string, undo Undo, err error)
	// Hear speaks words of the privileged auspex of the machine-spirit v,
	// hearing omen as SERVITOR_OMEN when it is not "" (a trial): printed
	// is the first of what they printed on their standard output, spoke
	// that they ended with the death-mark 0, and err that they were
	// silent — they outlasted the patience of the auspex, were slain, or
	// could not be awakened.
	Hear(ctx context.Context, v sanction.Verse, words, omen string) (printed string, spoke bool, err error)
	// Close ends the speech of the session, however it ends.
	Close() error
}

// Undo reverts one verse of exalted speech, returning the words its
// reversion uttered.
type Undo func(ctx context.Context) (heard string, err error)
