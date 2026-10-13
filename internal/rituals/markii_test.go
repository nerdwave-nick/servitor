package rituals

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

// TestInvoke_RefusesEveryStateRiteOfMarkII: the servitor reads scripture
// of pattern Mark II, yet performs no rite of aspects of it, nor foresees
// it: no subject or observance is touched, and nothing is kept on the
// data-slate or in the chronicle. Edicts of Mark II are proclaimed (see
// edict_markii_test.go).
func TestInvoke_RefusesEveryStateRiteOfMarkII(t *testing.T) {
	fx := newFixture(t)
	marker := fx.vessel("marker", "")
	fx.rite("gaming", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "m", "designation": "machine-spirit", "auspex": {"scan": "echo touched >> $DATA/marker; cat $DATA/m"},
	   "enact": {"*": "echo {{omen}} >> $DATA/marker"}, "introit": ["herald"]}],
	  "observances": [{"name": "herald", "designation": "incantation", "words": "echo heard >> $DATA/marker"}]}`)
	s := fx.servitor()
	for _, p := range []Petition{{Rite: "gaming", Aspect: "on"}, {Rite: "gaming", Aspect: "off", Foresee: true}} {
		res, err := s.Invoke(context.Background(), p)
		var refused *Unperformable
		if !errors.As(err, &refused) || refused.Name != p.Rite {
			t.Fatalf("%+v: error %v, want the lament of a rite not yet performed", p, err)
		}
		if res.Outcome != nil || res.Entry != nil || len(res.Foresight) != 0 {
			t.Fatalf("%+v: result %+v", p, res)
		}
		msg := err.Error()
		for _, want := range []string{`"` + p.Rite + `"`, `pattern "Mark II"`, "rite of performance", "not yet forged"} {
			if !strings.Contains(msg, want) {
				t.Errorf("%+v: the lament lacks %q: %s", p, want, msg)
			}
		}
	}
	if got := fx.read(marker); got != "" {
		t.Fatalf("a rite of Mark II was touched: %q", got)
	}
	if _, err := os.Stat(fx.state + "/servitor"); err == nil {
		t.Fatal("a refused invocation kept something in the state home")
	}
}
