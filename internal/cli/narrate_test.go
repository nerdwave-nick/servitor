package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

// visageJSON is a rite like the visage of the machine: a long incantation, a
// tether, a progress vox-cast, another incantation and a triumph.
const visageJSON = `{
  "pattern": "Mark I",
  "aspects": ["default", "porpl"],
  "liturgy": [
    {"incantation": "echo the screen is veiled in a transition of two hundred milliseconds or more || true"},
    {"tether": "$HALL/current-theme", "anchor": "$HALL/themes/{{aspect}}"},
    {"vox-cast": "progress"},
    {"incantation": "true"},
    {"vox-cast": "success"}
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
	if got[0] != "+++ mouse-autohide-toggle · (dormant) → on +++" || got[1] != "  ✔ verse 1 · sanctum ~/util.kdl" ||
		!strings.HasPrefix(got[2], "✠ ") || !strings.Contains(got[2], "the aspect «on»") {
		t.Fatalf("stdout:\n%s", out)
	}
	if e.notified() != "" {
		t.Fatalf("the desktop heard %q", e.notified())
	}

	again := lines(e.mustRun("invoke", "mouse-autohide-toggle", "on"))
	if again[0] != "+++ mouse-autohide-toggle · on → on +++" || len(again) != 3 {
		t.Fatalf("re-invoked:\n%s", strings.Join(again, "\n"))
	}
}

func TestInvoke_OnATerminalEveryVerseAndVoxCastIsToldInItsPlace(t *testing.T) {
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
		want := []string{"+++ theme · (dormant) → porpl +++", "  ✔ verse 1 · incantation echo the screen is veiled",
			"  ✔ verse 2 · tether ~/current-theme", "  ⋯ ", "  ✔ verse 4 · incantation true", "✠ "}
		for i, w := range want {
			if !strings.HasPrefix(got[i], w) {
				t.Errorf("%d columns, line %d is %q, want it to begin %q", cols, i, got[i], w)
			}
		}
		if !strings.HasSuffix(got[1], "…") || utf8.RuneCountInString(got[1]) != width {
			t.Errorf("%d columns: the long verse is not cut to fit: %q", cols, got[1])
		}
		if !strings.HasSuffix(got[3], "  (2/3)") || !strings.Contains(got[3], "tether") {
			t.Errorf("progress told as %q", got[3])
		}
		if !strings.Contains(got[5], "the aspect «porpl»") {
			t.Errorf("triumph told as %q", got[5])
		}
		if e.notified() != "" {
			t.Fatalf("the desktop heard %q", e.notified())
		}
	}
}

func TestInvoke_SilenceOnATerminalLeavesOnlyTheVoxCasts(t *testing.T) {
	e := newEnv(t)
	e.addTheme()
	e.terminal(0)

	got := lines(e.mustRun("invoke", "theme", "porpl", "--silence"))

	if len(got) != 2 || !strings.HasPrefix(got[0], "  ⋯ ") || !strings.HasSuffix(got[0], "  (2/3)") ||
		!strings.HasPrefix(got[1], "✠ ") {
		t.Fatalf("stdout:\n%s", strings.Join(got, "\n"))
	}
	e.addRite("rites/mute.json", `{"pattern": "Mark I", "aspects": ["on"], "liturgy": [{"incantation": "true"}]}`)
	if out := e.mustRun("invoke", "mute", "on", "-s"); out != "" {
		t.Fatalf("a rite without vox-casts spoke in silence: %q", out)
	}
}

func TestInvoke_OnATerminalAFallIsToldOnceInTheLament(t *testing.T) {
	e := newEnv(t)
	e.home()
	e.addRite("rites/falling.json", strings.Replace(fallingJSON, `{"incantation"`, `{"vox-cast": "progress"}, {"incantation"`, 1))
	e.terminal(0)

	out, errOut, code := e.run("invoke", "falling", "on", "--reason", "")

	got := lines(out)
	if code != 1 || len(got) != 3 || got[0] != "+++ falling · (dormant) → on +++" ||
		got[1] != "  ✔ verse 1 · sanctum ~/util.kdl" || !strings.HasPrefix(got[2], "  ⋯ ") || closings(out) != 0 {
		t.Fatalf("code=%d stdout:\n%s", code, out)
	}
	if strings.Count(errOut, "death-mark 3") != 1 || strings.Count(errOut, "fell at") != 1 ||
		!strings.Contains(errOut, "verse 1 · sanctum ~/util.kdl is undone") {
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

	if len(got) != 5 || got[0] != "+++ theme · (dormant) → porpl +++" || got[2] != "  ✔ verse 2 · tether ~/current-theme" ||
		got[3] != "  ✔ verse 4 · incantation true" || !strings.HasPrefix(got[4], "✠ ") {
		t.Fatalf("stdout:\n%s", strings.Join(got, "\n"))
	}
	notified := e.notified()
	if strings.Count(notified, "-a servitor") != 2 || !strings.Contains(notified, "-t 0") ||
		!strings.HasSuffix(notified, "-- theme → porpl "+strings.TrimPrefix(got[4], "✠ ")+"\n") {
		t.Fatalf("the report closes otherwise than the desktop heard:\n%s\nnotify-send heard:\n%s", got[4], notified)
	}

	e.addRite("rites/mute.json", `{"pattern": "Mark I", "aspects": ["on"], "liturgy": [{"incantation": "true"}]}`)
	mute := lines(e.mustRun("invoke", "mute", "on"))
	if len(mute) != 3 || !strings.HasPrefix(mute[2], "✠ ") || !strings.Contains(mute[2], "the aspect «on»") {
		t.Fatalf("a rite without vox-casts is not closed:\n%s", strings.Join(mute, "\n"))
	}
	if e.notified() != notified {
		t.Fatalf("the desktop heard a rite without vox-casts:\n%s", e.notified())
	}
}

func TestInvoke_TheHomeIsSpokenAsTildeWhereVersesAreNamed(t *testing.T) {
	e := newEnv(t)
	e.home()
	out := e.mustRun("invoke", "mouse-autohide-toggle", "on", "--foresee")
	if !slices.Contains(lines(out), "verse 1 · sanctum ~/util.kdl") {
		t.Fatalf("foreseen:\n%s", out)
	}
}
