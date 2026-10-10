package augury

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
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
	// The example of the specification, §8, key for key.
	codex := `{"rite": "theme", "aspect": "porpl", "standing": "performed",
 "desecrated": false,
 "inscriptions": {"reason": "gaming remnant"},
 "last_rite": {"verdict": "triumph", "at": "2026-10-10T13:30:00Z"},
 "omens": [{"verse": 2, "tether": "~/.local/share/nfluff/current-theme", "aspect": "porpl"},
           {"auspex": true, "aspect": "porpl"}],
 "taint": []}`
	a := Augury{Rite: "theme", Aspect: "porpl", Standing: Performed,
		Inscriptions: map[string]string{"reason": "gaming remnant"},
		LastRite:     &LastRite{Verdict: invocation.Triumph, At: noon},
		Omens: []Omen{
			{Verse: invocation.Verse{Number: 2, Kind: librarium.KindTether, Target: "~/.local/share/nfluff/current-theme"}, Aspect: "porpl"},
			{Auspex: true, Aspect: "porpl"},
		}}
	if got, want := marshal(t, a), compact(t, codex); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestAugury_MarshalsFormerOnlyWhenDesecrated(t *testing.T) {
	a := Augury{Rite: "dnd", Aspect: "on", Standing: Tainted, Desecrated: true, Former: "off",
		Omens: []Omen{{Verse: invocation.Verse{Number: 1, Kind: librarium.KindSanctum, Target: "/v"}, Aspect: "on"}},
		Taint: []Taint{{Verse: invocation.Verse{Number: 1, Kind: librarium.KindSanctum, Target: "/v"}}}}
	want := `{"rite":"dnd","aspect":"on","standing":"tainted","desecrated":true,"former":"off","inscriptions":{},` +
		`"last_rite":null,"omens":[{"verse":1,"sanctum":"/v","aspect":"on"}],"taint":[{"verse":1,"sanctum":"/v"}]}`
	if got := marshal(t, a); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestAugury_MarshalsADormantRiteWithEmptyLists(t *testing.T) {
	want := `{"rite":"dnd","aspect":"","standing":"dormant","desecrated":false,"inscriptions":{},` +
		`"last_rite":null,"omens":[],"taint":[]}`
	if got := marshal(t, Augury{Rite: "dnd", Standing: Dormant}); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestAugury_MarshalsEveryStepByItsOwnKey(t *testing.T) {
	fx := newFixture(t)
	write(t, filepath.Join(fx.data, "scroll.sh"), "true\n")
	r := fx.rite(t, `"on", "off"`, `
  "auspex": "echo on",`, everyStep)
	fx.invoke(t, r, "on", map[string]string{"reason": "gaming remnant"})
	got := marshal(t, fx.augur(t, r, Options{}))
	want := compact(t, `{"rite": "mouse", "aspect": "on", "standing": "performed", "desecrated": false,
 "inscriptions": {"reason": "gaming remnant"},
 "last_rite": {"verdict": "triumph", "at": "2026-10-10T13:30:00Z"},
 "omens": [{"verse": 1, "sanctum": "`+fx.data+`/niri.kdl", "aspect": "on"},
           {"verse": 2, "transcription": "`+fx.data+`/flag", "aspect": "on"},
           {"verse": 3, "tether": "`+fx.data+`/current", "aspect": "on"},
           {"auspex": true, "aspect": "on"}],
 "taint": []}`)
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
