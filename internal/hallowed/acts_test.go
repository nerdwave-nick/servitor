package hallowed

import (
	"errors"
	"os"
	"slices"
	"testing"
)

// TestTranscribe_KeepsTheOwnerAndSealOfTheVessel: a standing vessel keeps
// its owner and seal; the seal of the step serves only a vessel brought
// into being.
func TestTranscribe_KeepsTheOwnerAndSealOfTheVessel(t *testing.T) {
	for name, s := range map[string]*os.FileMode{"without a seal": nil, "with a seal": sealPtr(0o600)} {
		t.Run(name, func(t *testing.T) {
			tp := newTemple(t)
			tp.hall("/etc", 0o755)
			tp.scroll("/etc/vessel", "old\n", 0o640)
			before := tp.stat("/etc/vessel")
			if _, err := tp.g.Transcribe("/etc/vessel", []byte("new\n"), s); err != nil {
				t.Fatal(err)
			}
			after := tp.stat("/etc/vessel")
			if got := tp.content("/etc/vessel"); got != "new\n" {
				t.Fatalf("the vessel holds %q", got)
			}
			if seal(after) != 0o640 || after.Uid != before.Uid || after.Gid != before.Gid {
				t.Fatalf("the vessel is %o %d:%d, not %o %d:%d",
					seal(after), after.Uid, after.Gid, seal(before), before.Uid, before.Gid)
			}
			if after.Ino == before.Ino {
				t.Fatal("the vessel was written in place, not replaced whole")
			}
			if names := tp.names("/etc"); !slices.Equal(names, []string{"vessel"}) {
				t.Fatalf("the hall holds %v", names)
			}
		})
	}
}

// TestTranscribe_FollowsTheBondsOfTheVessel: a bonded vessel is transcribed
// where its bond leads, and the bond stays.
func TestTranscribe_FollowsTheBondsOfTheVessel(t *testing.T) {
	tp := newTemple(t)
	tp.hall("/etc", 0o755)
	tp.hall("/usr/lib", 0o755)
	tp.scroll("/usr/lib/vessel", "old", 0o644)
	tp.bond("/etc/vessel", "/usr/lib/vessel")
	if _, err := tp.g.Transcribe("/etc/vessel", []byte("new"), nil); err != nil {
		t.Fatal(err)
	}
	if tp.anchor("/etc/vessel") != "/usr/lib/vessel" || tp.content("/usr/lib/vessel") != "new" {
		t.Fatalf("bond %q, vessel %q", tp.anchor("/etc/vessel"), tp.content("/usr/lib/vessel"))
	}
}

func TestTranscribe_BringsAVesselIntoBeingForTheLord(t *testing.T) {
	for name, c := range map[string]struct {
		seal *os.FileMode
		want os.FileMode
	}{"without a seal": {nil, 0o644}, "with a seal": {sealPtr(0o600), 0o600}} {
		t.Run(name, func(t *testing.T) {
			tp := newTemple(t)
			tp.hall("/etc", 0o755)
			if _, err := tp.g.Transcribe("/etc/vessel", []byte("new"), c.seal); err != nil {
				t.Fatal(err)
			}
			st := tp.stat("/etc/vessel")
			if tp.content("/etc/vessel") != "new" || seal(st) != c.want || (Owner{st.Uid, st.Gid}) != tp.g.Lord {
				t.Fatalf("the vessel is %o %d:%d holding %q", seal(st), st.Uid, st.Gid, tp.content("/etc/vessel"))
			}
		})
	}
}

// TestTranscribe_CreatesWithoutReplacing: a vessel brought into being never
// replaces one that came to stand there after the walk.
func TestTranscribe_CreatesWithoutReplacing(t *testing.T) {
	tp := newTemple(t)
	tp.hall("/etc", 0o755)
	p, err := tp.g.walk("/etc/vessel", vessel)
	if err != nil {
		t.Fatal(err)
	}
	defer p.close()
	if p.stat != nil {
		t.Fatal("the vessel already stands")
	}
	tp.scroll("/etc/vessel", "intruder", 0o644)
	if err := inscribe(p, []byte("new"), tp.g.Lord, 0o644, true); err == nil {
		t.Fatal("a vessel brought into being replaced what came to stand there")
	}
	if got := tp.content("/etc/vessel"); got != "intruder" {
		t.Fatalf("the vessel holds %q", got)
	}
	if names := tp.names("/etc"); !slices.Equal(names, []string{"vessel"}) {
		t.Fatalf("the hall holds %v", names)
	}
}

// TestTranscribe_FailsClosedOnAttributes: what the servitor cannot keep it
// does not cast away.
func TestTranscribe_FailsClosedOnAttributes(t *testing.T) {
	t.Run("upon the vessel", func(t *testing.T) {
		tp := newTemple(t)
		tp.hall("/etc", 0o755)
		tp.scroll("/etc/vessel", "old", 0o644)
		attribute(t, tp.at("/etc/vessel"), "user.servitor", []byte("blessed"))
		if _, err := tp.g.Transcribe("/etc/vessel", []byte("new"), nil); !errors.Is(err, errAttributes) {
			t.Fatalf("transcribed a vessel bearing attributes: %v", err)
		}
		if _, err := tp.g.Strike("/etc/vessel"); !errors.Is(err, errAttributes) {
			t.Fatalf("struck a vessel bearing attributes: %v", err)
		}
		if tp.content("/etc/vessel") != "old" {
			t.Fatal("the vessel was written")
		}
	})
	t.Run("an access list upon the vessel", func(t *testing.T) {
		tp := newTemple(t)
		tp.hall("/etc", 0o755)
		tp.scroll("/etc/vessel", "old", 0o644)
		access := defaultAccess(tp.me.UID)
		attribute(t, tp.at("/etc/vessel"), "system.posix_acl_access", access)
		if _, err := tp.g.Transcribe("/etc/vessel", []byte("new"), nil); !errors.Is(err, errAttributes) {
			t.Fatalf("transcribed a vessel bearing an access list: %v", err)
		}
	})
	t.Run("an access list the hall gives its new vessels", func(t *testing.T) {
		tp := newTemple(t)
		tp.hall("/etc", 0o755)
		attribute(t, tp.at("/etc"), "system.posix_acl_default", defaultAccess(tp.me.UID))
		if _, err := tp.g.Transcribe("/etc/vessel", []byte("new"), nil); !errors.Is(err, errAttributes) {
			t.Fatalf("brought a vessel into being bearing an access list: %v", err)
		}
		if names := tp.names("/etc"); len(names) != 0 {
			t.Fatalf("the hall holds %v", names)
		}
	})
}

func TestTranscribe_RefusesAVesselKnownUnderManyNames(t *testing.T) {
	tp := newTemple(t)
	tp.hall("/etc", 0o755)
	tp.scroll("/etc/vessel", "old", 0o644)
	if err := os.Link(tp.at("/etc/vessel"), tp.at("/etc/other")); err != nil {
		t.Fatal(err)
	}
	if _, err := tp.g.Transcribe("/etc/vessel", []byte("new"), nil); !errors.Is(err, errManyNames) {
		t.Fatalf("transcribed a vessel of two names: %v", err)
	}
	if _, err := tp.g.Strike("/etc/vessel"); !errors.Is(err, errManyNames) {
		t.Fatalf("struck a vessel of two names: %v", err)
	}
	if tp.content("/etc/vessel") != "old" || tp.content("/etc/other") != "old" {
		t.Fatal("the vessel was written")
	}
}

// TestRevert_RestoresExactlyWhatStood: bytes, owner and seal of a vessel,
// and the absence of one brought into being.
func TestRevert_RestoresExactlyWhatStood(t *testing.T) {
	const old = "\x00\xffold scripture\r\nwithout end"
	tp := newTemple(t)
	tp.hall("/etc", 0o755)
	tp.scroll("/etc/vessel", old, 0o600)
	before := tp.stat("/etc/vessel")

	m, err := tp.g.Transcribe("/etc/vessel", []byte("new"), sealPtr(0o644))
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Revert(); err != nil {
		t.Fatal(err)
	}
	after := tp.stat("/etc/vessel")
	if tp.content("/etc/vessel") != old || seal(after) != 0o600 || after.Uid != before.Uid || after.Gid != before.Gid {
		t.Fatalf("reverted to %q, %o %d:%d", tp.content("/etc/vessel"), seal(after), after.Uid, after.Gid)
	}

	m, err = tp.g.Strike("/etc/vessel")
	if err != nil {
		t.Fatal(err)
	}
	if !tp.absent("/etc/vessel") {
		t.Fatal("the vessel was not struck")
	}
	if err := m.Revert(); err != nil {
		t.Fatal(err)
	}
	if tp.content("/etc/vessel") != old || seal(tp.stat("/etc/vessel")) != 0o600 {
		t.Fatalf("the struck vessel was raised as %q", tp.content("/etc/vessel"))
	}

	m, err = tp.g.Transcribe("/etc/new", []byte("new"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Revert(); err != nil {
		t.Fatal(err)
	}
	if !tp.absent("/etc/new") {
		t.Fatal("the vessel brought into being still stands")
	}
	if names := tp.names("/etc"); !slices.Equal(names, []string{"vessel"}) {
		t.Fatalf("the hall holds %v", names)
	}
}

func TestStrike_LeavesAnAbsentVesselAbsent(t *testing.T) {
	tp := newTemple(t)
	tp.hall("/etc", 0o755)
	m, err := tp.g.Strike("/etc/vessel")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Revert(); err != nil || !tp.absent("/etc/vessel") {
		t.Fatalf("reverted to presence: %v", err)
	}
}
