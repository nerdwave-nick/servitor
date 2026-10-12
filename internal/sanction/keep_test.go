package sanction

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
)

// keeping is the store of a trial as the exalted servitor keeps it: its
// root beneath a hall of the trials' own standing in for /var/lib, and a
// keeper for whom the invoker of the trials is the root.
type keeping struct {
	lib    string // the hall above the store's own hall, /var/lib
	store  Store
	keeper Keeper
	kin    int   // the primary group the keeper finds for the master
	asked  []int // the masters whose group the keeper sought
}

func newKeeping(t *testing.T) *keeping {
	t.Helper()
	lib := filepath.Join(t.TempDir(), "var", "lib")
	must(t, os.MkdirAll(lib, 0o755))
	k := &keeping{lib: lib, store: Store{Root: filepath.Join(lib, "servitor", "sanctions"), UID: 1000},
		kin: kinOfTrials()}
	k.keeper = Keeper{Sovereign: mine, Lord: Owner{UID: os.Getuid(), GID: os.Getgid()},
		Kin: func(uid int) (int, error) { k.asked = append(k.asked, uid); return k.kin, nil }}
	return k
}

// kinOfTrials is a group of the invoker of the trials other than its
// primary one when it has one, so that the master's group is told apart
// from the root's.
func kinOfTrials() int {
	groups, _ := os.Getgroups()
	for _, g := range groups {
		if g != os.Getgid() {
			return g
		}
	}
	return os.Getgid()
}

// owned asserts that path is owned by the invoker of the trials and the
// group gid, sealed seal.
func owned(t *testing.T, path string, gid int, seal fs.FileMode) {
	t.Helper()
	fi, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	st := fi.Sys().(*syscall.Stat_t)
	if int(st.Uid) != os.Getuid() || int(st.Gid) != gid || fi.Mode().Perm() != seal {
		t.Errorf("%s is %d:%d %04o, want %d:%d %04o", path, st.Uid, st.Gid, fi.Mode().Perm(), os.Getuid(), gid, seal)
	}
}

// TestLay_RaisesTheStoreOnFirstUse: the first sanction raises the store's
// halls — servitor/ and sanctions/ the root's and sealed 0755, the master's
// hall the root's and of the master's group, sealed 0750 — and lays the
// sanction as strict JSON, the root's and of the master's group, sealed
// 0640, where the servitor and the exalted servitor read it.
func TestLay_RaisesTheStoreOnFirstUse(t *testing.T) {
	k := newKeeping(t)
	u := build(t, splitLock, splitLockFiles())
	was, err := k.store.Lay(New(u, at), k.keeper)
	if err != nil || was.Stood {
		t.Fatalf("Lay = %+v, %v", was, err)
	}
	gid := os.Getgid()
	owned(t, filepath.Join(k.lib, "servitor"), gid, 0o755)
	owned(t, k.store.Root, gid, 0o755)
	owned(t, filepath.Join(k.store.Root, "1000"), k.kin, 0o750)
	owned(t, k.store.Path("split-lock"), k.kin, 0o640)
	if !slices.Equal(k.asked, []int{1000}) {
		t.Errorf("the keeper sought the group of %v, want [1000]", k.asked)
	}
	for _, read := range []func() (*Sanction, error){
		func() (*Sanction, error) { return k.store.Read("split-lock") },
		func() (*Sanction, error) { return k.store.ReadHallowed("split-lock", mine) },
	} {
		if s, err := read(); err != nil || s == nil || s.Sigil != u.Sigil() || !s.SanctionedAt.Equal(at) {
			t.Errorf("the sanction laid reads as %+v, %v", s, err)
		}
	}
	if entries, _ := os.ReadDir(filepath.Join(k.store.Root, "1000")); len(entries) != 1 {
		t.Errorf("the master's hall holds %v", entries)
	}
}

// TestLay_ReplacesTheFormerAtomically: a sanction laid anew takes the place
// of the former by a rename — another scroll at the same name, no remnant
// beside it — and tells the sigil it replaced; a former that is no true
// sanction is replaced all the same, its sigil unknown.
func TestLay_ReplacesTheFormerAtomically(t *testing.T) {
	k := newKeeping(t)
	first := build(t, splitLock, splitLockFiles())
	if _, err := k.store.Lay(New(first, at), k.keeper); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(k.store.Path("split-lock"))
	must(t, err)
	second := build(t, strings.Replace(splitLock, "kernel.split_lock_mitigate=0", "kernel.split_lock_mitigate=2", 1),
		splitLockFiles())
	was, err := k.store.Lay(New(second, at), k.keeper)
	if err != nil || !was.Stood || was.Sigil != first.Sigil() {
		t.Fatalf("Lay = %+v, %v; want the former sigil %s", was, err, first.Sigil())
	}
	after, err := os.Stat(k.store.Path("split-lock"))
	must(t, err)
	if os.SameFile(before, after) {
		t.Error("the former scroll was written over in place, not replaced")
	}
	if s, err := k.store.Read("split-lock"); err != nil || s.Sigil != second.Sigil() {
		t.Errorf("the store holds %v, %v", s, err)
	}
	owned(t, k.store.Path("split-lock"), k.kin, 0o640)

	must(t, os.WriteFile(k.store.Path("split-lock"), []byte("{corrupt"), 0o640))
	was, err = k.store.Lay(New(first, at), k.keeper)
	if err != nil || !was.Stood || was.Sigil != "" {
		t.Fatalf("Lay over a corrupt sanction = %+v, %v", was, err)
	}
	if entries, _ := os.ReadDir(filepath.Join(k.store.Root, "1000")); len(entries) != 1 {
		t.Errorf("the master's hall holds %v", entries)
	}
}

// TestLay_RefusesWhatTheRootDidNotLay: the store is laid only through halls
// the root owns and alone may write, following no bond; a sanction of
// another master or rite is never laid. Nothing is written upon a refusal.
func TestLay_RefusesWhatTheRootDidNotLay(t *testing.T) {
	u := build(t, splitLock, splitLockFiles())
	for what, mar := range map[string]func(t *testing.T, k *keeping) *Sanction{
		"a hall above the store others may write": func(t *testing.T, k *keeping) *Sanction {
			must(t, os.Chmod(k.lib, 0o777))
			return New(u, at)
		},
		"a store others may write": func(t *testing.T, k *keeping) *Sanction {
			must(t, os.MkdirAll(k.store.Root, 0o755))
			must(t, os.Chmod(k.store.Root, 0o757))
			return New(u, at)
		},
		"a bond for the store": func(t *testing.T, k *keeping) *Sanction {
			elsewhere := filepath.Join(k.lib, "elsewhere")
			must(t, os.MkdirAll(elsewhere, 0o755))
			must(t, os.MkdirAll(filepath.Join(k.lib, "servitor"), 0o755))
			must(t, os.Symlink(elsewhere, k.store.Root))
			return New(u, at)
		},
		"a bond for the sanction": func(t *testing.T, k *keeping) *Sanction {
			must(t, os.MkdirAll(filepath.Dir(k.store.Path("split-lock")), 0o750))
			must(t, os.Symlink(filepath.Join(k.lib, "x"), k.store.Path("split-lock")))
			return New(u, at)
		},
		"a store the root does not own": func(t *testing.T, k *keeping) *Sanction {
			k.keeper.Sovereign = func(uint32) bool { return false }
			return New(u, at)
		},
		"the sanction of another master": func(t *testing.T, k *keeping) *Sanction {
			other := *u
			other.UID = 1001
			return New(&other, at)
		},
		"a sanction whose sigil is not its own": func(t *testing.T, k *keeping) *Sanction {
			s := New(u, at)
			s.Sigil = strings.Repeat("0", 64)
			return s
		},
		"a master of no group": func(t *testing.T, k *keeping) *Sanction {
			k.keeper.Kin = func(int) (int, error) { return 0, errors.New("no such master") }
			return New(u, at)
		},
	} {
		t.Run(what, func(t *testing.T) {
			k := newKeeping(t)
			sn := mar(t, k)
			if was, err := k.store.Lay(sn, k.keeper); err == nil {
				t.Fatalf("Lay = %+v; want a lament", was)
			} else if !strings.Contains(err.Error(), "sanction") {
				t.Errorf("the lament does not name the sanction: %v", err)
			}
			if fi, err := os.Stat(k.store.Path("split-lock")); err == nil && fi.Mode().IsRegular() {
				t.Error("a sanction was laid")
			}
		})
	}
}

// TestStrike_RemovesTheSanction: a sanction struck is gone and tells its
// sigil, even one that is no true sanction; none kept is fs.ErrNotExist.
func TestStrike_RemovesTheSanction(t *testing.T) {
	k := newKeeping(t)
	if _, err := k.store.Strike("split-lock", k.keeper); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Strike of a store never raised = %v; want fs.ErrNotExist", err)
	}
	u := build(t, splitLock, splitLockFiles())
	if _, err := k.store.Lay(New(u, at), k.keeper); err != nil {
		t.Fatal(err)
	}
	if _, err := k.store.Strike("other", k.keeper); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Strike(other) = %v; want fs.ErrNotExist", err)
	}
	was, err := k.store.Strike("split-lock", k.keeper)
	if err != nil || !was.Stood || was.Sigil != u.Sigil() {
		t.Fatalf("Strike = %+v, %v", was, err)
	}
	if _, err := os.Lstat(k.store.Path("split-lock")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the sanction stands: %v", err)
	}
	must(t, os.WriteFile(k.store.Path("broken"), []byte("{"), 0o640))
	if was, err := k.store.Strike("broken", k.keeper); err != nil || !was.Stood || was.Sigil != "" {
		t.Errorf("Strike(broken) = %+v, %v", was, err)
	}
	must(t, os.Chmod(filepath.Join(k.store.Root, "1000"), 0o770))
	if _, err := k.store.Strike("split-lock", k.keeper); err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Strike through a hall others may write = %v; want a lament", err)
	}
}

// TestKept_NamesEveryRiteKept: the exalted servitor names the rites whose
// sanctions the master keeps, in order; a store never raised keeps none.
func TestKept_NamesEveryRiteKept(t *testing.T) {
	k := newKeeping(t)
	if names, err := k.store.Kept(k.keeper); err != nil || len(names) != 0 {
		t.Errorf("Kept of a store never raised = %v, %v", names, err)
	}
	u := build(t, splitLock, splitLockFiles())
	for _, rite := range []string{"zeta", "alpha"} {
		v := *u
		v.Rite = rite
		if _, err := k.store.Lay(New(&v, at), k.keeper); err != nil {
			t.Fatal(err)
		}
	}
	must(t, os.WriteFile(filepath.Join(k.store.Root, "1000", "notes.txt"), nil, 0o640))
	if names, err := k.store.Kept(k.keeper); err != nil || !slices.Equal(names, []string{"alpha", "zeta"}) {
		t.Errorf("Kept = %v, %v", names, err)
	}
}
