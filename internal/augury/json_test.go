package augury

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

func compact(t *testing.T, s string) string {
	t.Helper()
	var b bytes.Buffer
	if err := json.Compact(&b, []byte(s)); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func marshal(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestAugury_MarshalsAsWrittenInTheCodex(t *testing.T) {
	// The example of the specification, §8, key for key, with the
	// "edict": false that every rite gains (§16.3), its standing towards a
	// sanction (§17.2), here sanctioned, and its omens by subject (§19.6).
	codex := `{"rite": "theme", "edict": false, "aspect": "porpl", "standing": "performed",
 "sanction": "sanctioned", "auspex": "heeded", "desecrated": false,
 "inscriptions": {"reason": "gaming remnant"},
 "canon": [{"key": "reason", "purpose": "why the visage was changed", "mandatory": false,
            "decrees": {}, "decree": null}],
 "last_rite": {"verdict": "triumph", "at": "2026-10-10T13:30:00Z"},
 "fitting": ["porpl"],
 "omens": [{"subject": "visage", "designation": "tether", "aspects": ["porpl"], "aspect": "porpl"},
           {"subject": "hue", "designation": "machine-spirit", "omen": "porpl", "aspects": ["porpl"], "aspect": "porpl"}],
 "taint": []}`
	a := Augury{Rite: "theme", Aspect: "porpl", Standing: Performed, Sanction: sanction.Sanctioned, Auspex: Heeded,
		Inscriptions: map[string]string{"reason": "gaming remnant"},
		Canon:        []Declaration{{Key: "reason", Purpose: "why the visage was changed", Decrees: map[string]string{}}},
		LastRite:     &LastRite{Verdict: invocation.Triumph, At: noon}, Fitting: []string{"porpl"},
		Omens: []Omen{
			{Subject: "visage", Designation: "tether", Aspects: []string{"porpl"}, Aspect: "porpl"},
			{Subject: "hue", Designation: "machine-spirit", Read: "porpl", Aspects: []string{"porpl"}, Aspect: "porpl"},
		}}
	if got, want := marshal(t, a), compact(t, codex); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestAugury_MarshalsFormerOnlyWhenDesecrated(t *testing.T) {
	a := Augury{Rite: "dnd", Aspect: "on", Standing: Tainted, Desecrated: true, Former: "off",
		Fitting: []string{"on"},
		Omens:   []Omen{{Subject: "mako", Designation: "sanctum", Aspects: []string{"on"}, Aspect: "on"}},
		Taint:   []Taint{{Subject: "mako", Vessel: "/v"}}}
	want := `{"rite":"dnd","edict":false,"aspect":"on","standing":"tainted","sanction":"","auspex":"","desecrated":true,"former":"off","inscriptions":{},"canon":[],` +
		`"last_rite":null,"fitting":["on"],"omens":[{"subject":"mako","designation":"sanctum","aspects":["on"],"aspect":"on"}],` +
		`"taint":[{"subject":"mako","designation":"sanctum"}]}`
	if got := marshal(t, a); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestAugury_MarshalsADormantRiteWithEmptyLists(t *testing.T) {
	want := `{"rite":"dnd","edict":false,"aspect":"","standing":"dormant","sanction":"","auspex":"forgone","desecrated":false,"inscriptions":{},"canon":[],` +
		`"last_rite":null,"fitting":[],"omens":[],"taint":[]}`
	if got := marshal(t, Augury{Rite: "dnd", Standing: Dormant, Auspex: Forgone}); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

// TestAugury_MarshalsEverySubjectByItsName: the augury of a rite read from
// the machine names each omen by its subject, a machine-spirit's with the
// omen it showed.
func TestAugury_MarshalsEverySubjectByItsName(t *testing.T) {
	fx := newFixture(t)
	fx.invoke(t, fx.rite(t, `"on", "off"`, "", strings.Join([]string{niri, flag, current}, ", ")), "on",
		map[string]string{"reason": "gaming remnant"})
	r := fx.rite(t, `"on", "off"`, "", strings.Join([]string{niri, flag, current,
		spirit("hue", `{"on": "dark", "off": "light"}`, `{"scan": "echo dark"}`)}, ", "))
	got := marshal(t, fx.augur(t, r, Options{}))
	want := compact(t, `{"rite": "mouse", "edict": false, "aspect": "on", "standing": "performed", "sanction": "", "auspex": "heeded",
 "desecrated": false, "inscriptions": {"reason": "gaming remnant"},
 "canon": [{"key": "place", "purpose": "where", "mandatory": false, "decrees": {}, "decree": null},
           {"key": "reason", "purpose": "why", "mandatory": false, "decrees": {}, "decree": null}],
 "last_rite": {"verdict": "triumph", "at": "2026-10-10T13:30:00Z"},
 "fitting": ["on"],
 "omens": [{"subject": "niri", "designation": "sanctum", "aspects": ["on"], "aspect": "on"},
           {"subject": "flag", "designation": "transcription", "aspects": ["on"], "aspect": "on"},
           {"subject": "theme", "designation": "tether", "aspects": ["on"], "aspect": "on"},
           {"subject": "hue", "designation": "machine-spirit", "omen": "dark", "aspects": ["on"], "aspect": "on"}],
 "taint": []}`)
	if got != want {
		t.Errorf("got  %s\n want %s", got, want)
	}
}
