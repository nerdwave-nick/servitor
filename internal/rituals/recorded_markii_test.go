package rituals

import (
	"context"
	"maps"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// recordedOmens are the omens a data-slate recorded, by subject, as omen
// and the minute after noon it was read at.
func recordedOmens(t *testing.T, sl *augury.Slate) map[string]string {
	t.Helper()
	if sl == nil {
		t.Fatal("no data-slate")
	}
	if sl.Boot == "" || sl.Boot != augury.Boot() {
		t.Errorf("the data-slate's boot is %q, want %q", sl.Boot, augury.Boot())
	}
	got := map[string]string{}
	for name, o := range sl.Omens {
		got[name] = o.Omen
		if o.Exalted {
			got[name] += " exalted"
		}
	}
	return got
}

// TestInvokeII_RecordsTheOmensKnownAtItsEnd: at the end of every invocation
// of a state rite the data-slate records, with the boot, the omen of every
// machine-spirit whose last deed was a reading: an unchanged subject's, a
// changed subject's proof, or the proof of its restoration. Built-in
// subjects are never recorded.
func TestInvokeII_RecordsTheOmensKnownAtItsEnd(t *testing.T) {
	t.Run("triumph", func(t *testing.T) {
		fx := newFixture(t)
		fx.rite("gaming", gamingII)
		gamingOff(fx)
		s := fx.servitor()

		if res, err := s.Invoke(context.Background(), Petition{Rite: "gaming", Aspect: "on"}); err != nil ||
			res.Outcome.Verdict != invocation.Triumph {
			t.Fatalf("invocation: %v %+v", err, res.Outcome)
		}
		want := map[string]string{"smt": "off", "scheduler": "cosmos"}
		if got := recordedOmens(t, fx.slate("gaming")); !maps.Equal(got, want) {
			t.Fatalf("recorded %v, want %v", got, want)
		}
		// nothing changing: the readings before are recorded
		if res, err := s.Invoke(context.Background(), Petition{Rite: "gaming", Aspect: "on"}); err != nil ||
			len(res.Outcome.Changed) != 0 {
			t.Fatalf("second invocation: %v %+v", err, res.Outcome)
		}
		if got := recordedOmens(t, fx.slate("gaming")); !maps.Equal(got, want) {
			t.Fatalf("recorded %v, want %v", got, want)
		}
		if at := fx.slate("gaming").Omens["smt"].At; !at.Equal(time.Date(2026, 10, 10, 13, 30, 0, 0, time.UTC)) {
			t.Errorf("smt was recorded at %v", at)
		}
	})
	t.Run("reverted", func(t *testing.T) {
		fx := newFixture(t)
		fx.rite("power", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
		  {"name": "governor", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"},
		   "auspex": {"scan": "cat $DATA/governor"}, "enact": {"*": "echo {{omen}} > $DATA/governor"}},
		  {"name": "boost", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"},
		   "auspex": {"scan": "cat $DATA/boost"},
		   "enact": {"1": "echo 2 > $DATA/boost", "*": "echo {{omen}} > $DATA/boost"}}]}`)
		fx.vessel("governor", "0\n")
		fx.vessel("boost", "0\n")

		res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "power", Aspect: "on"})

		if err != nil || res.Outcome.Verdict != invocation.Reverted {
			t.Fatalf("invocation: %v %+v", err, res.Outcome)
		}
		want := map[string]string{"governor": "0", "boost": "0"} // the proofs of their restorations
		if got := recordedOmens(t, fx.slate("power")); !maps.Equal(got, want) {
			t.Fatalf("recorded %v, want %v", got, want)
		}
	})
	t.Run("faltered", func(t *testing.T) {
		fx := newFixture(t)
		// "a" is restored to an omen it does not show; "u" has no auspex;
		// "z" falls when set into on; "w" is never reached; "m" is mute.
		fx.rite("faltering", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
		  {"name": "a", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"},
		   "auspex": {"scan": "cat $DATA/a"}, "enact": {"1": "echo 1 > $DATA/a", "*": "echo 9 > $DATA/a"}},
		  {"name": "keep", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"},
		   "auspex": {"scan": "cat $DATA/keep"}, "enact": {"*": "echo {{omen}} > $DATA/keep"}},
		  {"name": "u", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"},
		   "enact": {"*": "echo {{omen}} > $DATA/u"}},
		  {"name": "z", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"},
		   "auspex": {"scan": "cat $DATA/z"}, "enact": {"1": "exit 3", "*": "echo {{omen}} > $DATA/z"}},
		  {"name": "w", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"},
		   "auspex": {"scan": "cat $DATA/w"}, "enact": {"*": "echo {{omen}} > $DATA/w"}},
		  {"name": "m", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"},
		   "auspex": {"scan": "cat $DATA/m"}, "enact": {"*": "true"}}]}`)
		for _, name := range []string{"a", "keep", "z", "w"} {
			fx.vessel(name, "0\n")
		}
		fx.vessel("m", "0\n")
		s := fx.servitor()
		if res, err := s.Invoke(context.Background(), Petition{Rite: "faltering", Aspect: "off"}); err != nil ||
			res.Outcome.Verdict != invocation.Triumph {
			t.Fatalf("first invocation: %v %+v", err, res.Outcome)
		}
		if got, want := recordedOmens(t, fx.slate("faltering")), map[string]string{"a": "0", "keep": "0", "z": "0",
			"w": "0", "m": "0"}; !maps.Equal(got, want) {
			t.Fatalf("first: recorded %v, want %v", got, want)
		}
		fx.vessel("keep", "1\n") // other hands: the rite is corrupted, so "u" cannot be restored
		fx.vessel("m", "mute\n")

		res, err := s.Invoke(context.Background(), Petition{Rite: "faltering", Aspect: "on"})

		if err != nil || res.Outcome.Verdict != invocation.Faltered || res.Outcome.Fell.Subject != "z" {
			t.Fatalf("invocation: %v %+v", err, res.Outcome)
		}
		// a: its restoration unproven, struck; keep: unchanged, its reading
		// before; u: no auspex; z: restored and proven; w: never reached,
		// its reading before; m: mute, struck.
		want := map[string]string{"keep": "1", "z": "0", "w": "0"}
		if got := recordedOmens(t, fx.slate("faltering")); !maps.Equal(got, want) {
			t.Fatalf("recorded %v, want %v", got, want)
		}
	})
}

// ticking is a clock that advances a minute every time it is asked.
type ticking struct {
	mu  sync.Mutex
	now time.Time
}

func (c *ticking) tick() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(time.Minute)
	return c.now
}

// TestInvokeII_RecordsEachOmenWhenItWasRead: an omen is recorded at the
// moment it was read: the reading before for an unchanged subject, the
// proof for a subject set, both before the invocation ended.
func TestInvokeII_RecordsEachOmenWhenItWasRead(t *testing.T) {
	fx := newFixture(t)
	fx.rite("gaming", gamingII)
	gamingOff(fx)
	fx.vessel("scheduler", "cosmos\n") // already in the aspect on
	s := fx.servitor()
	clock := &ticking{now: time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)}
	s.Now = clock.tick

	if res, err := s.Invoke(context.Background(), Petition{Rite: "gaming", Aspect: "on"}); err != nil ||
		res.Outcome.Verdict != invocation.Triumph {
		t.Fatalf("invocation: %v %+v", err, res.Outcome)
	}

	sl := fx.slate("gaming")
	unchanged, set, ended := sl.Omens["scheduler"].At, sl.Omens["smt"].At, sl.LastRite.At
	if !unchanged.Before(set) || !set.Before(ended) {
		t.Fatalf("scheduler read at %v, smt proven at %v, the invocation ended at %v", unchanged, set, ended)
	}
	if !strings.HasSuffix(unchanged.Format(time.RFC3339), "Z") {
		t.Errorf("not in UTC: %v", unchanged)
	}
}
