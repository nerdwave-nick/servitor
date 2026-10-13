package invocation

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// envOmen is the variable enact words hear the omen they set as.
const envOmen = "SERVITOR_OMEN"

// spirit examines a machine-spirit subject: the omen of the aspect invoked,
// the enact words that set it, and what it was read to show. It is
// unchanged when it was read to show that omen; one without an auspex, or
// whose auspex was mute, is changing. It returns no foresight for a subject
// found heretical.
func (p *preflight) spirit(i int, ms *librarium.MachineSpirit) (*Foresight, performer) {
	omen, okOmen := p.omen(ms)
	sp, okUtterance := p.utterance(librarium.Utterance{Tongue: ms.Tongue, Patience: ms.Patience})
	words, okWords := p.enact(ms, omen, okOmen)
	if !okOmen || !okUtterance || !okWords || !p.speakable(sp.Tongue) {
		return nil, nil
	}
	sp.Words, sp.Argv = words, []string{sp.Tongue, "-c", words}
	var read Reading
	if i < len(p.opts.Readings) {
		read = p.opts.Readings[i]
	}
	plan := &SubjectPlan{Read: read.Omen, Target: omen, Unread: ms.Auspex == nil}
	plan.Mute = !plan.Unread && len(read.Aspects) == 0
	plan.Changing = plan.Unread || !slices.Contains(read.Aspects, p.opts.Aspect)
	f := &Foresight{Verse: p.verse(0, ""), Plan: plan}
	if !plan.Changing {
		return f, nil
	}
	f.Speech = &sp
	return f, &spiritStep{v: f.Verse, index: i, ms: ms, rite: p.rite, opts: p.opts, values: p.values, speech: sp,
		omen: omen, read: read}
}

// omen renders the omen the invoked aspect demands of the machine-spirit:
// the aspect's own name when it declares no omens.
func (p *preflight) omen(ms *librarium.MachineSpirit) (string, bool) {
	if ms.Omens.IsZero() {
		return p.opts.Aspect, true
	}
	text, has := ms.Omens.For(p.opts.Aspect)
	field := mapField(p, "omens", ms.Omens)
	if !has {
		p.denounce(field, "no omen is written for the aspect %q, and no \"*\" serves in its stead", p.opts.Aspect)
		return "", false
	}
	omen, ok := p.render(field, text)
	return strings.TrimSpace(omen), ok
}

// enact renders the enact words that set omen, which hear it as {{omen}}.
func (p *preflight) enact(ms *librarium.MachineSpirit, omen string, known bool) (string, bool) {
	if !known {
		return "", false
	}
	key, has := ms.Enact.Key(omen)
	text, _ := ms.Enact.For(omen)
	if !has {
		p.denounce("enact", "no enact words are written for the omen %q, and no \"*\" serves in their stead", omen)
		return "", false
	}
	words, err := enactScope(p.rite).Render(text, withOmen(p.values, omen))
	if err != nil {
		p.denounce("enact/"+key, "%s", firstHeresy(err))
		return "", false
	}
	return words, true
}

func enactScope(r *librarium.Rite) placeholder.Scope {
	scope := r.Scope()
	scope.Omen = true
	return scope
}

func withOmen(v placeholder.Values, omen string) placeholder.Values {
	v.Omen = omen
	return v
}

// spiritStep is a changing machine-spirit made ready by the pre-flight.
type spiritStep struct {
	v      Verse
	index  int // of the subject among the rite's
	ms     *librarium.MachineSpirit
	rite   *librarium.Rite
	opts   Options
	values placeholder.Values
	speech Speech // the enact words that set omen
	omen   string // the omen commanded
	read   Reading
	begun  bool
	said   string // what perform uttered
	unsaid string // what revert uttered
}

func (s *spiritStep) verse() Verse { return s.v }

func (s *spiritStep) foresee() Foresight {
	sp := s.speech
	return Foresight{Verse: s.v, Speech: &sp}
}

func (s *spiritStep) uttered() (said, unsaid string) { return s.said, s.unsaid }

// voice speaks argv as the enact words of omen are spoken: in the rite's
// hall, with the environment of the rite's commands and the omen.
func (s *spiritStep) voice(argv []string, omen string) voice {
	env := unomened(environ(s.rite.Dir(), false, s.values, s.rite.InscriptionKeys()))
	return voice{argv: argv, dir: s.rite.Dir(), env: append(env, envOmen+"="+omen), patience: s.speech.Patience}
}

// perform speaks the enact words, then proves the subject shows the omen
// commanded; a machine-spirit without an auspex is never proven.
func (s *spiritStep) perform(ctx context.Context) error {
	s.begun = true
	var err error
	if s.said, err = s.voice(s.speech.Argv, s.omen).speak(ctx); err != nil {
		return err
	}
	return s.prove(ctx, s.opts.Aspect, s.omen)
}

// prove reads the subject anew: it must fit aspect, whose omen is omen.
func (s *spiritStep) prove(ctx context.Context, aspect, omen string) error {
	if s.ms.Auspex == nil {
		return nil
	}
	if s.opts.Reader == nil {
		return fmt.Errorf("the subject %q cannot be read anew to prove it shows the omen %q", s.v.Subject, omen)
	}
	got := s.opts.Reader(ctx, s.index)
	if slices.Contains(got.Aspects, aspect) {
		return nil
	}
	found := "no omen"
	if got.Omen != "" {
		found = fmt.Sprintf("the omen %q", got.Omen)
	}
	return fmt.Errorf("the subject %q shows %s; %q was commanded", s.v.Subject, found, omen)
}

// revert enacts the omen the subject was read to show before it was set,
// or, when it was not read to show one, the omen of the former aspect, and
// proves it. With neither it cannot be restored.
func (s *spiritStep) revert() (bool, error) {
	if !s.begun {
		return false, nil
	}
	aspect, omen, err := s.restoration()
	if err != nil {
		return true, err
	}
	text, _ := s.ms.Enact.For(omen)
	words, err := enactScope(s.rite).Render(text, withOmen(s.values, omen))
	if err != nil {
		return true, fmt.Errorf("the enact words that restore the omen %q cannot be illuminated: %s", omen, firstHeresy(err))
	}
	if s.unsaid, err = s.voice([]string{s.speech.Tongue, "-c", words}, omen).speak(context.Background()); err != nil {
		return true, fmt.Errorf("the restoration of the omen %q fell: %w", omen, err)
	}
	return true, s.prove(context.Background(), aspect, omen)
}

// restoration is the aspect whose omen restores the subject, and that omen.
func (s *spiritStep) restoration() (aspect, omen string, err error) {
	if s.read.Omen != "" && len(s.read.Aspects) > 0 {
		return s.read.Aspects[0], s.read.Omen, nil
	}
	former := s.opts.Former
	if former == "" {
		return "", "", fmt.Errorf("the subject %q cannot be restored: it showed no omen before it was set, and "+
			"the former aspect is unknown, so no omen is known to enact", s.v.Subject)
	}
	if s.ms.Omens.IsZero() {
		return former, former, nil
	}
	text, _ := s.ms.Omens.For(former)
	v := placeholder.Values{Aspect: former, Rite: s.rite.Name, Inscriptions: s.opts.Recorded}
	omen, err = s.rite.Scope().Render(text, v)
	if err != nil {
		return "", "", fmt.Errorf("the omen of the former aspect %q cannot be illuminated: %s", former, firstHeresy(err))
	}
	return former, strings.TrimSpace(omen), nil
}
