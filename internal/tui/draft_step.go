package tui

import (
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// stepDraft is the editable form of one step of the liturgy. The values
// that hold one value for every aspect are kept as typed in fixed, under
// their keys in the codex ("sanctum", "ward", "seal", "patience", …; a
// true flag is "true"); the values that may vary per aspect are varied.
type stepDraft struct {
	kind      librarium.Kind
	fixed     map[string]string
	further   bool // the further rites are unveiled
	scripture varied[librarium.Scripture]
	anchor    varied[librarium.Anchor]
	command   varied[string] // an incantation's command or a litany's scroll
	reversion varied[string]
	offerings varied[[]string]
}

// newStepDraft is a new step of kind k. Scripture differs per aspect
// unless told otherwise; commands, anchors and offerings, which name
// {{aspect}} where they differ, are the same for every aspect.
func newStepDraft(k librarium.Kind) stepDraft {
	sd := stepDraft{kind: k, fixed: map[string]string{}, scripture: newVaried[librarium.Scripture](true),
		anchor: newVaried[librarium.Anchor](false), command: newVaried[string](false),
		reversion: newVaried[string](false), offerings: newVaried[[]string](false)}
	if k == librarium.KindVoxCast {
		sd.fixed[k.Key()] = librarium.VoxProgress
	}
	return sd
}

// furtherKeys are the further rites of each kind that hold one value for
// every aspect; reversion and offerings vary per aspect.
var furtherKeys = map[librarium.Kind][]string{
	librarium.KindTranscription: {"seal", "zeal"},
	librarium.KindTether:        {"zeal"},
	librarium.KindIncantation:   {"tongue", "patience"},
	librarium.KindLitany:        {"tongue", "patience"},
}

// hasFurther reports whether a step of kind k bears further rites.
func hasFurther(k librarium.Kind) bool { return len(furtherKeys[k]) > 0 }

// stepFrom is the step s as the wizard edits it.
func stepFrom(s librarium.Step) stepDraft {
	sd := newStepDraft(s.Kind())
	sd.scripture.apart = false
	f := sd.fixed
	flag := func(key string, b bool) {
		if b {
			f[key] = "true"
		}
	}
	switch s := s.(type) {
	case *librarium.Sanctum:
		f["sanctum"], f["ward"], f["glyph"], f["closing-glyph"] = s.Vessel, s.Ward, s.Glyph, s.ClosingGlyph
		flag("consecrate", s.Consecrate)
		sd.scripture = variedFrom(s.Scripture)
	case *librarium.Transcription:
		f["transcription"] = s.Vessel
		if s.Seal != nil {
			f["seal"] = sealText(*s.Seal)
		}
		flag("zeal", s.Zeal)
		sd.scripture = variedFrom(s.Scripture)
	case *librarium.Tether:
		f["tether"] = s.Name
		flag("zeal", s.Zeal)
		sd.anchor = variedFrom(s.Anchor)
	case *librarium.Incantation:
		sd.command = variedFrom(s.Command)
		sd.utteranceFrom(s.Utterance)
	case *librarium.Litany:
		sd.command = variedFrom(s.Scroll)
		sd.utteranceFrom(s.Utterance)
		if s.Offerings != nil {
			sd.offerings = offeringsFrom(s.Offerings)
			sd.further = true
		}
	case *librarium.VoxCast:
		f["vox-cast"] = s.Tidings
	}
	for _, k := range furtherKeys[sd.kind] {
		sd.further = sd.further || f[k] != ""
	}
	return sd
}

func (sd *stepDraft) utteranceFrom(u librarium.Utterance) {
	sd.fixed["tongue"] = u.Tongue
	if u.Patience != 0 {
		sd.fixed["patience"] = u.Patience.String()
	}
	sd.reversion = variedFrom(u.Reversion)
	sd.further = !u.Reversion.IsZero()
}

// toStep writes the step for a rite of aspects.
func (sd stepDraft) toStep(aspects []string) librarium.Step {
	f := sd.fixed
	switch sd.kind {
	case librarium.KindSanctum:
		return &librarium.Sanctum{Vessel: f["sanctum"], Ward: f["ward"], Glyph: f["glyph"],
			ClosingGlyph: f["closing-glyph"], Consecrate: f["consecrate"] == "true",
			Scripture: sd.scripture.toMap(aspects)}
	case librarium.KindTranscription:
		return &librarium.Transcription{Vessel: f["transcription"], Scripture: sd.scripture.toMap(aspects),
			Seal: parseSeal(f["seal"]), Zeal: f["zeal"] == "true"}
	case librarium.KindTether:
		return &librarium.Tether{Name: f["tether"], Anchor: sd.anchor.toMap(aspects), Zeal: f["zeal"] == "true"}
	case librarium.KindIncantation:
		return &librarium.Incantation{Command: sd.command.toMap(aspects), Utterance: sd.utterance(aspects)}
	case librarium.KindLitany:
		return &librarium.Litany{Scroll: sd.command.toMap(aspects), Offerings: offeringsTo(sd.offerings, aspects),
			Utterance: sd.utterance(aspects)}
	}
	return &librarium.VoxCast{Tidings: f["vox-cast"]}
}

// utterance is what an incantation or litany speaks with; a reversion
// that reverts nothing for every aspect stays unwritten.
func (sd stepDraft) utterance(aspects []string) librarium.Utterance {
	u := librarium.Utterance{Tongue: sd.fixed["tongue"], Reversion: sd.reversion.toMap(aspects)}
	u.Patience, _ = time.ParseDuration(sd.fixed["patience"])
	if r := u.Reversion; r.IsUniform() && r.Entries()[0].Value == "" {
		u.Reversion = librarium.AspectMap[string]{}
	}
	return u
}

func sealText(m fs.FileMode) string { return fmt.Sprintf("%04o", uint32(m)) }

// parseSeal reads a seal of three or four octal numerals; nil when unwritten.
func parseSeal(s string) *fs.FileMode {
	n, err := strconv.ParseUint(strings.TrimSpace(s), 8, 32)
	if err != nil {
		return nil
	}
	m := fs.FileMode(n)
	return &m
}

// offeringsFrom gathers the offerings of a litany per aspect: an aspect
// that any offering names receives a list of its own.
func offeringsFrom(offs []librarium.AspectMap[string]) varied[[]string] {
	v := newVaried[[]string](false)
	shared := make([]string, len(offs))
	for i, o := range offs {
		for _, e := range o.Entries() {
			if e.Aspect == librarium.Fallback {
				shared[i] = e.Value
			} else if v.own[e.Aspect] == nil {
				v.own[e.Aspect] = make([]string, len(offs))
			}
		}
	}
	for a, list := range v.own {
		for i, o := range offs {
			list[i], _ = o.For(a)
		}
	}
	v.shared, v.hasShared = shared, true
	return v
}

// offeringsTo writes the offerings per aspect as one aspect map per
// offering, written once when every aspect offers the same.
func offeringsTo(v varied[[]string], aspects []string) []librarium.AspectMap[string] {
	n := 0
	for _, a := range aspects {
		list, _ := v.value(a)
		n = max(n, len(list))
	}
	if n == 0 {
		return nil
	}
	offs := make([]librarium.AspectMap[string], n)
	for i := range offs {
		offs[i] = offering(v, aspects, i)
	}
	return offs
}

// offering writes the offering i of every aspect as one aspect map.
func offering(v varied[[]string], aspects []string, i int) librarium.AspectMap[string] {
	at := func(list []string) (string, bool) {
		if i < len(list) {
			return list[i], true
		}
		return "", false
	}
	shared, hasShared := at(v.shared)
	type spoken struct {
		aspect, x  string
		ok, shares bool
	}
	all := make([]spoken, len(aspects))
	alike, fallback := true, false
	for j, a := range aspects {
		list, same := v.value(a)
		x, ok := at(list)
		all[j] = spoken{a, x, ok, same}
		alike = alike && ok && x == all[0].x
		fallback = fallback || same || !ok
	}
	if alike {
		return librarium.Uniform(all[0].x)
	}
	var entries []librarium.Entry[string]
	for _, s := range all {
		// an aspect offering what "*" offers needs no entry of its own
		echoes := fallback && hasShared && s.x == shared
		if !s.shares && s.ok && !echoes {
			entries = append(entries, librarium.Entry[string]{Aspect: s.aspect, Value: s.x})
		}
	}
	if fallback && hasShared {
		if len(entries) == 0 {
			return librarium.Uniform(shared)
		}
		entries = append(entries, librarium.Entry[string]{Aspect: librarium.Fallback, Value: shared})
	}
	return librarium.PerAspect(entries...)
}
