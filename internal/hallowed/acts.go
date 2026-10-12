package hallowed

import (
	"crypto/rand"
	"errors"
	"io"
	"io/fs"
	"os"

	"golang.org/x/sys/unix"
)

// form is what stood at the place of an act before it.
type form int

const (
	absent form = iota
	scroll
	bond
)

// former is what stood at the place of an act before it, kept whole so that
// the act can be undone exactly.
type former struct {
	form      form
	scripture []byte // scrolls only
	owner     Owner  // scrolls only
	seal      uint32 // scrolls only: the permission bits with setuid, setgid and sticky
	anchor    string // bonds only, as written
}

// Memory is what an act remembers, in memory alone, of what stood at its
// place before it. An act that fails returns none, unless it was done and
// only its hall could not be made durable.
type Memory struct {
	g      Ground
	path   string // the place itself, its bonds already followed
	former former
}

// Read reads the scroll at path, following its bonds, at most Capacity
// bytes of it.
func (g Ground) Read(path string) ([]byte, error) {
	p, err := g.walk(path, vessel)
	if err != nil {
		return nil, err
	}
	defer p.close()
	f, err := p.open()
	if err != nil {
		return nil, lament(p.path(), err)
	}
	defer func() { _ = f.Close() }()
	data, err := scripture(f)
	return data, lament(p.path(), err)
}

// Transcribe writes scripture to the vessel at path, following its bonds,
// atomically. A standing vessel keeps its owner and seal; one brought into
// being belongs to the ground's Lord and takes seal, else 0644.
func (g Ground) Transcribe(path string, scripture []byte, seal *fs.FileMode) (*Memory, error) {
	p, err := g.walk(path, vessel)
	if err != nil {
		return nil, err
	}
	defer p.close()
	f, err := p.former()
	if err != nil {
		return nil, lament(p.path(), err)
	}
	owner, mode := g.Lord, unixSeal(newSeal)
	if seal != nil {
		mode = unixSeal(*seal)
	}
	if f.form == scroll {
		owner, mode = f.owner, f.seal
	}
	if err := inscribe(p, scripture, owner, mode, f.form == absent); err != nil {
		return nil, lament(p.path(), err)
	}
	return p.remember(g, f)
}

// Strike removes the vessel at path, following its bonds; an absent vessel
// stays absent.
func (g Ground) Strike(path string) (*Memory, error) {
	p, err := g.walk(path, vessel)
	if err != nil {
		return nil, err
	}
	defer p.close()
	f, err := p.former()
	if err == nil && f.form != absent {
		err = p.strike()
	}
	if err != nil {
		return nil, lament(p.path(), err)
	}
	return p.remember(g, f)
}

// Revert raises again exactly what stood at the place before the act:
// scripture, owner and seal of a scroll, a bond as written, or nothing.
func (m *Memory) Revert() error {
	p, err := m.g.walk(m.path, itself)
	if err != nil {
		return err
	}
	defer p.close()
	f := m.former
	switch {
	case f.form == scroll:
		err = inscribe(p, f.scripture, f.owner, f.seal, p.stat == nil)
	case f.form == bond:
		err = lay(p, f.anchor, p.stat == nil)
	case p.stat != nil && p.stat.Mode&unix.S_IFMT == unix.S_IFDIR:
		err = errHall
	case p.stat != nil:
		err = p.strike()
	}
	if err == nil {
		err = p.settle()
	}
	return lament(p.path(), err)
}

// remember is the memory of an act done at the place over what stood there
// before, once its hall is made durable. When it could not be, the memory
// comes with the error all the same: the act was done and may be reverted.
func (p *place) remember(g Ground, f former) (*Memory, error) {
	return &Memory{g: g, path: p.path(), former: f}, lament(p.path(), p.settle())
}

// former reads what stands at the place: nothing, or a scroll of one name
// that bears no attributes, which an atomic transcription could not keep.
func (p *place) former() (former, error) {
	if p.stat == nil {
		return former{form: absent}, nil
	}
	f, err := p.open()
	if err != nil {
		return former{}, err
	}
	defer func() { _ = f.Close() }()
	var st unix.Stat_t
	if err := unix.Fstat(int(f.Fd()), &st); err != nil {
		return former{}, err
	}
	if st.Nlink > 1 {
		return former{}, errManyNames
	}
	if err := attributes(int(f.Fd())); err != nil {
		return former{}, err
	}
	data, err := scripture(f)
	if err != nil {
		return former{}, err
	}
	return former{form: scroll, scripture: data, owner: Owner{st.Uid, st.Gid}, seal: st.Mode & 0o7777}, nil
}

// open opens the scroll the walk reached for reading, never following a
// bond and never awaiting a writer; what it opened must be what the walk saw.
func (p *place) open() (*os.File, error) {
	if p.stat == nil {
		return nil, errMissing
	}
	if p.stat.Mode&unix.S_IFMT != unix.S_IFREG {
		return nil, errNoScroll
	}
	fd, err := unix.Openat(p.hall, p.name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil || st.Mode&unix.S_IFMT != unix.S_IFREG ||
		st.Dev != p.stat.Dev || st.Ino != p.stat.Ino {
		_ = unix.Close(fd)
		if err == nil {
			err = errNoScroll
		}
		return nil, err
	}
	return os.NewFile(uintptr(fd), p.path()), nil
}

// scripture reads f whole, at most Capacity bytes.
func scripture(f *os.File) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(f, Capacity+1))
	if err == nil && len(data) > Capacity {
		err = errOverflow
	}
	return data, err
}

// attributes refuses a scroll that bears any extended attribute: access
// lists and labels above all, but none survives an atomic transcription.
func attributes(fd int) error {
	n, err := unix.Flistxattr(fd, nil)
	switch {
	case errors.Is(err, unix.ENOTSUP):
		return nil
	case err != nil:
		return err
	case n > 0:
		return errAttributes
	}
	return nil
}

// inscribe writes scripture to the place atomically: a fresh scroll in the
// same hall, opened O_CREAT|O_EXCL|O_NOFOLLOW at 0600, receives it, then its
// owner, then its seal (after the owner, which may strip setuid); it is
// made durable and renamed into place, without replacing anything when
// create. The act settles the hall afterwards.
func inscribe(p *place, scripture []byte, o Owner, seal uint32, create bool) error {
	fd, fresh, err := p.fresh(func(name string) (int, error) {
		return unix.Openat(p.hall, name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	})
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(fd), fresh)
	_, err = f.Write(scripture)
	if err == nil {
		err = unix.Fchown(fd, int(o.UID), int(o.GID))
	}
	if err == nil {
		err = unix.Fchmod(fd, seal)
	}
	if err == nil {
		err = attributes(fd)
	}
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = unix.Renameat2(p.hall, fresh, p.hall, p.name, noReplace(create))
	}
	if err != nil {
		_ = unix.Unlinkat(p.hall, fresh, 0)
	}
	return err
}

// fresh brings a fresh name into being beside the place with bring, which
// fails with EEXIST when the name is taken.
func (p *place) fresh(bring func(name string) (int, error)) (int, string, error) {
	for {
		name := "." + p.name + ".servitor-" + rand.Text()[:12]
		fd, err := bring(name)
		if err != unix.EEXIST {
			return fd, name, err
		}
	}
}

// strike removes what stands at the place.
func (p *place) strike() error { return unix.Unlinkat(p.hall, p.name, 0) }

// settle makes the hall of the place durable.
func (p *place) settle() error {
	fd, err := unix.Openat(p.hall, ".", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	err = unix.Fsync(fd)
	if cerr := unix.Close(fd); err == nil {
		err = cerr
	}
	return err
}

func noReplace(create bool) uint {
	if create {
		return unix.RENAME_NOREPLACE
	}
	return 0
}

// unixSeal is the seal as the machine writes it.
func unixSeal(m fs.FileMode) uint32 {
	s := uint32(m.Perm())
	if m&fs.ModeSetuid != 0 {
		s |= unix.S_ISUID
	}
	if m&fs.ModeSetgid != 0 {
		s |= unix.S_ISGID
	}
	if m&fs.ModeSticky != 0 {
		s |= unix.S_ISVTX
	}
	return s
}
