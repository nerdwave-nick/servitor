package vox

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

var subjectTidings = []invocation.Tidings{invocation.Progress, invocation.Failure}

// TestSubjectTemplates_AboutFifteenNamingTheSubject: the progress and fall
// of a state rite of pattern Mark II are told by templates of their own,
// which name the subject by {{subject}} in place of a step's kind and
// target.
func TestSubjectTemplates_AboutFifteenNamingTheSubject(t *testing.T) {
	for _, tid := range subjectTidings {
		list := subjectTemplates[tid]
		if len(list) < 12 || len(list) > 20 {
			t.Errorf("subject %s holds %d templates, not about fifteen", tid, len(list))
		}
		seen := map[string]bool{}
		for _, tmpl := range list {
			if seen[tmpl] || slices.Contains(templates[tid], tmpl) || slices.Contains(edictTemplates[tid], tmpl) {
				t.Errorf("subject %s holds %q twice, or as a rite's or an edict's", tid, tmpl)
			}
			seen[tmpl] = true
			if hs := voxScope.Check(tmpl); len(hs) > 0 {
				t.Errorf("subject %s template %q is heretical: %v", tid, tmpl, hs)
			}
			grimdark(t, tmpl)
			if n := utf8.RuneCountInString(tmpl); n > longest[tid] {
				t.Errorf("subject %s template %q holds %d characters, more than %d", tid, tmpl, n, longest[tid])
			}
			if strings.Count(tmpl, "{{subject}}") != 1 || strings.Contains(tmpl, "{{step.") {
				t.Errorf("subject %s template %q names not the subject alone, once", tid, tmpl)
			}
			if strings.Count(tmpl, "{{aspect}}") != strings.Count(tmpl, "aspect {{aspect}}") {
				t.Errorf("subject %s template %q speaks a bare aspect", tid, tmpl)
			}
			if tid == invocation.Failure && !strings.HasSuffix(tmpl, ": {{heresy}}") {
				t.Errorf("subject failure template %q does not end in the heresy", tmpl)
			}
			if tid == invocation.Progress && strings.Contains(tmpl, "{{rite.name}}") {
				t.Errorf("subject progress template %q repeats what the header tells", tmpl)
			}
		}
	}
}

// TestCompose_AStateRiteNamesItsSubjects: the vox-casts of a state rite
// name the subject set last, or the one that fell, count the changing
// subjects ("subject x / n"), and tell which subjects could not be restored.
func TestCompose_AStateRiteNamesItsSubjects(t *testing.T) {
	v := values(2, 3, "", "", `the subject "smt" shows the omen "on"; "off" was commanded`)
	v.Tidings.Subject = "smt"
	for i := range subjectTemplates[invocation.Progress] {
		m := Compose(invocation.Proclamation{Tidings: invocation.Progress, Values: v}, func(int) int { return i })
		if !strings.Contains(m.Flavour, "smt") || m.Body != m.Flavour+"\nsubject 2 / 3" || m.Percent != 66 ||
			strings.Contains(m.Body, "{{") || m.Summary != "theme → porpl" {
			t.Errorf("progress template %d composed %+v", i, m)
		}
	}
	out := &invocation.Outcome{Verdict: invocation.Faltered, Reversions: []invocation.Reversion{
		{Verse: invocation.Verse{Subject: "smt"}, Heresy: errors.New("x")}, {Verse: invocation.Verse{Subject: "theme"}},
		{Verse: invocation.Verse{Subject: "governor"}, Heresy: errors.New("y")}}}
	for i := range subjectTemplates[invocation.Failure] {
		m := Compose(invocation.Proclamation{Tidings: invocation.Failure, Values: v, Outcome: out}, func(int) int { return i })
		if !strings.Contains(m.Flavour, "smt") || !strings.Contains(m.Flavour, `"on"; "off" was commanded`) ||
			!strings.HasSuffix(m.Body, "\nThe reversion faltered at subject smt, subject governor; the rite lies corrupted.") {
			t.Errorf("failure template %d composed %+v", i, m)
		}
	}
	success := Compose(invocation.Proclamation{Tidings: invocation.Success, Values: v}, first)
	if !slices.Contains(templates[invocation.Success], subjectTemplateOf(success.Flavour)) {
		t.Errorf("a state rite's triumph is not told as every rite's: %q", success.Flavour)
	}
}

// subjectTemplateOf is the success template that composed flavour first.
func subjectTemplateOf(flavour string) string {
	tmpl := templates[invocation.Success][0]
	if out, _ := (placeholder.Scope{Mode: placeholder.VoxCastMode}).Render(tmpl,
		placeholder.Values{Aspect: "«porpl»"}); out == flavour {
		return tmpl
	}
	return ""
}
