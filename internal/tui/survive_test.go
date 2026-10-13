package tui

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// lingeringRite halts at a gate between two subjects until $DATA/release
// exists, so that a test may strike at it mid-invocation: the introit of
// its second subject.
const lingeringRite = `{
  "pattern": "Mark II",
  "aspects": ["on", "off"],
  "subjects": [
    {"name": "x", "designation": "sanctum", "vessel": "$DATA/x.kdl", "ward": "lingering", "omens": {"*": "x"}},
    {"name": "after", "designation": "transcription", "vessel": "$DATA/after", "omens": {"*": "after"},
     "introit": ["gate"]}
  ],
  "observances": [
    {"name": "gate", "designation": "incantation",
     "words": "touch $DATA/spoken; while [ ! -e $DATA/release ]; do sleep 0.01; done"}
  ]
}`

// lingering makes the rite mouse linger at its gate; the returned release
// lets it pass.
func (h *harness) lingering() (release func()) {
	h.t.Helper()
	h.writeRite("mouse", lingeringRite)
	path := filepath.Join(h.dataDir, "release")
	release = func() { _ = os.WriteFile(path, nil, 0o644) }
	h.t.Cleanup(release) // no incantation outlives its test
	return release
}

// atTheGate waits until the lingering rite reached its gate.
func (h *harness) atTheGate() {
	h.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for _, err := os.Stat(filepath.Join(h.dataDir, "spoken")); err != nil; _, err = os.Stat(filepath.Join(h.dataDir, "spoken")) {
		if time.Now().After(deadline) {
			h.t.Fatal("the gate was never reached")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// chronicled is every line of the chronicle.
func (h *harness) chronicled() string {
	b, _ := os.ReadFile(filepath.Join(os.Getenv("XDG_STATE_HOME"), "servitor", "chronicle.jsonl"))
	return string(b)
}

// performedWhole fails unless the lingering rite was performed whole and
// its triumph chronicled.
func (h *harness) performedWhole() {
	h.t.Helper()
	if !strings.Contains(h.readData("x.kdl"), "+++ begin of sanctum lingering") || h.readData("after") != "after" {
		h.t.Fatalf("the rite was not performed whole: %q", h.readData("x.kdl"))
	}
	if c := h.chronicled(); strings.Count(c, "\n") != 1 || !strings.Contains(c, `"verdict":"triumph"`) {
		h.t.Fatalf("chronicle %q", c)
	}
}

// TestInvoke_TheCogitatorHeedsNoHangup: the terminal hangs up while the
// cogitator performs a rite; the rite is performed whole.
func TestInvoke_TheCogitatorHeedsNoHangup(t *testing.T) {
	h := newHarness(t)
	release := h.lingering()
	h.keys("r", "space")
	h.atTheGate()
	if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond) // a heeded hangup would end the cogitator here
	release()
	h.await()
	h.mustShow("The rite mouse is performed")
	h.performedWhole()
}

// awaken runs the cogitator upon the harness's Librarium, its keys spoken
// through the returned writer; the channel tells when it has ended.
func (h *harness) awaken() (keys io.WriteCloser, ended chan error) {
	in, keys := io.Pipe()
	ended = make(chan error, 1)
	go func() { ended <- Run(Options{Dir: h.dir, Input: in, Output: io.Discard}) }()
	return keys, ended
}

// TestRun_AHangupEndsTheCogitatorOnceTheRiteIsPerformed: the terminal hangs
// up mid-liturgy; the cogitator performs the rite whole, and only then
// ends, as a hangup would have ended it.
func TestRun_AHangupEndsTheCogitatorOnceTheRiteIsPerformed(t *testing.T) {
	h := newHarness(t)
	release := h.lingering()
	keys, ended := h.awaken()
	defer func() { _ = keys.Close() }()
	if _, err := keys.Write([]byte(" ")); err != nil { // cycle mouse into its next aspect
		t.Fatal(err)
	}
	h.atTheGate()
	if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ended:
		t.Fatal("the cogitator ended while its rite was still performed")
	case <-time.After(300 * time.Millisecond):
	}
	release()
	select {
	case <-ended:
	case <-time.After(10 * time.Second):
		t.Fatal("the cogitator lingers after the hangup")
	}
	h.performedWhole()
}

// TestRun_TermHaltsAndRevertsTheRunningRite: SIGTERM ends the cogitator,
// but only once the running rite is halted and reverted.
func TestRun_TermHaltsAndRevertsTheRunningRite(t *testing.T) {
	h := newHarness(t)
	h.lingering()
	keys, ended := h.awaken()
	defer func() { _ = keys.Close() }()
	if _, err := keys.Write([]byte(" ")); err != nil {
		t.Fatal(err)
	}
	h.atTheGate()
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ended:
	case <-time.After(10 * time.Second):
		t.Fatal("SIGTERM was not heeded")
	}
	if got := h.readData("x.kdl"); got != "input {}\n" {
		t.Fatalf("not reverted: %q", got)
	}
	if c := h.chronicled(); strings.Count(c, "\n") != 1 || !strings.Contains(c, `"verdict":"reverted"`) {
		t.Fatalf("chronicle %q", c)
	}
}
