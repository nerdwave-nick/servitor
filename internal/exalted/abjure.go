package exalted

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// abjure hears the Abjuring and strikes the sanction it names, or every
// sanction of the master, journaling each with its sigil.
func (s *session) abjure(k keeping) int {
	var a Abjuring
	err := s.hearFirst(&a)
	switch {
	case err != nil:
	case a.Protocol != Protocol:
		err = mismatched(a.Protocol)
	case a.UID != s.uid:
		err = fmt.Errorf("the words name uid %d, yet the channel's master is uid %d", a.UID, s.uid)
	case a.All != k.all || (a.All && a.Rite != ""):
		err = errors.New("the words and the words that roused the exalted servitor do not agree whether every " +
			"sanction is abjured")
	case a.Rite != k.rite:
		err = fmt.Errorf("the words name the rite %q, yet the exalted servitor was roused for %q", a.Rite, k.rite)
	}
	if err != nil {
		return s.forbid(k.deed, err)
	}
	store, keeper := s.storeOf(), s.keeper()
	names := []string{k.rite}
	if k.all {
		if names, err = store.Kept(keeper); err == nil && len(names) == 0 {
			err = fmt.Errorf("no sanction is kept for uid %d", s.uid)
		}
		if err != nil {
			return s.forbid(k.deed, err)
		}
	}
	var abjured []string
	for _, name := range names {
		was, err := store.Strike(name, keeper)
		if errors.Is(err, fs.ErrNotExist) {
			err = fmt.Errorf("no sanction of %q is kept for uid %d", name, s.uid)
		}
		if err != nil && len(abjured) == 0 {
			return s.forbid(k.deed, err)
		}
		if err != nil {
			s.record(Entry{Rite: name, Deed: k.deed, Verdict: Fell, Lament: err.Error()})
			_ = s.reply(Reply{To: k.deed, Verdict: Fell, Protocol: Protocol, Lament: err.Error(), Abjured: abjured})
			return markReverted
		}
		sigil := "unreadable"
		if was.Sigil != "" {
			sigil = sanction.Short(was.Sigil)
		}
		s.record(Entry{Rite: name, Sigil: sigil, Deed: k.deed, Verdict: Abjured})
		abjured = append(abjured, name)
	}
	_ = s.reply(Reply{To: k.deed, Verdict: Abjured, Protocol: Protocol, Abjured: abjured})
	return markFinished
}
