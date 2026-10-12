package sanction

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// ErrUnexalted is the lament for a rite that holds no exalted verse: there is
// nothing to sanction.
var ErrUnexalted = errors.New("the rite holds no exalted verse, and there is nothing to sanction")

// Options are what a unit is drawn with.
type Options struct {
	UID int // the master's
	// Orders are the settings resolved: the tongue and patience of exalted
	// speech that names neither.
	Orders librarium.Orders
	// LookPath finds the program of a tongue; nil is exec.LookPath.
	LookPath func(name string) (string, error)
}

func (o Options) lookPath(name string) (string, error) {
	if o.LookPath == nil {
		return exec.LookPath(name)
	}
	return o.LookPath(name)
}

// Build draws the exalted unit of the rite r, which must not be heretical,
// reading its tomes, scrolls and relics afresh. The error is ErrUnexalted
// for a rite without exalted verses, else names what cannot be drawn.
func Build(r *librarium.Rite, o Options) (*Unit, error) {
	if !r.Exalted() {
		return nil, ErrUnexalted
	}
	u := &Unit{Pattern: Pattern, UID: o.UID, Rite: r.Name, Edict: r.Edict, Environment: environment(r)}
	aspects := r.Aspects
	if r.Edict {
		aspects = []string{""}
	}
	for _, aspect := range aspects {
		d := &drawer{rite: r, opts: o, aspect: aspect, scope: r.ExaltedScope(),
			values: placeholder.Values{Aspect: aspect, Rite: r.Name, Inscriptions: map[string]string{}}}
		for _, in := range r.Inscriptions {
			d.values.Inscriptions[in.Key], _ = in.Decrees.For(aspect)
		}
		plan := Plan{Aspect: aspect, Inscriptions: map[string]string{}}
		for i, step := range r.Liturgy {
			if !librarium.IsExalted(step) {
				continue
			}
			d.verse = i + 1
			v, err := d.draw(step)
			if err != nil {
				return nil, err
			}
			plan.Liturgy = append(plan.Liturgy, v)
		}
		for _, key := range d.spliced {
			plan.Inscriptions[key] = d.values.Inscriptions[key]
		}
		u.Aspects = append(u.Aspects, plan)
	}
	return u, nil
}

// environment names the variables exalted speech of r hears: its rite, its
// aspect and former aspect (never for an edict) and every inscription.
func environment(r *librarium.Rite) Environment {
	keys := []string{"SERVITOR_RITE"}
	if !r.Edict {
		keys = append(keys, "SERVITOR_ASPECT", "SERVITOR_FORMER_ASPECT")
	}
	for _, in := range r.Inscriptions {
		keys = append(keys, placeholder.InscriptionVariable(in.Key))
	}
	slices.Sort(keys)
	return Environment{Pattern: EnvironmentPattern, Keys: slices.Compact(keys)}
}

// drawer draws the exalted verses of a rite for one aspect.
type drawer struct {
	rite    *librarium.Rite
	opts    Options
	aspect  string
	scope   placeholder.Scope
	values  placeholder.Values
	verse   int      // the verse being drawn
	spliced []string // the inscriptions spliced so far, in the order first spoken
}

// lament is why the verse being drawn cannot be drawn into the unit.
func (d *drawer) lament(format string, args ...any) error {
	of := fmt.Sprintf("for the aspect %q", d.aspect)
	if d.rite.Edict {
		of = "for the edict"
	}
	return fmt.Errorf("verse %d of %q cannot be drawn into its sanction %s: %s", d.verse, d.rite.Name, of,
		fmt.Sprintf(format, args...))
}

// render illuminates text with the words fixed at sanction.
func (d *drawer) render(text string) (string, error) {
	for _, key := range d.scope.Spliced(text) {
		if !slices.Contains(d.spliced, key) {
			d.spliced = append(d.spliced, key)
		}
	}
	out, err := d.scope.Render(text, d.values)
	if err != nil {
		return "", d.lament("%v", err)
	}
	return out, nil
}

// words renders what the aspect demands of m.
func (d *drawer) words(m librarium.AspectMap[string]) (string, error) {
	text, ok := m.For(d.aspect)
	if !ok {
		return "", d.lament("nothing is written for it, and no \"*\" serves in its stead")
	}
	return d.render(text)
}

// path renders text and resolves it against the rite's hall.
func (d *drawer) path(text string) (string, error) {
	out, err := d.render(text)
	if err != nil {
		return "", err
	}
	return d.rite.ResolvePath(out), nil
}

// read reads the bytes of the scripture at p.
func (d *drawer) read(p string) ([]byte, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, d.lament("%s cannot be read: %v", p, err)
	}
	return data, nil
}

// draw draws one exalted step.
func (d *drawer) draw(step librarium.Step) (Verse, error) {
	v := Verse{Verse: d.verse, Kind: step.Kind()}
	var err error
	switch st := step.(type) {
	case *librarium.Sanctum:
		v.Ward, v.Consecrate = st.WardFor(d.rite.Name), st.Consecrate
		v.Glyph, v.ClosingGlyph = st.Glyphs()
		if v.Target, err = d.path(st.Vessel); err == nil {
			v.Scripture, v.Null, err = d.scripture(st.Scripture)
		}
	case *librarium.Transcription:
		v.Seal, v.Zeal = st.Seal, st.Zeal
		if v.Target, err = d.path(st.Vessel); err == nil {
			v.Scripture, v.Null, err = d.scripture(st.Scripture)
		}
	case *librarium.Tether:
		v.Zeal = st.Zeal
		if v.Target, err = d.path(st.Name); err == nil {
			err = d.anchor(&v, st.Anchor)
		}
	case *librarium.Incantation:
		if v.Target, err = d.words(st.Command); err == nil {
			err = d.speech(&v, st.Utterance)
		}
	case *librarium.Litany:
		err = d.litany(&v, st)
	}
	return v, err
}

// scripture is what the aspect places in the vessel: inline scripture
// rendered, or the bytes of its tome, illuminated when it says so.
func (d *drawer) scripture(m librarium.AspectMap[librarium.Scripture]) (text []byte, null bool, err error) {
	sc, ok := m.For(d.aspect)
	switch {
	case !ok:
		return nil, false, d.lament("no scripture is written for it, and no \"*\" serves in its stead")
	case sc.Null:
		return nil, true, nil
	case !sc.IsTome():
		out, err := d.render(sc.Text)
		return []byte(out), false, err
	}
	p, err := d.path(sc.Tome)
	if err != nil {
		return nil, false, err
	}
	data, err := d.read(p)
	if err != nil || !sc.Illuminate {
		return data, false, err
	}
	out, err := d.render(string(data))
	return []byte(out), false, err
}

// anchor resolves where the tether leads for the aspect, or that it is
// unbound.
func (d *drawer) anchor(v *Verse, m librarium.AspectMap[librarium.Anchor]) error {
	a, ok := m.For(d.aspect)
	switch {
	case !ok:
		return d.lament("no anchor is written for it, and no \"*\" serves in its stead")
	case a.Null:
		v.Null = true
		return nil
	}
	var err error
	v.Anchor, err = d.path(a.Path)
	return err
}

// litany draws a litany: its scroll as written and its bytes, its
// offerings rendered, and its speech.
func (d *drawer) litany(v *Verse, st *librarium.Litany) error {
	var err error
	if v.Target, err = d.words(st.Scroll); err != nil {
		return err
	}
	if v.Scroll, err = d.read(d.rite.ResolvePath(v.Target)); err != nil {
		return err
	}
	v.Offerings = []string{}
	for _, o := range st.Offerings {
		offering, err := d.words(o)
		if err != nil {
			return err
		}
		v.Offerings = append(v.Offerings, librarium.ExpandHome(offering))
	}
	return d.speech(v, st.Utterance)
}

// speech resolves the tongue (step > rite > settings > bash) to its
// absolute program, the patience (step > settings > default), the
// reversion, and reads the relics.
func (d *drawer) speech(v *Verse, u librarium.Utterance) error {
	tongue := librarium.DefaultTongue
	for _, t := range []string{d.opts.Orders.Tongue, d.rite.Tongue, u.Tongue} {
		if t != "" {
			tongue = t
		}
	}
	program, err := d.opts.lookPath(tongue)
	if err == nil {
		program, err = filepath.Abs(program)
	}
	if err != nil {
		return d.lament("the tongue %q is spoken nowhere on this machine", tongue)
	}
	v.Tongue, v.Patience = program, librarium.DefaultPatience
	for _, p := range []time.Duration{d.opts.Orders.Patience, u.Patience} {
		if p > 0 {
			v.Patience = p
		}
	}
	if !u.Reversion.IsZero() {
		if v.Reversion, err = d.words(u.Reversion); err != nil {
			return err
		}
	}
	v.Relics = []Relic{}
	for _, relic := range u.Relics {
		data, err := d.read(filepath.Join(d.rite.Dir(), relic))
		if err != nil {
			return err
		}
		v.Relics = append(v.Relics, Relic{Relic: relic, Scripture: data})
	}
	return nil
}
