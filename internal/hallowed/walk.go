package hallowed

import (
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// end is what a walk does with the last name of its path.
type end int

const (
	vessel end = iota // follows the bonds of the last name to where they lead
	itself            // keeps the last name itself: a tether's bond
)

// place is where a walk ended: a name in a hall that passed the rule.
// Whatever stands at the name is told by stat, nil when nothing does; the
// walk judges it no further.
type place struct {
	hall  int      // an O_PATH descriptor of the hall holding name
	halls []string // the names of the halls from the ground down to hall
	name  string
	stat  *unix.Stat_t
}

// path is the place as the ground names it, every bond along it followed.
func (p *place) path() string {
	return "/" + filepath.Join(append(p.halls[:len(p.halls):len(p.halls)], p.name)...)
}

// hallPath is the hall of the place as the ground names it.
func (p *place) hallPath() string { return "/" + filepath.Join(p.halls...) }

func (p *place) close() { _ = unix.Close(p.hall) }

// walk walks path from the ground's root, one name after another, on
// descriptors opened with O_PATH|O_NOFOLLOW: bonds are read with readlinkat
// and walked anew, at most maxBonds, and every hall met must pass judge. A
// last name that is absent ends the walk well; an absent hall does not.
// The errors it returns are lamented already.
func (g Ground) walk(path string, e end) (*place, error) {
	if !filepath.IsAbs(path) {
		return nil, lament(path, errAstray)
	}
	root, err := unix.Open(g.Root, unix.O_PATH|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, lament("/", err)
	}
	stack, names := []int{root}, []string{}
	release := func(keep int) {
		for _, fd := range stack {
			if fd != keep {
				_ = unix.Close(fd)
			}
		}
	}
	if err := g.judge(root); err != nil {
		release(-1)
		return nil, lament("/", err)
	}
	queue, bonds := components(path), 0
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		top := stack[len(stack)-1]
		here := "/" + filepath.Join(append(names[:len(names):len(names)], c)...)
		if c == ".." {
			if len(stack) > 1 {
				_ = unix.Close(top)
				stack, names = stack[:len(stack)-1], names[:len(names)-1]
			}
			continue
		}
		last := len(queue) == 0
		at := func(st *unix.Stat_t) (*place, error) {
			release(top)
			return &place{hall: top, halls: names, name: c, stat: st}, nil
		}
		fd, err := unix.Openat(top, c, unix.O_PATH|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if err == unix.ENOENT && last {
			return at(nil)
		}
		if err != nil {
			release(-1)
			return nil, lament(here, err)
		}
		var st unix.Stat_t
		if err := unix.Fstat(fd, &st); err != nil {
			_ = unix.Close(fd)
			release(-1)
			return nil, lament(here, err)
		}
		switch st.Mode & unix.S_IFMT {
		case unix.S_IFLNK:
			if last && e == itself {
				_ = unix.Close(fd)
				return at(&st)
			}
			bonds++
			anchor, err := readlink(fd, "")
			_ = unix.Close(fd)
			switch {
			case bonds > maxBonds:
				err = errBondLoop
			case err == nil && filepath.IsAbs(anchor):
				release(root)
				stack, names = []int{root}, []string{}
			}
			if err != nil {
				release(-1)
				return nil, lament(path, err)
			}
			queue = append(components(anchor), queue...)
		case unix.S_IFDIR:
			if err := g.judge(fd); err != nil {
				_ = unix.Close(fd)
				release(-1)
				return nil, lament(here, err)
			}
			if last {
				_ = unix.Close(fd)
				return at(&st)
			}
			stack, names = append(stack, fd), append(names, c)
		default:
			_ = unix.Close(fd)
			if last {
				return at(&st)
			}
			release(-1)
			return nil, lament(here, unix.ENOTDIR)
		}
	}
	// The path ends upon a hall it named by "/" or "..", not by a name.
	release(-1)
	return nil, lament(path, errNoScroll)
}

// judge applies the trusted-path rule to the hall at fd: the root owns it,
// and none but its owner may write it. An access list upon the hall needs
// no judgement of its own: the machine shows its mask as the group's seal.
func (g Ground) judge(fd int) error {
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return err
	}
	switch {
	case g.Sovereign == nil || !g.Sovereign(st.Uid):
		return errUnsovereign
	case st.Mode&0o022 != 0:
		return errOpenHall
	}
	return nil
}

// components are the names of path, without the empty ones and ".".
func components(path string) []string {
	var cs []string
	for _, c := range strings.Split(path, "/") {
		if c != "" && c != "." {
			cs = append(cs, c)
		}
	}
	return cs
}

// readlink reads where the bond name in the hall dirfd leads, or the bond
// dirfd itself when name is "".
func readlink(dirfd int, name string) (string, error) {
	for size := 256; ; size *= 2 {
		buf := make([]byte, size)
		n, err := unix.Readlinkat(dirfd, name, buf)
		if err != nil {
			return "", err
		}
		if n < size {
			return string(buf[:n]), nil
		}
	}
}
