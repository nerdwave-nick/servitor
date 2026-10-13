package librarium

import (
	"slices"
	"time"
)

// Pattern is the pattern of every rite this package yields and writes:
// rites of aspects that declare their subjects and observances, and edicts
// written in the designation form.
const Pattern = "Mark II"

// patterns lists every pattern this package reads: PatternMarkI to convert
// it, and Pattern.
var patterns = []string{PatternMarkI, Pattern}

// Designations of the subjects and observances of pattern Mark II, and of
// an edict's entries: the key "designation" tells which shape each takes.
const (
	DesignationSanctum       = "sanctum"
	DesignationTranscription = "transcription"
	DesignationTether        = "tether"
	DesignationMachineSpirit = "machine-spirit"
	DesignationIncantation   = "incantation"
	DesignationLitany        = "litany"
)

var (
	// SubjectDesignations are the designations of a subject, in the order of
	// the codex.
	SubjectDesignations = []string{DesignationSanctum, DesignationTranscription, DesignationTether, DesignationMachineSpirit}
	// SpeechDesignations are the designations of an observance and of an
	// edict's entry.
	SpeechDesignations = []string{DesignationIncantation, DesignationLitany}
)

// Keys of the objects of a Mark II scripture, in the order of the codex.
var (
	RiteKeys = []string{"pattern", "purpose", "aspects", "edict", "inscriptions", "tongue", "vox-cast",
		"subjects", "observances", "liturgy", SchemaKey}
	// SubjectKeys are the keys every subject may hold, whatever its
	// designation; DesignationKeys tells the keys of each designation.
	SubjectKeys = []string{"name", "designation", "purpose", "omens", "exalted", "introit", "recessional"}
	// SpiritAuspexKeys are the keys of a machine-spirit's auspex.
	SpiritAuspexKeys = []string{"scan", "trials", "patience", "exalted"}
	// ObservanceKeys are the keys every observance holds beside the keys of
	// its speech (SpeechKeys); an edict's entry holds only "designation"
	// beside them.
	ObservanceKeys = []string{"name", "designation", "to", "from"}
)

var designationKeys = map[string][]string{
	DesignationSanctum:       {"vessel", "ward", "glyph", "closing-glyph", "consecrate"},
	DesignationTranscription: {"vessel", "seal", "zeal", "force"},
	DesignationTether:        {"tether", "zeal", "force"},
	DesignationMachineSpirit: {"auspex", "enact", "tongue", "patience", "relics"},
	DesignationIncantation:   {"words", "tongue", "patience", "reversion", "exalted", "relics"},
	DesignationLitany:        {"scroll", "offerings", "tongue", "patience", "reversion", "exalted", "relics"},
}

// DesignationKeys returns the keys a subject of designation d holds beside
// SubjectKeys, or the keys of the speech of an observance or edict entry of
// designation d.
func DesignationKeys(d string) []string { return slices.Clone(designationKeys[d]) }

// Matter is what a subject holds by its designation: *Sanctum,
// *Transcription, *Tether or *MachineSpirit. A sanctum's or transcription's
// Scripture and a tether's Anchor are its omens: each aspect's mark upon it.
type Matter interface {
	Designation() string
}

func (*Sanctum) Designation() string       { return DesignationSanctum }
func (*Transcription) Designation() string { return DesignationTranscription }
func (*Tether) Designation() string        { return DesignationTether }
func (*MachineSpirit) Designation() string { return DesignationMachineSpirit }
func (*Incantation) Designation() string   { return DesignationIncantation }
func (*Litany) Designation() string        { return DesignationLitany }

// Subject is one thing a rite of pattern Mark II reads and sets: its omen
// for each aspect, and how it is read and set, told by its designation.
type Subject struct {
	Name        string
	Purpose     string
	Introit     []string // observances spoken before it changes; nil when unwritten
	Recessional []string // observances spoken after it changed; nil when unwritten
	Matter      Matter
}

// Designation is the subject's designation.
func (s Subject) Designation() string { return s.Matter.Designation() }

// Exalted reports whether the subject is set with the machine's own
// authority.
func (s Subject) Exalted() bool {
	switch m := s.Matter.(type) {
	case *Sanctum:
		return m.Exalted
	case *Transcription:
		return m.Exalted
	case *Tether:
		return m.Exalted
	case *MachineSpirit:
		return m.Exalted
	}
	return false
}

// MachineSpirit is a subject read by its auspex and set by its enact words.
type MachineSpirit struct {
	// Omens are the omen of each aspect; zero when unwritten, and then each
	// aspect's own name is its omen.
	Omens    AspectMap[string]
	Auspex   *SpiritAuspex     // nil when it cannot be read
	Enact    AspectMap[string] // the words that set it, keyed by omen
	Tongue   string            // "" when it names none
	Patience time.Duration     // of its enact words; 0 when not written
	Relics   []string          // as written; nil when unwritten
	Exalted  bool
}

// SpiritAuspex reads a machine-spirit: its scan prints the omen it shows,
// and each of its trials tells by its death-mark whether it shows one omen.
type SpiritAuspex struct {
	Scan     string            // "" when unwritten
	Trials   AspectMap[string] // keyed by omen; zero when unwritten
	Patience time.Duration     // 0 when not written
	// Exalted marks a privileged auspex, spoken by the exalted servitor.
	Exalted bool
}

// Wait returns the auspex's patience, or DefaultAuspexPatience.
func (a SpiritAuspex) Wait() time.Duration {
	if a.Patience > 0 {
		return a.Patience
	}
	return DefaultAuspexPatience
}

// Observance is one named deed of a rite of pattern Mark II, spoken when a
// subject that names it changes.
type Observance struct {
	Name string
	// Speech is an *Incantation or a *Litany, every value of it written
	// once for every aspect.
	Speech Step
	To     []string // the aspects it is spoken into; nil when unwritten: every one
	From   []string // the aspects it is spoken out of; nil when unwritten: every one
}

// Designation is the observance's designation.
func (o Observance) Designation() string { return o.Speech.(Matter).Designation() }

// Exalted reports whether the observance is spoken with the machine's own
// authority.
func (o Observance) Exalted() bool { return IsExalted(o.Speech) }

// Subject returns the subject named name.
func (r *Rite) Subject(name string) (Subject, bool) {
	for _, s := range r.Subjects {
		if s.Name == name {
			return s, true
		}
	}
	return Subject{}, false
}

// Observance returns the observance named name.
func (r *Rite) Observance(name string) (Observance, bool) {
	for _, o := range r.Observances {
		if o.Name == name {
			return o, true
		}
	}
	return Observance{}, false
}
