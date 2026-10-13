package invocation

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// The observances of a rite of aspects are spoken in place
// among its subjects, never deferred. Each changing subject is preceded by
// its introits not yet spoken, and followed by the recessionals it is the
// last changing subject to name; an unchanged subject passes them over.
// An observance is spoken only when its guards admit the invocation: "to"
// when it names the aspect invoked, "from" when it names the former aspect,
// which an unknown former aspect never is. Observances are spoken as Mark
// I's incantations and litanies, with their tongue, patience, offerings and
// reversion; a fall reverts them among the subjects, in reverse order.

// examined is a subject of a rite of aspects as the pre-flight examined it: its
// foresight, nil when it is heretical, and its step, nil unless it changes.
type examined struct {
	foresight *Foresight
	step      performer
}

// observances examines every observance of the rite of aspects for the aspect
// invoked, whether or not it will be spoken, and makes each ready by its
// name; one found heretical is absent.
func (p *preflight) observances() map[string]performer {
	ready := map[string]performer{}
	for j, o := range p.rite.Observances {
		p.index, p.observance, p.exalted = j, o.Name, o.Exalted()
		var step performer
		switch s := o.Speech.(type) {
		case *librarium.Incantation:
			step = p.incantation(s)
		case *librarium.Litany:
			step = p.litany(s)
		}
		if sp, ok := step.(*speaker); ok {
			sp.env = unomened(sp.env)
			step = sp
			if p.exalted {
				step = &exaltedStep{planned: sp, at: p.pointer("exalted")}
			}
			ready[o.Name] = &observanceStep{performer: step, name: o.Name}
		}
	}
	p.observance, p.exalted = "", false
	return ready
}

// ordain sets the subjects examined in written order, and the observances
// ready around those that change.
func (inv *Invocation) ordain(subjects []examined, ready map[string]performer) {
	r, opts := inv.rite, inv.opts
	last := map[string]int{} // of each recessional: the last changing subject naming it
	for i, sub := range subjects {
		if sub.step != nil {
			for _, name := range r.Subjects[i].Recessional {
				last[name] = i
			}
		}
	}
	spoken := map[string]bool{}
	speak := func(name string) {
		o, _ := r.Observance(name)
		step, ok := ready[name]
		if !ok || spoken[name] || !admits(o, opts.Aspect, opts.Former) {
			return
		}
		spoken[name] = true
		inv.steps = append(inv.steps, step)
		inv.foresight = append(inv.foresight, step.foresee())
	}
	for i, sub := range subjects {
		if sub.step != nil {
			for _, name := range r.Subjects[i].Introit {
				speak(name)
			}
		}
		if sub.foresight != nil {
			inv.foresight = append(inv.foresight, *sub.foresight)
		}
		if sub.step == nil {
			continue
		}
		inv.steps = append(inv.steps, sub.step)
		for _, name := range r.Subjects[i].Recessional {
			if last[name] == i {
				speak(name)
			}
		}
	}
}

// admits reports whether the guards of o admit an invocation into aspect
// out of former, "" when the former aspect is unknown.
func admits(o librarium.Observance, aspect, former string) bool {
	return (o.To == nil || slices.Contains(o.To, aspect)) &&
		(o.From == nil || former != "" && slices.Contains(o.From, former))
}

// unomened casts the omen out of env: only enact words and trials hear it.
func unomened(env []string) []string {
	return slices.DeleteFunc(env, func(kv string) bool { return strings.HasPrefix(kv, envOmen+"=") })
}

// observanceStep is an observance made ready by the pre-flight: spoken as
// its incantation or litany, its fall named by the observance.
type observanceStep struct {
	performer
	name string
}

func (o *observanceStep) perform(ctx context.Context) error {
	if err := o.performer.perform(ctx); err != nil {
		return fmt.Errorf("the observance %q fell: %w", o.name, err)
	}
	return nil
}

func (o *observanceStep) uttered() (said, unsaid string) { return uttered(o.performer) }
func (o *observanceStep) wrapped() performer             { return o.performer }
