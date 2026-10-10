// Package placeholder illuminates the {{…}} marks in a rite's words and
// examines them for heresy without performing anything.
//
// In a rite (RiteMode) the marks are {{aspect}}, {{former}}, {{rite.name}}
// and {{inscription.<key>}}. Vox-cast tidings (VoxCastMode) may additionally
// speak {{heresy}}, {{step}}, {{steps}}, {{step.kind}} and {{step.target}}.
// {{{{ is a literal {{; a lone }} is literal. Every other mark, an inscription
// the rite never declared, and a mark that is never closed are heresies; they
// are never rendered empty.
package placeholder

import (
	"slices"
	"strconv"
	"strings"
)

// Mode names the words a placeholder appears in, which decides the marks it
// may speak.
type Mode int

const (
	// RiteMode is for every string of a rite: paths, anchors, commands,
	// reversions, offerings, inline and illuminated scripture, the auspex.
	RiteMode Mode = iota
	// VoxCastMode is for vox-cast tidings; it adds the marks of a step's
	// progress and of a heresy to those of RiteMode.
	VoxCastMode
)

// Scope is what the words may refer to: the mode and the inscriptions the
// rite declared.
type Scope struct {
	Mode         Mode
	Inscriptions []string // keys declared under the rite's inscriptions
}

// Values are what the marks are illuminated with during one invocation.
type Values struct {
	Aspect       string            // {{aspect}}: the aspect being invoked
	Former       string            // {{former}}: the aspect before; empty if unknown
	Rite         string            // {{rite.name}}
	Inscriptions map[string]string // {{inscription.<key>}}; an unset key is empty
	Tidings      Tidings           // only in VoxCastMode
}

// Tidings are the values only vox-casts may speak.
type Tidings struct {
	Heresy     string // {{heresy}}: why the invocation fell
	Step       int    // {{step}}: verse of the step just performed
	Steps      int    // {{steps}}: number of real steps in the liturgy
	StepKind   string // {{step.kind}}: the step's own key, e.g. "tether"
	StepTarget string // {{step.target}}: what that key holds, e.g. a vessel
}

// Check reports every heresy in text without rendering it.
func (s Scope) Check(text string) Heresies {
	_, hs := s.illuminate(text, Values{})
	return hs
}

// Render illuminates every mark in text with v. When text holds any heresy
// it returns "" and a non-nil error of type Heresies.
func (s Scope) Render(text string, v Values) (string, error) {
	out, hs := s.illuminate(text, v)
	if len(hs) > 0 {
		return "", hs
	}
	return out, nil
}

const (
	opening = "{{"
	closing = "}}"
	escape  = opening + opening
)

// illuminate walks text once, writing literals and illuminated marks and
// gathering every heresy in the order written.
func (s Scope) illuminate(text string, v Values) (string, Heresies) {
	var b strings.Builder
	var hs Heresies
	for i := 0; i < len(text); {
		j := strings.Index(text[i:], opening)
		if j < 0 {
			b.WriteString(text[i:])
			break
		}
		b.WriteString(text[i : i+j])
		i += j
		if strings.HasPrefix(text[i:], escape) {
			b.WriteString(opening)
			i += len(escape)
			continue
		}
		k := strings.Index(text[i+len(opening):], closing)
		if k < 0 {
			hs = append(hs, s.heresy(Unclosed, text, i, opening, ""))
			break
		}
		name := text[i+len(opening) : i+len(opening)+k]
		mark := text[i : i+len(opening)+k+len(closing)]
		if val, kind, ok := s.resolve(name, v); ok {
			b.WriteString(val)
		} else {
			hs = append(hs, s.heresy(kind, text, i, mark, name))
		}
		i += len(mark)
	}
	return b.String(), hs
}

const inscriptionPrefix = "inscription."

// resolve gives the value of one mark, or the kind of heresy it commits.
func (s Scope) resolve(name string, v Values) (string, Kind, bool) {
	switch name {
	case "aspect":
		return v.Aspect, 0, true
	case "former":
		return v.Former, 0, true
	case "rite.name":
		return v.Rite, 0, true
	}
	if key, ok := strings.CutPrefix(name, inscriptionPrefix); ok && key != "" {
		if !slices.Contains(s.Inscriptions, key) {
			return "", Undeclared, false
		}
		return v.Inscriptions[key], 0, true
	}
	if s.Mode == VoxCastMode {
		if val, ok := v.Tidings.resolve(name); ok {
			return val, 0, true
		}
	}
	return "", Unknown, false
}

// voxMarks are the marks only vox-casts may speak, in the order of the codex.
var voxMarks = []string{"heresy", "step", "steps", "step.kind", "step.target"}

func (t Tidings) resolve(name string) (string, bool) {
	switch name {
	case "heresy":
		return t.Heresy, true
	case "step":
		return strconv.Itoa(t.Step), true
	case "steps":
		return strconv.Itoa(t.Steps), true
	case "step.kind":
		return t.StepKind, true
	case "step.target":
		return t.StepTarget, true
	}
	return "", false
}

func (s Scope) heresy(kind Kind, text string, off int, mark, name string) Heresy {
	return Heresy{Kind: kind, Pos: locate(text, off), Mark: mark, Name: name, Mode: s.Mode}
}

// locate turns a byte offset into a position with line and byte column.
func locate(text string, off int) Position {
	before := text[:off]
	line := strings.Count(before, "\n") + 1
	col := off - (strings.LastIndexByte(before, '\n') + 1) + 1
	return Position{Offset: off, Line: line, Column: col}
}
