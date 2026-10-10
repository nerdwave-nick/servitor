package vox

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// stub places a notify-send on PATH that records the arguments of every
// call in its own record and prints the id written to <dir>/id, or fails
// when <dir>/fall exists.
type stub struct{ dir string }

func newStub(t *testing.T) stub {
	t.Helper()
	dir := t.TempDir()
	s := stub{dir: dir}
	script := fmt.Sprintf(`#!/bin/sh
d=%q
n=$(ls "$d/calls" | wc -l)
printf '%%s\000' "$@" > "$d/calls/$n"
[ -e "$d/fall" ] && exit 1
cat "$d/id"
`, dir)
	if err := os.MkdirAll(filepath.Join(dir, "calls"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notify-send"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	s.answer(t, "42\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return s
}

func (s stub) answer(t *testing.T, id string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(s.dir, "id"), []byte(id), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (s stub) fall(t *testing.T) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(s.dir, "fall"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

// calls returns the arguments of every call, in order.
func (s stub) calls(t *testing.T) [][]string {
	t.Helper()
	var out [][]string
	for i := 0; ; i++ {
		b, err := os.ReadFile(filepath.Join(s.dir, "calls", fmt.Sprint(i)))
		if os.IsNotExist(err) {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, strings.Split(strings.TrimSuffix(string(b), "\x00"), "\x00"))
	}
}

// flag returns the value following name in args, and whether it is there.
func flag(args []string, name string) (string, bool) {
	i := slices.Index(args, name)
	if i < 0 {
		return "", false
	}
	if i+1 < len(args) {
		return args[i+1], true
	}
	return "", true
}

func progress(step, steps int) invocation.Proclamation {
	return invocation.Proclamation{Tidings: invocation.Progress, Values: values(step, steps, "sanctum", "/x/util.kdl", ""),
		Verse: invocation.Verse{Number: step, Kind: librarium.KindSanctum, Target: "/x/util.kdl"}}
}

func success() invocation.Proclamation {
	return invocation.Proclamation{Tidings: invocation.Success, Values: values(2, 2, "sanctum", "/x/util.kdl", "")}
}

func failure() invocation.Proclamation {
	out := invocation.Outcome{Verdict: invocation.Reverted, Fell: &invocation.Fall{}}
	return invocation.Proclamation{Tidings: invocation.Failure, Values: values(2, 2, "incantation", "exit 3", "woe"), Outcome: &out}
}

// quiet raises a herald that always draws the first template.
func quiet(c Config) *Herald {
	c.Pick = first
	return New(c)
}

func TestHerald_DesktopFirstPrintsTheIdThenReplacesIt(t *testing.T) {
	s := newStub(t)
	h := quiet(Config{Vox: librarium.VoxNotifySend})

	h.Proclaim(progress(1, 2))
	h.Proclaim(progress(2, 2))
	h.Proclaim(success())
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}

	calls := s.calls(t)
	if len(calls) != 3 {
		t.Fatalf("notify-send was called %d times: %q", len(calls), calls)
	}
	if !slices.Contains(calls[0], "-p") {
		t.Errorf("the first vox-cast does not ask for its id: %q", calls[0])
	}
	if _, ok := flag(calls[0], "-r"); ok {
		t.Errorf("the first vox-cast replaces something: %q", calls[0])
	}
	for _, c := range calls[1:] {
		if id, _ := flag(c, "-r"); id != "42" {
			t.Errorf("a later vox-cast does not replace 42: %q", c)
		}
	}
	for i, pct := range []string{"int:value:50", "int:value:100"} {
		if v, _ := flag(calls[i], "-h"); v != pct {
			t.Errorf("progress %d hints %q, want %q: %q", i, v, pct, calls[i])
		}
		if v, _ := flag(calls[i], "-t"); v != "0" {
			t.Errorf("progress %d may expire mid-run: %q", i, calls[i])
		}
	}
	if _, ok := flag(calls[2], "-t"); ok {
		t.Errorf("the triumph never expires: %q", calls[2])
	}
	if _, ok := flag(calls[2], "-h"); ok {
		t.Errorf("the triumph hints progress: %q", calls[2])
	}
	want := Compose(success(), first)
	if got := calls[2][len(calls[2])-2:]; got[0] != want.Summary || got[1] != want.Body {
		t.Errorf("the triumph speaks %q, want %q / %q", got, want.Summary, want.Body)
	}
}

func TestHerald_FailureIsCritical(t *testing.T) {
	s := newStub(t)
	h := quiet(Config{})
	h.Proclaim(failure())
	calls := s.calls(t)
	if len(calls) != 1 {
		t.Fatalf("calls %q", calls)
	}
	if u, _ := flag(calls[0], "-u"); u != "critical" {
		t.Errorf("failure is spoken with urgency %q: %q", u, calls[0])
	}
	if body := calls[0][len(calls[0])-1]; !strings.Contains(body, "woe") || !strings.Contains(body, "undone") {
		t.Errorf("failure body %q", body)
	}
}

func TestHerald_CloseLetsALingeringProgressExpire(t *testing.T) {
	s := newStub(t)
	h := quiet(Config{})
	h.Proclaim(progress(1, 2))
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	calls := s.calls(t)
	if len(calls) != 2 {
		t.Fatalf("calls %q", calls)
	}
	if id, _ := flag(calls[1], "-r"); id != "42" {
		t.Errorf("the resting vox-cast replaces not 42: %q", calls[1])
	}
	if _, ok := flag(calls[1], "-t"); ok {
		t.Errorf("the resting vox-cast never expires: %q", calls[1])
	}
	if !slices.Equal(calls[0][len(calls[0])-2:], calls[1][len(calls[1])-2:]) {
		t.Errorf("the resting vox-cast speaks other words: %q vs %q", calls[0], calls[1])
	}
}

func TestHerald_SendShowsAMessageComposedBefore(t *testing.T) {
	s := newStub(t)
	h := quiet(Config{})
	m := Compose(success(), first)
	h.Send(m)
	calls := s.calls(t)
	if len(calls) != 1 || !slices.Equal(calls[0][len(calls[0])-2:], []string{m.Summary, m.Body}) {
		t.Fatalf("calls %q", calls)
	}
	if u, _ := flag(calls[0], "-u"); u != "normal" {
		t.Errorf("the triumph is spoken with urgency %q", u)
	}
}

func TestHerald_VoxOffSilencesTheDesktop(t *testing.T) {
	s := newStub(t)
	h := quiet(Config{Vox: librarium.VoxOff})
	h.Proclaim(progress(1, 2))
	h.Proclaim(failure())
	h.Send(Compose(success(), first))
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	if calls := s.calls(t); len(calls) != 0 {
		t.Fatalf("notify-send was called: %q", calls)
	}
}

func TestHerald_MissingNotifySendIsSilent(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	h := quiet(Config{})
	h.Proclaim(progress(1, 2))
	h.Proclaim(failure())
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestHerald_AFallenNotifySendIsSilentAndReplacesNothing(t *testing.T) {
	s := newStub(t)
	s.fall(t)
	h := quiet(Config{})
	h.Proclaim(progress(1, 2))
	h.Proclaim(success())
	calls := s.calls(t)
	if len(calls) != 2 {
		t.Fatalf("calls %q", calls)
	}
	if _, ok := flag(calls[1], "-r"); ok {
		t.Errorf("replacing an id never received: %q", calls[1])
	}
}

func TestHerald_FollowsTheIdTheDesktopAnswers(t *testing.T) {
	s := newStub(t)
	h := quiet(Config{})
	h.Proclaim(progress(1, 2))
	s.answer(t, "77\n")
	h.Proclaim(progress(2, 2))
	h.Proclaim(success())
	calls := s.calls(t)
	if len(calls) != 3 {
		t.Fatalf("calls %q", calls)
	}
	if id, _ := flag(calls[2], "-r"); id != "77" {
		t.Errorf("the triumph replaces %q, not the id last answered: %q", id, calls[2])
	}
}

func TestHerald_HearsAnInvocation(t *testing.T) {
	var _ invocation.Herald = New(Config{})
}
