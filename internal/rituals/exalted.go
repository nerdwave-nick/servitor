package rituals

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// scrollLimit is how much of a scroll the inquisition reads.
const scrollLimit = 1 << 20

// exaltation examines what the exalted verses of the rite read beside
// them, their relics, and the words every litany's scroll speaks.
func (e *examiner) exaltation() {
	amid := e.rite.Exalted()
	for i, step := range e.rite.Liturgy {
		var u librarium.Utterance
		switch st := step.(type) {
		case *librarium.Incantation:
			u = st.Utterance
		case *librarium.Litany:
			u = st.Utterance
			sp := librarium.Speech{Verse: i + 1, Exalted: u.Exalted, Relics: u.Relics, AmidExalted: amid}
			for _, aspect := range e.aspects() {
				text, ok := st.Scroll.For(aspect)
				if !ok {
					continue
				}
				if p, ok := e.path(aspect, text); ok && e.once("scroll "+p) {
					e.speech(p, sp)
				}
			}
		}
		for j, relic := range u.Relics {
			e.relic(fmt.Sprintf("/liturgy/%d/relics/%d", i, j), fmt.Sprintf("relic %d of the %s of verse %d, %s,",
				j+1, e.rite.Liturgy[i].Kind().Key(), i+1, relic), relic)
		}
		if u.Exalted {
			e.exaltedTongue(fmt.Sprintf("/liturgy/%d", i), u.Tongue, fmt.Sprintf("verse %d", i+1), "step")
		}
	}
	e.subjectsExaltation()
}

// exaltedTongue denounces the tongue of the exalted speech of place, a
// holder written at base, named by it, the rite or the settings, when the
// program it truly is is none of the tongues exalted speech knows. A tongue
// spoken nowhere is noted by the machine.
func (e *examiner) exaltedTongue(base, tongue, place, holder string) {
	ptr := base + "/tongue"
	switch {
	case tongue == "" && e.rite.Tongue != "":
		ptr = "/tongue"
	case tongue == "" && e.orders.Tongue != "":
		ptr = base + "/exalted" // the settings speak for the speaker
	}
	t := e.tongueOf(tongue)
	program, err := exec.LookPath(t)
	if err == nil {
		program, err = filepath.EvalSymlinks(program)
	}
	if err != nil {
		return
	}
	if _, known := librarium.ExaltedTongue(program); !known {
		e.note(librarium.Heresy, ptr, "the tongue %s of %s is %s, which exalted speech is never spoken in: "+
			"it knows only %s, each bidden to read no scripture of its own as it awakens — name one of them as "+
			"the \"tongue\" of the %s", t, place, filepath.Base(program), librarium.ExaltedTongueNames, holder)
	}
}

// relic denounces the relic written at ptr, whose words name what, when it
// is no scripture standing within the rite's hall.
func (e *examiner) relic(ptr, what, relic string) {
	hall := e.rite.Dir()
	p := filepath.Join(hall, relic)
	info, err := os.Stat(p)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		e.note(librarium.Heresy, ptr, "%s is nowhere to be found beside the rite: no sanction can seal its words, "+
			"and the exalted speech would read nothing there — lay it there, or strike it from the relics", what)
		return
	case err != nil:
		e.note(librarium.Heresy, ptr, "%s resists every attempt to read it: no sanction can seal its words", what)
		return
	case !info.Mode().IsRegular():
		e.note(librarium.Heresy, ptr, "%s is no scripture but a hall or another thing of the machine: a relic is "+
			"one scripture, whose words are sealed into the sanction — name each scripture within it", what)
		return
	}
	real, errHall := filepath.EvalSymlinks(hall)
	target, errTarget := filepath.EvalSymlinks(p)
	if rel, err := filepath.Rel(real, target); errHall == nil && errTarget == nil && (err != nil ||
		rel == ".." || strings.HasPrefix(rel, "../")) {
		e.note(librarium.Heresy, ptr, "%s leads to %s, which lies outside the hall of the rite: a relic is laid "+
			"beside the rite's scripture itself, never tethered from elsewhere — lay the scripture there", what, target)
	}
}

// speech notes the impurities of the words the scroll at p speaks, each
// once, at the line it is first found in.
func (e *examiner) speech(p string, sp librarium.Speech) {
	f, err := os.Open(p)
	if err != nil {
		return // the machine examines whether the scroll stands
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, scrollLimit))
	if err != nil {
		return
	}
	seen := map[string]bool{}
	for n, line := range strings.Split(string(data), "\n") {
		for _, msg := range sp.Impurities(line) {
			if !seen[msg] {
				seen[msg] = true
				e.found = append(e.found, librarium.Finding{Severity: librarium.Impurity, Scripture: p,
					Rite: e.rite.Name, Position: librarium.Position{Line: n + 1, Column: 1},
					Message: "within the scroll: " + msg})
			}
		}
	}
}
