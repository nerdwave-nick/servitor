// Package sanction draws the canonical exalted unit of a rite and its
// sigils, and reads the sanctions its master laid in the store.
//
// The unit (Build) holds everything a rite's exalted verses do — or the
// exalted subjects and observances of a state rite of pattern Mark II: the
// pattern of its form, the master's uid and the rite's name; for every
// aspect (once for an edict) a Plan of the exalted verses resolved for it —
// their verse numbers, rendered words and reversions, the absolute tongue,
// the patience, the bytes of their scrolls, relics, tomes and transcribed
// scripture, seal and zeal — and the words of every inscription spliced into
// them, taken from its decrees; and the variables of the environment exalted
// speech hears, with the pattern of the environment the exalted servitor
// grants. Common verses are no part of it: editing a vox-cast or a common
// incantation changes no sigil.
//
// The unit is written as canonical JSON (RFC 8785); its Sigil is the
// SHA-256 of that writing, and SigilOf the sigil of the unit narrowed to one
// aspect's plan. A Sanction is a unit with its sigil and the hour it was
// sanctioned, kept as strict JSON at <root>/<uid>/<rite>.json (Store).
//
// Judge tells a rite's standing towards its sanction: sanctioned, lapsed or
// unsanctioned; a sanction whose rite is gone or holds no exalted verse is
// forsaken, and a rite without exalted verses stands towards none ("").
package sanction

import (
	"crypto/sha256"
	"encoding/hex"
)

// Pattern is the pattern of the form of the unit of a rite of pattern Mark
// I, and PatternMarkII that of a rite of pattern Mark II: a state rite's
// plans hold its exalted subjects and observances (see Verse), an edict's
// its exalted verses as Mark I's do. Both are drawn and read until Mark I
// is retired.
const (
	Pattern       = "Mark I"
	PatternMarkII = "Mark II"
)

// Patterns are the patterns of the units this package draws and reads.
var Patterns = []string{Pattern, PatternMarkII}

// Standing is how a rite stands towards its sanction; it marshals as the
// augury's and the census's "sanction".
type Standing string

// The sanction standings.
const (
	// None is the standing of a rite without exalted verses: it needs no
	// sanction.
	None Standing = ""
	// Sanctioned: the rite's exalted verses match its sanction.
	Sanctioned Standing = "sanctioned"
	// Lapsed: they changed since they were sanctioned, or can no longer be
	// drawn into a unit at all.
	Lapsed Standing = "lapsed"
	// Unsanctioned: the rite holds exalted verses, yet no sanction.
	Unsanctioned Standing = "unsanctioned"
	// Forsaken: a sanction whose rite is gone or holds no exalted verse.
	Forsaken Standing = "forsaken"
)

// Judge tells how a rite holding exalted verses stands towards its sanction
// s (nil when none is kept): unit is its exalted unit as it stands now, nil
// when it cannot be drawn.
func Judge(unit *Unit, s *Sanction) Standing {
	switch {
	case s == nil:
		return Unsanctioned
	case unit == nil || unit.Sigil() != s.Sigil:
		return Lapsed
	}
	return Sanctioned
}

// Short is the first twelve letters of a sigil, by which prompts, argv and
// the chronicle name it.
func Short(sigil string) string {
	if len(sigil) > 12 {
		return sigil[:12]
	}
	return sigil
}

// digest is the sigil of canonical: the hexadecimal SHA-256 of its bytes.
func digest(canonical []byte) string {
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:])
}
