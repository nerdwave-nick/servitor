package placeholder

import (
	"fmt"
	"slices"
	"strings"
)

// Kind distinguishes the heresies a placeholder can commit.
type Kind int

const (
	// Unknown is a mark the codex does not know in this mode.
	Unknown Kind = iota
	// Undeclared is an {{inscription.<key>}} whose key the rite never declared.
	Undeclared
	// Unclosed is a {{ that no }} ever closes.
	Unclosed
)

// Position locates a heresy within the words it was found in.
type Position struct {
	Offset int // byte offset of the opening {{, counted from zero
	Line   int // line, counted from one
	Column int // byte column within that line, counted from one
}

// Heresy is one placeholder that cannot be illuminated.
type Heresy struct {
	Kind Kind
	Pos  Position
	Mark string // the mark as written, e.g. "{{stat}}"; "{{" when unclosed
	Name string // what stands between the braces, e.g. "stat"
	Mode Mode
}

// Message is the grimdark denunciation without its position.
func (h Heresy) Message() string {
	switch h.Kind {
	case Undeclared:
		key := strings.TrimPrefix(h.Name, inscriptionPrefix)
		return fmt.Sprintf("%s calls upon the inscription %q, which this rite never declared "+
			"under \"inscriptions\"; the servitor will not illuminate a name the rite does not bear, "+
			"lest nothing be spoken in its stead", h.Mark, key)
	case Unclosed:
		return "a mark is opened with {{ but never sealed with }}; an open mark is a breach " +
			"through which corruption seeps into the rite. To inscribe a literal {{, write {{{{"
	}
	if h.Mode == RiteMode && slices.Contains(voxMarks, h.Name) {
		return fmt.Sprintf("%s may be spoken only in the tidings of a vox-cast; within the liturgy "+
			"of a rite it is heresy", h.Mark)
	}
	return fmt.Sprintf("%s is no placeholder known to the codex and shall not be illuminated as "+
		"nothing; %s", h.Mark, h.Mode.speakable())
}

// speakable recites the marks a mode may speak.
func (m Mode) speakable() string {
	marks, words := riteMarks, "the words of a rite"
	if m == VoxCastMode {
		marks, words = append(slices.Clone(riteMarks), voxMarks...), "the tidings of a vox-cast"
	}
	last := len(marks) - 1
	return fmt.Sprintf("%s may speak only {{%s}} and {{%s}}", words,
		strings.Join(marks[:last], "}}, {{"), marks[last])
}

// riteMarks are the marks every rite may speak, in the order of the codex.
var riteMarks = []string{"aspect", "former", "rite.name", inscriptionPrefix + "<key>"}

// Error is the denunciation prefixed with line:column.
func (h Heresy) Error() string {
	return fmt.Sprintf("%d:%d: %s", h.Pos.Line, h.Pos.Column, h.Message())
}

// Heresies is every heresy found in one text, in the order written.
type Heresies []Heresy

// Error joins the denunciations, one per line.
func (hs Heresies) Error() string {
	lines := make([]string, len(hs))
	for i, h := range hs {
		lines[i] = h.Error()
	}
	return strings.Join(lines, "\n")
}
