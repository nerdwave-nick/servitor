package exalted

import (
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// The words after Mode that rouse the exalted servitor to keep the store of
// sanctions rather than to perform:
//
//	servitor __exalt --sanction <rite> <sigil12>
//	servitor __exalt --abjure <rite>
//	servitor __exalt --abjure --all
//
// No rite's name begins with a dash, so neither is ever the name of a rite.
const (
	SanctionWord = "--sanction"
	AbjureWord   = "--abjure"
	AllWord      = "--all"
)

// Sanctioning is the first line the master's servitor speaks to an exalted
// servitor roused to sanction: the protocol it speaks, who it is, the rite,
// the whole sigil of the unit to be sanctioned (argv names its first twelve
// letters), and how many bytes its canonical JSON holds. Parts follow,
// until they hold those bytes.
type Sanctioning struct {
	Protocol int    `json:"protocol"`
	UID      int    `json:"uid"`
	Rite     string `json:"rite"`
	Sigil    string `json:"sigil"`
	Size     int    `json:"size"`
}

// Part is one line of the canonical JSON of the unit to be sanctioned: at
// most PartCap of its bytes, in order.
type Part struct {
	Part []byte `json:"part"` // base64 in the line
}

// PartCap is the most bytes of the unit one Part holds, so that a Part
// written in base64 stays within LineCap.
const PartCap = 512 << 10

// UnitCap is the most bytes the canonical JSON of a unit to be sanctioned
// may hold, so that its sanction is read whole by every reader of the
// store.
const UnitCap = sanction.UnitCap

// Abjuring is the first line the master's servitor speaks to an exalted
// servitor roused to abjure: the protocol it speaks, who it is, and the rite
// whose sanction to strike, or All of them.
type Abjuring struct {
	Protocol int    `json:"protocol"`
	UID      int    `json:"uid"`
	Rite     string `json:"rite"` // "" with All
	All      bool   `json:"all"`
}

// Offer speaks, as the master's servitor, the words that bid an exalted
// servitor roused to sanction lay the unit u for uid: the Sanctioning, then
// the canonical JSON of u in Parts.
func (c *Channel) Offer(uid int, u *sanction.Unit) error {
	data := u.Canonical()
	if err := c.Send(Sanctioning{Protocol: Protocol, UID: uid, Rite: u.Rite, Sigil: u.Sigil(),
		Size: len(data)}); err != nil {
		return err
	}
	for len(data) > 0 {
		n := min(len(data), PartCap)
		if err := c.Send(Part{Part: data[:n]}); err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}
