package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

// visageJSON is a rite like the visage of the machine: a tether of a long
// name, an incantation spoken before it and a litany beside the rite after
// it, its progress and its triumph proclaimed.
const visageJSON = `{
  "pattern": "Mark II",
  "aspects": ["default", "porpl"],
  "vox-cast": ["progress", "success"],
  "subjects": [
    {"name": "visage", "designation": "tether",
     "tether": "$HALL/the-current-theme-veiled-in-a-transition-of-two-hundred-milliseconds-or-more",
     "omens": {"*": "$HALL/themes/{{aspect}}"}, "introit": ["veil"], "recessional": ["hooks"]}
  ],
  "observances": [
    {"name": "veil", "designation": "incantation", "words": "echo the screen is veiled || true"},
    {"name": "hooks", "designation": "litany", "scroll": "scripts/hooks", "offerings": ["~/themes/{{aspect}}"]}
  ]
}`

// home makes the hall of the env's vessel the home of the faithful and
// returns it.
func (e *env) home() string {
	hall := filepath.Dir(e.target)
	e.t.Setenv("HOME", hall)
	return hall
}

func (e *env) addTheme() {
	e.t.Helper()
	hall := filepath.Dir(e.target)
	if err := os.MkdirAll(filepath.Join(hall, "themes", "porpl"), 0o755); err != nil {
		e.t.Fatal(err)
	}
	e.addRite("rites/theme.json", strings.ReplaceAll(visageJSON, "$HALL", hall))
	if err := os.Chmod(e.addRite("rites/scripts/hooks", "#!/bin/sh\ntest -d \"$1\"\n"), 0o755); err != nil {
		e.t.Fatal(err)
	}
}

// speaksAspect reports whether line speaks of the aspect «aspect».
func speaksAspect(line, aspect string) bool {
	return strings.Contains(strings.ToLower(line), "the aspect «"+aspect+"»")
}

func lines(s string) []string { return strings.Split(strings.TrimSuffix(s, "\n"), "\n") }

// closings counts the lines that close a narration.
func closings(out string) int {
	n := 0
	for _, l := range lines(out) {
		if strings.HasPrefix(l, "✠ ") {
			n++
		}
	}
	return n
}

func TestInvoke_OnATerminalTheInvocationIsToldOnceAsItHappens(t *testing.T) {
	e := newEnv(t)
	e.home()
	e.terminal(0)

	out, errOut, code := e.run("invoke", "mouse-autohide-toggle", "on")

	got := lines(out)
	if code != 0 || errOut != "" || len(got) != 3 {
		t.Fatalf("code=%d stderr=%q stdout:\n%s", code, errOut, out)
	}
	if got[0] != "+++ mouse-autohide-toggle · (dormant) → on +++" || got[1] != "  ✔ subject cursor · sanctum ~/util.kdl" ||
		!strings.HasPrefix(got[2], "✠ ") || !speaksAspect(got[2], "on") {
		t.Fatalf("stdout:\n%s", out)
	}
	if e.notified() != "" {
		t.Fatalf("the desktop heard %q", e.notified())
	}

	again := lines(e.mustRun("invoke", "mouse-autohide-toggle", "on"))
	// the cursor already stands in its omen, and nothing is set or told
	if again[0] != "+++ mouse-autohide-toggle · on → on +++" || len(again) != 2 || !strings.HasPrefix(again[1], "✠ ") {
		t.Fatalf("re-invoked:\n%s", strings.Join(again, "\n"))
	}
}

func TestInvoke_OnATerminalEverySubjectObservanceAndVoxCastIsToldInItsPlace(t *testing.T) {
	for _, cols := range []int{0, 60} {
		e := newEnv(t)
		e.home()
		e.addTheme()
		e.terminal(cols)

		got := lines(e.mustRun("invoke", "theme", "porpl"))

		if len(got) != 6 {
			t.Fatalf("stdout:\n%s", strings.Join(got, "\n"))
		}
		width := cols
		if width == 0 {
			width = 80
		}
		want := []string{"+++ theme · (dormant) → porpl +++", "  ✔ observance veil · incantation",
			"  ✔ subject visage · tether ~/the-current-theme", "  ⋯ ", "  ✔ observance hooks · litany", "✠ "}
		for i, w := range want {
			if !strings.HasPrefix(got[i], w) {
				t.Errorf("%d columns, line %d is %q, want it to begin %q", cols, i, got[i], w)
			}
		}
		if !strings.HasSuffix(got[2], "…") || utf8.RuneCountInString(got[2]) != width {
			t.Errorf("%d columns: the long subject is not cut to fit: %q", cols, got[2])
		}
		if !strings.HasSuffix(got[3], "  (1/1)") || !strings.Contains(got[3], "visage") {
			t.Errorf("progress told as %q", got[3])
		}
		if !speaksAspect(got[5], "porpl") {
			t.Errorf("triumph told as %q", got[5])
		}
		if e.notified() != "" {
			t.Fatalf("the desktop heard %q", e.notified())
		}
	}
}

func TestInvoke_SilenceOnATerminalLeavesOnlyTheVoxCasts(t *testing.T) {
	e := newEnv(t)
	e.home()
	e.addTheme()
	e.terminal(0)

	got := lines(e.mustRun("invoke", "theme", "porpl", "--silence"))

	if len(got) != 2 || !strings.HasPrefix(got[0], "  ⋯ ") || !strings.HasSuffix(got[0], "  (1/1)") ||
		!strings.HasPrefix(got[1], "✠ ") {
		t.Fatalf("stdout:\n%s", strings.Join(got, "\n"))
	}
	e.addRite("rites/mute.json", `{"pattern": "Mark II", "aspects": ["on"], "subjects": [
	  {"name": "mute", "designation": "machine-spirit", "enact": {"*": "true"}}]}`)
	if out := e.mustRun("invoke", "mute", "on", "-s"); out != "" {
		t.Fatalf("a rite without vox-casts spoke in silence: %q", out)
	}
}

func TestInvoke_OnATerminalAFallIsToldOnceInTheLament(t *testing.T) {
	e := newEnv(t)
	e.home()
	e.addRite("rites/falling.json", strings.Replace(fallingJSON, `"aspects"`, `"vox-cast": ["progress"], "aspects"`, 1))
	e.terminal(0)

	out, errOut, code := e.run("invoke", "falling", "on", "--reason", "")

	got := lines(out)
	if code != 1 || len(got) != 3 || got[0] != "+++ falling · (dormant) → on +++" ||
		got[1] != "  ✔ subject warded · sanctum ~/util.kdl" || !strings.HasPrefix(got[2], "  ⋯ ") || closings(out) != 0 {
		t.Fatalf("code=%d stdout:\n%s", code, out)
	}
	if strings.Count(errOut, "death-mark 3") != 1 || strings.Count(errOut, "fell at") != 1 ||
		!strings.Contains(errOut, "subject warded · sanctum ~/util.kdl is undone") {
		t.Fatalf("the fall is not told once:\n%s", errOut)
	}
	if e.notified() != "" {
		t.Fatalf("the desktop heard %q", e.notified())
	}
}

func TestInvoke_WithoutATerminalTheDesktopHearsAndTheReportFollows(t *testing.T) {
	e := newEnv(t)
	e.home()
	e.addTheme()

	got := lines(e.mustRun("invoke", "theme", "porpl"))

	if len(got) != 5 || got[0] != "+++ theme · (dormant) → porpl +++" || !strings.HasPrefix(got[2], "  ✔ subject visage · tether ~/the-current-theme") ||
		got[3] != "  ✔ observance hooks · litany" || !strings.HasPrefix(got[4], "✠ ") {
		t.Fatalf("stdout:\n%s", strings.Join(got, "\n"))
	}
	notified := e.notified()
	if strings.Count(notified, "-a servitor") != 2 || !strings.Contains(notified, "-t 0") ||
		!strings.HasSuffix(notified, "-- theme → porpl "+strings.TrimPrefix(got[4], "✠ ")+"\n") {
		t.Fatalf("the report closes otherwise than the desktop heard:\n%s\nnotify-send heard:\n%s", got[4], notified)
	}

	e.addRite("rites/mute.json", `{"pattern": "Mark II", "aspects": ["on"], "subjects": [
	  {"name": "mute", "designation": "machine-spirit", "enact": {"*": "true"}}]}`)
	mute := lines(e.mustRun("invoke", "mute", "on"))
	if len(mute) != 3 || !strings.HasPrefix(mute[2], "✠ ") || !speaksAspect(mute[2], "on") {
		t.Fatalf("a rite without vox-casts is not closed:\n%s", strings.Join(mute, "\n"))
	}
	if e.notified() != notified {
		t.Fatalf("the desktop heard a rite without vox-casts:\n%s", e.notified())
	}
}

func TestInvoke_TheHomeIsSpokenAsTildeWhereSubjectsAreNamed(t *testing.T) {
	e := newEnv(t)
	e.home()
	out := e.mustRun("invoke", "mouse-autohide-toggle", "on", "--foresee")
	if !slices.Contains(lines(out), "subject cursor · sanctum ~/util.kdl: changing") {
		t.Fatalf("foreseen:\n%s", out)
	}
	e.addTheme()
	out = e.mustRun("invoke", "theme", "porpl", "--foresee")
	if !slices.Contains(lines(out), "  the tether is bound to ~/themes/porpl (it led to nothing)") {
		t.Fatalf("foreseen:\n%s", out)
	}
}
