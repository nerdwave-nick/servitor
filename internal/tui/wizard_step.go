package tui

import (
	"errors"
	"fmt"
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
}

// essencePage writes the essential fields of the step: those holding one
// value for every aspect.
func (w *wizard) essencePage(m *model) tea.Cmd {
	sd := w.work
	f := newForm(m.t)
	k := sd.kind
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
	case librarium.KindVoxCast:
		f.addChoice("vox-cast", "Tidings", []string{librarium.VoxProgress, librarium.VoxSuccess}, sd.fixed["vox-cast"],
			"progress tells of the step before it; success proclaims triumph and belongs last.")
	}
	if hasFurther(k) {
		f.addToggle("further", "Further rites", sd.further, furtherHints[k])
	}
	return w.setForm(m, pageEssence, 0, f)
}

func (w *wizard) keepEssence() {
	f := w.form
	for _, fl := range f.fields {
		if fl.key == "further" {
			w.work.further = fl.on
		} else {
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

// varying names the fields of a kind that vary per aspect, and how they
// are called on its pages.
func (w *wizard) varying() [][2]string {
	switch w.work.kind {
	case librarium.KindSanctum, librarium.KindTranscription:
		return [][2]string{{"scripture", "scripture"}}
	case librarium.KindTether:
		return [][2]string{{"anchor", "anchor"}}
	case librarium.KindIncantation:
		if w.work.further {
			return [][2]string{{"command", "command"}, {"reversion", "reversion"}}
		}
		return [][2]string{{"command", "command"}}
	}
	if w.work.further {
		return [][2]string{{"command", "scroll"}, {"offerings", "offerings"}, {"reversion", "reversion"}}
	}
	return [][2]string{{"command", "scroll"}}
}

// aspectPage writes the fields of the step that vary per aspect, for the
// aspect i; each bears the choice to be the same for every aspect.
func (w *wizard) aspectPage(m *model, i int) tea.Cmd {
	sd := &w.work
	a := w.d.aspectList()[i]
	f := newForm(m.t)
	for _, v := range w.varying() {
		key, name := v[0], v[1]
		var same bool
		switch key {
		case "scripture":
			var sc librarium.Scripture
			sc, same = sd.scripture.value(a)
			text := sc.Text
			f.addArea("scripture", fmt.Sprintf("Scripture for aspect %q", a), text,
				"The lines kept in the vessel in this aspect; empty is allowed.")
			f.addText("tome", "Tome", sc.Tome, "", "Draw the scripture from this tome instead; a relative path "+
				"lies beside the rite's scripture.", nil)
			f.addToggle("illuminate", "Illuminate the tome", sc.Illuminate, "Fill the tome's placeholders.")
			if sd.kind == librarium.KindTranscription {
				f.addToggle("null", "Strike the vessel", sc.Null, "In this aspect the vessel shall not be.")
			}
		case "anchor":
			var an librarium.Anchor
			an, same = sd.anchor.value(a)
			f.addText("anchor", fmt.Sprintf("Anchor for aspect %q", a), an.Path, "~/.config/nfluff/themes/{{aspect}}",
				"Where the tether leads in this aspect.", nil)
			f.addToggle("unbind", "Unbind the tether", an.Null, "In this aspect the tether shall lead nowhere.")
		case "offerings":
			var list []string
			list, same = sd.offerings.value(a)
			f.addArea("offerings", fmt.Sprintf("Offerings for aspect %q, one per line", a), strings.Join(list, "\n"),
				"Given to the scroll exactly as written, each illuminated.")
		default:
			var text string
			text, same = w.textOf(key).value(a)
			label := strings.ToUpper(name[:1]) + name[1:]
			var check func(string) error
			if key == "command" {
				check = required("the " + name + " must be spoken")
			}
			f.addText(key, fmt.Sprintf("%s for aspect %q", label, a), text, "", "Placeholders such as "+
				"{{aspect}} are illuminated.", check)
		}
		f.addToggle("same:"+key, "Same "+name+" for every aspect", same,
			`Written under "*": every aspect without a value of its own shares it.`)
	}
	return w.setForm(m, pageAspect, i, f)
}

// textOf is the varied text the key names.
func (w *wizard) textOf(key string) *varied[string] {
	if key == "reversion" {
		return &w.work.reversion
	}
	return &w.work.command
}

// keepAspect keeps the page of the aspect w.index and reports whether the
// step is complete: the last aspect is written, or every value of the
// page is the same for every aspect and no later aspect speaks its own.
func (w *wizard) keepAspect() bool {
	f, sd := w.form, &w.work
	aspects := w.d.aspectList()
	a := aspects[w.index]
	done := true
	for _, v := range w.varying() {
		key := v[0]
		same := f.get("same:"+key) == "true"
		switch key {
		case "scripture":
			sc := librarium.Inline(f.get("scripture"))
			if tome := strings.TrimSpace(f.get("tome")); tome != "" {
				sc = librarium.Tome(tome, f.get("illuminate") == "true")
			}
			if f.get("null") == "true" {
				sc = librarium.Scripture{Null: true}
			}
			sd.scripture.set(a, sc, same)
			done = done && same && !sd.scripture.ownAfter(aspects, w.index)
		case "anchor":
			an := librarium.Anchor{Path: strings.TrimSpace(f.get("anchor")), Null: f.get("unbind") == "true"}
			if an.Null {
				an.Path = ""
			}
			sd.anchor.set(a, an, same)
			done = done && same && !sd.anchor.ownAfter(aspects, w.index)
		case "offerings":
			sd.offerings.set(a, lines(f.get("offerings")), same)
			done = done && same && !sd.offerings.ownAfter(aspects, w.index)
		default:
			t := w.textOf(key)
			t.set(a, strings.TrimSpace(f.get(key)), same)
			done = done && same && !t.ownAfter(aspects, w.index)
		}
	}
	return done || w.index == len(aspects)-1
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
	if w.widx < 0 {
		at := 0
		if len(w.d.steps) > 0 {
			at = w.cursor + 1
		}
		w.d.steps = append(w.d.steps[:at], append([]stepDraft{w.work}, w.d.steps[at:]...)...)
		w.cursor = at
	} else {
		w.d.steps[w.widx] = w.work
	}
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
