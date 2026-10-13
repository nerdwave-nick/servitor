// Package librarium reads, examines and writes rites of pattern Mark II
// kept in a Librarium's rites/ directory, and reads the settings kept beside
// them in servitor.json (LoadSettings; Settings.Resolve applies runes, the
// environment and the defaults). A rite of aspects declares its Subjects and
// Observances; an edict keeps a Liturgy of incantations and litanies in the
// designation form, every value of it written once.
//
// A rite's scripture is JSONC. Every key is examined: unknown keys, malformed
// values, undeclared aspects, unknown placeholders and the other heresies of
// the codex are reported as Findings with line and column. Parse reads one
// scripture, Load the whole Librarium (where clashing wards and rites recorded
// twice are found), and Marshal writes a rite back as scripture.
//
// Scripture of pattern Mark I is read only to be converted (convert.go):
// every reading converts each pure scripture of that pattern in memory to
// the rite of pattern Mark II that does what it did, and each Conversion is
// written only when its Write is called. Heretical scripture of pattern
// Mark I is reported, and converted once its heresies are purged. Its model
// — a liturgy of steps whose values may vary per aspect, an auspex of the
// rite, vox-cast steps — lives on in this package for the conversion alone.
//
// A rite either declares its aspects or is an edict ("edict": true): a rite
// without aspects, proclaimed whole. An edict writes every value plainly (no
// aspect map, not even one keyed by "*" alone) and never speaks {{aspect}}
// or {{former}}.
//
// Placeholders are examined in every string that is spoken or written:
// vessels, bound names, omens, enact words, trials, scans, words, scrolls,
// offerings, reversions and tome paths. Wards, glyphs, tongues, seals,
// patience, decrees and purposes are never illuminated.
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

// PatternMarkI is the elder pattern of a liturgy of steps, read only to be
// converted to Pattern.
const PatternMarkI = "Mark I"

// DefaultAuspexPatience is how long an auspex may labour when its rite does
// not say otherwise.
const DefaultAuspexPatience = 2 * time.Second

// Rite is one rite of pattern Mark II: a rite of aspects holding Subjects
// and Observances, or an edict holding a Liturgy of incantations and
// litanies, every value of it written once. Both bear their vox-casts as
// VoxCasts.
type Rite struct {
	Name string // the rite's name: its scripture's file name without extension
	Path string // where the scripture was read from; "" when never read

	// Schema is the "$schema" the scripture bears for the faithful's
	// editors. The servitor heeds it not, but keeps it when writing.
	Schema string

	Purpose string
	Aspects []string // nil for an edict
	// Edict marks a rite without aspects ("edict": true): it is proclaimed
	// whole and never brought into an aspect.
	Edict        bool
	Inscriptions []Inscription // in the order written
	Tongue       string        // "" when the rite names none
	Liturgy      []Step        // an edict's alone
	Subjects     []Subject     // in written order
	Observances  []Observance  // in written order
	VoxCasts     []string      // VoxProgress and VoxSuccess; nil when unwritten

	// markI marks scripture of pattern Mark I as read, before conversion:
	// a liturgy of steps that may vary per aspect, vox-casts among them, and
	// auspex the rite's own (nil when it has none). Only conversion reads
	// such a rite; Parse and Load never return one.
	markI  bool
	auspex *riteAuspex

	src    *source // the scripture as read, for locating its keys
	verses []int   // verse of each step of Liturgy in the scripture, when read
	// subjectAt is the index of each of Subjects in the scripture, when read.
	subjectAt []int
}

// Inscription is one inscription declared by a rite.
type Inscription struct {
	Key       string
	Purpose   string
	Mandatory bool
	Decrees   AspectMap[string] // per-aspect defaults; may leave aspects without
}

// riteAuspex is the own command of a rite of pattern Mark I that reports
// the aspect of the whole rite; conversion makes it the scan of a
// machine-spirit, or strikes it.
type riteAuspex struct {
	Rite     string        // the command
	Patience time.Duration // 0 when not written
}

// Wait returns the auspex's patience, or DefaultAuspexPatience.
func (a riteAuspex) Wait() time.Duration {
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
	return placeholder.Scope{Mode: placeholder.RiteMode, Inscriptions: r.InscriptionKeys(), Edict: r.Edict}
}

// Dir is the directory the rite's scripture lies in; relative tomes and
// vessels resolve against it.
func (r *Rite) Dir() string { return filepath.Dir(r.Path) }

// ResolvePath expands a leading "~" and $VARS in p and resolves a relative
// result against the rite's directory.
func (r *Rite) ResolvePath(p string) string {
	p = os.ExpandEnv(ExpandHome(p))
	if !filepath.IsAbs(p) {
		p = filepath.Join(r.Dir(), p)
	}
	return filepath.Clean(p)
}

// ExpandHome replaces a leading "~/", or a "~" standing alone, with the home
// of the faithful; every other "~" is left as written.
func ExpandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return home + p[1:]
		}
	}
	return p
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

// subjectPointer is the JSON pointer of the subject at index i of
// Subjects.
func (r *Rite) subjectPointer(i int) string {
	if i < len(r.subjectAt) {
		return fmt.Sprintf("/subjects/%d", r.subjectAt[i])
	}
	return fmt.Sprintf("/subjects/%d", i)
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
	a.Path, a.src, a.verses, a.subjectAt = "", nil, nil, nil
	b.Path, b.src, b.verses, b.subjectAt = "", nil, nil, nil
	return reflect.DeepEqual(a, b)
}
