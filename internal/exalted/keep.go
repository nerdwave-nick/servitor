package exalted

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// keeping is what an exalted servitor roused to keep the store was roused
// for: to sanction the rite under the sigil (its first twelve letters), or
// to abjure the sanction of the rite, or every sanction (all).
type keeping struct {
	deed  string // "sanction" or "abjure"
	rite  string
	sigil string
	all   bool
}

// parseKeeping reads argv after Mode: --sanction <rite> <sigil12>, or
// --abjure <rite> | --abjure --all.
func parseKeeping(args []string) (keeping, error) {
	switch {
	case len(args) == 3 && args[0] == SanctionWord:
		k := keeping{deed: "sanction", rite: args[1], sigil: args[2]}
		switch {
		case !librarium.ValidName(k.rite):
			return k, fmt.Errorf("%q names no rite", k.rite)
		case !shortSigil.MatchString(k.sigil):
			return k, fmt.Errorf("%q is not the first twelve letters of a sigil", k.sigil)
		}
		return k, nil
	case len(args) == 2 && args[0] == AbjureWord && args[1] == AllWord:
		return keeping{deed: "abjure", all: true}, nil
	case len(args) == 2 && args[0] == AbjureWord:
		k := keeping{deed: "abjure", rite: args[1]}
		if !librarium.ValidName(k.rite) {
			return k, fmt.Errorf("%q names no rite", k.rite)
		}
		return k, nil
	}
	deed := "sanction"
	if len(args) > 0 && args[0] == AbjureWord {
		deed = "abjure"
	}
	return keeping{deed: deed}, fmt.Errorf("the exalted servitor keeps the store when roused as %s %s <rite> <sigil>, "+
		"%s %s <rite> or %s %s %s, yet %d words were spoken", Mode, SanctionWord, Mode, AbjureWord, Mode, AbjureWord,
		AllWord, len(args))
}

// keep is the exalted servitor roused to keep the store of sanctions,
// hearing the master's servitor on in: it sanctions or abjures for the
// master at the other end of the channel, and answers once, naming its
// protocol. Every refusal leaves the store as it stood, its death-mark 2.
func (s *session) keep(args []string, in *os.File) int {
	k, err := parseKeeping(args)
	s.argv = argv{rite: k.rite, sigil: k.sigil}
	if err != nil {
		return s.forbid(k.deed, err)
	}
	if s.uid, err = identify(in, s.cfg.Getenv); err != nil {
		return s.forbid(k.deed, err)
	}
	done := make(chan struct{})
	defer close(done)
	s.lines = hear(in, done)
	if k.deed == "sanction" {
		return s.sanction(k)
	}
	return s.abjure(k)
}

// keeper is the authority with which the exalted servitor keeps the store.
func (s *session) keeper() sanction.Keeper {
	lord := s.cfg.Ground.Lord
	return sanction.Keeper{Sovereign: s.cfg.Sovereign, Lord: sanction.Owner{UID: int(lord.UID), GID: int(lord.GID)},
		Kin: s.cfg.Kin}
}

func (s *session) storeOf() sanction.Store { return sanction.Store{Root: s.cfg.Store, UID: s.uid} }

// sanction hears the unit to be sanctioned, checks it against argv and the
// channel's master, judges it as every invocation judges its plans, and
// lays it in the store, journaling the sigil it replaced.
func (s *session) sanction(k keeping) int {
	u, err := s.hearUnit(k)
	if err == nil {
		err = s.sanctionable(u)
	}
	if err != nil {
		return s.forbid(k.deed, err)
	}
	now := s.cfg.Now
	if now == nil {
		now = Machine().Now
	}
	was, err := s.storeOf().Lay(sanction.New(u, now().UTC()), s.keeper())
	if err != nil {
		return s.forbid(k.deed, err)
	}
	former := "none"
	switch {
	case was.Stood && was.Sigil == "":
		former = "unreadable"
	case was.Stood:
		former = sanction.Short(was.Sigil)
	}
	s.record(Entry{Deed: k.deed, Verdict: Sanctioned, Former: former})
	_ = s.reply(Reply{To: k.deed, Verdict: Sanctioned, Protocol: Protocol})
	return markFinished
}

// hearUnit hears the Sanctioning and the Parts that follow it, and reads the
// unit they hold, whose sigil must be the one the Sanctioning names and argv
// begins.
func (s *session) hearUnit(k keeping) (*sanction.Unit, error) {
	var hs Sanctioning
	if err := s.hearFirst(&hs); err != nil {
		return nil, err
	}
	switch {
	case hs.Protocol != Protocol:
		return nil, mismatched(hs.Protocol)
	case hs.UID != s.uid:
		return nil, fmt.Errorf("the words name uid %d, yet the channel's master is uid %d", hs.UID, s.uid)
	case hs.Rite != k.rite:
		return nil, fmt.Errorf("the words name the rite %q, yet the exalted servitor was roused for %q", hs.Rite, k.rite)
	case len(hs.Sigil) != 64 || sanction.Short(hs.Sigil) != k.sigil:
		return nil, fmt.Errorf("the words name the sigil %q, yet the exalted servitor was roused for %s",
			sanction.Short(hs.Sigil), k.sigil)
	case hs.Size <= 0 || hs.Size > UnitCap:
		return nil, fmt.Errorf("the unit is told to hold %d bytes, yet a unit holds at least one and at most %d",
			hs.Size, UnitCap)
	}
	data := make([]byte, 0, hs.Size)
	for len(data) < hs.Size {
		line, err := s.await()
		if err != nil {
			return nil, err
		}
		var p Part
		if err := Decode(line, &p); err != nil {
			return nil, err
		}
		if len(p.Part) == 0 || len(data)+len(p.Part) > hs.Size {
			return nil, fmt.Errorf("the parts of the unit hold other than the %d bytes it was told to hold", hs.Size)
		}
		data = append(data, p.Part...)
	}
	var u sanction.Unit
	if err := json.Unmarshal(data, &u, json.RejectUnknownMembers(true)); err != nil {
		return nil, fmt.Errorf("the unit is no true unit: %v", err)
	}
	if u.Sigil() != hs.Sigil {
		return nil, fmt.Errorf("the unit heard bears the sigil %s, not the sigil %s it was sanctioned under",
			sanction.Short(u.Sigil()), k.sigil)
	}
	return &u, nil
}

// hearFirst hears the first line of the master's servitor into v.
func (s *session) hearFirst(v any) error {
	line, err := s.await()
	if errors.Is(err, io.EOF) {
		return errors.New("the channel ended before its first line")
	}
	if err != nil {
		return err
	}
	return Decode(line, v)
}

// mismatched is the lament for the master's servitor of another protocol.
func mismatched(spoken int) error {
	return fmt.Errorf("the invoker speaks the protocol %d, yet the exalted servitor installed at %s speaks %d; "+
		"install the servitor anew so that both are of one forging", spoken, Installed, Protocol)
}

// sanctionable judges u as the unit of the master and rite the exalted
// servitor was roused for, whose every plan it could perform by the rules
// of every invocation: of a pattern it reads, its plans holding what that
// pattern holds, one plan for an edict and one for each aspect otherwise,
// each holding exalted verses in written order — or, for a state rite,
// subjects in written order and observances each named once, some plan
// holding any — every vessel and bound name beginning at the root beneath
// none of the forbidden halls, and its speech one the exalted servitor
// could speak.
func (s *session) sanctionable(u *sanction.Unit) error {
	switch {
	case !slices.Contains(sanction.Patterns, u.Pattern):
		return fmt.Errorf("the unit is of the pattern %q, yet this exalted servitor reads only %q and %q", u.Pattern,
			sanction.Pattern, sanction.PatternMarkII)
	case u.Misshapen() != "":
		return errors.New(u.Misshapen())
	case u.UID != s.uid:
		return fmt.Errorf("the unit is of uid %d, yet the channel's master is uid %d", u.UID, s.uid)
	case u.Rite != s.argv.rite:
		return fmt.Errorf("the unit is of the rite %q, yet the exalted servitor was roused for %q", u.Rite, s.argv.rite)
	case len(u.Aspects) == 0:
		return errors.New("the unit holds no plan")
	case u.Edict && (len(u.Aspects) != 1 || u.Aspects[0].Aspect != ""):
		return errors.New("the unit of an edict holds one plan, of no aspect")
	case u.State() && !slices.ContainsFunc(u.Aspects, func(p sanction.Plan) bool { return len(items(p)) > 0 }):
		return errors.New("the unit of the state rite holds no exalted subject or observance in any plan")
	}
	sn := sanction.New(u, time.Time{})
	var seen []string
	for _, p := range u.Aspects {
		of := fmt.Sprintf("the aspect %q", p.Aspect)
		switch {
		case u.Edict:
			of = "the edict"
		case !librarium.ValidName(p.Aspect) || slices.Contains(seen, p.Aspect):
			return fmt.Errorf("the unit holds a plan of %q, which names no aspect, or one named before", p.Aspect)
		}
		seen = append(seen, p.Aspect)
		if err := ordered(p, u.State()); err != nil {
			return fmt.Errorf("the plan of %s %w", of, err)
		}
		if err := performable(p, s.cfg.Voice != nil); err != nil {
			return fmt.Errorf("the plan of %s: %w", of, err)
		}
		if s.cfg.Weigh != nil && speaks(p) {
			sigil, _ := u.SigilOf(p.Aspect)
			if err := s.cfg.Weigh(Sitting{UID: s.uid, Sanction: sn, Plan: p, Sigil: sigil}); err != nil {
				return fmt.Errorf("the plan of %s: %w", of, err)
			}
		}
	}
	return nil
}

// ordered judges the order of what plan holds: exalted verses, at least
// one, numbered in written order; or, for a state rite, subjects by their
// places in written order, each named once, and observances each named
// once.
func ordered(p sanction.Plan, state bool) error {
	if !state {
		if len(p.Liturgy) == 0 {
			return errors.New("holds no exalted verse")
		}
		for i, v := range p.Liturgy {
			if v.Verse < 1 || (i > 0 && v.Verse <= p.Liturgy[i-1].Verse) {
				return errors.New("holds verses not numbered in written order")
			}
		}
		return nil
	}
	names := map[string]bool{}
	for i, v := range p.Subjects {
		if v.Place < 0 || (i > 0 && v.Place <= p.Subjects[i-1].Place) || names["s:"+v.Subject] {
			return errors.New("holds subjects not in written order, or one named twice")
		}
		names["s:"+v.Subject] = true
	}
	for _, v := range p.Observances {
		if names["o:"+v.Observance] {
			return fmt.Errorf("names %s twice", v.Named())
		}
		names["o:"+v.Observance] = true
	}
	return nil
}

// forbid refuses the deed of an exalted servitor roused to keep the store
// before anything was written.
func (s *session) forbid(deed string, why error) int {
	if errors.Is(why, io.EOF) {
		why = errors.New("the channel ended before its first line")
	}
	s.record(Entry{Deed: deed, Verdict: Refused, Lament: why.Error()})
	if s.cfg.Lament != nil {
		fmt.Fprintln(s.cfg.Lament, "servitor ✠ "+why.Error())
	}
	_ = s.reply(Reply{To: deed, Verdict: Refused, Protocol: Protocol, Lament: why.Error()})
	return markRefused
}
