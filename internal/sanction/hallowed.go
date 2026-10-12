package sanction

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

// ReadHallowed reads the sanction of the rite as the exalted servitor does,
// with the root's authority, trusting nothing but what the root laid: the
// store, the master's hall and the sanction are opened with openat2 and
// RESOLVE_NO_SYMLINKS, so that no bond anywhere along the way is followed,
// and each must be owned by the root (sovereign reports whether a uid is
// the root's) and written by none but its owner. Like Read it returns nil,
// and no lament, when no sanction is kept; what is no true sanction of the
// rite and master is lamented.
func (s Store) ReadHallowed(rite string, sovereign func(uid uint32) bool) (*Sanction, error) {
	p := s.Path(rite)
	lament := func(err error) error {
		return fmt.Errorf("the sanction of %q at %s cannot be trusted: %w", rite, p, err)
	}
	store, err := hallowedOpen(unix.AT_FDCWD, s.Root, unix.O_PATH|unix.O_DIRECTORY, sovereign)
	if err == nil {
		defer func() { _ = unix.Close(store) }()
		var hall int
		hall, err = hallowedOpen(store, strconv.Itoa(s.UID), unix.O_PATH|unix.O_DIRECTORY, sovereign)
		if err == nil {
			defer func() { _ = unix.Close(hall) }()
			var fd int
			fd, err = hallowedOpen(hall, rite+".json", unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK, sovereign)
			if err == nil {
				return s.hallowedRead(os.NewFile(uintptr(fd), p), rite)
			}
		}
	}
	if errors.Is(err, unix.ENOENT) {
		return nil, nil
	}
	return nil, lament(err)
}

// hallowedRead reads the sanction of the rite from f, which it closes.
func (s Store) hallowedRead(f *os.File, rite string) (*Sanction, error) {
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, limit))
	if err != nil {
		return nil, fmt.Errorf("the sanction of %q at %s cannot be read: %w", rite, f.Name(), err)
	}
	return s.parse(data, rite)
}

// errUnhallowed is what the exalted servitor finds at a place of the store
// the root did not lay.
var errUnhallowed = errors.New("the root does not own it alone, or others than its owner may write it")

// errStranger is a place of the store that is neither hall nor scroll as
// the store keeps them.
var errStranger = errors.New("it is not what the store keeps there")

// hallowedOpen opens name beneath dirfd with how, following no bond, and
// judges what it opened: owned by the sovereign, written by its owner alone,
// a hall when how opens one, else a scroll.
func hallowedOpen(dirfd int, name string, how int, sovereign func(uint32) bool) (int, error) {
	resolve := uint64(unix.RESOLVE_NO_SYMLINKS)
	if dirfd != unix.AT_FDCWD {
		resolve |= unix.RESOLVE_BENEATH
	}
	var fd int
	var err error
	for {
		fd, err = unix.Openat2(dirfd, name, &unix.OpenHow{Flags: uint64(how | unix.O_CLOEXEC), Resolve: resolve})
		if !errors.Is(err, unix.EINTR) && !errors.Is(err, unix.EAGAIN) {
			break
		}
	}
	if err != nil {
		return -1, err
	}
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		_ = unix.Close(fd)
		return -1, err
	}
	want := uint32(unix.S_IFREG)
	if how&unix.O_DIRECTORY != 0 {
		want = unix.S_IFDIR
	}
	switch {
	case st.Mode&unix.S_IFMT != want:
		err = errStranger
	case sovereign == nil || !sovereign(st.Uid) || st.Mode&0o022 != 0:
		err = errUnhallowed
	}
	if err != nil {
		_ = unix.Close(fd)
		return -1, err
	}
	return fd, nil
}
