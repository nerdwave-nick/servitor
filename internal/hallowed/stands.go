package hallowed

import (
	"bytes"
	"io/fs"

	"golang.org/x/sys/unix"
)

// Stands reports whether the vessel at path, its bonds followed, already
// stands as Transcribe would leave it: holding scripture, under seal when
// one is given. An absent vessel never does. Nothing is touched.
func (g Ground) Stands(path string, scripture []byte, seal *fs.FileMode) (bool, error) {
	p, err := g.walk(path, vessel)
	if err != nil {
		return false, err
	}
	defer p.close()
	f, err := p.former()
	if err != nil {
		return false, lament(p.path(), err)
	}
	return f.form == scroll && (seal == nil || unixSeal(*seal) == f.seal) && bytes.Equal(f.scripture, scripture), nil
}

// Bonded tells what stands at name, never following it: where the bond
// there leads, as written (bond), or that nothing stands there (absent);
// neither for a scroll or a hall.
func (g Ground) Bonded(name string) (anchor string, bond, absent bool, err error) {
	p, err := g.walk(name, itself)
	if err != nil {
		return "", false, false, err
	}
	defer p.close()
	switch {
	case p.stat == nil:
		return "", false, true, nil
	case p.stat.Mode&unix.S_IFMT == unix.S_IFLNK:
		anchor, err := readlink(p.hall, p.name)
		return anchor, err == nil, false, lament(p.path(), err)
	}
	return "", false, false, nil
}
