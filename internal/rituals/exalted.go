package rituals

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// unexalted refuses the invocation of a rite holding exalted verses, with a
// heresy for each of them: no exalted verse is ever performed with its
// invoker's authority, and this servitor cannot yet raise an exalted
// servitor to perform one from a sanction. It is the one gate of that
// refusal, passed before anything of the rite is touched; foresight never
// reaches it.
func unexalted(r *librarium.Rite) error {
	var hs invocation.Heresies
	for _, verse := range r.ExaltedVerses() {
		hs = append(hs, invocation.Heresy{Verse: verse, Finding: librarium.Finding{
			Severity: librarium.Heresy, Scripture: r.Path, Rite: r.Name,
			Position: r.Locate(fmt.Sprintf("/liturgy/%d/exalted", verse-1)),
			Message: fmt.Sprintf("verse %d is exalted: it is performed with the machine's own authority alone, by an "+
				"exalted servitor from the sanction of its master, and this servitor performs no exalted verse "+
				"without that sanction — not one verse of %q was performed", verse, r.Name),
		}})
	}
	if len(hs) > 0 {
		return hs
	}
	return nil
}

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
			e.relic(i, j, relic)
		}
	}
}

// relic denounces relic j of the step at index i when it is no scripture
// standing within the rite's hall.
func (e *examiner) relic(i, j int, relic string) {
	hall := e.rite.Dir()
	p := filepath.Join(hall, relic)
	ptr := fmt.Sprintf("/liturgy/%d/relics/%d", i, j)
	what := fmt.Sprintf("relic %d of the %s of verse %d, %s,", j+1, e.rite.Liturgy[i].Kind().Key(), i+1, relic)
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
