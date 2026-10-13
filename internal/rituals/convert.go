package rituals

import (
	"fmt"

	"github.com/nerdwave-nick/servitor/internal/dominion"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// Converted tells what the servitor wrote of the conversions of its
// Librarium.
type Converted struct {
	Tidings []string // one line for every scripture converted on disk
	Laments []error  // why a scripture stays of pattern Mark I on disk
}

// Convert writes every conversion of the Librarium (Open reads it
// converted in memory): the
// settings, then each rite under its dominion. A rite whose dominion
// another holds is not awaited: it stays converted in memory alone, to be
// written at a later reading. Each conversion written is told once, and
// forgotten; one that cannot be written is lamented, and its scripture
// stays as it was.
//
// Only a ritual that may write calls it: never completion, foresight, a
// servitor bearing the machine's own authority, nor the exalted servitor.
func (s *Servitor) Convert() Converted {
	var told Converted
	lib := s.Librarium
	var pending []*librarium.Conversion
	for _, c := range lib.Conversions {
		held, err := s.holdForConversion(c)
		if err != nil {
			told.Laments = append(told.Laments, err)
			continue
		}
		if held == nil {
			pending = append(pending, c)
			continue
		}
		err = c.Write()
		held.Relinquish()
		if err != nil {
			told.Laments = append(told.Laments, err)
			continue
		}
		told.Tidings = append(told.Tidings, c.Tidings())
	}
	lib.Conversions = pending
	return told
}

// holdForConversion seizes the dominion over the rite c converts, if it is
// free; nil when another holds it. The settings need none.
func (s *Servitor) holdForConversion(c *librarium.Conversion) (*dominion.Dominion, error) {
	if c.Rite == "" {
		return &dominion.Dominion{}, nil
	}
	held, _, err := dominion.Try([]string{dominion.Rite(s.env(), c.Rite)})
	if err != nil {
		return nil, fmt.Errorf("the scripture %s stays of pattern %q, for the dominion over the rite %q cannot be "+
			"held: %v", c.Path, librarium.Pattern, c.Rite, err)
	}
	return held, nil
}
