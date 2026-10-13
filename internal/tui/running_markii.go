package tui

import (
	"fmt"
	"strconv"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
)

// What an invocation of a rite of aspects tells its running
// panel beyond the steps begun.
type (
	// commencedMsg tells, once the pre-flight has passed, what the
	// invocation plans for every subject, by name.
	commencedMsg struct {
		plans map[string]invocation.SubjectPlan
	}
	// spokenMsg tells that an observance was spoken.
	spokenMsg struct{ verse invocation.Verse }
)

// Performed tells the running panel of every observance spoken; the steps
// performed it watches as they begin.
func (h herald) Performed(v invocation.Verse, _, _ int) {
	if v.Observance != "" {
		h.send(spokenMsg{verse: v})
	}
}

// commence tells the running panel of a rite of aspects the plan of every
// subject once the pre-flight has passed; nil for every other rite.
func commence(r *librarium.Rite, send func(tea.Msg)) func(rituals.Result) {
	if r.Edict {
		return nil
	}
	return func(res rituals.Result) {
		plans := map[string]invocation.SubjectPlan{}
		for _, f := range res.Foresight {
			if f.Plan != nil {
				plans[f.Subject] = *f.Plan
			}
		}
		send(commencedMsg{plans: plans})
	}
}

// changing counts the subjects plans would set.
func changing(plans map[string]invocation.SubjectPlan) int {
	n := 0
	for _, p := range plans {
		if p.Changing {
			n++
		}
	}
	return n
}

// named names a verse of the rite r as verseName does, a subject with its
// designation ("subject smt · machine-spirit") and an observance with its
// own ("observance veil · incantation"), each marked ✠ when exalted.
func named(r *librarium.Rite, v invocation.Verse) string {
	switch {
	case r == nil:
	case v.Subject != "":
		if sub, ok := r.Subject(v.Subject); ok {
			return verseName(v) + " · " + sub.Designation() + exaltedMark(v)
		}
	case v.Observance != "":
		if o, ok := r.Observance(v.Observance); ok {
			return verseName(v) + " · " + o.Designation() + exaltedMark(v)
		}
	}
	return verseName(v)
}

// exaltedMark is " ✠" for an exalted subject or observance, as the
// foresight of the command line marks it; "" otherwise.
func exaltedMark(v invocation.Verse) string {
	if v.Exalted {
		return " ✠"
	}
	return ""
}

// spirit reports whether the subject a verse of r sets is a machine-spirit.
func spirit(r *librarium.Rite, v invocation.Verse) bool {
	if r == nil || v.Subject == "" {
		return false
	}
	sub, ok := r.Subject(v.Subject)
	return ok && sub.Designation() == librarium.DesignationMachineSpirit
}

// foreseenPlan tells what a subject was read to show and whether it
// changes: a machine-spirit also the omen read and the omen commanded,
// «"on" → "off", changing», as the ritual of foresight tells it.
func foreseenPlan(p invocation.SubjectPlan, spirit bool) string {
	told := "unchanged"
	if p.Changing {
		told = "changing"
	}
	if !spirit {
		return told
	}
	return fmt.Sprintf("%s → %q, %s", readOmen(p), p.Target, told)
}

// readOmen is the omen a machine-spirit was read to show, quoted, or why
// none was.
func readOmen(p invocation.SubjectPlan) string {
	switch {
	case p.Unread:
		return "(no auspex)"
	case p.Privileged && p.Stale:
		return strconv.Quote(p.Read) + " (stale)"
	case p.Privileged:
		return "(read at invocation)"
	case p.Mute:
		return "(mute)"
	}
	return strconv.Quote(p.Read)
}

// readBefore is what a fallen machine-spirit was read to show before it
// was set: the omen read, quoted, marked when it fits no aspect.
func readBefore(p invocation.SubjectPlan) string {
	switch {
	case p.Unread:
		return "none (no auspex)"
	case p.Mute && p.Read == "":
		return "none (mute)"
	case p.Mute:
		return strconv.Quote(p.Read) + " (mute)"
	}
	return strconv.Quote(p.Read)
}

// planOf is the plan the foresight of res drew for the subject of v.
func planOf(res rituals.Result, v invocation.Verse) (invocation.SubjectPlan, bool) {
	for _, f := range res.Foresight {
		if f.Plan != nil && v.Subject != "" && f.Subject == v.Subject {
			return *f.Plan, true
		}
	}
	return invocation.SubjectPlan{}, false
}
