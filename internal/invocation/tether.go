package invocation

import (
	"fmt"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// tether examines a tether step: the bound name, the anchor of the invoked
// aspect, and what stands at the name now.
func (p *preflight) tether(s *librarium.Tether) performer {
	name, okName := p.path("tether", s.Name)
	anchor, has := s.Anchor.For(p.opts.Aspect)
	field := mapField(p, "anchor", s.Anchor)
	if !has {
		p.denounce(field, "no anchor is written for the aspect %q, and no \"*\" serves in its stead", p.opts.Aspect)
		return nil
	}
	t := &tethered{name: name, unbind: anchor.Null, zeal: s.Zeal}
	okAnchor := true
	if !anchor.Null {
		t.anchor, okAnchor = p.path(field, anchor.Path)
	}
	if !okName || !okAnchor {
		return nil
	}
	t.v = p.verse(s.Kind(), name)
	c, _, err := t.plan(p.world)
	if err != nil {
		p.denounce("tether", "%v", err)
		return nil
	}
	t.planned = c
	after := entry{form: absent}
	if !t.unbind {
		after = entry{form: bond, anchor: t.anchor}
	}
	p.world.set(name, after)
	return t
}

// tethered is a tether made ready by the pre-flight. The bound name itself
// is never followed: the tether is the bond.
type tethered struct {
	v       Verse
	name    string
	anchor  string // "" when unbound
	unbind  bool
	zeal    bool
	planned TetherChange
	done    *entry // what stood at the name before perform changed it
}

// plan computes the change of the bond and returns what stands at the name.
// Only bonds are replaced or struck, unless with zeal a scroll gives way;
// halls never do.
func (t *tethered) plan(w world) (TetherChange, entry, error) {
	e, err := w.entry(t.name)
	if err != nil {
		return TetherChange{}, entry{}, lament(t.name, err)
	}
	c := TetherChange{Name: t.name, To: t.anchor}
	switch e.form {
	case bond:
		c.From = e.anchor
	case absent:
		if !t.unbind {
			if err := hallAwaits(w, t.name); err != nil {
				return TetherChange{}, entry{}, err
			}
		}
	case scroll:
		if !t.zeal {
			return TetherChange{}, entry{}, fmt.Errorf("a vessel that is no tether stands at %s; the servitor "+
				"keeps no copy of what it casts down, so it will make it give way only with \"zeal\": true", t.name)
		}
		c.Displaced = true
	default:
		return TetherChange{}, entry{}, fmt.Errorf("a hall stands at %s; the servitor will not cast down a hall "+
			"to make room for a tether, not even in zeal", t.name)
	}
	return c, e, nil
}

func (t *tethered) verse() Verse { return t.v }

func (t *tethered) foresee() Foresight {
	c := t.planned
	return Foresight{Verse: t.v, Tether: &c}
}

func (t *tethered) perform() error {
	c, before, err := t.plan(disk{})
	if err != nil {
		return err
	}
	if !c.Changed() {
		return nil
	}
	if t.unbind {
		err = strike(t.name)
	} else {
		err = bind(t.name, t.anchor)
	}
	if err != nil {
		return lament(t.name, err)
	}
	t.done = &before
	return nil
}

func (t *tethered) revert() (bool, error) {
	if t.done == nil {
		return false, nil
	}
	return true, lament(t.name, restore(t.name, *t.done))
}
