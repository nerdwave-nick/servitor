package invocation

import (
	"context"
	"io/fs"
)

// newSeal is the seal of a vessel the servitor brings into being, unless
// a transcription names another.
const newSeal fs.FileMode = 0o644

// planner computes what a step does to its vessel in a given world.
type planner func(w world) (VesselChange, error)

// vesselStep is what sanctums and transcriptions share: a change of one
// vessel planned in the pre-flight and planned afresh against the machine
// itself when performed, so that it builds on what the steps before it did.
type vesselStep struct {
	v       Verse
	plan    planner
	planned VesselChange
	done    *VesselChange // what perform changed, for reversion
}

// vessel plans a step against the pre-flight's shadow and records its
// change there; a failing plan is denounced at field.
func (p *preflight) vessel(v Verse, field string, plan planner) performer {
	c, err := plan(p.world)
	if err != nil {
		p.denounce(field, "%v", err)
		return nil
	}
	p.world.set(c.Path, entryAfter(c))
	return &vesselStep{v: v, plan: plan, planned: c}
}

func entryAfter(c VesselChange) entry {
	if !c.Exists {
		return entry{form: absent}
	}
	return entry{form: scroll, content: c.After, mode: c.AfterSeal}
}

func (s *vesselStep) verse() Verse { return s.v }

func (s *vesselStep) foresee() Foresight {
	c := s.planned
	return Foresight{Verse: s.v, Vessel: &c}
}

func (s *vesselStep) perform(context.Context) error {
	c, err := s.plan(disk{})
	if err != nil {
		return err
	}
	if !c.Changed() {
		return nil
	}
	if c.Exists {
		err = writeScroll(c.Path, c.After, c.AfterSeal)
	} else {
		err = strike(c.Path)
	}
	if err != nil {
		return lament(c.Path, err)
	}
	s.done = &c
	return nil
}

func (s *vesselStep) revert() (bool, error) {
	if s.done == nil {
		return false, nil
	}
	c := s.done
	before := entry{form: absent}
	if c.Existed {
		before = entry{form: scroll, content: c.Before, mode: c.BeforeSeal}
	}
	return true, lament(c.Path, restore(c.Path, before))
}
