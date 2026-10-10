package invocation

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// form is what stands at a place in the machine.
type form int

const (
	absent   form = iota
	scroll        // a regular vessel holding scripture
	bond          // a symbolic link
	hall          // a directory
	stranger      // anything else
)

// entry is what stands at one place, as far as the steps care.
type entry struct {
	form    form
	content string      // scrolls only
	mode    fs.FileMode // scrolls only
	anchor  string      // bonds only: where the link leads, as written
}

// world is the machine as the steps see it: the disk itself while
// performing, or the disk overlaid with the steps already planned during the
// pre-flight.
type world interface {
	entry(path string) (entry, error)
}

// disk reads the machine itself.
type disk struct{}

func (disk) entry(path string) (entry, error) {
	info, err := os.Lstat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return entry{form: absent}, nil
	case err != nil:
		return entry{}, err
	}
	switch mode := info.Mode(); {
	case mode&fs.ModeSymlink != 0:
		anchor, err := os.Readlink(path)
		return entry{form: bond, anchor: anchor}, err
	case mode.IsDir():
		return entry{form: hall}, nil
	case mode.IsRegular():
		data, err := os.ReadFile(path)
		return entry{form: scroll, content: string(data), mode: mode.Perm()}, err
	}
	return entry{form: stranger}, nil
}

// shadow is the disk overlaid with what the planned steps will have done.
type shadow struct {
	planned map[string]entry
}

func newShadow() *shadow { return &shadow{planned: map[string]entry{}} }

func (s *shadow) entry(path string) (entry, error) {
	if e, ok := s.planned[filepath.Clean(path)]; ok {
		return e, nil
	}
	return disk{}.entry(path)
}

func (s *shadow) set(path string, e entry) { s.planned[filepath.Clean(path)] = e }

// maxBonds is how many symbolic links are followed before giving up.
const maxBonds = 40

// errBondLoop is returned when symbolic links lead in circles.
var errBondLoop = errors.New("the bonds of this vessel lead in endless circles")

// follow walks the symbolic links at path to the place they finally lead
// to, which may be absent, and returns that place and what stands there.
func follow(w world, path string) (string, entry, error) {
	for range maxBonds {
		e, err := w.entry(path)
		if err != nil || e.form != bond {
			return path, e, err
		}
		next := e.anchor
		if !filepath.IsAbs(next) {
			next = filepath.Join(filepath.Dir(path), next)
		}
		path = filepath.Clean(next)
	}
	return path, entry{}, errBondLoop
}

// hallExists reports whether the directory that would hold path stands.
func hallExists(w world, path string) (bool, error) {
	_, e, err := follow(w, filepath.Dir(path))
	return e.form == hall, err
}

// writeScroll writes content to path atomically with the given mode.
func writeScroll(path, content string, mode fs.FileMode) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".servitor-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.Remove(tmp.Name())
		}
	}()
	_, err = tmp.WriteString(content)
	if err == nil {
		err = tmp.Chmod(mode)
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// bind points the symbolic link name at anchor atomically, replacing a link
// or scroll that stood there.
func bind(name, anchor string) error {
	tmp, err := os.CreateTemp(filepath.Dir(name), "."+filepath.Base(name)+".servitor-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	_ = tmp.Close()
	if err := os.Remove(tmpName); err != nil {
		return err
	}
	if err := os.Symlink(anchor, tmpName); err != nil {
		return err
	}
	if err := os.Rename(tmpName, name); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}

// strike removes what stands at path; an absent path is no failure.
func strike(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// restore puts back what stood at path before a step changed it.
func restore(path string, e entry) error {
	switch e.form {
	case absent:
		return strike(path)
	case scroll:
		return writeScroll(path, e.content, e.mode)
	case bond:
		return bind(path, e.anchor)
	}
	return errUnrestorable
}

var errUnrestorable = errors.New("what stood here cannot be raised again by the servitor")
