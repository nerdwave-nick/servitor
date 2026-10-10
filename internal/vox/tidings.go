// Package vox is the herald of invocations: it turns the proclamations of
// an invocation (see invocation.Herald) into vox-casts drawn from embedded
// grimdark templates, and sends them where the user will hear them —
// printed on the controlling terminal when there is one, otherwise as one
// desktop notification through notify-send, replaced in place by every
// later vox-cast of the same invocation.
package vox

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// Message is one vox-cast, composed and ready to be sent.
type Message struct {
	Tidings invocation.Tidings
	Summary string // the rite and the aspect invoked
	Body    string // the tidings themselves; lines are joined by "\n"
	Percent int    // progress only: the share of real steps performed
}

var (
	//go:embed progress.txt
	progressFile string
	//go:embed success.txt
	successFile string
	//go:embed failure.txt
	failureFile string
)

// templates are the embedded template lists by tidings.
var templates = map[invocation.Tidings][]string{
	invocation.Progress: parseTemplates("progress.txt", progressFile),
	invocation.Success:  parseTemplates("success.txt", successFile),
	invocation.Failure:  parseTemplates("failure.txt", failureFile),
}

// parseTemplates takes one template per line, ignoring blank lines and
// lines starting with "#". An empty list is a build defect.
func parseTemplates(name, s string) []string {
	var out []string
	for line := range strings.Lines(s) {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			out = append(out, line)
		}
	}
	if len(out) == 0 {
		panic("vox: " + name + " holds no templates")
	}
	return out
}

// voxScope illuminates the templates; they speak no inscriptions.
var voxScope = placeholder.Scope{Mode: placeholder.VoxCastMode}

// awakening is the progress told before any real step was performed, when
// there is no step before to describe.
const awakening = "The machine spirit stirs; the liturgy of {{rite.name}} begins."

// Compose draws a template for p's tidings with pick (which returns an
// index below n) and illuminates it. Progress ends in "step x / n";
// failure ends in how the reversion went.
func Compose(p invocation.Proclamation, pick func(n int) int) Message {
	list := templates[p.Tidings]
	tmpl := list[pick(len(list))]
	tid := p.Values.Tidings
	if p.Tidings == invocation.Progress && tid.Step == 0 {
		tmpl = awakening
	}
	body := illuminate(tmpl, p.Values)
	m := Message{Tidings: p.Tidings, Summary: p.Values.Rite + " → " + p.Values.Aspect}
	switch p.Tidings {
	case invocation.Progress:
		m.Percent = 100
		if tid.Steps > 0 {
			m.Percent = tid.Step * 100 / tid.Steps
		}
		body += fmt.Sprintf("\nstep %d / %d", tid.Step, tid.Steps)
	case invocation.Failure:
		body += "\n" + reversion(p.Outcome)
	}
	m.Body = body
	return m
}

// illuminate renders a template; one that would not render is spoken as
// written rather than lost (the tests forbid such templates).
func illuminate(tmpl string, v placeholder.Values) string {
	out, err := voxScope.Render(tmpl, v)
	if err != nil {
		return tmpl
	}
	return out
}

// reversion tells how the reversion of a fallen invocation went.
func reversion(out *invocation.Outcome) string {
	if out == nil || out.Verdict != invocation.Faltered {
		return "Every deed is undone; the machine stands as it stood before."
	}
	var fallen []string
	for _, r := range out.Reversions {
		if r.Heresy != nil {
			fallen = append(fallen, fmt.Sprintf("verse %d", r.Number))
		}
	}
	return "The reversion faltered at " + strings.Join(fallen, ", ") + "; the rite lies corrupted."
}
