package exalted

import (
	"encoding/base64"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// performedOn is what the ground holds once every verse of "on" was
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
// performs each exalted verse of the plan from the sanction, in written
// order, keeps the owner and seal of a standing vessel, keeps it all upon
// finish, and journals every act as the root's.
func TestSession_PerformsThePlanAndFinishes(t *testing.T) {
	sh := newShrine(t, nil)
	inv := sh.rouseFor("on")
	r := inv.say(sh.handshake("on"))
	if r.To != "handshake" || r.Verdict != Accepted || r.Protocol != Protocol || !slices.Equal(r.Verses, []int{1, 3, 4}) {
		t.Fatalf("accepted with %+v", r)
	}
	first := inv.perform(1)
	if !first.Changed || first.Before != nil || first.After == nil || first.After.Size != len(performedOn["/etc/sysctl.d/90-hallow.conf"])-4 {
		t.Errorf("verse 1: %+v", first)
	}
	third := inv.perform(3)
	if !third.Changed || third.Before == nil || third.Before.Size != len(original) || third.Before.SHA256 == third.After.SHA256 {
		t.Errorf("verse 3: %+v", third)
	}
	if fourth := inv.perform(4); !fourth.Changed {
		t.Errorf("verse 4: %+v", fourth)
	}
	if r := inv.say(Request{Finish: new(true)}); r.To != "finish" || r.Verdict != Finished {
		t.Fatalf("finish: %+v", r)
	}
	if m := inv.ended(); m != 0 {
		t.Fatalf("finished with the death-mark %d", m)
	}
	sh.holds(performedOn)
	if want := []string{"accept accepted", "perform 1 triumph", "perform 3 triumph", "perform 4 triumph",
		"finish finished"}; !slices.Equal(sh.journal.deeds(), want) {
		t.Errorf("journaled %v, want %v", sh.journal.deeds(), want)
	}
	for _, e := range sh.journal.all() {
		if e.UID != me || e.Rite != "hallow" || e.Aspect != "on" || e.Sigil != sanction.Short(sh.sigil("on")) {
			t.Errorf("journaled %+v", e)
		}
	}
}

// TestSession_AVerseThatFindsItsWorkDoneChangesNothing: performed again,
// the plan touches nothing, and its verses stand performed all the same.
func TestSession_AVerseThatFindsItsWorkDoneChangesNothing(t *testing.T) {
	sh := newShrine(t, nil)
	inv := sh.accepted("on")
	for _, n := range []int{1, 3, 4} {
		inv.perform(n)
	}
	inv.say(Request{Finish: new(true)})
	inv.ended()
	inv = sh.accepted("on")
	for _, n := range []int{1, 3, 4} {
		if r := inv.perform(n); r.Changed {
			t.Errorf("verse %d changed what it found done: %+v", n, r)
		}
	}
	for _, n := range []int{4, 3, 1} {
		if r := inv.say(Request{Revert: new(n)}); r.Verdict != Reverted || r.Verse != n || r.Changed {
			t.Errorf("revert %d: %+v", n, r)
		}
	}
	inv.say(Request{Finish: new(true)})
	inv.ended()
	sh.holds(performedOn)
}

// TestSession_TheEndOfTheChannelRevertsInReverse: when the invoker's
// servitor falls before finish, the exalted servitor reverts every verse it
// performed, from the last to the first, journals it, and ends non-zero.
func TestSession_TheEndOfTheChannelRevertsInReverse(t *testing.T) {
	sh := newShrine(t, nil)
	inv := sh.accepted("on")
	for _, n := range []int{1, 3, 4} {
		inv.perform(n)
	}
	sh.holds(performedOn)
	must(t, inv.conn.Close())
	if m := inv.ended(); m != 1 {
		t.Fatalf("ended with the death-mark %d, want 1", m)
	}
	sh.untouched()
	want := []string{"accept accepted", "perform 1 triumph", "perform 3 triumph", "perform 4 triumph",
		"revert 4 reverted", "revert 3 reverted", "revert 1 reverted", "fall reverted"}
	if !slices.Equal(sh.journal.deeds(), want) {
		t.Errorf("journaled %v, want %v", sh.journal.deeds(), want)
	}
}

// TestSession_AbortRevertsInReverse: abort reverts every verse performed
// and tells each reversion; revert reverts the last verse performed alone.
func TestSession_AbortRevertsInReverse(t *testing.T) {
	sh := newShrine(t, nil)
	inv := sh.accepted("on")
	for _, n := range []int{1, 3, 4} {
		inv.perform(n)
	}
	if r := inv.say(Request{Revert: new(4)}); r.To != "revert" || r.Verse != 4 || r.Verdict != Reverted || !r.Changed {
		t.Fatalf("revert 4: %+v", r)
	}
	if got := sh.look("/etc/hallow-theme"); got != "absent" {
		t.Fatalf("the tether stands reverted as %q", got)
	}
	r := inv.say(Request{Abort: new(true)})
	if r.To != "abort" || r.Verdict != Reverted || len(r.Reversions) != 2 || r.Reversions[0].Verse != 3 ||
		r.Reversions[1].Verse != 1 {
		t.Fatalf("abort: %+v", r)
	}
	if m := inv.ended(); m != 1 {
		t.Fatalf("aborted with the death-mark %d", m)
	}
	sh.untouched()
}

// TestSession_RefusesWhatComesOutOfOrder: perform names strictly the next
// exalted verse, revert the last performed; nothing is performed once
// reverting began. A refused line ends the session, reverting what was
// performed.
func TestSession_RefusesWhatComesOutOfOrder(t *testing.T) {
	for what, c := range map[string]struct {
		before []Request
		wrong  Request
		to     string
	}{
		"a verse before its turn":   {wrong: Request{Perform: new(3)}, to: "perform"},
		"a verse skipped":           {before: []Request{{Perform: new(1)}}, wrong: Request{Perform: new(4)}, to: "perform"},
		"a verse performed twice":   {before: []Request{{Perform: new(1)}}, wrong: Request{Perform: new(1)}, to: "perform"},
		"a common verse":            {wrong: Request{Perform: new(2)}, to: "perform"},
		"a verse beyond the plan":   {before: []Request{{Perform: new(1)}, {Perform: new(3)}, {Perform: new(4)}}, wrong: Request{Perform: new(5)}, to: "perform"},
		"a revert of no verse":      {wrong: Request{Revert: new(1)}, to: "revert"},
		"a revert out of reverse":   {before: []Request{{Perform: new(1)}, {Perform: new(3)}}, wrong: Request{Revert: new(1)}, to: "revert"},
		"a perform after reverting": {before: []Request{{Perform: new(1)}, {Revert: new(1)}}, wrong: Request{Perform: new(3)}, to: "perform"},
		"two words at once":         {wrong: Request{Perform: new(1), Abort: new(true)}},
		"a finish that is false":    {wrong: Request{Finish: new(false)}},
		"no word at all":            {wrong: Request{}},
		"a second plan":             {before: []Request{{Perform: new(1)}}},
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
	for _, n := range []int{1, 3, 4} {
		inv.perform(n)
	}
	inv.say(Request{Revert: new(4)})
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

// TestSession_AVerseFallsThroughTheTrustedPath: the vessels are acted upon
// through the root-safe acts: a hall others may write fells the verse,
// touching nothing, and a fallen verse that changed nothing is not
// reverted.
func TestSession_AVerseFallsThroughTheTrustedPath(t *testing.T) {
	sh := newShrine(t, nil)
	must(t, os.Chmod(sh.at("/etc/sysctl.d"), 0o777))
	inv := sh.accepted("on")
	r := inv.say(Request{Perform: new(1)})
	if r.Verdict != Fell || r.Changed || !strings.Contains(r.Lament, "others than its owner may write") {
		t.Fatalf("verse 1: %+v", r)
	}
	if r := inv.say(Request{Perform: new(3)}); r.Verdict != Refused {
		t.Fatalf("a verse was performed after one fell: %+v", r)
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
	if r := inv.say(Request{Perform: new(1)}); r.Verdict != Fell || !strings.Contains(r.Lament, "zeal") {
		t.Fatalf("verse 1: %+v", r)
	}
	inv.say(Request{Abort: new(true)})
	inv.ended()
	if got := sh.look("/etc/sysctl.d/90-hallow.conf"); got != "640 kernel.hallow = mine\n" {
		t.Fatalf("the vessel holds %q", got)
	}

	sh.scroll("/etc/sysctl.d/90-hallow.conf", "kernel.hallow = off # "+secret+"\n", 0o640)
	inv = sh.accepted("on")
	inv.perform(1)
	inv.say(Request{Finish: new(true)})
	inv.ended()
	if got := sh.look("/etc/sysctl.d/90-hallow.conf"); got != "640 kernel.hallow = on # "+secret+"\n" {
		t.Fatalf("the vessel of another aspect holds %q, keeping its seal", got)
	}
}

// TestSession_SilenceAndHaltRevert: ten minutes without a word (here less)
// and SIGTERM (Halt) revert every verse performed.
func TestSession_SilenceAndHaltRevert(t *testing.T) {
	t.Run("silence", func(t *testing.T) {
		sh := newShrine(t, nil)
		sh.cfg.Idle = 200 * time.Millisecond
		inv := sh.accepted("on")
		inv.perform(1)
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
		inv.perform(1)
		inv.perform(3)
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
