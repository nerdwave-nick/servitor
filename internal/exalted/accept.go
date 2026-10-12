package exalted

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/dominion"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// argv is what the exalted servitor was roused for: <rite> <sigil12>
// [<aspect>]; an edict bears no aspect.
type argv struct {
	rite, sigil, aspect string
	edict               bool
}

var shortSigil = regexp.MustCompile(`^[0-9a-f]{12}$`)

// parseArgv reads argv.
func parseArgv(args []string) (argv, error) {
	if len(args) < 2 || len(args) > 3 {
		return argv{}, fmt.Errorf("the exalted servitor is roused as %s <rite> <sigil> [<aspect>], yet %d words "+
			"were spoken", Mode, len(args))
	}
	a := argv{rite: args[0], sigil: args[1], edict: len(args) == 2}
	if !a.edict {
		a.aspect = args[2]
	}
	switch {
	case !librarium.ValidName(a.rite):
		return argv{}, fmt.Errorf("%q names no rite", a.rite)
	case !shortSigil.MatchString(a.sigil):
		return argv{}, fmt.Errorf("%q is not the first twelve letters of a sigil", a.sigil)
	case !a.edict && !librarium.ValidName(a.aspect):
		return argv{}, fmt.Errorf("%q names no aspect", a.aspect)
	}
	return a, nil
}

// wordCap is the most a word given at invocation may hold.
const wordCap = 4 << 10

// accept hears the handshake and accepts the session: the handshake agrees
// with argv and the channel's master, the sanction is the root's and bears
// the sigil, its plan may be performed, and the root-side dominions are
// seized. Nothing is touched before it returns.
func (s *session) accept() error {
	line, err := s.await()
	if err != nil {
		return err
	}
	if err := Decode(line, &s.hs); err != nil {
		return err
	}
	if err := s.agrees(); err != nil {
		return err
	}
	sn, err := sanction.Store{Root: s.cfg.Store, UID: s.uid}.ReadHallowed(s.argv.rite, s.cfg.Sovereign)
	switch {
	case err != nil:
		return err
	case sn == nil:
		return fmt.Errorf("the rite %q of uid %d stands unsanctioned: no sanction is kept for it", s.argv.rite, s.uid)
	case sn.Edict != s.argv.edict:
		return fmt.Errorf("the sanction of %q is that of %s", s.argv.rite,
			map[bool]string{true: "an edict", false: "a rite of aspects"}[sn.Edict])
	}
	s.sn = sn
	if sigil, ok := sn.SigilOf(s.argv.aspect); !ok || sigil != s.hs.Sigil {
		return fmt.Errorf("the sanction of %q bears no plan of the sigil %s for %s; it has lapsed, or it was "+
			"never sanctioned so", s.argv.rite, s.argv.sigil, s.of())
	}
	n, _ := sn.Narrow(s.argv.aspect)
	s.plan = n.Aspects[0]
	if err := s.heedable(); err != nil {
		return err
	}
	if err := s.performable(); err != nil {
		return err
	}
	if err := s.seize(); err != nil {
		return err
	}
	return s.lendVoice()
}

// of names the aspect of the session, or that it is an edict.
func (s *session) of() string {
	if s.argv.edict {
		return "the edict"
	}
	return fmt.Sprintf("the aspect %q", s.argv.aspect)
}

// agrees checks the handshake against argv and the channel's master.
func (s *session) agrees() error {
	h := s.hs
	switch {
	case h.Protocol != Protocol:
		return fmt.Errorf("the invoker speaks the protocol %d, yet the exalted servitor installed at "+
			"/usr/bin/servitor speaks %d; install the servitor anew so that both are of one forging",
			h.Protocol, Protocol)
	case h.UID != s.uid:
		return fmt.Errorf("the handshake names uid %d, yet the channel's master is uid %d", h.UID, s.uid)
	case h.Rite != s.argv.rite:
		return fmt.Errorf("the handshake names the rite %q, yet the exalted servitor was roused for %q", h.Rite,
			s.argv.rite)
	case h.Aspect != s.argv.aspect:
		return fmt.Errorf("the handshake names the aspect %q, yet the exalted servitor was roused for %q",
			h.Aspect, s.argv.aspect)
	case len(h.Sigil) != 64 || sanction.Short(h.Sigil) != s.argv.sigil:
		return fmt.Errorf("the handshake names the sigil %q, yet the exalted servitor was roused for %s",
			sanction.Short(h.Sigil), s.argv.sigil)
	case s.argv.edict && h.Former != "":
		return errors.New("an edict bears no former aspect")
	}
	return nil
}

// heedable checks the words given at invocation: the former aspect and
// inscriptions the sanction declares, none longer than wordCap, none
// bearing a NUL or a newline.
func (s *session) heedable() error {
	words := map[string]string{"SERVITOR_FORMER_ASPECT": s.hs.Former}
	for key, v := range s.hs.Inscriptions {
		name := placeholder.InscriptionVariable(key)
		if !slices.Contains(s.sn.Environment.Keys, name) {
			return fmt.Errorf("the inscription %q is no inscription of %q", key, s.argv.rite)
		}
		words[name] = v
	}
	for name, v := range words {
		switch {
		case len(v) > wordCap:
			return fmt.Errorf("the words of %s exceed %d bytes", name, wordCap)
		case strings.ContainsAny(v, "\x00\n"):
			return fmt.Errorf("the words of %s bear a NUL or a newline", name)
		}
	}
	return nil
}

// forbiddenHalls are where no exalted vessel lies.
var forbiddenHalls = []string{"/proc", "/sys", "/dev"}

// performable checks every verse of the plan: a vessel or bound name begins
// at the root, beneath none of the forbidden halls, and speech has a voice.
func (s *session) performable() error {
	for _, v := range s.plan.Liturgy {
		switch v.Kind {
		case librarium.KindIncantation, librarium.KindLitany:
			if s.cfg.Voice == nil {
				return fmt.Errorf("verse %d is exalted %s, which this exalted servitor cannot yet speak; "+
					"nothing was touched", v.Verse, v.Kind.Key())
			}
			continue
		}
		if !filepath.IsAbs(v.Target) {
			return fmt.Errorf("the vessel %q of verse %d does not begin at the root", v.Target, v.Verse)
		}
		clean := filepath.Clean(v.Target)
		for _, hall := range forbiddenHalls {
			if clean == hall || strings.HasPrefix(clean, hall+"/") {
				return fmt.Errorf("the vessel %s of verse %d lies beneath %s, where no vessel is kept", v.Target,
					v.Verse, hall)
			}
		}
	}
	return nil
}

// seize seizes the root-side dominions, waiting as every invocation waits:
// the rite's, rites/<uid>-<rite>, then that of every vessel and bound name
// of the plan, vessels/<sha256 of the place it truly dwells>, in sorted
// order. A word, the end of the channel or a halt end the waiting.
func (s *session) seize() error {
	var vessels []string
	for _, v := range s.plan.Liturgy {
		if v.Kind == librarium.KindIncantation || v.Kind == librarium.KindLitany {
			continue
		}
		sum := sha256.Sum256([]byte(s.cfg.Ground.Resolve(v.Target)))
		vessels = append(vessels, filepath.Join(s.cfg.Dominions, "vessels", hex.EncodeToString(sum[:])))
	}
	slices.Sort(vessels)
	places := append([]string{filepath.Join(s.cfg.Dominions, "rites", fmt.Sprintf("%d-%s", s.uid, s.argv.rite))},
		slices.Compact(vessels)...)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type seized struct {
		d   *dominion.Dominion
		err error
	}
	done := make(chan seized, 1)
	go func() {
		d, err := dominion.Seize(ctx, places, nil)
		done <- seized{d, err}
	}()
	var why error
	select {
	case r := <-done:
		if r.err != nil {
			return fmt.Errorf("the dominion of the exalted servitor cannot be seized: %v", r.err)
		}
		s.dominion = r.d
		return nil
	case h := <-s.lines:
		why = errors.New("the invoker spoke before the session was accepted")
		if h.err != nil {
			why = errors.New("the channel ended while the exalted servitor awaited its dominion")
		}
	case <-s.cfg.Halt:
		why = errors.New("the exalted servitor was halted while it awaited its dominion")
	}
	cancel()
	if r := <-done; r.d != nil {
		r.d.Relinquish()
	}
	return why
}

// lendVoice raises the speech of the session when its plan holds speech.
func (s *session) lendVoice() error {
	speaks := slices.ContainsFunc(s.plan.Liturgy, func(v sanction.Verse) bool {
		return v.Kind == librarium.KindIncantation || v.Kind == librarium.KindLitany
	})
	if !speaks {
		return nil
	}
	sp, err := s.cfg.Voice(Sitting{UID: s.uid, Sanction: s.sn, Plan: s.plan, Sigil: s.hs.Sigil,
		Former: s.hs.Former, Inscriptions: s.hs.Inscriptions})
	if err != nil {
		return err
	}
	s.speaker = sp
	return nil
}
