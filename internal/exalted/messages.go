package exalted

// Handshake is the first line the invoker's servitor speaks: the protocol it
// speaks, who it is, what it would have performed, and the words given at
// invocation that exalted speech hears from its environment alone.
type Handshake struct {
	Protocol int    `json:"protocol"`
	UID      int    `json:"uid"`  // the master's; the channel's credentials must agree
	Rite     string `json:"rite"` // as argv names it
	Aspect   string `json:"aspect"`
	// Sigil is the whole sigil of the plan for the aspect (of the unit, for
	// an edict); argv names its first twelve letters.
	Sigil string `json:"sigil"`
	// Former is the former aspect, heard as SERVITOR_FORMER_ASPECT.
	Former string `json:"former"`
	// Inscriptions are the words of the inscriptions given at invocation,
	// by key, heard as SERVITOR_INSCRIPTION_<KEY>.
	Inscriptions map[string]string `json:"inscriptions"`
}

// Request is every later line the invoker's servitor speaks: exactly one of
// its members.
type Request struct {
	// Read names subjects of a state rite of pattern Mark II the invoker
	// cannot read itself — those whose auspex is privileged, and the
	// sanctums, transcriptions and tethers hidden from it — to be read by
	// the exalted servitor, once, before any item is performed.
	Read []string `json:"read,omitzero"`
	// Perform performs the exalted item named: of a liturgy, the verse of
	// that number, which must be the next of the plan; of a state rite of
	// pattern Mark II, the subject or observance, which must lie after
	// every item performed (see order.go).
	Perform *Item `json:"perform,omitzero"`
	// Revert reverts the exalted item named, which must be the last
	// performed and not yet reverted.
	Revert *Item `json:"revert,omitzero"`
	// Finish ends the session, keeping what was performed.
	Finish *bool `json:"finish,omitzero"`
	// Abort reverts every verse performed, in reverse, and ends the
	// session.
	Abort *bool `json:"abort,omitzero"`
}

// Verdict is the verdict the exalted servitor speaks upon a line.
type Verdict string

// The verdicts of the exalted servitor.
const (
	// Accepted: the session is accepted; Verses names the plan.
	Accepted Verdict = "accepted"
	// Refused: the line was refused, and the session ends.
	Refused Verdict = "refused"
	// Triumph: the verse was performed.
	Triumph Verdict = "triumph"
	// Fell: the verse fell; it stands performed only when Changed.
	Fell Verdict = "fell"
	// Reverted: the verse, or every verse of an abort, was reverted.
	Reverted Verdict = "reverted"
	// Faltered: a reversion fell.
	Faltered Verdict = "faltered"
	// Beheld: the subjects a read named were read; Sights tell what each
	// shows.
	Beheld Verdict = "beheld"
	// Finished: the session ends, and what was performed is kept.
	Finished Verdict = "finished"
	// Sanctioned: the sanction was laid in the store.
	Sanctioned Verdict = "sanctioned"
	// Abjured: the sanctions named in Abjured were struck from the store.
	Abjured Verdict = "abjured"
)

// Reply is the exalted servitor's answer to one line. It never carries the
// scripture of a vessel: of a vessel it tells only sizes and SHA-256 sums.
type Reply struct {
	// To names the line answered: handshake, perform, revert, finish or
	// abort; "" for a line that was no true word of the protocol.
	To      string  `json:"to"`
	Verdict Verdict `json:"verdict"`
	Verse   int     `json:"verse,omitzero"`
	// Subject and Observance name the item of a state rite answered.
	Subject    string `json:"subject,omitzero"`
	Observance string `json:"observance,omitzero"`
	Protocol   int    `json:"protocol,omitzero"` // in every answer to the first line
	// Verses are the exalted verses of the plan, in written order, upon
	// acceptance; Subjects and Observances those of the plan of a state
	// rite.
	Verses      []int    `json:"verses,omitzero"`
	Subjects    []string `json:"subjects,omitzero"`
	Observances []string `json:"observances,omitzero"`
	// Changed tells that the verse changed the machine; a fallen verse that
	// changed it stands performed and is reverted like any other.
	Changed bool  `json:"changed,omitzero"`
	Before  *Seen `json:"before,omitzero"` // the vessel before the verse; nil when absent
	After   *Seen `json:"after,omitzero"`  // the vessel after it; nil when absent
	// Heard are the words exalted speech uttered.
	Heard string `json:"heard,omitzero"`
	// Sights tell what each subject a read named shows, in the order named.
	Sights []Sight `json:"sights,omitzero"`
	// Omens are the declared omens a privileged subject was read to show
	// once it was set or restored, which proved it: never the words its
	// auspex printed.
	Omens []string `json:"omens,omitzero"`
	// Lament tells why the line was refused or the verse fell or faltered.
	Lament string `json:"lament,omitzero"`
	// Reversions are the reversions of an abort, from the last verse
	// performed back to the first.
	Reversions []Reply `json:"reversions,omitzero"`
	// Abjured names the rites whose sanctions an abjuring struck, in order.
	Abjured []string `json:"abjured,omitzero"`
}

// Seen is what the exalted servitor tells of a vessel: its size and the
// hexadecimal SHA-256 of its scripture, never the scripture itself.
type Seen struct {
	Size   int    `json:"size"`
	SHA256 string `json:"sha256"`
}

// Sight is what the exalted servitor tells of one subject a read named,
// never the words its auspex printed nor the scripture of its vessel: of a
// machine-spirit, the declared omens it fits (none when it is mute); of a
// sanctum, transcription or tether, whether it already stands as the plan
// demands.
type Sight struct {
	Subject   string   `json:"subject"`
	Omens     []string `json:"omens,omitzero"`
	Unchanged bool     `json:"unchanged,omitzero"`
}
