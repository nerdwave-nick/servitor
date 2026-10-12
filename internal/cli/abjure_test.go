package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// abjuring is a Librarium whose store, as the exalted servitor keeps it,
// holds the sanctions of split-lock, of "gone" (forsaken: its rite is gone)
// and of "other".
func abjuring(t *testing.T) (*env, *altar) {
	t.Helper()
	e := newEnv(t)
	e.addRite("rites/split-lock.json", splitLockJSON)
	e.addRite("rites/other.json", splitLockJSON)
	a := e.raiseAltar("sudo", "run0", "pkexec")
	e.mustRun("sanction", "split-lock")
	e.mustRun("sanction", "other")
	if err := os.Rename(a.store.Path("other"), a.store.Path("gone")); err != nil {
		t.Fatal(err)
	}
	e.mustRun("sanction", "other")
	a.mu.Lock()
	a.calls = nil
	a.mu.Unlock()
	return e, a
}

// sanctioned names the rites whose sanctions the store keeps.
func (a *altar) sanctioned() []string {
	entries, _ := os.ReadDir(filepath.Dir(a.store.Path("x")))
	var out []string
	for _, en := range entries {
		out = append(out, strings.TrimSuffix(en.Name(), ".json"))
	}
	return out
}

// TestAbjure_StrikesASanctionWithOneProof: the abjuring of a sanction calls
// upon the intercessor once, with its exact words, and the exalted servitor
// strikes it and journals it; a forsaken sanction is abjured alike, and
// --all strikes every sanction. No terminal need attend.
func TestAbjure_StrikesASanctionWithOneProof(t *testing.T) {
	e, a := abjuring(t)
	out, errOut, code := e.run("abjure", "split-lock")
	if code != 0 || errOut != "" || !strings.Contains(out, `"split-lock"`) {
		t.Fatalf("abjure: exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	grimdark(t, out)
	want := []string{filepath.Join(a.bin, "sudo"), "-k", "-p", "servitor: authenticate to abjure the sanction of " +
		"split-lock: ", "--", "/usr/bin/servitor", "__exalt", "--abjure", "split-lock"}
	if calls := a.called(); len(calls) != 1 || !slices.Equal(calls[0], want) {
		t.Errorf("called upon as %q\nwant %q", calls, want)
	}
	if got := a.sanctioned(); !slices.Equal(got, []string{"gone", "other"}) {
		t.Errorf("the store keeps %v", got)
	}
	if lines := a.journal.lines(); !strings.Contains(lines[len(lines)-1], `rite="split-lock"`) ||
		!strings.Contains(lines[len(lines)-1], "deed=abjure verdict=abjured") {
		t.Errorf("journaled %q", lines)
	}
	if got := e.mustRun("augury", "split-lock", "sanction"); got != "unsanctioned\n" {
		t.Errorf("the rite stands %q", got)
	}

	a.attended = false
	if out, errOut, code := e.run("abjure", "gone"); code != 0 || !slices.Equal(a.sanctioned(), []string{"other"}) {
		t.Errorf("abjure gone: exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	if calls := a.called(); len(calls) != 2 || calls[1][0] != filepath.Join(a.bin, "run0") ||
		calls[1][1] != "--unit=servitor-abjure-gone" {
		t.Errorf("without a terminal, called upon as %q", calls)
	}
	if out, errOut, code := e.run("abjure", "--all"); code != 0 || len(a.sanctioned()) != 0 ||
		!strings.Contains(out, `"other"`) {
		t.Errorf("abjure --all: exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	if calls := a.called(); len(calls) != 3 || !slices.Contains(calls[2], "--all") {
		t.Errorf("--all: called upon as %q", calls)
	}
}

// TestAbjure_RefusesWhatIsNotKept: a sanction not laid, every sanction when
// none is, and words that name no rite or two are refused with the
// death-mark 2, and no intercessor is called upon; a dismissed proof ends
// with the death-mark 1 and strikes nothing; as root the ritual is refused.
func TestAbjure_RefusesWhatIsNotKept(t *testing.T) {
	e, a := abjuring(t)
	for _, args := range [][]string{{"abjure", "never"}, {"abjure"}, {"abjure", "split-lock", "other"},
		{"abjure", "split-lock", "--all"}, {"abjure", "../split-lock"}} {
		out, errOut, code := e.run(args...)
		if code != 2 || out != "" || len(a.called()) != 0 {
			t.Errorf("%v: exit %d, %d calls\nstdout: %s\nstderr: %s", args, code, len(a.called()), out, errOut)
		}
		grimdark(t, errOut)
	}
	if _, errOut, _ := e.run("abjure", "never"); !strings.Contains(errOut, `no sanction of "never"`) {
		t.Errorf("abjure never: %s", errOut)
	}
	a.vouch = false
	out, errOut, code := e.run("abjure", "split-lock")
	if code != 1 || !strings.Contains(errOut, "did not vouch") || len(a.sanctioned()) != 3 {
		t.Errorf("dismissed: exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	e2 := newEnv(t)
	a2 := e2.raiseAltar("sudo")
	if _, errOut, code := e2.run("abjure", "--all"); code != 2 || !strings.Contains(errOut, "no sanction is laid") ||
		len(a2.called()) != 0 {
		t.Errorf("--all of none: exit %d, %d calls\n%s", code, len(a2.called()), errOut)
	}
	if _, err := os.Stat(a2.store.Root); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the store was raised: %v", err)
	}
	asRoot(t)
	if _, errOut, code := e.run("abjure", "split-lock"); code != 2 || !strings.Contains(errOut, "machine's own authority") {
		t.Errorf("as root: exit %d\n%s", code, errOut)
	}
	if len(a.sanctioned()) != 3 {
		t.Errorf("the store keeps %v", a.sanctioned())
	}
}
