package augury

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// envOmen is the variable a trial hears its own omen as.
const envOmen = "SERVITOR_OMEN"

// readSpirit is what the auspex of a machine-spirit read.
type readSpirit struct {
	// omen is the one declared omen read; "" when none, or several. A mute
	// scan keeps what it printed, which no aspect fits.
	omen    string
	aspects []string // the aspects it fits, in the order of the rite's; none when mute
}

// hearing is the auspex of one machine-spirit, rendered and ready to be
// heard apart from the seer that prepared it.
type hearing struct {
	aspects []string          // the rite's aspects
	omens   map[string]string // the omen of each aspect, rendered and trimmed
	named   bool              // the spirit declares no omens: it shows aspect names
	scan    *speech           // nil when it is read by its trials
	trials  map[string]speech // the trial of every declared omen bearing one
	untried []string          // the declared omens bearing no trial
	silent  bool              // its words could not be rendered: it is mute
}

// spirit prepares the hearing of the machine-spirit subject sub: its omens
// rendered for their own aspects with the data-slate's inscriptions, or its
// decrees, and its words with the values of every auspex. It is spoken in
// the subject's tongue, else the rite's, else the settings', else bash,
// from the rite's hall, with the environment of the rite's commands; each
// trial also hears its own omen as {{omen}} and SERVITOR_OMEN.
func (s *seer) spirit(sub librarium.Subject, opts Options) hearing {
	ms := sub.Matter.(*librarium.MachineSpirit)
	h := hearing{aspects: s.rite.Aspects, omens: map[string]string{}, named: ms.Omens.IsZero()}
	var declared []string
	for _, aspect := range s.rite.Aspects {
		omen := aspect
		if !h.named {
			words, _ := ms.Omens.For(aspect)
			rendered, ok := s.render(words, s.valuesOf(aspect)[0])
			h.silent = h.silent || !ok
			omen = strings.TrimSpace(rendered)
		}
		h.omens[aspect] = omen
		if !slices.Contains(declared, omen) {
			declared = append(declared, omen)
		}
	}
	v := s.auspexValues(opts)
	base := speech{tongue: tongueOf(opts.Orders, ms.Tongue, s.rite.Tongue), hall: s.rite.Dir(),
		env: withoutOmen(invocation.Environ(s.rite, v)), patience: ms.Auspex.Wait()}
	if ms.Auspex.Scan != "" {
		sp := base
		var ok bool
		sp.words, ok = s.render(ms.Auspex.Scan, v)
		h.scan, h.silent = &sp, h.silent || !ok
		return h
	}
	scope := s.rite.Scope()
	scope.Omen = true
	h.trials = map[string]speech{}
	for _, omen := range declared {
		words, ok := ms.Auspex.Trials.For(omen)
		if !ok {
			h.untried = append(h.untried, omen)
			continue
		}
		tv := v
		tv.Omen = omen
		rendered, err := scope.Render(words, tv)
		h.silent = h.silent || err != nil
		sp := base
		sp.words, sp.env = rendered, append(slices.Clone(base.env), envOmen+"="+omen)
		h.trials[omen] = sp
	}
	return h
}

// withoutOmen strips an omen the servitor itself was given from env.
func withoutOmen(env []string) []string {
	return slices.DeleteFunc(env, func(kv string) bool { return strings.HasPrefix(kv, envOmen+"=") })
}

// hear speaks the auspex, its processes taking their places among places,
// and returns the aspects the machine-spirit fits.
func (h hearing) hear(ctx context.Context, places Places) readSpirit {
	if h.silent {
		return readSpirit{}
	}
	if h.scan != nil {
		return h.scanned(ctx, places)
	}
	return h.tried(ctx, places)
}

// scanned speaks the scan once: what it prints, trimmed, is the omen read,
// fitting every aspect whose omen it is; a spirit without omens must print
// aspect names alone, and fits each one printed. A scan that falls or is
// silent, or prints nothing or an omen never declared, fits none.
func (h hearing) scanned(ctx context.Context, places Places) readSpirit {
	heard, f := h.scan.speak(ctx, places, true)
	read := strings.TrimSpace(heard)
	if f != spoke || read == "" {
		return readSpirit{}
	}
	shown := []string{read}
	if h.named {
		shown = strings.Fields(read)
		for _, word := range shown {
			if !slices.Contains(h.aspects, word) {
				return readSpirit{omen: read}
			}
		}
	}
	if r := h.fits(shown); len(r.aspects) > 0 {
		return r
	}
	return readSpirit{omen: read} // mute, yet what it printed is kept for the tidings
}

// tried speaks every trial at once: the omens whose trial ends with the
// death-mark 0 are shown, and when none does, the omens bearing no trial.
// A trial that is silent mutes the whole auspex, and the others are slain.
func (h hearing) tried(ctx context.Context, places Places) readSpirit {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var mu sync.Mutex
	var shown []string
	silenced := false
	var wg sync.WaitGroup
	for omen, sp := range h.trials {
		wg.Go(func() {
			_, f := sp.speak(ctx, places, false)
			mu.Lock()
			defer mu.Unlock()
			switch f {
			case spoke:
				shown = append(shown, omen)
			case silent:
				silenced = true
				cancel()
			}
		})
	}
	wg.Wait()
	switch {
	case silenced:
		return readSpirit{}
	case len(shown) == 0:
		shown = h.untried
	}
	return h.fits(shown)
}

// fits is the reading of a machine-spirit showing the omens shown: every
// aspect whose omen is among them, and the omen read when it is one.
func (h hearing) fits(shown []string) readSpirit {
	var r readSpirit
	for _, aspect := range h.aspects {
		if slices.Contains(shown, h.omens[aspect]) {
			r.aspects = append(r.aspects, aspect)
		}
	}
	if len(shown) == 1 && len(r.aspects) > 0 {
		r.omen = shown[0]
	}
	return r
}
