package sanction

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mine holds the invoker of the trials the root, as the exalted servitor's
// trials do.
func mine(uid uint32) bool { return uid == uint32(os.Getuid()) }

// TestReadHallowed_ReadsTheSanctionLaid: the exalted servitor reads a
// sanction laid as the store keeps it; none laid is no sanction and no
// lament.
func TestReadHallowed_ReadsTheSanctionLaid(t *testing.T) {
	u := buildGaming(t, gaming, gamingFiles())
	root := t.TempDir()
	lay(t, root, "gaming", New(u, at))
	st := Store{Root: root, UID: 1000}
	s, err := st.ReadHallowed("gaming", mine)
	if err != nil || s == nil {
		t.Fatalf("ReadHallowed = %v, %v", s, err)
	}
	if s.Sigil != u.Sigil() || string(s.Canonical()) != string(u.Canonical()) {
		t.Errorf("read %s\n%s", s.Sigil, s.Canonical())
	}
	for _, st := range []Store{{Root: root, UID: 1001}, {Root: filepath.Join(root, "nowhere"), UID: 1000}} {
		if s, err := st.ReadHallowed("gaming", mine); s != nil || err != nil {
			t.Errorf("%+v: ReadHallowed = %v, %v; want nothing", st, s, err)
		}
	}
	if s, err := st.ReadHallowed("other", mine); s != nil || err != nil {
		t.Errorf("ReadHallowed(other) = %v, %v", s, err)
	}
}

// TestReadHallowed_RefusesWhatTheRootDidNotLay: a bond anywhere along the
// way, a hall or sanction others may write, or one the root does not own,
// is lamented and is no sanction; so is a sanction of another rite.
func TestReadHallowed_RefusesWhatTheRootDidNotLay(t *testing.T) {
	u := buildGaming(t, gaming, gamingFiles())
	for what, c := range map[string]struct {
		mar       func(t *testing.T, root, laid string) string // returns the store's root
		sovereign func(uint32) bool
		rite      string
	}{
		"a bond for the sanction": {mar: func(t *testing.T, root, laid string) string {
			moved := filepath.Join(filepath.Dir(laid), "elsewhere.json")
			must(t, os.Rename(laid, moved))
			must(t, os.Symlink("elsewhere.json", laid))
			return root
		}},
		"a bond for the master's hall": {mar: func(t *testing.T, root, laid string) string {
			must(t, os.Rename(filepath.Join(root, "1000"), filepath.Join(root, "hall")))
			must(t, os.Symlink("hall", filepath.Join(root, "1000")))
			return root
		}},
		"a bond for the store": {mar: func(t *testing.T, root, laid string) string {
			bond := filepath.Join(t.TempDir(), "store")
			must(t, os.Symlink(root, bond))
			return bond
		}},
		"a sanction others may write": {mar: func(t *testing.T, root, laid string) string {
			must(t, os.Chmod(laid, 0o660))
			return root
		}},
		"a master's hall others may write": {mar: func(t *testing.T, root, laid string) string {
			must(t, os.Chmod(filepath.Dir(laid), 0o770))
			return root
		}},
		"a store others may write": {mar: func(t *testing.T, root, laid string) string {
			must(t, os.Chmod(root, 0o757))
			return root
		}},
		"a store the root does not own": {sovereign: func(uint32) bool { return false }},
		"a scroll that is no sanction": {mar: func(t *testing.T, root, laid string) string {
			must(t, os.Remove(laid))
			must(t, os.Mkdir(laid, 0o750))
			return root
		}},
		"the sanction of another rite": {rite: "other", mar: func(t *testing.T, root, laid string) string {
			must(t, os.Rename(laid, filepath.Join(filepath.Dir(laid), "other.json")))
			return root
		}},
	} {
		t.Run(what, func(t *testing.T) {
			root := t.TempDir()
			laid := lay(t, root, "gaming", New(u, at))
			store := root
			if c.mar != nil {
				store = c.mar(t, root, laid)
			}
			sovereign, rite := c.sovereign, c.rite
			if sovereign == nil {
				sovereign = mine
			}
			if rite == "" {
				rite = "gaming"
			}
			s, err := Store{Root: store, UID: 1000}.ReadHallowed(rite, sovereign)
			if s != nil || err == nil {
				t.Fatalf("ReadHallowed = %v, %v; want a lament", s, err)
			}
			if !strings.Contains(err.Error(), "sanction") {
				t.Errorf("the lament does not name the sanction: %v", err)
			}
		})
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
