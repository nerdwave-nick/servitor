package tui

import (
	"context"
	"strconv"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// The cogitator draws every rite at once, as the first telling of a vigil
// reads it: without its auspexes, a subject they read shown by the omen its
// data-slate recorded, stale, where one serves. The rest of the vigil then
// awakens the auspexes of the rites awaited, and each rite is drawn anew as
// they settle. Every reading begins a vigil of its own and slays the one
// before; what a vigil of an older reading tells is not heeded.

type (
	// settledMsg is a rite of the reading gen whose auspexes settled.
	settledMsg struct {
		gen     int
		reading rituals.Reading
	}
	// attendedMsg tells that the vigil of the reading gen has ended.
	attendedMsg struct{ gen int }
)

// vigil is the vigil of one reading of the Librarium.
type vigil struct {
	gen       int
	halt      context.CancelFunc // ends it, slaying its auspexes
	attending bool               // its auspexes still labour, or are yet to be awakened
	pending   func()             // awakens them; nil once awakened
}

// read reads every rite of the Librarium at once, without its auspexes,
// and leaves the vigil of the reading to awaken them once the cogitator
// listens (watch); the vigil of the reading before is slain.
func (m *model) read(lib *librarium.Librarium, kept map[string]*rituals.Sanctioned) {
	m.vigil.halt()
	ctx, halt := context.WithCancel(context.Background())
	m.vigil = vigil{gen: m.vigil.gen + 1, halt: halt}
	readings := map[string]rituals.Reading{}
	awaited := m.s.Herald(ctx, false, func(h rituals.Heralding) {
		if h.Reading != nil {
			readings[h.Entry.Rite] = *h.Reading
		}
	})
	m.kept = kept
	m.all = m.all[:0]
	for _, name := range lib.Names() {
		m.all = append(m.all, m.rowOf(name, readings[name]))
	}
	if len(awaited) == 0 {
		return
	}
	s, gen := m.s, m.vigil.gen
	m.vigil.attending = true
	m.vigil.pending = func() {
		send := m.send
		m.watching.Go(func() {
			s.Attend(ctx, awaited, func(h rituals.Heralding) { send(settledMsg{gen: gen, reading: *h.Reading}) })
			send(attendedMsg{gen: gen})
		})
	}
}

// rowOf is the row of the rite name read as reading.
func (m *model) rowOf(name string, reading rituals.Reading) row {
	lib := m.s.Librarium
	r := row{name: name, reading: reading, paths: lib.Scriptures[name], sanction: reading.Sanction, kept: m.kept[name]}
	if r.kept != nil && r.kept.Standing == sanction.Forsaken {
		r.sanction = sanction.Forsaken
	}
	for _, f := range lib.Findings {
		if f.Rite == name && f.Severity == librarium.Heresy {
			r.heresies = append(r.heresies, f)
		}
	}
	return r
}

// watch awakens the auspexes of the current reading, once the cogitator
// listens to what they tell.
func (m *model) watch() {
	if p := m.vigil.pending; p != nil {
		m.vigil.pending = nil
		p()
	}
}

// settled draws anew the rite whose auspexes settled, unless an older
// reading awaited them.
func (m *model) settled(msg settledMsg) {
	if msg.gen != m.vigil.gen {
		return
	}
	for i := range m.all {
		if m.all[i].name == msg.reading.Augury.Rite {
			m.all[i] = m.rowOf(m.all[i].name, msg.reading)
		}
	}
	m.applyFilter(nameOf(m.current()))
}

// attended ends the vigil of the reading gen.
func (m *model) attended(msg attendedMsg) {
	if msg.gen == m.vigil.gen {
		m.vigil.attending = false
	}
}

// forsake slays the vigil of the current reading, and returns once none
// labours.
func (m *model) forsake() {
	m.vigil.pending = nil
	m.vigil.halt()
	m.watching.Wait()
}

// age tells the duration d in its greatest whole unit: seconds, minutes,
// hours or days.
func age(d time.Duration) string {
	switch d = max(d, 0); {
	case d < time.Minute:
		return strconv.Itoa(int(d/time.Second)) + "s"
	case d < time.Hour:
		return strconv.Itoa(int(d/time.Minute)) + "m"
	case d < 48*time.Hour:
		return strconv.Itoa(int(d/time.Hour)) + "h"
	}
	return strconv.Itoa(int(d/(24*time.Hour))) + "d"
}
