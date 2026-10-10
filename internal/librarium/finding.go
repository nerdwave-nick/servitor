package librarium

import (
	"fmt"
	"sort"
)

// Severity says whether a finding must be purged or is only noted.
type Severity int

const (
	// Heresy makes a rite heretical: it cannot be invoked until purged.
	Heresy Severity = iota
	// Impurity is noted but forgiven; the rite can still be invoked.
	Impurity
)

// String is the grimdark name of the severity.
func (s Severity) String() string {
	if s == Impurity {
		return "impurity"
	}
	return "heresy"
}

// Position locates a finding within a scripture, both counted from one;
// the column counts bytes.
type Position struct {
	Line, Column int
}

// Finding is one heresy or impurity in a rite's scripture.
type Finding struct {
	Severity  Severity
	Scripture string // path of the scripture the finding is in
	Rite      string // name of the rite
	Position
	Message string // verbose grimdark denunciation, without position
}

// String renders the finding as path:line:column: severity: message.
func (f Finding) String() string {
	return fmt.Sprintf("%s:%d:%d: %s: %s", f.Scripture, f.Line, f.Column, f.Severity, f.Message)
}

// Findings is every finding of one or more scriptures.
type Findings []Finding

// Heretical reports whether any finding is a heresy.
func (fs Findings) Heretical() bool {
	for _, f := range fs {
		if f.Severity == Heresy {
			return true
		}
	}
	return false
}

// Sort orders the findings by scripture, line and column.
func (fs Findings) Sort() {
	sort.SliceStable(fs, func(i, j int) bool {
		a, b := fs[i], fs[j]
		if a.Scripture != b.Scripture {
			return a.Scripture < b.Scripture
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Column < b.Column
	})
}
