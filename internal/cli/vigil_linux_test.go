package cli

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// sloths writes n rites whose auspex labours long beside a child of its
// own, each telling its process and its child's in the returned file.
func (e *env) sloths(n int) (pids string) {
	e.t.Helper()
	pids = filepath.Join(e.state, "pids")
	for i := range n {
		e.addRite(fmt.Sprintf("rites/sloth%d.json", i), `{"pattern": "Mark I", "aspects": ["on", "off"],
		  "auspex": {"rite": "sleep 30 & echo $$ $! >> `+pids+`; wait", "patience": "60s"}, "liturgy": []}`)
	}
	return pids
}

// awake returns the processes the auspexes told in pids, once there are n.
func awake(t *testing.T, pids string, n int) []int {
	t.Helper()
	var told []int
	within(t, func() bool {
		b, _ := os.ReadFile(pids)
		told = told[:0]
		for _, f := range strings.Fields(string(b)) {
			pid, _ := strconv.Atoi(f)
			told = append(told, pid)
		}
		return len(told) >= n
	}, "the auspexes never awoke")
	return told
}

// TestVigil_ADepartedListenerEndsTheVigil: a write finding no reader ends
// the vigil at once: it slays every auspex still awake, with its process
// group, tells nothing more and laments nothing.
func TestVigil_ADepartedListenerEndsTheVigil(t *testing.T) {
	e := newEnv(t)
	pids := e.sloths(2)
	e.addRite("rites/hasty.json", `{"pattern": "Mark I", "aspects": ["on", "off"], "auspex": "sleep 0.3; echo on", "liturgy": []}`)
	l := &listener{t: t, bear: 4} // the first telling only
	var errb bytes.Buffer

	start := time.Now()
	Execute([]string{"--librarium", e.cfgDir, "census", "--vigil"}, l, &errb)

	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("the vigil outlasted its listener by %v", elapsed)
	}
	if errb.Len() > 0 || len(l.lines) != 4 {
		t.Errorf("stderr %q; lines %v", errb.String(), l.lines)
	}
	for _, pid := range awake(t, pids, 4) {
		within(t, func() bool { return ended(pid) }, fmt.Sprintf("process %d of an auspex outlived the vigil", pid))
	}
}

// TestVigil_AClosedStdoutEndsTheVigil: the reader of the vigil closes its
// end before the conclusion, while auspexes still labour and nothing more
// is told; the servitor slays them with their process groups at once and
// ends, struck down by nothing, without a word upon stderr.
func TestVigil_AClosedStdoutEndsTheVigil(t *testing.T) {
	e := newEnv(t)
	pids := e.sloths(2)
	cmd := e.servitorProcess("census", "--vigil")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errs, errW := pipe(t)
	cmd.Stdout, cmd.Stderr = w, errW
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	_, _ = w.Close(), errW.Close()
	lines := bufio.NewScanner(r)
	for range 3 {
		if !lines.Scan() {
			t.Fatalf("the vigil ended before its first telling: %v", lines.Err())
		}
	}
	slain := awake(t, pids, 4)

	start := time.Now()
	_ = r.Close()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err = <-done:
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("the vigil outlasted the closing of its stdout")
	}
	t.Logf("ended %v after its stdout was closed", time.Since(start))
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		if ws := exit.Sys().(syscall.WaitStatus); ws.Signaled() {
			t.Errorf("the servitor was struck down by %v", ws.Signal())
		}
	}
	errs.depart()
	if errs.heard.Len() > 0 {
		t.Errorf("stderr: %q", errs.heard.String())
	}
	for _, pid := range slain {
		within(t, func() bool { return ended(pid) }, fmt.Sprintf("process %d of an auspex outlived the vigil", pid))
	}
}
