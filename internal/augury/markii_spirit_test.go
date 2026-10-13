package augury

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAugurMarkII_TheScanOfAMachineSpirit: what a scan prints, trimmed and
// at most 4 KiB of it, is the omen read; the aspects whose omen it is fit.
// Without omens every word printed must name a declared aspect. An empty,
// undeclared, falling or impatient scan leaves the subject mute.
func TestAugurMarkII_TheScanOfAMachineSpirit(t *testing.T) {
	shared := `{"on": "off", "*": "on"}`
	for _, tc := range []struct {
		name, omens, scan, shows string
		want                     string // readings
		heed                     Heed
	}{
		{"an omen of one aspect", shared, `cat $DATA/smt`, "off", "smt=on(off)", Heeded},
		{"a shared omen", shared, `cat $DATA/smt`, "on", "smt=off|auto(on)", Heeded},
		{"trimmed", shared, `cat $DATA/smt`, "  off\n\n", "smt=on(off)", Heeded},
		{"rendered for its own aspect", `{"*": "mode-{{aspect}}"}`, `cat $DATA/smt`, "mode-auto", "smt=auto(mode-auto)", Heeded},
		{"heard to 4 KiB", shared, `cat $DATA/smt`, "off" + strings.Repeat(" ", 5000) + "porpl", "smt=on(off)", Heeded},
		{"aspect names", ``, `cat $DATA/smt`, "auto\n", "smt=auto(auto)", Heeded},
		{"several aspect names", ``, `cat $DATA/smt`, "on auto", "smt=on|auto", Heeded},
		{"an undeclared omen", shared, `cat $DATA/smt`, "porpl", "", Mute},
		{"a word naming no aspect", ``, `cat $DATA/smt`, "on porpl", "", Mute},
		{"empty", shared, `cat $DATA/smt`, "  \n", "", Mute},
		{"falling", shared, `cat $DATA/smt; exit 1`, "off", "", Mute},
		{"only its standard output", shared, `echo on >&2; cat $DATA/smt`, "off", "smt=on(off)", Heeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newFixture(t)
			fx.shows(t, "smt", tc.shows)
			r := fx.stateRite(t, `"on", "off", "auto"`, "", spirit("smt", tc.omens, `{"scan": `+quote(tc.scan)+`}`))
			a := fx.augur(t, r, Options{})
			if got := readings(a); got != tc.want || a.Auspex != tc.heed {
				t.Errorf("readings %q heed %q, want %q %q", got, a.Auspex, tc.want, tc.heed)
			}
		})
	}
}

func TestAugurMarkII_AnImpatientScanIsSlainAndMute(t *testing.T) {
	fx := newFixture(t)
	r := fx.stateRite(t, `"on", "off"`, "", spirit("smt", "", `{"scan": "sleep 3 & sleep 3; echo on", "patience": "100ms"}`))
	start := time.Now()
	a := fx.augur(t, r, Options{})
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("the augury waited %v for an impatient scan", elapsed)
	}
	if got := readings(a); got != "" || a.Auspex != Mute {
		t.Errorf("readings %q heed %q, want none and mute", got, a.Auspex)
	}
}

// TestAugurMarkII_TheTrialsOfAMachineSpirit: every trial is spoken at once
// under one patience, hearing its own omen; the omens whose trial ends with
// the death-mark 0 fit, and when none does, the omens without a trial. One
// impatient, unspeakable or slain trial mutes the whole auspex.
func TestAugurMarkII_TheTrialsOfAMachineSpirit(t *testing.T) {
	scheduler := `{"on": "cosmos", "*": "eevdf"}`
	for _, tc := range []struct {
		name, omens, trials, keys string
		shows                     string // the content of $DATA/s; "" leaves it absent
		want                      string
		heed                      Heed
	}{
		{"its trial succeeds", scheduler, `{"cosmos": "test -e $DATA/s"}`, "", "x", "scheduler=on(cosmos)", Heeded},
		{"the omen without a trial", scheduler, `{"cosmos": "test -e $DATA/s"}`, "", "", "scheduler=off|auto(eevdf)", Heeded},
		{"its own omen in the environment", "", `{"*": "[ \"$(cat $DATA/s)\" = \"$SERVITOR_OMEN\" ]"}`, "", "auto", "scheduler=auto(auto)", Heeded},
		{"its own omen as a placeholder", "", `{"*": "grep -qx {{omen}} $DATA/s"}`, "", "off", "scheduler=off(off)", Heeded},
		{"several omens", "", `{"*": "grep -qx {{omen}} $DATA/s"}`, "", "off\non", "scheduler=on|off", Heeded},
		{"every omen tried and none fits", "", `{"*": "grep -qx {{omen}} $DATA/s"}`, "", "porpl", "", Mute},
		{"one impatient trial", scheduler, `{"cosmos": "sleep 3", "eevdf": "true"}`, "", "", "", Mute},
		{"one slain trial", scheduler, `{"cosmos": "kill -9 $$", "eevdf": "true"}`, "", "", "", Mute},
		{"an unspeakable tongue", scheduler, `{"cosmos": "true"}`, `"tongue": "no-such-tongue-of-the-machine", `, "", "", Mute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newFixture(t)
			if tc.shows != "" {
				fx.shows(t, "s", tc.shows)
			}
			sub := spirit("scheduler", tc.omens, `{"trials": `+tc.trials+`, "patience": "300ms"}`)
			sub = strings.Replace(sub, `"auspex"`, tc.keys+`"auspex"`, 1)
			r := fx.stateRite(t, `"on", "off", "auto"`, "", sub)
			start := time.Now()
			a := fx.augur(t, r, Options{})
			if elapsed := time.Since(start); elapsed > 2*time.Second {
				t.Errorf("the augury waited %v", elapsed)
			}
			if got := readings(a); got != tc.want || a.Auspex != tc.heed {
				t.Errorf("readings %q heed %q, want %q %q", got, a.Auspex, tc.want, tc.heed)
			}
		})
	}
}

// TestAugurMarkII_TheScanReadsBesideTrials: an auspex holding both reads
// by its scan alone; its trials are not spoken.
func TestAugurMarkII_TheScanReadsBesideTrials(t *testing.T) {
	fx := newFixture(t)
	mark := filepath.Join(fx.data, "tried")
	fx.shows(t, "smt", "on")
	r := fx.stateRite(t, `"on", "off"`, "", spirit("smt", "", `{"scan": "cat $DATA/smt", "trials": {"*": "touch `+mark+`"}}`))
	if got := readings(fx.augur(t, r, Options{})); got != "smt=on(on)" {
		t.Errorf("readings %q", got)
	}
	if exists(mark) {
		t.Error("a trial was spoken beside a scan")
	}
}

// TestAugurMarkII_EveryAuspexLaboursAtOnce: the auspexes of every subject
// and every trial are awakened together, so an augury lasts as long as its
// slowest auspex.
func TestAugurMarkII_EveryAuspexLaboursAtOnce(t *testing.T) {
	fx := newFixture(t)
	nap := `sleep 0.5; `
	r := fx.stateRite(t, `"on", "off"`, "",
		spirit("a", "", `{"scan": "`+nap+`echo on"}`)+", "+
			spirit("b", "", `{"scan": "`+nap+`echo on"}`)+", "+
			spirit("c", "", `{"trials": {"on": "`+nap+`true", "off": "`+nap+`false"}}`))
	start := time.Now()
	a := fx.augur(t, r, Options{})
	if elapsed := time.Since(start); elapsed > 900*time.Millisecond {
		t.Errorf("the augury took %v, its auspexes labouring one after another", elapsed)
	}
	if got := readings(a); got != "a=on(on) b=on(on) c=on(on)" {
		t.Errorf("readings %q", got)
	}
}

// TestAugurMarkII_TheAuspexHearsTheRiteAndTheTarget: a scan is spoken in
// the subject's tongue, else the rite's, from the rite's hall, hearing the
// rite, the data-slate's inscriptions and, during an invocation alone, the
// target aspect.
func TestAugurMarkII_TheAuspexHearsTheRiteAndTheTarget(t *testing.T) {
	fx := newFixture(t)
	scan := `[ "$0" = sh ] && [ "$PWD" = HALL ] && [ "$SERVITOR_RITE" = gaming ] || exit 1; ` +
		`echo "${SERVITOR_ASPECT:-$SERVITOR_INSCRIPTION_PLACE}"`
	scan = strings.ReplaceAll(scan, "HALL", filepath.Join(fx.lib, "rites"))
	sub := strings.Replace(spirit("smt", "", `{"scan": `+quote(scan)+`}`), `"auspex"`, `"tongue": "sh", "auspex"`, 1)
	r := fx.stateRite(t, `"on", "off"`, `
  "tongue": "no-such-tongue-of-the-machine",`, sub)
	fx.slate(t, r, "off", "triumph", map[string]string{"place": "on"})
	if got := readings(fx.augur(t, r, Options{})); got != "smt=on(on)" {
		t.Errorf("outside an invocation: readings %q", got)
	}
	if got := readings(fx.augur(t, r, Options{Target: "off"})); got != "smt=off(off)" {
		t.Errorf("during an invocation into off: readings %q", got)
	}
}

// TestAugurMarkII_APrivilegedAuspexIsNeverSpoken: the augury never speaks
// an exalted auspex, whose subject yields no omen; the heed is told of the
// other auspexes alone.
func TestAugurMarkII_APrivilegedAuspexIsNeverSpoken(t *testing.T) {
	fx := newFixture(t)
	mark := filepath.Join(fx.data, "spoken")
	fx.shows(t, "smt", "on")
	privileged := `{"name": "nft", "designation": "machine-spirit", "exalted": true,
	  "auspex": {"exalted": true, "scan": "touch ` + mark + `; echo on", "trials": {"*": "touch ` + mark + `"}},
	  "enact": {"*": "true"}}`
	for _, tc := range []struct {
		name, subjects, want string
		heed                 Heed
	}{
		{"alone", privileged, "", NoAuspex},
		{"beside another", privileged + ", " + spirit("smt", "", `{"scan": "cat $DATA/smt"}`), "smt=on(on)", Heeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := fx.stateRite(t, `"on", "off"`, "", tc.subjects)
			for _, opts := range []Options{{}, {Target: "on"}} {
				a := fx.augur(t, r, opts)
				if got := readings(a); got != tc.want || a.Auspex != tc.heed {
					t.Errorf("%+v: readings %q heed %q, want %q %q", opts, got, a.Auspex, tc.want, tc.heed)
				}
			}
			if exists(mark) {
				t.Fatal("a privileged auspex was spoken")
			}
		})
	}
}

// TestAugurMarkII_TheAuspexMayBeForgone: a forgone augury awakens no scan
// nor trial, and marks the rite forgone.
func TestAugurMarkII_TheAuspexMayBeForgone(t *testing.T) {
	fx := newFixture(t)
	mark := filepath.Join(fx.data, "awakened")
	r := fx.stateRite(t, `"on", "off"`, "", spirit("smt", "", `{"scan": "touch `+mark+`; echo on"}`)+", "+
		spirit("sched", "", `{"trials": {"*": "touch `+mark+`"}}`))
	a := fx.augur(t, r, Options{ForgoAuspex: true})
	if exists(mark) {
		t.Error("a forgone auspex was awakened")
	}
	if got := readings(a); got != "" || a.Auspex != Forgone {
		t.Errorf("readings %q heed %q, want none and forgone", got, a.Auspex)
	}
}
