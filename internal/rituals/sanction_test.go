package rituals

import (
	"context"
	"encoding/json/v2"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// The trials never read the store of the machine: a servitor given no
// store reads one that is never kept.
func init() { SanctionRoot = filepath.Join(os.TempDir(), "servitor-trials-keep-no-sanctions") }

// sanctionedAt is the hour of every sanction these trials lay.
var sanctionedAt = time.Date(2026, 10, 12, 18, 0, 0, 0, time.UTC)

// sanctuary is a Librarium whose sanctions lie in a store of its own: rites
// of every standing towards a sanction, and sanctions forsaken.
type sanctuary struct {
	*fixture
	s     *Servitor
	store sanction.Store
}

// exaltedRite is a rite of one exalted verse speaking word.
func exaltedRite(word string) string {
	return `{"pattern": "Mark I", "aspects": ["on", "off"], "liturgy": [
	  {"incantation": "echo ` + word + ` >> $DATA/said", "exalted": true},
	  {"vox-cast": "success"}]}`
}

// newSanctuary keeps "split-lock" sanctioned, "lapsing" lapsed (sanctioned,
// then edited), "fresh" unsanctioned, "plain" without exalted verses yet
// bearing a forsaken sanction, "broken" heretical with a sanction, and a
// forsaken sanction of the rite "a-gone", which is recorded nowhere.
func newSanctuary(t *testing.T) *sanctuary {
	t.Helper()
	fx := newFixture(t)
	sc := &sanctuary{fixture: fx, store: sanction.Store{Root: t.TempDir(), UID: 1000}}
	fx.rite("split-lock", exaltedRite("sanctioned"))
	fx.rite("lapsing", exaltedRite("before"))
	fx.rite("fresh", exaltedRite("fresh"))
	fx.rite("plain", `{"pattern": "Mark I", "aspects": ["on"], "liturgy": [{"incantation": "echo plain"}]}`)
	fx.rite("broken", exaltedRite("broken"))
	fx.rite("a-gone", exaltedRite("gone"))
	sc.s = sc.servitor()
	for _, name := range []string{"split-lock", "lapsing", "broken", "a-gone", "plain"} {
		sc.sanctify(name)
	}
	fx.rite("lapsing", exaltedRite("after"))
	fx.rite("broken", `{"pattern": "Mark I", "aspects": []}`)
	if err := os.Remove(filepath.Join(fx.lib, "rites", "a-gone.json")); err != nil {
		t.Fatal(err)
	}
	sc.s = sc.servitor()
	return sc
}

// servitor opens the Librarium anew upon the sanctuary's store.
func (sc *sanctuary) servitor() *Servitor {
	s := sc.fixture.servitor()
	s.Sanctions = &sc.store
	return s
}

// sanctify lays the sanction of the rite name as it stands; a rite without
// exalted verses is sanctioned as "fresh", under its own name.
func (sc *sanctuary) sanctify(name string) {
	sc.t.Helper()
	r := sc.s.Librarium.Rites[name]
	if !r.Exalted() {
		r = sc.s.Librarium.Rites["fresh"]
	}
	u, err := sc.s.Unit(r)
	if err != nil {
		sc.t.Fatal(err)
	}
	u.Rite = name
	sc.lay(name, sanction.New(u, sanctionedAt))
}

func (sc *sanctuary) lay(name string, s *sanction.Sanction) {
	sc.t.Helper()
	data, err := json.Marshal(s)
	if err != nil {
		sc.t.Fatal(err)
	}
	p := sc.store.Path(name)
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		sc.t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o640); err != nil {
		sc.t.Fatal(err)
	}
}

// sigil is the sigil the sanction of name bears.
func (sc *sanctuary) sigil(name string) string {
	sc.t.Helper()
	kept, err := sc.store.Read(name)
	if err != nil || kept == nil {
		sc.t.Fatalf("the sanction of %s: %v", name, err)
	}
	return kept.Sigil
}

// forsakenWords are the laments of the forsaken sanctions of the sanctuary.
var forsakenWords = []string{
	`the sanction of "a-gone" is forsaken: no rite named "a-gone" is recorded in the Librarium any more, and ` +
		`the sanction serves nothing — abjure it: servitor abjure a-gone`,
	`the sanction of "plain" is forsaken: the rite "plain" holds no exalted verse any more, and the sanction ` +
		`serves nothing — abjure it: servitor abjure plain`,
}

// TestSanction_CensusAndAuguryTellEachStanding: every rite of the census and
// its augury bears its standing towards its sanction, and the census laments
// every forsaken sanction after the laments of its rites.
func TestSanction_CensusAndAuguryTellEachStanding(t *testing.T) {
	sc := newSanctuary(t)
	want := map[string]sanction.Standing{"split-lock": sanction.Sanctioned, "lapsing": sanction.Lapsed,
		"fresh": sanction.Unsanctioned, "plain": sanction.None, "broken": sanction.Lapsed}

	entries, laments := sc.s.Census(true)
	if len(entries) != len(want) {
		t.Fatalf("census %+v", entries)
	}
	for _, e := range entries {
		if e.Sanction != want[e.Rite] {
			t.Errorf("the census tells %s %q, want %q", e.Rite, e.Sanction, want[e.Rite])
		}
		r, err := sc.s.Augur(e.Rite, true)
		if v, ok := r.Value("sanction"); err != nil || r.Sanction != want[e.Rite] || !ok || v != string(want[e.Rite]) {
			t.Errorf("the augury of %s tells %q (%q, %v), want %q", e.Rite, r.Sanction, v, err, want[e.Rite])
		}
	}
	if len(laments) != len(forsakenWords) {
		t.Fatalf("laments %v", laments)
	}
	for i, l := range laments {
		var f *ForsakenSanction
		if !errors.As(l, &f) || l.Error() != forsakenWords[i] {
			t.Errorf("lament %d: %v", i, l)
		}
		grimdark(t, l.Error())
	}

	got, err := json.Marshal(entries[len(entries)-1])
	if err != nil {
		t.Fatal(err)
	}
	shape := `{"rite":"split-lock","edict":false,"aspect":"","standing":"dormant","sanction":"sanctioned",` +
		`"auspex":"","aspects":["on","off"],"purpose":"","desecrated":false,"last_rite":null,` +
		`"recorded_in":"` + filepath.Join(sc.lib, "rites", "split-lock.json") + `","canon":[]}`
	if string(got) != shape {
		t.Errorf("census entry\n got %s\nwant %s", got, shape)
	}
}

// TestSanction_AStoreNeverKeptHoldsNoSanction: without a store, every rite
// with exalted verses stands unsanctioned, and nothing is lamented.
func TestSanction_AStoreNeverKeptHoldsNoSanction(t *testing.T) {
	fx := newFixture(t)
	fx.rite("fresh", exaltedRite("fresh"))
	s := fx.servitor()
	s.Sanctions = &sanction.Store{Root: filepath.Join(t.TempDir(), "never"), UID: 1000}
	entries, laments := s.Census(true)
	if len(entries) != 1 || entries[0].Sanction != sanction.Unsanctioned || len(laments) != 0 {
		t.Fatalf("census %+v %v", entries, laments)
	}
	if list, laments := s.ListSanctions(); len(list) != 0 || len(laments) != 0 {
		t.Fatalf("sanctions %v %v", list, laments)
	}
}

// TestSanction_ListsEverySanctionWithItsStanding: the sanctions of the
// master are listed by rite with their standing, sigil and hour.
func TestSanction_ListsEverySanctionWithItsStanding(t *testing.T) {
	sc := newSanctuary(t)
	list, laments := sc.s.ListSanctions()
	if len(laments) != 0 {
		t.Fatalf("laments %v", laments)
	}
	want := []Sanctioned{
		{"a-gone", sanction.Forsaken, sc.sigil("a-gone"), sanctionedAt},
		{"broken", sanction.Lapsed, sc.sigil("broken"), sanctionedAt},
		{"lapsing", sanction.Lapsed, sc.sigil("lapsing"), sanctionedAt},
		{"plain", sanction.Forsaken, sc.sigil("plain"), sanctionedAt},
		{"split-lock", sanction.Sanctioned, sc.sigil("split-lock"), sanctionedAt},
	}
	if len(list) != len(want) {
		t.Fatalf("listed %+v", list)
	}
	for i := range want {
		if list[i] != want[i] {
			t.Errorf("listed %+v, want %+v", list[i], want[i])
		}
	}
	got, _ := json.Marshal(list[4])
	if shape := `{"rite":"split-lock","standing":"sanctioned","sigil":"` + sc.sigil("split-lock") +
		`","sanctioned_at":"2026-10-12T18:00:00Z"}`; string(got) != shape {
		t.Errorf("binharic %s, want %s", got, shape)
	}
}

// TestSanction_TheInquisitionNotesEveryStandingAmiss: an exalted rite
// unsanctioned or lapsed, and a forsaken sanction, are impurities of the
// machine; --spare-vessels spares them.
func TestSanction_TheInquisitionNotesEveryStandingAmiss(t *testing.T) {
	sc := newSanctuary(t)
	found, err := sc.s.Inquire(nil, false)
	if err != nil {
		t.Fatal(err)
	}
	expect := []struct {
		rite, scripture string
		line, col       int
		words           []string
	}{
		{"fresh", filepath.Join(sc.lib, "rites", "fresh.json"), 2, exaltedAt(sc, "fresh"),
			[]string{"stands unsanctioned", "servitor sanction fresh"}},
		{"lapsing", filepath.Join(sc.lib, "rites", "lapsing.json"), 2, exaltedAt(sc, "lapsing"),
			[]string{"has lapsed", "sigil " + sanction.Short(sc.sigil("lapsing")), "servitor sanction lapsing"}},
		{"a-gone", sc.store.Path("a-gone"), 1, 1, []string{forsakenWords[0]}},
		{"plain", sc.store.Path("plain"), 1, 1, []string{forsakenWords[1]}},
	}
	for _, e := range expect {
		got := denounced(found, e.rite, librarium.Impurity, e.words...)
		if len(got) != 1 {
			t.Errorf("%s %q: found %d in\n%v", e.rite, e.words, len(got), found)
			continue
		}
		if f := got[0]; f.Scripture != e.scripture || f.Line != e.line || f.Column != e.col {
			t.Errorf("%s found at %s:%d:%d, want %s:%d:%d", e.rite, f.Scripture, f.Line, f.Column, e.scripture, e.line, e.col)
		}
		grimdark(t, got[0].Message)
	}
	for _, rite := range []string{"split-lock", "fresh", "lapsing", "plain"} {
		if got := denounced(found, rite, librarium.Impurity); len(got) > 1 {
			t.Errorf("%s is noted more than once: %v", rite, got)
		}
	}
	if got := denounced(found, "split-lock", librarium.Impurity); len(got) != 0 {
		t.Errorf("a sanctioned rite is noted: %v", got)
	}
	if found, _ := sc.s.Inquire([]string{"fresh"}, false); len(denounced(found, "a-gone", librarium.Impurity)) != 0 {
		t.Errorf("the inquisition of one rite notes another's forsaken sanction: %v", found)
	}
	spared, _ := sc.s.Inquire(nil, true)
	for _, rite := range []string{"fresh", "lapsing", "a-gone", "plain"} {
		if got := denounced(spared, rite, librarium.Impurity); len(got) != 0 {
			t.Errorf("--spare-vessels notes %s: %v", rite, got)
		}
	}
}

// exaltedAt is the column of the "exalted" of the first verse of the rite.
func exaltedAt(sc *sanctuary, rite string) int {
	return sc.s.Librarium.Rites[rite].Locate("/liturgy/0/exalted").Column
}

// TestSanction_TheVigilLamentsTheForsakenFirst: the vigil tells every
// forsaken sanction before its first rite, and every rite with its
// standing towards its sanction.
func TestSanction_TheVigilLamentsTheForsakenFirst(t *testing.T) {
	sc := newSanctuary(t)
	var told []Heralding
	sc.s.Vigil(context.Background(), true, func(h Heralding) { told = append(told, h) })
	if len(told) != 2+5 || told[0].Lament == nil || told[0].Rite != "a-gone" || told[1].Rite != "plain" ||
		told[0].Lament.Error() != forsakenWords[0] {
		t.Fatalf("told %+v", told)
	}
	for _, h := range told[2:] {
		if h.Entry == nil || h.Entry.Rite == "split-lock" && h.Entry.Sanction != sanction.Sanctioned {
			t.Errorf("told %+v", h)
		}
	}
}

// TestInvoke_TheRefusalNamesTheStanding: the invocation of an exalted rite
// unsanctioned or lapsed is refused before any proof is asked, and the
// refusal names the standing and the ritual of the sanction; a sanctioned
// rite lent no intercessor is refused too. Nothing is touched.
func TestInvoke_TheRefusalNamesTheStanding(t *testing.T) {
	sc := newSanctuary(t)
	for rite, c := range map[string]struct {
		x     *Exaltation
		words []string
	}{
		"fresh":      {&Exaltation{}, []string{"stands unsanctioned", "servitor sanction fresh"}},
		"lapsing":    {&Exaltation{}, []string{"has lapsed", "servitor sanction lapsing"}},
		"split-lock": {nil, []string{"lent no intercessor", "servitor invoke split-lock on"}},
	} {
		res, err := sc.s.Invoke(context.Background(), Petition{Rite: rite, Aspect: "on", Herald: &herald{},
			Exaltation: c.x})
		var hs invocation.Heresies
		if !errors.As(err, &hs) || len(hs) != 1 || res.Outcome != nil || res.Entry != nil {
			t.Fatalf("%s: %v %+v", rite, err, res)
		}
		for _, w := range append(c.words, "verse 1 is exalted", "not one verse of") {
			if !strings.Contains(hs[0].Message, w) {
				t.Errorf("%s: the refusal lacks %q: %s", rite, w, hs[0].Message)
			}
		}
		grimdark(t, hs[0].Message)
	}
	if exists(filepath.Join(sc.data, "said")) || exists(filepath.Join(sc.state, "servitor")) {
		t.Error("a refused invocation touched the machine")
	}
}
