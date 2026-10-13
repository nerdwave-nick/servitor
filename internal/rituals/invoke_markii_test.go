package rituals

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/sanctum"
)

// gamingII is a state rite of pattern Mark II: a machine-spirit read by its
// scan, one read by its trials, a tether and a transcription. Every enact
// word leaves its omen in the log.
const gamingII = `{"pattern": "Mark II", "aspects": ["on", "off"], "vox-cast": ["progress", "success"], "subjects": [
  {"name": "smt", "designation": "machine-spirit", "omens": {"on": "off", "*": "on"},
   "auspex": {"scan": "cat $DATA/smt"},
   "enact": {"*": "echo {{omen}} > $DATA/smt; echo smt-$SERVITOR_OMEN >> $DATA/log"}},
  {"name": "scheduler", "designation": "machine-spirit", "omens": {"on": "cosmos", "*": "eevdf"},
   "auspex": {"trials": {"cosmos": "grep -qx cosmos $DATA/scheduler"}},
   "enact": {"*": "echo {{omen}} > $DATA/scheduler; echo scheduler-{{omen}} >> $DATA/log"}},
  {"name": "theme", "designation": "tether", "tether": "$DATA/current", "omens": {"*": "$DATA/{{aspect}}"}},
  {"name": "conf", "designation": "transcription", "vessel": "$DATA/conf", "omens": {"on": "fast", "off": null}}]}`

// proclaimed records every proclamation whole.
type proclaimed struct{ heard []invocation.Proclamation }

func (h *proclaimed) Proclaim(p invocation.Proclamation) { h.heard = append(h.heard, p) }

func (h *proclaimed) told() []string {
	var out []string
	for _, p := range h.heard {
		t := p.Values.Tidings
		out = append(out, string(p.Tidings)+" "+t.Subject+" "+strings.Repeat("·", t.Step)+"/"+strings.Repeat("·", t.Steps))
	}
	return out
}

// gamingOff lays the machine as gamingII leaves it in the aspect off.
func gamingOff(fx *fixture) {
	fx.t.Helper()
	fx.vessel("smt", "on\n")
	fx.vessel("scheduler", "eevdf\n")
	if err := os.Symlink(filepath.Join(fx.data, "off"), filepath.Join(fx.data, "current")); err != nil {
		fx.t.Fatal(err)
	}
}

func subjectsOf(deeds []invocation.Deed) []string {
	names := []string{}
	for _, d := range deeds {
		names = append(names, d.Subject)
	}
	return names
}

// TestInvokeII_SetsWhatDiffersAndASecondInvocationChangesNothing: a state
// rite reads every subject, sets only those that do not show the omen of
// the aspect invoked, in written order, and proves each; invoked again into
// the same aspect it changes nothing and still triumphs, is kept and
// chronicled, and proclaims its success.
func TestInvokeII_SetsWhatDiffersAndASecondInvocationChangesNothing(t *testing.T) {
	fx := newFixture(t)
	fx.rite("gaming", gamingII)
	gamingOff(fx)
	s := fx.servitor()

	h := &proclaimed{}
	first, err := s.Invoke(context.Background(), Petition{Rite: "gaming", Aspect: "on", Herald: h})
	if err != nil || first.Outcome == nil || first.Outcome.Verdict != invocation.Triumph {
		t.Fatalf("first invocation: %v %+v", err, first)
	}
	every := []string{"smt", "scheduler", "theme", "conf"}
	if got := subjectsOf(first.Outcome.Deeds); !slices.Equal(got, every) || !slices.Equal(first.Outcome.Changed, every) ||
		first.Options.Former != "off" {
		t.Fatalf("performed %q, changed %q, former %q", got, first.Outcome.Changed, first.Options.Former)
	}
	if want := []string{"progress smt ·/····", "progress scheduler ··/····", "progress theme ···/····",
		"progress conf ····/····", "success conf ····/····"}; !slices.Equal(h.told(), want) {
		t.Fatalf("the herald heard %q, want %q", h.told(), want)
	}
	if got := fx.read(filepath.Join(fx.data, "log")); got != "smt-off\nscheduler-cosmos\n" {
		t.Fatalf("log %q", got)
	}
	if fx.read(filepath.Join(fx.data, "smt")) != "off\n" || fx.read(filepath.Join(fx.data, "scheduler")) != "cosmos\n" ||
		fx.read(filepath.Join(fx.data, "conf")) != "fast" {
		t.Fatal("the subjects were not set")
	}
	if a, _ := os.Readlink(filepath.Join(fx.data, "current")); a != filepath.Join(fx.data, "on") {
		t.Fatalf("the tether leads to %q", a)
	}

	h = &proclaimed{}
	second, err := s.Invoke(context.Background(), Petition{Rite: "gaming", Aspect: "on", Herald: h})
	if err != nil || second.Outcome == nil || second.Outcome.Verdict != invocation.Triumph {
		t.Fatalf("second invocation: %v %+v", err, second)
	}
	if len(second.Outcome.Deeds) != 0 || second.Outcome.Changed == nil || len(second.Outcome.Changed) != 0 ||
		second.Options.Former != "on" {
		t.Fatalf("the second invocation performed %+v (former %q)", second.Outcome, second.Options.Former)
	}
	if got := fx.read(filepath.Join(fx.data, "log")); got != "smt-off\nscheduler-cosmos\n" {
		t.Fatalf("the second invocation enacted: log %q", got)
	}
	if want := []string{"success  /"}; !slices.Equal(h.told(), want) {
		t.Fatalf("the herald heard %q, want %q", h.told(), want)
	}
	if sl := fx.slate("gaming"); sl.Aspect != "on" || sl.LastRite.Verdict != invocation.Triumph {
		t.Fatalf("data-slate %+v", sl)
	}
	lines := strings.Split(strings.TrimSpace(fx.read(filepath.Join(fx.state, "servitor", "chronicle.jsonl"))), "\n")
	want := []string{
		`{"at":"2026-10-10T13:30:00Z","rite":"gaming","aspect":"on","former":"off","inscriptions":{},"verdict":"triumph","changed":["smt","scheduler","theme","conf"]}`,
		`{"at":"2026-10-10T13:30:00Z","rite":"gaming","aspect":"on","former":"on","inscriptions":{},"verdict":"triumph","changed":[]}`,
	}
	if !slices.Equal(lines, want) {
		t.Fatalf("chronicle\n got %q\nwant %q", lines, want)
	}
	if a, _ := augury.Augur(fx.servitor().Librarium.Rites["gaming"], augury.Options{}); a.Standing != augury.Performed ||
		a.Aspect != "on" {
		t.Fatalf("augury %+v", a)
	}

	// another hand resets smt: only it is set again
	fx.vessel("smt", "on\n")
	third, err := s.Invoke(context.Background(), Petition{Rite: "gaming", Aspect: "on"})
	if err != nil || third.Outcome == nil || third.Outcome.Verdict != invocation.Triumph ||
		!slices.Equal(third.Outcome.Changed, []string{"smt"}) || third.Former != augury.Corrupted {
		t.Fatalf("third invocation: %v %+v", err, third)
	}
	if got := fx.read(filepath.Join(fx.data, "log")); got != "smt-off\nscheduler-cosmos\nsmt-off\n" {
		t.Fatalf("log %q", got)
	}
}

// TestInvokeII_ASanctumMarkedForAnotherAspectChanges: a sanctum whose
// region already holds the scripture of the aspect invoked, but whose
// marker names another, is changing: its marker is rewritten.
func TestInvokeII_ASanctumMarkedForAnotherAspectChanges(t *testing.T) {
	fx := newFixture(t)
	vessel := fx.vessel("niri.kdl", sanctum.Render(sanctum.Marker{Glyph: "#", Ward: "gaming"}, "", "off", "same"))
	fx.rite("gaming", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "niri", "designation": "sanctum", "vessel": "$DATA/niri.kdl", "ward": "gaming", "glyph": "#",
	   "omens": {"*": "same"}}]}`)

	res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "gaming", Aspect: "on"})

	if err != nil || res.Outcome == nil || !slices.Equal(res.Outcome.Changed, []string{"niri"}) {
		t.Fatalf("invocation: %v %+v", err, res)
	}
	if want := sanctum.Render(sanctum.Marker{Glyph: "#", Ward: "gaming"}, "", "on", "same"); fx.read(vessel) != want {
		t.Fatalf("vessel %q, want %q", fx.read(vessel), want)
	}
}

// TestInvokeII_AFailedProofRestoresWhatWasRead: a subject that shows another
// omen than the one commanded fells the invocation; every subject set is
// restored, in reverse order, the fallen one too, to what was read before
// (a machine-spirit by enacting the omen read and proving it), and the
// tidings name the subject, the omen commanded and the omen found.
func TestInvokeII_AFailedProofRestoresWhatWasRead(t *testing.T) {
	fx := newFixture(t)
	fx.rite("power", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "governor", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"},
	   "auspex": {"scan": "cat $DATA/governor"}, "enact": {"*": "echo {{omen}} > $DATA/governor"}},
	  {"name": "theme", "designation": "tether", "tether": "$DATA/current", "omens": {"*": "$DATA/{{aspect}}"}},
	  {"name": "boost", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"},
	   "auspex": {"scan": "cat $DATA/boost"},
	   "enact": {"1": "echo 2 > $DATA/boost", "*": "echo {{omen}} > $DATA/boost"}}]}`)
	fx.vessel("governor", "0\n")
	fx.vessel("boost", "0\n")
	if err := os.Symlink(filepath.Join(fx.data, "off"), filepath.Join(fx.data, "current")); err != nil {
		t.Fatal(err)
	}
	h := &proclaimed{}

	res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "power", Aspect: "on", Herald: h})

	if err != nil || res.Outcome == nil {
		t.Fatalf("invocation: %v %+v", err, res)
	}
	out := res.Outcome
	if out.Verdict != invocation.Reverted || out.Fell == nil || out.Fell.Subject != "boost" {
		t.Fatalf("outcome %+v", out)
	}
	if want := `the subject "boost" shows the omen "2"; "1" was commanded`; !strings.Contains(out.Fell.Heresy.Error(), want) {
		t.Fatalf("the heresy %q does not tell %q", out.Fell.Heresy, want)
	}
	grimdark(t, out.Fell.Heresy.Error())
	var reverted []string
	for _, r := range out.Reversions {
		reverted = append(reverted, r.Subject)
		if r.Heresy != nil {
			t.Errorf("the reversion of %s fell: %v", r.Subject, r.Heresy)
		}
	}
	if !slices.Equal(reverted, []string{"boost", "theme", "governor"}) {
		t.Fatalf("reverted %q", reverted)
	}
	if fx.read(filepath.Join(fx.data, "governor")) != "0\n" || fx.read(filepath.Join(fx.data, "boost")) != "0\n" {
		t.Fatal("the subjects were not restored to the omens read")
	}
	if a, _ := os.Readlink(filepath.Join(fx.data, "current")); a != filepath.Join(fx.data, "off") {
		t.Fatalf("the tether leads to %q", a)
	}
	last := h.heard[len(h.heard)-1]
	if last.Tidings != invocation.Failure || last.Values.Tidings.Subject != "boost" ||
		!strings.Contains(last.Values.Tidings.Heresy, `"2"; "1" was commanded`) {
		t.Fatalf("the herald heard %+v", last)
	}
	if sl := fx.slate("power"); sl.Aspect != "off" || sl.LastRite.Verdict != invocation.Reverted {
		t.Fatalf("data-slate %+v", sl)
	}
	line := strings.TrimSpace(fx.read(filepath.Join(fx.state, "servitor", "chronicle.jsonl")))
	for _, want := range []string{`"verdict":"reverted"`, `"fell_at":{"subject":"boost"}`, `"changed":["governor","theme"]`,
		`"reversions":[{"subject":"boost","verdict":"triumph"},{"subject":"theme","verdict":"triumph"},{"subject":"governor","verdict":"triumph"}]`} {
		if !strings.Contains(line, want) {
			t.Errorf("the chronicle lacks %s:\n%s", want, line)
		}
	}
}

// unreadII is a state rite whose subject "u" has no auspex: it cannot be
// read, so it is set at every invocation and never proven; "z" falls when
// set into the aspect on.
const unreadII = `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
  {"name": "a", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"},
   "auspex": {"scan": "cat $DATA/a"}, "enact": {"*": "echo {{omen}} > $DATA/a"}},
  {"name": "u", "designation": "machine-spirit", "omens": {"on": "up", "*": "down"},
   "enact": {"*": "echo {{omen}} > $DATA/u"}},
  {"name": "z", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"},
   "auspex": {"scan": "cat $DATA/z"}, "enact": {"1": "exit 3", "*": "echo {{omen}} > $DATA/z"}}]}`

// TestInvokeII_AnUnreadSpiritIsRestoredByTheFormerAspect: a machine-spirit
// that could not be read is restored by enacting the omen of the former
// aspect; with no former aspect it cannot be restored, and the invocation
// falters.
func TestInvokeII_AnUnreadSpiritIsRestoredByTheFormerAspect(t *testing.T) {
	t.Run("restored", func(t *testing.T) {
		fx := newFixture(t)
		fx.rite("unread", unreadII)
		fx.vessel("a", "0\n")
		fx.vessel("z", "0\n")

		res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "unread", Aspect: "on"})

		if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Reverted || res.Options.Former != "off" {
			t.Fatalf("invocation: %v %+v", err, res)
		}
		if res.Outcome.Fell.Subject != "z" || !strings.Contains(res.Outcome.Fell.Heresy.Error(), "death-mark 3") {
			t.Fatalf("fell %+v", res.Outcome.Fell)
		}
		if fx.read(filepath.Join(fx.data, "u")) != "down\n" || fx.read(filepath.Join(fx.data, "a")) != "0\n" ||
			fx.read(filepath.Join(fx.data, "z")) != "0\n" {
			t.Fatal("the subjects were not restored")
		}
	})
	t.Run("faltered", func(t *testing.T) {
		fx := newFixture(t)
		fx.rite("unread", unreadII) // nothing can be read: no former aspect

		res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "unread", Aspect: "on"})

		if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Faltered || res.Options.Former != "" {
			t.Fatalf("invocation: %v %+v", err, res)
		}
		var u *invocation.Reversion
		for i, r := range res.Outcome.Reversions {
			if r.Subject == "u" {
				u = &res.Outcome.Reversions[i]
			}
		}
		if u == nil || u.Heresy == nil || !strings.Contains(u.Heresy.Error(), "former aspect") {
			t.Fatalf("reversions %+v", res.Outcome.Reversions)
		}
		grimdark(t, u.Heresy.Error())
		if fx.read(filepath.Join(fx.data, "u")) != "up\n" {
			t.Fatal("the unread subject was touched by its reversion")
		}
		if sl := fx.slate("unread"); sl.LastRite.Verdict != invocation.Faltered {
			t.Fatalf("data-slate %+v", sl)
		}
	})
}

// TestInvokeII_ForeseesEverySubjectWithoutPerforming: foresight reads every
// subject, tells for each the omen read, the omen commanded, whether it
// changes and what would be performed, and sets nothing.
func TestInvokeII_ForeseesEverySubjectWithoutPerforming(t *testing.T) {
	fx := newFixture(t)
	fx.rite("gaming", gamingII)
	gamingOff(fx)
	fx.vessel("scheduler", "cosmos\n") // already in the aspect on

	res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "gaming", Aspect: "on", Foresee: true})

	if err != nil || res.Outcome != nil || res.Entry != nil || len(res.Foresight) != 4 {
		t.Fatalf("foresight: %v %+v", err, res)
	}
	smt, sched, theme, conf := res.Foresight[0], res.Foresight[1], res.Foresight[2], res.Foresight[3]
	if smt.Subject != "smt" || smt.Plan == nil || smt.Plan.Read != "on" || smt.Plan.Target != "off" ||
		!smt.Plan.Changing || smt.Speech == nil || !strings.Contains(smt.Speech.Words, "echo off >") {
		t.Fatalf("smt %+v %+v", smt, smt.Plan)
	}
	if sched.Plan == nil || sched.Plan.Read != "cosmos" || sched.Plan.Changing || sched.Speech != nil {
		t.Fatalf("scheduler %+v %+v", sched, sched.Plan)
	}
	if theme.Plan == nil || !theme.Plan.Changing || theme.Tether == nil || conf.Plan == nil || !conf.Plan.Changing ||
		conf.Vessel == nil {
		t.Fatalf("theme %+v, conf %+v", theme, conf)
	}
	if exists(filepath.Join(fx.data, "log")) || exists(filepath.Join(fx.data, "conf")) ||
		fx.read(filepath.Join(fx.data, "smt")) != "on\n" {
		t.Fatal("foresight touched the machine")
	}
	if _, err := os.Stat(filepath.Join(fx.state, "servitor")); err == nil {
		t.Fatal("foresight kept something in the state home")
	}
}

// TestInvokeII_OmensOfTheAspectInvokedSpeakItsInscriptions: the omen
// commanded is rendered with the inscriptions of the invocation, both when
// it is enacted and when it is proven, whatever the data-slate recorded.
func TestInvokeII_OmensOfTheAspectInvokedSpeakItsInscriptions(t *testing.T) {
	fx := newFixture(t)
	fx.rite("power", `{"pattern": "Mark II", "aspects": ["on", "off"],
	  "inscriptions": {"governor": {"decrees": {"on": "performance", "off": "powersave"}}},
	  "subjects": [{"name": "governor", "designation": "machine-spirit", "omens": {"*": "{{inscription.governor}}"},
	   "auspex": {"scan": "cat $DATA/governor"}, "enact": {"*": "echo \"$SERVITOR_OMEN\" > $DATA/governor"}}]}`)
	fx.vessel("governor", "powersave\n")
	s := fx.servitor()

	for _, governor := range []string{"schedutil", "performance"} {
		res, err := s.Invoke(context.Background(), Petition{Rite: "power", Aspect: "on",
			Runes: map[string]string{"governor": governor}})
		if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Triumph {
			t.Fatalf("%s: %v %+v", governor, err, res.Outcome)
		}
		if got := fx.read(filepath.Join(fx.data, "governor")); got != governor+"\n" {
			t.Fatalf("the governor holds %q, want %q", got, governor)
		}
	}
}
