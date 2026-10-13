package tui

import (
	"errors"
	"regexp"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// furtherHints name the further rites each kind unveils.
var furtherHints = map[librarium.Kind]string{
	librarium.KindTranscription: "Unveil the seal and zeal of the transcription; rarely needed.",
	librarium.KindTether:        "Unveil the zeal of the tether; rarely needed.",
	librarium.KindIncantation:   "Unveil tongue, patience and reversion; rarely needed.",
	librarium.KindLitany:        "Unveil offerings, tongue, patience and reversion.",
	kindMachineSpirit:           "Unveil the tongue and patience of its enact words; rarely needed.",
}

// essencePage writes the essential fields of the step, subject or
// observance: those holding one value for every aspect.
func (w *wizard) essencePage(m *model) tea.Cmd {
	sd := w.work
	f := newForm(m.t)
	k := sd.kind
	switch {
	case w.station == stationObservances:
		w.observanceFields(f)
	case w.d.bearsSubjects():
		w.subjectFields(f)
	}
	switch k {
	case librarium.KindSanctum:
		glyph, closing := librarium.DefaultGlyphs(sd.fixed["sanctum"])
		f.addText("sanctum", "Vessel", sd.fixed["sanctum"], "~/.config/niri/util.kdl",
			"The vessel whose warded region the sanctum keeps. ~ and $VARS are expanded; a relative path "+
				"lies beside the rite's scripture.", required("a sanctum must name its vessel"))
		f.addText("ward", "Ward", sd.fixed["ward"], w.d.name,
			"Names the sanctum within its vessel; unspoken, it is the rite's name.", validWord)
		f.addText("glyph", "Glyph", sd.fixed["glyph"], glyph,
			"Opens the sanctum's markers; unspoken, it is divined from the vessel's extension.", nil)
		f.addText("closing-glyph", "Closing glyph", sd.fixed["closing-glyph"], closing,
			"Closes the markers, such as */ or -->; unspoken, it is divined with the glyph.", nil)
		f.addToggle("consecrate", "Consecrate the vessel when it is absent", sd.fixed["consecrate"] == "true",
			"space turns it")
	case librarium.KindTranscription:
		f.addText("transcription", "Vessel", sd.fixed["transcription"], "~/.config/nfluff/whole.conf",
			"The vessel transcribed whole in every aspect, or struck where its scripture is null.",
			required("a transcription must name its vessel"))
	case librarium.KindTether:
		f.addText("tether", "Bound name", sd.fixed["tether"], "~/.local/share/nfluff/current-theme",
			"The name the tether binds; each aspect binds it to its own anchor.",
			required("a tether must name what it binds"))
	}
	addExaltation(f, sd, w.exaltedNoun())
	if hasFurther(k) {
		f.addToggle("further", "Further rites", sd.further, furtherHints[k])
	}
	return w.setForm(m, pageEssence, 0, f)
}

func (w *wizard) keepEssence() {
	f := w.form
	for _, fl := range f.fields {
		switch fl.key {
		case "further":
			w.work.further = fl.on
		case "name":
			if w.station == stationObservances {
				w.observance.name = strings.TrimSpace(fl.value())
			} else {
				w.subject.name = strings.TrimSpace(fl.value())
			}
		case "purpose":
			w.subject.purpose = strings.TrimSpace(fl.value())
		case "relics":
			w.work.relics = keptRelics(fl.value(), w.work.relics)
		default:
			w.work.fixed[fl.key] = strings.TrimSpace(fl.value())
		}
	}
}

// furtherPage writes the further rites holding one value for every aspect.
func (w *wizard) furtherPage(m *model) tea.Cmd {
	fx := w.work.fixed
	f := newForm(m.t)
	for _, k := range furtherKeys[w.work.kind] {
		switch k {
		case "seal":
			f.addText("seal", "Seal", fx["seal"], "0644",
				"Three or four octal numerals; unspoken, the vessel keeps its seal, and a new one is sealed 0644.", validSeal)
		case "zeal":
			f.addToggle("zeal", "Zeal", fx["zeal"] == "true",
				"Cast down what stands in the way though the rite did not place it there.")
		case "tongue":
			f.addText("tongue", "Tongue", fx["tongue"], "bash",
				"The program that speaks the command; unspoken, the rite's tongue, the settings' or bash.", validWord)
		case "patience":
			f.addText("patience", "Patience", fx["patience"], "30s",
				"How long the step may labour before it is struck down, such as 5s or 2m.", validPatience)
		}
	}
	return w.setForm(m, pageFurther, 0, f)
}

// varying names the words of an incantation or litany, spoken once, and
// how they are called on its page.
func (w *wizard) varying() [][2]string {
	switch w.work.kind {
	case librarium.KindIncantation:
		command := "command"
		if w.station == stationObservances {
			command = "words" // as the observance writes it
		}
		if w.work.further {
			return [][2]string{{"command", command}, {"reversion", "reversion"}}
		}
		return [][2]string{{"command", command}}
	}
	if w.work.further {
		return [][2]string{{"command", "scroll"}, {"offerings", "offerings"}, {"reversion", "reversion"}}
	}
	return [][2]string{{"command", "scroll"}}
}

// textOf is the varied text the key names.
func (w *wizard) textOf(key string) *varied[string] {
	if key == "reversion" {
		return &w.work.reversion
	}
	return &w.work.command
}

// lines are the lines of text, trailing empty ones dropped.
func lines(text string) []string {
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return []string{}
	}
	return strings.Split(text, "\n")
}

// finishStep places the step written into the liturgy.
func (w *wizard) finishStep() tea.Cmd {
	w.d.steps = place(w.d.steps, w.widx, &w.cursor, w.work)
	w.form = nil
	return nil
}

func required(lament string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New(lament)
		}
		return nil
	}
}

func validPatience(s string) error {
	if s = strings.TrimSpace(s); s == "" {
		return nil
	}
	if p, err := time.ParseDuration(s); err != nil || p <= 0 {
		return errors.New("patience is a span greater than nothing, such as 500ms, 5s or 2m")
	}
	return nil
}

func validWord(s string) error {
	if strings.ContainsAny(strings.TrimSpace(s), " \t") {
		return errors.New("a single word, without whitespace")
	}
	return nil
}

var sealRe = regexp.MustCompile(`^[0-7]{3,4}$`)

func validSeal(s string) error {
	if s = strings.TrimSpace(s); s != "" && !sealRe.MatchString(s) {
		return errors.New("a seal is three or four octal numerals, such as 0644")
	}
	return nil
}
