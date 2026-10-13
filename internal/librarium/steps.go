package librarium

import (
	"io/fs"
	"time"
)

// Kind is the kind of a step, named in its scripture by its own key.
type Kind int

const (
	KindSanctum Kind = iota + 1
	KindTranscription
	KindTether
	KindIncantation
	KindLitany
	kindVoxCast
)

// Kinds lists the kinds of what a subject, observance or verse holds, in
// the order of the codex: the matter of a sanctum, transcription or tether
// subject, and the speech of an incantation or litany. A sanction names
// each item by the key of its kind.
var Kinds = []Kind{KindSanctum, KindTranscription, KindTether, KindIncantation, KindLitany}

// markIKinds are the kinds of the steps of a liturgy of pattern Mark I,
// vox-casts among them, read only to be converted.
var markIKinds = append(Kinds[:len(Kinds):len(Kinds)], kindVoxCast)

var kindKeys = map[Kind]string{
	KindSanctum: "sanctum", KindTranscription: "transcription", KindTether: "tether",
	KindIncantation: "incantation", KindLitany: "litany", kindVoxCast: "vox-cast",
}

// Key is the step's own key, e.g. "tether".
func (k Kind) Key() string { return kindKeys[k] }

// Step is one step of a liturgy: *Sanctum, *Transcription, *Tether,
// *Incantation, *Litany or *voxCast.
type Step interface {
	Kind() Kind
}

// Sanctum keeps a warded region inside a vessel.
type Sanctum struct {
	Vessel       string
	Ward         string // "" means the rite's name
	Glyph        string // "" means inferred from the vessel's extension
	ClosingGlyph string // see Glyphs
	Consecrate   bool
	Scripture    AspectMap[Scripture]
	Exalted      bool // performed with the machine's own authority
}

// Transcription writes a whole vessel anew, or removes it.
type Transcription struct {
	Vessel    string
	Scripture AspectMap[Scripture] // Null removes the vessel
	Seal      *fs.FileMode         // nil when not written
	Zeal      bool                 // written as "zeal" or its alias "force"
	Exalted   bool                 // performed with the machine's own authority
}

// Anchor is where a tether leads; Null unbinds it.
type Anchor struct {
	Path string
	Null bool
}

// Tether binds a name to an anchor.
type Tether struct {
	Name    string
	Anchor  AspectMap[Anchor]
	Zeal    bool
	Exalted bool // performed with the machine's own authority
}

// Utterance holds what incantations and litanies share.
type Utterance struct {
	Tongue    string            // "" when the step names none
	Patience  time.Duration     // 0 when not written
	Reversion AspectMap[string] // zero when not written; "" reverts nothing
	Exalted   bool              // spoken with the machine's own authority
	// Relics are the scriptures the exalted speech reads beside its
	// scroll, as written: paths within the rite's hall. Nil when unwritten.
	Relics []string
}

// Incantation speaks one line of command in a tongue.
type Incantation struct {
	Command AspectMap[string]
	Utterance
}

// Litany recites a scroll of commands, offered exactly its offerings.
type Litany struct {
	Scroll    AspectMap[string]
	Offerings []AspectMap[string]
	Utterance
}

// Tidings a vox-cast proclaims.
const (
	VoxProgress = "progress"
	VoxSuccess  = "success"
)

// voxCast sends word of progress or triumph.
type voxCast struct {
	Tidings string // VoxProgress or VoxSuccess
}

func (*Sanctum) Kind() Kind       { return KindSanctum }
func (*Transcription) Kind() Kind { return KindTranscription }
func (*Tether) Kind() Kind        { return KindTether }
func (*Incantation) Kind() Kind   { return KindIncantation }
func (*Litany) Kind() Kind        { return KindLitany }
func (*voxCast) Kind() Kind       { return kindVoxCast }

// WardFor returns the sanctum's ward, defaulting to the rite's name.
func (s *Sanctum) WardFor(rite string) string {
	if s.Ward != "" {
		return s.Ward
	}
	return rite
}

// Glyphs returns the glyph and closing glyph of the sanctum's markers. When
// neither is written both are inferred from the vessel's extension; when
// only the closing glyph is written the glyph is inferred.
func (s *Sanctum) Glyphs() (glyph, closing string) {
	open, close := DefaultGlyphs(s.Vessel)
	switch {
	case s.Glyph == "" && s.ClosingGlyph == "":
		return open, close
	case s.Glyph == "":
		return open, s.ClosingGlyph
	}
	return s.Glyph, s.ClosingGlyph
}
