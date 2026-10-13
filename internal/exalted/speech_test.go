package exalted

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/nerdwave-nick/servitor/internal/hallowed"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// tongue is the bash of the machine, by the path the root keeps it at.
func tongue(t *testing.T) string {
	t.Helper()
	bash, err := exec.LookPath("bash")
	if err == nil {
		bash, err = filepath.EvalSymlinks(bash)
	}
	if err != nil {
		t.Skip("no bash upon this machine")
	}
	if _, err := hallowed.Machine.Program(bash); err != nil {
		t.Skipf("the bash of this machine is no program of the root: %v", err)
	}
	return bash
}

// incantation is an exalted incantation of the trials, spoken in bash.
func incantation(t *testing.T, n int, words string) sanction.Verse {
	return sanction.Verse{Verse: n, Kind: librarium.KindIncantation, Target: words, Tongue: tongue(t),
		Patience: patience, Relics: []sanction.Relic{}}
}

// speakingUnit is the unit of hallow, its plans holding exalted observances
// alone: the speech written as verse n becomes the observance "v<n>" (see
// spoken), the introit of the subject at place n, so that they are spoken
// in the order of their numbers.
func speakingUnit(verses ...sanction.Verse) *sanction.Unit {
	u := hallowUnit()
	for i := range u.Aspects {
		p := &u.Aspects[i]
		p.Subjects, p.Observances = nil, make([]sanction.Verse, len(verses))
		for j, v := range verses {
			v.Observance, v.Introit, v.Recessional = "v"+strconv.Itoa(v.Verse), []int{v.Verse}, []int{}
			v.Verse = 0
			p.Observances[j] = v
		}
	}
	return u
}

// spoken names the observance speakingUnit makes of the speech of verse n.
func spoken(n int) *Item { return observance("v" + strconv.Itoa(n)) }

// speakingShrine is the world of a trial of exalted speech for u: its
// working halls lie beneath a temporary hall, its tongues are judged upon
// the machine itself, and it reads a locale of its invoker's choosing.
func speakingShrine(t *testing.T, u *sanction.Unit) *shrine {
	sh := newShrine(t, u)
	sh.cfg.Voice = sh.speech().Voice
	return sh
}

func (sh *shrine) speech() Speech {
	return Speech{
		Halls:     filepath.Join(sh.ground, "run", "servitor"),
		Sovereign: func(uid uint32) bool { return uid == uint32(me) },
		Tongues:   hallowed.Machine,
		Getenv: env(map[string]string{"LANG": "C.UTF-8", "LC_TIME": "en_DK.UTF-8", "LC_ALL": "../../xenos",
			"LANGUAGE": "de", "BASH_ENV": "/tmp/xenos", "LC_CTYPE": "en_US.UTF-8\nBASH_ENV=/tmp/xenos"}),
	}
}

// hall is the working hall of the session of aspect.
func (sh *shrine) hall(aspect string) string {
	return filepath.Join(sh.ground, "run", "servitor", sh.sigil(aspect))
}

// TestSpeech_HearsItsEnvironmentAlone: exalted speech hears the variables
// the exalted servitor grants and nothing of the environment it was roused
// in; an edict hears neither aspect nor former aspect.
func TestSpeech_HearsItsEnvironmentAlone(t *testing.T) {
	xenos := filepath.Join(t.TempDir(), "xenos")
	must(t, os.WriteFile(xenos, []byte("echo INJECTED\n"), 0o644))
	for k, v := range map[string]string{"BASH_ENV": xenos, "ENV": xenos, "SHELLOPTS": "xtrace", "BASHOPTS": "extglob",
		"IFS": "x", "PS4": "INJECTED", "CDPATH": "/tmp", "BASH_FUNC_echo%%": "() { builtin echo INJECTED; }",
		"LD_LIBRARY_PATH": "/nowhere", "SERVITOR_INSCRIPTION_STALE": "stale", "HOME": "/home/xenos"} {
		t.Setenv(k, v)
	}
	// bash's own SHLVL and _ aside
	const words = "env | grep -v -e '^SHLVL=' -e '^_=' | sort"
	common := []string{"HOME=/root", "LANG=C.UTF-8", "LC_TIME=en_DK.UTF-8", "PATH=/usr/local/sbin:/usr/local/bin:/usr/bin"}

	sh := speakingShrine(t, speakingUnit(incantation(t, 1, words)))
	inv := sh.accepted("on")
	want := append(slices.Clone(common), "PWD="+sh.hall("on"), "SERVITOR_ASPECT=on", "SERVITOR_FORMER_ASPECT=off",
		"SERVITOR_INSCRIPTION_WHO=x; echo INJECTED", "SERVITOR_RITE=hallow")
	slices.Sort(want)
	if got := strings.Split(strings.TrimSpace(inv.perform(spoken(1)).Heard), "\n"); !slices.Equal(got, want) {
		t.Errorf("exalted speech heard\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	inv.say(Request{Finish: new(true)})

	u := hallowUnit()
	u.Edict, u.Aspects = true, []sanction.Plan{{Inscriptions: map[string]string{},
		Liturgy: []sanction.Verse{incantation(t, 1, words)}}}
	u.Environment.Keys = []string{"SERVITOR_INSCRIPTION_WHO", "SERVITOR_RITE"}
	sh = speakingShrine(t, u)
	inv = sh.rouse("hallow", sanction.Short(sh.sigil("")))
	if r := inv.say(Handshake{Protocol: Protocol, UID: me, Rite: "hallow", Sigil: sh.sigil(""),
		Inscriptions: map[string]string{"who": "edict"}}); r.Verdict != Accepted {
		t.Fatalf("the edict was not accepted: %+v", r)
	}
	want = append(slices.Clone(common), "PWD="+sh.hall(""), "SERVITOR_INSCRIPTION_WHO=edict", "SERVITOR_RITE=hallow")
	slices.Sort(want)
	if got := strings.Split(strings.TrimSpace(inv.perform(numbered(1)).Heard), "\n"); !slices.Equal(got, want) {
		t.Errorf("the edict heard\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestSpeech_TheInjectionIsHeardAsAValue: the inscription of the research
// (--who 'x; echo INJECTED') reaches exalted speech as the value of
// SERVITOR_INSCRIPTION_WHO, and its words are never spoken.
func TestSpeech_TheInjectionIsHeardAsAValue(t *testing.T) {
	sh := speakingShrine(t, speakingUnit(incantation(t, 1, `echo "who=$SERVITOR_INSCRIPTION_WHO"`)))
	inv := sh.accepted("on")
	if r := inv.perform(spoken(1)); r.Heard != "who=x; echo INJECTED\n" {
		t.Errorf("exalted speech uttered %q", r.Heard)
	}
}

// TestSpeech_LaysTheRelicsInAHallOfItsOwn: exalted speech is spoken in a
// fresh hall of the root's, sealed 0700, holding its relics and the scrolls
// of its litanies under their own names; the hall is gone once the session
// ends. A litany is recited through its tongue, or by the interpreter its
// shebang names.
func TestSpeech_LaysTheRelicsInAHallOfItsOwn(t *testing.T) {
	bash := tongue(t)
	relics := []sanction.Relic{{Relic: "tuning.list", Scripture: []byte("split-lock\n")},
		{Relic: "lib/governor.conf", Scripture: []byte("performance\n")}}
	speaking := incantation(t, 1, `pwd; stat -c %a .; cat ./tuning.list ./lib/governor.conf`)
	speaking.Relics = relics
	through := sanction.Verse{Verse: 2, Kind: librarium.KindLitany, Target: "scrolls/governor",
		Scroll: []byte(`echo "recited $0 $*"; cat ./tuning.list`), Offerings: []string{"on", "a b"}, Tongue: bash,
		Patience: patience, Relics: relics[:1]}
	shebang := sanction.Verse{Verse: 3, Kind: librarium.KindLitany, Target: "/home/faithful/bin/dusk",
		Scroll:    []byte("#!" + bash + " -e\necho \"by its shebang $0 $1\"; case $- in *e*) echo unbending; esac\n"),
		Offerings: []string{"dusk"}, Tongue: bash, Patience: patience, Relics: []sanction.Relic{}}
	sh := speakingShrine(t, speakingUnit(speaking, through, shebang))
	inv := sh.accepted("on")
	hall := sh.hall("on")
	if got, want := inv.perform(spoken(1)).Heard, hall+"\n700\nsplit-lock\nperformance\n"; got != want {
		t.Errorf("verse 1 uttered %q, want %q", got, want)
	}
	if got, want := inv.perform(spoken(2)).Heard, "recited "+hall+"/scrolls/governor on a b\nsplit-lock\n"; got != want {
		t.Errorf("verse 2 uttered %q, want %q", got, want)
	}
	if got, want := inv.perform(spoken(3)).Heard, "by its shebang "+hall+"/dusk dusk\nunbending\n"; got != want {
		t.Errorf("verse 3 uttered %q, want %q", got, want)
	}
	inv.say(Request{Finish: new(true)})
	if m := inv.ended(); m != 0 {
		t.Fatalf("ended with the death-mark %d", m)
	}
	if _, err := os.Lstat(hall); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the working hall outlived its session: %v", err)
	}
}

// TestSpeech_TheHallIsSealedWhileItSpeaks: while the session lasts, its
// hall stands sealed 0700, holding the relics alone.
func TestSpeech_TheHallIsSealedWhileItSpeaks(t *testing.T) {
	v := incantation(t, 1, "true")
	v.Relics = []sanction.Relic{{Relic: "tuning.list", Scripture: []byte("split-lock\n")}}
	sh := speakingShrine(t, speakingUnit(v))
	inv := sh.accepted("on")
	fi, err := os.Lstat(sh.hall("on"))
	must(t, err)
	if !fi.IsDir() || fi.Mode().Perm() != 0o700 {
		t.Errorf("the working hall is %v", fi.Mode())
	}
	entries, err := os.ReadDir(sh.hall("on"))
	must(t, err)
	if len(entries) != 1 || entries[0].Name() != "tuning.list" {
		t.Errorf("the working hall holds %v", entries)
	}
	must(t, inv.conn.Close())
	if m := inv.ended(); m != 1 {
		t.Fatalf("ended with the death-mark %d", m)
	}
	if _, err := os.Lstat(sh.hall("on")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the working hall outlived its session: %v", err)
	}
}

// alive reports whether the process pid still lives: it is neither gone
// nor a corpse awaiting its reaper.
func alive(pid int) bool {
	stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return false
	}
	s := string(stat)
	f := strings.Fields(s[strings.LastIndexByte(s, ')')+1:])
	return len(f) > 0 && f[0] != "Z" && f[0] != "X"
}

// slain awaits the end of the process pid.
func slain(t *testing.T, pid int) {
	t.Helper()
	for deadline := time.Now().Add(patience); alive(pid); time.Sleep(10 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("the process %d lives on", pid)
		}
	}
}

// spare slays the process pid, should a trial that fell have left it.
func spare(pid int) {
	if alive(pid) {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}
}

func pidOf(t *testing.T, heard string) int {
	t.Helper()
	pid, err := strconv.Atoi(strings.TrimSpace(heard))
	if err != nil {
		t.Fatalf("no process was told: %q", heard)
	}
	return pid
}

// TestSpeech_SlaysWhatItLeavesBehind: a child that exalted speech sends
// into the background outlives its verse, but not the session — not even
// one led into a process group of its own.
func TestSpeech_SlaysWhatItLeavesBehind(t *testing.T) {
	sh := speakingShrine(t, speakingUnit(incantation(t, 1, "sleep 300 >/dev/null 2>&1 & echo $!"),
		incantation(t, 2, "set -m; sleep 300 >/dev/null 2>&1 & echo $!")))
	inv := sh.accepted("on")
	lingering, apart := pidOf(t, inv.perform(spoken(1)).Heard), pidOf(t, inv.perform(spoken(2)).Heard)
	t.Cleanup(func() { spare(lingering); spare(apart) })
	if !alive(lingering) || !alive(apart) {
		t.Fatal("the children of exalted speech did not outlive their verses")
	}
	inv.say(Request{Finish: new(true)})
	if m := inv.ended(); m != 0 {
		t.Fatalf("ended with the death-mark %d", m)
	}
	slain(t, lingering)
	slain(t, apart)
}

// TestSpeech_PatienceSlaysTheProcessGroup: when the patience of exalted
// speech runs out, the verse falls, and every process it summoned is slain
// with it.
func TestSpeech_PatienceSlaysTheProcessGroup(t *testing.T) {
	v := incantation(t, 1, "sleep 300 >/dev/null 2>&1 & echo $!; wait")
	v.Patience = 300 * time.Millisecond
	sh := speakingShrine(t, speakingUnit(v))
	inv := sh.accepted("on")
	r := inv.say(Request{Perform: spoken(1)})
	if r.Verdict != Fell || !strings.Contains(r.Lament, "patience") {
		t.Fatalf("verse 1: %+v", r)
	}
	pid := pidOf(t, r.Heard)
	t.Cleanup(func() { spare(pid) })
	slain(t, pid)
}

// TestSpeech_TellsItsLastWordsAlone: of what exalted speech utters, its
// last 64 KiB are told, as words the channel may carry.
func TestSpeech_TellsItsLastWordsAlone(t *testing.T) {
	sh := speakingShrine(t, speakingUnit(
		incantation(t, 1, `head -c 100000 /dev/zero | tr '\0' a; echo; echo END`),
		incantation(t, 2, `printf 'a\xff\xfeb\n'`)))
	inv := sh.accepted("on")
	if got := inv.perform(spoken(1)).Heard; len(got) != 64<<10 || !strings.HasSuffix(got, "a\nEND\n") {
		t.Errorf("verse 1 told %d bytes, ending %q", len(got), got[max(0, len(got)-8):])
	}
	if got := inv.perform(spoken(2)).Heard; !utf8.ValidString(got) || !strings.HasPrefix(got, "a") ||
		!strings.HasSuffix(got, "b\n") {
		t.Errorf("verse 2 told %q", got)
	}
}

// TestSpeech_RevertsInTheExaltedServitor: the reversion of exalted speech
// is spoken by the exalted servitor, in the same hall and environment,
// whether the verse is reverted at the invoker's word, fell, or the channel
// ended; its words are told.
func TestSpeech_RevertsInTheExaltedServitor(t *testing.T) {
	v := incantation(t, 1, "echo performed")
	v.Reversion = `echo "reverted $SERVITOR_ASPECT to $SERVITOR_FORMER_ASPECT"; cat ./tuning.list`
	v.Relics = []sanction.Relic{{Relic: "tuning.list", Scripture: []byte("split-lock\n")}}
	falls := incantation(t, 2, "echo falling; exit 3")
	falls.Reversion = "echo undone"
	sh := speakingShrine(t, speakingUnit(v, falls))
	inv := sh.accepted("on")
	if r := inv.perform(spoken(1)); !r.Changed {
		t.Errorf("speech with a reversion stands unchanged: %+v", r)
	}
	r := inv.say(Request{Revert: spoken(1)})
	if r.Verdict != Reverted || r.Heard != "reverted on to off\nsplit-lock\n" {
		t.Errorf("the reversion: %+v", r)
	}

	sh = speakingShrine(t, speakingUnit(v, falls))
	inv = sh.accepted("on")
	inv.perform(spoken(1))
	r = inv.say(Request{Perform: spoken(2)})
	if r.Verdict != Fell || !r.Changed || r.Heard != "falling\n" || !strings.Contains(r.Lament, "death-mark 3") {
		t.Fatalf("verse 2: %+v", r)
	}
	must(t, inv.conn.Close())
	if m := inv.ended(); m != 1 {
		t.Fatalf("ended with the death-mark %d", m)
	}
	want := []string{"accept accepted", "perform v1 triumph", "perform v2 fell", "revert v2 reverted", "revert v1 reverted",
		"fall reverted"}
	if got := sh.journal.deeds(); !slices.Equal(got, want) {
		t.Errorf("journaled %v, want %v", got, want)
	}
}

// TestSpeech_RefusesWhatItCannotSpeakSafely: a session whose speech names a
// tongue that is no program of the root, a relic beyond its hall, two
// relics of one name, or a variable the exalted servitor never grants, or
// whose halls are no root's, is refused before anything is touched, and no
// working hall is left.
func TestSpeech_RefusesWhatItCannotSpeakSafely(t *testing.T) {
	open := t.TempDir()
	must(t, os.Chmod(open, 0o777))
	must(t, os.WriteFile(filepath.Join(open, "bash"), []byte("#!/bin/sh\n"), 0o755))
	for _, c := range []struct {
		name   string
		mend   func(sh *shrine, v *sanction.Verse)
		lament string
	}{
		{"a tongue spoken from no root", func(_ *shrine, v *sanction.Verse) { v.Tongue = "bash" }, "begin at the root"},
		{"a tongue in an open hall", func(_ *shrine, v *sanction.Verse) { v.Tongue = filepath.Join(open, "bash") },
			"others than its owner may write"},
		{"a relic beyond its hall", func(_ *shrine, v *sanction.Verse) {
			v.Relics = []sanction.Relic{{Relic: "../xenos", Scripture: []byte("x")}}
		}, "../xenos"},
		{"a relic from the root", func(_ *shrine, v *sanction.Verse) {
			v.Relics = []sanction.Relic{{Relic: "/etc/xenos", Scripture: []byte("x")}}
		}, "/etc/xenos"},
		{"two relics of one name", func(_ *shrine, v *sanction.Verse) {
			v.Relics = []sanction.Relic{{Relic: "a", Scripture: []byte("x")}, {Relic: "./a", Scripture: []byte("y")}}
		}, `"a"`},
		{"a variable never granted", func(sh *shrine, _ *sanction.Verse) {
			sh.unit.Environment.Keys = append(sh.unit.Environment.Keys, "LD_PRELOAD")
		}, "LD_PRELOAD"},
		{"halls of a stranger", func(sh *shrine, _ *sanction.Verse) {
			sp := sh.speech()
			sp.Sovereign = func(uint32) bool { return false }
			sh.cfg.Voice = sp.Voice
		}, "root does not own"},
	} {
		t.Run(c.name, func(t *testing.T) {
			v := incantation(t, 1, "echo INJECTED")
			u := speakingUnit()
			sh := speakingShrine(t, u)
			c.mend(sh, &v)
			for i, p := range speakingUnit(v).Aspects {
				u.Aspects[i].Observances = p.Observances
			}
			sh.lay(sh.cfg.Store, u)
			inv := sh.rouseFor("on")
			r := inv.say(sh.handshake("on"))
			if r.Verdict != Refused || !strings.Contains(r.Lament, c.lament) {
				t.Fatalf("heard %+v, want a refusal naming %s", r, c.lament)
			}
			if m := inv.ended(); m != 2 {
				t.Fatalf("refused with the death-mark %d", m)
			}
			if entries, _ := os.ReadDir(filepath.Join(sh.ground, "run", "servitor")); len(entries) != 0 {
				t.Errorf("a working hall was left: %v", entries)
			}
		})
	}
}

// TestMachine_LendsExaltedSpeech: the exalted servitor of the machine
// speaks exalted incantations and litanies.
func TestMachine_LendsExaltedSpeech(t *testing.T) {
	if Machine().Voice == nil {
		t.Fatal("the exalted servitor of the machine has no voice")
	}
}

// TestSpeech_WeighsWithoutRaising: the speech of a plan is weighed by the
// rules of Voice before it is sanctioned, and no working hall is raised.
func TestSpeech_WeighsWithoutRaising(t *testing.T) {
	u := speakingUnit(incantation(t, 1, "echo weighed"))
	sh := speakingShrine(t, u)
	sitting := Sitting{UID: me, Sanction: sanction.New(u, time.Time{}), Plan: u.Aspects[0], Sigil: sh.sigil("on")}
	if err := sh.speech().Weigh(sitting); err != nil {
		t.Errorf("Weigh = %v", err)
	}
	sitting.Plan.Observances[0].Tongue = "bash"
	if err := sh.speech().Weigh(sitting); err == nil || !strings.Contains(err.Error(), "begin at the root") {
		t.Errorf("Weigh of a tongue spoken from no root = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(sh.ground, "run")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a hall was raised: %v", err)
	}
}

// TestSpeech_BidsEachTongueReadNoScriptureOfItsOwn: each tongue exalted
// speech knows is bidden, in its own words, to read no scripture of its
// own as it awakens, by the name of the program it truly is: bash
// --noprofile --norc, zsh -f, fish --no-config, sh and dash nothing. A
// reversion is spoken alike. Any other tongue refuses the session before
// anything is touched, naming the tongues it knows.
func TestSpeech_BidsEachTongueReadNoScriptureOfItsOwn(t *testing.T) {
	ground := t.TempDir()
	must(t, os.Chmod(ground, 0o755))
	must(t, os.MkdirAll(filepath.Join(ground, "bin"), 0o755))
	const stub = "#!/bin/sh\nprintf '%s|' \"$@\"\n"
	for _, name := range []string{"bash", "zsh", "fish", "dash", "python3"} {
		must(t, os.WriteFile(filepath.Join(ground, "bin", name), []byte(stub), 0o755))
	}
	must(t, os.Symlink("dash", filepath.Join(ground, "bin", "sh")))
	must(t, os.Symlink("bash", filepath.Join(ground, "bin", "rbash-alias")))
	tongues := hallowed.Ground{Root: ground, Sovereign: func(uid uint32) bool { return uid == uint32(me) }}
	for tongue, want := range map[string]string{
		"/bin/bash":        "--noprofile|--norc|-c|spoken|",
		"/bin/rbash-alias": "--noprofile|--norc|-c|spoken|",
		"/bin/zsh":         "-f|-c|spoken|",
		"/bin/fish":        "--no-config|-c|spoken|",
		"/bin/dash":        "-c|spoken|",
		"/bin/sh":          "-c|spoken|",
	} {
		t.Run(tongue, func(t *testing.T) {
			v := sanction.Verse{Verse: 1, Kind: librarium.KindIncantation, Target: "spoken", Tongue: tongue,
				Patience: patience, Relics: []sanction.Relic{}, Reversion: "unspoken"}
			sh := newShrine(t, speakingUnit(v))
			sp := sh.speech()
			sp.Tongues = tongues
			sh.cfg.Voice = sp.Voice
			inv := sh.accepted("on")
			if r := inv.perform(spoken(1)); r.Heard != want {
				t.Errorf("the tongue heard %q, want %q", r.Heard, want)
			}
			unspoken := strings.Replace(want, "spoken", "unspoken", 1)
			if r := inv.say(Request{Revert: spoken(1)}); r.Verdict != Reverted || r.Heard != unspoken {
				t.Errorf("the reversion: %+v, want heard %q", r, unspoken)
			}
			inv.say(Request{Finish: new(true)})
			inv.ended()
		})
	}
	t.Run("a tongue of no shell", func(t *testing.T) {
		v := sanction.Verse{Verse: 1, Kind: librarium.KindIncantation, Target: "print(1)", Tongue: "/bin/python3",
			Patience: patience, Relics: []sanction.Relic{}}
		sh := newShrine(t, speakingUnit(v))
		sp := sh.speech()
		sp.Tongues = tongues
		sh.cfg.Voice = sp.Voice
		inv := sh.rouseFor("on")
		r := inv.say(sh.handshake("on"))
		if r.Verdict != Refused || !strings.Contains(r.Lament, "python3") ||
			!strings.Contains(r.Lament, librarium.ExaltedTongueNames) {
			t.Fatalf("heard %+v, want a refusal naming python3 and the tongues known", r)
		}
		if m := inv.ended(); m != 2 {
			t.Fatalf("refused with the death-mark %d", m)
		}
		if err := sp.Weigh(Sitting{UID: me, Sanction: sanction.New(sh.unit, time.Time{}), Plan: sh.unit.Aspects[0],
			Sigil: sh.sigil("on")}); err == nil || !strings.Contains(err.Error(), "python3") {
			t.Errorf("Weigh = %v", err)
		}
	})
}
