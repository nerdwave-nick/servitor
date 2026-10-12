package hallowed

import (
	"path/filepath"

	"golang.org/x/sys/unix"
)

// Bind binds the name to anchor atomically, binding it anew when a bond
// stands there already. The name itself is never followed. A scroll at the
// name gives way only in zeal, and is remembered whole; a hall never does.
// The anchor, relative to the hall of the name unless absolute, must lead
// through halls that pass the rule like the name's own.
func (g Ground) Bind(name, anchor string, zeal bool) (*Memory, error) {
	p, err := g.walk(name, itself)
	if err != nil {
		return nil, err
	}
	defer p.close()
	f, err := p.displaced(zeal)
	if err != nil {
		return nil, lament(p.path(), err)
	}
	if err := g.hallowed(p, anchor); err != nil {
		return nil, err
	}
	if f.form != bond || f.anchor != anchor {
		if err := lay(p, anchor, f.form == absent); err != nil {
			return nil, lament(p.path(), err)
		}
	}
	return p.remember(g, f)
}

// Unbind strikes the bond at name; an absent name stays absent. A scroll
// at the name gives way only in zeal, and is remembered whole; a hall never
// does.
func (g Ground) Unbind(name string, zeal bool) (*Memory, error) {
	p, err := g.walk(name, itself)
	if err != nil {
		return nil, err
	}
	defer p.close()
	f, err := p.displaced(zeal)
	if err == nil && f.form != absent {
		err = p.strike()
	}
	if err != nil {
		return nil, lament(p.path(), err)
	}
	return p.remember(g, f)
}

// displaced reads what stands at the name of a tether, which gives way to
// it: nothing, a bond, or in zeal a scroll.
func (p *place) displaced(zeal bool) (former, error) {
	switch {
	case p.stat == nil:
		return former{form: absent}, nil
	case p.stat.Mode&unix.S_IFMT == unix.S_IFLNK:
		anchor, err := readlink(p.hall, p.name)
		return former{form: bond, anchor: anchor}, err
	case p.stat.Mode&unix.S_IFMT == unix.S_IFDIR:
		return former{}, errHall
	case !zeal:
		return former{}, errNoBond
	}
	return p.former()
}

// hallowed walks the anchor of a bond at the place: every hall it leads
// through must pass the rule. What it finally leads to may be absent.
func (g Ground) hallowed(p *place, anchor string) error {
	if !filepath.IsAbs(anchor) {
		anchor = filepath.Join(p.hallPath(), anchor)
	}
	q, err := g.walk(anchor, vessel)
	if err != nil {
		return err
	}
	q.close()
	return nil
}

// lay lays a bond leading to anchor at the place atomically: a fresh bond
// beside it is renamed into place, without replacing anything when create.
// The act settles the hall afterwards.
func lay(p *place, anchor string, create bool) error {
	_, fresh, err := p.fresh(func(name string) (int, error) {
		return 0, unix.Symlinkat(anchor, p.hall, name)
	})
	if err != nil {
		return err
	}
	if err := unix.Renameat2(p.hall, fresh, p.hall, p.name, noReplace(create)); err != nil {
		_ = unix.Unlinkat(p.hall, fresh, 0)
	}
	return err
}
