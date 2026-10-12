package intercessor

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/hallowed"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// awakenAs makes the trials' binary awaken as a process of its own that
// tells whether a terminal attends it.
const awakenAs = "SERVITOR_TEST_AWAKEN_AS"

func TestMain(m *testing.M) {
	if os.Getenv(awakenAs) == "attended" {
		fmt.Print(attended())
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// record is where a stub intercessor records the words it was called with.
const record = "SERVITOR_TRIAL_RECORD"

// stub is an intercessor of the trials: it records its argv, its own path
// first, one word per line of NULs, and vouches for nobody.
const stub = "#!/bin/sh\nprintf '%s\\0' \"$0\" \"$@\" > \"$" + record + "\"\n"

const (
	sigil  = "4f0c2a9e1b7d55aa0123456789abcdef0123456789abcdef0123456789abcdef"
	rite   = "split-lock"
	aspect = "on"
)

// shrine is a ground of the trials' own holding a servitor installed as
// the root's, the invoker of the trials standing in for the root.
func shrine(t *testing.T) hallowed.Ground {
	t.Helper()
	root := t.TempDir()
	must(t, os.Chmod(root, 0o755))
	must(t, os.MkdirAll(filepath.Join(root, "usr/bin"), 0o755))
	must(t, os.WriteFile(filepath.Join(root, Servitor), []byte("\x7fELF"), 0o755))
	me := uint32(os.Getuid())
	return hallowed.Ground{Root: root, Sovereign: func(uid uint32) bool { return uid == me }}
}

// world is a world of the trials: the intercessors present stand as stubs
// upon a PATH of their own, and a terminal attends it or not.
func world(t *testing.T, attended bool, present ...string) (World, string) {
	t.Helper()
	bin := t.TempDir()
	for _, name := range present {
		must(t, os.WriteFile(filepath.Join(bin, name), []byte(stub), 0o755))
	}
	t.Setenv("PATH", bin)
	return World{LookPath: exec.LookPath, Attended: func() bool { return attended }, Ground: shrine(t),
		Servitor: Servitor}, bin
}

// called calls upon the intercessor as the invocation would, and returns
// the argv the stub recorded.
func called(t *testing.T, cmd *exec.Cmd) []string {
	t.Helper()
	rec := filepath.Join(t.TempDir(), "argv")
	cmd.Env = append(os.Environ(), record+"="+rec)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the stub fell: %v: %s", err, out)
	}
	data, err := os.ReadFile(rec)
	must(t, err)
	return strings.Split(strings.TrimSuffix(string(data), "\x00"), "\x00")
}

// argv is the exact argv each intercessor is called with to raise the
// exalted servitor for the aspect (none for an edict).
func argv(name, aspect string) []string {
	tail := []string{"/usr/bin/servitor", "__exalt", rite, "4f0c2a9e1b7d"}
	if aspect != "" {
		tail = append(tail, aspect)
	}
	switch name {
	case "sudo":
		return append([]string{"-k", "-p", "servitor: authenticate to perform split-lock (sigil 4f0c2a9e1b7d): ",
			"--"}, tail...)
	case "run0":
		return append([]string{"--unit=servitor-split-lock-4f0c2a9e", "--pipe", "--chdir=/"}, tail...)
	}
	return tail
}

// TestFind_ChoosesByTheSettingsAndTheTerminal: "auto" (and the settings
// unwritten) calls upon sudo, else run0, else pkexec when a terminal
// attends, and upon run0, else pkexec, when none does; a value of its own
// calls upon that one alone. Every intercessor is called with its exact
// argv; one that is nowhere to be found refuses.
func TestFind_ChoosesByTheSettingsAndTheTerminal(t *testing.T) {
	sought := map[bool][]string{true: {"sudo", "run0", "pkexec"}, false: {"run0", "pkexec"}}
	everyone := []string{"sudo", "run0", "pkexec"}
	for mask := range 8 {
		var present []string
		for i, name := range everyone {
			if mask&(1<<i) != 0 {
				present = append(present, name)
			}
		}
		for _, attended := range []bool{true, false} {
			for _, setting := range []string{"", librarium.IntercessorAuto, librarium.IntercessorSudo,
				librarium.IntercessorRun0, librarium.IntercessorPkexec} {
				want := ""
				switch setting {
				case "", librarium.IntercessorAuto:
					for _, name := range sought[attended] {
						if slices.Contains(present, name) {
							want = name
							break
						}
					}
				default:
					if slices.Contains(present, setting) {
						want = setting
					}
				}
				what := fmt.Sprintf("%v attended=%v setting=%q", present, attended, setting)
				t.Run(what, func(t *testing.T) {
					w, bin := world(t, attended, present...)
					i, err := w.Find(setting)
					if want == "" {
						var absent *Absent
						if !errors.As(err, &absent) {
							t.Fatalf("found %+v, %v; want none", i, err)
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					if i.Name != want || i.Path != filepath.Join(bin, want) {
						t.Fatalf("found %s at %s, want %s", i.Name, i.Path, want)
					}
					for _, a := range []string{aspect, ""} {
						got := called(t, i.Command(rite, sigil, a))
						if exp := append([]string{i.Path}, argv(want, a)...); !slices.Equal(got, exp) {
							t.Errorf("aspect %q: called as\n%q, want\n%q", a, got, exp)
						}
					}
				})
			}
		}
	}
}

// TestFind_RefusesWithoutAnEnshrinedServitor: the exalted servitor is
// always /usr/bin/servitor, which must stand where only the root writes,
// it and every hall above it, before any intercessor is sought.
func TestFind_RefusesWithoutAnEnshrinedServitor(t *testing.T) {
	for what, c := range map[string]struct {
		mar  func(t *testing.T, root string)
		want error
	}{
		"missing": {mar: func(t *testing.T, root string) { must(t, os.Remove(filepath.Join(root, Servitor))) },
			want: fs.ErrNotExist},
		"a servitor others may write": {mar: func(t *testing.T, root string) {
			must(t, os.Chmod(filepath.Join(root, Servitor), 0o777))
		}},
		"beneath a hall others may write": {mar: func(t *testing.T, root string) {
			must(t, os.Chmod(filepath.Join(root, "usr"), 0o777))
		}},
	} {
		t.Run(what, func(t *testing.T) {
			w, _ := world(t, true, "sudo", "run0", "pkexec")
			c.mar(t, w.Ground.Root)
			i, err := w.Find(librarium.IntercessorAuto)
			var un *Unenshrined
			if !errors.As(err, &un) || (c.want != nil && !errors.Is(err, c.want)) {
				t.Fatalf("found %+v, %v", i, err)
			}
			if !strings.Contains(err.Error(), "/usr/bin/servitor") || !strings.Contains(err.Error(), "anew") {
				t.Errorf("the lament names neither the servitor nor its remedy: %v", err)
			}
			if err := w.Enshrined(); !errors.As(err, &un) {
				t.Errorf("judged alone: %v", err)
			}
		})
	}
	w, _ := world(t, true)
	if err := w.Enshrined(); err != nil {
		t.Fatalf("an enshrined servitor: %v", err)
	}
}

// TestFind_RefusesAnIntercessorTheCodexDoesNotKnow: doas, or any other
// word, calls upon no one.
func TestFind_RefusesAnIntercessorTheCodexDoesNotKnow(t *testing.T) {
	w, _ := world(t, true, "sudo", "run0", "pkexec", "doas")
	for _, setting := range []string{"doas", "SUDO", "su"} {
		if i, err := w.Find(setting); err == nil {
			t.Errorf("%q found %+v", setting, i)
		}
	}
}

// TestAbsent_NamesWhatWasSought: the lament of a missing intercessor names
// every intercessor sought, and the settings key for one of its own.
func TestAbsent_NamesWhatWasSought(t *testing.T) {
	for _, c := range []struct {
		setting  string
		attended bool
		words    []string
	}{
		{"auto", true, []string{"sudo", "run0", "pkexec", `"intercessor"`}},
		{"auto", false, []string{"run0", "pkexec", "terminal", `"intercessor"`}},
		{"run0", true, []string{`"run0"`, `"intercessor"`}},
	} {
		w, _ := world(t, c.attended)
		_, err := w.Find(c.setting)
		for _, word := range c.words {
			if err == nil || !strings.Contains(err.Error(), word) {
				t.Errorf("%s attended=%v: %v does not name %s", c.setting, c.attended, err, word)
			}
		}
	}
}

// TestMachine_AttendedTellsWhetherATerminalAttends: a process in a session
// of its own has no terminal; /dev/tty does not open for it.
func TestMachine_AttendedTellsWhetherATerminalAttends(t *testing.T) {
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), awakenAs+"=attended")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "false" {
		t.Fatalf("a process without a terminal is attended: %s", out)
	}
	if w := Machine(); w.Servitor != "/usr/bin/servitor" || w.Ground.Root != "/" || w.LookPath == nil ||
		w.Attended == nil {
		t.Fatalf("the machine's world is %+v", w)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
