// Package placeholder illuminates the {{…}} marks in a rite's words and
// examines them for heresy without performing anything.
//
// In a rite (RiteMode) the marks are {{aspect}}, {{former}}, {{rite.name}}
// and {{inscription.<key>}}. Vox-cast tidings (VoxCastMode) may additionally
// speak {{heresy}}, {{step}}, {{steps}}, {{step.kind}} and {{step.target}}.
// The words of an edict (Scope.Edict) bear no aspect: {{aspect}} and
// {{former}} are heresy there.
// The words of an exalted verse (Scope.Exalted) are fixed when its master
// sanctions them: {{former}}, and an inscription only its rune may fill,
// are heresy there.
// {{omen}} names the omen a machine-spirit of pattern Mark II is enacted
// into or tried for; it is heresy in every other word (Scope.Omen).
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
	// Edict marks the words of an edict, which bears no aspect: {{aspect}}
	// and {{former}} are heresy there.
	Edict bool
	// Exalted marks the words of an exalted verse, which are fixed when its
	// master sanctions them: {{former}} and the Fleeting inscriptions are
	// heresy there.
	Exalted bool
	// Fleeting are the inscriptions whose word may come from their rune
	// alone, at invocation: mandatory, yet not decreed for every aspect.
	Fleeting []string
	// Omen marks the words that enact or try an omen of a machine-spirit
	// (pattern Mark II), the only words that may speak {{omen}}.
	Omen bool
}

// Values are what the marks are illuminated with during one invocation.
type Values struct {
	Aspect       string            // {{aspect}}: the aspect being invoked
	Former       string            // {{former}}: the aspect before; empty if unknown
	Rite         string            // {{rite.name}}
	Inscriptions map[string]string // {{inscription.<key>}}; an unset key is empty
	Omen         string            // {{omen}}: the omen being enacted or tried
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
	unclosed := walk(text, func(at int, mark, name string) {
		if val, kind, ok := s.resolve(name, v); ok {
			b.WriteString(val)
		} else {
			hs = append(hs, s.heresy(kind, text, at, mark, name))
		}
	}, func(literal string) { b.WriteString(literal) })
	if unclosed >= 0 {
		hs = append(hs, s.heresy(Unclosed, text, unclosed, opening, ""))
	}
	return b.String(), hs
}

// walk tells every mark of text, with its offset, to mark and every literal
// between them to literal, in the order written. It returns the offset of a
// mark never closed, after which nothing is told; -1 when every mark closes.
func walk(text string, mark func(at int, mark, name string), literal func(string)) int {
	for i := 0; i < len(text); {
		j := strings.Index(text[i:], opening)
		if j < 0 {
			literal(text[i:])
			break
		}
		literal(text[i : i+j])
		i += j
		if strings.HasPrefix(text[i:], escape) {
			literal(opening)
			i += len(escape)
			continue
		}
		k := strings.Index(text[i+len(opening):], closing)
		if k < 0 {
			return i
		}
		name := text[i+len(opening) : i+len(opening)+k]
		m := text[i : i+len(opening)+k+len(closing)]
		mark(i, m, name)
		i += len(m)
	}
	return -1
}

const inscriptionPrefix = "inscription."

// resolve gives the value of one mark, or the kind of heresy it commits.
func (s Scope) resolve(name string, v Values) (string, Kind, bool) {
	switch name {
	case "aspect", "former":
		if s.Edict {
			return "", Aspectless, false
		}
		if name == "aspect" {
			return v.Aspect, 0, true
		}
		if s.Exalted {
			return "", Fleeting, false
		}
		return v.Former, 0, true
	case "rite.name":
		return v.Rite, 0, true
	case "omen":
		if !s.Omen {
			return "", Omenless, false
		}
		return v.Omen, 0, true
	}
	if key, ok := strings.CutPrefix(name, inscriptionPrefix); ok && key != "" {
		if !slices.Contains(s.Inscriptions, key) {
			return "", Undeclared, false
		}
		if s.Exalted && slices.Contains(s.Fleeting, key) {
			return "", Fleeting, false
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
	return Heresy{Kind: kind, Pos: locate(text, off), Mark: mark, Name: name, Mode: s.Mode, Edict: s.Edict}
}

// locate turns a byte offset into a position with line and byte column.
func locate(text string, off int) Position {
	before := text[:off]
	line := strings.Count(before, "\n") + 1
	col := off - (strings.LastIndexByte(before, '\n') + 1) + 1
	return Position{Offset: off, Line: line, Column: col}
}

// Spliced names the declared inscriptions whose marks text speaks, once
// each, in the order first spoken. Escaped marks and marks never closed
// splice nothing.
func (s Scope) Spliced(text string) []string {
	var keys []string
	walk(text, func(_ int, _, name string) {
		key, ok := strings.CutPrefix(name, inscriptionPrefix)
		if ok && slices.Contains(s.Inscriptions, key) && !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}, func(string) {})
	return keys
}
