package sanction

import (
	"crypto/rand"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// Owner is who owns a hall or a sanction the exalted servitor lays.
type Owner struct{ UID, GID int }

// Keeper is the authority with which the exalted servitor keeps the store
// of sanctions, as the root: whom it holds the root (Sovereign), whom the
// store's halls belong to (Lord: the root and its group), and the primary
// group of a master (Kin), whose hall and sanctions belong to that group so
// that the master may read their own. The trials keep stores of their own,
// the invoker of the trials standing in for the root.
type Keeper struct {
	Sovereign func(uid uint32) bool
	Lord      Owner
	Kin       func(uid int) (gid int, err error)
}

// Replaced tells what stood at the place of a sanction before it was laid
// anew or struck: whether one stood, and its sigil ("" when it was no true
// sanction).
type Replaced struct {
	Stood bool
	Sigil string
}

// The seals of the store: its halls, the master's hall, and a sanction.
const (
	storeSeal    = 0o755
	masterSeal   = 0o750
	sanctionSeal = 0o640
)

// Lay lays the sanction sn in the store as the exalted servitor does, with
// the root's authority, and tells what it replaced. On first use it raises
// the halls of the store — the hall of the servitor above it and the store
// itself, the Lord's and sealed 0755, then the master's hall, of the Lord
// and the master's group and sealed 0750 — each opened with openat2 and
// RESOLVE_NO_SYMLINKS beneath the hall above the store, which must stand,
// and judged as ReadHallowed judges them. The sanction is written as strict
// JSON to a fresh scroll beside its place, owned by the Lord and the group
// of the master's hall and sealed 0640, made durable and renamed over its
// place, so that no reader ever sees half a sanction. A sanction that is no
// true sanction of the store's master is never laid.
func (s Store) Lay(sn *Sanction, k Keeper) (Replaced, error) {
	rite := sn.Rite
	lament := func(err error) (Replaced, error) {
		return Replaced{}, fmt.Errorf("the sanction of %q cannot be laid at %s: %w", rite, s.Path(rite), err)
	}
	if !librarium.ValidName(rite) {
		return lament(errors.New("it names no rite"))
	}
	if why := sn.untrue(rite, s.UID); why != "" {
		return lament(errors.New(why))
	}
	data, err := json.Marshal(sn, json.Deterministic(true))
	if err != nil {
		return lament(err)
	}
	hall, err := s.open(k, true)
	if err != nil {
		return lament(err)
	}
	defer func() { _ = unix.Close(hall) }()
	was, err := s.former(hall, rite, k)
	if err == nil {
		err = inscribe(hall, rite+".json", append(data, '\n'), k.Lord.UID)
	}
	if err != nil {
		return lament(err)
	}
	return was, nil
}

// Strike removes the sanction of the rite from the store as the exalted
// servitor does, through halls judged as Lay judges them, and tells what it
// removed: even a sanction that is no true one is struck. None kept is a
// lament that is fs.ErrNotExist.
func (s Store) Strike(rite string, k Keeper) (Replaced, error) {
	lament := func(err error) (Replaced, error) {
		return Replaced{}, fmt.Errorf("the sanction of %q at %s cannot be struck: %w", rite, s.Path(rite), err)
	}
	if !librarium.ValidName(rite) {
		return lament(errors.New("it names no rite"))
	}
	hall, err := s.open(k, false)
	if err != nil {
		return lament(err)
	}
	defer func() { _ = unix.Close(hall) }()
	was, err := s.former(hall, rite, k)
	switch {
	case err != nil:
		return lament(err)
	case !was.Stood:
		return lament(unix.ENOENT)
	}
	if err := unix.Unlinkat(hall, rite+".json", 0); err != nil {
		return lament(err)
	}
	if err := unix.Fsync(hall); err != nil {
		return lament(err)
	}
	return was, nil
}

// Kept names, in order, the rites whose sanctions the store keeps for its
// master, read as the exalted servitor reads the store; a store or hall
// never raised keeps none.
func (s Store) Kept(k Keeper) ([]string, error) {
	hall, err := s.open(k, false)
	if errors.Is(err, unix.ENOENT) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("the sanctions at %s cannot be read: %w", s.hall(), err)
	}
	f := os.NewFile(uintptr(hall), s.hall())
	defer func() { _ = f.Close() }()
	entries, err := f.ReadDir(-1)
	if err != nil {
		return nil, fmt.Errorf("the sanctions at %s cannot be read: %w", s.hall(), err)
	}
	var names []string
	for _, e := range entries {
		if name, ok := strings.CutSuffix(e.Name(), ".json"); ok && e.Type().IsRegular() && librarium.ValidName(name) {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names, nil
}

// open opens the master's hall of the store for reading, from the hall
// above the store down, raising every hall that does not stand when raise.
// Unraised, a hall that does not stand is unix.ENOENT.
func (s Store) open(k Keeper, raise bool) (int, error) {
	if !filepath.IsAbs(s.Root) {
		return -1, errors.New("the store does not begin at the root")
	}
	servitor := filepath.Dir(s.Root)
	fd, err := hallowedOpen(unix.AT_FDCWD, filepath.Dir(servitor), unix.O_RDONLY|unix.O_DIRECTORY, k.Sovereign)
	if err != nil {
		return -1, err
	}
	lord := func() (int, error) { return k.Lord.GID, nil }
	kin := func() (int, error) {
		if k.Kin == nil {
			return -1, errors.New("the group of the master is unknown")
		}
		gid, err := k.Kin(s.UID)
		if err != nil {
			return -1, fmt.Errorf("the group of the master, uid %d, cannot be found: %w", s.UID, err)
		}
		return gid, nil
	}
	for _, h := range []struct {
		name  string
		seal  uint32
		group func() (int, error)
	}{
		{filepath.Base(servitor), storeSeal, lord},
		{filepath.Base(s.Root), storeSeal, lord},
		{strconv.Itoa(s.UID), masterSeal, kin},
	} {
		next, err := k.hall(fd, h.name, raise, h.seal, h.group)
		_ = unix.Close(fd)
		if err != nil {
			return -1, err
		}
		fd = next
	}
	return fd, nil
}

// hall opens the hall name beneath dirfd, judged; when raise and it does
// not stand, it is raised first, the Lord's and of group's, sealed seal,
// and dirfd is made durable.
func (k Keeper) hall(dirfd int, name string, raise bool, seal uint32, group func() (int, error)) (int, error) {
	raised := false
	if raise {
		switch err := unix.Mkdirat(dirfd, name, 0o700); {
		case err == nil:
			raised = true
		case !errors.Is(err, unix.EEXIST):
			return -1, err
		}
	}
	fd, err := hallowedOpen(dirfd, name, unix.O_RDONLY|unix.O_DIRECTORY, k.Sovereign)
	if err != nil || !raised {
		return fd, err
	}
	gid, err := group()
	if err == nil {
		err = unix.Fchown(fd, k.Lord.UID, gid)
	}
	if err == nil {
		err = unix.Fchmod(fd, seal)
	}
	if err == nil {
		err = unix.Fsync(dirfd)
	}
	if err != nil {
		_ = unix.Close(fd)
		_ = unix.Unlinkat(dirfd, name, unix.AT_REMOVEDIR)
		return -1, err
	}
	return fd, nil
}

// former tells what stands at the place of the rite's sanction in the
// master's hall, judged as ReadHallowed judges it.
func (s Store) former(hall int, rite string, k Keeper) (Replaced, error) {
	fd, err := hallowedOpen(hall, rite+".json", unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK, k.Sovereign)
	if errors.Is(err, unix.ENOENT) {
		return Replaced{}, nil
	}
	if err != nil {
		return Replaced{}, err
	}
	f := os.NewFile(uintptr(fd), s.Path(rite))
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, limit))
	if err != nil {
		return Replaced{}, err
	}
	was := Replaced{Stood: true}
	if old, err := s.parse(data, rite); err == nil {
		was.Sigil = old.Sigil
	}
	return was, nil
}

// inscribe writes data to name in the hall atomically: a fresh scroll
// beside it, opened O_CREAT|O_EXCL|O_NOFOLLOW at 0600, receives it, then its
// owner (uid and the group of the hall), then its seal; it is made durable
// and renamed over name, and the hall is made durable.
func inscribe(hall int, name string, data []byte, uid int) error {
	var st unix.Stat_t
	if err := unix.Fstat(hall, &st); err != nil {
		return err
	}
	var fresh string
	var fd int
	for {
		fresh = "." + name + ".servitor-" + rand.Text()[:12]
		var err error
		fd, err = unix.Openat(hall, fresh, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
		if err == nil {
			break
		}
		if !errors.Is(err, unix.EEXIST) {
			return err
		}
	}
	f := os.NewFile(uintptr(fd), fresh)
	_, err := f.Write(data)
	if err == nil {
		err = unix.Fchown(fd, uid, int(st.Gid))
	}
	if err == nil {
		err = unix.Fchmod(fd, sanctionSeal)
	}
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = unix.Renameat(hall, fresh, hall, name)
	}
	if err != nil {
		_ = unix.Unlinkat(hall, fresh, 0)
		return err
	}
	return unix.Fsync(hall)
}
