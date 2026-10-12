package sanction

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

var update = flag.Bool("update", false, "write the golden units anew")

// splitLock holds every kind of exalted verse, common verses between them,
// aspect maps, a decree-only inscription spliced into exalted words and an
// inscription whose word comes only from its rune.
const splitLock = `{
  "pattern": "Mark I",
  "purpose": "tune the machine for war",
  "aspects": ["on", "off"],
  "auspex": "echo off",
  "inscriptions": {
    "mode": {"mandatory": true, "decrees": {"on": "war", "off": "peace"}},
    "reason": {"purpose": "why the rite was invoked"}
  },
  "liturgy": [
    {"vox-cast": "progress"},
    {"sanctum": "/etc/split-lock.conf", "exalted": true, "consecrate": true,
     "scripture": {"tome": "tomes/split-lock.conf", "illuminate": true}},
    {"incantation": "echo common {{former}} {{inscription.reason}}"},
    {"incantation": {"on": "sysctl -wq kernel.split_lock_mitigate=0", "*": "sysctl -wq kernel.split_lock_mitigate=1"},
     "exalted": true, "reversion": {"on": "sysctl -wq kernel.split_lock_mitigate=1", "off": ""},
     "relics": ["scrolls/tune.conf"]},
    {"litany": "scrolls/tune", "offerings": ["{{aspect}}", "{{rite.name}}"], "exalted": true, "tongue": "sh",
     "patience": "5s", "relics": ["scrolls/tune.conf"]},
    {"transcription": "/etc/sysctl.d/90-{{rite.name}}.conf", "exalted": true, "seal": "0640", "zeal": true,
     "scripture": {"on": "kernel.split_lock_mitigate = 0 # {{inscription.mode}}", "off": null}},
    {"tether": "/etc/split-lock.d/current", "exalted": true,
     "anchor": {"on": "/usr/share/split-lock/war", "off": null}},
    {"vox-cast": "success"}
  ]
}`

// librariumOf writes a Librarium holding the rite name with scripture, and
// the files beside it (relative to rites/), and loads it.
func librariumOf(t *testing.T, name, scripture string, files map[string]string) *librarium.Rite {
	t.Helper()
	dir := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(dir, "rites", rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(name+".json", scripture)
	for rel, content := range files {
		write(rel, content)
	}
	lib := librarium.Load(dir)
	r := lib.Rites[name]
	if r == nil {
		t.Fatalf("the rite %s is heretical: %v", name, lib.Findings)
	}
	return r
}

// splitLockFiles are the tome, scroll and relic of splitLock.
func splitLockFiles() map[string]string {
	return map[string]string{
		"tomes/split-lock.conf": "split-lock {{aspect}} {{inscription.mode}}\n",
		"scrolls/tune":          "#!/bin/sh\n. ./scrolls/tune.conf\n",
		"scrolls/tune.conf":     "level=3\n",
	}
}

// options draw a unit for uid 1000, finding every tongue in /usr/bin.
var options = Options{
	UID:      1000,
	Orders:   librarium.Orders{Tongue: "bash", Patience: 30 * time.Second},
	LookPath: func(name string) (string, error) { return "/usr/bin/" + name, nil },
}

func build(t *testing.T, scripture string, files map[string]string) *Unit {
	t.Helper()
	u, err := Build(librariumOf(t, "split-lock", scripture, files), options)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// TestBuild_TheGoldenUnit: the canonical unit of splitLock and its sigils
// are as recorded.
func TestBuild_TheGoldenUnit(t *testing.T) {
	u := build(t, splitLock, splitLockFiles())
	golden := filepath.Join("testdata", "split-lock.json")
	if *update {
		if err := os.WriteFile(golden, append(u.Canonical(), '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got := u.Canonical(); !bytes.Equal(got, bytes.TrimSuffix(want, []byte("\n"))) {
		t.Errorf("the unit drifts from %s:\n%s", golden, got)
	}
	on, _ := u.SigilOf("on")
	off, _ := u.SigilOf("off")
	sigils := []string{u.Sigil(), on, off}
	for i, want := range goldenSigils {
		if sigils[i] != want {
			t.Errorf("sigil %d is %s, want %s", i, sigils[i], want)
		}
	}
	if on == off || on == u.Sigil() || Short(on) != on[:12] || len(on) != 64 {
		t.Errorf("sigils %v", sigils)
	}
}

// goldenSigils are the sigils of the golden unit: whole, of "on", of "off".
var goldenSigils = []string{
	"d66ec48808fddd8016d0f9c324c36f39e62f351f4b4c51498799baf80994f6cb",
	"e55534a5ffa5537e5fa439420bd8370c76c6df60b5e35357664f0ba0f4b38cf4",
	"1ce8e46e717805c1bb2a01e8c2766eb314fa635c3b4bbd9ee0f92c11c72c09e0",
}

// TestBuild_AspectMapsAndDecreesResolvePerAspect: each aspect's plan holds
// the exalted verses alone, resolved for it, with the decree of every
// inscription spliced into them.
func TestBuild_AspectMapsAndDecreesResolvePerAspect(t *testing.T) {
	u := build(t, splitLock, splitLockFiles())
	if u.Pattern != Pattern || u.UID != 1000 || u.Rite != "split-lock" || u.Edict || len(u.Aspects) != 2 {
		t.Fatalf("unit %+v", u)
	}
	want := []string{"SERVITOR_ASPECT", "SERVITOR_FORMER_ASPECT", "SERVITOR_INSCRIPTION_MODE",
		"SERVITOR_INSCRIPTION_REASON", "SERVITOR_RITE"}
	if u.Environment.Pattern != EnvironmentPattern || strings.Join(u.Environment.Keys, " ") != strings.Join(want, " ") {
		t.Errorf("environment %+v", u.Environment)
	}
	on, off := u.Aspects[0], u.Aspects[1]
	if on.Aspect != "on" || off.Aspect != "off" || len(on.Inscriptions) != 1 || on.Inscriptions["mode"] != "war" ||
		off.Inscriptions["mode"] != "peace" {
		t.Fatalf("plans %+v %+v", on.Inscriptions, off.Inscriptions)
	}
	var verses []int
	for _, v := range on.Liturgy {
		verses = append(verses, v.Verse)
	}
	if len(verses) != 5 || verses[0] != 2 || verses[4] != 7 {
		t.Fatalf("verses %v", verses)
	}
	sanctum, incantation, litany, transcription, tether := on.Liturgy[0], on.Liturgy[1], on.Liturgy[2], on.Liturgy[3], on.Liturgy[4]
	if string(sanctum.Scripture) != "split-lock on war\n" || sanctum.Ward != "split-lock" || sanctum.Glyph != "#" ||
		!sanctum.Consecrate || string(off.Liturgy[0].Scripture) != "split-lock off peace\n" {
		t.Errorf("sanctum %+v", sanctum)
	}
	if incantation.Target != "sysctl -wq kernel.split_lock_mitigate=0" || incantation.Tongue != "/usr/bin/bash" ||
		incantation.Patience != 30*time.Second || incantation.Reversion != "sysctl -wq kernel.split_lock_mitigate=1" ||
		off.Liturgy[1].Target != "sysctl -wq kernel.split_lock_mitigate=1" || off.Liturgy[1].Reversion != "" ||
		len(incantation.Relics) != 1 || string(incantation.Relics[0].Scripture) != "level=3\n" {
		t.Errorf("incantation %+v / %+v", incantation, off.Liturgy[1])
	}
	if litany.Target != "scrolls/tune" || string(litany.Scroll) != "#!/bin/sh\n. ./scrolls/tune.conf\n" ||
		strings.Join(litany.Offerings, " ") != "on split-lock" || litany.Tongue != "/usr/bin/sh" ||
		litany.Patience != 5*time.Second {
		t.Errorf("litany %+v", litany)
	}
	if transcription.Target != "/etc/sysctl.d/90-split-lock.conf" || transcription.Null ||
		string(transcription.Scripture) != "kernel.split_lock_mitigate = 0 # war" || *transcription.Seal != 0o640 ||
		!transcription.Zeal || !off.Liturgy[3].Null {
		t.Errorf("transcription %+v / %+v", transcription, off.Liturgy[3])
	}
	if tether.Target != "/etc/split-lock.d/current" || tether.Anchor != "/usr/share/split-lock/war" || tether.Null ||
		!off.Liturgy[4].Null {
		t.Errorf("tether %+v / %+v", tether, off.Liturgy[4])
	}
}

// TestBuild_CommonVersesKeepEverySigil: what no exalted verse does — vox-
// casts, common incantations, purpose, auspex — leaves every sigil as it
// was.
func TestBuild_CommonVersesKeepEverySigil(t *testing.T) {
	base := build(t, splitLock, splitLockFiles())
	for _, c := range []struct{ old, new string }{
		{`{"vox-cast": "progress"}`, `{"vox-cast": "success"}`},
		{`"echo common {{former}} {{inscription.reason}}"`, `"echo otherwise"`},
		{`"tune the machine for war"`, `"tune it for peace"`},
		{`"auspex": "echo off"`, `"auspex": "echo on"`},
		{`"why the rite was invoked"`, `"why"`},
	} {
		edited := strings.Replace(splitLock, c.old, c.new, 1)
		if edited == splitLock {
			t.Fatalf("%s is not in the rite", c.old)
		}
		sameSigils(t, c.new, base, build(t, edited, splitLockFiles()))
	}
}

// TestBuild_ExaltedEditsLapseTheSigil: an exalted command, a tome, a relic
// or a scroll changed changes the sigil; an edit for one aspect leaves the
// sigil of the other.
func TestBuild_ExaltedEditsLapseTheSigil(t *testing.T) {
	base := build(t, splitLock, splitLockFiles())
	baseOff, _ := base.SigilOf("off")
	edits := map[string]func() (string, map[string]string){
		"an exalted command": func() (string, map[string]string) {
			return strings.Replace(splitLock, "split_lock_mitigate=0\"", "split_lock_mitigate=2\"", 1), splitLockFiles()
		},
		"a tome": func() (string, map[string]string) {
			f := splitLockFiles()
			f["tomes/split-lock.conf"] += "# more\n"
			return splitLock, f
		},
		"a relic": func() (string, map[string]string) {
			f := splitLockFiles()
			f["scrolls/tune.conf"] = "level=4\n"
			return splitLock, f
		},
		"a scroll": func() (string, map[string]string) {
			f := splitLockFiles()
			f["scrolls/tune"] += "echo more\n"
			return splitLock, f
		},
	}
	for what, edit := range edits {
		scripture, files := edit()
		u := build(t, scripture, files)
		if u.Sigil() == base.Sigil() {
			t.Errorf("%s changed, yet the sigil stays %s", what, u.Sigil())
		}
	}
	scripture, files := edits["an exalted command"]()
	u := build(t, scripture, files)
	on, _ := u.SigilOf("on")
	baseOn, _ := base.SigilOf("on")
	if off, _ := u.SigilOf("off"); off != baseOff || on == baseOn {
		t.Errorf("an edit for \"on\" alone: on %s (was %s), off %s (was %s)", on, baseOn, off, baseOff)
	}
}

func sameSigils(t *testing.T, what string, a, b *Unit) {
	t.Helper()
	if a.Sigil() != b.Sigil() {
		t.Errorf("%s changed the sigil:\n%s\n%s", what, a.Canonical(), b.Canonical())
	}
	for _, aspect := range []string{"on", "off"} {
		x, _ := a.SigilOf(aspect)
		y, _ := b.SigilOf(aspect)
		if x != y {
			t.Errorf("%s changed the sigil of %q", what, aspect)
		}
	}
}

// TestBuild_AnEdictHasOnePlan: an edict is drawn once, without aspect, and
// hears neither aspect nor former aspect.
func TestBuild_AnEdictHasOnePlan(t *testing.T) {
	r := librariumOf(t, "reboot", `{"pattern": "Mark I", "edict": true,
	  "inscriptions": {"when": {"decrees": "now"}},
	  "liturgy": [{"vox-cast": "progress"}, {"incantation": "systemctl reboot --when={{inscription.when}}", "exalted": true}]}`, nil)
	u, err := Build(r, options)
	if err != nil {
		t.Fatal(err)
	}
	if !u.Edict || len(u.Aspects) != 1 || u.Aspects[0].Aspect != "" || u.Aspects[0].Inscriptions["when"] != "now" ||
		len(u.Aspects[0].Liturgy) != 1 || u.Aspects[0].Liturgy[0].Target != "systemctl reboot --when=now" ||
		strings.Join(u.Environment.Keys, " ") != "SERVITOR_INSCRIPTION_WHEN SERVITOR_RITE" {
		t.Fatalf("unit %s", u.Canonical())
	}
	if sigil, ok := u.SigilOf(""); !ok || sigil != u.Sigil() {
		t.Errorf("the sigil of the one plan %s is not the sigil of the edict %s", sigil, u.Sigil())
	}
}

// TestBuild_NothingToSanction: a rite without exalted verses is no unit; one
// whose tome cannot be read is none either.
func TestBuild_NothingToSanction(t *testing.T) {
	r := librariumOf(t, "plain", `{"pattern": "Mark I", "aspects": ["on"], "liturgy": [
	  {"incantation": "echo", "exalted": false}]}`, nil)
	if u, err := Build(r, options); !errors.Is(err, ErrUnexalted) || u != nil {
		t.Errorf("Build = %v, %v; want ErrUnexalted", u, err)
	}
	files := splitLockFiles()
	delete(files, "tomes/split-lock.conf")
	if u, err := Build(librariumOf(t, "split-lock", splitLock, files), options); err == nil ||
		!strings.Contains(err.Error(), "split-lock.conf") || u != nil {
		t.Errorf("Build = %v, %v; want the lament of the tome", u, err)
	}
}
