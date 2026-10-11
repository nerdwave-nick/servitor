package cli

import (
	"encoding/json/v2"
	"errors"
	"io"

	"github.com/nerdwave-nick/servitor/internal/chronicle"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
)

// The death-marks of the ritual invoke, with or without --binharic. A
// triumph bears none (0).
const (
	markReverted = 1 // a step fell, and every deed was undone
	markRefused  = 2 // refused before anything was touched
	markFaltered = 3 // a step fell, and a reversion fell too
)

// refused marks err as an invocation refused before anything was touched.
func refused(err error) error {
	var ee *ExitError
	if errors.As(err, &ee) {
		return err
	}
	return &ExitError{Code: markRefused, Err: err}
}

// errForeseenInBinharic refuses --foresee spoken together with --binharic.
var errForeseenInBinharic = errors.New("the rune --foresee reveals its foresight to the faithful alone and " +
	"knows no binharic; speak it without --binharic")

// forbiddance is the binharic of an invocation the pre-flight refused: the
// rite, the aspect it was to enter (none for an edict) and every heresy.
type forbiddance struct {
	Rite      string   `json:"rite"`
	Aspect    string   `json:"aspect,omitempty"`
	Forbidden []string `json:"forbidden"`
}

// judged ends a performed invocation: in binharic it speaks the line the
// chronicle recorded upon w, and the verdict becomes the death-mark.
func judged(w io.Writer, r *librarium.Rite, res rituals.Result, binharic bool) error {
	if binharic && res.Entry != nil {
		if line, err := chronicle.Line(*res.Entry); err == nil {
			_, _ = w.Write(line) // the verdict stands, whatever became of the reader
		}
	}
	switch res.Outcome.Verdict {
	case invocation.Reverted:
		return &ExitError{Code: markReverted, Err: fallen(r, *res.Outcome)}
	case invocation.Faltered:
		return &ExitError{Code: markFaltered, Err: fallen(r, *res.Outcome)}
	}
	return nil
}

// forbid ends an invocation the pre-flight refused: in binharic it speaks
// its forbiddance upon w; the lament goes to stderr either way.
func forbid(w io.Writer, r *librarium.Rite, aspect string, hs invocation.Heresies, binharic bool) error {
	if binharic {
		f := forbiddance{Rite: r.Name, Aspect: aspect, Forbidden: make([]string, len(hs))}
		for i, h := range hs {
			f.Forbidden[i] = h.Error()
		}
		if line, err := json.Marshal(f, json.Deterministic(true)); err == nil {
			_, _ = w.Write(append(line, '\n'))
		}
	}
	return refused(forbidden(r, aspect, hs))
}
