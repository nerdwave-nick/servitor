package exalted

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// TestServe_RefusesUnlessItStandsEnshrined: the exalted servitor judges the
// scroll it runs from (/proc/self/exe) before it hears a word: the root's,
// written by none but it, beneath halls of the root's own, and the very
// scroll at its name. Otherwise it refuses with words naming the remedy,
// and touches nothing.
func TestServe_RefusesUnlessItStandsEnshrined(t *testing.T) {
	for what, mar := range map[string]func(sh *shrine){
		"a scroll others may write": func(sh *shrine) { must(sh.t, os.Chmod(sh.at("/usr/bin/servitor"), 0o777)) },
		"beneath a hall others may write": func(sh *shrine) {
			must(sh.t, os.Chmod(sh.at("/usr/bin"), 0o777))
		},
		"outside the ground": func(sh *shrine) {
			p := filepath.Join(sh.t.TempDir(), "servitor")
			must(sh.t, os.WriteFile(p, []byte("\x7fELF"), 0o755))
			sh.cfg.Self = p + "-self"
			must(sh.t, os.Symlink(p, sh.cfg.Self))
		},
		"unnamed": func(sh *shrine) { sh.cfg.Self = "" },
	} {
		t.Run(what, func(t *testing.T) {
			sh := newShrine(t, nil)
			mar(sh)
			inv := sh.rouseFor("on")
			r := inv.say(sh.handshake("on"))
			if r.Verdict != Refused || r.To != "handshake" || r.Protocol != Protocol ||
				!strings.Contains(r.Lament, "install the servitor anew") {
				t.Fatalf("heard %+v", r)
			}
			if m := inv.ended(); m != 2 {
				t.Fatalf("refused with the death-mark %d", m)
			}
			if got := sh.journal.deeds(); len(got) != 1 || got[0] != "accept refused" {
				t.Errorf("journaled %v", got)
			}
			sh.untouched()
		})
	}
}

// TestAnswer_TellsTheProtocolOfTheInstalledServitor: the invoker's servitor
// hears the protocol of the answer to its handshake before anything else
// of it, however the installed servitor's forging words the rest: an answer
// of another protocol, or naming none, is a Mismatch whose words bid the
// master install the servitor anew; only an answer of this protocol is
// heard strictly.
func TestAnswer_TellsTheProtocolOfTheInstalledServitor(t *testing.T) {
	for what, c := range map[string]struct {
		line      string
		installed int // the protocol of a Mismatch; -1 for none
		verdict   Verdict
		fails     bool // another lament than a Mismatch
	}{
		"accepted":                {line: `{"to":"handshake","verdict":"accepted","protocol":1,"verses":[1,3]}`, installed: -1, verdict: Accepted},
		"refused":                 {line: `{"to":"handshake","verdict":"refused","protocol":1,"lament":"no"}`, installed: -1, verdict: Refused},
		"a later forging":         {line: `{"to":"handshake","verdict":"refused","protocol":2,"lament":"x","omen":{"a":1}}`, installed: 2},
		"an earlier forging":      {line: `{"to":"handshake","verdict":"accepted","verses":[1]}`, installed: 0},
		"no protocol at all":      {line: `{"verdict":"refused","lament":"the protocol is unknown"}`, installed: 0},
		"words of another tongue": {line: `{"protocol":"one"}`, installed: -1, fails: true},
		"no true word":            {line: `ritual __exalt is unknown`, installed: -1, fails: true},
		"an unknown member of this protocol": {line: `{"to":"handshake","verdict":"accepted","protocol":1,"omen":1}`,
			installed: -1, fails: true},
	} {
		t.Run(what, func(t *testing.T) {
			r, err := NewChannel(strings.NewReader(c.line+"\n"), nil).Answer()
			var mm *Mismatch
			switch {
			case c.installed >= 0:
				if !errors.As(err, &mm) || mm.Installed != c.installed {
					t.Fatalf("heard %+v, %v", r, err)
				}
				if !strings.Contains(err.Error(), "install the servitor anew") ||
					!strings.Contains(err.Error(), "/usr/bin/servitor") {
					t.Errorf("the words do not bid the master install the servitor anew: %v", err)
				}
			case c.fails:
				if err == nil || errors.As(err, &mm) {
					t.Fatalf("heard %+v, %v", r, err)
				}
			default:
				if err != nil || r.Verdict != c.verdict || r.Protocol != Protocol {
					t.Fatalf("heard %+v, %v", r, err)
				}
			}
		})
	}
	if _, err := NewChannel(&bytes.Buffer{}, nil).Answer(); !errors.Is(err, io.EOF) {
		t.Errorf("an answer never spoken: %v", err)
	}
}

// TestServe_NamesItsProtocolInEveryAnswerToTheHandshake: accepting or
// refusing, the exalted servitor names its protocol, so that the invoker's
// servitor can tell a forging of its own from another's; one that speaks
// another protocol is refused with the death-mark 2.
func TestServe_NamesItsProtocolInEveryAnswerToTheHandshake(t *testing.T) {
	sh := newShrine(t, nil)
	inv := sh.rouseFor("on")
	_ = inv.ch.Send(sh.handshake("on"))
	if r, err := inv.ch.Answer(); err != nil || r.Verdict != Accepted {
		t.Fatalf("accepted: %+v, %v", r, err)
	}
	_ = inv.ch.Send(Request{Finish: new(true)})
	if m := inv.ended(); m != 0 {
		t.Fatalf("finished with the death-mark %d", m)
	}

	for what, edit := range map[string]func(h *Handshake){
		"a lapsed sigil":   func(h *Handshake) { h.Sigil = h.Sigil[:12] + strings.Repeat("0", 52) },
		"another protocol": func(h *Handshake) { h.Protocol = Protocol + 1 },
	} {
		t.Run(what, func(t *testing.T) {
			sh := newShrine(t, nil)
			inv := sh.rouse("hallow", sanction.Short(sh.sigil("on")), "on")
			h := sh.handshake("on")
			edit(&h)
			_ = inv.ch.Send(h)
			r, err := inv.ch.Answer()
			if err != nil || r.Verdict != Refused || r.Protocol != Protocol {
				t.Fatalf("heard %+v, %v", r, err)
			}
			if m := inv.ended(); m != 2 {
				t.Fatalf("refused with the death-mark %d", m)
			}
			sh.untouched()
		})
	}
}
