package vox

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// plainGlosses are plain words no vox-cast may speak.
var plainGlosses = regexp.MustCompile(`(?i)\b(error|warning|fail(ed|ure)?|success(ful)?|switch|states?|metadata|config|invalid|file|command|timeout|symlink|directory|rollback|undo|notification)\b`)

func grimdark(t *testing.T, words string) {
	t.Helper()
	if m := plainGlosses.FindString(words); m != "" {
		t.Errorf("%q speaks the plain word %q", words, m)
	}
}

var allTidings = []invocation.Tidings{invocation.Success, invocation.Progress, invocation.Failure}

// longest is how many characters a template of each tidings may hold before
// it is illuminated: a progress line also bears its indent and count.
var longest = map[invocation.Tidings]int{invocation.Success: 70, invocation.Progress: 66, invocation.Failure: 70}

func TestTemplates_AboutThirtyFlavourfulEntriesEach(t *testing.T) {
	voxScope := placeholder.Scope{Mode: placeholder.VoxCastMode}
	for _, tid := range allTidings {
		list := templates[tid]
		if len(list) < 27 || len(list) > 40 {
			t.Errorf("%s holds %d templates, not about thirty", tid, len(list))
		}
		seen := map[string]bool{}
		for _, tmpl := range list {
			if seen[tmpl] {
				t.Errorf("%s holds %q twice", tid, tmpl)
			}
			seen[tmpl] = true
			if hs := voxScope.Check(tmpl); len(hs) > 0 {
				t.Errorf("%s template %q is heretical: %v", tid, tmpl, hs)
			}
			grimdark(t, tmpl)
			if n := utf8.RuneCountInString(tmpl); n > longest[tid] {
				t.Errorf("%s template %q holds %d characters, more than %d", tid, tmpl, n, longest[tid])
			}
			if strings.Count(tmpl, "{{aspect}}") != strings.Count(tmpl, "aspect {{aspect}}") {
				t.Errorf("%s template %q speaks a bare aspect, not \"the aspect {{aspect}}\"", tid, tmpl)
			}
			switch tid {
			case invocation.Progress:
				if !strings.Contains(tmpl, "{{step.kind}}") {
					t.Errorf("progress template %q does not describe the step before", tmpl)
				}
				if strings.Contains(tmpl, "{{step.target}}") || strings.Contains(tmpl, "{{rite.name}}") {
					t.Errorf("progress template %q repeats what the verse above tells", tmpl)
				}
			case invocation.Success:
				if !strings.Contains(tmpl, "{{aspect}}") {
					t.Errorf("success template %q does not speak the aspect attained", tmpl)
				}
				if strings.Contains(tmpl, "{{rite.name}}") || strings.Contains(tmpl, "{{former}}") {
					t.Errorf("success template %q repeats what the header tells", tmpl)
				}
			case invocation.Failure:
				if !strings.HasSuffix(tmpl, ": {{heresy}}") {
					t.Errorf("failure template %q does not end in the heresy", tmpl)
				}
			}
		}
	}
}

func TestCompose_ReadsWellWithAnyAspect(t *testing.T) {
	for _, tid := range allTidings {
		for i := range templates[tid] {
			for _, aspect := range []string{"on", "off", "porpl", "dark-mode"} {
				v := values(1, 2, "tether", "/x", "the tether is severed")
				v.Aspect = aspect
				out := Compose(invocation.Proclamation{Tidings: tid, Values: v, Outcome: &invocation.Outcome{}},
					func(int) int { return i })
				if strings.Contains(out.Body, "{{") {
					t.Errorf("%s template %d reads %q", tid, i, out.Flavour)
				}
				if strings.Contains(templates[tid][i], "{{aspect}}") && !strings.Contains(out.Flavour, "the aspect «"+aspect+"»") &&
					!strings.Contains(out.Flavour, "The aspect «"+aspect+"»") {
					t.Errorf("%s template %d hides the aspect: %q", tid, i, out.Flavour)
				}
			}
		}
	}
}

func values(step, steps int, kind, target, heresy string) placeholder.Values {
	return placeholder.Values{Aspect: "porpl", Former: "default", Rite: "theme",
		Tidings: placeholder.Tidings{Step: step, Steps: steps, StepKind: kind, StepTarget: target, Heresy: heresy}}
}

func first(int) int { return 0 }

func TestCompose_ProgressDescribesTheStepBeforeAndCountsRealSteps(t *testing.T) {
	for i := range templates[invocation.Progress] {
		m := Compose(invocation.Proclamation{Tidings: invocation.Progress,
			Values: values(2, 5, "tether", "/home/x/.local/share/nfluff/current-theme", "")},
			func(int) int { return i })
		if m.Tidings != invocation.Progress || m.Percent != 40 || m.Body != m.Flavour+"\nstep 2 / 5" {
			t.Fatalf("composed %+v", m)
		}
		if !strings.Contains(m.Flavour, "tether") {
			t.Errorf("progress %q describes not the step before", m.Body)
		}
		if strings.Contains(m.Body, "{{") || m.Summary != "theme → porpl" {
			t.Errorf("composed %+v", m)
		}
	}
}

func TestCompose_ProgressBeforeAnyRealStep(t *testing.T) {
	m := Compose(invocation.Proclamation{Tidings: invocation.Progress, Values: values(0, 3, "", "", "")}, first)
	if m.Percent != 0 || !strings.HasSuffix(m.Body, "\nstep 0 / 3") || strings.Contains(m.Body, "  ") ||
		strings.Contains(m.Body, "theme") {
		t.Fatalf("composed %+v", m)
	}
	grimdark(t, m.Body)
	if m := Compose(invocation.Proclamation{Tidings: invocation.Progress, Values: values(0, 0, "", "", "")}, first); m.Percent != 100 {
		t.Fatalf("a liturgy of vox-casts alone is complete, got %d%%", m.Percent)
	}
}

func TestCompose_SuccessAnnouncesTriumph(t *testing.T) {
	m := Compose(invocation.Proclamation{Tidings: invocation.Success, Values: values(3, 3, "litany", "x", "")}, first)
	if m.Tidings != invocation.Success || m.Body == "" || strings.Contains(m.Body, "{{") || strings.Contains(m.Body, "step 3") {
		t.Fatalf("composed %+v", m)
	}
	if m.Flavour != m.Body || !strings.Contains(m.Body, "the aspect «porpl»") || m.Summary != "theme → porpl" {
		t.Fatalf("composed %+v", m)
	}
}

func TestCompose_FailureSpeaksTheHeresyAndTheReversion(t *testing.T) {
	fell := &invocation.Fall{Verse: invocation.Verse{Number: 4, Kind: librarium.KindIncantation, Target: "exit 3"}}
	cases := []struct {
		name    string
		outcome invocation.Outcome
		want    string
	}{
		{"reverted", invocation.Outcome{Verdict: invocation.Reverted, Fell: fell, Reversions: []invocation.Reversion{
			{Verse: invocation.Verse{Number: 4}}, {Verse: invocation.Verse{Number: 1}}}},
			"Every deed is undone"},
		{"faltered", invocation.Outcome{Verdict: invocation.Faltered, Fell: fell, Reversions: []invocation.Reversion{
			{Verse: invocation.Verse{Number: 4}, Heresy: errors.New("x")}, {Verse: invocation.Verse{Number: 2}},
			{Verse: invocation.Verse{Number: 1}, Heresy: errors.New("y")}}},
			"verse 4, verse 1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := Compose(invocation.Proclamation{Tidings: invocation.Failure,
				Values: values(3, 3, "incantation", "exit 3", "the incantation fell silent with code 3"), Outcome: &c.outcome}, first)
			if m.Tidings != invocation.Failure || !strings.Contains(m.Flavour, "the incantation fell silent with code 3") ||
				!strings.HasPrefix(m.Body, m.Flavour+"\n") {
				t.Fatalf("composed %+v", m)
			}
			if !strings.Contains(m.Body, c.want) {
				t.Fatalf("body %q tells not %q", m.Body, c.want)
			}
			grimdark(t, strings.ReplaceAll(m.Body, "the incantation fell silent with code 3", ""))
		})
	}
}

func TestCompose_DrawsByPick(t *testing.T) {
	list := templates[invocation.Success]
	var asked int
	m := Compose(invocation.Proclamation{Tidings: invocation.Success, Values: values(1, 1, "", "", "")},
		func(n int) int { asked = n; return n - 1 })
	if asked != len(list) {
		t.Fatalf("pick asked for %d, the list holds %d", asked, len(list))
	}
	v := values(1, 1, "", "", "")
	v.Aspect = "«porpl»"
	want, _ := placeholder.Scope{Mode: placeholder.VoxCastMode}.Render(list[len(list)-1], v)
	if m.Body != want {
		t.Fatalf("body %q, want %q", m.Body, want)
	}
}
