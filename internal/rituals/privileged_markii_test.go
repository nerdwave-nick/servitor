package rituals

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// firewall is a state rite whose exalted machine-spirit nft only the root
// can read: its privileged scan tells it was spoken in $DATA/marker and
// prints what $DATA/nft holds; its enact words take no effect while
// $DATA/stubborn stands. Beside it stand the exalted recessional herald,
// admitted only into "on", and the common machine-spirit boost, whose
// enact words into "1" fall while $DATA/fall stands.
const firewall = `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
  {"name": "nft", "designation": "machine-spirit", "exalted": true, "omens": {"on": "gaming", "*": "none"},
   "auspex": {"scan": "echo read >> $DATA/marker; cat $DATA/nft", "exalted": true},
   "enact": {"*": "[ -e $DATA/stubborn ] || echo {{omen}} > $DATA/nft; echo \"enact {{omen}}\" >> $DATA/said"},
   "recessional": ["herald"]},
  {"name": "boost", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"}, "auspex": {"scan": "cat $DATA/boost"},
   "enact": {"1": "test ! -e $DATA/fall && echo 1 > $DATA/boost && echo boost 1 >> $DATA/said",
             "*": "echo {{omen}} > $DATA/boost && echo boost {{omen}} >> $DATA/said"}}],
  "observances": [
  {"name": "herald", "designation": "incantation", "exalted": true, "to": ["on"], "words": "echo herald >> $DATA/said"}]}`

// firewallAs lays the machine of firewall: what nft and boost hold.
func (tp *temple) firewallAs(nft, boost string) {
	tp.fixture.vessel("nft", nft+"\n")
	tp.fixture.vessel("boost", boost+"\n")
}

// marked is how often the privileged auspex of nft was spoken.
func (tp *temple) marked() int {
	p := filepath.Join(tp.data, "marker")
	if !exists(p) {
		return 0
	}
	return strings.Count(tp.read(p), "read\n")
}

// acts names every act the exalted servitor journaled: deed, item and
// verdict.
func (l *ledger) acts() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []string
	for _, e := range l.entries {
		out = append(out, strings.Join(strings.Fields(e.Deed+" "+e.Subject+e.Observance+" "+string(e.Verdict)), " "))
	}
	return out
}

// TestInvokeII_AnUnchangedPrivilegedSubjectIsSkipped: a rite holding a
// privileged auspex always raises the exalted servitor, which reads it
// before anything is performed; a subject already showing the omen of the
// aspect is then skipped, and so is its recessional. Its omen is recorded
// on the data-slate as read by the exalted servitor.
func TestInvokeII_AnUnchangedPrivilegedSubjectIsSkipped(t *testing.T) {
	tp := newTemple(t, "sudo")
	tp.rite("firewall", firewall)
	tp.firewallAs("gaming", "1")
	tp.sanctify("firewall")
	res, err := tp.invoke("firewall", "on", nil)
	if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Triumph {
		t.Fatalf("%v %+v", err, res.Outcome)
	}
	if len(res.Outcome.Changed) != 0 || tp.said() != nil || tp.marked() != 1 {
		t.Errorf("changed %q, said %q, the scan spoken %d times", res.Outcome.Changed, tp.said(), tp.marked())
	}
	if got := tp.journal.acts(); !slices.Equal(got, []string{"accept accepted", "read nft beheld", "finish finished"}) {
		t.Errorf("journaled %q", got)
	}
	if n := len(tp.called()); n != 1 || res.Intercessor != "sudo" {
		t.Errorf("the intercessor was called upon %d times (%q)", n, res.Intercessor)
	}
	if res.Foresight[0].Plan == nil || res.Foresight[0].Plan.Read != "gaming" || res.Foresight[0].Plan.Changing {
		t.Errorf("nft was planned %+v", res.Foresight[0].Plan)
	}
	slate := tp.slate("firewall")
	if got := slate.Omens["nft"]; got.Omen != "gaming" || !got.Exalted || got.At.IsZero() {
		t.Errorf("the data-slate records nft as %+v", got)
	}
	if got := slate.Omens["boost"]; got.Omen != "1" || got.Exalted {
		t.Errorf("the data-slate records boost as %+v", got)
	}
}

// TestInvokeII_AChangedPrivilegedSubjectIsProvenByTheExaltedServitor: a
// privileged subject that shows another omen is set by the exalted
// servitor, which proves it by its own reading; its recessional follows,
// and the proof is recorded.
func TestInvokeII_AChangedPrivilegedSubjectIsProvenByTheExaltedServitor(t *testing.T) {
	tp := newTemple(t, "sudo")
	tp.rite("firewall", firewall)
	tp.firewallAs("none", "0")
	tp.sanctify("firewall")
	res, err := tp.invoke("firewall", "on", nil)
	if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Triumph {
		t.Fatalf("%v %+v", err, res.Outcome)
	}
	if got := tp.said(); !slices.Equal(got, []string{"enact gaming", "herald", "boost 1"}) {
		t.Errorf("said %q", got)
	}
	if !slices.Equal(res.Outcome.Changed, []string{"nft", "boost"}) || tp.marked() != 2 {
		t.Errorf("changed %q, the scan spoken %d times", res.Outcome.Changed, tp.marked())
	}
	want := []string{"accept accepted", "read nft beheld", "perform nft triumph", "perform herald triumph",
		"finish finished"}
	if got := tp.journal.acts(); !slices.Equal(got, want) {
		t.Errorf("journaled %q, want %q", got, want)
	}
	if got := tp.slate("firewall").Omens["nft"]; got.Omen != "gaming" || !got.Exalted {
		t.Errorf("the data-slate records nft as %+v", got)
	}
}

// TestInvokeII_AFallRestoresThePrivilegedSubject: a privileged subject the
// exalted servitor cannot prove falls, naming the omen found and the omen
// commanded; one set when a later subject falls is restored to the omen
// the exalted servitor read before, and the restoration is proven by it.
func TestInvokeII_AFallRestoresThePrivilegedSubject(t *testing.T) {
	tp := newTemple(t, "sudo")
	tp.rite("firewall", firewall)
	tp.firewallAs("none", "0")
	tp.sanctify("firewall")
	tp.fixture.vessel("stubborn", "")
	res, err := tp.invoke("firewall", "on", nil)
	if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Reverted {
		t.Fatalf("%v %+v", err, res.Outcome)
	}
	fell := res.Outcome.Fell
	if fell == nil || fell.Subject != "nft" || !strings.Contains(fell.Heresy.Error(),
		`the subject "nft" shows the omen "none"; "gaming" was commanded`) {
		t.Fatalf("fell %+v", fell)
	}
	grimdark(t, fell.Heresy.Error())
	if got := tp.slate("firewall").Omens["nft"]; got.Omen != "none" || !got.Exalted {
		t.Errorf("the data-slate records nft as %+v", got)
	}

	tp = newTemple(t, "sudo")
	tp.rite("firewall", firewall)
	tp.firewallAs("none", "0")
	tp.sanctify("firewall")
	tp.fixture.vessel("fall", "")
	res, err = tp.invoke("firewall", "on", nil)
	if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Reverted ||
		res.Outcome.Fell.Subject != "boost" {
		t.Fatalf("%v %+v", err, res.Outcome)
	}
	if got := strings.TrimSpace(tp.read(filepath.Join(tp.data, "nft"))); got != "none" {
		t.Errorf("nft shows %q once restored", got)
	}
	want := []string{"accept accepted", "read nft beheld", "perform nft triumph", "perform herald triumph",
		"revert herald reverted", "revert nft reverted", "finish finished"}
	if got := tp.journal.acts(); !slices.Equal(got, want) {
		t.Errorf("journaled %q, want %q", got, want)
	}
	if tp.marked() != 3 {
		t.Errorf("the scan was spoken %d times, want read, proof and proof of the restoration", tp.marked())
	}
	if got := tp.slate("firewall").Omens["nft"]; got.Omen != "none" || !got.Exalted {
		t.Errorf("the data-slate records nft as %+v", got)
	}
}

// TestAugurII_APrivilegedAuspexIsNeverSpokenOutsideAnInvocation: augury and
// census never speak a privileged auspex, and tell the omen the exalted
// servitor read at the last invocation, stale; foresight never speaks it
// either, telling the recorded omen stale, or that it is read at
// invocation when none is recorded.
func TestAugurII_APrivilegedAuspexIsNeverSpokenOutsideAnInvocation(t *testing.T) {
	tp := newTemple(t, "sudo")
	tp.rite("firewall", firewall)
	tp.firewallAs("gaming", "1")
	tp.sanctify("firewall")
	s := tp.servitor()
	foresee := func(aspect string) *invocation.SubjectPlan {
		t.Helper()
		res, err := s.Invoke(context.Background(), Petition{Rite: "firewall", Aspect: aspect, Foresee: true})
		if err != nil || res.Outcome != nil || len(res.Foresight) == 0 || res.Foresight[0].Subject != "nft" {
			t.Fatalf("foresight: %v %+v", err, res)
		}
		return res.Foresight[0].Plan
	}
	if p := foresee("on"); p == nil || !p.Privileged || p.Stale || p.Read != "" || p.Mute || !p.Changing {
		t.Errorf("foreseen before any invocation: %+v", p)
	}
	if r, err := s.Augur("firewall", false); err != nil || slices.ContainsFunc(r.Omens, func(o augury.Omen) bool {
		return o.Subject == "nft"
	}) {
		t.Errorf("augured before any invocation: %v %+v", err, r.Omens)
	}

	if _, err := tp.invoke("firewall", "on", nil); err != nil {
		t.Fatal(err)
	}
	must(t, os.Remove(filepath.Join(tp.data, "marker")))
	for _, forgo := range []bool{false, true} {
		r, err := s.Augur("firewall", forgo)
		i := slices.IndexFunc(r.Omens, func(o augury.Omen) bool { return o.Subject == "nft" })
		if err != nil || i < 0 || !r.Omens[i].Stale || r.Omens[i].Read != "gaming" || r.Omens[i].ReadAt.IsZero() ||
			r.Aspect != "on" || r.Standing != augury.Performed {
			t.Errorf("augured, sparing the auspex %v: %v %+v", forgo, err, r.Augury)
		}
	}
	entries, _ := s.Census(false)
	if i := slices.IndexFunc(entries, func(e Entry) bool { return e.Rite == "firewall" }); i < 0 ||
		!entries[i].Stale || entries[i].Aspect != "on" {
		t.Errorf("the census tells %+v", entries)
	}
	read := s.now()
	s.Now = func() time.Time { return read.Add(time.Hour) }
	if p := foresee("off"); p == nil || !p.Privileged || !p.Stale || p.Read != "gaming" || !p.ReadAt.Equal(read) ||
		!p.Changing {
		t.Errorf("foreseen after an invocation: %+v", p)
	}
	if tp.marked() != 0 {
		t.Errorf("the privileged auspex was spoken %d times outside an invocation", tp.marked())
	}
	if n := len(tp.called()); n != 1 {
		t.Errorf("the intercessor was called upon %d times", n)
	}
}

// veiledConf is a state rite whose exalted transcription conf writes a
// vessel in a hall the invoker cannot search, beside the common
// machine-spirit boost of firewall.
const veiledConf = `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
  {"name": "conf", "designation": "transcription", "exalted": true, "vessel": "$DATA/veiled/conf",
   "omens": {"on": "rules = on", "off": "rules = off"}},
  {"name": "boost", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"}, "auspex": {"scan": "cat $DATA/boost"},
   "enact": {"*": "echo {{omen}} > $DATA/boost"}}]}`

// TestInvokeII_TheExaltedServitorDecidesAVesselHiddenFromTheInvoker: an
// exalted transcription whose vessel the invoker cannot read is decided
// by the exalted servitor, which tells only whether it stands as the
// aspect demands: one that does is skipped, one that does not is set.
func TestInvokeII_TheExaltedServitorDecidesAVesselHiddenFromTheInvoker(t *testing.T) {
	for _, held := range []string{"rules = on", "rules = off"} {
		tp := newTemple(t, "sudo")
		tp.rite("conf", veiledConf)
		tp.fixture.vessel("boost", "1\n")
		veiled := filepath.Join(tp.data, "veiled")
		must(t, os.MkdirAll(filepath.Join(tp.ground, veiled), 0o755))
		must(t, os.WriteFile(filepath.Join(tp.ground, veiled, "conf"), []byte(held), 0o644))
		must(t, os.Mkdir(veiled, 0o755))
		must(t, os.Chmod(veiled, 0))
		t.Cleanup(func() { _ = os.Chmod(veiled, 0o755) })
		tp.sanctify("conf")
		res, err := tp.invoke("conf", "on", nil)
		if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Triumph {
			t.Fatalf("%s: %v %+v", held, err, res.Outcome)
		}
		want := []string{"accept accepted", "read conf beheld", "finish finished"}
		if held != "rules = on" {
			want = []string{"accept accepted", "read conf beheld", "perform conf triumph", "finish finished"}
		}
		if got := tp.journal.acts(); !slices.Equal(got, want) {
			t.Errorf("%s: journaled %q, want %q", held, got, want)
		}
		if got := tp.read(filepath.Join(tp.ground, veiled, "conf")); got != "rules = on" {
			t.Errorf("%s: the vessel holds %q", held, got)
		}
		if changed := res.Outcome.Changed; (held == "rules = on") != (len(changed) == 0) {
			t.Errorf("%s: changed %q", held, changed)
		}
	}
}
