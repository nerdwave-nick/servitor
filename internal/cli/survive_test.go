package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// gate is an incantation that marks that it was spoken, then waits until
// the test lets it pass by creating the release.
func gate(marker, release string) string {
	return "touch " + marker + "; while [ ! -e " + release + " ]; do sleep 0.01; done"
}

// lingering is a rite (or, with edict, an edict) whose liturgy halts at a
// gate between two deeds, so that a test may strike at it mid-liturgy.
// Its second deed leaves $TARGET.after.
func (e *env) lingering(name string, edict bool) (marker, release string) {
	e.t.Helper()
	marker, release = filepath.Join(e.state, name+".spoken"), filepath.Join(e.state, name+".release")
	if edict {
		e.addRite("rites/"+name+".json", `{"pattern": "Mark I", "edict": true, "liturgy": [
		  {"incantation": "echo before >> $TARGET.before"},
		  {"incantation": "`+gate(marker, release)+`"},
		  {"incantation": "echo after >> $TARGET.after"}]}`)
		return marker, release
	}
	e.addRite("rites/"+name+".json", `{"pattern": "Mark I", "aspects": ["on"], "liturgy": [
	  {"sanctum": "$TARGET", "ward": "`+name+`", "scripture": "x"},
	  {"incantation": "`+gate(marker, release)+`"},
	  {"transcription": "$TARGET.after", "scripture": "after"}]}`)
	return marker, release
}

// await waits until path exists.
func await(t *testing.T, path, what string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for _, err := os.Stat(path); err != nil; _, err = os.Stat(path) {
		if time.Now().After(deadline) {
			t.Fatal(what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestInvoke_AHangupIsNotHeeded: the terminal of the one who invoked hangs
// up mid-liturgy; the invocation is no request to undo, and the rite or
// edict is performed whole and chronicled.
func TestInvoke_AHangupIsNotHeeded(t *testing.T) {
	for _, edict := range []bool{false, true} {
		e := newEnv(t)
		marker, release := e.lingering("lingering", edict)
		args := []string{"-l", e.cfgDir, "invoke", "lingering", "on"}
		if edict {
			args = args[:len(args)-1]
		}
		type result struct {
			out, errOut string
			code        int
		}
		done := make(chan result, 1)
		go func() {
			var out, errb bytes.Buffer
			code := Execute(args, &out, &errb)
			done <- result{out.String(), errb.String(), code}
		}()
		await(t, marker, "the gate was never reached")
		if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
			t.Fatal(err)
		}
		time.Sleep(50 * time.Millisecond) // a heeded hangup would end the liturgy here
		touch(t, release)
		select {
		case r := <-done:
			if r.code != 0 || !strings.Contains(r.out, "✠ ") {
				t.Fatalf("edict=%v: code=%d stdout=%s stderr=%s", edict, r.code, r.out, r.errOut)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("the invocation never ended")
		}
		if b, err := os.ReadFile(e.target + ".after"); err != nil || len(b) == 0 {
			t.Fatalf("edict=%v: the deed after the gate was not performed: %v", edict, err)
		}
		if lines := e.chronicleLines(); len(lines) != 1 || !strings.Contains(lines[0], `"verdict":"triumph"`) {
			t.Fatalf("edict=%v: chronicle %q", edict, lines)
		}
		want := "performed\n"
		if edict {
			want = "proclaimed\n"
		}
		if got := e.mustRun("augury", "lingering", "standing"); got != want {
			t.Fatalf("edict=%v: the data-slate tells %q", edict, got)
		}
	}
}

// TestInvoke_ASteadfastInvocationStillHeedsInterruptAndTerm: steadfast
// against its caller's departure, an invocation is still halted and
// reverted by an interrupt or SIGTERM.
func TestInvoke_ASteadfastInvocationStillHeedsInterruptAndTerm(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		e := newEnv(t)
		marker, _ := e.lingering("lingering", false)
		done := make(chan int, 1)
		var errb bytes.Buffer
		go func() {
			done <- Execute([]string{"-l", e.cfgDir, "invoke", "lingering", "on", "--binharic"}, io.Discard, &errb)
		}()
		await(t, marker, "the gate was never reached")
		if err := syscall.Kill(os.Getpid(), sig); err != nil {
			t.Fatal(err)
		}
		select {
		case code := <-done:
			if code != 1 || !strings.Contains(errb.String(), "halted") {
				t.Fatalf("%v: code=%d stderr=%s", sig, code, errb.String())
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("%v was not heeded", sig)
		}
		if e.targetContent() != "input {}\n" {
			t.Fatalf("%v: not reverted:\n%s", sig, e.targetContent())
		}
		if lines := e.chronicleLines(); len(lines) != 1 || !strings.Contains(lines[0], `"verdict":"reverted"`) {
			t.Fatalf("%v: chronicle %q", sig, lines)
		}
	}
}
