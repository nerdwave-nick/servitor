package sanction

import (
	"bytes"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// gaming is a state rite of pattern Mark II holding every designation
// exalted beside common subjects and observances: two exalted
// machine-spirits (one whose omens splice a decree-only inscription, with
// enact words of its own for one omen and {{aspect}} spliced in them), a
// common tether, an exalted transcription, sanctum and tether, a common
// introit, an exalted recessional guarded "to" "on" and an exalted litany
// guarded "from" "off".
const gaming = `{
  "pattern": "Mark II",
  "purpose": "tune the machine for war",
  "aspects": ["on", "off"],
  "vox-cast": ["progress", "success"],
  "inscriptions": {
    "level": {"mandatory": true, "decrees": {"on": "0", "*": "1"}},
    "reason": {"purpose": "why the rite was invoked"}
  },
  "subjects": [
    {"name": "smt", "designation": "machine-spirit", "exalted": true, "omens": {"on": "off", "*": "on"},
     "auspex": {"scan": "cat /sys/devices/system/cpu/smt/control"},
     "enact": {"*": "echo {{omen}} > /sys/devices/system/cpu/smt/control"},
     "introit": ["warn"], "recessional": ["herald"]},
    {"name": "split-lock", "designation": "machine-spirit", "exalted": true, "omens": {"*": "{{inscription.level}}"},
     "auspex": {"scan": "sysctl -n kernel.split_lock_mitigate"},
     "enact": {"0": "sysctl -wq kernel.split_lock_mitigate=0 # {{aspect}}", "*": "sysctl -wq kernel.split_lock_mitigate={{omen}}"},
     "tongue": "sh", "patience": "5s", "relics": ["scrolls/tune.conf"], "recessional": ["herald", "tune"]},
    {"name": "theme", "designation": "tether", "tether": "/home/x/.theme", "omens": {"*": "/usr/share/themes/{{aspect}}"}},
    {"name": "conf", "designation": "transcription", "exalted": true, "vessel": "/etc/sysctl.d/90-{{rite.name}}.conf",
     "seal": "0640", "omens": {"on": "kernel.split_lock_mitigate = {{inscription.level}}", "off": null}},
    {"name": "lock", "designation": "sanctum", "exalted": true, "vessel": "/etc/split-lock.conf", "consecrate": true,
     "omens": {"*": {"tome": "tomes/split-lock.conf", "illuminate": true}}},
    {"name": "current", "designation": "tether", "exalted": true, "tether": "/etc/split-lock.d/current",
     "omens": {"on": "/usr/share/split-lock/war", "off": null}}
  ],
  "observances": [
    {"name": "warn", "designation": "incantation", "words": "echo common {{former}} {{inscription.reason}}"},
    {"name": "herald", "designation": "incantation", "exalted": true, "to": ["on"], "words": "logger war",
     "reversion": "logger peace"},
    {"name": "tune", "designation": "litany", "exalted": true, "scroll": "scrolls/tune", "offerings": ["{{aspect}}"],
     "from": ["off"], "relics": ["scrolls/tune.conf"]}
  ]
}`

// gamingFiles are the tome, scroll and relic of gaming.
func gamingFiles() map[string]string {
	f := splitLockFiles()
	f["tomes/split-lock.conf"] = "split-lock {{aspect}} {{inscription.level}}\n"
	return f
}

func buildGaming(t *testing.T, scripture string, files map[string]string) *Unit {
	t.Helper()
	u, err := Build(librariumOf(t, "gaming", scripture, files), options)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// golden compares got with the golden file name, written anew under -update.
func golden(t *testing.T, name string, got []byte) {
	t.Helper()
	p := filepath.Join("testdata", name)
	if *update {
		must(t, os.WriteFile(p, got, 0o644))
	}
	want, err := os.ReadFile(p)
	must(t, err)
	if !bytes.Equal(got, want) {
		t.Errorf("drifts from %s:\n%s", p, got)
	}
}

// TestBuildII_TheGoldenUnit: the canonical unit of the state rite gaming,
// as its master beholds it, and its sigils are as recorded.
func TestBuildII_TheGoldenUnit(t *testing.T) {
	u := buildGaming(t, gaming, gamingFiles())
	golden(t, "gaming.json", append(u.Canonical(), '\n'))
	golden(t, "gaming.shown", []byte(strings.Join(Show(u), "\n")+"\n"))
	on, _ := u.SigilOf("on")
	off, _ := u.SigilOf("off")
	if got := []string{u.Sigil(), on, off}; !slices.Equal(got, goldenSigilsII) {
		t.Errorf("sigils %q, want %q", got, goldenSigilsII)
	}
}

// goldenSigilsII are the sigils of the golden unit of gaming: whole, of
// "on", of "off".
var goldenSigilsII = []string{
	"fd0c44679a6b05b250d070331bb2d6e29766cd11e556e4e25beb2d75c15f600d",
	"012cb0d671b4599d0ec4cf51d6102b51b4bd527fbc0ce73d836cf5831b6d51d9",
	"aeb232826223b3a19bf2e0e913043bc74106a61c0642676c96dccdd4a07f5449",
}

// TestBuildII_EveryExaltedItemIsResolvedForEachAspect: each plan holds the
// exalted subjects in written order, each with its place among the rite's
// subjects, and the exalted observances its aspect admits by "to". A
// machine-spirit holds the omen of the plan's aspect and its enact words
// rendered for every omen it declares; an observance the places of the
// subjects that name it, and its "from".
func TestBuildII_EveryExaltedItemIsResolvedForEachAspect(t *testing.T) {
	u := buildGaming(t, gaming, gamingFiles())
	if u.Pattern != PatternMarkII || u.Edict || len(u.Aspects) != 2 {
		t.Fatalf("unit %s", u.Canonical())
	}
	want := []string{"SERVITOR_ASPECT", "SERVITOR_FORMER_ASPECT", "SERVITOR_INSCRIPTION_LEVEL",
		"SERVITOR_INSCRIPTION_REASON", "SERVITOR_OMEN", "SERVITOR_RITE"}
	if u.Environment.Pattern != PatternMarkII || !slices.Equal(u.Environment.Keys, want) {
		t.Errorf("environment %+v", u.Environment)
	}
	on, off := u.Aspects[0], u.Aspects[1]
	names := func(vs []Verse) (out []string) {
		for _, v := range vs {
			out = append(out, v.Named())
		}
		return out
	}
	if got := names(on.Subjects); !slices.Equal(got, []string{`the subject "smt"`, `the subject "split-lock"`,
		`the subject "conf"`, `the subject "lock"`, `the subject "current"`}) || on.Liturgy != nil {
		t.Fatalf("subjects of on %q", got)
	}
	if got, gotOff := names(on.Observances), names(off.Observances); !slices.Equal(got,
		[]string{`the observance "herald"`, `the observance "tune"`}) ||
		!slices.Equal(gotOff, []string{`the observance "tune"`}) {
		t.Fatalf("observances of on %q, of off %q", got, gotOff)
	}
	if on.Inscriptions["level"] != "0" || off.Inscriptions["level"] != "1" || len(on.Inscriptions) != 1 {
		t.Errorf("inscriptions %v / %v", on.Inscriptions, off.Inscriptions)
	}
	smt, lock := on.Subjects[0], on.Subjects[1]
	if !smt.Spirit() || smt.Place != 0 || smt.Omen != "off" || off.Subjects[0].Omen != "on" ||
		smt.Tongue != "/usr/bin/bash" || smt.Patience != 30*time.Second ||
		smt.Enact["off"] != "echo off > /sys/devices/system/cpu/smt/control" ||
		smt.Enact["on"] != "echo on > /sys/devices/system/cpu/smt/control" || len(smt.Enact) != 2 {
		t.Errorf("smt %+v", smt)
	}
	if lock.Place != 1 || lock.Omen != "0" || off.Subjects[1].Omen != "1" || lock.Tongue != "/usr/bin/sh" ||
		lock.Patience != 5*time.Second || lock.Enact["0"] != "sysctl -wq kernel.split_lock_mitigate=0 # on" ||
		lock.Enact["1"] != "sysctl -wq kernel.split_lock_mitigate=1" ||
		off.Subjects[1].Enact["0"] != "sysctl -wq kernel.split_lock_mitigate=0 # off" ||
		len(lock.Relics) != 1 || string(lock.Relics[0].Scripture) != "level=3\n" {
		t.Errorf("split-lock %+v", lock)
	}
	conf, sanctum, tether := on.Subjects[2], on.Subjects[3], on.Subjects[4]
	if conf.Place != 3 || conf.Target != "/etc/sysctl.d/90-gaming.conf" ||
		string(conf.Scripture) != "kernel.split_lock_mitigate = 0" || !off.Subjects[2].Null || *conf.Seal != 0o640 {
		t.Errorf("conf %+v", conf)
	}
	if sanctum.Place != 4 || sanctum.Ward != "gaming" || string(sanctum.Scripture) != "split-lock on 0\n" {
		t.Errorf("lock %+v", sanctum)
	}
	if tether.Place != 5 || tether.Anchor != "/usr/share/split-lock/war" || !off.Subjects[4].Null {
		t.Errorf("current %+v", tether)
	}
	herald, tune := on.Observances[0], on.Observances[1]
	if herald.Target != "logger war" || herald.Reversion != "logger peace" || herald.From != nil ||
		!slices.Equal(herald.Recessional, []int{0, 1}) || len(herald.Introit) != 0 {
		t.Errorf("herald %+v", herald)
	}
	if tune.Target != "scrolls/tune" || !slices.Equal(tune.Offerings, []string{"on"}) ||
		!slices.Equal(tune.From, []string{"off"}) || !slices.Equal(tune.Recessional, []int{1}) {
		t.Errorf("tune %+v", tune)
	}
}

// TestBuildII_TheEnactWordsOfAnExaltedSpiritLapseEveryAspect: a rollback
// enacts the omen read before, which belongs to another aspect, so every
// plan holds every enact word of an exalted machine-spirit: editing the
// words of one omen lapses the plan of every aspect.
func TestBuildII_TheEnactWordsOfAnExaltedSpiritLapseEveryAspect(t *testing.T) {
	base := buildGaming(t, gaming, gamingFiles())
	edited := strings.Replace(gaming, `"0": "sysctl -wq kernel.split_lock_mitigate=0 # {{aspect}}"`,
		`"0": "sysctl -wq kernel.split_lock_mitigate=0"`, 1)
	if edited == gaming {
		t.Fatal("the enact words are not in the rite")
	}
	u := buildGaming(t, edited, gamingFiles())
	for _, aspect := range []string{"on", "off"} {
		if a, b := mustSigil(t, base, aspect), mustSigil(t, u, aspect); a == b {
			t.Errorf("the enact words of the omen of \"on\" changed, yet the sigil of %q stays %s", aspect, a)
		}
	}
}

// TestBuildII_AGuardedObservanceLapsesOnlyTheAspectsItAdmits: an exalted
// observance guarded "to" lies only in the plans of the aspects it admits;
// editing it lapses those alone.
func TestBuildII_AGuardedObservanceLapsesOnlyTheAspectsItAdmits(t *testing.T) {
	base := buildGaming(t, gaming, gamingFiles())
	u := buildGaming(t, strings.Replace(gaming, `"logger war"`, `"logger wars"`, 1), gamingFiles())
	if mustSigil(t, base, "on") == mustSigil(t, u, "on") || mustSigil(t, base, "off") != mustSigil(t, u, "off") {
		t.Errorf("an edit of an observance guarded to \"on\": on %s → %s, off %s → %s", mustSigil(t, base, "on"),
			mustSigil(t, u, "on"), mustSigil(t, base, "off"), mustSigil(t, u, "off"))
	}
}

// TestBuildII_CommonItemsKeepEverySigil: what nothing exalted does — a
// common subject or observance, the vox-cast list, the purpose, the
// unprivileged auspex of an exalted subject — leaves every sigil as it was.
func TestBuildII_CommonItemsKeepEverySigil(t *testing.T) {
	base := buildGaming(t, gaming, gamingFiles())
	for _, c := range []struct{ old, new string }{
		{`"echo common {{former}} {{inscription.reason}}"`, `"echo otherwise"`},
		{`"/usr/share/themes/{{aspect}}"`, `"/usr/share/themes/x"`},
		{`["progress", "success"]`, `["success"]`},
		{`"tune the machine for war"`, `"tune it for peace"`},
		{`"cat /sys/devices/system/cpu/smt/control"`, `"cat /sys/devices/system/cpu/smt/active"`},
		{`"why the rite was invoked"`, `"why"`},
	} {
		edited := strings.Replace(gaming, c.old, c.new, 1)
		if edited == gaming {
			t.Fatalf("%s is not in the rite", c.old)
		}
		sameSigils(t, c.new, base, buildGaming(t, edited, gamingFiles()))
	}
}

// TestBuildII_APlanMayHoldNothing: an aspect no exalted item serves — here
// no exalted observance is admitted into "off" — has a plan that holds
// nothing, under a sigil of its own.
func TestBuildII_APlanMayHoldNothing(t *testing.T) {
	u := buildGaming(t, `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "m", "designation": "machine-spirit", "enact": {"*": "true"}, "recessional": ["herald"]}],
	  "observances": [{"name": "herald", "designation": "incantation", "exalted": true, "to": ["on"],
	    "words": "logger war"}]}`, nil)
	on, off := u.Aspects[0], u.Aspects[1]
	if len(on.Observances) != 1 || len(off.Observances) != 0 || len(off.Subjects) != 0 {
		t.Fatalf("plans %s", u.Canonical())
	}
	if !strings.Contains(string(u.Canonical()), `{"aspect":"off","inscriptions":{},"observances":[],"subjects":[]}`) {
		t.Errorf("the empty plan is written %s", u.Canonical())
	}
}

// TestBuildII_AnEdictOfMarkIIIsOfItsPattern: an edict of pattern Mark II is
// drawn as an edict of Mark I is, under the pattern Mark II, so that its
// conversion lapses its sanction.
func TestBuildII_AnEdictOfMarkIIIsOfItsPattern(t *testing.T) {
	markI := librariumOf(t, "reboot", `{"pattern": "Mark I", "edict": true,
	  "liturgy": [{"incantation": "systemctl reboot", "exalted": true}]}`, nil)
	markII := librariumOf(t, "reboot", `{"pattern": "Mark II", "edict": true,
	  "liturgy": [{"designation": "incantation", "words": "systemctl reboot", "exalted": true}]}`, nil)
	a, err := Build(markI, options)
	must(t, err)
	b, err := Build(markII, options)
	must(t, err)
	if a.Pattern != Pattern || b.Pattern != PatternMarkII || b.Environment.Pattern != PatternMarkII ||
		len(b.Aspects[0].Liturgy) != 1 || b.Aspects[0].Liturgy[0].Verse != 1 || a.Sigil() == b.Sigil() {
		t.Errorf("Mark I %s\nMark II %s", a.Canonical(), b.Canonical())
	}
	if slices.Contains(b.Environment.Keys, "SERVITOR_OMEN") {
		t.Errorf("an edict hears an omen: %v", b.Environment.Keys)
	}
}

// TestBuildII_ASanctionIsReadBackWhole: the sanction of a state rite is
// written and read back strictly, its sigil unchanged.
func TestBuildII_ASanctionIsReadBackWhole(t *testing.T) {
	u := buildGaming(t, gaming, gamingFiles())
	store := Store{Root: t.TempDir(), UID: 1000}
	data, err := json.Marshal(New(u, time.Date(2026, 10, 13, 18, 0, 0, 0, time.UTC)))
	must(t, err)
	must(t, os.MkdirAll(filepath.Dir(store.Path("gaming")), 0o750))
	must(t, os.WriteFile(store.Path("gaming"), data, 0o640))
	kept, err := store.Read("gaming")
	if err != nil || kept == nil || kept.Sigil != u.Sigil() || kept.Unit.Sigil() != u.Sigil() {
		t.Fatalf("read back %v: %v", kept, err)
	}
	if !bytes.Equal(kept.Canonical(), u.Canonical()) {
		t.Errorf("read back\n%s\nwant\n%s", kept.Canonical(), u.Canonical())
	}
}

// TestBuildII_AMisshapenSanctionIsNoTrueOne: a sanction whose plans hold
// what its pattern does not — subjects in a unit of Mark I, verses in the
// unit of a state rite — is no true sanction, whatever its sigil.
func TestBuildII_AMisshapenSanctionIsNoTrueOne(t *testing.T) {
	markI := build(t, splitLock, splitLockFiles())
	markI.Aspects[0].Subjects = buildGaming(t, gaming, gamingFiles()).Aspects[0].Subjects
	state := buildGaming(t, gaming, gamingFiles())
	state.Aspects[1].Liturgy = build(t, splitLock, splitLockFiles()).Aspects[1].Liturgy
	for _, u := range []*Unit{markI, state} {
		store := Store{Root: t.TempDir(), UID: 1000}
		data, err := json.Marshal(New(u, time.Date(2026, 10, 13, 18, 0, 0, 0, time.UTC)))
		must(t, err)
		must(t, os.MkdirAll(filepath.Dir(store.Path(u.Rite)), 0o750))
		must(t, os.WriteFile(store.Path(u.Rite), data, 0o640))
		if kept, err := store.Read(u.Rite); err == nil || kept != nil || !strings.Contains(err.Error(), "no true sanction") {
			t.Errorf("a misshapen sanction of %s was read: %v %v", u.Pattern, kept, err)
		}
	}
}
