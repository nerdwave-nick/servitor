package rituals

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// capture is an edict that leaves its words beside the vessels each time it
// is proclaimed.
const capture = `{
  "pattern": "Mark I",
  "purpose": "Capture the visage of the machine",
  "edict": true,
  "inscriptions": {"region": {"purpose": "the slurp geometry", "decrees": "full"}},
  "liturgy": [
    {"incantation": "echo {{inscription.region}}-${SERVITOR_ASPECT-none} >> $DATA/said"},
    {"vox-cast": "success"}
  ]
}`

func TestInvoke_ProclaimsAnEdictWholeAtEveryInvocation(t *testing.T) {
	fx := newFixture(t)
	fx.rite("capture", capture)
	slatePath := augury.SlatePath(os.LookupEnv, "capture")
	h := &herald{}

	first, err := fx.servitor().Invoke(context.Background(),
		Petition{Rite: "capture", Runes: map[string]string{"region": "window"}, Herald: h})
	if err != nil || first.Outcome == nil || first.Outcome.Verdict != invocation.Triumph || len(first.Laments) != 0 {
		t.Fatalf("first proclaiming: %v %+v", err, first)
	}
	second, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "capture"})
	if err != nil || second.Outcome == nil || second.Outcome.Verdict != invocation.Triumph || len(second.Laments) != 0 {
		t.Fatalf("second proclaiming: %v %+v", err, second)
	}

	if got := fx.read(filepath.Join(fx.data, "said")); got != "window-none\nfull-none\n" {
		t.Fatalf("the edict said %q", got)
	}
	if !slices.Equal(h.heard, []invocation.Tidings{invocation.Success}) {
		t.Fatalf("the herald heard %v", h.heard)
	}
	if o := second.Options; !o.Edict || o.Aspect != "" || o.Former != "" || o.Inscriptions["region"] != "full" ||
		o.Recorded["region"] != "window" {
		t.Fatalf("options %+v", o)
	}
	if want := `{"inscriptions":{"region":"full"},"last_rite":{"verdict":"triumph","at":"2026-10-10T13:30:00Z"}}` + "\n"; fx.read(slatePath) != want {
		t.Fatalf("data-slate\n got %s\nwant %s", fx.read(slatePath), want)
	}
	lines := strings.Split(strings.TrimSpace(fx.read(filepath.Join(fx.state, "servitor", "chronicle.jsonl"))), "\n")
	want := []string{
		`{"at":"2026-10-10T13:30:00Z","rite":"capture","edict":true,"inscriptions":{"region":"window"},"verdict":"triumph"}`,
		`{"at":"2026-10-10T13:30:00Z","rite":"capture","edict":true,"inscriptions":{"region":"full"},"verdict":"triumph"}`,
	}
	if !slices.Equal(lines, want) {
		t.Fatalf("chronicle\n got %q\nwant %q", lines, want)
	}
}

func TestInvoke_AFallenEdictIsRevertedKeptAndChronicled(t *testing.T) {
	fx := newFixture(t)
	fx.rite("capture", `{"pattern": "Mark I", "edict": true, "liturgy": [
	  {"incantation": "echo spoke >> $DATA/said", "reversion": "echo unspoke >> $DATA/said"},
	  {"incantation": "exit 3"}]}`)

	res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "capture"})

	if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Reverted {
		t.Fatalf("%v %+v", err, res)
	}
	if got := fx.read(filepath.Join(fx.data, "said")); got != "spoke\nunspoke\n" {
		t.Fatalf("the edict said %q", got)
	}
	slate := fx.read(augury.SlatePath(os.LookupEnv, "capture"))
	if want := `{"inscriptions":{},"last_rite":{"verdict":"reverted","at":"2026-10-10T13:30:00Z"}}` + "\n"; slate != want {
		t.Fatalf("data-slate\n got %s\nwant %s", slate, want)
	}
	entries := fx.chronicled()
	if len(entries) != 1 || !entries[0].Edict || entries[0].Aspect != "" || entries[0].Verdict != invocation.Reverted ||
		entries[0].FellAt == nil || entries[0].FellAt.Number != 2 {
		t.Fatalf("chronicled %+v", entries)
	}
}

func TestInvoke_AnEdictBearsNoAspectAndForeseeingPerformsNothing(t *testing.T) {
	fx := newFixture(t)
	fx.rite("capture", capture)

	_, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "capture", Aspect: "taken"})
	var hs invocation.Heresies
	if !errors.As(err, &hs) || !strings.Contains(err.Error(), `the edict "capture" bears no aspect`) {
		t.Fatalf("an aspect upon an edict was not refused: %v", err)
	}
	res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "capture", Foresee: true})
	if err != nil || res.Outcome != nil || len(res.Foresight) != 2 || res.Foresight[0].Speech == nil {
		t.Fatalf("foresight %v %+v", err, res)
	}
	if exists(filepath.Join(fx.data, "said")) || fx.slate("capture") != nil || len(fx.chronicled()) != 0 {
		t.Fatal("a refused or foreseen edict touched the machine, its data-slate or the chronicle")
	}
}

func TestAugur_AnEdictAnswersItsStandingAndInscriptionsAlone(t *testing.T) {
	fx := newFixture(t)
	fx.rite("capture", capture)
	s := fx.servitor()
	if _, err := s.Invoke(context.Background(), Petition{Rite: "capture", Runes: map[string]string{"region": "window"}}); err != nil {
		t.Fatal(err)
	}
	r, err := s.Augur("capture", false)
	if err != nil || !r.Edict || r.Standing != augury.Proclaimed {
		t.Fatalf("augury %+v, %v", r, err)
	}
	if got := Keys(r.Rite); !slices.Equal(got, []string{"standing", "region"}) {
		t.Errorf("an edict's keys are %v", got)
	}
	for key, want := range map[string]string{"standing": "proclaimed", "region": "window"} {
		if v, ok := r.Value(key); !ok || v != want {
			t.Errorf("%s = %q, %v; want %q", key, v, ok, want)
		}
	}
	for _, key := range []string{"aspect", "desecrated", "former"} {
		if v, ok := r.Value(key); ok {
			t.Errorf("an edict answers %s = %q", key, v)
		}
	}
}
