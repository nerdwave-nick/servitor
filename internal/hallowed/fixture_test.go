package hallowed

import (
	"encoding/binary"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"golang.org/x/sys/unix"
)

// temple is a ground of its own in a temporary hall, where the invoker of
// the trials stands in for the root: halls it owns are held sovereign, and
// vessels brought into being are given to it.
type temple struct {
	t    *testing.T
	root string
	g    Ground
	me   Owner
}

func newTemple(t *testing.T) *temple {
	t.Helper()
	root := t.TempDir()
	me := Owner{UID: uint32(os.Getuid()), GID: uint32(os.Getgid())}
	return &temple{t: t, root: root, me: me, g: Ground{
		Root:      root,
		Sovereign: func(uid uint32) bool { return uid == me.UID },
		Lord:      me,
	}}
}

// at is the place on the disk of path, a path of the temple's ground.
func (tp *temple) at(path string) string { return filepath.Join(tp.root, path) }

// hall raises the hall at path with exactly mode (beyond the umask).
func (tp *temple) hall(path string, mode fs.FileMode) {
	tp.t.Helper()
	if err := os.MkdirAll(tp.at(path), 0o755); err != nil {
		tp.t.Fatal(err)
	}
	if err := os.Chmod(tp.at(path), mode); err != nil {
		tp.t.Fatal(err)
	}
}

// scroll writes content at path with exactly mode.
func (tp *temple) scroll(path, content string, mode fs.FileMode) {
	tp.t.Helper()
	if err := os.WriteFile(tp.at(path), []byte(content), mode); err != nil {
		tp.t.Fatal(err)
	}
	if err := os.Chmod(tp.at(path), mode); err != nil {
		tp.t.Fatal(err)
	}
}

// bond lays a bond at path, leading to anchor as written.
func (tp *temple) bond(path, anchor string) {
	tp.t.Helper()
	if err := os.Symlink(anchor, tp.at(path)); err != nil {
		tp.t.Fatal(err)
	}
}

// content is what the scroll at path holds.
func (tp *temple) content(path string) string {
	tp.t.Helper()
	data, err := os.ReadFile(tp.at(path))
	if err != nil {
		tp.t.Fatal(err)
	}
	return string(data)
}

// stat is what the machine knows of path itself, not following a bond.
func (tp *temple) stat(path string) unix.Stat_t {
	tp.t.Helper()
	var st unix.Stat_t
	if err := unix.Lstat(tp.at(path), &st); err != nil {
		tp.t.Fatal(err)
	}
	return st
}

// absent reports whether nothing stands at path.
func (tp *temple) absent(path string) bool {
	_, err := os.Lstat(tp.at(path))
	return errors.Is(err, fs.ErrNotExist)
}

// anchor is where the bond at path leads, as written.
func (tp *temple) anchor(path string) string {
	tp.t.Helper()
	a, err := os.Readlink(tp.at(path))
	if err != nil {
		tp.t.Fatal(err)
	}
	return a
}

// names are the names standing in the hall at path, so that no fresh
// scroll left behind goes unseen.
func (tp *temple) names(path string) []string {
	tp.t.Helper()
	entries, err := os.ReadDir(tp.at(path))
	if err != nil {
		tp.t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	slices.Sort(names)
	return names
}

func seal(st unix.Stat_t) fs.FileMode { return fs.FileMode(st.Mode & 0o7777) }

func sealPtr(m fs.FileMode) *fs.FileMode { return &m }

// attribute sets the extended attribute name on path, or skips the trial
// when the temporary halls bear none.
func attribute(t *testing.T, path, name string, value []byte) {
	t.Helper()
	if err := unix.Lsetxattr(path, name, value, 0); err != nil {
		if errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.EOPNOTSUPP) {
			t.Skipf("the temporary halls bear no attribute %s: %v", name, err)
		}
		t.Fatal(err)
	}
}

// defaultAccess is a default access list in the machine's binary form that
// grants uid of its own, beyond the seal, so that the machine keeps it.
func defaultAccess(uid uint32) []byte {
	const undefined = 0xffffffff
	entries := []struct {
		tag, perm uint16
		id        uint32
	}{
		{0x01, 7, undefined}, // the owner
		{0x02, 7, uid},       // a named owner
		{0x04, 5, undefined}, // the owning group
		{0x10, 7, undefined}, // the mask
		{0x20, 5, undefined}, // all others
	}
	b := binary.LittleEndian.AppendUint32(nil, 2)
	for _, e := range entries {
		b = binary.LittleEndian.AppendUint16(b, e.tag)
		b = binary.LittleEndian.AppendUint16(b, e.perm)
		b = binary.LittleEndian.AppendUint32(b, e.id)
	}
	return b
}
