package invocation

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// transcription examines a transcription step: its vessel, its scripture
// for the invoked aspect, and whether the vessel standing there is its own.
func (p *preflight) transcription(s *librarium.Transcription) performer {
	vessel, okVessel := p.path("transcription", s.Vessel)
	text, null, okText := p.scripture("scripture", s.Scripture)
	if !okVessel || !okText {
		return nil
	}
	t := transcribed{vessel: vessel, text: text, null: null, seal: s.Seal, zeal: s.Zeal}
	if !s.Zeal {
		t.known = p.known(s.Scripture)
	}
	return p.vessel(p.verse(s.Kind(), vessel), "transcription", t.plan)
}

// transcribed is a transcription made ready by the pre-flight.
type transcribed struct {
	vessel string
	text   string
	null   bool         // the vessel is struck
	seal   *fs.FileMode // nil keeps the seal of a standing vessel
	zeal   bool
	known  []string // the scripture of every aspect, by which the vessel is recognised
}

// plan computes the vessel transcribed anew or struck. A standing vessel is
// replaced only when its scripture is that of an aspect, or with zeal.
func (t transcribed) plan(w world) (VesselChange, error) {
	real, e, err := follow(w, t.vessel)
	if err != nil {
		return VesselChange{}, lament(t.vessel, err)
	}
	c := VesselChange{Path: real, Before: e.content, Existed: e.form == scroll, BeforeSeal: e.mode}
	switch e.form {
	case absent:
		if t.null {
			return c, nil
		}
		if err := hallAwaits(w, real); err != nil {
			return VesselChange{}, err
		}
		c.AfterSeal = newSeal
	case scroll:
		if !t.zeal && !recognised(e.content, t.known) {
			return VesselChange{}, fmt.Errorf("the vessel %s holds scripture of no aspect of this rite; the "+
				"servitor keeps no copy of what it overwrites, so it will %s it only with \"zeal\": true",
				real, map[bool]string{true: "strike", false: "replace"}[t.null])
		}
		c.AfterSeal = e.mode
	default:
		return VesselChange{}, fmt.Errorf("%s is no vessel that can hold transcribed scripture", real)
	}
	if t.seal != nil {
		c.AfterSeal = *t.seal
	}
	if !t.null {
		c.Exists, c.After = true, t.text
	}
	return c, nil
}

// recognised reports whether content is one of the known scriptures; one
// final newline is forgiven.
func recognised(content string, known []string) bool {
	content = strings.TrimSuffix(content, "\n")
	for _, k := range known {
		if strings.TrimSuffix(k, "\n") == content {
			return true
		}
	}
	return false
}
