// Package librarium reads, examines and writes rites of pattern Mark I kept
// in a Librarium's rites/ directory.
//
// A rite's scripture is JSONC. Every key is examined: unknown keys, malformed
// values, undeclared aspects, unknown placeholders and the other heresies of
// the codex are reported as Findings with line and column. Parse reads one
// scripture, Load the whole Librarium (where clashing wards and rites recorded
// twice are found), and Marshal writes a rite back as Mark I scripture.
//
// These values may vary per aspect (an AspectMap): the scripture of sanctums
// and transcriptions, a tether's anchor, an incantation's command, a litany's
// scroll, each of its offerings, every reversion, and an inscription's
// decrees. Everything else holds one value for all aspects.
//
// Placeholders are examined in every step string: vessels, tether names,
// anchors, commands, scrolls, offerings, reversions, inline scripture, tome
// paths and the auspex. Wards, glyphs, tongues, seals, patience, decrees and
// purposes are never illuminated.
package librarium

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// Pattern is the only pattern this package reads and writes.
const Pattern = "Mark I"

// DefaultAuspexPatience is how long an auspex may labour when its rite does
// not say otherwise.
const DefaultAuspexPatience = 2 * time.Second

// Rite is one rite of pattern Mark I.
type Rite struct {
	Name string // the rite's name: its scripture's file name without extension
	Path string // where the scripture was read from; "" when never read

	Purpose      string
	Aspects      []string
	Inscriptions []Inscription // in the order written
	Auspex       *Auspex       // nil when the rite has none
	Tongue       string        // "" when the rite names none
	Liturgy      []Step

	src    *source // the scripture as read, for locating its keys
	verses []int   // verse of each step of Liturgy in the scripture, when read
}

// Inscription is one inscription declared by a rite.
type Inscription struct {
	Key       string
	Purpose   string
	Mandatory bool
	Decrees   AspectMap[string] // per-aspect defaults; may leave aspects without
}

// Auspex is a rite's own command that reports the current aspect.
type Auspex struct {
	Rite     string        // the command
	Patience time.Duration // 0 when not written
}

// Wait returns the auspex's patience, or DefaultAuspexPatience.
func (a Auspex) Wait() time.Duration {
	if a.Patience > 0 {
		return a.Patience
	}
	return DefaultAuspexPatience
}

// Scripture is what a sanctum or transcription places into its vessel:
// inline text, a tome, or — for transcriptions only — null.
type Scripture struct {
	Text       string // inline scripture; lines are joined with "\n"
	Tome       string // path of the tome; "" for inline scripture
	Illuminate bool   // fill the tome's placeholders
	Null       bool   // the vessel is removed (transcriptions only)
}

// Inline is inline scripture.
func Inline(text string) Scripture { return Scripture{Text: text} }

// Tome is scripture drawn from the tome at path.
func Tome(path string, illuminate bool) Scripture {
	return Scripture{Tome: path, Illuminate: illuminate}
}

// IsTome reports whether the scripture is drawn from a tome.
func (s Scripture) IsTome() bool { return s.Tome != "" }

// HasAspect reports whether aspect is declared by the rite.
func (r *Rite) HasAspect(aspect string) bool {
	for _, a := range r.Aspects {
		if a == aspect {
			return true
		}
	}
	return false
}

// Inscription returns the declared inscription key.
func (r *Rite) Inscription(key string) (Inscription, bool) {
	for _, in := range r.Inscriptions {
		if in.Key == key {
			return in, true
		}
	}
	return Inscription{}, false
}

// InscriptionKeys returns the declared inscription keys in written order.
func (r *Rite) InscriptionKeys() []string {
	keys := make([]string, len(r.Inscriptions))
	for i, in := range r.Inscriptions {
		keys[i] = in.Key
	}
	return keys
}

// Scope is the placeholder scope of the rite's step strings.
func (r *Rite) Scope() placeholder.Scope {
	return placeholder.Scope{Mode: placeholder.RiteMode, Inscriptions: r.InscriptionKeys()}
}

// Dir is the directory the rite's scripture lies in; relative tomes and
// vessels resolve against it.
func (r *Rite) Dir() string { return filepath.Dir(r.Path) }

// ResolvePath expands a leading "~" and $VARS in p and resolves a relative
// result against the rite's directory.
func (r *Rite) ResolvePath(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			p = home + p[1:]
		}
	}
	p = os.ExpandEnv(p)
	if !filepath.IsAbs(p) {
		p = filepath.Join(r.Dir(), p)
	}
	return filepath.Clean(p)
}

// Locate returns where the value at the JSON pointer ptr (e.g.
// "/liturgy/2/anchor/on") is written in the scripture, falling back to the
// closest written parent. A rite never read from scripture locates nothing.
func (r *Rite) Locate(ptr string) Position {
	if r.src == nil {
		return Position{}
	}
	return r.src.locate(ptr)
}

// verse returns the verse (counted from one) of the step at index i of the
// liturgy, as written in the scripture; steps that could not be read do not
// shift the verses of the others.
func (r *Rite) verse(i int) int {
	if i < len(r.verses) {
		return r.verses[i]
	}
	return i + 1
}

// stepPointer is the JSON pointer of the step at index i of the liturgy.
func (r *Rite) stepPointer(i int) string { return fmt.Sprintf("/liturgy/%d", r.verse(i)-1) }

// Equal reports whether r and o hold the same rite, regardless of where and
// how their scriptures were read or written.
func (r *Rite) Equal(o *Rite) bool {
	if r == nil || o == nil {
		return r == o
	}
	a, b := *r, *o
	a.Path, a.src, a.verses = "", nil, nil
	b.Path, b.src, b.verses = "", nil, nil
	return reflect.DeepEqual(a, b)
}
