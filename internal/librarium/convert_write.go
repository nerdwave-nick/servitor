package librarium

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tailscale/hujson"
)

// MarkIDir is the directory, beside a scripture converted to pattern Mark
// II, in which its scripture of pattern Mark I is kept. The Librarium
// never reads it: it passes over directories and dot-files.
const MarkIDir = ".mark-i"

// Conversion is a scripture of pattern Mark I converted to pattern Mark II,
// once, on its first reading: the scripture as read, what it becomes, and
// what changed in meaning on the way, which comments in the scripture tell.
type Conversion struct {
	Rite     string // the rite converted; "" for the settings
	Path     string // where the scripture was read
	Original []byte // the scripture of pattern Mark I, as read
	Data     []byte // the scripture of pattern Mark II it becomes
	// Changes name what was struck or changed in meaning, in the order of
	// the scripture, such as "the auspex of the rite is struck".
	Changes []string
	// Exalted tells that the rite held exalted verses: their sanction
	// lapses with the conversion, for the unit sanctioned changes.
	Exalted bool
}

// Copy is where the scripture of pattern Mark I is kept once converted.
func (c *Conversion) Copy() string {
	return filepath.Join(filepath.Dir(c.Path), MarkIDir, filepath.Base(c.Path))
}

// Tidings is the one line that tells the conversion: the rite, its copy,
// what changed in meaning, and the sanction a rite of exalted verses must
// be given anew.
func (c *Conversion) Tidings() string {
	if c.Rite == "" {
		return fmt.Sprintf("the settings are converted to pattern %q, only their \"pattern\" rewritten; their "+
			"scripture of pattern %q is kept at %s", PatternMarkII, Pattern, c.Copy())
	}
	line := fmt.Sprintf("the rite %q is converted to pattern %q; its scripture of pattern %q is kept at %s",
		c.Rite, PatternMarkII, Pattern, c.Copy())
	if len(c.Changes) > 0 {
		line += "; changed in meaning, as the comments of its scripture tell: " + strings.Join(c.Changes, ", ")
	}
	if c.Exalted {
		line += fmt.Sprintf("; the sanction of its exalted verses lapses with its pattern — sanction it anew: "+
			"servitor sanction %s", c.Rite)
	}
	return line
}

// convertSettings converts the settings s, read from data with the
// findings found, when they are of pattern Mark I: only their "pattern" is
// rewritten, so that their comments survive. Heretical settings are not
// converted, and one more heresy tells that they are once purged.
func convertSettings(s *Settings, found Findings, data []byte) (*Settings, Findings, *Conversion) {
	start, end, ok := elderPattern(data)
	if !ok {
		return s, found, nil
	}
	if found.Heretical() {
		src := &source{path: s.Path, data: data}
		return s, append(found, Finding{Severity: Heresy, Scripture: s.Path, Position: src.position(start),
			Message: fmt.Sprintf("the settings of pattern %q are not converted to pattern %q, which this servitor "+
				"reads in their stead; they are converted once their heresies are purged", Pattern, PatternMarkII)}), nil
	}
	s.MarkII = true
	converted := slices.Concat(data[:start], []byte(fmt.Sprintf("%q", PatternMarkII)), data[end:])
	return s, found, &Conversion{Path: s.Path, Original: data, Data: converted}
}

// elderPattern finds where the "pattern" of the scripture data is written
// when it is pattern Mark I.
func elderPattern(data []byte) (start, end int, ok bool) {
	v, err := hujson.Parse(data)
	if err != nil {
		return 0, 0, false
	}
	obj, isObj := v.Value.(*hujson.Object)
	if !isObj {
		return 0, 0, false
	}
	for _, m := range obj.Members {
		lit, isLit := m.Value.Value.(hujson.Literal)
		if m.Name.Value.(hujson.Literal).String() == "pattern" && isLit && lit.Kind() == '"' && lit.String() == Pattern {
			return m.Value.StartOffset, m.Value.EndOffset, true
		}
	}
	return 0, 0, false
}

// Write keeps the scripture of pattern Mark I in the directory MarkIDir
// beside it (Copy), never overwriting a copy that stands there and differs,
// then writes the converted scripture in its place, atomically and with its
// mode kept. A scripture tethered into the Librarium is written where it
// lies. A scripture changed since it was read is not written over. On any
// failure the scripture stays as it was.
func (c *Conversion) Write() error {
	path, err := filepath.EvalSymlinks(c.Path)
	if err != nil {
		return c.unwritten(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		return c.unwritten(err)
	}
	if now, err := os.ReadFile(path); err != nil || !bytes.Equal(now, c.Original) {
		return fmt.Errorf("the scripture %s was changed since the servitor read it; it is not written over, and is "+
			"converted at its next reading", c.Path)
	}
	if err := c.keep(fi.Mode().Perm()); err != nil {
		return err
	}
	return c.unwritten(replace(path, c.Data, fi.Mode().Perm()))
}

// unwritten tells why the conversion could not be written; nil for nil.
func (c *Conversion) unwritten(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("the scripture %s stays of pattern %q, for the servitor cannot write its conversion: %v",
		c.Path, Pattern, err)
}

// keep writes the original to its copy, which must not stand already
// unless it holds the original.
func (c *Conversion) keep(perm fs.FileMode) error {
	at := c.Copy()
	if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
		return c.unwritten(err)
	}
	f, err := os.OpenFile(at, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if errors.Is(err, fs.ErrExist) {
		if kept, err := os.ReadFile(at); err == nil && bytes.Equal(kept, c.Original) {
			return nil
		}
		return fmt.Errorf("the scripture %s stays of pattern %q: a copy already stands at %s and differs from it, "+
			"and a copy is never overwritten — move that copy aside, and the scripture is converted at its next "+
			"reading", c.Path, Pattern, at)
	}
	if err != nil {
		return c.unwritten(err)
	}
	_, err = f.Write(c.Original)
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(at)
		return c.unwritten(err)
	}
	return nil
}

// replace writes data in place of path at once: into a hidden scripture
// beside it, which then takes its place.
func replace(path string, data []byte, perm fs.FileMode) error {
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Chmod(perm)
	}
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(f.Name(), path)
	}
	if err != nil {
		_ = os.Remove(f.Name())
	}
	return err
}
