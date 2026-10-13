package exalted

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// stateShrine is the world of a trial of a state rite of pattern Mark II,
// "hallow": its plans hold the machine-spirit smt (place 1), whose enact
// words write its omen to <hall>/smt and tell it in <hall>/said; the
// tether theme (place 3; place 2 is a common subject); and the observances
// prep (the introit of places 1 and 3), post (their recessional, reverted
// by word) and late (the recessional of place 4, a common subject, admitted
// only out of "off"). The hall holds what exalted speech wrote.
func stateShrine(t *testing.T) (*shrine, string) {
	t.Helper()
	bash, hall := tongue(t), t.TempDir()
	say := func(words string) string { return words + " >> " + filepath.Join(hall, "said") }
	enact := map[string]string{}
	for _, omen := range []string{"on", "off"} {
		enact[omen] = "echo " + omen + " > " + filepath.Join(hall, "smt") + "; " +
			say(`echo "enact $SERVITOR_OMEN"`)
	}
	observance := func(name, words, reversion string) sanction.Verse {
		return sanction.Verse{Observance: name, Kind: librarium.KindIncantation, Target: say(words),
			Reversion: reversion, Tongue: bash, Patience: patience, Relics: []sanction.Relic{},
			Introit: []int{}, Recessional: []int{}}
	}
	plan := func(aspect, omen string) sanction.Plan {
		prep, post, late := observance("prep", `echo "prep omen=${SERVITOR_OMEN-unheard}"`, ""),
			observance("post", "echo post", say("echo unpost")), observance("late", "echo late", "")
		prep.Introit, post.Recessional, late.Recessional, late.From = []int{0, 2}, []int{0, 2}, []int{3}, []string{"off"}
		return sanction.Plan{Aspect: aspect, Inscriptions: map[string]string{},
			Subjects: []sanction.Verse{
				{Subject: "smt", Place: 0, Omen: omen, Enact: enact, Tongue: bash, Patience: patience,
					Relics: []sanction.Relic{}},
				{Subject: "theme", Place: 2, Kind: librarium.KindTether, Target: "/etc/hallow-theme",
					Anchor: "/usr/share/hallow/" + aspect},
			},
			Observances: []sanction.Verse{prep, post, late}}
	}
	u := &sanction.Unit{Pattern: sanction.PatternMarkII, UID: me, Rite: "hallow",
		Environment: sanction.Environment{Pattern: sanction.PatternMarkII, Keys: []string{"SERVITOR_ASPECT",
			"SERVITOR_FORMER_ASPECT", "SERVITOR_INSCRIPTION_WHO", "SERVITOR_OMEN", "SERVITOR_RITE"}},
		Aspects: []sanction.Plan{plan("on", "off"), plan("off", "on")}}
	return speakingShrine(t, u), hall
}

func subject(name string) *Item         { return &Item{Subject: name} }
func spirit(name, restore string) *Item { return &Item{Subject: name, Omen: restore} }
func observance(name string) *Item      { return &Item{Observance: name} }
func numbered(n int) *Item              { return &Item{Verse: n} }
func lines(t *testing.T, p string) []string {
	t.Helper()
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil
	}
	must(t, err)
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

// TestSessionII_AcceptsThePlanOfAStateRite: the session of a state rite
// names the subjects and observances of its plan on acceptance, and
// performs each as it is named: a machine-spirit by the enact words of the
// omen of the plan's aspect, which alone hear the omen as SERVITOR_OMEN;
// a tether from the sanction; an observance as its speech.
func TestSessionII_AcceptsThePlanOfAStateRite(t *testing.T) {
	sh, hall := stateShrine(t)
	inv := sh.rouseFor("on")
	r := inv.say(sh.handshake("on"))
	if r.Verdict != Accepted || r.Verses != nil || !slices.Equal(r.Subjects, []string{"smt", "theme"}) ||
		!slices.Equal(r.Observances, []string{"prep", "post", "late"}) {
		t.Fatalf("accepted with %+v", r)
	}
	for _, it := range []*Item{observance("prep"), spirit("smt", "on"), subject("theme"), observance("post")} {
		r := inv.say(Request{Perform: it})
		if r.To != "perform" || r.Verdict != Triumph || r.Subject != it.Subject || r.Observance != it.Observance ||
			r.Changed != (it.Observance != "prep") {
			t.Fatalf("%+v: %+v", it, r)
		}
	}
	if got := lines(t, filepath.Join(hall, "said")); !slices.Equal(got, []string{"prep omen=unheard", "enact off",
		"post"}) {
		t.Errorf("exalted speech said %q", got)
	}
	if got := lines(t, filepath.Join(hall, "smt")); !slices.Equal(got, []string{"off"}) {
		t.Errorf("smt holds %q", got)
	}
	sh.holds(map[string]string{"/etc/hallow-theme": "→ /usr/share/hallow/on"})
	if r := inv.say(Request{Finish: new(true)}); r.Verdict != Finished {
		t.Fatalf("finish: %+v", r)
	}
	if m := inv.ended(); m != 0 {
		t.Fatalf("finished with the death-mark %d", m)
	}
	want := []string{"accept accepted", "perform prep triumph", "perform smt triumph", "perform theme triumph",
		"perform post triumph", "finish finished"}
	if got := sh.journal.deeds(); !slices.Equal(got, want) {
		t.Errorf("journaled %q, want %q", got, want)
	}
}

// TestSessionII_SkipsForwardNeverBack: an item of a state rite may be
// performed only after every item performed before it — subjects by their
// place, an observance at the place of a subject that names it, each at
// most once — so unchanged subjects and unadmitted observances may be
// skipped, and nothing may be performed again or out of its place. An
// observance whose "from" does not admit the former aspect, an item no
// plan holds, a verse, and an omen of restoration the machine-spirit does
// not declare are refused too. A refused line ends the session, reverting
// what was performed.
func TestSessionII_SkipsForwardNeverBack(t *testing.T) {
	for what, c := range map[string]struct {
		before []*Item
		wrong  *Item
		former string // the former aspect of the handshake; "" is "off"
	}{
		"a subject before one performed":        {before: []*Item{subject("theme")}, wrong: spirit("smt", "on")},
		"an introit after its subjects":         {before: []*Item{subject("theme")}, wrong: observance("prep")},
		"an observance performed twice":         {before: []*Item{observance("prep")}, wrong: observance("prep")},
		"a subject performed twice":             {before: []*Item{spirit("smt", "on")}, wrong: spirit("smt", "on")},
		"an observance of a later place":        {before: []*Item{observance("late")}, wrong: subject("theme")},
		"an observance its from does not admit": {wrong: observance("late"), former: "on"},
		"a subject no plan holds":               {wrong: subject("common")},
		"an observance no plan holds":           {wrong: observance("hidden")},
		"a verse of a state rite":               {wrong: numbered(1)},
		"an undeclared omen of restoration":     {wrong: spirit("smt", "auto")},
		"an omen named for a tether":            {wrong: spirit("theme", "on")},
		"an omen named for an observance":       {wrong: &Item{Observance: "prep", Omen: "on"}},
		"a subject and observance at once":      {wrong: &Item{Subject: "smt", Observance: "prep"}},
	} {
		t.Run(what, func(t *testing.T) {
			sh, _ := stateShrine(t)
			inv := sh.rouseFor("on")
			h := sh.handshake("on")
			if c.former != "" {
				h.Former = c.former
			}
			if r := inv.say(h); r.Verdict != Accepted {
				t.Fatalf("not accepted: %+v", r)
			}
			for _, it := range c.before {
				if r := inv.say(Request{Perform: it}); r.Verdict != Triumph {
					t.Fatalf("%+v: %+v", it, r)
				}
			}
			r := inv.say(Request{Perform: c.wrong})
			if r.Verdict != Refused || r.To != "perform" || r.Lament == "" {
				t.Fatalf("heard %+v, want a refusal", r)
			}
			if m := inv.ended(); m != 1 {
				t.Fatalf("ended with the death-mark %d, want 1", m)
			}
			sh.holds(map[string]string{"/etc/hallow-theme": "absent"})
		})
	}
	// Each legal path is performed whole.
	for _, path := range [][]*Item{
		{observance("prep"), spirit("smt", "on"), subject("theme"), observance("post"), observance("late")},
		{observance("prep"), subject("theme"), observance("post")},
		{spirit("smt", "on"), observance("post"), observance("late")},
		{observance("late")},
		{spirit("smt", ""), observance("prep"), subject("theme")},
	} {
		sh, _ := stateShrine(t)
		inv := sh.accepted("on")
		for _, it := range path {
			if r := inv.say(Request{Perform: it}); r.Verdict != Triumph {
				t.Fatalf("%v: %+v refused: %+v", path, it, r)
			}
		}
		inv.say(Request{Finish: new(true)})
		inv.ended()
	}
}

// TestSessionII_RestoresTheOmenNamed: a machine-spirit is reverted by the
// enact words of the declared omen its revert names, or else of the omen
// named when it was set; an undeclared one is refused, and so is the revert
// of another item than the last performed.
func TestSessionII_RestoresTheOmenNamed(t *testing.T) {
	sh, hall := stateShrine(t)
	inv := sh.accepted("on")
	inv.say(Request{Perform: spirit("smt", "on")})
	inv.say(Request{Perform: observance("post")})
	if r := inv.say(Request{Revert: observance("post")}); r.Verdict != Reverted || r.Observance != "post" ||
		r.To != "revert" {
		t.Fatalf("revert post: %+v", r)
	}
	if r := inv.say(Request{Revert: spirit("smt", "off")}); r.Verdict != Reverted || r.Subject != "smt" || !r.Changed {
		t.Fatalf("revert smt: %+v", r)
	}
	if got := lines(t, filepath.Join(hall, "said")); !slices.Equal(got, []string{"enact off", "post", "unpost",
		"enact off"}) {
		t.Errorf("exalted speech said %q", got)
	}
	inv.say(Request{Abort: new(true)})
	inv.ended()

	for what, c := range map[string]struct{ before, wrong *Item }{
		"an undeclared omen":        {before: spirit("smt", "on"), wrong: spirit("smt", "auto")},
		"another item":              {before: spirit("smt", "on"), wrong: subject("theme")},
		"an omen for an observance": {before: observance("prep"), wrong: &Item{Observance: "prep", Omen: "on"}},
		"a verse":                   {before: spirit("smt", "on"), wrong: numbered(1)},
	} {
		t.Run(what, func(t *testing.T) {
			sh, _ := stateShrine(t)
			inv := sh.accepted("on")
			if r := inv.say(Request{Perform: c.before}); r.Verdict != Triumph {
				t.Fatalf("%+v", r)
			}
			if r := inv.say(Request{Revert: c.wrong}); r.Verdict != Refused || r.To != "revert" {
				t.Fatalf("heard %+v, want a refusal", r)
			}
			if m := inv.ended(); m != 1 {
				t.Fatalf("ended with the death-mark %d, want 1", m)
			}
		})
	}
}

// TestSessionII_TheEndOfTheChannelRestoresByItself: when the invoker's
// servitor falls before finish, the exalted servitor reverts every item it
// performed by itself, in reverse: a machine-spirit by the enact words of
// the omen named when it was set. One set with no such omen cannot be
// restored, and the session falters.
func TestSessionII_TheEndOfTheChannelRestoresByItself(t *testing.T) {
	sh, hall := stateShrine(t)
	inv := sh.accepted("on")
	for _, it := range []*Item{spirit("smt", "on"), subject("theme"), observance("post")} {
		inv.say(Request{Perform: it})
	}
	must(t, inv.conn.Close())
	if m := inv.ended(); m != 1 {
		t.Fatalf("ended with the death-mark %d, want 1", m)
	}
	if got := lines(t, filepath.Join(hall, "smt")); !slices.Equal(got, []string{"on"}) {
		t.Errorf("smt holds %q", got)
	}
	sh.holds(map[string]string{"/etc/hallow-theme": "absent"})
	want := []string{"accept accepted", "perform smt triumph", "perform theme triumph", "perform post triumph",
		"revert post reverted", "revert theme reverted", "revert smt reverted", "fall reverted"}
	if got := sh.journal.deeds(); !slices.Equal(got, want) {
		t.Errorf("journaled %q, want %q", got, want)
	}

	sh, _ = stateShrine(t)
	inv = sh.accepted("on")
	inv.say(Request{Perform: spirit("smt", "")})
	must(t, inv.conn.Close())
	if m := inv.ended(); m != 3 {
		t.Fatalf("an unrestorable spirit ended with the death-mark %d, want 3", m)
	}
}

// TestSanctionII_LaysTheUnitOfAStateRite: the unit of a state rite is
// sanctioned as any other; one holding no exalted item in any plan,
// subjects out of their written order, an observance named twice or a
// machine-spirit without the enact words of its omen is refused, and
// nothing is written.
func TestSanctionII_LaysTheUnitOfAStateRite(t *testing.T) {
	unit := func(t *testing.T) (*shrine, *sanction.Unit) {
		state, _ := stateShrine(t)
		sh, _ := keepingShrine(t)
		sh.cfg.Voice, sh.cfg.Weigh = state.cfg.Voice, sh.speech().Weigh
		return sh, state.unit
	}
	sh, u := unit(t)
	inv := sh.sanctionFor(u, "")
	if r := inv.answer(); r.Verdict != Sanctioned {
		t.Fatalf("answered %+v", r)
	}
	inv.ended()
	if kept, err := sh.store().Read("hallow"); err != nil || kept == nil || kept.Sigil != u.Sigil() ||
		kept.Pattern != sanction.PatternMarkII {
		t.Fatalf("the store keeps %+v, %v", kept, err)
	}
	for what, c := range map[string]struct {
		mar    func(u *sanction.Unit)
		lament string
	}{
		"nothing in any plan": {func(u *sanction.Unit) {
			for i := range u.Aspects {
				u.Aspects[i].Subjects, u.Aspects[i].Observances = nil, nil
			}
		}, "no exalted subject or observance"},
		"subjects out of order": {func(u *sanction.Unit) { u.Aspects[0].Subjects[1].Place = 0 }, "written order"},
		"an observance twice": {func(u *sanction.Unit) {
			u.Aspects[1].Observances = append(u.Aspects[1].Observances, u.Aspects[1].Observances[0])
		}, `the observance "prep" twice`},
		"no words of its omen": {func(u *sanction.Unit) { u.Aspects[1].Subjects[0].Omen = "auto" }, `"auto"`},
	} {
		t.Run(what, func(t *testing.T) {
			sh, u := unit(t)
			c.mar(u)
			inv := sh.sanctionFor(u, "")
			if r := inv.answer(); r.Verdict != Refused || !strings.Contains(r.Lament, c.lament) {
				t.Fatalf("answered %+v, want a refusal naming %q", r, c.lament)
			}
			if m := inv.ended(); m != 2 {
				t.Errorf("refused with the death-mark %d", m)
			}
			if _, err := os.Lstat(sh.store().Path(u.Rite)); err == nil {
				t.Error("a sanction was laid")
			}
		})
	}
}
