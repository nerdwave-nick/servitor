// Package vox is the herald of invocations: it turns the proclamations of
// an invocation (see invocation.Herald) into vox-casts drawn from embedded
// grimdark templates, and sends those of an invocation roused without a
// terminal to the desktop as one notification through notify-send,
// replaced in place by every later vox-cast of the same invocation. On a
// terminal the servitor tells the composed vox-casts itself.
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
	Summary string // the rite and the aspect invoked; an edict's name alone
	Flavour string // the template drawn, illuminated
	Body    string // the flavour and what follows it; lines are joined by "\n"
	Percent int    // progress only: the share of real steps performed
}

var (
	//go:embed progress.txt
	progressFile string
	//go:embed success.txt
	successFile string
	//go:embed failure.txt
	failureFile string
	//go:embed edict-progress.txt
	edictProgressFile string
	//go:embed edict-success.txt
	edictSuccessFile string
	//go:embed edict-failure.txt
	edictFailureFile string
	//go:embed dominion.txt
	dominionFile string
	//go:embed edict-dominion.txt
	edictDominionFile string
	//go:embed exaltation.txt
	exaltationFile string
	//go:embed edict-exaltation.txt
	edictExaltationFile string
)

// templates are the embedded template lists by tidings.
var templates = map[invocation.Tidings][]string{
	invocation.Progress:   parseTemplates("progress.txt", progressFile),
	invocation.Success:    parseTemplates("success.txt", successFile),
	invocation.Failure:    parseTemplates("failure.txt", failureFile),
	invocation.Dominion:   parseTemplates("dominion.txt", dominionFile),
	invocation.Exaltation: parseTemplates("exaltation.txt", exaltationFile),
}

// edictTemplates are the embedded template lists of an edict by tidings:
// they name the edict and speak no aspect.
var edictTemplates = map[invocation.Tidings][]string{
	invocation.Progress:   parseTemplates("edict-progress.txt", edictProgressFile),
	invocation.Success:    parseTemplates("edict-success.txt", edictSuccessFile),
	invocation.Failure:    parseTemplates("edict-failure.txt", edictFailureFile),
	invocation.Dominion:   parseTemplates("edict-dominion.txt", edictDominionFile),
	invocation.Exaltation: parseTemplates("edict-exaltation.txt", edictExaltationFile),
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

// voxScope illuminates the templates; they speak no inscriptions. An
// edict's speak no aspect either.
var (
	voxScope   = placeholder.Scope{Mode: placeholder.VoxCastMode}
	edictScope = placeholder.Scope{Mode: placeholder.VoxCastMode, Edict: true}
)

// awakening is the progress told before any real step was performed, when
// there is no step before to describe.
const awakening = "The machine spirit stirs; the liturgy begins."

// Compose draws a template for p's tidings with pick (which returns an
// index below n) and illuminates it, the aspect written «aspect». The body
// of progress ends in "step x / n", that of failure in how the reversion
// went. An edict's vox-cast is drawn from its own templates, and neither it
// nor its summary speaks an aspect.
func Compose(p invocation.Proclamation, pick func(n int) int) Message {
	list, scope := templates[p.Tidings], voxScope
	if p.Edict {
		list, scope = edictTemplates[p.Tidings], edictScope
	}
	tmpl := list[pick(len(list))]
	tid := p.Values.Tidings
	if p.Tidings == invocation.Progress && tid.Step == 0 {
		tmpl = awakening
	}
	v, summary := p.Values, p.Values.Rite
	if !p.Edict {
		v.Aspect = "«" + v.Aspect + "»"
		summary += " → " + p.Values.Aspect
	}
	if p.Tidings == invocation.Exaltation {
		tmpl = exaltationMarks(p).Replace(tmpl)
	}
	body := illuminate(scope, tmpl, v)
	m := Message{Tidings: p.Tidings, Summary: summary, Flavour: body}
	switch p.Tidings {
	case invocation.Progress:
		m.Percent = 100
		if tid.Steps > 0 {
			m.Percent = tid.Step * 100 / tid.Steps
		}
		body += fmt.Sprintf("\nstep %d / %d", tid.Step, tid.Steps)
	case invocation.Failure:
		body += "\n" + reversion(p.Outcome, p.Edict)
	}
	m.Body = body
	return m
}

// exaltationMarks illuminate the marks only the templates of an exaltation
// speak, the intercessor and the sigil, before the rest is rendered: they
// are no placeholders of any rite's scripture. What they name is spoken
// with its own marks escaped.
func exaltationMarks(p invocation.Proclamation) *strings.Replacer {
	escape := strings.NewReplacer("{{", "{{{{")
	return strings.NewReplacer("{{intercessor}}", escape.Replace(p.Intercessor), "{{sigil}}", escape.Replace(p.Sigil))
}

// illuminate renders a template; one that would not render is spoken as
// written rather than lost (the tests forbid such templates).
func illuminate(scope placeholder.Scope, tmpl string, v placeholder.Values) string {
	out, err := scope.Render(tmpl, v)
	if err != nil {
		return tmpl
	}
	return out
}

// reversion tells how the reversion of a fallen invocation went.
func reversion(out *invocation.Outcome, edict bool) string {
	if out == nil || out.Verdict != invocation.Faltered {
		return "Every deed is undone; the machine stands as it stood before."
	}
	var fallen []string
	for _, r := range out.Reversions {
		if r.Heresy != nil {
			fallen = append(fallen, fmt.Sprintf("verse %d", r.Number))
		}
	}
	lies := "the rite lies corrupted."
	if edict {
		lies = "the edict lies corrupted."
	}
	return "The reversion faltered at " + strings.Join(fallen, ", ") + "; " + lies
}
