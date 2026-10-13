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
	// Aspectless is {{aspect}} or {{former}} in the words of an edict.
	Aspectless
	// Fleeting is {{former}}, or an inscription that only its rune may
	// fill, in the words of an exalted verse.
	Fleeting
	// Omenless is {{omen}} in words that neither enact nor try an omen.
	Omenless
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
	// Edict tells that the words were an edict's, which bears no aspect.
	Edict bool
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
	case Fleeting:
		return h.fleeting()
	case Omenless:
		return fmt.Sprintf("%s names the omen a machine-spirit is enacted into or tried for, and is spoken only "+
			"in the words of its \"enact\" and of its \"trials\"; nowhere else is an omen being set or tried, "+
			"and the servitor will not illuminate it as nothing", h.Mark)
	case Aspectless:
		which := "the aspect being invoked"
		if h.Name == "former" {
			which = "the former aspect"
		}
		return fmt.Sprintf("%s calls upon %s, yet an edict bears no aspect: it is proclaimed whole, "+
			"never invoked into an aspect, so there is neither an aspect nor a former aspect to "+
			"illuminate — %s", h.Mark, which, h.speakable())
	}
	if h.Mode == RiteMode && slices.Contains(voxMarks, h.Name) {
		return fmt.Sprintf("%s may be spoken only in the tidings of a vox-cast; within the liturgy "+
			"of a rite it is heresy", h.Mark)
	}
	return fmt.Sprintf("%s is no placeholder known to the codex and shall not be illuminated as "+
		"nothing; %s", h.Mark, h.speakable())
}

// speakable recites the marks the words of the heresy may speak.
func (h Heresy) speakable() string {
	marks, words := riteMarks, "the words of a rite"
	if h.Edict {
		marks, words = edictMarks, "the words of an edict"
	}
	if h.Mode == VoxCastMode {
		marks, words = append(slices.Clone(marks), voxMarks...), "the tidings of a vox-cast"
		if h.Edict {
			words = "the tidings of an edict's vox-cast"
		}
	}
	last := len(marks) - 1
	return fmt.Sprintf("%s may speak only {{%s}} and {{%s}}", words,
		strings.Join(marks[:last], "}}, {{"), marks[last])
}

// riteMarks are the marks every rite may speak, in the order of the codex;
// edictMarks are those an edict may speak.
var (
	riteMarks  = []string{"aspect", "former", "rite.name", inscriptionPrefix + "<key>"}
	edictMarks = riteMarks[2:]
)

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

// InscriptionVariable names the variable of the environment that carries
// the inscription key: SERVITOR_INSCRIPTION_ and the key in capitals, every
// "-" and "." made "_".
func InscriptionVariable(key string) string {
	return "SERVITOR_INSCRIPTION_" + strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(key))
}

// fleeting denounces a word given only at invocation in the words of an
// exalted verse, naming the variable that bears it to exalted speech.
func (h Heresy) fleeting() string {
	const fixed = "yet the words of an exalted verse are fixed when its master sanctions them, and nothing given " +
		"at invocation is spliced into them"
	if h.Name == "former" {
		return fmt.Sprintf("%s names the aspect the machine stood in before, a word known only at invocation, %s; "+
			"exalted speech hears it only from its environment — speak it as \"$SERVITOR_FORMER_ASPECT\"",
			h.Mark, fixed)
	}
	key := strings.TrimPrefix(h.Name, inscriptionPrefix)
	return fmt.Sprintf("%s calls upon the inscription %q, which is mandatory yet not decreed for every "+
		"invocation, so that its word may come from its rune alone, %s; an inscription is spliced into exalted "+
		"words from its decrees alone — decree it for every aspect, or speak it as \"$%s\"",
		h.Mark, key, fixed, InscriptionVariable(key))
}
