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
	want := `[{"rite":"broken","aspect":"","standing":"heretical","aspects":[],"purpose":"","desecrated":false,` +
		`"last_rite":null,"recorded_in":"` + broken + `"},` +
		`{"rite":"dnd","aspect":"","standing":"dormant","aspects":["on","off"],"purpose":"","desecrated":false,` +
		`"last_rite":null,"recorded_in":"` + filepath.Join(fx.lib, "rites", "dnd.json") + `"},` +
		`{"rite":"mouse","aspect":"on","standing":"performed","aspects":["on","off"],` +
		`"purpose":"Hide the cursor after inactivity","desecrated":false,` +
		`"last_rite":{"verdict":"triumph","at":"2026-10-10T13:30:00Z"},"recorded_in":"` + path + `"}]`
	if string(data) != want {
		t.Fatalf("census\n%s\nwant\n%s", data, want)
	}
	if entries, _ := s.Census(false); entries[1].Aspect != "off" || !exists(filepath.Join(fx.data, "awoken")) {
		t.Fatalf("the census heeded no auspex: %+v", entries[1])
	}
}
