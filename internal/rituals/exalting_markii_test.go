package rituals

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/exalted"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// exaltedGaming is a state rite of exalted subjects and an exalted
// observance among a common subject: the exalted machine-spirit smt, whose
// enact words tell what omen they set and in whose home; the exalted
// recessional herald, admitted only into "on"; the common machine-spirit
// boost; and the exalted transcription conf, whose vessel stands only in
// "off".
const exaltedGaming = `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
  {"name": "smt", "designation": "machine-spirit", "exalted": true, "omens": {"on": "off", "*": "on"},
   "auspex": {"scan": "cat $DATA/smt"},
   "enact": {"*": "echo {{omen}} > $DATA/smt; echo \"exalted $SERVITOR_OMEN $HOME\" >> $DATA/said"},
   "recessional": ["herald"]},
  {"name": "boost", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"}, "auspex": {"scan": "cat $DATA/boost"},
   "enact": {"*": "echo {{omen}} > $DATA/boost; echo boost {{omen}} >> $DATA/said"}},
  {"name": "conf", "designation": "transcription", "exalted": true, "vessel": "/etc/servitor-trials-gaming.conf",
   "omens": {"on": null, "off": "gaming = off"}}],
  "observances": [
  {"name": "herald", "designation": "incantation", "exalted": true, "to": ["on"], "words": "echo herald $HOME >> $DATA/said"}]}`

// gamingOff lays the machine of exaltedGaming as it stands in "off".
func (tp *temple) gamingOff() {
	tp.fixture.vessel("smt", "on\n")
	tp.fixture.vessel("boost", "0\n")
}

// confOfGaming is what the exalted vessel of exaltedGaming holds in the
// ground; "absent" when none stands.
func (tp *temple) confOfGaming() string {
	data, err := os.ReadFile(filepath.Join(tp.ground, "/etc/servitor-trials-gaming.conf"))
	if err != nil {
		return "absent"
	}
	return string(data)
}

// TestInvokeII_ExaltedSubjectsAreSetByTheExaltedServitor: the exalted
// subjects and observances of a sanctioned state rite that change are set
// by the exalted servitor alone, raised with one proof of the master after
// the reading and the plan, in their places among the common subjects;
// the invoker proves the machine-spirit by its own reading and records
// it. An invocation that changes nothing exalted raises no exalted
// servitor and calls upon no intercessor.
func TestInvokeII_ExaltedSubjectsAreSetByTheExaltedServitor(t *testing.T) {
	tp := newTemple(t, "sudo")
	tp.rite("gaming", exaltedGaming)
	tp.gamingOff()
	kept := tp.sanctify("gaming")

	h := &teller{}
	res, err := tp.invoke("gaming", "on", h)
	if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Triumph {
		t.Fatalf("%v %+v", err, res.Outcome)
	}
	if got := tp.said(); !slices.Equal(got, []string{"exalted off /root", "herald /root", "boost 1"}) {
		t.Errorf("said %q", got)
	}
	if !slices.Equal(res.Outcome.Changed, []string{"smt", "boost"}) || res.Outcome.Known["smt"].Omen != "off" {
		t.Errorf("changed %q, known %+v", res.Outcome.Changed, res.Outcome.Known)
	}
	sigil, _ := kept.SigilOf("on")
	want := []string{filepath.Join(tp.bin, "sudo"), "-k", "-p", "servitor: authenticate to perform gaming (sigil " +
		sanction.Short(sigil) + "): ", "--", "/usr/bin/servitor", exalted.Mode, "gaming", sanction.Short(sigil), "on"}
	if calls := tp.called(); len(calls) != 1 || !slices.Equal(calls[0], want) {
		t.Errorf("the intercessors were called upon as %q, want once as %q", calls, want)
	}
	if got := tp.journal.deeds(); !slices.Equal(got, []string{"accept accepted", "perform triumph", "perform triumph",
		"finish finished"}) {
		t.Errorf("journaled %q", got)
	}
	if got := tp.journal.items(); !slices.Equal(got, []string{"smt", "herald"}) {
		t.Errorf("the exalted servitor acted upon %q", got)
	}
	if len(h.told) == 0 || h.told[0].Tidings != invocation.Exaltation {
		t.Errorf("the herald heard %+v", h.told)
	}
	if e := tp.chronicled(); len(e) != 1 || e[0].Intercessor != "sudo" || e[0].Sigil != sanction.Short(sigil) {
		t.Errorf("chronicled %+v", e)
	}
	if slate := tp.slate("gaming"); slate == nil || slate.Omens["smt"].Omen != "off" || slate.Omens["smt"].Exalted {
		t.Errorf("the data-slate records %+v", slate)
	}

	// Again into "on": nothing changes, and no proof is asked.
	h = &teller{}
	res, err = tp.invoke("gaming", "on", h)
	if err != nil || res.Outcome.Verdict != invocation.Triumph || len(res.Outcome.Changed) != 0 {
		t.Fatalf("again: %v %+v", err, res.Outcome)
	}
	if n := len(tp.called()); n != 1 {
		t.Errorf("an invocation changing nothing exalted called upon the intercessor (%d calls)", n)
	}
	if res.Intercessor != "" || slices.ContainsFunc(h.told, func(p invocation.Proclamation) bool {
		return p.Tidings == invocation.Exaltation
	}) {
		t.Errorf("an invocation changing nothing exalted told of exaltation: %q %+v", res.Intercessor, h.told)
	}
	if e := tp.chronicled(); len(e) != 2 || e[0].Intercessor != "" || e[0].Sigil != "" {
		t.Errorf("chronicled %+v", e)
	}

	// Into "off": the vessel of conf is transcribed by the exalted servitor.
	if res, err = tp.invoke("gaming", "off", nil); err != nil || res.Outcome.Verdict != invocation.Triumph {
		t.Fatalf("off: %v %+v", err, res.Outcome)
	}
	if got := tp.confOfGaming(); got != "gaming = off" || exists("/etc/servitor-trials-gaming.conf") {
		t.Errorf("the exalted vessel holds %q", got)
	}
	if n := len(tp.called()); n != 2 {
		t.Errorf("the intercessor was called upon %d times", n)
	}
}

// items names the subject or observance of every act journaled with one.
func (l *ledger) items() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []string
	for _, e := range l.entries {
		if e.Subject+e.Observance != "" {
			out = append(out, e.Subject+e.Observance)
		}
	}
	return out
}

// TestInvokeII_TheSanctionIsJudgedEvenWhenNothingExaltedChanges: an
// invocation of a state rite whose exalted items stand unsanctioned or
// whose sanction lapsed is refused before anything is read or touched,
// even when nothing exalted would change: at each exalted item, naming the
// ritual that sanctions it.
func TestInvokeII_TheSanctionIsJudgedEvenWhenNothingExaltedChanges(t *testing.T) {
	for _, lapsed := range []bool{false, true} {
		tp := newTemple(t, "sudo")
		tp.rite("gaming", exaltedGaming)
		tp.fixture.vessel("smt", "off\n")
		tp.fixture.vessel("boost", "1\n")
		if lapsed {
			tp.sanctify("gaming")
			tp.rite("gaming", strings.Replace(exaltedGaming, "herald $HOME", "herald anew $HOME", 1))
		}
		res, err := tp.invoke("gaming", "on", nil)
		var hs invocation.Heresies
		if !errors.As(err, &hs) || len(hs) != 3 || res.Outcome != nil {
			t.Fatalf("lapsed %v: %v %+v", lapsed, err, res)
		}
		r := tp.servitor().Librarium.Rites["gaming"]
		for i, ptr := range []string{"/subjects/0/exalted", "/subjects/2/exalted", "/observances/0/exalted"} {
			if hs[i].Position != r.Locate(ptr) || !strings.Contains(hs[i].Message, "servitor sanction gaming") {
				t.Errorf("heresy %d: %+v", i, hs[i])
			}
			grimdark(t, hs[i].Message)
		}
		if !strings.Contains(hs[0].Message, `the subject "smt" is exalted`) {
			t.Errorf("the heresy does not name its subject: %s", hs[0].Message)
		}
		if len(tp.called()) != 0 || tp.said() != nil || exists(filepath.Join(tp.state, "servitor")) {
			t.Errorf("lapsed %v: touched", lapsed)
		}
	}
}

// TestInvokeII_AFallRestoresTheExaltedSubjects: when a common subject falls
// after exalted ones, the exalted servitor reverts them in reverse order
// among the common reversions: an exalted machine-spirit is restored to
// the omen it was read to show, and proven so by the invoker; no second
// proof of the master is asked.
func TestInvokeII_AFallRestoresTheExaltedSubjects(t *testing.T) {
	tp := newTemple(t, "sudo")
	tp.rite("gaming", strings.Replace(exaltedGaming, `"enact": {"*": "echo {{omen}} > $DATA/boost;`,
		`"enact": {"1": "echo 7 > $DATA/boost", "*": "echo {{omen}} > $DATA/boost;`, 1))
	tp.gamingOff()
	tp.sanctify("gaming")
	res, err := tp.invoke("gaming", "on", nil)
	if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Reverted {
		t.Fatalf("%v %+v", err, res.Outcome)
	}
	if got := tp.said(); !slices.Equal(got, []string{"exalted off /root", "herald /root", "boost 0",
		"exalted on /root"}) {
		t.Errorf("said %q", got)
	}
	if got := strings.TrimSpace(tp.read(filepath.Join(tp.data, "smt"))); got != "on" {
		t.Errorf("smt shows %q once restored", got)
	}
	if got := tp.journal.deeds(); !slices.Equal(got, []string{"accept accepted", "perform triumph", "perform triumph",
		"revert reverted", "revert reverted", "finish finished"}) {
		t.Errorf("journaled %q", got)
	}
	if got := tp.journal.items(); !slices.Equal(got, []string{"smt", "herald", "herald", "smt"}) {
		t.Errorf("the exalted servitor acted upon %q", got)
	}
	if n := len(tp.called()); n != 1 {
		t.Errorf("the intercessor was called upon %d times", n)
	}
}

// TestInvokeII_NoRuntimeWordIsSplicedIntoExaltedWords: the injection of the
// research (--who 'x; echo INJECTED') never reaches the words of an
// exalted subject. {{omen}} is spliced into exalted enact words, fixed at
// sanction, and a rune reaches them as a variable alone. A rune for an
// inscription spliced into the omen of an exalted subject, or into its
// enact words, is refused; an inscription whose word only a rune gives may
// be spliced into neither.
func TestInvokeII_NoRuntimeWordIsSplicedIntoExaltedWords(t *testing.T) {
	const rite = `{"pattern": "Mark II", "aspects": ["on", "off"],
	  "inscriptions": {"who": {"purpose": "who asks"}, "level": {"decrees": {"on": "0", "*": "1"}}},
	  "subjects": [{"name": "lock", "designation": "machine-spirit", "exalted": true,
	    "omens": {"*": "{{inscription.level}}"}, "auspex": {"scan": "cat $DATA/lock"},
	    "enact": {"*": "echo {{omen}} > $DATA/lock; echo \"omen={{omen}} who=$SERVITOR_INSCRIPTION_WHO\" >> $DATA/said"}}]}`
	tp := newTemple(t, "sudo")
	tp.rite("lock", rite)
	tp.fixture.vessel("lock", "1\n")
	tp.sanctify("lock")
	p := Petition{Rite: "lock", Aspect: "on", Exaltation: &Exaltation{}, Runes: map[string]string{"who": "x; echo INJECTED"}}
	res, err := tp.servitor().Invoke(context.Background(), p)
	if err != nil || res.Outcome.Verdict != invocation.Triumph {
		t.Fatalf("%v %+v", err, res.Outcome)
	}
	if got := tp.said(); !slices.Equal(got, []string{"omen=0 who=x; echo INJECTED"}) {
		t.Errorf("said %q", got)
	}

	p.Runes = map[string]string{"level": "0; echo INJECTED"}
	res, err = tp.servitor().Invoke(context.Background(), p)
	var hs invocation.Heresies
	if !errors.As(err, &hs) || len(hs) != 1 || !strings.Contains(hs[0].Message, "--level") || res.Outcome != nil {
		t.Fatalf("a rune of an inscription spliced into an exalted omen: %v %+v", err, res)
	}

	for what, scripture := range map[string]string{
		"an omen":       strings.Replace(rite, `"{{inscription.level}}"}`, `"{{inscription.who}}"}`, 1),
		"enact words":   strings.Replace(rite, `who=$SERVITOR_INSCRIPTION_WHO`, `who={{inscription.who}}`, 1),
		"former aspect": strings.Replace(rite, `who=$SERVITOR_INSCRIPTION_WHO`, `who={{former}}`, 1),
	} {
		mandatory := strings.Replace(scripture, `{"purpose": "who asks"}`, `{"purpose": "who asks", "mandatory": true}`, 1)
		_, findings := librarium.Parse("lock", "/lib/rites/lock.json", []byte(mandatory))
		if !findings.Heretical() {
			t.Errorf("a runtime word spliced into exalted %s is no heresy", what)
		}
	}
	if len(tp.called()) != 1 {
		t.Errorf("the intercessor was called upon %d times", len(tp.called()))
	}
}

// TestInquireII_NotesTheSanctionOfAStateRite: the inquisition notes an
// exalted state rite that stands unsanctioned, or whose sanction lapsed,
// at its first exalted subject or observance.
func TestInquireII_NotesTheSanctionOfAStateRite(t *testing.T) {
	tp := newTemple(t, "sudo")
	tp.rite("gaming", exaltedGaming)
	for _, want := range []string{"stands unsanctioned", "has lapsed"} {
		if want == "has lapsed" {
			tp.sanctify("gaming")
			tp.rite("gaming", strings.Replace(exaltedGaming, "herald $HOME", "herald anew $HOME", 1))
		}
		s := tp.servitor()
		findings, err := s.Inquire([]string{"gaming"}, false)
		if err != nil {
			t.Fatal(err)
		}
		r := s.Librarium.Rites["gaming"]
		i := slices.IndexFunc(findings, func(f librarium.Finding) bool { return strings.Contains(f.Message, want) })
		if i < 0 || findings[i].Position != r.Locate("/subjects/0/exalted") ||
			!strings.Contains(findings[i].Message, "servitor sanction gaming") {
			t.Errorf("findings %v, want %q at the first exalted subject", findings, want)
		}
	}
}

// TestInquireII_TheInstalledServitorOfAStateRite: while a rite of exalted
// subjects and observances is examined, the servitor installed at
// /usr/bin/servitor that is missing is an impurity of the machine.
func TestInquireII_TheInstalledServitorOfAStateRite(t *testing.T) {
	fx := newFixture(t)
	fx.rite("gaming", exaltedGaming)
	intercede(t, func(root string) { _ = os.Remove(filepath.Join(root, "usr/bin/servitor")) })
	found := inquire(t, fx.servitor(), false)
	if got := denounced(found, "", librarium.Impurity, "/usr/bin/servitor", `the rite "gaming" holds exalted subjects `+
		"and observances"); len(got) != 1 {
		t.Fatalf("noted %d times in\n%v", len(got), found)
	}
}
