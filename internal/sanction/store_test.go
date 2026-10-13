package sanction

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// at is the hour every sanction of these trials was sanctioned.
var at = time.Date(2026, 10, 12, 18, 0, 0, 0, time.UTC)

// lay writes the sanction s into the store at root, for uid 1000, as rite.
func lay(t *testing.T, root, rite string, s any) string {
	t.Helper()
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, "1000", rite+".json")
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o640); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestStore_ReadsTheSanctionLaid: a sanction is read back as it was laid,
// strictly; none laid, or no store at all, is no sanction and no lament.
func TestStore_ReadsTheSanctionLaid(t *testing.T) {
	u := buildGaming(t, gaming, gamingFiles())
	root := t.TempDir()
	st := Store{Root: root, UID: 1000}
	p := lay(t, root, "gaming", New(u, at))
	if st.Path("gaming") != p {
		t.Errorf("Path = %s, want %s", st.Path("gaming"), p)
	}
	s, err := st.Read("gaming")
	if err != nil || s == nil {
		t.Fatalf("Read = %v, %v", s, err)
	}
	if s.Sigil != u.Sigil() || !s.SanctionedAt.Equal(at) || string(s.Canonical()) != string(u.Canonical()) {
		t.Errorf("read %s %s\n%s", s.Sigil, s.SanctionedAt, s.Canonical())
	}
	if Judge(u, s) != Sanctioned {
		t.Errorf("Judge = %q", Judge(u, s))
	}
	for _, st := range []Store{{Root: root, UID: 1001}, {Root: filepath.Join(root, "nowhere"), UID: 1000}, {}} {
		if s, err := st.Read("gaming"); s != nil || err != nil {
			t.Errorf("%+v: Read = %v, %v; want nothing", st, s, err)
		}
	}
	if s, err := st.Read("other"); s != nil || err != nil {
		t.Errorf("Read(other) = %v, %v", s, err)
	}
}

// TestStore_RefusesWhatIsNoTrueSanction: an unknown key, a sigil that is not
// its unit's, a sanction laid under another rite's name or uid, or another
// pattern, is lamented and is no sanction.
func TestStore_RefusesWhatIsNoTrueSanction(t *testing.T) {
	u := buildGaming(t, gaming, gamingFiles())
	good, err := json.Marshal(New(u, at))
	if err != nil {
		t.Fatal(err)
	}
	edits := map[string]func(m map[string]any){
		"unknown":  func(m map[string]any) { m["blessing"] = true },
		"sigil":    func(m map[string]any) { m["sigil"] = strings.Repeat("0", 64) },
		"rite":     func(m map[string]any) { m["rite"] = "other" },
		"uid":      func(m map[string]any) { m["uid"] = 1001 },
		"pattern":  func(m map[string]any) { m["pattern"] = "Mark III" },
		"verse":    func(m map[string]any) { verse(m)["shrine"] = "x" },
		"two keys": func(m map[string]any) { verse(m)["incantation"] = "x" },
	}
	for what, edit := range edits {
		root := t.TempDir()
		m := map[string]any{}
		_ = json.Unmarshal(good, &m)
		edit(m)
		lay(t, root, "gaming", m)
		s, err := Store{Root: root, UID: 1000}.Read("gaming")
		if s != nil || err == nil || !strings.Contains(err.Error(), "gaming") {
			t.Errorf("%s: Read = %v, %v; want a lament", what, s, err)
		}
	}
}

// verse is the first subject of the first plan of the sanction tree m.
func verse(m map[string]any) map[string]any {
	return m["aspects"].([]any)[0].(map[string]any)["subjects"].([]any)[0].(map[string]any)
}

// TestStore_ListsEverySanctionByRite: the master's sanctions are listed by
// the name of their rite; one that cannot be read is lamented apart.
func TestStore_ListsEverySanctionByRite(t *testing.T) {
	u := buildGaming(t, gaming, gamingFiles())
	root := t.TempDir()
	st := Store{Root: root, UID: 1000}
	if got, laments := st.List(); len(got) != 0 || len(laments) != 0 {
		t.Fatalf("an empty store lists %v %v", got, laments)
	}
	second := *u
	second.Rite = "a-second"
	lay(t, root, "gaming", New(u, at))
	lay(t, root, "a-second", New(&second, at))
	if err := os.WriteFile(filepath.Join(root, "1000", "broken.json"), []byte("{"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "1000", "stray.txt"), []byte("x"), 0o640); err != nil {
		t.Fatal(err)
	}
	got, laments := st.List()
	if len(got) != 2 || got[0].Rite != "a-second" || got[1].Rite != "gaming" {
		t.Errorf("List = %v", got)
	}
	if len(laments) != 1 || !strings.Contains(laments[0].Error(), "broken") {
		t.Errorf("laments %v", laments)
	}
}

// TestJudge_TellsTheStanding: no sanction is unsanctioned; a sanction whose
// sigil the unit no longer bears, or a unit that cannot be drawn, lapsed.
func TestJudge_TellsTheStanding(t *testing.T) {
	u := buildGaming(t, gaming, gamingFiles())
	s := New(u, at)
	other := *s
	other.Sigil = strings.Repeat("0", 64)
	for _, c := range []struct {
		unit *Unit
		s    *Sanction
		want Standing
	}{{u, nil, Unsanctioned}, {nil, nil, Unsanctioned}, {u, s, Sanctioned}, {u, &other, Lapsed}, {nil, s, Lapsed}} {
		if got := Judge(c.unit, c.s); got != c.want {
			t.Errorf("Judge(%v, %v) = %q, want %q", c.unit != nil, c.s != nil, got, c.want)
		}
	}
}

// TestStore_ASanctionOfMarkIIsReadAndLapses: a sanction laid while its rite
// was of pattern Mark I is still read, listed and judged, and lapses
// against the unit of the rite converted, so that the master sanctions it
// anew rather than hear it lamented as no true sanction.
func TestStore_ASanctionOfMarkIIsReadAndLapses(t *testing.T) {
	elder := &Unit{Pattern: PatternMarkI, UID: 1000, Rite: "gaming",
		Environment: Environment{Pattern: PatternMarkI, Keys: []string{"SERVITOR_ASPECT", "SERVITOR_FORMER_ASPECT",
			"SERVITOR_RITE"}},
		Aspects: []Plan{{Aspect: "on", Inscriptions: map[string]string{}, Liturgy: []Verse{
			{Verse: 2, Kind: librarium.KindTranscription, Target: "/etc/sysctl.d/90-gaming.conf",
				Scripture: []byte("kernel.split_lock_mitigate = 0\n")}}}}}
	root := t.TempDir()
	lay(t, root, "gaming", New(elder, at))
	st := Store{Root: root, UID: 1000}
	s, err := st.Read("gaming")
	if err != nil || s == nil || s.Pattern != PatternMarkI {
		t.Fatalf("Read = %v, %v", s, err)
	}
	if got, laments := st.List(); len(got) != 1 || len(laments) != 0 {
		t.Errorf("List = %v, %v", got, laments)
	}
	if got := Judge(buildGaming(t, gaming, gamingFiles()), s); got != Lapsed {
		t.Errorf("Judge = %q, want %q", got, Lapsed)
	}
}
