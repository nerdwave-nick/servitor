// Package lexicon holds the one vocabulary of servitor, the liturgy of the
// Adeptus Mechanicus: the words shared by the rituals and the cogitator, and
// the Thoughts for the Day.
package lexicon

import (
	_ "embed"
	"strings"
)

// Standings of a rite, as named by the census and the cogitator.
const (
	Performed = "performed"
	Dormant   = "dormant"
	Corrupted = "corrupted"
	Heretical = "heretical"
)

// Severities of the Inquisition's findings.
const (
	Heresy   = "heresy"
	Impurity = "impurity"
)

// thoughtsFile holds one Thought for the Day per line. Blank lines and lines
// starting with "#" are ignored.
//
//go:embed thoughts.txt
var thoughtsFile string

// Thoughts are the Thoughts for the Day shown by the cogitator.
var Thoughts = parseThoughts(thoughtsFile)

// parseThoughts extracts the thoughts from the embedded file. An empty file
// is a build defect, so it panics at startup rather than failing later.
func parseThoughts(s string) []string {
	var out []string
	for line := range strings.Lines(s) {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			out = append(out, line)
		}
	}
	if len(out) == 0 {
		panic("lexicon: thoughts.txt holds no thoughts")
	}
	return out
}

// Thought returns the Thought for the Day with index n (any integer).
func Thought(n int) string {
	i := n % len(Thoughts)
	if i < 0 {
		i += len(Thoughts)
	}
	return Thoughts[i]
}
