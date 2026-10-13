package tui

import (
	"strings"
)

// spiritGrid is the grid of the page p of the machine-spirit being
// written: its omens, its auspex, or its enact words.
func (w *wizard) spiritGrid(p pageKind) grid {
	sp := &w.subject.spirit
	omens := w.d.declaredOmens(*sp)
	words := func(label, hint string) func(f *form, at, where string, x string) {
		return func(f *form, at, where string, x string) {
			f.addText("omen:"+at, label+" "+where, x, "", hint, nil)
		}
	}
	read := func(f *form, at string) string { return strings.TrimSpace(f.get("omen:" + at)) }
	switch p {
	case pageAuspex:
		return auspexGrid{sp: sp, exalted: w.work.fixed["exalted"] == "true", trials: omenGrid[string]{v: &sp.trials,
			keys: omens, noun: "omen", shares: "is tried by the trial of every other omen", read: read,
			add: words("Trial of", "Words that end with the death-mark 0 while the subject shows this omen; {{omen}} "+
				"and $SERVITOR_OMEN name it. Empty is none: the omen remains when every trial fails.")}}
	case pageEnact:
		return omenGrid[string]{v: &sp.enact, keys: omens, noun: "omen", shares: "is enacted by the words of every other omen",
			read: read, add: words("Words that enact", "Words that set the subject to this omen; {{omen}} and "+
				"$SERVITOR_OMEN name it.")}
	}
	return spiritOmens{sp: sp, rows: omenGrid[string]{v: &sp.omens, keys: w.d.aspectList(), noun: "aspect",
		shares: "bears the omen of every other aspect", read: read,
		add: words("Omen of", "The words the subject shows in this aspect; {{aspect}} names the aspect whose omen it is.")}}
}

// spiritOmens is the page of the omens of a machine-spirit: whether each
// aspect is its own omen, else a row for each aspect.
type spiritOmens struct {
	sp   *spiritDraft
	rows omenGrid[string]
}

func (g spiritOmens) page(f *form) {
	f.addToggle("named", "Each aspect is its own omen", g.sp.named,
		"The scan prints the names of aspects, and no omens are written. space turns it.")
	if !g.sp.named {
		g.rows.page(f)
	}
}

func (g spiritOmens) keep(f *form) {
	if f.has("same:" + g.rows.keys[0]) {
		g.rows.keep(f)
	}
	g.sp.named = f.get("named") == "true"
}

func (g spiritOmens) turned(f *form) string {
	if (f.get("named") == "true") == f.has("same:"+g.rows.keys[0]) {
		return "named"
	}
	if g.sp.named {
		return ""
	}
	return g.rows.turned(f)
}

// auspexGrid is the page of the auspex of a machine-spirit: its scan, a
// trial for each omen, its patience, and, while the subject is exalted,
// whether the exalted servitor speaks it.
type auspexGrid struct {
	sp      *spiritDraft
	exalted bool
	trials  omenGrid[string]
}

func (g auspexGrid) page(f *form) {
	f.addText("scan", "Scan", g.sp.scan, "cat /sys/devices/system/cpu/smt/control",
		"Words that print the omen the subject shows. With trials beside it, the scan reads and the trials "+
			"judge the subject once set. Leave the scan and every trial empty for a subject never read.", nil)
	g.trials.page(f)
	f.addText("patience", "Patience of the auspex", g.sp.patience, "2s",
		"How long the scan and the trials may labour, such as 500ms or 5s.", validPatience)
	if g.exalted {
		f.addToggle("privileged", "Exalted auspex", g.sp.privileged, "Spoken by the exalted servitor at "+
			"invocation alone, for a subject only the machine's own authority may read; space turns it.")
	}
}

func (g auspexGrid) keep(f *form) {
	g.sp.scan, g.sp.patience = strings.TrimSpace(f.get("scan")), strings.TrimSpace(f.get("patience"))
	if f.has("privileged") {
		g.sp.privileged = f.get("privileged") == "true"
	}
	g.trials.keep(f)
}

func (g auspexGrid) turned(f *form) string { return g.trials.turned(f) }
