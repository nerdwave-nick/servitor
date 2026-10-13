package rituals

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// sought is one finding a trial of the inquisition expects: its rite, its
// severity, words its denunciation speaks, and where it is found — the
// scripture and the first place the needle is written there.
type sought struct {
	rite      string
	sev       librarium.Severity
	words     []string
	scripture string
	needle    string
}

// placeOf returns the line and column (counted from one) of the first
// needle written in the scripture at p.
func placeOf(t *testing.T, p, needle string) (int, int) {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	i := strings.Index(string(data), needle)
	if i < 0 {
		t.Fatalf("%q is not written in %s", needle, p)
	}
	before := string(data[:i])
	return strings.Count(before, "\n") + 1, i - strings.LastIndex(before, "\n")
}

// expectFindings checks that each sought finding is found exactly once in
// fs, at its place.
func expectFindings(t *testing.T, fs librarium.Findings, want []sought) {
	t.Helper()
	for _, w := range want {
		got := denounced(fs, w.rite, w.sev, w.words...)
		if len(got) != 1 {
			t.Errorf("%s %v %q: found %d in\n%v", w.rite, w.sev, w.words, len(got), fs)
			continue
		}
		line, col := placeOf(t, w.scripture, w.needle)
		if f := got[0]; f.Scripture != w.scripture || f.Line != line || f.Column != col {
			t.Errorf("%q found at %s:%d:%d, want %s:%d:%d", w.words, f.Scripture, f.Line, f.Column, w.scripture, line, col)
		}
	}
}

// TestInquireII_TheMachineTheSubjectsAndObservancesActUpon: the machine of a
// state rite is examined per subject and observance, as the liturgy of
// Mark I is per verse — a sanctum subject's vessel that may not be
// consecrated, a tether subject's anchor missing for some aspect, the
// tongues of machine-spirits, of their auspexes and of observances — each
// denounced where the scripture writes it. --spare-vessels spares them.
func TestInquireII_TheMachineTheSubjectsAndObservancesActUpon(t *testing.T) {
	fx := newFixture(t)
	if err := os.Chmod(fx.vessel("run.sh", "#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(fx.data, "themes", "on"), 0o755); err != nil {
		t.Fatal(err)
	}
	machine := fx.rite("machine", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "absent", "designation": "sanctum", "vessel": "$DATA/absent.kdl", "omens": {"on": "x", "*": ""}},
	  {"name": "raised", "designation": "sanctum", "vessel": "$DATA/raised.kdl", "consecrate": true, "omens": {"on": "x", "*": ""}},
	  {"name": "theme", "designation": "tether", "tether": "$DATA/current", "omens": {"*": "$DATA/themes/{{aspect}}"}},
	  {"name": "smt", "designation": "machine-spirit", "tongue": "no-such-tongue-of-spirits", "omens": {"on": "1", "off": "0"},
	   "auspex": {"scan": "cat x"}, "enact": {"*": "echo {{omen}}"}, "recessional": ["herald", "recite"]}],
	  "observances": [
	  {"name": "herald", "designation": "incantation", "tongue": "no-such-tongue-of-heralds", "words": "true"},
	  {"name": "recite", "designation": "litany", "scroll": "$DATA/run.sh", "reversion": "true", "tongue": "no-such-tongue-of-scrolls"},
	  {"name": "quiet", "designation": "litany", "scroll": "$DATA/run.sh", "tongue": "no-such-tongue-of-silence"}]}`)
	ritely := fx.rite("ritely", `{"pattern": "Mark II", "aspects": ["on", "off"], "tongue": "no-such-tongue-of-rites", "subjects": [
	  {"name": "mode", "designation": "machine-spirit", "auspex": {"trials": {"on": "true"}}, "enact": {"*": "true"}}]}`)
	fx.vessel("broken.kdl", "// +++ begin of sanctum broken -- aspect|on +++\nx\n")
	broken := fx.rite("broken", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "util", "designation": "sanctum", "vessel": "$DATA/broken.kdl", "omens": {"on": "x", "*": ""}}]}`)

	fs := inquire(t, fx.servitor(), false)

	expectFindings(t, fs, []sought{
		{"machine", librarium.Impurity, []string{filepath.Join(fx.data, "absent.kdl"), `the sanctum subject "absent"`,
			`"consecrate"`}, machine, `"` + fx.data + `/absent.kdl"`},
		{"machine", librarium.Impurity, []string{filepath.Join(fx.data, "themes", "off"), `the tether subject "theme"`,
			filepath.Join(fx.data, "current"), `aspect "off"`}, machine, `"` + fx.data + `/themes/{{aspect}}"`},
		{"machine", librarium.Impurity, []string{`"no-such-tongue-of-spirits"`, `the subject "smt"`, "could never be set"},
			machine, `"no-such-tongue-of-spirits"`},
		{"machine", librarium.Impurity, []string{`"no-such-tongue-of-spirits"`, `the auspex of the subject "smt"`,
			"could never be heard"}, machine, `{"scan"`},
		{"machine", librarium.Impurity, []string{`"no-such-tongue-of-heralds"`, `the observance "herald"`}, machine,
			`"no-such-tongue-of-heralds"`},
		{"machine", librarium.Impurity, []string{`"no-such-tongue-of-scrolls"`, `the observance "recite"`}, machine,
			`"no-such-tongue-of-scrolls"`},
		{"ritely", librarium.Impurity, []string{`"no-such-tongue-of-rites"`, `the subject "mode"`, "could never be set"},
			ritely, `"no-such-tongue-of-rites"`},
		{"ritely", librarium.Impurity, []string{`"no-such-tongue-of-rites"`, `the auspex of the subject "mode"`}, ritely,
			`{"trials"`},
		{"broken", librarium.Heresy, []string{filepath.Join(fx.data, "broken.kdl"), "defiled"}, broken,
			`"` + fx.data + `/broken.kdl"`},
	})
	if got := denounced(fs, "machine", librarium.Impurity, "no-such-tongue-of-silence"); len(got) != 0 {
		t.Errorf("a litany that needs no tongue was denounced for its tongue: %v", got)
	}
	if got := denounced(fs, "machine", librarium.Impurity, "raised.kdl"); len(got) != 0 {
		t.Errorf("a consecrating sanctum subject was denounced: %v", got)
	}
	// The observance "quiet" is named by no subject (an impurity of the
	// scripture); nothing else is found.
	if len(fs) != 10 {
		t.Errorf("found %d, expected 10:\n%v", len(fs), fs)
	}
	for _, f := range fs {
		if strings.Contains(f.Message, "verse") {
			t.Errorf("a state rite was denounced by verse: %v", f)
		}
	}
	if spared := inquire(t, fx.servitor(), true); len(spared) != 1 {
		t.Fatalf("sparing the vessels:\n%v", spared)
	}
}

// TestInquireII_ADesecratedStateRite: a state rite whose subjects agree upon
// an aspect its data-slate does not record is desecrated.
func TestInquireII_ADesecratedStateRite(t *testing.T) {
	fx := newFixture(t)
	for _, a := range []string{"on", "off"} {
		if err := os.MkdirAll(filepath.Join(fx.data, "themes", a), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(fx.data, "themes", "off"), filepath.Join(fx.data, "current")); err != nil {
		t.Fatal(err)
	}
	theme := fx.rite("theme", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "theme", "designation": "tether", "tether": "$DATA/current", "omens": {"*": "$DATA/themes/{{aspect}}"}}]}`)
	if err := augury.WriteSlate(os.LookupEnv, "theme", augury.Slate{Aspect: "on",
		LastRite: &augury.LastRite{Verdict: invocation.Triumph, At: time.Date(2026, 10, 10, 13, 30, 0, 0, time.UTC)}}); err != nil {
		t.Fatal(err)
	}

	fs := inquire(t, fx.servitor(), false)

	expectFindings(t, fs, []sought{{"theme", librarium.Impurity, []string{"desecrated", `"off"`, `"on"`}, theme, `["on", "off"]`}})
	if len(fs) != 1 {
		t.Errorf("found %d, expected 1:\n%v", len(fs), fs)
	}
}

// TestInquireII_TheTomesOfSubjects: the tomes a sanctum or transcription
// subject draws its omens from are opened as those of Mark I's verses.
func TestInquireII_TheTomesOfSubjects(t *testing.T) {
	fx := newFixture(t)
	fx.vessel("util.kdl", "")
	hide := fx.write(fx.lib, "tomes/hide.kdl", "hide {{aspect}}\nafter {{inscription.mood}}\n")
	tomes := fx.rite("tomes", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "lost", "designation": "sanctum", "vessel": "$DATA/util.kdl", "omens": {"on": {"tome": "../tomes/missing.kdl"}, "off": ""}},
	  {"name": "lit", "designation": "transcription", "vessel": "$DATA/lit.kdl",
	   "omens": {"*": {"tome": "../tomes/hide.kdl", "illuminate": true}}}]}`)

	fs := inquire(t, fx.servitor(), true)

	expectFindings(t, fs, []sought{
		{"tomes", librarium.Heresy, []string{filepath.Join(fx.lib, "tomes", "missing.kdl"), "nowhere to be found"}, tomes,
			`"../tomes/missing.kdl"`},
		{"tomes", librarium.Heresy, []string{"within the illuminated tome", "mood"}, hide, "{{inscription.mood}}"},
	})
	if len(fs) != 2 {
		t.Errorf("found %d, expected 2:\n%v", len(fs), fs)
	}
}

// TestInquireII_TheRelicsScrollsAndTonguesOfExaltedSubjectsAndObservances:
// the relics of exalted machine-spirits and observances, the words the
// scrolls of observances speak, and the tongues of exalted speech, are
// judged as Mark I judges them of exalted verses, named by subject and
// observance.
func TestInquireII_TheRelicsScrollsAndTonguesOfExaltedSubjectsAndObservances(t *testing.T) {
	fx := newFixture(t)
	bin := t.TempDir()
	fx.write(bin, "python3", "#!/bin/sh\n")
	if err := os.Chmod(filepath.Join(bin, "python3"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	fx.write(fx.lib, "rites/scrolls/tune.conf", "x=1\n")
	tune := fx.write(fx.lib, "rites/scrolls/tune", "#!/bin/sh\ncat $HOME/x\n")
	common := fx.write(fx.lib, "rites/scrolls/common", "#!/bin/sh\nsudo sysctl -w x=1\n")
	outside := fx.write(fx.data, "outside.conf", "x=2\n")
	if err := os.Symlink(outside, filepath.Join(fx.lib, "rites", "scrolls", "astray.conf")); err != nil {
		t.Fatal(err)
	}
	rite := fx.rite("tune", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "smt", "designation": "machine-spirit", "exalted": true, "omens": {"on": "1", "off": "0"},
	   "relics": ["scrolls/tune.conf", "scrolls/lost.conf", "scrolls/astray.conf", "scrolls"],
	   "auspex": {"scan": "cat /sys/x"}, "enact": {"*": "echo {{omen}} > /sys/x"}, "introit": ["recite", "common"]},
	  {"name": "py", "designation": "machine-spirit", "exalted": true, "tongue": "python3", "omens": {"on": "1", "off": "0"},
	   "auspex": {"scan": "cat /sys/y"}, "enact": {"*": "print(1)"}, "recessional": ["pray"]}],
	  "observances": [
	  {"name": "recite", "designation": "litany", "scroll": "scrolls/tune", "exalted": true, "relics": ["gone"]},
	  {"name": "common", "designation": "litany", "scroll": "scrolls/common"},
	  {"name": "pray", "designation": "incantation", "words": "print(2)", "tongue": "python3", "exalted": true}]}`)

	fs := inquire(t, fx.servitor(), true)

	expectFindings(t, fs, []sought{
		{"tune", librarium.Heresy, []string{`relic 2 of the subject "smt", scrolls/lost.conf,`, "nowhere to be found"}, rite,
			`"scrolls/lost.conf"`},
		{"tune", librarium.Heresy, []string{`relic 3 of the subject "smt", scrolls/astray.conf,`, "outside the hall"}, rite,
			`"scrolls/astray.conf"`},
		{"tune", librarium.Heresy, []string{`relic 4 of the subject "smt", scrolls,`, "no scripture"}, rite, `"scrolls"]`},
		{"tune", librarium.Heresy, []string{`relic 1 of the observance "recite", gone,`, "nowhere to be found"}, rite,
			`"gone"`},
		{"tune", librarium.Impurity, []string{"within the scroll", `the exalted speech of the observance "recite" reaches into the home`},
			tune, "cat"},
		{"tune", librarium.Impurity, []string{"within the scroll", `the observance "common" calls upon the intercessor "sudo"`},
			common, "sudo"},
		{"tune", librarium.Heresy, []string{`the tongue python3 of the subject "py" is python3`, librarium.ExaltedTongueNames,
			`the "tongue" of the subject`}, rite, `"python3", "omens"`},
		{"tune", librarium.Heresy, []string{`the tongue python3 of the observance "pray" is python3`,
			`the "tongue" of the observance`}, rite, `"python3", "exalted"`},
	})
	if len(fs) != 8 {
		t.Errorf("found %d, expected 8:\n%v", len(fs), fs)
	}
}

// TestInquireII_AnExaltedTetherSubjectLeadsThroughTheRootsHallsAlone: an
// exalted tether subject whose anchor lies beneath a hall its invoker may
// write is heresy, as an exalted tether of Mark I.
func TestInquireII_AnExaltedTetherSubjectLeadsThroughTheRootsHallsAlone(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("the root may write every hall")
	}
	fx := newFixture(t)
	mine := fx.vessel("dotfiles/x.conf", "x\n")
	rite := fx.rite("tethers", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "x", "designation": "tether", "tether": "/etc/servitor-trials-x.conf", "exalted": true,
	   "omens": {"on": "`+mine+`", "off": "/usr/bin/env"}}]}`)

	fs := inquire(t, fx.servitor(), false, "tethers")

	expectFindings(t, fs, []sought{{"tethers", librarium.Heresy, []string{`the exalted tether subject "x"`,
		"/etc/servitor-trials-x.conf", filepath.Dir(mine), `aspect "on"`, "exalted transcription"}, rite, `"` + mine + `"`}})
	if n := len(denounced(fs, "tethers", librarium.Heresy)); n != 1 {
		t.Errorf("found %d heresies in\n%v", n, fs)
	}
	if got := denounced(inquire(t, fx.servitor(), true, "tethers"), "tethers", librarium.Heresy); len(got) != 0 {
		t.Errorf("--spare-vessels found %v", got)
	}
}

// TestInquireII_TheImpuritiesOfTheScripture: what the scripture of a state
// rite alone reveals reaches the inquisition at its place — an observance
// named by no subject, a machine-spirit without an auspex (more sharply
// when exalted, naming a privileged auspex), omens alike for every aspect,
// and a mandatory inscription some aspect decrees nothing for.
func TestInquireII_TheImpuritiesOfTheScripture(t *testing.T) {
	fx := newFixture(t)
	rite := fx.rite("impure", `{"pattern": "Mark II", "aspects": ["on", "off"],
	  "inscriptions": {"mood": {"mandatory": true, "decrees": {"on": "glad"}}},
	  "subjects": [
	  {"name": "blind", "designation": "machine-spirit", "enact": {"*": "true"}},
	  {"name": "veiled", "designation": "machine-spirit", "exalted": true, "enact": {"*": "true"}},
	  {"name": "same", "designation": "tether", "tether": "$DATA/same", "omens": {"*": "$DATA/one"}}],
	  "observances": [{"name": "forgotten", "designation": "incantation", "words": "true"}]}`)

	fs := inquire(t, fx.servitor(), true)

	expectFindings(t, fs, []sought{
		{"impure", librarium.Impurity, []string{`the inscription "mood" is mandatory`, `"off"`}, rite, `"mood"`},
		{"impure", librarium.Impurity, []string{`the machine-spirit "blind" bears no "auspex"`}, rite, `"blind"`},
		{"impure", librarium.Impurity, []string{`the machine-spirit "veiled" bears no "auspex"`,
			`give it an auspex marked "exalted": true`}, rite, `"veiled"`},
		{"impure", librarium.Impurity, []string{`the omens of the subject "same" are alike for every aspect`}, rite,
			`{"*": "` + fx.data + `/one"}`},
		{"impure", librarium.Impurity, []string{`the observance "forgotten" is named by no subject`}, rite, `"forgotten"`},
	})
	if len(fs) != 5 {
		t.Errorf("found %d, expected 5:\n%v", len(fs), fs)
	}
}

// TestInquireII_TheMachineAnEdictActsUpon: an edict of pattern Mark II is
// examined as one of Mark I — its scroll missing, its tongues, its
// sanction — each found at the key the designation form writes.
func TestInquireII_TheMachineAnEdictActsUpon(t *testing.T) {
	fx := newFixture(t)
	store := sanction.Store{Root: t.TempDir(), UID: 1000}
	capture := fx.rite("capture", `{"pattern": "Mark II", "edict": true, "liturgy": [
	  {"designation": "litany", "tongue": "no-such-tongue-of-edicts", "reversion": "true", "scroll": "$DATA/lost.sh"},
	  {"designation": "incantation", "words": "true", "exalted": true}]}`)
	s := fx.servitor()
	s.Sanctions = &store

	fs := inquire(t, s, false, "capture")

	expectFindings(t, fs, []sought{
		{"capture", librarium.Impurity, []string{filepath.Join(fx.data, "lost.sh"), "scroll", "verse 1"}, capture,
			`"` + fx.data + `/lost.sh"`},
		{"capture", librarium.Impurity, []string{`"no-such-tongue-of-edicts"`, "verse 1"}, capture, `"no-such-tongue-of-edicts"`},
		{"capture", librarium.Impurity, []string{"stands unsanctioned", "servitor sanction capture"}, capture, `true}]}`},
	})
}
