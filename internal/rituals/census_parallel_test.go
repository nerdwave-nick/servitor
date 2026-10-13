package rituals

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/augury"
)

// sequentialCensus is the census read one rite after another, by name: the
// measure the census must keep however it gathers its readings.
func sequentialCensus(s *Servitor, forgoAuspex bool) (entries []Entry, laments []error) {
	entries = []Entry{}
	for _, name := range s.Librarium.Names() {
		r, err := s.Augur(name, forgoAuspex)
		if err != nil {
			laments = append(laments, err)
			continue
		}
		if r.Lament != nil {
			laments = append(laments, r.Lament)
		}
		entries = append(entries, s.entry(name, r))
	}
	return entries, laments
}

// TestCensus_EqualsASequentialRun: however the auspexes are gathered, the
// census holds every rite by name with the values a sequential run reads,
// and laments the unreadable data-slates in the order of their rites.
func TestCensus_EqualsASequentialRun(t *testing.T) {
	fx := newFixture(t)
	fx.vessel("util.kdl", "")
	fx.rite("mouse", mouse)
	fx.rite("capture", captureII)
	fx.rite("broken", broken)
	fx.rite("fallen", spirit(`{"scan": "exit 1"}`))
	fx.rite("nonsense", spirit(`{"scan": "echo porpl"}`))
	// The slower an auspex, the earlier its rite in the census: they settle
	// against the order of their names.
	for i, nap := range []string{"0.4", "0.3", "0.2", "0.1", "0"} {
		aspect := []string{"on", "off"}[i%2]
		fx.rite(fmt.Sprintf("slow%d", i),
			spirit(`{"scan": "sleep `+nap+`; echo `+aspect+`"}`))
	}
	s := fx.servitor()
	if _, err := s.Invoke(context.Background(), Petition{Rite: "mouse", Aspect: "on"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Invoke(context.Background(), Petition{Rite: "slow1", Aspect: "on"}); err != nil {
		t.Fatal(err)
	}
	for _, rite := range []string{"fallen", "broken", "slow3"} {
		fx.write(fx.state, filepath.Join("servitor", augury.SlatesDir, rite+".json"), "{")
	}

	for _, forgo := range []bool{false, true} {
		entries, laments := s.Census(forgo)
		wantEntries, wantLaments := sequentialCensus(s, forgo)

		if len(entries) != 10 || entries[5].Rite != "slow0" || entries[5].Auspex != augury.Heeded && !forgo {
			t.Errorf("forgo %v: census %+v", forgo, entries)
		}
		got, want := marshal(t, entries), marshal(t, wantEntries)
		if got != want {
			t.Errorf("forgo %v: census\n%s\nwant\n%s", forgo, got, want)
		}
		if got, want := fmt.Sprint(laments), fmt.Sprint(wantLaments); got != want || len(laments) != 3 {
			t.Errorf("forgo %v: laments\n%s\nwant\n%s", forgo, got, want)
		}
	}
}

func marshal(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
