package augury

import (
	"testing"

	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// TestAugurMarkII_MarshalsOmensBySubject: the augury of a state rite names
// each omen by its subject and designation, with the omen read when there
// is one, every aspect it fits and its aspect when it fits one; the rite
// tells its fitting aspects, and its taint is named by subject.
func TestAugurMarkII_MarshalsOmensBySubject(t *testing.T) {
	fx := newFixture(t)
	r := fx.stateRite(t, `"performance", "balanced", "powersave"`, "",
		spirit("smt", `{"performance": "off", "*": "on"}`, `{"scan": "cat $DATA/smt"}`)+", "+
			spirit("profile", "", `{"scan": "cat $DATA/profile"}`)+", "+
			spirit("governor", "", `{"trials": {"*": "grep -qx {{omen}} $DATA/governor"}}`)+", "+
			`{"name": "flag", "designation": "transcription", "vessel": "$DATA/flag",
			  "omens": {"performance": "raised", "*": null}}`)
	fx.shows(t, "smt", "on\n")
	fx.shows(t, "profile", "porpl")
	fx.shows(t, "governor", "balanced\npowersave\n")
	fx.slate(t, r, "powersave", invocation.Triumph, map[string]string{"place": "desk"})

	got := marshal(t, fx.augur(t, r, Options{}))

	want := compact(t, `{"rite": "gaming", "edict": false, "aspect": "powersave", "standing": "performed",
	  "sanction": "", "auspex": "mute", "desecrated": false, "inscriptions": {"place": "desk"},
	  "canon": [{"key": "place", "purpose": "where", "mandatory": false, "decrees": {}, "decree": null}],
	  "last_rite": {"verdict": "triumph", "at": "2026-10-10T13:30:00Z"},
	  "fitting": ["balanced", "powersave"],
	  "omens": [
	    {"subject": "smt", "designation": "machine-spirit", "omen": "on", "aspects": ["balanced", "powersave"]},
	    {"subject": "governor", "designation": "machine-spirit", "aspects": ["balanced", "powersave"]},
	    {"subject": "flag", "designation": "transcription", "aspects": ["balanced", "powersave"]}],
	  "taint": []}`)
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestAugurMarkII_MarshalsOneAspectAndTaintBySubject(t *testing.T) {
	a := Augury{Rite: "gaming", Standing: Tainted, Aspect: "on", Fitting: []string{"on"},
		Omens: []Omen{{Subject: "niri", Designation: "sanctum", Aspects: []string{"on"}, Aspect: "on"},
			{Subject: "smt", Designation: "machine-spirit", Read: "off", Aspects: []string{"on"}, Aspect: "on"}},
		Taint: []Taint{{Subject: "niri"}}}
	want := compact(t, `{"rite": "gaming", "edict": false, "aspect": "on", "standing": "tainted", "sanction": "",
	  "auspex": "", "desecrated": false, "inscriptions": {}, "canon": [], "last_rite": null, "fitting": ["on"],
	  "omens": [{"subject": "niri", "designation": "sanctum", "aspects": ["on"], "aspect": "on"},
	            {"subject": "smt", "designation": "machine-spirit", "omen": "off", "aspects": ["on"], "aspect": "on"}],
	  "taint": [{"subject": "niri", "designation": "sanctum"}]}`)
	if got := marshal(t, a); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	a = Augury{Rite: "gaming", Standing: Corrupted}
	if got := marshal(t, a); got != compact(t, `{"rite": "gaming", "edict": false, "aspect": "", "standing": "corrupted",
	  "sanction": "", "auspex": "", "desecrated": false, "inscriptions": {}, "canon": [], "last_rite": null,
	  "fitting": [], "omens": [], "taint": []}`) {
		t.Errorf("a corrupted state rite: %s", got)
	}
}
