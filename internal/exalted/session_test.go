package exalted

import (
	"encoding/base64"
	"io/fs"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// performedOn is what the ground holds once every subject of "on" was
// performed.
var performedOn = map[string]string{
	"/etc/sysctl.d/90-hallow.conf": "644 kernel.hallow = on # " + secret + "\n",
	"/etc/hallow.conf": "600 " + original + "\n# +++ begin of sanctum hallow -- aspect|on +++\nmode = on " + secret +
		"\n# +++ end of sanctum hallow +++\n",
	"/etc/hallow-theme": "→ /usr/share/hallow/on",
}

func (sh *shrine) holds(want map[string]string) {
	sh.t.Helper()
	for path, w := range want {
		if got := sh.look(path); got != w {
			sh.t.Errorf("%s holds %q, want %q", path, got, w)
		}
	}
}

// TestSession_PerformsThePlanAndFinishes: accepted, the exalted servitor
// performs each exalted subject of the plan from the sanction, in written
// order, keeps the owner and seal of a standing vessel, keeps it all upon
// finish, and journals every act as the root's.
func TestSession_PerformsThePlanAndFinishes(t *testing.T) {
	sh := newShrine(t, nil)
	inv := sh.rouseFor("on")
	r := inv.say(sh.handshake("on"))
	if r.To != "handshake" || r.Verdict != Accepted || r.Protocol != Protocol || r.Verses != nil ||
		!slices.Equal(r.Subjects, []string{"conf", "lock", "theme"}) {
		t.Fatalf("accepted with %+v", r)
	}
	first := inv.perform(conf)
	if !first.Changed || first.Before != nil || first.After == nil || first.After.Size != len(performedOn["/etc/sysctl.d/90-hallow.conf"])-4 {
		t.Errorf("conf: %+v", first)
	}
	third := inv.perform(lock)
	if !third.Changed || third.Before == nil || third.Before.Size != len(original) || third.Before.SHA256 == third.After.SHA256 {
		t.Errorf("lock: %+v", third)
	}
	if fourth := inv.perform(theme); !fourth.Changed {
		t.Errorf("theme: %+v", fourth)
	}
	if r := inv.say(Request{Finish: new(true)}); r.To != "finish" || r.Verdict != Finished {
		t.Fatalf("finish: %+v", r)
	}
	if m := inv.ended(); m != 0 {
		t.Fatalf("finished with the death-mark %d", m)
	}
	sh.holds(performedOn)
	if want := []string{"accept accepted", "perform conf triumph", "perform lock triumph", "perform theme triumph",
		"finish finished"}; !slices.Equal(sh.journal.deeds(), want) {
		t.Errorf("journaled %v, want %v", sh.journal.deeds(), want)
	}
	for _, e := range sh.journal.all() {
		if e.UID != me || e.Rite != "hallow" || e.Aspect != "on" || e.Sigil != sanction.Short(sh.sigil("on")) {
			t.Errorf("journaled %+v", e)
		}
	}
}

// TestSession_ASubjectThatFindsItsWorkDoneChangesNothing: performed again,
// the plan touches nothing, and its subjects stand performed all the same.
func TestSession_ASubjectThatFindsItsWorkDoneChangesNothing(t *testing.T) {
	sh := newShrine(t, nil)
	inv := sh.accepted("on")
	for _, it := range []*Item{conf, lock, theme} {
		inv.perform(it)
	}
	inv.say(Request{Finish: new(true)})
	inv.ended()
	inv = sh.accepted("on")
	for _, it := range []*Item{conf, lock, theme} {
		if r := inv.perform(it); r.Changed {
			t.Errorf("%s changed what it found done: %+v", it.Named(), r)
		}
	}
	for _, it := range []*Item{theme, lock, conf} {
		if r := inv.say(Request{Revert: it}); r.Verdict != Reverted || r.Subject != it.Subject || r.Changed {
			t.Errorf("revert %s: %+v", it.Named(), r)
		}
	}
	inv.say(Request{Finish: new(true)})
	inv.ended()
	sh.holds(performedOn)
}

// TestSession_AStandingVesselTakesTheSealDecreed: an exalted transcription
// heeds its seal as every transcription does: a vessel that already holds
// its scripture takes the seal decreed, and reverting restores the former.
func TestSession_AStandingVesselTakesTheSealDecreed(t *testing.T) {
	u := hallowUnit()
	for i := range u.Aspects {
		u.Aspects[i].Subjects[0].Seal = new(fs.FileMode(0o600))
	}
	sh := newShrine(t, u)
	const vessel = "/etc/sysctl.d/90-hallow.conf"
	sh.scroll(vessel, string(u.Aspects[0].Subjects[0].Scripture), 0o644)
	inv := sh.accepted("on")
	if r := inv.perform(conf); !r.Changed {
		t.Errorf("the seal was not taken: %+v", r)
	}
	if got := sh.look(vessel); !strings.HasPrefix(got, "600 ") {
		t.Errorf("the vessel stands %q", got)
	}
	if r := inv.say(Request{Revert: conf}); r.Verdict != Reverted || !r.Changed {
		t.Errorf("revert: %+v", r)
	}
	if got := sh.look(vessel); !strings.HasPrefix(got, "644 ") {
		t.Errorf("the vessel stands %q after reverting", got)
	}
	inv.say(Request{Finish: new(true)})
	inv.ended()
}

// TestSession_TheEndOfTheChannelRevertsInReverse: when the invoker's
// servitor falls before finish, the exalted servitor reverts every subject it
// performed, from the last to the first, journals it, and ends non-zero.
func TestSession_TheEndOfTheChannelRevertsInReverse(t *testing.T) {
	sh := newShrine(t, nil)
	inv := sh.accepted("on")
	for _, it := range []*Item{conf, lock, theme} {
		inv.perform(it)
	}
	sh.holds(performedOn)
	must(t, inv.conn.Close())
	if m := inv.ended(); m != 1 {
		t.Fatalf("ended with the death-mark %d, want 1", m)
	}
	sh.untouched()
	want := []string{"accept accepted", "perform conf triumph", "perform lock triumph", "perform theme triumph",
		"revert theme reverted", "revert lock reverted", "revert conf reverted", "fall reverted"}
	if !slices.Equal(sh.journal.deeds(), want) {
		t.Errorf("journaled %v, want %v", sh.journal.deeds(), want)
	}
}

// TestSession_AbortRevertsInReverse: abort reverts every subject performed
// and tells each reversion; revert reverts the last one performed alone.
func TestSession_AbortRevertsInReverse(t *testing.T) {
	sh := newShrine(t, nil)
	inv := sh.accepted("on")
	for _, it := range []*Item{conf, lock, theme} {
		inv.perform(it)
	}
	if r := inv.say(Request{Revert: theme}); r.To != "revert" || r.Subject != "theme" || r.Verdict != Reverted ||
		!r.Changed {
		t.Fatalf("revert theme: %+v", r)
	}
	if got := sh.look("/etc/hallow-theme"); got != "absent" {
		t.Fatalf("the tether stands reverted as %q", got)
	}
	r := inv.say(Request{Abort: new(true)})
	if r.To != "abort" || r.Verdict != Reverted || len(r.Reversions) != 2 || r.Reversions[0].Subject != "lock" ||
		r.Reversions[1].Subject != "conf" {
		t.Fatalf("abort: %+v", r)
	}
	if m := inv.ended(); m != 1 {
		t.Fatalf("aborted with the death-mark %d", m)
	}
	sh.untouched()
}

// TestSession_RefusesWhatComesOutOfOrder: perform names an exalted subject
// of the plan that lies after every one performed, never a verse; revert
// names the last performed; nothing is performed once reverting began. A refused line ends the session, reverting what was
// performed.
func TestSession_RefusesWhatComesOutOfOrder(t *testing.T) {
	for what, c := range map[string]struct {
		before []Request
		wrong  Request
		to     string
	}{
		"a subject behind what was performed": {before: []Request{{Perform: lock}}, wrong: Request{Perform: conf},
			to: "perform"},
		"a subject performed twice":    {before: []Request{{Perform: conf}}, wrong: Request{Perform: conf}, to: "perform"},
		"a common subject":             {wrong: Request{Perform: subject("common")}, to: "perform"},
		"a verse of a rite of aspects": {wrong: Request{Perform: numbered(1)}, to: "perform"},
		"a revert of nothing":          {wrong: Request{Revert: conf}, to: "revert"},
		"a revert out of reverse":      {before: []Request{{Perform: conf}, {Perform: lock}}, wrong: Request{Revert: conf}, to: "revert"},
		"a perform after reverting":    {before: []Request{{Perform: conf}, {Revert: conf}}, wrong: Request{Perform: lock}, to: "perform"},
		"two words at once":            {wrong: Request{Perform: conf, Abort: new(true)}},
		"a finish that is false":       {wrong: Request{Finish: new(false)}},
		"no word at all":               {wrong: Request{}},
		"a second plan":                {before: []Request{{Perform: conf}}},
	} {
		t.Run(what, func(t *testing.T) {
			sh := newShrine(t, nil)
			inv := sh.accepted("on")
			for _, r := range c.before {
				if got := inv.say(r); got.Verdict != Triumph && got.Verdict != Reverted {
					t.Fatalf("%+v: %+v", r, got)
				}
			}
			var r Reply
			if what == "a second plan" {
				r = inv.say(sh.handshake("on"))
			} else {
				r = inv.say(c.wrong)
			}
			if r.Verdict != Refused || r.To != c.to || r.Lament == "" {
				t.Fatalf("heard %+v, want a refusal of %q", r, c.to)
			}
			if m := inv.ended(); m != 1 {
				t.Fatalf("ended with the death-mark %d, want 1", m)
			}
			sh.untouched()
		})
	}
}

// TestHandshake_IsCheckedAgainstArgvAndTheChannel: a handshake of another
// protocol, master, rite, aspect or sigil, of words too long or bearing a
// newline, an inscription the sanction does not declare, an overlong or
// unknown line, or no sanction to perform from, is refused with the
// death-mark 2, and nothing is touched.
func TestHandshake_IsCheckedAgainstArgvAndTheChannel(t *testing.T) {
	for what, c := range map[string]struct {
		edit  func(sh *shrine, h *Handshake)
		args  func(sh *shrine) []string
		line  string
		env   map[string]string // the environment the intercessor left
		mar   func(sh *shrine)
		words string
	}{
		"another protocol":             {edit: func(_ *shrine, h *Handshake) { h.Protocol = Protocol + 1 }, words: "install the servitor anew"},
		"another master":               {edit: func(_ *shrine, h *Handshake) { h.UID = me + 1 }, words: "uid"},
		"a SUDO_UID of another master": {env: map[string]string{"SUDO_UID": "4242"}, words: "SUDO_UID"},
		"a PKEXEC_UID of another master": {env: map[string]string{"SUDO_UID": strconv.Itoa(me), "PKEXEC_UID": "4242"},
			words: "PKEXEC_UID"},
		"another rite":   {edit: func(_ *shrine, h *Handshake) { h.Rite = "other" }, words: "rite"},
		"another aspect": {edit: func(_ *shrine, h *Handshake) { h.Aspect = "off" }, words: "aspect"},
		"the sigil of another aspect": {edit: func(sh *shrine, h *Handshake) { h.Sigil = sh.sigil("off") },
			words: "sigil"},
		"the sigil of the whole unit": {edit: func(sh *shrine, h *Handshake) { h.Sigil = sh.unit.Sigil() },
			words: "sigil"},
		"a sigil argv does not name": {args: func(sh *shrine) []string {
			return []string{"hallow", sanction.Short(sh.sigil("off")), "on"}
		}, words: "sigil"},
		"a lapsed sigil": {edit: func(_ *shrine, h *Handshake) { h.Sigil = h.Sigil[:12] + strings.Repeat("0", 52) },
			words: "lapsed"},
		"a former aspect bearing a newline": {edit: func(_ *shrine, h *Handshake) { h.Former = "off\nrm -rf /" },
			words: "newline"},
		"an inscription too long": {edit: func(_ *shrine, h *Handshake) {
			h.Inscriptions["who"] = strings.Repeat("x", wordCap+1)
		}, words: "exceed"},
		"an inscription bearing a NUL": {edit: func(_ *shrine, h *Handshake) { h.Inscriptions["who"] = "x\x00" },
			words: "NUL"},
		"an inscription not declared": {edit: func(_ *shrine, h *Handshake) { h.Inscriptions["whom"] = "x" },
			words: "whom"},
		"an unknown member": {line: `{"protocol":1,"blessing":true}`, words: "protocol"},
		"an overlong line":  {line: strings.Repeat(" ", LineCap+1), words: "exceeds"},
		"no sanction": {mar: func(sh *shrine) {
			must(sh.t, os.Remove(sanction.Store{Root: sh.cfg.Store, UID: me}.Path("hallow")))
		}, words: "unsanctioned"},
		"a sanction others may write": {mar: func(sh *shrine) {
			must(sh.t, os.Chmod(sanction.Store{Root: sh.cfg.Store, UID: me}.Path("hallow"), 0o666))
		}, words: "cannot be trusted"},
		"a store the root does not own": {mar: func(sh *shrine) {
			sh.cfg.Sovereign = func(uint32) bool { return false }
		}, words: "cannot be trusted"},
		"a sanction of pattern Mark I": {mar: func(sh *shrine) { sh.lay(sh.cfg.Store, elderUnit()) },
			words: `of the pattern "Mark I"`},
		"an edict's argv for a rite": {args: func(sh *shrine) []string {
			return []string{"hallow", sanction.Short(sh.unit.Sigil())}
		}, edit: func(sh *shrine, h *Handshake) { h.Aspect, h.Former, h.Sigil = "", "", sh.unit.Sigil() },
			words: "rite of aspects"},
	} {
		t.Run(what, func(t *testing.T) {
			sh := newShrine(t, nil)
			if c.env != nil {
				sh.cfg.Getenv = env(c.env)
			}
			if c.mar != nil {
				c.mar(sh)
			}
			args := []string{"hallow", sanction.Short(sh.sigil("on")), "on"}
			if c.args != nil {
				args = c.args(sh)
			}
			inv := sh.rouse(args...)
			var r Reply
			if c.line != "" {
				_, _ = inv.conn.Write([]byte(c.line + "\n"))
				r = inv.hear()
			} else {
				h := sh.handshake("on")
				if c.edit != nil {
					c.edit(sh, &h)
				}
				r = inv.say(h)
			}
			if r.To != "handshake" || r.Verdict != Refused || !strings.Contains(r.Lament, c.words) {
				t.Fatalf("heard %+v, want a refusal naming %q", r, c.words)
			}
			if m := inv.ended(); m != 2 {
				t.Fatalf("refused with the death-mark %d, want 2", m)
			}
			sh.untouched()
			if d := sh.journal.deeds(); !slices.Equal(d, []string{"accept refused"}) {
				t.Errorf("journaled %v", d)
			}
		})
	}
}

// TestSession_NoWordCarriesTheScriptureOfAVessel: the invoker hears
// verdicts, sizes and sums, never what a vessel holds, and neither does
// the journal.
func TestSession_NoWordCarriesTheScriptureOfAVessel(t *testing.T) {
	sh := newShrine(t, nil)
	inv := sh.accepted("on")
	for _, it := range []*Item{conf, lock, theme} {
		inv.perform(it)
	}
	inv.say(Request{Revert: theme})
	inv.say(Request{Abort: new(true)})
	inv.ended()
	var journal strings.Builder
	for _, e := range sh.journal.all() {
		journal.WriteString(e.String() + "\n")
	}
	for _, where := range []string{inv.heard.String(), journal.String()} {
		for _, word := range []string{secret, base64.StdEncoding.EncodeToString([]byte("kernel.hallow")),
			base64.StdEncoding.EncodeToString([]byte(original)), "base = ", "kernel.hallow", "mode = on"} {
			if strings.Contains(where, word) {
				t.Errorf("the scripture of a vessel (%q) left the exalted servitor:\n%s", word, where)
			}
		}
	}
	if !strings.Contains(inv.heard.String(), `"sha256":`) {
		t.Errorf("the invoker heard no sum of a vessel:\n%s", inv.heard)
	}
}

// TestSession_ASubjectFallsThroughTheTrustedPath: the vessels are acted
// upon through the root-safe acts: a hall others may write fells the
// subject, touching nothing, and a fallen subject that changed nothing is
// not reverted.
func TestSession_ASubjectFallsThroughTheTrustedPath(t *testing.T) {
	sh := newShrine(t, nil)
	must(t, os.Chmod(sh.at("/etc/sysctl.d"), 0o777))
	inv := sh.accepted("on")
	r := inv.say(Request{Perform: conf})
	if r.Verdict != Fell || r.Changed || !strings.Contains(r.Lament, "others than its owner may write") {
		t.Fatalf("conf: %+v", r)
	}
	if r := inv.say(Request{Perform: lock}); r.Verdict != Refused {
		t.Fatalf("a subject was performed after one fell: %+v", r)
	}
	if m := inv.ended(); m != 1 {
		t.Fatalf("ended with the death-mark %d", m)
	}
	must(t, os.Chmod(sh.at("/etc/sysctl.d"), 0o755))
	sh.untouched()
}

// TestSession_AVesselOfNoAspectGivesWayOnlyWithZeal: as in a common
// transcription, a standing vessel holding scripture of no aspect of the
// sanction is never replaced without zeal.
func TestSession_AVesselOfNoAspectGivesWayOnlyWithZeal(t *testing.T) {
	sh := newShrine(t, nil)
	sh.scroll("/etc/sysctl.d/90-hallow.conf", "kernel.hallow = mine\n", 0o640)
	inv := sh.accepted("on")
	if r := inv.say(Request{Perform: conf}); r.Verdict != Fell || !strings.Contains(r.Lament, "zeal") {
		t.Fatalf("conf: %+v", r)
	}
	inv.say(Request{Abort: new(true)})
	inv.ended()
	if got := sh.look("/etc/sysctl.d/90-hallow.conf"); got != "640 kernel.hallow = mine\n" {
		t.Fatalf("the vessel holds %q", got)
	}

	sh.scroll("/etc/sysctl.d/90-hallow.conf", "kernel.hallow = off # "+secret+"\n", 0o640)
	inv = sh.accepted("on")
	inv.perform(conf)
	inv.say(Request{Finish: new(true)})
	inv.ended()
	if got := sh.look("/etc/sysctl.d/90-hallow.conf"); got != "640 kernel.hallow = on # "+secret+"\n" {
		t.Fatalf("the vessel of another aspect holds %q, keeping its seal", got)
	}
}

// TestSession_SilenceAndHaltRevert: ten minutes without a word (here less)
// and SIGTERM (Halt) revert every subject performed.
func TestSession_SilenceAndHaltRevert(t *testing.T) {
	t.Run("silence", func(t *testing.T) {
		sh := newShrine(t, nil)
		sh.cfg.Idle = 200 * time.Millisecond
		inv := sh.accepted("on")
		inv.perform(conf)
		if m := inv.ended(); m != 1 {
			t.Fatalf("ended with the death-mark %d", m)
		}
		sh.untouched()
	})
	t.Run("halt", func(t *testing.T) {
		sh := newShrine(t, nil)
		halt := make(chan struct{})
		sh.cfg.Halt = halt
		inv := sh.accepted("on")
		inv.perform(conf)
		inv.perform(lock)
		close(halt)
		r := inv.hear()
		if r.To != "abort" || r.Verdict != Reverted || len(r.Reversions) != 2 {
			t.Fatalf("halted with %+v", r)
		}
		if m := inv.ended(); m != 1 {
			t.Fatalf("ended with the death-mark %d", m)
		}
		sh.untouched()
	})
}
