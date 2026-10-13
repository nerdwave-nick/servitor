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
	"github.com/nerdwave-nick/servitor/internal/exalted"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// captureII is the edict of the captured visage (examples/rites/screenshot.json)
// written in pattern Mark II, its words leaving their marks beside the
// vessels each time it is proclaimed.
const captureII = `{
  "pattern": "Mark II",
  "purpose": "Capture the visage of the machine",
  "edict": true,
  "inscriptions": {"region": {"purpose": "\"chosen\" or \"whole\"", "decrees": "chosen"}},
  "vox-cast": ["success"],
  "liturgy": [
    {"designation": "incantation", "patience": "2m",
     "words": "echo {{inscription.region}}-${SERVITOR_ASPECT-none} >> $DATA/said",
     "reversion": "echo struck >> $DATA/said"},
    {"designation": "incantation", "words": "echo copied >> $DATA/said"}
  ]
}`

func TestInvokeII_ProclaimsAnEdictWholeAtEveryInvocation(t *testing.T) {
	fx := newFixture(t)
	fx.rite("capture", captureII)
	slatePath := augury.SlatePath(os.LookupEnv, "capture")
	h := &herald{}

	first, err := fx.servitor().Invoke(context.Background(),
		Petition{Rite: "capture", Runes: map[string]string{"region": "whole"}, Herald: h})
	if err != nil || first.Outcome == nil || first.Outcome.Verdict != invocation.Triumph || len(first.Laments) != 0 {
		t.Fatalf("first proclaiming: %v %+v", err, first)
	}
	second, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "capture"})
	if err != nil || second.Outcome == nil || second.Outcome.Verdict != invocation.Triumph || len(second.Laments) != 0 {
		t.Fatalf("second proclaiming: %v %+v", err, second)
	}

	if got := fx.read(filepath.Join(fx.data, "said")); got != "whole-none\ncopied\nchosen-none\ncopied\n" {
		t.Fatalf("the edict said %q", got)
	}
	if !slices.Equal(h.heard, []invocation.Tidings{invocation.Success}) {
		t.Fatalf("the herald heard %v", h.heard)
	}
	if o := second.Options; !o.Edict || o.Aspect != "" || o.Former != "" || o.Inscriptions["region"] != "chosen" ||
		o.Recorded["region"] != "whole" {
		t.Fatalf("options %+v", o)
	}
	if want := `{"inscriptions":{"region":"chosen"},"last_rite":{"verdict":"triumph","at":"2026-10-10T13:30:00Z"}}` + "\n"; fx.read(slatePath) != want {
		t.Fatalf("data-slate\n got %s\nwant %s", fx.read(slatePath), want)
	}
	lines := strings.Split(strings.TrimSpace(fx.read(filepath.Join(fx.state, "servitor", "chronicle.jsonl"))), "\n")
	want := []string{
		`{"at":"2026-10-10T13:30:00Z","rite":"capture","edict":true,"inscriptions":{"region":"whole"},"verdict":"triumph"}`,
		`{"at":"2026-10-10T13:30:00Z","rite":"capture","edict":true,"inscriptions":{"region":"chosen"},"verdict":"triumph"}`,
	}
	if !slices.Equal(lines, want) {
		t.Fatalf("chronicle\n got %q\nwant %q", lines, want)
	}
}

func TestInvokeII_AFallenEdictIsRevertedKeptAndChronicled(t *testing.T) {
	fx := newFixture(t)
	fx.rite("capture", strings.Replace(captureII, "echo copied >> $DATA/said", "exit 3", 1))
	h := &herald{}

	res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "capture", Herald: h})

	if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Reverted {
		t.Fatalf("%v %+v", err, res)
	}
	if got := fx.read(filepath.Join(fx.data, "said")); got != "chosen-none\nstruck\n" {
		t.Fatalf("the edict said %q", got)
	}
	if !slices.Equal(h.heard, []invocation.Tidings{invocation.Failure}) {
		t.Fatalf("the herald heard %v", h.heard)
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
	if r, err := fx.servitor().Augur("capture", false); err != nil || r.Standing != augury.Proclaimed {
		t.Fatalf("augury %+v, %v", r, err)
	}
}

func TestInvokeII_AnEdictBearsNoAspectAndForeseeingPerformsNothing(t *testing.T) {
	fx := newFixture(t)
	fx.rite("capture", captureII)

	_, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "capture", Aspect: "taken"})
	var hs invocation.Heresies
	if !errors.As(err, &hs) || !strings.Contains(err.Error(), `the edict "capture" bears no aspect`) {
		t.Fatalf("an aspect upon an edict was not refused: %v", err)
	}
	res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "capture", Foresee: true})
	if err != nil || res.Outcome != nil || len(res.Foresight) != 2 || res.Foresight[0].Speech == nil ||
		res.Foresight[1].Number != 2 {
		t.Fatalf("foresight %v %+v", err, res)
	}
	if exists(filepath.Join(fx.data, "said")) || fx.slate("capture") != nil || len(fx.chronicled()) != 0 {
		t.Fatal("a refused or foreseen edict touched the machine, its data-slate or the chronicle")
	}
}

func TestAugurII_AnEdictAnswersItsStandingAndInscriptionsAlone(t *testing.T) {
	fx := newFixture(t)
	fx.rite("capture", captureII)
	s := fx.servitor()
	if r, err := s.Augur("capture", false); err != nil || r.Standing != augury.Dormant {
		t.Fatalf("augury before any proclaiming %+v, %v", r, err)
	}
	if _, err := s.Invoke(context.Background(), Petition{Rite: "capture", Runes: map[string]string{"region": "whole"}}); err != nil {
		t.Fatal(err)
	}
	r, err := s.Augur("capture", false)
	if err != nil || !r.Edict || r.Standing != augury.Proclaimed {
		t.Fatalf("augury %+v, %v", r, err)
	}
	if got := Keys(r.Rite); !slices.Equal(got, []string{"standing", "sanction", "region"}) {
		t.Errorf("an edict's keys are %v", got)
	}
	for key, want := range map[string]string{"standing": "proclaimed", "region": "whole"} {
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

// rebootII is an edict of pattern Mark II with a common entry and an
// exalted one.
const rebootII = `{"pattern": "Mark II", "edict": true, "vox-cast": ["progress", "success"], "liturgy": [
  {"designation": "incantation", "words": "echo common >> $DATA/said"},
  {"designation": "incantation", "words": "echo exalted >> $DATA/said", "exalted": true}]}`

// TestInvokeII_AnExaltedEdictIsProclaimedByTheExaltedServitor: the exalted
// entry of an edict of Mark II is performed by the exalted servitor alone,
// from the sanction of its master, roused with no aspect; the common entry
// by the invoker's servitor.
func TestInvokeII_AnExaltedEdictIsProclaimedByTheExaltedServitor(t *testing.T) {
	tp := newTemple(t, "pkexec")
	tp.rite("reboot", rebootII)
	kept := tp.sanctify("reboot")
	h := &teller{}

	res, err := tp.invoke("reboot", "", h)

	if err != nil || res.Outcome.Verdict != invocation.Triumph || res.Sanction != sanction.Sanctioned {
		t.Fatalf("%v %+v", err, res)
	}
	if said := tp.said(); !slices.Equal(said, []string{"common", "exalted"}) {
		t.Errorf("said %q", said)
	}
	want := []string{filepath.Join(tp.bin, "pkexec"), "/usr/bin/servitor", exalted.Mode, "reboot",
		sanction.Short(kept.Sigil)}
	if calls := tp.called(); len(calls) != 1 || !slices.Equal(calls[0], want) {
		t.Errorf("called %q, want %q", calls, want)
	}
	if got := tp.journal.deeds(); !slices.Equal(got, []string{"accept accepted", "perform 2 triumph",
		"finish finished"}) {
		t.Errorf("journaled %q", got)
	}
	var heard []invocation.Tidings
	for _, p := range h.told {
		heard = append(heard, p.Tidings)
	}
	if !slices.Equal(heard, []invocation.Tidings{invocation.Exaltation, invocation.Progress, invocation.Progress,
		invocation.Success}) {
		t.Errorf("heard %v", heard)
	}
	if entries := tp.chronicled(); len(entries) != 1 || !entries[0].Edict || entries[0].Intercessor != "pkexec" {
		t.Errorf("chronicled %+v", entries)
	}
}

// TestInvokeII_AnExaltedEdictIsNeverPerformedWithoutItsSanction: an edict
// of Mark II whose exalted entry was never sanctioned, or whose sanction
// lapsed, or that is lent no intercessor, is refused before anything is
// touched; nothing of it is spoken with the invoker's authority.
func TestInvokeII_AnExaltedEdictIsNeverPerformedWithoutItsSanction(t *testing.T) {
	for _, c := range []struct {
		name  string
		lay   func(tp *temple)
		p     Petition
		words string
	}{
		{name: "never sanctioned", p: Petition{Rite: "reboot", Exaltation: &Exaltation{}}, words: "unsanctioned"},
		{name: "lapsed", lay: func(tp *temple) {
			tp.sanctify("reboot")
			tp.rite("reboot", strings.Replace(rebootII, "echo exalted", "echo exalted anew", 1))
		}, p: Petition{Rite: "reboot", Exaltation: &Exaltation{}}, words: "lapsed"},
		{name: "no exaltation lent", lay: func(tp *temple) { tp.sanctify("reboot") }, p: Petition{Rite: "reboot"},
			words: "no intercessor"},
	} {
		t.Run(c.name, func(t *testing.T) {
			tp := newTemple(t, "pkexec")
			tp.rite("reboot", rebootII)
			if c.lay != nil {
				c.lay(tp)
			}

			res, err := tp.servitor().Invoke(context.Background(), c.p)

			if err == nil || res.Outcome != nil || !strings.Contains(err.Error(), c.words) {
				t.Fatalf("err %v\nresult %+v", err, res)
			}
			grimdark(t, err.Error())
			if said := tp.said(); said != nil || len(tp.called()) != 0 {
				t.Errorf("touched: said %q, called %q", said, tp.called())
			}
			if exists(filepath.Join(tp.state, "servitor")) {
				t.Error("a refused edict kept something in the state home")
			}
		})
	}
}
