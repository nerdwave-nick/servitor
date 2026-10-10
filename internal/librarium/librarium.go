package librarium

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// RitesDir is the directory of the Librarium holding the rites' scriptures.
const RitesDir = "rites"

// Extensions are the extensions a scripture may bear.
var Extensions = []string{".json", ".jsonc"}

// Librarium is every rite read from one Librarium.
type Librarium struct {
	Dir        string
	Rites      map[string]*Rite    // rites free of heresy, fit to be invoked
	Heretical  map[string]bool     // rites whose scripture holds heresy
	Scriptures map[string][]string // every scripture found, per rite name
	Findings   Findings
}

// Overlay substitutes one scripture, for examining a rite before it is
// written to the Librarium.
type Overlay struct {
	Name    string // the rite's name, as it would be written
	Data    []byte // its scripture
	Replace string // the rite it supersedes; "" for a new rite
}

// NewPath returns where the scripture of a new rite name is written.
func NewPath(dir, name string) string { return filepath.Join(dir, RitesDir, name+Extensions[0]) }

// Names returns the names of every rite, heretical or not, sorted.
func (l *Librarium) Names() []string {
	names := make([]string, 0, len(l.Scriptures))
	for n := range l.Scriptures {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Load reads and examines every scripture in dir's rites/.
func Load(dir string) *Librarium { return load(dir, nil) }

// Preview loads dir as if ov were written, without touching the Librarium.
func Preview(dir string, ov Overlay) *Librarium { return load(dir, &ov) }

// LoadFile reads and examines one scripture; the rite is named after it.
func LoadFile(path string) (*Rite, Findings) {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, Findings{readHeresy(path, name, err)}
	}
	return Parse(name, path, data)
}

func readHeresy(path, name string, err error) Finding {
	msg := "the scripture resists every attempt to read it; its letters stay hidden from the servitor"
	switch {
	case errors.Is(err, fs.ErrNotExist):
		msg = "no scripture is to be found at this place in the Librarium"
	case errors.Is(err, fs.ErrPermission):
		msg = "the machine spirit denies the servitor access to this scripture"
	}
	return Finding{Severity: Heresy, Scripture: path, Rite: name, Position: Position{1, 1}, Message: msg}
}

func load(dir string, ov *Overlay) *Librarium {
	lib := &Librarium{Dir: dir, Rites: map[string]*Rite{}, Heretical: map[string]bool{}}
	lib.Scriptures = lib.scan(ov)
	var read []*Rite
	for _, name := range lib.Names() {
		paths := lib.Scriptures[name]
		if len(paths) > 1 {
			lib.heretical(name, twice(name, paths)...)
			continue
		}
		var r *Rite
		var fs Findings
		if ov != nil && name == ov.Name {
			r, fs = Parse(name, paths[0], ov.Data)
		} else {
			r, fs = LoadFile(paths[0])
		}
		lib.Findings = append(lib.Findings, fs...)
		if fs.Heretical() {
			lib.Heretical[name] = true
		}
		if r != nil {
			read = append(read, r)
		}
	}
	for _, f := range wardClashes(read) {
		lib.heretical(f.Rite, f)
	}
	for _, r := range read {
		if !lib.Heretical[r.Name] {
			lib.Rites[r.Name] = r
		}
	}
	lib.Findings.Sort()
	return lib
}

func (l *Librarium) heretical(name string, fs ...Finding) {
	l.Heretical[name] = true
	l.Findings = append(l.Findings, fs...)
}

// scan finds every scripture of rites/, with the overlay in place.
func (l *Librarium) scan(ov *Overlay) map[string][]string {
	found := map[string][]string{}
	ritesDir := filepath.Join(l.Dir, RitesDir)
	entries, err := os.ReadDir(ritesDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		l.Findings = append(l.Findings, Finding{Severity: Heresy, Scripture: ritesDir, Position: Position{1, 1},
			Message: "the rites of the Librarium cannot be read; the machine spirit bars the way to them"})
	}
	for _, e := range entries {
		ext := filepath.Ext(e.Name())
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") || !slices.Contains(Extensions, ext) {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ext)
		found[name] = append(found[name], filepath.Join(ritesDir, e.Name()))
	}
	if ov != nil {
		if ov.Replace != "" {
			delete(found, ov.Replace)
		}
		p := NewPath(l.Dir, ov.Name)
		found[ov.Name] = append(slices.DeleteFunc(found[ov.Name], func(q string) bool { return q == p }), p)
	}
	return found
}

func twice(name string, paths []string) Findings {
	fs := make(Findings, len(paths))
	for i, p := range paths {
		fs[i] = Finding{Severity: Heresy, Scripture: p, Rite: name, Position: Position{1, 1},
			Message: fmt.Sprintf("the rite %q is recorded in more than one scripture (%s); a rite's name must "+
				"belong to a single scripture in %q — strike or rename all but one", name, strings.Join(paths, ", "), RitesDir+"/")}
	}
	return fs
}

// claim is one sanctum keeping a ward in a vessel.
type claim struct {
	rite  *Rite
	index int // in the rite's liturgy
}

func (c claim) verse() int { return c.rite.verse(c.index) }

// wardClashes denounces every sanctum whose ward another sanctum keeps in
// the same vessel, across all rites.
func wardClashes(rites []*Rite) Findings {
	seen := map[[2]string][]claim{}
	var keys [][2]string
	for _, r := range rites {
		for i, s := range r.Liturgy {
			if s, ok := s.(*Sanctum); ok {
				key := [2]string{r.ResolvePath(s.Vessel), s.WardFor(r.Name)}
				if seen[key] == nil {
					keys = append(keys, key)
				}
				seen[key] = append(seen[key], claim{r, i})
			}
		}
	}
	var fs Findings
	for _, key := range keys {
		claims := seen[key]
		if len(claims) < 2 {
			continue
		}
		for i, c := range claims {
			other := claims[0]
			if i == 0 {
				other = claims[1]
			}
			ptr := c.rite.stepPointer(c.index)
			at := ptr + "/ward"
			if c.rite.Liturgy[c.index].(*Sanctum).Ward == "" {
				at = ptr + "/sanctum"
			}
			fs = append(fs, Finding{Severity: Heresy, Scripture: c.rite.Path, Rite: c.rite.Name,
				Position: c.rite.Locate(at), Message: fmt.Sprintf("the ward %q in the vessel %s is claimed "+
					"twice: by verse %d of the rite %q (%s) and by verse %d of the rite %q (%s); two sanctums "+
					"sharing one ward in one vessel would overwrite each other's scripture — give each its own "+
					"\"ward\"", key[1], key[0], c.verse(), c.rite.Name, c.rite.Path, other.verse(), other.rite.Name,
					other.rite.Path)})
		}
	}
	return fs
}
