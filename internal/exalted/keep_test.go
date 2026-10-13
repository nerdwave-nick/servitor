package exalted

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// sanctionedNow is the hour the exalted servitor of these trials lays a
// sanction.
var sanctionedNow = time.Date(2026, 10, 12, 21, 0, 0, 0, time.UTC)

// keepingShrine is the world of a trial of the store: its store lies
// beneath a hall of the trial's own standing in for /var/lib, never raised
// yet, and the master's group is kin.
func keepingShrine(t *testing.T) (*shrine, int) {
	t.Helper()
	sh := newShrine(t, nil)
	lib := filepath.Join(t.TempDir(), "var", "lib")
	must(t, os.MkdirAll(lib, 0o755))
	sh.cfg.Store = filepath.Join(lib, "servitor", "sanctions")
	kin := os.Getgid()
	if groups, _ := os.Getgroups(); len(groups) > 0 {
		for _, g := range groups {
			if g != os.Getgid() {
				kin = g
			}
		}
	}
	sh.cfg.Kin = func(uid int) (int, error) {
		if uid != me {
			return -1, errors.New("a stranger")
		}
		return kin, nil
	}
	sh.cfg.Now = func() time.Time { return sanctionedNow }
	return sh, kin
}

func (sh *shrine) store() sanction.Store { return sanction.Store{Root: sh.cfg.Store, UID: me} }

// sanctionFor rouses an exalted servitor to sanction the rite of u under
// the sigil argv names (the unit's own when ""), and speaks the words of u.
func (sh *shrine) sanctionFor(u *sanction.Unit, argvSigil string) *invoker {
	sh.t.Helper()
	if argvSigil == "" {
		argvSigil = sanction.Short(u.Sigil())
	}
	inv := sh.rouse(SanctionWord, u.Rite, argvSigil)
	go func() { _ = inv.ch.Offer(me, u) }()
	return inv
}

// answer hears the answer to the first line as the master's servitor does.
func (inv *invoker) answer() Reply {
	inv.t.Helper()
	r, err := inv.ch.Answer()
	if err != nil {
		inv.t.Fatalf("no answer was heard: %v", err)
	}
	return r
}

// sealOf tells the owner's group and seal of path.
func sealOf(t *testing.T, path string) (int, fs.FileMode) {
	t.Helper()
	fi, err := os.Lstat(path)
	must(t, err)
	return int(fi.Sys().(*syscall.Stat_t).Gid), fi.Mode().Perm()
}

// TestSanction_LaysTheUnitOfTheMaster: roused to sanction, the exalted
// servitor hears the unit in parts, checks its sigil against argv, lays it
// as the master's sanction — raising the store on first use, the master's
// hall and sanction of the master's group — and journals uid, rite, sigil
// and the sigil it replaced.
func TestSanction_LaysTheUnitOfTheMaster(t *testing.T) {
	sh, kin := keepingShrine(t)
	u := hallowUnit()
	// a scripture longer than one part
	u.Aspects[0].Subjects[0].Scripture = bytes.Repeat([]byte("kernel.hallow = on\n"), PartCap/10)
	inv := sh.sanctionFor(u, "")
	if r := inv.answer(); r.To != "sanction" || r.Verdict != Sanctioned || r.Protocol != Protocol {
		t.Fatalf("answered %+v", r)
	}
	if m := inv.ended(); m != 0 {
		t.Fatalf("sanctioned with the death-mark %d", m)
	}
	kept, err := sh.store().ReadHallowed("hallow", func(uid uint32) bool { return uid == uint32(me) })
	if err != nil || kept == nil || kept.Sigil != u.Sigil() || !kept.SanctionedAt.Equal(sanctionedNow) {
		t.Fatalf("the store keeps %+v, %v", kept, err)
	}
	for path, want := range map[string][2]int{
		filepath.Dir(sh.cfg.Store):        {os.Getgid(), 0o755},
		sh.cfg.Store:                      {os.Getgid(), 0o755},
		filepath.Dir(sh.store().Path("")): {kin, 0o750},
		sh.store().Path("hallow"):         {kin, 0o640},
	} {
		if gid, seal := sealOf(t, path); gid != want[0] || int(seal) != want[1] {
			t.Errorf("%s is of group %d sealed %04o, want %d %04o", path, gid, seal, want[0], want[1])
		}
	}
	short := sanction.Short(u.Sigil())
	if got, want := sh.journal.all(), []Entry{{UID: me, Rite: "hallow", Sigil: short, Former: "none",
		Deed: "sanction", Verdict: Sanctioned}}; !slices.Equal(got, want) {
		t.Errorf("journaled %+v\nwant      %+v", got, want)
	}

	// sanctioned anew, it names the sigil it replaced
	u.Aspects[1].Subjects[1].Scripture = []byte("mode = off, anew")
	inv = sh.sanctionFor(u, "")
	if r := inv.answer(); r.Verdict != Sanctioned {
		t.Fatalf("answered %+v", r)
	}
	inv.ended()
	if entries := sh.journal.all(); len(entries) != 2 || entries[1].Former != short ||
		entries[1].Sigil != sanction.Short(u.Sigil()) {
		t.Errorf("journaled %+v", entries)
	}
	if line := sh.journal.all()[1].String(); !strings.Contains(line, "deed=sanction verdict=sanctioned") ||
		!strings.Contains(line, "former="+short) {
		t.Errorf("the journal line is %s", line)
	}
	if kept, _ := sh.store().Read("hallow"); kept == nil || kept.Sigil != u.Sigil() {
		t.Errorf("the store keeps %+v", kept)
	}
}

// TestSanction_RefusesWhatItWasNotBidden: the exalted servitor lays only the
// unit whose sigil argv names, of the master at the other end of the
// channel and of the rite argv names, a unit it could perform by the rules
// of every invocation; whatever else it hears is refused with the death-mark
// 2, and nothing is written.
func TestSanction_RefusesWhatItWasNotBidden(t *testing.T) {
	for what, c := range map[string]struct {
		argv   func(u *sanction.Unit) []string
		speak  func(inv *invoker, u *sanction.Unit)
		mar    func(sh *shrine, u *sanction.Unit)
		lament string
	}{
		"another sigil in argv": {argv: func(u *sanction.Unit) []string {
			return []string{SanctionWord, "hallow", "0123456789ab"}
		}, lament: "sigil"},
		"another rite in argv": {argv: func(u *sanction.Unit) []string {
			return []string{SanctionWord, "other", sanction.Short(u.Sigil())}
		}, lament: "rite"},
		"a malformed argv": {argv: func(u *sanction.Unit) []string { return []string{SanctionWord, "hallow"} },
			lament: SanctionWord},
		"another master in the words": {speak: func(inv *invoker, u *sanction.Unit) {
			_ = inv.ch.Send(Sanctioning{Protocol: Protocol, UID: me + 1, Rite: u.Rite, Sigil: u.Sigil(), Size: 2})
		}, lament: "uid"},
		"another protocol": {speak: func(inv *invoker, u *sanction.Unit) {
			_ = inv.ch.Send(Sanctioning{Protocol: Protocol + 1, UID: me, Rite: u.Rite, Sigil: u.Sigil(), Size: 2})
		}, lament: "install the servitor anew"},
		"a unit unlike its sigil": {speak: func(inv *invoker, u *sanction.Unit) {
			data := u.Canonical()
			_ = inv.ch.Send(Sanctioning{Protocol: Protocol, UID: me, Rite: u.Rite, Sigil: u.Sigil(), Size: len(data)})
			data = bytes.Replace(data, []byte(`"/etc/hallow.conf"`), []byte(`"/etc/hallox.conf"`), 1)
			_ = inv.ch.Send(Part{Part: data})
		}, lament: "sigil"},
		"more than it was told": {speak: func(inv *invoker, u *sanction.Unit) {
			data := u.Canonical()
			_ = inv.ch.Send(Sanctioning{Protocol: Protocol, UID: me, Rite: u.Rite, Sigil: u.Sigil(), Size: 10})
			_ = inv.ch.Send(Part{Part: data})
		}, lament: "bytes"},
		"a unit too great": {speak: func(inv *invoker, u *sanction.Unit) {
			_ = inv.ch.Send(Sanctioning{Protocol: Protocol, UID: me, Rite: u.Rite, Sigil: u.Sigil(), Size: UnitCap + 1})
		}, lament: "bytes"},
		"a member unknown": {speak: func(inv *invoker, u *sanction.Unit) {
			_, _ = inv.conn.Write([]byte(`{"protocol":1,"uid":1,"rite":"hallow","sigil":"x","size":1,"xenos":1}` + "\n"))
		}, lament: "xenos"},
		"the unit of another master": {mar: func(_ *shrine, u *sanction.Unit) { u.UID = me + 1 }, lament: "uid"},
		"a vessel beneath /proc": {mar: func(_ *shrine, u *sanction.Unit) {
			u.Aspects[1].Subjects[0].Target = "/proc/sys/kernel/hallow"
		}, lament: "/proc"},
		"a vessel not from the root": {mar: func(_ *shrine, u *sanction.Unit) {
			u.Aspects[0].Subjects[2].Target = "etc/hallow-theme"
		}, lament: "root"},
		"plans without subjects": {mar: func(_ *shrine, u *sanction.Unit) {
			for i := range u.Aspects {
				u.Aspects[i].Subjects = nil
			}
		}, lament: "no exalted subject or observance"},
		"a unit of pattern Mark I": {mar: func(_ *shrine, u *sanction.Unit) { *u = *elderUnit() },
			lament: `only units of the pattern "Mark II"`},
		"an edict of an aspect": {mar: func(_ *shrine, u *sanction.Unit) { u.Edict = true; u.Aspects = u.Aspects[:1] },
			lament: "edict"},
		"two plans of one aspect": {mar: func(_ *shrine, u *sanction.Unit) { u.Aspects[1].Aspect = "on" },
			lament: `"on"`},
		"another pattern": {mar: func(_ *shrine, u *sanction.Unit) { u.Pattern = "Mark III" }, lament: "Mark III"},
		"speech it cannot speak": {mar: func(sh *shrine, u *sanction.Unit) {
			for i := range u.Aspects {
				u.Aspects[i].Observances = append(u.Aspects[i].Observances, sanction.Verse{Observance: "chant",
					Kind: librarium.KindIncantation, Target: "echo", Tongue: "/usr/bin/bash", Patience: time.Second,
					Introit: []int{2}, Recessional: []int{}})
			}
			sh.cfg.Voice = func(Sitting) (Speaker, error) { return nil, errors.New("never") }
			sh.cfg.Weigh = func(Sitting) error { return errors.New("the tongue of chant is no program of the root") }
		}, lament: "chant"},
		"a master of no group": {mar: func(sh *shrine, _ *sanction.Unit) {
			sh.cfg.Kin = func(int) (int, error) { return -1, errors.New("no such master") }
		}, lament: "no such master"},
		"no authority": {mar: func(sh *shrine, _ *sanction.Unit) { sh.cfg.Exalted = func() bool { return false } },
			lament: "machine's own authority"},
	} {
		t.Run(what, func(t *testing.T) {
			sh, _ := keepingShrine(t)
			u := hallowUnit()
			if c.mar != nil {
				c.mar(sh, u)
			}
			args := []string{SanctionWord, u.Rite, sanction.Short(u.Sigil())}
			if c.argv != nil {
				args = c.argv(u)
			}
			inv := sh.rouse(args...)
			speak := c.speak
			if speak == nil {
				speak = func(inv *invoker, u *sanction.Unit) { _ = inv.ch.Offer(me, u) }
			}
			go speak(inv, u)
			r := inv.answer()
			if r.Verdict != Refused || r.Protocol != Protocol || !strings.Contains(r.Lament, c.lament) {
				t.Fatalf("answered %+v, want a refusal naming %q", r, c.lament)
			}
			if m := inv.ended(); m != 2 {
				t.Errorf("refused with the death-mark %d", m)
			}
			if _, err := os.Lstat(sh.store().Path(u.Rite)); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("a sanction was laid: %v", err)
			}
			if entries := sh.journal.all(); len(entries) != 1 || entries[0].Verdict != Refused {
				t.Errorf("journaled %+v", entries)
			}
		})
	}
}

// TestAbjure_StrikesTheSanctionsOfTheMaster: roused to abjure, the exalted
// servitor strikes the sanction of the rite, or every sanction of the
// master, and journals each with its sigil; a sanction not kept is refused,
// its death-mark 2.
func TestAbjure_StrikesTheSanctionsOfTheMaster(t *testing.T) {
	sh, _ := keepingShrine(t)
	keeper := sanction.Keeper{Sovereign: sh.cfg.Sovereign, Lord: sanction.Owner{UID: me, GID: os.Getgid()},
		Kin: sh.cfg.Kin}
	sigils := map[string]string{}
	for _, rite := range []string{"hallow", "zeal", "alpha"} {
		u := hallowUnit()
		u.Rite = rite
		if _, err := sh.store().Lay(sanction.New(u, sanctionedNow), keeper); err != nil {
			t.Fatal(err)
		}
		sigils[rite] = sanction.Short(u.Sigil())
	}
	abjure := func(words Abjuring, args ...string) (Reply, int) {
		t.Helper()
		inv := sh.rouse(append([]string{AbjureWord}, args...)...)
		go func() { _ = inv.ch.Send(words) }()
		return inv.answer(), inv.ended()
	}

	r, m := abjure(Abjuring{Protocol: Protocol, UID: me, Rite: "hallow"}, "hallow")
	if r.To != "abjure" || r.Verdict != Abjured || r.Protocol != Protocol || !slices.Equal(r.Abjured, []string{"hallow"}) ||
		m != 0 {
		t.Fatalf("answered %+v, death-mark %d", r, m)
	}
	if kept, _ := sh.store().Kept(keeper); !slices.Equal(kept, []string{"alpha", "zeal"}) {
		t.Errorf("the store keeps %v", kept)
	}
	for what, c := range map[string]struct {
		words  Abjuring
		args   []string
		lament string
	}{
		"a sanction not kept":     {Abjuring{Protocol: Protocol, UID: me, Rite: "hallow"}, []string{"hallow"}, "hallow"},
		"another rite in argv":    {Abjuring{Protocol: Protocol, UID: me, Rite: "zeal"}, []string{"alpha"}, "rite"},
		"another master":          {Abjuring{Protocol: Protocol, UID: me + 1, Rite: "zeal"}, []string{"zeal"}, "uid"},
		"all, yet argv names one": {Abjuring{Protocol: Protocol, UID: me, All: true}, []string{"zeal"}, "every"},
		"a malformed argv":        {Abjuring{Protocol: Protocol, UID: me, Rite: "zeal"}, []string{"zeal", "x"}, AbjureWord},
		"a rite and all":          {Abjuring{Protocol: Protocol, UID: me, Rite: "zeal", All: true}, []string{AllWord}, "every"},
	} {
		if r, m := abjure(c.words, c.args...); r.Verdict != Refused || !strings.Contains(r.Lament, c.lament) || m != 2 {
			t.Errorf("%s: answered %+v, death-mark %d", what, r, m)
		}
	}
	if kept, _ := sh.store().Kept(keeper); !slices.Equal(kept, []string{"alpha", "zeal"}) {
		t.Errorf("a refusal struck a sanction: the store keeps %v", kept)
	}

	r, m = abjure(Abjuring{Protocol: Protocol, UID: me, All: true}, AllWord)
	if r.Verdict != Abjured || !slices.Equal(r.Abjured, []string{"alpha", "zeal"}) || m != 0 {
		t.Fatalf("all: answered %+v, death-mark %d", r, m)
	}
	var struck []string
	for _, e := range sh.journal.all() {
		if e.Deed == "abjure" && e.Verdict == Abjured {
			struck = append(struck, e.Rite+" "+e.Sigil)
		}
	}
	if want := []string{"hallow " + sigils["hallow"], "alpha " + sigils["alpha"], "zeal " + sigils["zeal"]}; !slices.Equal(struck, want) {
		t.Errorf("journaled %v, want %v", struck, want)
	}
	if r, m := abjure(Abjuring{Protocol: Protocol, UID: me, All: true}, AllWord); r.Verdict != Refused ||
		!strings.Contains(r.Lament, "no sanction") || m != 2 {
		t.Errorf("all of none: answered %+v, death-mark %d", r, m)
	}
}
