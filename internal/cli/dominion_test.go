package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/dominion"
)

// heard is an output the test may read while the servitor still speaks.
type heard struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (h *heard) Write(p []byte) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.b.Write(p)
}

func (h *heard) String() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.b.String()
}

// started is a servitor that speaks in a goroutine of its own.
type started struct {
	stdout, stderr heard
	code           int
	done           chan struct{}
}

func (e *env) start(args ...string) *started {
	s := &started{done: make(chan struct{})}
	go func() {
		defer close(s.done)
		s.code = Execute(append([]string{"--librarium", e.cfgDir}, args...), &s.stdout, &s.stderr)
	}()
	return s
}

// ended waits until the servitor has spoken its last.
func (s *started) ended(t *testing.T) {
	t.Helper()
	select {
	case <-s.done:
	case <-time.After(10 * time.Second):
		t.Fatalf("the servitor never ended\nstdout: %s\nstderr: %s", s.stdout.String(), s.stderr.String())
	}
}

// until waits until told holds.
func until(t *testing.T, what string, told func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !told() {
		if time.Now().After(deadline) {
			t.Fatal(what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// hold takes dominion over the rite as another invocation would; the
// returned relinquish lets it go.
func (e *env) hold(rite string) (relinquish func()) {
	e.t.Helper()
	d, err := dominion.Seize(context.Background(), []string{dominion.Rite(os.LookupEnv, rite)}, nil)
	if err != nil {
		e.t.Fatal(err)
	}
	e.t.Cleanup(d.Relinquish)
	return d.Relinquish
}

// awaitingLine matches the line that tells the awaiting of dominion; a
// progress is told indented beneath the header.
var awaitingLine = regexp.MustCompile(`(?m)^⋯ .*$`)

func (e *env) voxNotifySend() {
	e.addRite("servitor.json", `{"pattern": "Mark II", "vox": "notify-send"}`)
}

// TestInvoke_TheAwaitingOfDominionIsToldOnceBeforeTheHeader: upon a
// terminal the invocation tells, once and before its header, that it waits
// for dominion; then it performs. Binharic tells it upon stderr; silence
// withholds it.
func TestInvoke_TheAwaitingOfDominionIsToldOnceBeforeTheHeader(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		edict   bool
		onError bool // the line is told upon stderr
		silence bool
	}{
		{"rite", []string{"invoke", "mouse-autohide-toggle", "on"}, false, false, false},
		{"edict", []string{"invoke", "capture"}, true, false, false},
		{"binharic", []string{"invoke", "mouse-autohide-toggle", "on", "--binharic"}, false, true, false},
		{"silence", []string{"invoke", "mouse-autohide-toggle", "on", "--silence"}, false, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newEnv(t)
			e.terminal(80)
			e.addRite("rites/capture.json", edictJSON)
			e.voxNotifySend() // the desktop hears the awaiting even in silence
			relinquish := e.hold(c.args[1])
			s := e.start(c.args...)
			// Nothing else reaches the desktop before the dominion is held.
			until(t, "the awaiting never reached the desktop", func() bool { return e.notified() != "" })
			select {
			case <-s.done:
				t.Fatalf("the invocation did not wait for dominion\nstdout: %s", s.stdout.String())
			case <-time.After(50 * time.Millisecond):
			}
			relinquish()
			s.ended(t)
			if s.code != 0 {
				t.Fatalf("exit %d\nstdout: %s\nstderr: %s", s.code, s.stdout.String(), s.stderr.String())
			}

			told, other := s.stdout.String(), s.stderr.String()
			if c.onError {
				told, other = other, told
			}
			lines := awaitingLine.FindAllString(told, -1)
			switch {
			case c.silence && len(lines)+len(awaitingLine.FindAllString(other, -1)) != 0:
				t.Fatalf("silence told the awaiting:\n%s", told)
			case c.silence:
				return
			case len(lines) != 1:
				t.Fatalf("the awaiting was told %d times:\n%s", len(lines), told)
			case awaitingLine.MatchString(other):
				t.Fatalf("the awaiting was told upon the wrong stream:\n%s", other)
			}
			header := "+++ " + c.args[1]
			if !strings.HasPrefix(told, lines[0]+"\n"+header) {
				t.Fatalf("the awaiting is not told right before the header:\n%s", told)
			}
			if c.edict != strings.Contains(strings.ToLower(lines[0]), "the edict capture") ||
				!strings.Contains(lines[0], c.args[1]) {
				t.Fatalf("the awaiting names otherwise: %q", lines[0])
			}
		})
	}
}

// TestInvoke_TheAwaitingOfDominionIsTheDesktopsOneMissive: without a
// terminal the awaiting is the invocation's one desktop missive, replaced
// in place by the triumph, and never told in the report.
func TestInvoke_TheAwaitingOfDominionIsTheDesktopsOneMissive(t *testing.T) {
	e := newEnv(t)
	relinquish := e.hold("mouse-autohide-toggle")
	s := e.start("invoke", "mouse-autohide-toggle", "on")
	until(t, "the awaiting never reached the desktop", func() bool { return e.notified() != "" })
	relinquish()
	s.ended(t)
	if s.code != 0 {
		t.Fatalf("exit %d\nstderr: %s", s.code, s.stderr.String())
	}
	calls := strings.Split(strings.TrimSpace(e.notified()), "\n")
	if len(calls) != 2 || strings.Contains(calls[0], "-r") || !strings.Contains(calls[0], "dominion") ||
		!strings.Contains(calls[1], "-r 7") {
		t.Fatalf("the desktop heard:\n%s", e.notified())
	}
	if strings.Contains(s.stdout.String(), "⋯") || !strings.HasPrefix(s.stdout.String(), "+++ mouse-autohide-toggle") {
		t.Fatalf("the report:\n%s", s.stdout.String())
	}
}

// TestInvoke_AnUncontendedInvocationTellsNoAwaiting: dominion free at once
// is never spoken of, upon the terminal or the desktop.
func TestInvoke_AnUncontendedInvocationTellsNoAwaiting(t *testing.T) {
	e := newEnv(t)
	e.terminal(80)
	e.voxNotifySend()
	out := e.mustRun("invoke", "mouse-autohide-toggle", "on")
	if !strings.HasPrefix(out, "+++ mouse-autohide-toggle") || strings.Contains(out, "⋯") {
		t.Fatalf("the report:\n%s", out)
	}
	if n := strings.Count(e.notified(), "\n"); n != 1 || strings.Contains(e.notified(), "dominion") {
		t.Fatalf("the desktop heard:\n%s", e.notified())
	}
	if _, err := os.Stat(filepath.Join(e.state, "servitor", "dominions", "rites", "mouse-autohide-toggle")); err != nil {
		t.Fatalf("no dominion was held: %v", err)
	}
}

// TestRituals_ThatTouchNothingNeverAwaitDominion: foresight, augury, census
// and the inquisition read while another holds dominion over the rite.
func TestRituals_ThatTouchNothingNeverAwaitDominion(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", edictJSON)
	e.hold("mouse-autohide-toggle")
	e.hold("capture")
	for _, args := range [][]string{
		{"invoke", "mouse-autohide-toggle", "on", "--foresee"}, {"invoke", "capture", "--foresee"},
		{"augury", "mouse-autohide-toggle"}, {"augury", "capture"}, {"census"}, {"census", "--binharic"},
		{"inquisition"}, {"inquisition", "mouse-autohide-toggle"},
	} {
		s := e.start(args...)
		s.ended(t)
		if s.code != 0 || strings.Contains(s.stdout.String()+s.stderr.String(), "dominion") {
			t.Errorf("servitor %v: exit %d\nstdout: %s\nstderr: %s", args, s.code, s.stdout.String(), s.stderr.String())
		}
	}
}
