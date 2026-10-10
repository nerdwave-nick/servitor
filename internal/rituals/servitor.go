// Package rituals performs the servitor's rituals upon one Librarium, end
// to end, for every vessel of the servitor that speaks them: the command
// line and the cogitator alike.
//
// Invoke performs (or foresees) one invocation: it resolves the settings
// with the runes and the environment, resolves the inscriptions, reads the
// former aspect by augury, runs the pre-flight, performs the liturgy with
// the caller's herald, and keeps the data-slate and the chronicle. Augur,
// Census and Inquire read the standing of rites and examine them.
package rituals

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// Servitor performs rituals upon one Librarium.
type Servitor struct {
	Librarium *librarium.Librarium
	// Runes are the servitor's runes that sway the settings (--chronicle).
	Runes librarium.Runes
	// Env is the environment that sways the settings and places the
	// data-slates; nil is the servitor's own.
	Env librarium.Environment
	// Now tells the moment an invocation ends; nil is time.Now.
	Now func() time.Time
}

// Open loads the Librarium at dir.
func Open(dir string, runes librarium.Runes) *Servitor {
	return &Servitor{Librarium: librarium.Load(dir), Runes: runes}
}

func (s *Servitor) env() librarium.Environment {
	if s.Env == nil {
		return os.LookupEnv
	}
	return s.Env
}

func (s *Servitor) now() time.Time {
	if s.Now == nil {
		return time.Now()
	}
	return s.Now()
}

// Orders are the settings resolved with the runes and the environment.
func (s *Servitor) Orders() librarium.Orders {
	return s.Librarium.Settings.Resolve(s.Runes, s.env())
}

// Unrecorded is the lament for a rite the Librarium does not hold.
type Unrecorded struct {
	Name, Librarium string
}

func (e *Unrecorded) Error() string {
	return fmt.Sprintf("no rite named %q is recorded in the Librarium (%s)", e.Name, e.Librarium)
}

// Heretical is the lament for a rite whose scripture holds heresy.
type Heretical struct {
	Name     string
	Findings librarium.Findings // the heresies of its scripture
}

func (e *Heretical) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "the rite %q is tainted by heresy:", e.Name)
	for _, f := range e.Findings {
		b.WriteString("\n  " + f.String())
	}
	b.WriteString("\nSummon the Inquisition for the full verdict: servitor inquisition " + e.Name)
	return b.String()
}

// Rite returns the rite name, fit to be invoked; the error is *Unrecorded
// or *Heretical.
func (s *Servitor) Rite(name string) (*librarium.Rite, error) {
	lib := s.Librarium
	if r := lib.Rites[name]; r != nil {
		return r, nil
	}
	if _, ok := lib.Scriptures[name]; !ok {
		return nil, &Unrecorded{Name: name, Librarium: lib.Dir}
	}
	h := &Heretical{Name: name}
	for _, f := range lib.Findings {
		if f.Rite == name && f.Severity == librarium.Heresy {
			h.Findings = append(h.Findings, f)
		}
	}
	return nil, h
}
