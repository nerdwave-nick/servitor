package rituals

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/augury"
)

func TestAugur_ReadsTheRiteAndAnswersItsKeys(t *testing.T) {
	fx := newFixture(t)
	fx.vessel("util.kdl", "")
	fx.rite("mouse", mouse)
	s := fx.servitor()
	if _, err := s.Invoke(context.Background(), Petition{Rite: "mouse", Aspect: "on",
		Runes: map[string]string{"reason": "gaming remnant"}}); err != nil {
		t.Fatal(err)
	}

	r, err := s.Augur("mouse", false)

	if err != nil || r.Lament != nil || r.Rite == nil {
		t.Fatalf("augury %+v, %v", r, err)
	}
	if r.Aspect != "on" || r.Standing != augury.Performed || len(r.Omens) != 1 {
		t.Fatalf("augury %+v", r.Augury)
	}
	answers := map[string]string{"aspect": "on", "standing": "performed", "desecrated": "false", "former": "",
		"reason": "gaming remnant"}
	for key, want := range answers {
		if got, ok := r.Value(key); !ok || got != want {
			t.Errorf("%s = %q, %v; want %q", key, got, ok, want)
		}
	}
	if _, ok := r.Value("mood"); ok {
		t.Error("a key the rite knows not was answered")
	}
	if got := Keys(r.Rite); !slices.Equal(got, []string{"aspect", "standing", "desecrated", "former", "reason"}) {
		t.Fatalf("keys %v", got)
	}
}

func TestAugur_HereticalAndUnrecordedRites(t *testing.T) {
	fx := newFixture(t)
	fx.rite("broken", `{"pattern": "Mark I", "aspects": []}`)
	s := fx.servitor()

	r, err := s.Augur("broken", false)
	if err != nil || r.Rite != nil || r.Standing != augury.Heretical {
		t.Fatalf("heretical rite: %+v, %v", r, err)
	}
	if got, ok := r.Value("standing"); !ok || got != "heretical" {
		t.Fatalf("standing %q, %v", got, ok)
	}
	var unrecorded *Unrecorded
	if _, err := s.Augur("nope", false); !errors.As(err, &unrecorded) {
		t.Fatalf("unknown rite: %v", err)
	}
}

func TestAugur_ForgoingTheAuspexAwakensItNot(t *testing.T) {
	fx := newFixture(t)
	fx.rite("dnd", `{"pattern": "Mark I", "aspects": ["on", "off"], "auspex": "touch $DATA/awoken; echo on", "liturgy": []}`)
	s := fx.servitor()
	if r, err := s.Augur("dnd", true); err != nil || r.Aspect != "" || exists(filepath.Join(fx.data, "awoken")) {
		t.Fatalf("forgone auspex: %+v, %v", r, err)
	}
	if r, _ := s.Augur("dnd", false); r.Aspect != "on" {
		t.Fatalf("auspex unheeded: %+v", r)
	}
}

func TestCensus_EveryRiteInTheShapeOfTheCodex(t *testing.T) {
	fx := newFixture(t)
	fx.vessel("util.kdl", "")
	path := fx.rite("mouse", mouse)
	broken := fx.rite("broken", `{"pattern": "Mark I", "aspects": []}`)
	fx.rite("dnd", `{"pattern": "Mark I", "aspects": ["on", "off"], "auspex": "touch $DATA/awoken; echo off", "liturgy": []}`)
	s := fx.servitor()
	if _, err := s.Invoke(context.Background(), Petition{Rite: "mouse", Aspect: "on"}); err != nil {
		t.Fatal(err)
	}

	entries, laments := s.Census(true)

	if len(laments) != 0 || len(entries) != 3 {
		t.Fatalf("census %+v, laments %v", entries, laments)
	}
	if exists(filepath.Join(fx.data, "awoken")) {
		t.Fatal("the census awoke a forgone auspex")
	}
	data, err := json.Marshal(entries)
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"rite":"broken","edict":false,"aspect":"","standing":"heretical","auspex":"","aspects":[],"purpose":"",` +
		`"desecrated":false,"last_rite":null,"recorded_in":"` + broken + `","canon":[]},` +
		`{"rite":"dnd","edict":false,"aspect":"","standing":"dormant","auspex":"forgone","aspects":["on","off"],"purpose":"",` +
		`"desecrated":false,"last_rite":null,"recorded_in":"` + filepath.Join(fx.lib, "rites", "dnd.json") + `","canon":[]},` +
		`{"rite":"mouse","edict":false,"aspect":"on","standing":"performed","auspex":"","aspects":["on","off"],` +
		`"purpose":"Hide the cursor after inactivity","desecrated":false,` +
		`"last_rite":{"verdict":"triumph","at":"2026-10-10T13:30:00Z"},"recorded_in":"` + path + `",` +
		`"canon":[{"key":"reason","purpose":"why the rite was invoked","mandatory":false,"decrees":{"off":"restored"},"decree":null}]}]`
	if string(data) != want {
		t.Fatalf("census\n%s\nwant\n%s", data, want)
	}
	if entries, _ := s.Census(false); entries[1].Aspect != "off" || entries[1].Auspex != augury.Heeded ||
		!exists(filepath.Join(fx.data, "awoken")) {
		t.Fatalf("the census heeded no auspex: %+v", entries[1])
	}
}

func TestCensus_AnEdictBearsNoAspects(t *testing.T) {
	fx := newFixture(t)
	path := fx.rite("capture", capture)
	fx.rite("dormant", `{"pattern": "Mark I", "edict": true, "liturgy": [{"incantation": "true"}]}`)
	s := fx.servitor()
	if _, err := s.Invoke(context.Background(), Petition{Rite: "capture"}); err != nil {
		t.Fatal(err)
	}

	entries, laments := s.Census(true)

	if len(laments) != 0 {
		t.Fatalf("laments %v", laments)
	}
	data, err := json.Marshal(entries)
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"rite":"capture","edict":true,"aspect":"","standing":"proclaimed","auspex":"","aspects":[],` +
		`"purpose":"Capture the visage of the machine","desecrated":false,` +
		`"last_rite":{"verdict":"triumph","at":"2026-10-10T13:30:00Z"},"recorded_in":"` + path + `",` +
		`"canon":[{"key":"region","purpose":"the slurp geometry","mandatory":false,"decrees":{},"decree":"full"}]},` +
		`{"rite":"dormant","edict":true,"aspect":"","standing":"dormant","auspex":"","aspects":[],"purpose":"","desecrated":false,` +
		`"last_rite":null,"recorded_in":"` + filepath.Join(fx.lib, "rites", "dormant.json") + `","canon":[]}]`
	if string(data) != want {
		t.Fatalf("census\n%s\nwant\n%s", data, want)
	}
}

// TestCensus_MarksWhatEveryAuspexGave: every entry tells a heeded auspex
// from a mute one, and both from a rite that keeps none.
func TestCensus_MarksWhatEveryAuspexGave(t *testing.T) {
	fx := newFixture(t)
	fx.rite("heeded", `{"pattern": "Mark I", "aspects": ["on", "off"], "auspex": "echo on", "liturgy": []}`)
	fx.rite("fallen", `{"pattern": "Mark I", "aspects": ["on", "off"], "auspex": "echo on; exit 1", "liturgy": []}`)
	fx.rite("nonsense", `{"pattern": "Mark I", "aspects": ["on", "off"], "auspex": "echo porpl", "liturgy": []}`)
	fx.rite("impatient", `{"pattern": "Mark I", "aspects": ["on", "off"],
		"auspex": {"rite": "sleep 3; echo on", "patience": "100ms"}, "liturgy": []}`)
	fx.rite("silent", `{"pattern": "Mark I", "aspects": ["on", "off"], "liturgy": []}`)

	entries, laments := fx.servitor().Census(false)

	if len(laments) != 0 {
		t.Fatalf("laments %v", laments)
	}
	want := map[string]augury.Heed{"heeded": augury.Heeded, "fallen": augury.Mute, "nonsense": augury.Mute,
		"impatient": augury.Mute, "silent": augury.NoAuspex}
	for _, e := range entries {
		if e.Auspex != want[e.Rite] {
			t.Errorf("%s: auspex %q, want %q", e.Rite, e.Auspex, want[e.Rite])
		}
	}
	if len(entries) != len(want) {
		t.Fatalf("census %+v", entries)
	}
}
