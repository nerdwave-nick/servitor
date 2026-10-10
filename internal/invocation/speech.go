package invocation

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// incantation examines an incantation: its command and its utterance.
func (p *preflight) incantation(s *librarium.Incantation) performer {
	words, okWords := p.words("incantation", s.Command)
	sp, okUtterance := p.utterance(s.Utterance)
	if !okWords || !okUtterance {
		return nil
	}
	sp.Words, sp.Argv = words, []string{sp.Tongue, "-c", words}
	if !p.speakable(sp.Tongue) {
		return nil
	}
	return p.spoken(s.Kind(), sp)
}

// litany examines a litany: its scroll, which must stand once the steps
// before it are performed, its offerings, rendered and with a leading "~"
// borne to the home of the faithful, and its utterance.
func (p *preflight) litany(s *librarium.Litany) performer {
	field := mapField(p, "litany", s.Scroll)
	text, okWords := p.words("litany", s.Scroll)
	sp, ok := p.utterance(s.Utterance)
	ok = ok && okWords
	for i, o := range s.Offerings {
		offering, okOffering := p.words(fmt.Sprintf("offerings/%d", i), o)
		sp.Offerings = append(sp.Offerings, librarium.ExpandHome(offering))
		ok = ok && okOffering
	}
	if !ok {
		return nil
	}
	sp.Words = p.rite.ResolvePath(text)
	executable, err := recitable(p.world, sp.Words)
	if err != nil {
		p.denounce(field, "%v", err)
		return nil
	}
	sp.Shebang = executable
	sp.Argv = litanyArgv(sp)
	if (!sp.Shebang || sp.Reversion != "") && !p.speakable(sp.Tongue) {
		return nil
	}
	return p.spoken(s.Kind(), sp)
}

// words renders what the invoked aspect demands of the aspect map written
// at field.
func (p *preflight) words(field string, m librarium.AspectMap[string]) (string, bool) {
	text, has := m.For(p.opts.Aspect)
	field = mapField(p, field, m)
	if !has {
		p.denounce(field, "nothing is written for the aspect %q, and no \"*\" serves in its stead", p.opts.Aspect)
		return "", false
	}
	return p.render(field, text)
}

// utterance resolves the tongue (step > rite > settings > bash), the
// patience (step > settings > default) and renders the reversion.
func (p *preflight) utterance(u librarium.Utterance) (Speech, bool) {
	sp := Speech{Tongue: librarium.DefaultTongue, Patience: librarium.DefaultPatience}
	for _, t := range []string{p.opts.Orders.Tongue, p.rite.Tongue, u.Tongue} {
		if t != "" {
			sp.Tongue = t
		}
	}
	for _, d := range []time.Duration{p.opts.Orders.Patience, u.Patience} {
		if d > 0 {
			sp.Patience = d
		}
	}
	if u.Reversion.IsZero() {
		return sp, true
	}
	var ok bool
	sp.Reversion, ok = p.words("reversion", u.Reversion)
	return sp, ok
}

// speakable denounces a tongue this machine does not speak.
func (p *preflight) speakable(tongue string) bool {
	if _, err := exec.LookPath(tongue); err != nil {
		p.denounce("tongue", "the tongue %q is spoken nowhere on this machine: no program of that name is to "+
			"be found along its PATH, so the step could never be uttered — name another \"tongue\" for the step, "+
			"the rite or the settings", tongue)
		return false
	}
	return true
}

// recitable reports whether the scroll that will stand at path, once the
// steps before it are performed, may be executed; a scroll that will not
// stand is denounced.
func recitable(w world, path string) (bool, error) {
	real, e, err := follow(w, path)
	switch {
	case err != nil:
		return false, lament(path, err)
	case e.form == absent:
		return false, fmt.Errorf("no scroll stands at %s for the litany to recite", real)
	case e.form != scroll:
		return false, fmt.Errorf("%s is no scroll that a litany could recite", real)
	}
	return e.mode&0o111 != 0, nil
}

// litanyArgv is how a litany is recited: by its own shebang when it may be
// executed, else through its tongue; it is offered exactly its offerings.
func litanyArgv(sp Speech) []string {
	argv := []string{sp.Words}
	if !sp.Shebang {
		argv = []string{sp.Tongue, sp.Words}
	}
	return append(argv, sp.Offerings...)
}

// spoken makes the examined speech ready to be performed.
func (p *preflight) spoken(k librarium.Kind, sp Speech) performer {
	target := sp.Words
	return &speaker{
		v:      p.verse(k, target),
		speech: sp,
		litany: k == librarium.KindLitany,
		dir:    p.rite.Dir(),
		env:    environ(p.rite.Dir(), p.values, p.rite.InscriptionKeys()),
	}
}

// speaker is an incantation or litany made ready by the pre-flight.
type speaker struct {
	v      Verse
	speech Speech
	litany bool
	dir    string
	env    []string
	said   string // what perform uttered
	unsaid string // what revert uttered
}

func (s *speaker) verse() Verse { return s.v }

func (s *speaker) foresee() Foresight {
	sp := s.speech
	return Foresight{Verse: s.v, Speech: &sp}
}

func (s *speaker) voice(argv []string) voice {
	return voice{argv: argv, dir: s.dir, env: s.env, patience: s.speech.Patience}
}

// perform speaks the step. Whether a litany may be executed is asked of the
// machine anew, for the steps before it may have changed its scroll.
func (s *speaker) perform(ctx context.Context) error {
	argv := s.speech.Argv
	if s.litany {
		sp := s.speech
		if executable, err := recitable(disk{}, sp.Words); err == nil {
			sp.Shebang = executable
		}
		argv = litanyArgv(sp)
	}
	var err error
	s.said, err = s.voice(argv).speak(ctx)
	return err
}

// revert speaks the reversion in the tongue, whether or not the step itself
// triumphed; a step without reversion reverts nothing.
func (s *speaker) revert() (bool, error) {
	if s.speech.Reversion == "" {
		return false, nil
	}
	var err error
	s.unsaid, err = s.voice([]string{s.speech.Tongue, "-c", s.speech.Reversion}).speak(context.Background())
	if err != nil {
		return true, fmt.Errorf("the reversion fell: %w", err)
	}
	return true, nil
}

func (s *speaker) uttered() (said, unsaid string) { return s.said, s.unsaid }
