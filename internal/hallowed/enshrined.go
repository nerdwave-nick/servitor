package hallowed

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	errUnsovereignScroll doom = "is a scroll the root does not own"
	errOpenScroll        doom = "is a scroll that others than its owner may write"
	errNotSelf           doom = "is not the scroll the servitor runs from"
)

// Enshrined judges the scroll at path, following its bonds, as the
// installed servitor must stand to be roused with the machine's own
// authority: every hall above it passes the rule, and the scroll itself is
// the root's, written by none but it. The invoker's servitor judges
// /usr/bin/servitor so before it calls upon an intercessor; a servitor
// nowhere to be found is fs.ErrNotExist.
func (g Ground) Enshrined(path string) error {
	p, err := g.walk(path, vessel)
	if err != nil {
		return err
	}
	defer p.close()
	if p.stat == nil {
		return lament(p.path(), errMissing)
	}
	return lament(p.path(), g.enshrined(p.stat))
}

// EnshrinedSelf judges the scroll the servitor truly runs from, which self
// names as /proc/self/exe does, as Enshrined judges the installed servitor:
// the name the machine shows for it must lie within the ground, and the
// scroll standing at that name must be the very scroll self is, not one
// laid there since. The exalted servitor judges itself so before it hears
// a word.
func (g Ground) EnshrinedSelf(self string) error {
	target, err := os.Readlink(self)
	if err != nil {
		return lament(self, err)
	}
	var st unix.Stat_t
	if err := unix.Stat(self, &st); err != nil {
		return lament(self, err)
	}
	path, ok := g.within(target)
	if !ok {
		return lament(target, errAstray)
	}
	p, err := g.walk(path, vessel)
	if err != nil {
		return err
	}
	defer p.close()
	switch {
	case p.stat == nil:
		return lament(p.path(), errMissing)
	case p.stat.Dev != st.Dev || p.stat.Ino != st.Ino:
		return lament(p.path(), errNotSelf)
	}
	return lament(p.path(), g.enshrined(p.stat))
}

// within is the path of the ground that names target, a path of the
// machine itself; ok is false when target lies outside the ground.
func (g Ground) within(target string) (path string, ok bool) {
	if !filepath.IsAbs(target) {
		return "", false
	}
	rel, err := filepath.Rel(g.Root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return "/" + rel, true
}

// enshrined judges the stat of the installed servitor itself: a scroll the
// root owns, which none but its owner may write.
func (g Ground) enshrined(st *unix.Stat_t) error {
	switch {
	case st.Mode&unix.S_IFMT != unix.S_IFREG:
		return errNoScroll
	case g.Sovereign == nil || !g.Sovereign(st.Uid):
		return errUnsovereignScroll
	case st.Mode&0o022 != 0:
		return errOpenScroll
	}
	return nil
}
