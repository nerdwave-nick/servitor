package hallowed

import (
	"errors"
	"io/fs"
	"testing"
)

// TestRead_AnAbsentVesselIsNotToBeFound: the exalted servitor tells an
// absent vessel, which a consecrating sanctum or a transcription brings into
// being, from one it may not read.
func TestRead_AnAbsentVesselIsNotToBeFound(t *testing.T) {
	tp := altar(t)
	if _, err := tp.g.Read("/etc/absent"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("an absent vessel was read with %v", err)
	}
	tp.hall("/etc/open", 0o777)
	tp.scroll("/etc/open/x", "x", 0o644)
	if _, err := tp.g.Read("/etc/open/x"); err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a vessel beneath an open hall was read with %v", err)
	}
}

// TestMemory_TellsWhetherTheActChangedAnything: an act that found what it
// would leave remembers that it changed nothing.
func TestMemory_TellsWhetherTheActChangedAnything(t *testing.T) {
	tp := altar(t)
	tp.scroll("/etc/x", "x", 0o644)
	for _, c := range []struct {
		what string
		act  func() (*Memory, error)
		want bool
	}{
		{"binding", func() (*Memory, error) { return tp.g.Bind("/etc/theme", "/usr/share/themes/on", false) }, true},
		{"binding the same", func() (*Memory, error) { return tp.g.Bind("/etc/theme", "/usr/share/themes/on", false) }, false},
		{"rebinding", func() (*Memory, error) { return tp.g.Bind("/etc/theme", "/usr/share/themes/off", false) }, true},
		{"unbinding", func() (*Memory, error) { return tp.g.Unbind("/etc/theme", false) }, true},
		{"unbinding the absent", func() (*Memory, error) { return tp.g.Unbind("/etc/theme", false) }, false},
		{"transcribing", func() (*Memory, error) { return tp.g.Transcribe("/etc/x", []byte("y"), nil) }, true},
		{"striking", func() (*Memory, error) { return tp.g.Strike("/etc/x") }, true},
		{"striking the absent", func() (*Memory, error) { return tp.g.Strike("/etc/x") }, false},
	} {
		m, err := c.act()
		if err != nil {
			t.Fatalf("%s: %v", c.what, err)
		}
		if m.Changed() != c.want {
			t.Errorf("%s: changed %v, want %v", c.what, m.Changed(), c.want)
		}
	}
}

// TestResolve_FollowsTheBondsOfTheHallsAlone: the place of the dominion
// over a vessel is where it truly dwells within the ground; a path the walk
// cannot follow is kept as written, made clean.
func TestResolve_FollowsTheBondsOfTheHallsAlone(t *testing.T) {
	tp := altar(t)
	tp.bond("/etc/themes", "/usr/share/themes")
	tp.bond("/etc/theme", "/usr/share/themes/on")
	for path, want := range map[string]string{
		"/etc/themes/on":     "/usr/share/themes/on",
		"/etc/theme":         "/etc/theme",
		"/etc/./themes/../x": "/usr/share/x",
		"/absent/hall/x":     "/absent/hall/x",
		"/etc//absent/x":     "/etc/absent/x",
	} {
		if got := tp.g.Resolve(path); got != want {
			t.Errorf("Resolve(%q) = %q, want %q", path, got, want)
		}
	}
}
