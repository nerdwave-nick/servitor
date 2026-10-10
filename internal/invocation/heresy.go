package invocation

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// Heresy is one problem the pre-flight found; while any remains, nothing is
// performed.
type Heresy struct {
	Verse int // the step it was found in, counted from one; 0 for the invocation as a whole
	librarium.Finding
}

// Error renders the heresy as path:line:column: heresy: message.
func (h Heresy) Error() string { return h.String() }

// Heresies is every problem the pre-flight found, in the order of the liturgy.
type Heresies []Heresy

// Error joins the denunciations, one per line.
func (hs Heresies) Error() string {
	lines := make([]string, len(hs))
	for i, h := range hs {
		lines[i] = h.Error()
	}
	return strings.Join(lines, "\n")
}

// lament speaks the grimdark words for what the machine answered when it
// was asked to read or change place.
func lament(place string, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, fs.ErrPermission):
		return fmt.Errorf("the machine spirit denies the servitor access to %s", place)
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("nothing is to be found at %s", place)
	case errors.Is(err, errBondLoop), errors.Is(err, errUnrestorable):
		return fmt.Errorf("%s: %w", place, err)
	}
	return fmt.Errorf("the machine resists every attempt to touch %s", place)
}
