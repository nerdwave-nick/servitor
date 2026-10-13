package augury

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// SlatesDir is the directory of the servitor's state directory holding the
// data-slates.
const SlatesDir = "data-slates"

// Slate is a rite's data-slate: what the servitor last did with it. It is
// written by the invoking servitor after each invocation, never by a step.
type Slate struct {
	// Edict marks the slate of an edict, which is written without aspect.
	// It is never read back: the rite tells whether it is an edict.
	Edict bool `json:"-"`
	// Aspect is the aspect the last invocation left the rite in: the aspect
	// invoked after triumph, the former aspect after a failure; "" when not
	// known, and for an edict.
	Aspect string `json:"aspect"`
	// Inscriptions are those of that aspect.
	Inscriptions map[string]string `json:"inscriptions"`
	// LastRite is how the last invocation ended.
	LastRite *LastRite `json:"last_rite"`
	// Boot is the boot of the machine the last invocation of a state rite
	// of pattern Mark II was performed in (BootPath); "" when it could not
	// be read, and for every other rite.
	Boot string `json:"boot,omitempty"`
	// Omens are the omens the last invocation of a state rite of pattern
	// Mark II knew at its end, by subject; nil for every other rite.
	Omens map[string]Recorded `json:"omens,omitzero"`
}

// Recorded is an omen recorded on the data-slate: what one subject was
// last read to show, and when.
type Recorded struct {
	Omen string    `json:"omen"` // a declared omen of the subject
	At   time.Time `json:"at"`   // in UTC, to the second
	// Exalted marks an omen the exalted servitor read, the invoker being
	// unable to.
	Exalted bool `json:"exalted,omitempty"`
}

// BootPath names the boot of the machine: it changes with every boot.
const BootPath = "/proc/sys/kernel/random/boot_id"

// Boot is the boot of the machine; "" when it cannot be read.
func Boot() string {
	b, err := os.ReadFile(BootPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// MarshalJSON writes the slate as the codex shows it; an edict's bears no
// "aspect" key.
func (s Slate) MarshalJSON() ([]byte, error) {
	if !s.Edict {
		type plain Slate
		return json.Marshal(plain(s))
	}
	return json.Marshal(struct {
		Inscriptions map[string]string `json:"inscriptions"`
		LastRite     *LastRite         `json:"last_rite"`
	}{s.Inscriptions, s.LastRite})
}

// LastRite is how an invocation ended, and when.
type LastRite struct {
	Verdict invocation.Verdict `json:"verdict"`
	At      time.Time          `json:"at"` // in UTC, to the second
}

// SlatePath is where the data-slate of the rite named rite is kept:
// $XDG_STATE_HOME/servitor/data-slates/<rite>.json.
func SlatePath(env librarium.Environment, rite string) string {
	return filepath.Join(librarium.StateDir(env), SlatesDir, rite+".json")
}

// ReadSlate reads the data-slate of the rite named rite; it is nil, without
// error, when the rite was never invoked.
func ReadSlate(env librarium.Environment, rite string) (*Slate, error) {
	p := SlatePath(env, rite)
	data, err := os.ReadFile(p)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("the data-slate %s resists every attempt to read it", p)
	}
	var s Slate
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("the data-slate %s is garbled beyond the servitor's reading; strike it, and the "+
			"next invocation will inscribe it anew", p)
	}
	return &s, nil
}

// WriteSlate writes the data-slate of the rite named rite atomically,
// raising the halls of the state directory when they are absent.
func WriteSlate(env librarium.Environment, rite string, s Slate) (err error) {
	p := SlatePath(env, rite)
	if s.Inscriptions == nil {
		s.Inscriptions = map[string]string{}
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("the hall of the data-slates cannot be raised at %s", filepath.Dir(p))
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), "."+filepath.Base(p)+".servitor-*")
	if err != nil {
		return fmt.Errorf("the data-slate %s cannot be inscribed", p)
	}
	defer func() {
		if err != nil {
			_ = os.Remove(tmp.Name())
		}
	}()
	_, err = tmp.Write(append(data, '\n'))
	if err == nil {
		err = tmp.Chmod(0o644)
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), p)
	}
	if err != nil {
		return fmt.Errorf("the data-slate %s cannot be inscribed", p)
	}
	return nil
}

// SlateOf is the data-slate an invocation performed with opts leaves: after
// triumph the aspect invoked and its inscriptions; after a failure the
// former aspect and the inscriptions recorded before, which the reversion
// meant to restore. An edict's slate keeps its inscriptions alike, but no
// aspect. The verdict is kept with the moment at, in UTC to the
// second. A state rite of pattern Mark II records the omens its subjects
// were known to show at the end (invocation.Outcome.Known), whatever the
// verdict, each with the moment it was read; every other is struck.
func SlateOf(out invocation.Outcome, opts invocation.Options, at time.Time) Slate {
	s := Slate{Edict: opts.Edict, Aspect: opts.Former, Inscriptions: maps.Clone(opts.Recorded),
		LastRite: &LastRite{Verdict: out.Verdict, At: at.UTC().Truncate(time.Second)}}
	if out.Verdict == invocation.Triumph {
		s.Aspect, s.Inscriptions = opts.Aspect, maps.Clone(opts.Inscriptions)
	}
	if s.Inscriptions == nil {
		s.Inscriptions = map[string]string{}
	}
	if out.Known != nil {
		s.Omens = map[string]Recorded{}
		for name, read := range out.Known {
			when := read.At
			if when.IsZero() {
				when = at // a reading untold of its moment is as old as the verdict
			}
			s.Omens[name] = Recorded{Omen: read.Omen, At: when.UTC().Truncate(time.Second), Exalted: read.Exalted}
		}
	}
	return s
}

// Record writes the data-slate an invocation of the rite named rite,
// performed with opts and ended in out, leaves at the moment at; that of a
// state rite of pattern Mark II also records the omens known at its end,
// and the boot they were read in.
func Record(env librarium.Environment, rite string, out invocation.Outcome, opts invocation.Options, at time.Time) error {
	s := SlateOf(out, opts, at)
	if s.Omens != nil {
		s.Boot = Boot()
	}
	return WriteSlate(env, rite, s)
}
