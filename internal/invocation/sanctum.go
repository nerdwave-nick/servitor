package invocation

import (
	"fmt"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanctum"
)

// sanctum examines a sanctum step: its vessel, its scripture for the
// invoked aspect and the vessel's present sanctum.
func (p *preflight) sanctum(s *librarium.Sanctum) performer {
	vessel, okVessel := p.path("sanctum", s.Vessel)
	text, null, okText := p.scripture("scripture", s.Scripture)
	if null {
		p.denounce(mapField(p, "scripture", s.Scripture), "the scripture of a sanctum may not be null: a sanctum "+
			"is never struck from its vessel, for its marker records the aspect — write \"\" to leave it empty")
		return nil
	}
	if !okVessel || !okText {
		return nil
	}
	glyph, closing := s.Glyphs()
	m := sanctum.Marker{Glyph: glyph, ClosingGlyph: closing, Ward: s.WardFor(p.rite.Name)}
	for _, line := range strings.Split(text, "\n") {
		if m.IsMarkerLine(line) {
			p.denounce(mapField(p, "scripture", s.Scripture), "the scripture would carry a marker of its own "+
				"sanctum %q (%q) into the vessel and break the sanctum asunder", m.Ward, line)
			return nil
		}
	}
	aspect, consecrate := p.opts.Aspect, s.Consecrate
	return p.vessel(p.verse(s.Kind(), vessel), "sanctum", func(w world) (VesselChange, error) {
		return planSanctum(w, vessel, consecrate, m, aspect, text)
	})
}

// planSanctum computes the vessel with the sanctum of m rewritten for
// aspect, consecrating the vessel when allowed.
func planSanctum(w world, vessel string, consecrate bool, m sanctum.Marker, aspect, text string) (VesselChange, error) {
	real, e, err := follow(w, vessel)
	if err != nil {
		return VesselChange{}, lament(vessel, err)
	}
	c := VesselChange{Path: real, Before: e.content, Existed: e.form == scroll, Exists: true,
		BeforeSeal: e.mode, AfterSeal: e.mode}
	switch e.form {
	case absent:
		if !consecrate {
			return VesselChange{}, fmt.Errorf("no vessel stands at %s; set \"consecrate\": true to bring it "+
				"into being", real)
		}
		if err := hallAwaits(w, real); err != nil {
			return VesselChange{}, err
		}
		c.AfterSeal = newSeal
	case scroll:
	default:
		return VesselChange{}, fmt.Errorf("%s is no vessel that can hold a sanctum", real)
	}
	after, err := sanctum.Upsert(e.content, m, aspect, text)
	if err != nil {
		return VesselChange{}, fmt.Errorf("the vessel %s: %w", real, err)
	}
	c.After = after
	return c, nil
}

// hallAwaits denounces a place whose directory is absent: the servitor
// never builds the halls its vessels and tethers are to stand in.
func hallAwaits(w world, place string) error {
	ok, err := hallExists(w, place)
	if err != nil {
		return lament(place, err)
	}
	if !ok {
		return fmt.Errorf("the hall that is to hold %s is absent, and the servitor does not raise halls", place)
	}
	return nil
}
