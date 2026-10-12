package rituals

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// splitLock is a rite whose third verse is exalted; a common sanctum and
// incantation stand before it, and its auspex marks that it was heard.
const splitLock = `{
  "pattern": "Mark I",
  "aspects": ["on", "off"],
  "auspex": "touch $DATA/augured; echo off",
  "liturgy": [
    {"sanctum": "$DATA/util.kdl", "scripture": {"on": "split-lock 0", "off": ""}},
    {"incantation": "echo common >> $DATA/said"},
    {"incantation": "echo exalted >> $DATA/said", "exalted": true},
    {"vox-cast": "success"}
  ]
}`

// reboot is an edict whose only verse is exalted.
const reboot = `{"pattern": "Mark I", "edict": true, "liturgy": [
  {"vox-cast": "progress"},
  {"incantation": "echo exalted >> $DATA/said", "exalted": true}]}`

// TestInvoke_RefusesEveryExaltedVerse: no exalted verse is performed with
// the invoker's authority. The invocation is refused with a heresy for
// every exalted verse before anything is touched — no auspex, no dominion,
// no vessel, no speech, no data-slate, no chronicle — while foresight
// still reveals what it would do.
func TestInvoke_RefusesEveryExaltedVerse(t *testing.T) {
	for _, c := range []struct{ rite, scripture, aspect string }{{"split-lock", splitLock, "on"}, {"reboot", reboot, ""}} {
		t.Run(c.rite, func(t *testing.T) {
			fx := newFixture(t)
			util := fx.vessel("util.kdl", "input {}\n")
			fx.rite(c.rite, c.scripture)
			s := fx.servitor()

			res, err := s.Invoke(context.Background(), Petition{Rite: c.rite, Aspect: c.aspect, Herald: &herald{}})

			var hs invocation.Heresies
			if !errors.As(err, &hs) || len(hs) != 1 {
				t.Fatalf("err %v, want one heresy", err)
			}
			verse := 3
			if c.rite == "reboot" {
				verse = 2
			}
			if h := hs[0]; h.Verse != verse || h.Rite != c.rite || h.Line != verse+5 && c.rite == "split-lock" ||
				!strings.Contains(h.Message, "sanction") || !strings.Contains(h.Message, "exalted") {
				t.Fatalf("heresy %+v", h)
			}
			grimdark(t, hs[0].Message)
			if res.Rite == nil || res.Outcome != nil || res.Entry != nil || res.Options.Aspect != c.aspect {
				t.Fatalf("result %+v", res)
			}
			for _, p := range []string{filepath.Join(fx.data, "said"), filepath.Join(fx.data, "augured"),
				filepath.Join(fx.state, "servitor")} {
				if exists(p) {
					t.Errorf("%s was touched", p)
				}
			}
			if got := fx.read(util); got != "input {}\n" {
				t.Errorf("the vessel was touched:\n%s", got)
			}

			res, err = s.Invoke(context.Background(), Petition{Rite: c.rite, Aspect: c.aspect, Foresee: true})
			if err != nil || len(res.Foresight) == 0 {
				t.Fatalf("foresight: %v %+v", err, res)
			}
			if exists(filepath.Join(fx.data, "said")) || exists(filepath.Join(fx.state, "servitor")) {
				t.Error("foresight touched the machine")
			}
		})
	}
}

func TestInvoke_PerformsARiteWithoutExaltedVerses(t *testing.T) {
	fx := newFixture(t)
	fx.rite("plain", `{"pattern": "Mark I", "aspects": ["on"], "liturgy": [
	  {"incantation": "echo common >> $DATA/said", "exalted": false}]}`)
	res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "plain", Aspect: "on"})
	if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Triumph {
		t.Fatalf("%v %+v", err, res)
	}
}

func TestInquire_TheRelicsAndScrollsOfExaltedVerses(t *testing.T) {
	fx := newFixture(t)
	tune := fx.write(fx.lib, "rites/scrolls/tune", "#!/bin/sh\n. ./scrolls/tune.conf\ncat $HOME/x\ncp $HOME/y /etc\n")
	fx.write(fx.lib, "rites/scrolls/tune.conf", "x=1\n")
	common := fx.write(fx.lib, "rites/scrolls/common", "#!/bin/sh\nsudo sysctl -w x=1\n")
	tome := fx.write(fx.lib, "tomes/exalted.conf", "was {{former}}\n")
	outside := fx.write(fx.data, "outside.conf", "x=2\n")
	if err := os.Symlink(outside, filepath.Join(fx.lib, "rites", "scrolls", "astray.conf")); err != nil {
		t.Fatal(err)
	}
	rite := fx.rite("tune", `{"pattern": "Mark I", "aspects": ["on", "off"], "liturgy": [
	  {"litany": "scrolls/tune", "exalted": true,
	   "relics": ["scrolls/tune.conf", "scrolls/lost.conf", "scrolls/astray.conf", "scrolls"]},
	  {"litany": "scrolls/common"},
	  {"transcription": "/etc/tune.conf", "scripture": {"tome": "../tomes/exalted.conf", "illuminate": true}, "exalted": true},
	  {"incantation": "./gone", "exalted": true, "relics": ["gone"]}
	]}`)

	fs := inquire(t, fx.servitor(), true)

	expect := []struct {
		sev       librarium.Severity
		words     []string
		scripture string
		line, col int
	}{
		{librarium.Heresy, []string{"relic 2 of the litany of verse 1", "scrolls/lost.conf", "nowhere to be found"}, rite, 3, 37},
		{librarium.Heresy, []string{"relic 3 of the litany of verse 1", "scrolls/astray.conf", "outside the hall"}, rite, 3, 58},
		{librarium.Heresy, []string{"relic 4 of the litany of verse 1", "no scripture"}, rite, 3, 81},
		{librarium.Impurity, []string{"within the scroll", "the exalted speech of verse 1 reaches into the home"}, tune, 3, 1},
		{librarium.Impurity, []string{"within the scroll", `verse 2 calls upon the intercessor "sudo"`}, common, 2, 1},
		{librarium.Heresy, []string{"within the illuminated tome", `"$SERVITOR_FORMER_ASPECT"`}, tome, 1, 5},
		{librarium.Heresy, []string{"relic 1 of the incantation of verse 4, gone,", "nowhere to be found"}, rite, 6, 58},
	}
	for _, e := range expect {
		got := denounced(fs, "tune", e.sev, e.words...)
		if len(got) != 1 {
			t.Errorf("%v %q: found %d in\n%v", e.sev, e.words, len(got), fs)
			continue
		}
		if f := got[0]; f.Scripture != e.scripture || f.Line != e.line || f.Column != e.col {
			t.Errorf("%q found at %s:%d:%d, want %s:%d:%d", e.words, f.Scripture, f.Line, f.Column, e.scripture, e.line, e.col)
		}
	}
	if len(fs) != len(expect) {
		t.Errorf("found %d, expected %d:\n%v", len(fs), len(expect), fs)
	}
}
