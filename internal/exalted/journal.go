package exalted

import (
	"fmt"
	"log/syslog"
	"strconv"
	"strings"
)

// Entry is one act of the exalted servitor, as its journal keeps it.
type Entry struct {
	UID    int
	Rite   string
	Aspect string // "" for an edict
	Verse  int    // 0 for an act of the session itself
	Sigil  string // the first twelve letters
	// Former is the sigil a sanction replaced (its first twelve letters),
	// "none" or "unreadable"; only a sanctioning tells it.
	Former  string
	Deed    string // accept, perform, revert, finish, abort, fall, sanction or abjure
	Verdict Verdict
	Lament  string // why it was refused, fell or faltered
}

// String is the entry as one line of the journal.
func (e Entry) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "uid=%d rite=%s", e.UID, strconv.Quote(e.Rite))
	if e.Aspect != "" {
		fmt.Fprintf(&b, " aspect=%s", strconv.Quote(e.Aspect))
	}
	if e.Verse > 0 {
		fmt.Fprintf(&b, " verse=%d", e.Verse)
	}
	fmt.Fprintf(&b, " sigil=%s", e.Sigil)
	if e.Former != "" {
		fmt.Fprintf(&b, " former=%s", e.Former)
	}
	fmt.Fprintf(&b, " deed=%s verdict=%s", e.Deed, e.Verdict)
	if e.Lament != "" {
		fmt.Fprintf(&b, " lament=%s", strconv.Quote(e.Lament))
	}
	return b.String()
}

// Journal keeps every act of the exalted servitor where the root keeps its
// record. The trials keep it in memory.
type Journal interface {
	Record(Entry) error
}

// syslogJournal keeps the acts in the machine's journal, as the root's own
// tidings of authority.
type syslogJournal struct{ w *syslog.Writer }

// SystemJournal opens the machine's journal; the exalted servitor accepts
// no session it could not record.
func SystemJournal() (Journal, error) {
	w, err := syslog.New(syslog.LOG_AUTHPRIV|syslog.LOG_NOTICE, "servitor")
	if err != nil {
		return nil, fmt.Errorf("the journal of the machine cannot be opened, and the exalted servitor performs "+
			"nothing it cannot record: %v", err)
	}
	return syslogJournal{w}, nil
}

func (j syslogJournal) Record(e Entry) error {
	switch e.Verdict {
	case Refused, Fell, Faltered:
		return j.w.Warning(e.String())
	}
	return j.w.Notice(e.String())
}
