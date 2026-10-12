package hallowed

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

// sanctuary raises a temple holding a servitor installed as the root's at
// /usr/bin/servitor.
func sanctuary(t *testing.T) *temple {
	tp := newTemple(t)
	tp.hall("/usr/bin", 0o755)
	tp.scroll("/usr/bin/servitor", "\x7fELF", 0o755)
	return tp
}

// TestEnshrined_AcceptsAServitorOfTheRootsOwn: a scroll the root owns and
// none but it may write, reached through halls of the root's own, stands
// enshrined, also when a bond through such halls names it.
func TestEnshrined_AcceptsAServitorOfTheRootsOwn(t *testing.T) {
	tp := sanctuary(t)
	tp.bond("/usr/bin/servitor-bond", "servitor")
	tp.hall("/usr/local/bin", 0o755)
	tp.bond("/usr/local/bin/servitor", "/usr/bin/servitor")
	for _, path := range []string{"/usr/bin/servitor", "/usr/bin/servitor-bond", "/usr/local/bin/servitor",
		"/usr/./bin/../bin/servitor"} {
		if err := tp.g.Enshrined(path); err != nil {
			t.Errorf("%s: %v", path, err)
		}
	}
	tp.scroll("/usr/bin/sealed", "\x7fELF", 0o500)
	if err := tp.g.Enshrined("/usr/bin/sealed"); err != nil {
		t.Errorf("a scroll only its owner may read and run: %v", err)
	}
}

// TestEnshrined_RefusesWhatOthersMayTurnAside: the installed servitor, and
// every hall above it, are written by none but the root; it is a scroll
// that stands, and it is named from the root of the machine.
func TestEnshrined_RefusesWhatOthersMayTurnAside(t *testing.T) {
	for what, c := range map[string]struct {
		mar  func(tp *temple)
		path string
		want error
	}{
		"missing": {mar: func(tp *temple) { must(tp.t, os.Remove(tp.at("/usr/bin/servitor"))) },
			want: fs.ErrNotExist},
		"beneath a missing hall": {path: "/opt/servitor/bin/servitor", want: fs.ErrNotExist},
		"a scroll its group may write": {mar: func(tp *temple) { tp.scroll("/usr/bin/servitor", "x", 0o775) },
			want: errOpenScroll},
		"a scroll all may write": {mar: func(tp *temple) { tp.scroll("/usr/bin/servitor", "x", 0o757) },
			want: errOpenScroll},
		"beneath a hall its group may write": {mar: func(tp *temple) { tp.hall("/usr/bin", 0o775) },
			want: errOpenHall},
		"beneath a hall all may write":   {mar: func(tp *temple) { tp.hall("/usr", 0o777) }, want: errOpenHall},
		"beneath a ground all may write": {mar: func(tp *temple) { tp.hall("/", 0o757) }, want: errOpenHall},
		"through a bond into an open hall": {mar: func(tp *temple) {
			tp.hall("/tmp", 0o777)
			tp.scroll("/tmp/servitor", "x", 0o755)
			must(tp.t, os.Remove(tp.at("/usr/bin/servitor")))
			tp.bond("/usr/bin/servitor", "/tmp/servitor")
		}, want: errOpenHall},
		"a hall": {mar: func(tp *temple) {
			must(tp.t, os.Remove(tp.at("/usr/bin/servitor")))
			tp.hall("/usr/bin/servitor", 0o755)
		}, want: errNoScroll},
		"a pipe": {mar: func(tp *temple) {
			must(tp.t, os.Remove(tp.at("/usr/bin/servitor")))
			must(tp.t, unix.Mkfifo(tp.at("/usr/bin/servitor"), 0o755))
		}, want: errNoScroll},
		"a relative name": {path: "usr/bin/servitor", want: errAstray},
		"halls the root does not own": {mar: func(tp *temple) {
			tp.g.Sovereign = func(uint32) bool { return false }
		}, want: errUnsovereign},
	} {
		t.Run(what, func(t *testing.T) {
			tp := sanctuary(t)
			if c.mar != nil {
				c.mar(tp)
			}
			path := "/usr/bin/servitor"
			if c.path != "" {
				path = c.path
			}
			if err := tp.g.Enshrined(path); !errors.Is(err, c.want) {
				t.Fatalf("judged %s with %v, want %v", path, err, c.want)
			}
		})
	}
}

// TestEnshrined_JudgesTheOwnerAndSealOfTheScroll: the scroll itself is the
// root's and written by none but it; a seal that only others may not run,
// or a setuid seal, is no concern of the rule. (Its owner cannot be made
// another's without the root's authority, so the stat is judged here.)
func TestEnshrined_JudgesTheOwnerAndSealOfTheScroll(t *testing.T) {
	g := Ground{Sovereign: func(uid uint32) bool { return uid == 0 }}
	for _, c := range []struct {
		uid  uint32
		mode uint32
		want error
	}{
		{0, unix.S_IFREG | 0o755, nil},
		{0, unix.S_IFREG | 0o700, nil},
		{0, unix.S_IFREG | 0o4755, nil},
		{1000, unix.S_IFREG | 0o755, errUnsovereignScroll},
		{0, unix.S_IFREG | 0o775, errOpenScroll},
		{0, unix.S_IFREG | 0o757, errOpenScroll},
		{0, unix.S_IFDIR | 0o755, errNoScroll},
		{0, unix.S_IFLNK | 0o777, errNoScroll},
		{0, unix.S_IFCHR | 0o755, errNoScroll},
	} {
		st := &unix.Stat_t{Uid: c.uid, Mode: c.mode}
		if err := g.enshrined(st); !errors.Is(err, c.want) {
			t.Errorf("uid %d, mode %o: %v, want %v", c.uid, c.mode, err, c.want)
		}
	}
}

// TestEnshrinedSelf_JudgesTheScrollTheServitorRunsFrom: the exalted
// servitor judges the scroll it truly runs from, as /proc/self/exe names
// it, which must be the very scroll that stands at that name in the ground.
// The trials name a scroll they hold open by /proc/self/fd, which the
// machine shows as it shows /proc/self/exe, or by a bond.
func TestEnshrinedSelf_JudgesTheScrollTheServitorRunsFrom(t *testing.T) {
	held := func(t *testing.T, path string) string {
		f, err := os.Open(path)
		must(t, err)
		t.Cleanup(func() { _ = f.Close() })
		return fmt.Sprintf("/proc/self/fd/%d", f.Fd())
	}
	bond := func(t *testing.T, target string) string {
		p := filepath.Join(t.TempDir(), "exe")
		must(t, os.Symlink(target, p))
		return p
	}
	for what, c := range map[string]struct {
		self func(t *testing.T, tp *temple) string
		want error // nil: enshrined; errAny: refused, whatever the words
	}{
		"held":      {self: func(t *testing.T, tp *temple) string { return held(t, tp.at("/usr/bin/servitor")) }},
		"by a bond": {self: func(t *testing.T, tp *temple) string { return bond(t, tp.at("/usr/bin/servitor")) }},
		"beneath an open hall": {self: func(t *testing.T, tp *temple) string {
			tp.hall("/usr/bin", 0o777)
			return held(t, tp.at("/usr/bin/servitor"))
		}, want: errOpenHall},
		"an open scroll": {self: func(t *testing.T, tp *temple) string {
			must(t, os.Chmod(tp.at("/usr/bin/servitor"), 0o777))
			return held(t, tp.at("/usr/bin/servitor"))
		}, want: errOpenScroll},
		"cast aside, another laid at its name": {self: func(t *testing.T, tp *temple) string {
			self := held(t, tp.at("/usr/bin/servitor"))
			must(t, os.Remove(tp.at("/usr/bin/servitor")))
			tp.scroll("/usr/bin/servitor", "\x7fELF", 0o755)
			return self
		}, want: errAny},
		"cast aside, another laid at the name the machine shows": {self: func(t *testing.T, tp *temple) string {
			self := held(t, tp.at("/usr/bin/servitor"))
			must(t, os.Remove(tp.at("/usr/bin/servitor")))
			tp.scroll("/usr/bin/servitor (deleted)", "\x7fELF", 0o755)
			return self
		}, want: errNotSelf},
		"outside the ground": {self: func(t *testing.T, tp *temple) string {
			p := filepath.Join(t.TempDir(), "servitor")
			must(t, os.WriteFile(p, []byte("\x7fELF"), 0o755))
			return held(t, p)
		}, want: errAstray},
		"missing": {self: func(t *testing.T, tp *temple) string { return filepath.Join(t.TempDir(), "absent") },
			want: errAny},
	} {
		t.Run(what, func(t *testing.T) {
			tp := sanctuary(t)
			err := tp.g.EnshrinedSelf(c.self(t, tp))
			switch {
			case c.want == nil && err != nil:
				t.Fatalf("refused with %v", err)
			case c.want == errAny && err == nil, c.want != nil && c.want != errAny && !errors.Is(err, c.want):
				t.Fatalf("judged with %v, want %v", err, c.want)
			}
		})
	}
}

// errAny stands for any refusal in the trials.
var errAny = errors.New("any refusal")

// TestMachine_EnshrinesOnlyTheRootsOwn: upon the machine itself, read
// only. The trials' own binary lies beneath halls others may write; a
// scroll the machine holds as the root's own stands enshrined, and so does
// the installed servitor when the machine holds one, exactly when its owner
// and seals and those of every hall above it say so.
func TestMachine_EnshrinesOnlyTheRootsOwn(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if os.Geteuid() != 0 {
		if err := Machine.Enshrined(exe); err == nil {
			t.Errorf("the trials' own binary %s stands enshrined", exe)
		}
		if err := Machine.EnshrinedSelf("/proc/self/exe"); err == nil {
			t.Errorf("the trials' own binary stands enshrined as itself")
		}
	}
	for _, path := range []string{"/usr/bin/env", "/usr/bin/servitor"} {
		want, absent := rootsOwn(t, path)
		err := Machine.Enshrined(path)
		switch {
		case absent && !errors.Is(err, fs.ErrNotExist):
			t.Errorf("%s stands nowhere, yet was judged with %v", path, err)
		case !absent && want && err != nil:
			t.Errorf("%s is the root's own, yet was judged with %v", path, err)
		case !absent && !want && err == nil:
			t.Errorf("%s is not the root's own, yet stands enshrined", path)
		}
	}
}

// rootsOwn judges path upon the machine by its stats alone: the scroll and
// every hall above it belong to the root and none but it may write them.
func rootsOwn(t *testing.T, path string) (own, absent bool) {
	real, err := filepath.EvalSymlinks(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, true
	}
	if err != nil {
		t.Fatal(err)
	}
	for p := real; ; p = filepath.Dir(p) {
		var st unix.Stat_t
		if err := unix.Lstat(p, &st); err != nil {
			t.Fatal(err)
		}
		if st.Uid != 0 || st.Mode&0o022 != 0 {
			return false, false
		}
		if p == "/" {
			break
		}
	}
	for p := filepath.Dir(path); p != "/"; p = filepath.Dir(p) {
		var st unix.Stat_t
		if err := unix.Lstat(p, &st); err != nil {
			t.Fatal(err)
		}
		if st.Mode&unix.S_IFMT == unix.S_IFDIR && (st.Uid != 0 || st.Mode&0o022 != 0) {
			return false, false
		}
	}
	return true, false
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(fmt.Sprint(err))
	}
}
