package rituals

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// whose names the subject or the observance (marked "+") a verse of a
// state rite tells of.
func whose(v invocation.Verse) string {
	if v.Observance != "" {
		return "+" + v.Observance
	}
	return v.Subject
}

func deedsOf(out *invocation.Outcome) []string {
	names := []string{}
	for _, d := range out.Deeds {
		names = append(names, whose(d.Verse))
	}
	return names
}

func reversionsOf(out *invocation.Outcome) []string {
	names := []string{}
	for _, r := range out.Reversions {
		names = append(names, whose(r.Verse))
	}
	return names
}

// themeII is the owner's theme rite in pattern Mark II: one tether, a
// screen transition spoken as its introit and the hooks of the theme
// recited as its recessional. Each observance leaves in the log where the
// tether led when it was spoken.
const themeII = `{"pattern": "Mark II", "aspects": ["default", "finii", "porpl"], "vox-cast": ["progress", "success"],
  "subjects": [{"name": "theme", "designation": "tether", "tether": "$DATA/current",
    "omens": {"*": "$DATA/themes/{{aspect}}"}, "introit": ["transition"], "recessional": ["hooks"]}],
  "observances": [
    {"name": "transition", "designation": "incantation", "patience": "5s",
     "words": "echo transition-$(basename $(readlink $DATA/current)) >> $DATA/log"},
    {"name": "hooks", "designation": "litany", "scroll": "$DATA/recite-hooks", "offerings": ["$DATA/themes/{{aspect}}"],
     "reversion": "echo unhooked >> $DATA/log"}]}`

// visageFixture lays the themes, the hooks' scroll and the tether in the
// aspect default.
func visageFixture(t *testing.T) *fixture {
	fx := newFixture(t)
	fx.rite("theme", themeII)
	fx.vessel("recite-hooks", `echo hooks-$(basename "$1")-$(basename $(readlink `+fx.data+`/current)) >> `+fx.data+"/log\n")
	for _, theme := range []string{"default", "finii", "porpl"} {
		fx.vessel("themes/"+theme+"/colours", theme)
	}
	if err := os.Symlink(filepath.Join(fx.data, "themes", "default"), filepath.Join(fx.data, "current")); err != nil {
		t.Fatal(err)
	}
	return fx
}

// TestInvokeII_ATethersObservancesAreSpokenAroundItOnce: the introit of a
// changing subject is spoken once, before it is set; its recessional once,
// after it is set and proven; both in place. A subject left unchanged
// speaks neither, and its rite still triumphs. Progress counts subjects
// alone.
func TestInvokeII_ATethersObservancesAreSpokenAroundItOnce(t *testing.T) {
	fx := visageFixture(t)
	s := fx.servitor()
	log := filepath.Join(fx.data, "log")

	h := &proclaimed{}
	res, err := s.Invoke(context.Background(), Petition{Rite: "theme", Aspect: "porpl", Herald: h})
	if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Triumph {
		t.Fatalf("invocation: %v %+v", err, res)
	}
	if got := deedsOf(res.Outcome); !slices.Equal(got, []string{"+transition", "theme", "+hooks"}) {
		t.Fatalf("performed %q", got)
	}
	if !slices.Equal(res.Outcome.Changed, []string{"theme"}) {
		t.Fatalf("changed %q", res.Outcome.Changed)
	}
	if got := fx.read(log); got != "transition-default\nhooks-porpl-porpl\n" {
		t.Fatalf("log %q", got)
	}
	if want := []string{"progress theme ·/·", "success theme ·/·"}; !slices.Equal(h.told(), want) {
		t.Fatalf("the herald heard %q, want %q", h.told(), want)
	}

	again, err := s.Invoke(context.Background(), Petition{Rite: "theme", Aspect: "porpl"})
	if err != nil || again.Outcome == nil || again.Outcome.Verdict != invocation.Triumph || len(again.Outcome.Deeds) != 0 {
		t.Fatalf("the second invocation: %v %+v", err, again.Outcome)
	}
	if got := fx.read(log); got != "transition-default\nhooks-porpl-porpl\n" {
		t.Fatalf("an unchanged subject spoke its observances: log %q", got)
	}

	if _, err := s.Invoke(context.Background(), Petition{Rite: "theme", Aspect: "finii"}); err != nil {
		t.Fatal(err)
	}
	if got := fx.read(log); got != "transition-default\nhooks-porpl-porpl\ntransition-porpl\nhooks-finii-finii\n" {
		t.Fatalf("log %q", got)
	}
}

// dndII is the owner's do-not-disturb rite in pattern Mark II: the herald
// speaks only when the rite enters the aspect on.
const dndII = `{"pattern": "Mark II", "aspects": ["on", "off"], "vox-cast": ["success"], "subjects": [
  {"name": "mako", "designation": "machine-spirit", "omens": {"on": "do-not-disturb", "*": "default"},
   "auspex": {"scan": "cat $DATA/mode"}, "enact": {"*": "echo \"$SERVITOR_OMEN\" > $DATA/mode"},
   "recessional": ["herald-silence"]}],
  "observances": [{"name": "herald-silence", "designation": "incantation", "to": ["on"],
    "words": "echo silence-$SERVITOR_ASPECT-$(cat $DATA/mode) >> $DATA/log"}]}`

// TestInvokeII_AnObservanceGuardedToIsSpokenOnlyIntoItsAspects: a
// recessional guarded by "to" is spoken after its subject changes into an
// aspect it names, and passed over into any other.
func TestInvokeII_AnObservanceGuardedToIsSpokenOnlyIntoItsAspects(t *testing.T) {
	fx := newFixture(t)
	fx.rite("dnd", dndII)
	fx.vessel("mode", "default\n")
	log := filepath.Join(fx.data, "log")
	s := fx.servitor()

	for _, step := range []struct {
		aspect, deeds, log string
	}{
		{"on", "mako +herald-silence", "silence-on-do-not-disturb\n"},
		{"on", "", "silence-on-do-not-disturb\n"},
		{"off", "mako", "silence-on-do-not-disturb\n"},
		{"off", "", "silence-on-do-not-disturb\n"},
		{"on", "mako +herald-silence", "silence-on-do-not-disturb\nsilence-on-do-not-disturb\n"},
	} {
		res, err := s.Invoke(context.Background(), Petition{Rite: "dnd", Aspect: step.aspect})
		if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Triumph {
			t.Fatalf("%s: %v %+v", step.aspect, err, res)
		}
		if got := strings.Join(deedsOf(res.Outcome), " "); got != step.deeds {
			t.Fatalf("%s: performed %q, want %q", step.aspect, got, step.deeds)
		}
		if got, _ := os.ReadFile(log); string(got) != step.log {
			t.Fatalf("%s: log %q, want %q", step.aspect, got, step.log)
		}
	}
}

// sharedII is a state rite of three subjects that share one introit and
// one recessional.
const sharedII = `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
  {"name": "a", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"}, "auspex": {"scan": "cat $DATA/a"},
   "enact": {"*": "echo {{omen}} > $DATA/a; echo a >> $DATA/log"}, "introit": ["prep"], "recessional": ["post"]},
  {"name": "b", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"}, "auspex": {"scan": "cat $DATA/b"},
   "enact": {"*": "echo {{omen}} > $DATA/b; echo b >> $DATA/log"}, "introit": ["prep"], "recessional": ["post"]},
  {"name": "c", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"}, "auspex": {"scan": "cat $DATA/c"},
   "enact": {"*": "echo {{omen}} > $DATA/c; echo c >> $DATA/log"}, "introit": ["prep"], "recessional": ["post"]}],
  "observances": [
    {"name": "prep", "designation": "incantation", "words": "echo prep >> $DATA/log"},
    {"name": "post", "designation": "incantation", "words": "echo post >> $DATA/log"}]}`

// TestInvokeII_AnObservanceSharedBySubjectsIsSpokenOnce: an introit shared
// by several subjects is spoken once, before the first of them that
// changes; a recessional once, after the last of them that changes.
func TestInvokeII_AnObservanceSharedBySubjectsIsSpokenOnce(t *testing.T) {
	for _, tc := range []struct {
		name    string
		set     string // the subjects already showing the omen of the aspect on
		deeds   string
		changed []string
	}{
		{"every subject changes", "", "+prep a b c +post", []string{"a", "b", "c"}},
		{"only the middle one changes", "ac", "+prep b +post", []string{"b"}},
		{"the first and the last change", "b", "+prep a c +post", []string{"a", "c"}},
		{"only the first changes", "bc", "+prep a +post", []string{"a"}},
		{"nothing changes", "abc", "", []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newFixture(t)
			fx.rite("shared", sharedII)
			for _, sub := range []string{"a", "b", "c"} {
				omen := "0\n"
				if strings.Contains(tc.set, sub) {
					omen = "1\n"
				}
				fx.vessel(sub, omen)
			}

			res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "shared", Aspect: "on"})

			if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Triumph {
				t.Fatalf("invocation: %v %+v", err, res)
			}
			if got := strings.Join(deedsOf(res.Outcome), " "); got != tc.deeds {
				t.Fatalf("performed %q, want %q", got, tc.deeds)
			}
			if !slices.Equal(res.Outcome.Changed, tc.changed) {
				t.Fatalf("changed %q, want %q", res.Outcome.Changed, tc.changed)
			}
			want := strings.ReplaceAll(strings.ReplaceAll(tc.deeds, "+", ""), " ", "\n")
			if got, _ := os.ReadFile(filepath.Join(fx.data, "log")); strings.TrimSpace(string(got)) != want {
				t.Fatalf("log %q, want %q", got, want)
			}
		})
	}
}

// TestInvokeII_AFallenRecessionalRevertsSubjectsAndIntroits: a recessional
// that falls reverts everything performed, in reverse order, itself
// included: the reversions of the observances are spoken, and the subjects
// restored to what was read. The chronicle and the herald name the fallen
// observance.
func TestInvokeII_AFallenRecessionalRevertsSubjectsAndIntroits(t *testing.T) {
	fx := newFixture(t)
	fx.rite("visage", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "theme", "designation": "tether", "tether": "$DATA/current", "omens": {"*": "$DATA/{{aspect}}"},
	   "introit": ["veil"]},
	  {"name": "boost", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"}, "auspex": {"scan": "cat $DATA/boost"},
	   "enact": {"*": "echo {{omen}} > $DATA/boost; echo boost-{{omen}} >> $DATA/log"}, "recessional": ["hooks"]}],
	  "observances": [
	    {"name": "veil", "designation": "incantation", "words": "echo veil >> $DATA/log",
	     "reversion": "echo unveil-$(basename $(readlink $DATA/current)) >> $DATA/log"},
	    {"name": "hooks", "designation": "incantation", "words": "echo hooks >> $DATA/log; exit 4",
	     "reversion": "echo unhooks >> $DATA/log"}]}`)
	fx.vessel("boost", "0\n")
	if err := os.Symlink(filepath.Join(fx.data, "off"), filepath.Join(fx.data, "current")); err != nil {
		t.Fatal(err)
	}
	h := &proclaimed{}

	res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "visage", Aspect: "on", Herald: h})

	if err != nil || res.Outcome == nil {
		t.Fatalf("invocation: %v %+v", err, res)
	}
	out := res.Outcome
	if out.Verdict != invocation.Reverted || out.Fell == nil || out.Fell.Observance != "hooks" {
		t.Fatalf("outcome %+v", out)
	}
	if msg := out.Fell.Heresy.Error(); !strings.Contains(msg, `observance "hooks"`) || !strings.Contains(msg, "death-mark 4") {
		t.Fatalf("the heresy %q names not the observance and its death-mark", msg)
	}
	grimdark(t, out.Fell.Heresy.Error())
	if got := deedsOf(out); !slices.Equal(got, []string{"+veil", "theme", "boost", "+hooks"}) {
		t.Fatalf("performed %q", got)
	}
	if got := reversionsOf(out); !slices.Equal(got, []string{"+hooks", "boost", "theme", "+veil"}) {
		t.Fatalf("reverted %q", got)
	}
	for _, r := range out.Reversions {
		if r.Heresy != nil {
			t.Errorf("the reversion of %s fell: %v", whose(r.Verse), r.Heresy)
		}
	}
	if got := fx.read(filepath.Join(fx.data, "log")); got != "veil\nboost-1\nhooks\nunhooks\nboost-0\nunveil-off\n" {
		t.Fatalf("log %q", got)
	}
	if a, _ := os.Readlink(filepath.Join(fx.data, "current")); a != filepath.Join(fx.data, "off") {
		t.Fatalf("the tether leads to %q", a)
	}
	last := h.heard[len(h.heard)-1]
	if last.Tidings != invocation.Failure || last.Verse.Observance != "hooks" || last.Values.Tidings.Subject != "" ||
		!strings.Contains(last.Values.Tidings.Heresy, `observance "hooks"`) {
		t.Fatalf("the herald heard %+v", last)
	}
	line := strings.TrimSpace(fx.read(filepath.Join(fx.state, "servitor", "chronicle.jsonl")))
	for _, want := range []string{`"verdict":"reverted"`, `"fell_at":{"observance":"hooks"}`, `"changed":["theme","boost"]`,
		`"reversions":[{"observance":"hooks","verdict":"triumph"},{"subject":"boost","verdict":"triumph"},` +
			`{"subject":"theme","verdict":"triumph"},{"observance":"veil","verdict":"triumph"}]`} {
		if !strings.Contains(line, want) {
			t.Errorf("the chronicle lacks %s:\n%s", want, line)
		}
	}
}

// levelII is a state rite of three aspects whose one subject names its
// aspect itself; one recessional is guarded by "from", one is not.
const levelII = `{"pattern": "Mark II", "aspects": ["low", "mid", "high"], "subjects": [
  {"name": "level", "designation": "machine-spirit", "auspex": {"scan": "cat $DATA/level"},
   "enact": {"*": "echo {{omen}} > $DATA/level"}, "recessional": ["out-of-low", "any"]}],
  "observances": [
    {"name": "out-of-low", "designation": "incantation", "from": ["low"], "words": "echo out-of-low >> $DATA/log"},
    {"name": "any", "designation": "incantation",
     "words": "echo any-$SERVITOR_FORMER_ASPECT${SERVITOR_OMEN:+-omen} >> $DATA/log"}]}`

// TestInvokeII_AnObservanceGuardedFromIsSpokenOnlyOutOfItsAspects: "from"
// admits an observance when it names the former aspect; an unknown former
// aspect admits none guarded by "from". No observance hears an omen, not
// even one inherited.
func TestInvokeII_AnObservanceGuardedFromIsSpokenOnlyOutOfItsAspects(t *testing.T) {
	t.Setenv("SERVITOR_OMEN", "stray")
	for _, tc := range []struct {
		name, level, former, log string
	}{
		{"out of the aspect named", "low", "low", "out-of-low\nany-low\n"},
		{"out of another aspect", "mid", "mid", "any-mid\n"},
		{"out of an unknown aspect", "astray", "", "any-\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newFixture(t)
			fx.rite("level", levelII)
			fx.vessel("level", tc.level+"\n")

			res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "level", Aspect: "high"})

			if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Triumph ||
				res.Options.Former != tc.former {
				t.Fatalf("invocation: %v %+v", err, res)
			}
			if got := fx.read(filepath.Join(fx.data, "log")); got != tc.log {
				t.Fatalf("log %q, want %q", got, tc.log)
			}
		})
	}
}

// TestInvokeII_ForeseesTheObservancesThatWouldBeSpoken: foresight tells
// the observances that would be spoken, in their place among the subjects,
// and speaks none of them.
func TestInvokeII_ForeseesTheObservancesThatWouldBeSpoken(t *testing.T) {
	fx := visageFixture(t)
	s := fx.servitor()

	res, err := s.Invoke(context.Background(), Petition{Rite: "theme", Aspect: "porpl", Foresee: true})

	if err != nil || res.Outcome != nil {
		t.Fatalf("foresight: %v %+v", err, res)
	}
	var told []string
	for _, f := range res.Foresight {
		told = append(told, whose(f.Verse))
	}
	if !slices.Equal(told, []string{"+transition", "theme", "+hooks"}) {
		t.Fatalf("foreseen %q", told)
	}
	transition, hooks := res.Foresight[0], res.Foresight[2]
	if transition.Speech == nil || !strings.Contains(transition.Speech.Words, "echo transition-") ||
		transition.Speech.Patience.String() != "5s" || transition.Plan != nil {
		t.Fatalf("transition %+v", transition)
	}
	if hooks.Speech == nil || hooks.Speech.Reversion != "echo unhooked >> "+fx.data+"/log" ||
		!slices.Equal(hooks.Speech.Offerings, []string{filepath.Join(fx.data, "themes", "porpl")}) {
		t.Fatalf("hooks %+v", hooks.Speech)
	}
	if exists(filepath.Join(fx.data, "log")) {
		t.Fatal("foresight spoke an observance")
	}

	if err := os.Remove(filepath.Join(fx.data, "current")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(fx.data, "themes", "porpl"), filepath.Join(fx.data, "current")); err != nil {
		t.Fatal(err)
	}
	res, err = s.Invoke(context.Background(), Petition{Rite: "theme", Aspect: "porpl", Foresee: true})
	if err != nil || len(res.Foresight) != 1 || res.Foresight[0].Plan == nil || res.Foresight[0].Plan.Changing {
		t.Fatalf("foresight of an unchanged subject: %v %+v", err, res.Foresight)
	}
}

// TestInvokeII_ObservancesAreSpokenAsMarkISpeech: an observance is spoken
// in its own tongue, else the rite's, within its own patience; one that
// outlasts it falls and is reverted with the rest.
func TestInvokeII_ObservancesAreSpokenAsMarkISpeech(t *testing.T) {
	fx := newFixture(t)
	fx.rite("tongues", `{"pattern": "Mark II", "aspects": ["on", "off"], "tongue": "sh", "subjects": [
	  {"name": "m", "designation": "machine-spirit", "auspex": {"scan": "cat $DATA/m"},
	   "enact": {"*": "echo {{omen}} > $DATA/m"}, "introit": ["rite-tongue", "own-tongue"], "recessional": ["slow"]}],
	  "observances": [
	    {"name": "rite-tongue", "designation": "incantation", "words": "echo $(basename $0) >> $DATA/log"},
	    {"name": "own-tongue", "designation": "incantation", "tongue": "bash", "words": "echo $(basename $0) >> $DATA/log",
	     "reversion": "echo unspoken >> $DATA/log"},
	    {"name": "slow", "designation": "incantation", "patience": "100ms", "words": "sleep 5"}]}`)
	fx.vessel("m", "off\n")

	res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "tongues", Aspect: "on"})

	if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Reverted || res.Outcome.Fell.Observance != "slow" {
		t.Fatalf("invocation: %v %+v", err, res.Outcome)
	}
	if msg := res.Outcome.Fell.Heresy.Error(); !strings.Contains(msg, "patience of 100ms ran out") {
		t.Fatalf("heresy %q", msg)
	}
	if got := reversionsOf(res.Outcome); !slices.Equal(got, []string{"m", "+own-tongue"}) {
		t.Fatalf("reverted %q", got)
	}
	if got := fx.read(filepath.Join(fx.data, "log")); got != "sh\nbash\nunspoken\n" {
		t.Fatalf("log %q", got)
	}
	if got := fx.read(filepath.Join(fx.data, "m")); got != "off\n" {
		t.Fatalf("the subject holds %q", got)
	}
}

// TestInvokeII_EveryObservanceIsExaminedBeforeAnythingIsTouched: the
// pre-flight examines every observance, even one its guards would pass
// over, and a heresy found in it refuses the invocation, located at the
// observance, before anything is touched.
func TestInvokeII_EveryObservanceIsExaminedBeforeAnythingIsTouched(t *testing.T) {
	fx := newFixture(t)
	fx.rite("dnd", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "mako", "designation": "machine-spirit", "auspex": {"scan": "cat $DATA/mode"},
	   "enact": {"*": "echo {{omen}} > $DATA/mode"}, "recessional": ["herald"]}],
	  "observances": [{"name": "herald", "designation": "incantation", "to": ["on"], "tongue": "no-such-tongue",
	    "words": "true"}]}`)
	fx.vessel("mode", "on\n")

	s := fx.servitor()
	res, err := s.Invoke(context.Background(), Petition{Rite: "dnd", Aspect: "off"})

	var hs invocation.Heresies
	if !errors.As(err, &hs) || len(hs) != 1 || res.Outcome != nil {
		t.Fatalf("invocation: %v %+v", err, res)
	}
	r := s.Librarium.Rites["dnd"]
	if want := r.Locate("/observances/0/tongue"); hs[0].Position != want || want == r.Locate("/observances/0") ||
		hs[0].Verse != 0 || !strings.Contains(hs[0].Message, `"no-such-tongue"`) {
		t.Fatalf("heresy %+v, want it at %+v", hs[0], want)
	}
	if got := fx.read(filepath.Join(fx.data, "mode")); got != "on\n" {
		t.Fatalf("the subject was touched: %q", got)
	}
}
