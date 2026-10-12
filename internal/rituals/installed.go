package rituals

import (
	"fmt"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/intercessor"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// Intercession is the world in which the servitor seeks its intercessor and
// judges the servitor installed at /usr/bin/servitor; the trials raise one
// of their own, so that the machine's own /usr/bin never sways them.
var Intercession = intercessor.Machine()

// unenshrined notes, once, the installed servitor that is missing or does
// not stand where the root alone may write, while any of the rites named
// holds exalted verses: none of them could be invoked.
func (s *Servitor) unenshrined(wanted func(string) bool) librarium.Findings {
	var exalted []string
	for _, name := range s.Librarium.Names() {
		if r := s.Librarium.Rites[name]; r != nil && wanted(name) && r.Exalted() {
			exalted = append(exalted, fmt.Sprintf("%q", name))
		}
	}
	if len(exalted) == 0 {
		return nil
	}
	err := Intercession.Enshrined()
	if err == nil {
		return nil
	}
	held, them := "the rite "+exalted[0]+" holds", "it"
	if len(exalted) > 1 {
		held, them = "the rites "+strings.Join(exalted, ", ")+" hold", "them"
	}
	return librarium.Findings{{Severity: librarium.Impurity, Scripture: Intercession.Servitor,
		Position: librarium.Position{Line: 1, Column: 1},
		Message: fmt.Sprintf("%s exalted verses, yet %v; every invocation of %s is refused before any proof is "+
			"asked", held, err, them)}}
}
