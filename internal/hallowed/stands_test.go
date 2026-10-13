package hallowed

import (
	"os"
	"testing"
)

// TestStands_TellsWhetherTranscribeWouldChangeNothing: a vessel stands as
// decreed when it holds the scripture under the seal decreed (any seal
// when none is); an absent one never does. Nothing is touched.
func TestStands_TellsWhetherTranscribeWouldChangeNothing(t *testing.T) {
	tp := newTemple(t)
	tp.hall("/etc", 0o755)
	tp.scroll("/etc/vessel", "old\n", 0o640)
	before := tp.stat("/etc/vessel")
	for name, c := range map[string]struct {
		path, scripture string
		seal            *os.FileMode
		want            bool
	}{
		"the same, without a seal": {"/etc/vessel", "old\n", nil, true},
		"the same, under its seal": {"/etc/vessel", "old\n", sealPtr(0o640), true},
		"another seal":             {"/etc/vessel", "old\n", sealPtr(0o600), false},
		"other scripture":          {"/etc/vessel", "old", nil, false},
		"absent":                   {"/etc/absent", "", nil, false},
	} {
		got, err := tp.g.Stands(c.path, []byte(c.scripture), c.seal)
		if err != nil || got != c.want {
			t.Errorf("%s: %v %v, want %v", name, got, err, c.want)
		}
	}
	if after := tp.stat("/etc/vessel"); after.Ino != before.Ino || after.Mtim != before.Mtim {
		t.Error("the vessel was touched")
	}
}

// TestBonded_TellsWhatStandsAtAName: the anchor of a bond as written,
// nothing standing, or a scroll; the bond is never followed.
func TestBonded_TellsWhatStandsAtAName(t *testing.T) {
	tp := newTemple(t)
	tp.hall("/etc", 0o755)
	tp.bond("/etc/theme", "/usr/share/theme/on")
	tp.scroll("/etc/scroll", "x", 0o644)
	for name, want := range map[string]struct {
		anchor       string
		bond, absent bool
	}{
		"/etc/theme":  {anchor: "/usr/share/theme/on", bond: true},
		"/etc/absent": {absent: true},
		"/etc/scroll": {},
	} {
		anchor, bond, absent, err := tp.g.Bonded(name)
		if err != nil || anchor != want.anchor || bond != want.bond || absent != want.absent {
			t.Errorf("%s: %q %v %v %v, want %+v", name, anchor, bond, absent, err, want)
		}
	}
}
