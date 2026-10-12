package exalted

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/nerdwave-nick/servitor/internal/dominion"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// TestServe_RefusesUnlessExalted: an exalted servitor that does not bear
// the machine's authority refuses before it hears anything.
func TestServe_RefusesUnlessExalted(t *testing.T) {
	sh := newShrine(t, nil)
	sh.cfg.Exalted = func() bool { return false }
	inv := sh.rouseFor("on")
	if r := inv.hear(); r.Verdict != Refused || !strings.Contains(r.Lament, "machine's own authority") {
		t.Fatalf("heard %+v", r)
	}
	if m := inv.ended(); m != 2 {
		t.Fatalf("refused with the death-mark %d", m)
	}
	sh.untouched()
}

// TestServe_RefusesAMalformedArgv: argv names a rite, twelve letters of a
// sigil and an aspect (none for an edict), and nothing else.
func TestServe_RefusesAMalformedArgv(t *testing.T) {
	sh := newShrine(t, nil)
	short := sanction.Short(sh.sigil("on"))
	for _, args := range [][]string{
		{}, {"hallow"}, {"hallow", short, "on", "more"}, {"../hallow", short, "on"}, {"hallow", "ABCDEF012345", "on"},
		{"hallow", short + "0", "on"}, {"hallow", short, "../on"}, {"hallow", short, ""},
	} {
		inv := sh.rouse(args...)
		if r := inv.hear(); r.Verdict != Refused || r.To != "handshake" {
			t.Errorf("%q: heard %+v", args, r)
		}
		if m := inv.ended(); m != 2 {
			t.Errorf("%q: refused with the death-mark %d", args, m)
		}
	}
	sh.untouched()
}

// TestIdentity_PipesFallBackToSudoUID: when sudo lays pipes between the
// servitors, SUDO_UID names the master; without it the master is unknown
// and nothing is accepted.
func TestIdentity_PipesFallBackToSudoUID(t *testing.T) {
	for what, c := range map[string]struct {
		env      map[string]string
		accepted bool
		words    string
	}{
		"SUDO_UID":            {env: map[string]string{"SUDO_UID": strconv.Itoa(me)}, accepted: true},
		"no SUDO_UID":         {env: map[string]string{"PKEXEC_UID": strconv.Itoa(me)}, words: "no SUDO_UID"},
		"a SUDO_UID of words": {env: map[string]string{"SUDO_UID": "me"}, words: "names no uid"},
	} {
		t.Run(what, func(t *testing.T) {
			sh := newShrine(t, nil)
			sh.cfg.Getenv = env(c.env)
			theirsIn, oursOut, err := os.Pipe()
			must(t, err)
			oursIn, theirsOut, err := os.Pipe()
			must(t, err)
			t.Cleanup(func() { _ = oursOut.Close() })
			inv := sh.rouseOn(oursIn, theirsIn, theirsOut, []string{"hallow", sanction.Short(sh.sigil("on")), "on"})
			inv.ch = NewChannel(oursIn, oursOut)
			r := inv.say(sh.handshake("on"))
			if (r.Verdict == Accepted) != c.accepted || !strings.Contains(r.Lament, c.words) {
				t.Fatalf("heard %+v", r)
			}
			if c.accepted {
				inv.perform(1)
				inv.say(Request{Finish: new(true)})
			}
			want := map[bool]int{true: 0, false: 2}[c.accepted]
			if m := inv.ended(); m != want {
				t.Fatalf("ended with the death-mark %d, want %d", m, want)
			}
		})
	}
}

// TestIdentity_TheRootIsNeverTheMaster: a channel whose master is the root
// itself is refused.
func TestIdentity_TheRootIsNeverTheMaster(t *testing.T) {
	r, w, err := os.Pipe()
	must(t, err)
	defer func() { _ = r.Close(); _ = w.Close() }()
	if _, err := identify(r, env(map[string]string{"SUDO_UID": "0"})); err == nil {
		t.Fatal("the root was accepted as the master")
	}
}

// TestSession_SpeechAwaitsItsVoice: a plan holding exalted speech is
// refused before anything is touched while the exalted servitor has no
// voice; a voice lent to it speaks the verse in its turn, reverts it in
// reverse and is closed when the session ends.
func TestSession_SpeechAwaitsItsVoice(t *testing.T) {
	speaking := func() *sanction.Unit {
		u := hallowUnit()
		for i := range u.Aspects {
			p := &u.Aspects[i]
			p.Liturgy = slices.Insert(p.Liturgy, 1, sanction.Verse{Verse: 2, Kind: librarium.KindIncantation,
				Target: "sysctl -wq kernel.hallow=1", Tongue: "/usr/bin/bash", Patience: time.Second,
				Reversion: "sysctl -wq kernel.hallow=0", Relics: []sanction.Relic{}})
		}
		return u
	}
	sh := newShrine(t, speaking())
	inv := sh.rouseFor("on")
	if r := inv.say(sh.handshake("on")); r.Verdict != Refused || !strings.Contains(r.Lament, "verse 2") {
		t.Fatalf("heard %+v", r)
	}
	if m := inv.ended(); m != 2 {
		t.Fatalf("refused with the death-mark %d", m)
	}
	sh.untouched()

	sh = newShrine(t, speaking())
	v := &voice{}
	sh.cfg.Voice = v.lend
	inv = sh.accepted("on")
	inv.perform(1)
	if r := inv.perform(2); r.Heard != "spoke 2" || !r.Changed {
		t.Fatalf("verse 2: %+v", r)
	}
	inv.perform(3)
	must(t, inv.conn.Close())
	if m := inv.ended(); m != 1 {
		t.Fatalf("ended with the death-mark %d", m)
	}
	sh.untouched()
	if want := []string{"sitting on off x; echo INJECTED", "speak 2", "undo 2", "close"}; !slices.Equal(v.said(), want) {
		t.Errorf("the voice heard %v, want %v", v.said(), want)
	}
}

// voice is a voice of the trials, which records what it is bidden.
type voice struct {
	mu   sync.Mutex
	acts []string
}

func (v *voice) record(act string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.acts = append(v.acts, act)
}

func (v *voice) said() []string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return slices.Clone(v.acts)
}

func (v *voice) lend(s Sitting) (Speaker, error) {
	v.record("sitting " + s.Plan.Aspect + " " + s.Former + " " + s.Inscriptions["who"])
	return v, nil
}

func (v *voice) Speak(_ context.Context, verse sanction.Verse) (string, Undo, error) {
	n := strconv.Itoa(verse.Verse)
	v.record("speak " + n)
	return "spoke " + n, func(context.Context) (string, error) { v.record("undo " + n); return "", nil }, nil
}

func (v *voice) Close() error { v.record("close"); return nil }

// TestSession_HoldsTheRootSideDominion: the session is accepted only once
// the exalted servitor holds dominion over the rite of its master and over
// every vessel and bound name of the plan, waiting while another holds it;
// a dominion that cannot be held refuses the session before verse 1.
func TestSession_HoldsTheRootSideDominion(t *testing.T) {
	sh := newShrine(t, nil)
	rite := filepath.Join(sh.cfg.Dominions, "rites", strconv.Itoa(me)+"-hallow")
	held, err := dominion.Seize(context.Background(), []string{rite}, nil)
	must(t, err)
	inv := sh.rouseFor("on")
	must(t, inv.ch.Send(sh.handshake("on")))
	replied := make(chan Reply, 1)
	go func() {
		var r Reply
		_ = inv.ch.Receive(&r)
		replied <- r
	}()
	select {
	case r := <-replied:
		t.Fatalf("accepted while another held the dominion: %+v", r)
	case <-time.After(100 * time.Millisecond):
	}
	held.Relinquish()
	if r := <-replied; r.Verdict != Accepted {
		t.Fatalf("heard %+v", r)
	}
	vessels, err := os.ReadDir(filepath.Join(sh.cfg.Dominions, "vessels"))
	must(t, err)
	if len(vessels) != 3 {
		t.Errorf("dominion is held over %d vessels, want 3", len(vessels))
	}
	probe, stop := context.WithCancel(context.Background())
	busy, err := dominion.Seize(probe, []string{rite}, stop)
	stop()
	if err == nil {
		busy.Relinquish()
		t.Error("the rite's dominion was seized beside the session")
	}
	inv.say(Request{Finish: new(true)})
	inv.ended()

	stone := filepath.Join(t.TempDir(), "stone")
	must(t, os.WriteFile(stone, nil, 0o644))
	sh.cfg.Dominions = stone
	inv = sh.rouseFor("on")
	if r := inv.say(sh.handshake("on")); r.Verdict != Refused || !strings.Contains(r.Lament, "dominion") {
		t.Fatalf("heard %+v", r)
	}
	if m := inv.ended(); m != 2 {
		t.Fatalf("refused with the death-mark %d", m)
	}
	sh.untouched()
}

// TestSession_AwaitingDominionEndsWithTheChannel: an invoker that falls
// while the exalted servitor awaits its dominion leaves it waiting no more.
func TestSession_AwaitingDominionEndsWithTheChannel(t *testing.T) {
	sh := newShrine(t, nil)
	rite := filepath.Join(sh.cfg.Dominions, "rites", strconv.Itoa(me)+"-hallow")
	held, err := dominion.Seize(context.Background(), []string{rite}, nil)
	must(t, err)
	defer held.Relinquish()
	inv := sh.rouseFor("on")
	must(t, inv.ch.Send(sh.handshake("on")))
	inv.heeded()
	must(t, inv.conn.Close())
	if m := inv.ended(); m != 2 {
		t.Fatalf("ended with the death-mark %d", m)
	}
	if d := sh.journal.all(); len(d) != 1 || !strings.Contains(d[0].Lament, "dominion") {
		t.Errorf("journaled %+v", d)
	}
	sh.untouched()
}

// TestJournal_TellsEachActOnOneLine: the journal names uid, rite, aspect,
// verse, sigil and verdict of every act.
func TestJournal_TellsEachActOnOneLine(t *testing.T) {
	e := Entry{UID: 1000, Rite: "split-lock", Aspect: "on", Verse: 3, Sigil: "0123456789ab", Deed: "perform",
		Verdict: Fell, Lament: "the vessel /etc/x\nfell"}
	want := `uid=1000 rite="split-lock" aspect="on" verse=3 sigil=0123456789ab deed=perform verdict=fell ` +
		`lament="the vessel /etc/x\nfell"`
	if e.String() != want {
		t.Errorf("journaled %s\nwant      %s", e.String(), want)
	}
	if got := (Entry{UID: 1000, Rite: "shot", Sigil: "0123456789ab", Deed: "finish", Verdict: Finished}).String(); got != `uid=1000 rite="shot" sigil=0123456789ab deed=finish verdict=finished` {
		t.Errorf("journaled %s", got)
	}
}

// heeded waits until the exalted servitor has heard every byte spoken to
// it.
func (inv *invoker) heeded() {
	inv.t.Helper()
	deadline := time.Now().Add(patience)
	for time.Now().Before(deadline) {
		unheard, err := unix.IoctlGetInt(int(inv.conn.Fd()), unix.SIOCOUTQ)
		must(inv.t, err)
		if unheard == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	inv.t.Fatal("the exalted servitor never heard what was spoken")
}
