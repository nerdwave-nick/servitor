package rituals

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// TestInvoke_APrivilegedAuspexIsNeverSpokenWithoutItsSanction: a state rite
// whose subject bears a privileged auspex stands unsanctioned until its
// master sanctions it: its invocation is refused before anything is read
// or touched, naming the ritual that sanctions it, and nothing is kept.
// Its foresight speaks the common auspexes alone, and tells that the
// privileged one is read at invocation. Sanctioned rites are invoked (see
// privileged_markii_test.go).
func TestInvoke_APrivilegedAuspexIsNeverSpokenWithoutItsSanction(t *testing.T) {
	fx := newFixture(t)
	marker := fx.vessel("marker", "")
	fx.vessel("m", "off\n")
	fx.rite("gaming", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "m", "designation": "machine-spirit", "auspex": {"scan": "cat $DATA/m"},
	   "enact": {"*": "echo {{omen}} >> $DATA/marker"}, "introit": ["herald"]},
	  {"name": "nft", "designation": "machine-spirit", "exalted": true,
	   "auspex": {"scan": "echo touched >> $DATA/marker; echo on", "exalted": true},
	   "enact": {"*": "echo {{omen}} >> $DATA/marker"}}],
	  "observances": [{"name": "herald", "designation": "incantation", "exalted": true,
	    "words": "echo heard >> $DATA/marker"}]}`)
	s := fx.servitor()
	res, err := s.Invoke(context.Background(), Petition{Rite: "gaming", Aspect: "on", Exaltation: &Exaltation{}})
	var hs invocation.Heresies
	if !errors.As(err, &hs) || len(hs) != 2 || res.Outcome != nil || res.Entry != nil {
		t.Fatalf("invoked: %v %+v", err, res)
	}
	for _, h := range hs {
		if !strings.Contains(h.Message, "stands unsanctioned") || !strings.Contains(h.Message, "servitor sanction gaming") {
			t.Errorf("the refusal lacks the sanction: %s", h.Message)
		}
		grimdark(t, h.Message)
	}

	res, err = s.Invoke(context.Background(), Petition{Rite: "gaming", Aspect: "on", Foresee: true})
	if err != nil || res.Outcome != nil || len(res.Foresight) < 2 {
		t.Fatalf("foreseen: %v %+v", err, res)
	}
	var m, nft invocation.Foresight
	for _, f := range res.Foresight {
		switch f.Subject {
		case "m":
			m = f
		case "nft":
			nft = f
		}
	}
	if m.Plan == nil || m.Plan.Read != "off" || !m.Plan.Changing || m.Plan.Privileged {
		t.Errorf("m foreseen %+v", m.Plan)
	}
	if nft.Subject != "nft" || nft.Plan == nil || !nft.Plan.Privileged || nft.Plan.Stale || nft.Plan.Mute ||
		!nft.Plan.Changing || !nft.Exalted {
		t.Errorf("nft foreseen %+v %+v", nft, nft.Plan)
	}
	if got := fx.read(marker); got != "" {
		t.Fatalf("the rite was spoken or touched: %q", got)
	}
	if _, err := os.Stat(fx.state + "/servitor"); err == nil {
		t.Fatal("a refused or foreseen invocation kept something in the state home")
	}
}
