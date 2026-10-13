package librarium

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// spirits converts a rite of speech alone: each incantation or litany
// becomes a machine-spirit without omens, enacting its words per aspect;
// its reversion is struck, for a machine-spirit is restored by enacting
// the omen it showed before. The rite's auspex becomes the scan of its only
// machine-spirit, and is struck when there are several.
func (c *converting) spirits() {
	for i, s := range c.from.Liturgy {
		var u Utterance
		m := &MachineSpirit{}
		switch s := s.(type) {
		case *Incantation:
			m.Enact, u = s.Command, s.Utterance
		case *Litany:
			m.Enact, u = c.recited(i, len(c.to.Subjects)), s.Utterance
		default:
			continue
		}
		m.Tongue, m.Patience, m.Relics, m.Exalted = u.Tongue, u.Patience, u.Relics, u.Exalted
		if !u.Reversion.IsZero() {
			c.note(fmt.Sprintf("/subjects/%d", len(c.to.Subjects)),
				fmt.Sprintf("pattern %q spoke the reversion %s should a later verse fall;", Pattern, quoteMap(u.Reversion)),
				"it is struck, for a machine-spirit is restored by enacting the omen it showed before")
			c.changes = append(c.changes, fmt.Sprintf("the reversion of verse %d is struck", c.from.verse(i)))
		}
		c.to.Subjects = append(c.to.Subjects, Subject{Name: c.nameOf(i), Matter: m})
	}
	a := c.from.Auspex
	switch {
	case a == nil:
	case len(c.to.Subjects) == 1:
		c.to.Subjects[0].Matter.(*MachineSpirit).Auspex = &SpiritAuspex{Scan: a.Rite, Patience: a.Patience}
	default:
		c.strikeAuspex(a, fmt.Sprintf("it read the whole rite, and none of its %d machine-spirits may bear it alone",
			len(c.to.Subjects)))
	}
}

// recited converts the litany of verse i, which becomes the subject at
// index k, into enact words that call its scroll with its offerings, per
// aspect.
func (c *converting) recited(i, k int) AspectMap[string] {
	s := c.from.Liturgy[i].(*Litany)
	groups := c.groups(s)
	m := AspectMap[string]{entries: []Entry[string]{}}
	for _, aspects := range groups {
		scroll, _ := s.Scroll.For(aspects[0])
		words := []string{shellWord(scroll, true)}
		for _, o := range s.Offerings {
			offering, _ := o.For(aspects[0])
			words = append(words, shellWord(offering, false))
		}
		if len(groups) == 1 && len(aspects) == len(c.from.Aspects) {
			aspects = []string{Fallback}
		}
		for _, a := range aspects {
			m.entries = append(m.entries, Entry[string]{Aspect: a, Value: strings.Join(words, " ")})
		}
	}
	c.note(fmt.Sprintf("/subjects/%d", k), fmt.Sprintf("pattern %q recited the scroll %s with its offerings;",
		Pattern, quoteMap(s.Scroll)), "its enact words now call that scroll, spoken in the hall of the rite")
	c.changes = append(c.changes, fmt.Sprintf("the litany of verse %d is spoken as enact words", c.from.verse(i)))
	return m
}

// plainWord is a word the tongue reads as written.
var plainWord = regexp.MustCompile(`^[A-Za-z0-9_./~${}:=,@%+-]+$`)

// shellWord writes w as one word for the tongue: as written when plain,
// else quoted, a leading "~/" left outside the quotes to be borne home. A
// scroll is a path: a relative one is called within the hall.
func shellWord(w string, scroll bool) string {
	if scroll && !filepath.IsAbs(w) && !strings.HasPrefix(w, "~") && !strings.HasPrefix(w, "$") &&
		!strings.HasPrefix(w, "./") && !strings.HasPrefix(w, "../") {
		w = "./" + w
	}
	if plainWord.MatchString(w) {
		return w
	}
	home := ""
	if strings.HasPrefix(w, "~/") {
		home, w = "~/", w[2:]
	}
	return home + "'" + strings.ReplaceAll(w, "'", `'\''`) + "'"
}

// quoteMap quotes the words of m as they were written: one word, or an
// object keyed by aspect.
func quoteMap(m AspectMap[string]) string {
	if m.IsUniform() {
		return strconv.Quote(m.entries[0].Value)
	}
	parts := make([]string, len(m.entries))
	for i, e := range m.entries {
		parts[i] = strconv.Quote(e.Aspect) + ": " + strconv.Quote(e.Value)
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
