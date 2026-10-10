package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/config"
)

// amend opens the consecration wizard upon the rite of the row, to amend
// it (e) or to replicate it (c). The wizard still writes the elder
// scripture until it learns pattern Mark I; a rite of that pattern is
// amended by hand meanwhile.
func (m *model) amend(key string, r *row) tea.Cmd {
	sw := config.Load(m.dir).Switches[r.name]
	if sw == nil {
		return m.notify(toastInfo, "The consecration wizard cannot yet read scripture of pattern Mark I; "+
			"amend it with o ($EDITOR).")
	}
	d := draftFrom(sw)
	if key == "c" {
		d.origName, d.origPath, d.name = "", "", r.name+"-copy"
	}
	return m.openWizard(d)
}

// formatLoc places an elder diagnostic for the wizard's review.
func formatLoc(d config.Diagnostic) string {
	if d.Line > 0 {
		return fmt.Sprintf("%d:%d", d.Line, d.Col)
	}
	return shortPath(d.File)
}
