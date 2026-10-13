package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
)

// awakenAs is the environment variable that makes the test binary awaken
// as a process of its own: as the servitor itself (its arguments are the
// servitor's), or as a caller who invokes it upon a terminal (beCaller).
const awakenAs = "SERVITOR_TEST_AWAKEN_AS"

func TestMain(m *testing.M) {
	// The servitor serves the faithful in every test, whoever runs them;
	// asRoot lends it the machine's own authority.
	geteuid = func() int { return 1000 }
	tearDown := enshrineTrials()
	switch os.Getenv(awakenAs) {
	case "servitor":
		raiseAltarOfTheProcess()
		os.Exit(Execute(os.Args[1:], os.Stdout, os.Stderr))
	case "exalted":
		os.Exit(serveAsExalted(os.Args[1:]))
	case "caller":
		os.Exit(beCaller(os.Args[1:]))
	}
	code := m.Run()
	tearDown()
	os.Exit(code)
}

// servitorProcess prepares the servitor as a process of its own upon the
// test Librarium, in a session of its own without a terminal.
func (e *env) servitorProcess(args ...string) *exec.Cmd {
	cmd := exec.Command(os.Args[0], append([]string{"--librarium", e.cfgDir}, args...)...)
	cmd.Env = append(os.Environ(), awakenAs+"=servitor")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd
}

// stream is one of the servitor's outputs as its reader holds it.
type stream struct {
	r     *os.File
	heard bytes.Buffer
	done  chan struct{}
}

// pipe gives the servitor a pipe for an output; its reader listens until
// it departs.
func pipe(t *testing.T) (*stream, *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	s := &stream{r: r, done: make(chan struct{})}
	go func() {
		defer close(s.done)
		_, _ = io.Copy(&s.heard, r)
	}()
	return s, w
}

// depart closes the reading end: whatever the servitor writes later finds
// no reader.
func (s *stream) depart() { _ = s.r.Close(); <-s.done }

// TestInvoke_ADepartedReaderNeverHalfPerformsTheRite: the reader of stdout,
// of stderr or of both departs mid-invocation; the servitor performs or
// reverts the rite whole, chronicles the verdict and bears its death-mark.
func TestInvoke_ADepartedReaderNeverHalfPerformsTheRite(t *testing.T) {
	cases := []struct {
		name           string
		args           []string
		edict, falls   bool
		stdout, stderr bool // which readers depart
		code           int
		verdict        string
	}{
		{"triumph, stdout departs", []string{"on"}, false, false, true, false, 0, "triumph"},
		{"triumph in binharic, both depart", []string{"on", "--binharic"}, false, false, true, true, 0, "triumph"},
		{"edict, both depart", nil, true, false, true, true, 0, "triumph"},
		{"reverted, stderr departs", []string{"on"}, false, true, false, true, 1, "reverted"},
		{"reverted in binharic, both depart", []string{"on", "--binharic"}, false, true, true, true, 1, "reverted"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newEnv(t)
			marker, release := e.lingering("lingering", c.edict)
			if c.falls {
				e.addRite("rites/lingering.json", `{"pattern": "Mark II", "aspects": ["on"], "subjects": [
				  {"name": "warded", "designation": "sanctum", "vessel": "$TARGET", "ward": "lingering", "omens": {"*": "x"},
				   "introit": ["gate"], "recessional": ["dying"]}],
				  "observances": [{"name": "gate", "designation": "incantation", "words": "`+gate(marker, release)+`"},
				    {"name": "dying", "designation": "incantation", "words": "echo dying words; exit 3"}]}`)
			}
			cmd := e.servitorProcess(append([]string{"invoke", "lingering"}, c.args...)...)
			out, outW := pipe(t)
			errs, errW := pipe(t)
			cmd.Stdout, cmd.Stderr = outW, errW
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			_, _ = outW.Close(), errW.Close()
			await(t, marker, "the gate was never reached")
			if c.stdout {
				out.depart()
			}
			if c.stderr {
				errs.depart()
			}
			touch(t, release)
			err := cmd.Wait()
			var exit *exec.ExitError
			code := 0
			if errors.As(err, &exit) {
				if ws := exit.Sys().(syscall.WaitStatus); ws.Signaled() {
					t.Fatalf("the servitor was struck down by %v", ws.Signal())
				}
				code = exit.ExitCode()
			} else if err != nil {
				t.Fatal(err)
			}
			if code != c.code {
				t.Errorf("death-mark %d, want %d\nstdout: %s\nstderr: %s", code, c.code, &out.heard, &errs.heard)
			}
			if lines := e.chronicleLines(); len(lines) != 1 || !strings.Contains(lines[0], `"verdict":"`+c.verdict+`"`) {
				t.Fatalf("chronicle %q", lines)
			}
			wholeOrNothing(t, e, c.edict, c.falls)
		})
	}
}

// wholeOrNothing fails unless the lingering rite was performed whole, or,
// when it fell, wholly undone.
func wholeOrNothing(t *testing.T, e *env, edict, fell bool) {
	t.Helper()
	_, err := os.Stat(e.target + ".after")
	switch {
	case fell && e.targetContent() != "input {}\n":
		t.Fatalf("the fallen rite was not undone:\n%s", e.targetContent())
	case !fell && err != nil:
		t.Fatalf("the deed after the gate was not performed: %v", err)
	case !fell && !edict && !strings.Contains(e.targetContent(), "+++ begin of sanctum lingering"):
		t.Fatalf("the sanctum was not kept:\n%s", e.targetContent())
	}
}
