package hallowed

import (
	"errors"
	"io/fs"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestProgram_IsAScrollOfTheRootThroughHallowedHalls: the program of a
// tongue is reached through halls that pass the rule, its bonds followed;
// it is a scroll the root owns, that none but the root may write and that
// the machine may awaken. Anything else is refused.
func TestProgram_IsAScrollOfTheRootThroughHallowedHalls(t *testing.T) {
	tp := newTemple(t)
	tp.hall("/usr/bin", 0o755)
	tp.scroll("/usr/bin/bash", "#!", 0o755)
	tp.bond("/bin", "usr/bin")
	tp.bond("/usr/bin/sh", "bash")
	for path, want := range map[string]string{"/usr/bin/bash": "/usr/bin/bash", "/bin/sh": "/usr/bin/bash"} {
		if got, err := tp.g.Program(path); err != nil || got != want {
			t.Errorf("the program %s is %q, %v; want %s", path, got, err, want)
		}
	}

	tp.hall("/opt/open", 0o777)
	tp.scroll("/opt/open/bash", "#!", 0o755)
	tp.scroll("/usr/bin/open", "#!", 0o775)
	tp.scroll("/usr/bin/inert", "#!", 0o644)
	tp.hall("/usr/bin/hall", 0o755)
	for path, want := range map[string]error{
		"bash":           errAstray,
		"/opt/open/bash": errOpenHall,
		"/usr/bin/open":  errOpenProgram,
		"/usr/bin/inert": errInert,
		"/usr/bin/hall":  errNoScroll,
		"/usr/bin/none":  fs.ErrNotExist,
	} {
		if got, err := tp.g.Program(path); !errors.Is(err, want) {
			t.Errorf("the program %s: %q, %v; want %v", path, got, err, want)
		}
	}

	strange := tp.g
	strange.Sovereign = func(uint32) bool { return false }
	if _, err := strange.Program("/usr/bin/bash"); !errors.Is(err, errUnsovereign) && !errors.Is(err, errAlien) {
		t.Errorf("a program of a stranger: %v", err)
	}
}

// TestProgram_JudgesTheRealTongue: upon the machine itself, the bash the
// machine speaks is a program of the root; one in the temporary halls is
// not.
func TestProgram_JudgesTheRealTongue(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("no bash upon this machine")
	}
	bash, _ = filepath.EvalSymlinks(bash)
	if got, err := Machine.Program(bash); err != nil || got != bash {
		t.Fatalf("the program %s is %q, %v", bash, got, err)
	}
	tp := newTemple(t)
	tp.scroll("/bash", "#!", 0o755)
	if _, err := Machine.Program(tp.at("/bash")); err == nil {
		t.Error("a program in the temporary halls was held the root's")
	}
}
