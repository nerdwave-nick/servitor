package invocation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// perform prepares and performs r with opts and fails on any heresy of the
// pre-flight.
func perform(t *testing.T, r *librarium.Rite, opts Options) (Outcome, []Foresight) {
	t.Helper()
	inv, err := Prepare(r, opts)
	if err != nil {
		t.Fatalf("pre-flight of %q:\n%v", opts.Aspect, err)
	}
	out, err := inv.Perform()
	if err != nil {
		t.Fatal(err)
	}
	return out, inv.Foresee()
}

// tongueOnPath places an executable named name on PATH that records its
// name in log and then speaks through bash.
func tongueOnPath(t *testing.T, bin, name, log string) {
	t.Helper()
	write(t, bin, name, "#!/bin/sh\necho "+name+" >> "+log+"\nexec bash \"$@\"\n", 0o755)
}

func lines(t *testing.T, p string) []string {
	t.Helper()
	if !exists(p) {
		return nil
	}
	return strings.Split(strings.TrimSuffix(read(t, p), "\n"), "\n")
}

func TestIncantation_SpeaksInBashWithTheInvocationsEnvironment(t *testing.T) {
	fx := newFixture(t)
	t.Setenv("SERVITOR_INSCRIPTION_STALE", "from a former invocation")
	t.Setenv("SERVITOR_ASPECT", "stale")
	r := fx.rite(t, `{"incantation": "printf '%s|' \"$SERVITOR_ASPECT\" \"$SERVITOR_FORMER_ASPECT\" \"$SERVITOR_RITE\" \"$SERVITOR_INSCRIPTION_REASON\" \"$SERVITOR_INSCRIPTION_STALE\" \"{{aspect}}-{{inscription.reason}}\" \"${BASH_VERSION:+bash}\" \"$PWD\" > $DATA/said; echo uttered; echo lamented >&2"}`)

	out, seen := perform(t, r, Options{Aspect: "on", Former: "off", Inscriptions: map[string]string{"reason": "gaming remnant"}})

	if out.Verdict != Triumph || out.Fell != nil || len(out.Reversions) != 0 {
		t.Fatalf("outcome %+v", out)
	}
	want := "on|off|mouse|gaming remnant||on-gaming remnant|bash|" + r.Dir() + "|"
	if got := read(t, filepath.Join(fx.data, "said")); got != want {
		t.Fatalf("the incantation said %q, want %q", got, want)
	}
	if len(out.Deeds) != 1 || out.Deeds[0].Output != "uttered\nlamented\n" || out.Deeds[0].Heresy != nil {
		t.Fatalf("deeds %+v", out.Deeds)
	}
	if d := out.Deeds[0]; d.Number != 1 || d.Kind != librarium.KindIncantation || !strings.HasPrefix(d.Target, "printf") {
		t.Fatalf("deed names %+v", d.Verse)
	}
	if s := seen[0].Speech; s == nil || s.Tongue != "bash" || s.Argv[0] != "bash" || s.Argv[1] != "-c" || s.Argv[2] != s.Words {
		t.Fatalf("foresaw %+v", seen[0].Speech)
	}
}

func TestIncantation_TongueOfStepOverRiteOverSettingsOverBash(t *testing.T) {
	bin, log := t.TempDir(), filepath.Join(t.TempDir(), "spoken")
	for _, name := range []string{"tongue-of-step", "tongue-of-rite", "tongue-of-settings"} {
		tongueOnPath(t, bin, name, log)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	cases := []struct {
		name, riteKeys, step, settings, want string
	}{
		{"step", `"tongue": "tongue-of-rite",`, `"tongue": "tongue-of-step",`, "tongue-of-settings", "tongue-of-step"},
		{"rite", `"tongue": "tongue-of-rite",`, "", "tongue-of-settings", "tongue-of-rite"},
		{"settings", "", "", "tongue-of-settings", "tongue-of-settings"},
		{"bash", "", "", "", "bash"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_ = os.Remove(log)
			fx := newFixture(t)
			r := fx.riteWith(t, c.riteKeys, `{`+c.step+` "incantation": "true"}`)
			out, seen := perform(t, r, Options{Aspect: "on", Orders: librarium.Orders{Tongue: c.settings}})
			if out.Verdict != Triumph {
				t.Fatalf("outcome %+v", out)
			}
			if s := seen[0].Speech; s.Tongue != c.want || s.Argv[0] != c.want {
				t.Fatalf("foresaw tongue %q (%v), want %q", s.Tongue, s.Argv, c.want)
			}
			spoken := lines(t, log)
			if c.want == "bash" {
				if spoken != nil {
					t.Fatalf("bash was not the tongue: %v", spoken)
				}
				return
			}
			if !slices.Equal(spoken, []string{c.want}) {
				t.Fatalf("spoken through %v, want %s", spoken, c.want)
			}
		})
	}
}

// recorder is a litany that writes how many offerings it received, and
// each of them, to the vessel named by its first offering.
const recorder = "out=$1\nshift\nprintf '%s\\n' \"$#\" \"$@\" > \"$out\"\n"

func TestLitany_RecitedByItsShebangWithExactlyItsOfferings(t *testing.T) {
	fx := newFixture(t)
	scroll := write(t, fx.lib, "rites/scripts/recite", "#!/bin/bash\n"+recorder, 0o755)
	args := filepath.Join(fx.data, "args")
	r := fx.rite(t, `{"litany": "scripts/recite", "offerings": ["$DATA/args", "two words", "$HOME; rm -rf /",
	  "~/themes/{{aspect}}", {"on": "lit", "*": "unlit"}, "it's \"quoted\"", ""]}`)

	out, seen := perform(t, r, Options{Aspect: "on"})

	if out.Verdict != Triumph {
		t.Fatalf("outcome %+v", out)
	}
	want := []string{"6", "two words", "$HOME; rm -rf /", "~/themes/on", "lit", `it's "quoted"`, ""}
	if got := strings.Split(read(t, args), "\n"); !slices.Equal(got[:len(got)-1], want) {
		t.Fatalf("offered %q, want %q", got, want)
	}
	s := seen[0].Speech
	if !s.Shebang || s.Words != scroll || !slices.Equal(s.Argv, append([]string{scroll, args}, want[1:]...)) {
		t.Fatalf("foresaw %+v", s)
	}
	if out.Deeds[0].Target != scroll {
		t.Fatalf("deed names %+v", out.Deeds[0].Verse)
	}
}

func TestLitany_ReceivesNothingItIsNotOffered(t *testing.T) {
	fx := newFixture(t)
	write(t, fx.lib, "rites/scripts/count", "#!/bin/bash\necho \"$#\"\n", 0o755)
	r := fx.rite(t, `{"litany": "scripts/count"}`)
	out, _ := perform(t, r, Options{Aspect: "on", Former: "off"})
	if out.Deeds[0].Output != "0\n" {
		t.Fatalf("the litany was offered %q arguments", out.Deeds[0].Output)
	}
}

func TestLitany_RecitedThroughTheTongueWhenItCannotBeExecuted(t *testing.T) {
	fx := newFixture(t)
	scroll := write(t, fx.data, "scroll", recorder+"echo \"${BASH_VERSION:+bash}\"\n", 0o644)
	args := filepath.Join(fx.data, "args")
	r := fx.rite(t, `{"litany": "$DATA/scroll", "offerings": ["$DATA/args", "{{aspect}}"]}`)

	out, seen := perform(t, r, Options{Aspect: "off"})

	if out.Verdict != Triumph || out.Deeds[0].Output != "bash\n" {
		t.Fatalf("outcome %+v", out)
	}
	if got := read(t, args); got != "1\noff\n" {
		t.Fatalf("offered %q", got)
	}
	if s := seen[0].Speech; s.Shebang || s.Tongue != "bash" || !slices.Equal(s.Argv, []string{"bash", scroll, args, "off"}) {
		t.Fatalf("foresaw %+v", s)
	}
}

func TestLitany_MayBeWrittenByAnEarlierStep(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `{"transcription": "$DATA/scroll", "scripture": "#!/bin/bash\necho recited {{aspect}}", "seal": "0755"},
	  {"litany": "$DATA/scroll"}`)
	out, seen := perform(t, r, Options{Aspect: "on"})
	if out.Verdict != Triumph || out.Deeds[1].Output != "recited on\n" {
		t.Fatalf("outcome %+v", out)
	}
	if !seen[1].Speech.Shebang {
		t.Fatalf("foresaw %+v", seen[1].Speech)
	}
}

// gone reports whether the process pid has ended (a zombie has ended too).
func gone(pid int) bool {
	if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
		return true
	}
	stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return true
	}
	fields := strings.Fields(string(stat[strings.LastIndexByte(string(stat), ')')+1:]))
	return len(fields) > 0 && fields[0] == "Z"
}

func pidIn(t *testing.T, p string) int {
	t.Helper()
	pid, err := strconv.Atoi(strings.TrimSpace(read(t, p)))
	if err != nil {
		t.Fatal(err)
	}
	return pid
}

func TestPatience_ExpiryKillsTheWholeGroupAndTheStepFalls(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `{"incantation": "sleep 60 & echo $! > $DATA/child; echo waiting; sleep 60", "patience": "300ms"},
	  {"incantation": "touch $DATA/never"}`)

	start := time.Now()
	out, seen := perform(t, r, Options{Aspect: "on"})
	elapsed := time.Since(start)

	if elapsed > 10*time.Second {
		t.Fatalf("patience of 300ms ran out only after %v", elapsed)
	}
	if out.Verdict != Reverted || out.Fell == nil || out.Fell.Number != 1 || out.Fell.Output != "waiting\n" {
		t.Fatalf("outcome %+v", out)
	}
	grimdark(t, out.Fell.Heresy.Error())
	if !strings.Contains(out.Fell.Heresy.Error(), "300ms") {
		t.Errorf("the heresy %q does not tell the patience", out.Fell.Heresy)
	}
	if exists(filepath.Join(fx.data, "never")) {
		t.Fatal("a step after the fallen one was spoken")
	}
	child := pidIn(t, filepath.Join(fx.data, "child"))
	deadline := time.Now().Add(5 * time.Second)
	for !gone(child) {
		if time.Now().After(deadline) {
			_ = syscall.Kill(child, syscall.SIGKILL)
			t.Fatal("a child of the impatient step outlived it")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if seen[0].Speech.Patience != 300*time.Millisecond {
		t.Fatalf("foresaw patience %v", seen[0].Speech.Patience)
	}
}

func TestPatience_OfStepOverSettingsOverDefault(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `{"incantation": "sleep 0.5", "patience": "20s"}, {"incantation": "sleep 20"}`)

	out, seen := perform(t, r, Options{Aspect: "on", Orders: librarium.Orders{Patience: 200 * time.Millisecond}})

	if out.Fell == nil || out.Fell.Number != 2 {
		t.Fatalf("outcome %+v", out)
	}
	if seen[0].Speech.Patience != 20*time.Second || seen[1].Speech.Patience != 200*time.Millisecond {
		t.Fatalf("foresaw patience %v and %v", seen[0].Speech.Patience, seen[1].Speech.Patience)
	}
	inv, err := Prepare(r, Options{Aspect: "on"})
	if err != nil {
		t.Fatal(err)
	}
	if p := inv.Foresee()[1].Speech.Patience; p != librarium.DefaultPatience {
		t.Fatalf("patience without settings %v", p)
	}
}

func TestIncantation_BackgroundChildrenOfATriumphantStepSurvive(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `{"incantation": "sleep 60 & echo $! > $DATA/child; echo spoken"},
	  {"incantation": "echo after"}`)

	start := time.Now()
	out, _ := perform(t, r, Options{Aspect: "on"})
	elapsed := time.Since(start)

	child := pidIn(t, filepath.Join(fx.data, "child"))
	t.Cleanup(func() { _ = syscall.Kill(child, syscall.SIGKILL) })
	if out.Verdict != Triumph || out.Deeds[0].Output != "spoken\n" || out.Deeds[1].Output != "after\n" {
		t.Fatalf("outcome %+v", out)
	}
	if elapsed > 10*time.Second {
		t.Fatalf("the child held the invocation for %v", elapsed)
	}
	if gone(child) {
		t.Fatal("the child of a triumphant step was slain")
	}
}

func TestCommands_RevertedFromTheFallenStepBackToTheFirst(t *testing.T) {
	fx := newFixture(t)
	util := write(t, fx.data, "util.kdl", "input {}\n", 0o644)
	scroll := write(t, fx.data, "scroll", "echo recited >> \"$1\"\necho dying words\nexit 3\n", 0o644)
	log := filepath.Join(fx.data, "log")
	r := fx.rite(t, `{"sanctum": "$DATA/util.kdl", "scripture": "x"},
	  {"incantation": "echo spoke-{{aspect}} >> $DATA/log",
	   "reversion": "echo unspoke-{{former}}-$SERVITOR_FORMER_ASPECT >> $DATA/log"},
	  {"incantation": "echo quiet >> $DATA/log"},
	  {"vox-cast": "progress"},
	  {"litany": "$DATA/scroll", "offerings": ["$DATA/log"],
	   "reversion": {"on": "echo unrecited-{{aspect}} >> $DATA/log; echo undone", "*": "false"}},
	  {"incantation": "echo never >> $DATA/log"}`)

	out, _ := perform(t, r, Options{Aspect: "on", Former: "off"})

	if out.Verdict != Reverted {
		t.Fatalf("verdict %q", out.Verdict)
	}
	f := out.Fell
	if f == nil || f.Number != 5 || f.Kind != librarium.KindLitany || f.Target != scroll || f.Output != "dying words\n" {
		t.Fatalf("fell %+v", f)
	}
	grimdark(t, f.Heresy.Error())
	if !strings.Contains(f.Heresy.Error(), "3") {
		t.Errorf("the heresy %q does not tell the status", f.Heresy)
	}
	var verses []int
	for _, rv := range out.Reversions {
		if rv.Heresy != nil {
			t.Errorf("reversion of verse %d fell: %v", rv.Number, rv.Heresy)
		}
		verses = append(verses, rv.Number)
	}
	if !slices.Equal(verses, []int{5, 2, 1}) {
		t.Fatalf("reverted verses %v", verses)
	}
	if out.Reversions[0].Output != "undone\n" {
		t.Fatalf("reversion uttered %q", out.Reversions[0].Output)
	}
	if want := []string{"spoke-on", "quiet", "recited", "unrecited-on", "unspoke-off-off"}; !slices.Equal(lines(t, log), want) {
		t.Fatalf("log %q, want %q", lines(t, log), want)
	}
	if read(t, util) != "input {}\n" {
		t.Fatal("the sanctum was not reverted")
	}
	if len(out.Deeds) != 5 || out.Deeds[4].Heresy == nil || out.Deeds[2].Output != "" {
		t.Fatalf("deeds %+v", out.Deeds)
	}
}

func TestCommands_FalterWhenAReversionFalls(t *testing.T) {
	fx := newFixture(t)
	log := filepath.Join(fx.data, "log")
	r := fx.rite(t, `{"incantation": "true", "reversion": "echo first-undone >> $DATA/log"},
	  {"incantation": "true", "reversion": "echo broken; exit 1"},
	  {"incantation": "true", "reversion": "sleep 20", "patience": "200ms"},
	  {"incantation": "exit 1"}`)

	out, _ := perform(t, r, Options{Aspect: "on"})

	if out.Verdict != Faltered || out.Fell == nil || out.Fell.Number != 4 {
		t.Fatalf("outcome %+v", out)
	}
	if len(out.Reversions) != 3 {
		t.Fatalf("reversions %+v", out.Reversions)
	}
	r3, r2, r1 := out.Reversions[0], out.Reversions[1], out.Reversions[2]
	if r3.Number != 3 || r3.Heresy == nil || r2.Number != 2 || r2.Heresy == nil || r2.Output != "broken\n" ||
		r1.Number != 1 || r1.Heresy != nil {
		t.Fatalf("reversions %+v", out.Reversions)
	}
	grimdark(t, r2.Heresy.Error())
	grimdark(t, r3.Heresy.Error())
	if !slices.Equal(lines(t, log), []string{"first-undone"}) {
		t.Fatalf("log %q", lines(t, log))
	}
}

func TestForesee_ListsRenderedCommandsAndSpeaksNothing(t *testing.T) {
	fx := newFixture(t)
	scroll := write(t, fx.data, "scroll", "#!/bin/bash\ntouch \"$1\"\n", 0o755)
	r := fx.rite(t, `{"incantation": "touch $DATA/spoken-{{aspect}}", "reversion": "rm -f $DATA/spoken-{{former}}"},
	  {"litany": "$DATA/scroll", "offerings": ["$DATA/recited-{{inscription.reason}}"], "tongue": "sh"}`)
	before := tree(t, fx.data)

	inv, err := Prepare(r, Options{Aspect: "on", Former: "off", Inscriptions: map[string]string{"reason": "why"}})
	if err != nil {
		t.Fatal(err)
	}
	seen := inv.Foresee()
	sameTree(t, before, tree(t, fx.data))

	inc := seen[0].Speech
	if inc.Words != "touch "+fx.data+"/spoken-on" || inc.Reversion != "rm -f "+fx.data+"/spoken-off" || inc.Tongue != "bash" {
		t.Fatalf("incantation foreseen as %+v", inc)
	}
	lit := seen[1].Speech
	if lit.Tongue != "sh" || !lit.Shebang || lit.Words != scroll || !slices.Equal(lit.Offerings, []string{fx.data + "/recited-why"}) {
		t.Fatalf("litany foreseen as %+v", lit)
	}
	if seen[0].Vessel != nil || seen[0].Tether != nil || seen[0].Kind != librarium.KindIncantation {
		t.Fatalf("foresaw %+v", seen[0])
	}
}

func TestPrepare_RefusesCommandsItCannotSpeak(t *testing.T) {
	cases := []struct {
		name, liturgy string
		orders        librarium.Orders
		want          string
	}{
		{"tongue of the step unknown", `{"incantation": "true", "tongue": "no-such-tongue-of-the-machine"}`, librarium.Orders{},
			"no-such-tongue-of-the-machine"},
		{"tongue of the settings unknown", `{"incantation": "true"}`, librarium.Orders{Tongue: "no-such-tongue-of-the-settings"},
			"no-such-tongue-of-the-settings"},
		{"tongue of a reversion unknown", `{"litany": "$DATA/exe", "reversion": "true", "tongue": "no-such-tongue"}`, librarium.Orders{},
			"no-such-tongue"},
		{"tongue of an unexecutable litany unknown", `{"litany": "$DATA/plain", "tongue": "no-such-tongue"}`, librarium.Orders{},
			"no-such-tongue"},
		{"scroll absent", `{"litany": "$DATA/absent"}`, librarium.Orders{}, "absent"},
		{"scroll a hall", `{"litany": "$DATA"}`, librarium.Orders{}, "scroll"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fx := newFixture(t)
			write(t, fx.data, "exe", "#!/bin/sh\ntouch spoken\n", 0o755)
			write(t, fx.data, "plain", "touch spoken\n", 0o644)
			r := fx.rite(t, `{"sanctum": "$DATA/util.kdl", "consecrate": true, "scripture": "x"}, `+c.liturgy)
			before := tree(t, fx.data)
			hs := refused(t, r, Options{Aspect: "on", Orders: c.orders}, c.want)
			sameTree(t, before, tree(t, fx.data))
			if len(hs) != 1 || hs[0].Verse != 2 || hs[0].Line == 0 {
				t.Fatalf("heresies %+v", hs)
			}
			grimdark(t, hs[0].Message)
		})
	}
}

func TestPrepare_ExecutableLitanyNeedsNoTongue(t *testing.T) {
	fx := newFixture(t)
	write(t, fx.data, "exe", "#!/bin/sh\necho recited\n", 0o755)
	r := fx.rite(t, `{"litany": "$DATA/exe", "tongue": "no-such-tongue"}`)
	out, _ := perform(t, r, Options{Aspect: "on"})
	if out.Verdict != Triumph || out.Deeds[0].Output != "recited\n" {
		t.Fatalf("outcome %+v", out)
	}
}

func TestPerformContext_HaltSlaysTheSpokenStepAndRevertsTheRest(t *testing.T) {
	fx := newFixture(t)
	util := write(t, fx.data, "util.kdl", "input {}\n", 0o644)
	r := fx.rite(t, `{"sanctum": "$DATA/util.kdl", "scripture": "x"},
	  {"incantation": "sleep 30 & echo $! > $DATA/child; wait", "reversion": "echo unspoken >> $DATA/log"},
	  {"incantation": "echo never >> $DATA/log"}`)
	inv, err := Prepare(r, Options{Aspect: "on"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, halt := context.WithCancel(context.Background())
	defer halt()
	childFile := filepath.Join(fx.data, "child")
	go func() {
		for b, _ := os.ReadFile(childFile); len(b) == 0; b, _ = os.ReadFile(childFile) {
			time.Sleep(10 * time.Millisecond)
		}
		halt()
	}()

	start := time.Now()
	out, err := inv.PerformContext(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("the halt was not heeded for %v", elapsed)
	}
	if out.Verdict != Reverted || out.Fell == nil || out.Fell.Number != 2 {
		t.Fatalf("outcome %+v", out)
	}
	grimdark(t, out.Fell.Heresy.Error())
	if !strings.Contains(out.Fell.Heresy.Error(), "halted") {
		t.Errorf("the heresy %q does not tell of the halt", out.Fell.Heresy)
	}
	if want := []string{"unspoken"}; !slices.Equal(lines(t, filepath.Join(fx.data, "log")), want) {
		t.Fatalf("log %q, want %q", lines(t, filepath.Join(fx.data, "log")), want)
	}
	if read(t, util) != "input {}\n" {
		t.Fatal("the sanctum was not reverted")
	}
	child, _ := strconv.Atoi(strings.TrimSpace(read(t, childFile)))
	deadline := time.Now().Add(5 * time.Second)
	for !gone(child) {
		if time.Now().After(deadline) {
			_ = syscall.Kill(child, syscall.SIGKILL)
			t.Fatal("a child of the halted step outlived it")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
