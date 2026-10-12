package sanction

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	u := build(t, splitLock, splitLockFiles())
	root := t.TempDir()
	st := Store{Root: root, UID: 1000}
	p := lay(t, root, "split-lock", New(u, at))
	if st.Path("split-lock") != p {
		t.Errorf("Path = %s, want %s", st.Path("split-lock"), p)
	}
	s, err := st.Read("split-lock")
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
		if s, err := st.Read("split-lock"); s != nil || err != nil {
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
	u := build(t, splitLock, splitLockFiles())
	good, err := json.Marshal(New(u, at))
	if err != nil {
		t.Fatal(err)
	}
	edits := map[string]func(m map[string]any){
		"unknown":  func(m map[string]any) { m["blessing"] = true },
		"sigil":    func(m map[string]any) { m["sigil"] = strings.Repeat("0", 64) },
		"rite":     func(m map[string]any) { m["rite"] = "other" },
		"uid":      func(m map[string]any) { m["uid"] = 1001 },
		"pattern":  func(m map[string]any) { m["pattern"] = "Mark II" },
		"verse":    func(m map[string]any) { verse(m)["shrine"] = "x" },
		"two keys": func(m map[string]any) { verse(m)["incantation"] = "x" },
	}
	for what, edit := range edits {
		root := t.TempDir()
		m := map[string]any{}
		_ = json.Unmarshal(good, &m)
		edit(m)
		lay(t, root, "split-lock", m)
		s, err := Store{Root: root, UID: 1000}.Read("split-lock")
		if s != nil || err == nil || !strings.Contains(err.Error(), "split-lock") {
			t.Errorf("%s: Read = %v, %v; want a lament", what, s, err)
		}
	}
}

// verse is the first verse of the first plan of the sanction tree m.
func verse(m map[string]any) map[string]any {
	return m["aspects"].([]any)[0].(map[string]any)["liturgy"].([]any)[0].(map[string]any)
}

// TestStore_ListsEverySanctionByRite: the master's sanctions are listed by
// the name of their rite; one that cannot be read is lamented apart.
func TestStore_ListsEverySanctionByRite(t *testing.T) {
	u := build(t, splitLock, splitLockFiles())
	root := t.TempDir()
	st := Store{Root: root, UID: 1000}
	if got, laments := st.List(); len(got) != 0 || len(laments) != 0 {
		t.Fatalf("an empty store lists %v %v", got, laments)
	}
	second := *u
	second.Rite = "a-second"
	lay(t, root, "split-lock", New(u, at))
	lay(t, root, "a-second", New(&second, at))
	if err := os.WriteFile(filepath.Join(root, "1000", "broken.json"), []byte("{"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "1000", "stray.txt"), []byte("x"), 0o640); err != nil {
		t.Fatal(err)
	}
	got, laments := st.List()
	if len(got) != 2 || got[0].Rite != "a-second" || got[1].Rite != "split-lock" {
		t.Errorf("List = %v", got)
	}
	if len(laments) != 1 || !strings.Contains(laments[0].Error(), "broken") {
		t.Errorf("laments %v", laments)
	}
}

// TestJudge_TellsTheStanding: no sanction is unsanctioned; a sanction whose
// sigil the unit no longer bears, or a unit that cannot be drawn, lapsed.
func TestJudge_TellsTheStanding(t *testing.T) {
	u := build(t, splitLock, splitLockFiles())
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
