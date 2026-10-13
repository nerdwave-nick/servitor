package invocation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// exaltedState is a state rite of exalted subjects and observances among
// common ones: the exalted machine-spirit smt, whose enact words would
// leave $DATA/marker behind were they ever spoken by the invoker, with the
// common introit prep; the exalted tether theme, with the exalted
// recessional herald, admitted only into "on"; and the common
// machine-spirit boost, whose enact words say what they set.
const exaltedState = `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
  {"name": "smt", "designation": "machine-spirit", "exalted": true, "omens": {"on": "off", "*": "on"},
   "auspex": {"scan": "cat $DATA/smt"}, "enact": {"*": "echo {{omen}} > $DATA/marker"}, "introit": ["prep"]},
  {"name": "theme", "designation": "tether", "exalted": true, "tether": "$DATA/current",
   "omens": {"*": "$DATA/{{aspect}}"}, "recessional": ["herald"]},
  {"name": "boost", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"}, "auspex": {"scan": "cat $DATA/boost"},
   "enact": {"*": "echo {{omen}} > $DATA/boost; echo boost {{omen}} >> $DATA/said"}}],
  "observances": [
  {"name": "prep", "designation": "incantation", "words": "echo prep >> $DATA/said",
   "reversion": "echo unprep >> $DATA/said"},
  {"name": "herald", "designation": "incantation", "exalted": true, "to": ["on"], "words": "touch $DATA/marker"}]}`

// stateRite writes the state rite scripture ($DATA is the vessels' place)
// and loads it.
func (fx fixture) stateRite(t *testing.T, scripture string) *librarium.Rite {
	t.Helper()
	path := write(t, fx.lib, "rites/gaming.json", strings.ReplaceAll(scripture, "$DATA", fx.data), 0o644)
	r, findings := librarium.LoadFile(path)
	if findings.Heretical() {
		t.Fatalf("rite is heretical:\n%v", findings)
	}
	return r
}

// spiritOmens are the aspects each omen of the machine-spirits of
// exaltedState fits.
var spiritOmens = map[string]map[string]string{"smt": {"off": "on", "on": "off"}, "boost": {"1": "on", "0": "off"}}

// reading reads the machine-spirit of exaltedState at index i of r from
// its state file, as an auspex would.
func (fx fixture) reading(r *librarium.Rite, i int) Reading {
	name := r.Subjects[i].Name
	data, err := os.ReadFile(filepath.Join(fx.data, name))
	if err != nil {
		return Reading{}
	}
	omen := strings.TrimSpace(string(data))
	if aspect, ok := spiritOmens[name][omen]; ok {
		return Reading{Omen: omen, Aspects: []string{aspect}}
	}
	return Reading{Omen: omen}
}

// options are those of an invocation of r into aspect, its subjects read
// as they stand.
func (fx fixture) options(r *librarium.Rite, aspect, former string) Options {
	o := Options{Aspect: aspect, Former: former, Reader: func(_ context.Context, i int) Reading {
		return fx.reading(r, i)
	}}
	for i := range r.Subjects {
		o.Readings = append(o.Readings, fx.reading(r, i))
	}
	return o
}

// TestPerformII_ExaltedItemsAreBiddenOfTheExaltedServitor: the exalted
// subjects and observances of a state rite that are performed are bidden
// of the exalted servitor in their places among the common ones — a
// machine-spirit with the omen it is restored to, then proven by the
// invoker's own reading — and the invoker speaks none of their words. They
// are foreseen as exalted. Without an exalted servitor the invocation is
// refused at each exalted item, and nothing is touched.
func TestPerformII_ExaltedItemsAreBiddenOfTheExaltedServitor(t *testing.T) {
	fx := newFixture(t)
	write(t, fx.data, "smt", "on\n", 0o644)
	write(t, fx.data, "boost", "0\n", 0o644)
	r := fx.stateRite(t, exaltedState)
	inv, err := Prepare(r, fx.options(r, "on", "off"))
	if err != nil {
		t.Fatal(err)
	}
	var exalted []string
	for _, f := range inv.Foresee() {
		if f.Exalted {
			exalted = append(exalted, f.Subject+f.Observance)
		}
	}
	if !slices.Equal(exalted, []string{"smt", "theme", "herald"}) || !inv.Exalts() {
		t.Errorf("foreseen exalted %q, exalts %v", exalted, inv.Exalts())
	}
	_, err = inv.PerformExalted(t.Context(), nil)
	var hs Heresies
	if !errors.As(err, &hs) || len(hs) != 3 || hs[0].Position != r.Locate("/subjects/0/exalted") ||
		hs[2].Position != r.Locate("/observances/1/exalted") || !strings.Contains(hs[0].Message, `the subject "smt"`) {
		t.Fatalf("performed without an exalted servitor: %v", err)
	}
	if exists(filepath.Join(fx.data, "said")) || exists(filepath.Join(fx.data, "current")) {
		t.Fatal("a refused invocation touched the machine")
	}

	inv, _ = Prepare(r, fx.options(r, "on", "off"))
	a := &altar{said: filepath.Join(fx.data, "said"), omens: map[string]string{"smt": "off"}, data: fx.data}
	out, err := inv.PerformExalted(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	if out.Verdict != Triumph || !slices.Equal(out.Changed, []string{"smt", "theme", "boost"}) {
		t.Fatalf("outcome %+v", out)
	}
	if want := []string{"enact smt restoring on", "exalted theme", "exalted herald", "finish"}; !slices.Equal(a.calls,
		want) {
		t.Errorf("the exalted servitor was bidden %q, want %q", a.calls, want)
	}
	if got := lines(t, a.said); !slices.Equal(got, []string{"prep", "enact smt restoring on", "exalted theme",
		"exalted herald", "boost 1"}) {
		t.Errorf("said %q", got)
	}
	if exists(filepath.Join(fx.data, "marker")) || exists(filepath.Join(fx.data, "current")) {
		t.Error("the invoker spoke exalted words or bound an exalted tether")
	}
	if known := out.Known["smt"]; known.Omen != "off" {
		t.Errorf("the proof of smt is known as %+v", known)
	}
}

// TestPerformII_NothingExaltedChanging: an invocation in which no exalted
// item changes needs no exalted servitor, and is performed without one.
func TestPerformII_NothingExaltedChanging(t *testing.T) {
	fx := newFixture(t)
	write(t, fx.data, "smt", "off\n", 0o644)
	write(t, fx.data, "boost", "0\n", 0o644)
	if err := os.Symlink(filepath.Join(fx.data, "on"), filepath.Join(fx.data, "current")); err != nil {
		t.Fatal(err)
	}
	r := fx.stateRite(t, exaltedState)
	inv, err := Prepare(r, fx.options(r, "on", "off"))
	if err != nil || inv.Exalts() {
		t.Fatalf("prepared %v, exalts %v", err, inv.Exalts())
	}
	out, err := inv.PerformExalted(t.Context(), nil)
	if err != nil || out.Verdict != Triumph || !slices.Equal(out.Changed, []string{"boost"}) {
		t.Fatalf("%v %+v", err, out)
	}
	if got := lines(t, filepath.Join(fx.data, "said")); !slices.Equal(got, []string{"boost 1"}) {
		t.Errorf("said %q", got)
	}
}

// TestPerformII_AFallRestoresTheExaltedItems: when a common subject falls
// after exalted ones, every item performed is reverted in reverse order,
// the exalted ones by the exalted servitor: an exalted machine-spirit is
// restored to the omen it was read to show, and proven so by the invoker.
func TestPerformII_AFallRestoresTheExaltedItems(t *testing.T) {
	fx := newFixture(t)
	write(t, fx.data, "smt", "on\n", 0o644)
	write(t, fx.data, "boost", "0\n", 0o644)
	r := fx.stateRite(t, strings.Replace(exaltedState, `"enact": {"*": "echo {{omen}} > $DATA/boost;`,
		`"enact": {"1": "echo 7 > $DATA/boost", "*": "echo {{omen}} > $DATA/boost;`, 1))
	inv, err := Prepare(r, fx.options(r, "on", "off"))
	if err != nil {
		t.Fatal(err)
	}
	a := &altar{said: filepath.Join(fx.data, "said"), omens: map[string]string{"smt": "off"}, data: fx.data}
	out, err := inv.PerformExalted(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	var reverted []string
	for _, rv := range out.Reversions {
		reverted = append(reverted, rv.Subject+rv.Observance)
	}
	if out.Verdict != Reverted || !slices.Equal(reverted, []string{"boost", "herald", "theme", "smt", "prep"}) {
		t.Fatalf("verdict %s, reverted %q: %+v", out.Verdict, reverted, out)
	}
	want := []string{"enact smt restoring on", "exalted theme", "exalted herald", "undo exalted herald",
		"undo exalted theme", "restore smt to on", "finish"}
	if !slices.Equal(a.calls, want) {
		t.Errorf("the exalted servitor was bidden %q, want %q", a.calls, want)
	}
	if got := strings.TrimSpace(read(t, filepath.Join(fx.data, "smt"))); got != "on" {
		t.Errorf("smt shows %q once restored", got)
	}
}

// TestPerformII_AnUnreadExaltedSpiritNamesNoOmen: an exalted machine-spirit
// that was not read, invoked from an unknown former aspect, is set naming
// no omen to restore: the exalted servitor cannot restore it by itself.
func TestPerformII_AnUnreadExaltedSpiritNamesNoOmen(t *testing.T) {
	fx := newFixture(t)
	r := fx.stateRite(t, `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "smt", "designation": "machine-spirit", "exalted": true, "enact": {"*": "touch $DATA/marker"}}]}`)
	inv, err := Prepare(r, Options{Aspect: "on"})
	if err != nil {
		t.Fatal(err)
	}
	a := &altar{said: filepath.Join(fx.data, "said")}
	if out, err := inv.PerformExalted(t.Context(), a); err != nil || out.Verdict != Triumph {
		t.Fatalf("%v %+v", err, out)
	}
	if !slices.Equal(a.calls, []string{"enact smt restoring ", "finish"}) {
		t.Errorf("the exalted servitor was bidden %q", a.calls)
	}
}
