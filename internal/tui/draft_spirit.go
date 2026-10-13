package tui

import (
	"slices"
	"strings"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// spiritDraft is what a machine-spirit holds beside its step draft: its
// omens per aspect, unless each aspect is its own omen (named); its auspex,
// whose trials are keyed by omen and an empty trial is none; and its enact
// words keyed by omen.
type spiritDraft struct {
	named      bool // no omens are written: each aspect is its own omen
	omens      varied[string]
	scan       string
	trials     varied[string]
	patience   string // of the auspex
	privileged bool   // the auspex is exalted, while the subject is
	enact      varied[string]
}

// newSpiritDraft is the spirit of a new machine-spirit: each aspect its own
// omen, never read, and set by the same words for every omen.
func newSpiritDraft() spiritDraft {
	return spiritDraft{named: true, omens: newVaried[string](true), trials: newVaried[string](true),
		enact: newVaried[string](false)}
}

// spiritFrom is the machine-spirit ms as the wizard edits it: its step
// draft and its spirit.
func spiritFrom(ms *librarium.MachineSpirit) (stepDraft, spiritDraft) {
	sd := newStepDraft(kindMachineSpirit)
	sd.fixed["tongue"] = ms.Tongue
	if ms.Patience != 0 {
		sd.fixed["patience"] = ms.Patience.String()
	}
	if ms.Exalted {
		sd.fixed["exalted"] = "true"
	}
	sd.relics = ms.Relics
	sd.further = ms.Tongue != "" || ms.Patience != 0
	sp := spiritDraft{named: ms.Omens.IsZero(), omens: variedFrom(ms.Omens), trials: newVaried[string](true),
		enact: variedFrom(ms.Enact)}
	sp.omens.apart = true
	if a := ms.Auspex; a != nil {
		sp.scan, sp.privileged = a.Scan, a.Exalted
		sp.trials = variedFrom(a.Trials)
		sp.trials.apart = true
		if a.Patience != 0 {
			sp.patience = a.Patience.String()
		}
	}
	return sd, sp
}

// toSpirit writes the machine-spirit of the step sd over aspects, its
// trials and enact words keyed by its declared omens. Relics and an exalted
// auspex are written only while the subject is exalted; an auspex without
// scan, trials or patience is none.
func (sp spiritDraft) toSpirit(sd stepDraft, aspects, omens []string) *librarium.MachineSpirit {
	ms := &librarium.MachineSpirit{Tongue: strings.TrimSpace(sd.fixed["tongue"]), Exalted: sd.fixed["exalted"] == "true",
		Enact: omensObject(sp.enact.toMap(omens))}
	ms.Patience, _ = time.ParseDuration(strings.TrimSpace(sd.fixed["patience"]))
	if ms.Exalted {
		ms.Relics = sd.relics
	}
	if !sp.named {
		ms.Omens = omensObject(sp.omens.toMap(aspects))
	}
	a := &librarium.SpiritAuspex{Scan: strings.TrimSpace(sp.scan), Trials: trialsMap(sp.trials, omens),
		Exalted: sp.privileged && ms.Exalted}
	a.Patience, _ = time.ParseDuration(strings.TrimSpace(sp.patience))
	if a.Scan != "" || !a.Trials.IsZero() || a.Patience != 0 || a.Exalted {
		ms.Auspex = a
	}
	return ms
}

// trialsMap writes the trials over omens: each omen's own, unless empty,
// then "*" for the omens that share it; unwritten when none is spoken.
func trialsMap(v varied[string], omens []string) librarium.AspectMap[string] {
	var entries []librarium.Entry[string]
	sharing := false
	for _, o := range omens {
		x, same := v.value(o)
		switch {
		case same:
			sharing = true
		case x != "":
			entries = append(entries, librarium.Entry[string]{Aspect: o, Value: x})
		}
	}
	if sharing && v.hasShared && v.shared != "" {
		entries = append(entries, librarium.Entry[string]{Aspect: librarium.Fallback, Value: v.shared})
	}
	if len(entries) == 0 {
		return librarium.AspectMap[string]{}
	}
	return librarium.PerAspect(entries...)
}

// declaredOmens are the omens the machine-spirit sp declares within the
// draft: its omen of each aspect, illuminated for that aspect and trimmed,
// once each in the order of the aspects; or the aspects themselves when
// each is its own omen. An omen that cannot be illuminated is told as
// written.
func (d riteDraft) declaredOmens(sp spiritDraft) []string {
	aspects := d.aspectList()
	if sp.named {
		return aspects
	}
	scope := placeholder.Scope{Mode: placeholder.RiteMode, Inscriptions: d.keys()}
	var out []string
	for _, a := range aspects {
		words, _ := sp.omens.value(a)
		values := placeholder.Values{Aspect: a, Rite: d.name, Inscriptions: map[string]string{}}
		for _, in := range d.inscriptions {
			values.Inscriptions[in.key] = in.decrees[a]
			if values.Inscriptions[in.key] == "" {
				values.Inscriptions[in.key] = in.decrees[librarium.Fallback]
			}
		}
		if lit, err := scope.Render(words, values); err == nil {
			words = lit
		}
		if o := strings.TrimSpace(words); !slices.Contains(out, o) {
			out = append(out, o)
		}
	}
	return out
}
