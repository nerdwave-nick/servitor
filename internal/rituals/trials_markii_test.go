package rituals

import (
	"context"
	"maps"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// fanII is a state rite whose machine-spirit holds a scan and a trial: the
// scan prints what the machine shows and leaves "scan" in the log of
// words spoken; the trial of "loud" leaves "trial-loud" there and holds
// only while $DATA/loud-holds stands. "quiet" bears no trial.
const fanII = `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
  {"name": "fan", "designation": "machine-spirit", "omens": {"on": "loud", "*": "quiet"},
   "auspex": {"scan": "echo scan >> $DATA/spoken; cat $DATA/fan",
              "trials": {"loud": "echo trial-$SERVITOR_OMEN >> $DATA/spoken; test -e $DATA/loud-holds"}},
   "enact": {"*": "echo {{omen}} > $DATA/fan"}}]}`

// TestInvokeII_TrialsProveWhatTheScanReads: a machine-spirit whose auspex
// holds both is read by its scan, yet what was set or restored is proven
// by its trials: the trial of the omen commanded must hold, and an omen
// bearing no trial is proven when every trial fails, whatever the scan
// would print. The omen proven is recorded.
func TestInvokeII_TrialsProveWhatTheScanReads(t *testing.T) {
	for _, tc := range []struct {
		name      string
		shows     string // what the machine shows of the fan before
		holds     bool   // whether the trial of "loud" holds
		aspect    string // invoked
		verdict   invocation.Verdict
		heresy    string // the proof that fell
		spoken    string // the words spoken, in order
		fan, omen string // what the machine shows after, and the omen recorded
	}{
		{name: "the trial holds", shows: "quiet", holds: true, aspect: "on", verdict: invocation.Triumph,
			spoken: "scan trial-loud", fan: "loud", omen: "loud"},
		{name: "the trial fails while the scan would print the omen", shows: "quiet", aspect: "on",
			verdict: invocation.Reverted, heresy: `the subject "fan" shows the omen "quiet"; "loud" was commanded`,
			spoken: "scan trial-loud trial-loud", fan: "quiet", omen: "quiet"},
		{name: "an omen without a trial is proven when every trial fails", shows: "loud", aspect: "off",
			verdict: invocation.Triumph, spoken: "scan trial-loud", fan: "quiet", omen: "quiet"},
		{name: "an omen without a trial falls when a trial holds", shows: "loud", holds: true, aspect: "off",
			verdict: invocation.Reverted, heresy: `the subject "fan" shows the omen "loud"; "quiet" was commanded`,
			spoken: "scan trial-loud trial-loud", fan: "loud", omen: "loud"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newFixture(t)
			fx.rite("fan", fanII)
			fx.vessel("fan", tc.shows+"\n")
			if tc.holds {
				fx.vessel("loud-holds", "")
			}

			res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "fan", Aspect: tc.aspect})

			if err != nil || res.Outcome == nil {
				t.Fatalf("invocation: %v %+v", err, res)
			}
			out := res.Outcome
			if out.Verdict != tc.verdict {
				t.Fatalf("verdict %q, want %q (fell %+v)", out.Verdict, tc.verdict, out.Fell)
			}
			if tc.heresy != "" {
				if out.Fell == nil || out.Fell.Subject != "fan" || !strings.Contains(out.Fell.Heresy.Error(), tc.heresy) {
					t.Fatalf("fell %+v, want the heresy %q", out.Fell, tc.heresy)
				}
				grimdark(t, out.Fell.Heresy.Error())
				if len(out.Reversions) != 1 || out.Reversions[0].Heresy != nil {
					t.Fatalf("reversions %+v", out.Reversions)
				}
			}
			if got := strings.Join(strings.Fields(fx.read(filepath.Join(fx.data, "spoken"))), " "); got != tc.spoken {
				t.Errorf("the words spoken were %q, want %q", got, tc.spoken)
			}
			if got := fx.read(filepath.Join(fx.data, "fan")); got != tc.fan+"\n" {
				t.Errorf("the fan shows %q, want %q", got, tc.fan)
			}
			if got, want := recordedOmens(t, fx.slate("fan")), map[string]string{"fan": tc.omen}; !maps.Equal(got, want) {
				t.Errorf("recorded %v, want %v", got, want)
			}
		})
	}
}
