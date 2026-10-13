package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// edictIIJSON is the edict of the captured visage written in pattern
// Mark II, leaving a mark beside the vessel each time it is proclaimed.
const edictIIJSON = `{"pattern": "Mark II", "purpose": "Capture the visage of the machine", "edict": true,
  "inscriptions": {"region": {"purpose": "the slurp geometry", "decrees": "full"}},
  "vox-cast": ["progress", "success"],
  "liturgy": [
    {"designation": "incantation", "patience": "2m", "words": "echo {{inscription.region}} >> $TARGET.edict",
     "reversion": "echo struck >> $TARGET.edict"},
    {"designation": "incantation", "words": "true"}]}`

// TestEdictII_NoRitualFallsBeforeIt: an edict of Mark II is read, listed,
// examined, expounded and completed by every ritual; its own help names
// its entries by their verses, and its vox-casts are no verses.
func TestEdictII_NoRitualFallsBeforeIt(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", edictIIJSON)
	for _, args := range [][]string{
		{"census"}, {"census", "--binharic"}, {"inquisition"}, {"inquisition", "capture", "--binharic"},
		{"completion", "bash"}, {"invoke", "--help"}, {"invoke", "capture", "--help"},
	} {
		if out, errOut, code := e.run(args...); code != 0 {
			t.Errorf("servitor %v: exit %d\nstdout: %s\nstderr: %s", args, code, out, errOut)
		}
	}
	help := e.mustRun("invoke", "capture", "--help")
	for _, want := range []string{"servitor invoke capture --region", "Edict: proclaimed whole, bearing no aspect",
		"verse 1 · incantation echo {{inscription.region}} >> " + e.target + ".edict", "verse 2 · incantation true"} {
		if !strings.Contains(help, want) {
			t.Errorf("the edict's own help lacks %q:\n%s", want, help)
		}
	}
	if strings.Contains(help, "vox-cast") || strings.Contains(help, "verse 3") || strings.Contains(help, "Aspects:") {
		t.Errorf("the edict's own help speaks of vox-cast verses or aspects:\n%s", help)
	}
	if aspects, directive := e.complete("invoke", "capture", ""); len(aspects) != 0 || directive != noFileComp {
		t.Errorf("an edict completes aspects %v (%s)", aspects, directive)
	}
	if words, _ := e.complete("invoke", "capture", "--region", ""); !slices.Contains(words, "full") {
		t.Errorf("the edict's decree does not complete: %v", words)
	}
}

// TestEdictII_ProclaimedWholeAtEveryInvocation: an edict of Mark II is
// invoked by its name alone, performs its whole liturgy each time, sends
// its rite's vox-casts to the desktop as one missive replaced in place —
// progress after every verse, then success — and is kept on its
// data-slate and chronicled without an aspect.
func TestEdictII_ProclaimedWholeAtEveryInvocation(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", edictIIJSON)

	first := e.mustRun("invoke", "capture")
	second := e.mustRun("invoke", "capture", "--region", "window")

	if got, _ := os.ReadFile(e.target + ".edict"); string(got) != "full\nwindow\n" {
		t.Fatalf("the edict left %q", got)
	}
	for _, out := range []string{first, second} {
		got := lines(out)
		if len(got) != 4 || got[0] != "+++ capture · edict +++" || !strings.HasPrefix(got[1], "  ✔ verse 1 · incantation echo ") ||
			!strings.HasPrefix(got[2], "  ✔ verse 2 · incantation true") || !strings.HasPrefix(got[3], "✠ ") ||
			!strings.Contains(strings.ToLower(got[3]), "the edict capture") || aspectless.MatchString(out) {
			t.Fatalf("the report:\n%s", out)
		}
	}
	notified := strings.Split(strings.TrimPrefix(e.notified(), "-a servitor "), "-a servitor ")
	if len(notified) != 6 || strings.Contains(notified[0], "-r ") || !strings.Contains(notified[1], "-r 7") ||
		!strings.Contains(notified[2], "-r 7") || !strings.Contains(notified[0], "1 / 2") ||
		!strings.Contains(notified[1], "2 / 2") || !strings.Contains(notified[5], "-r 7") {
		t.Fatalf("the desktop heard:\n%s", e.notified())
	}
	for _, n := range notified {
		if !strings.Contains(n, "-- capture ") || aspectless.MatchString(n) {
			t.Errorf("the desktop heard %q", n)
		}
	}
	slate, _ := os.ReadFile(filepath.Join(e.state, "servitor", "data-slates", "capture.json"))
	if !regexp.MustCompile(`^\{"inscriptions":\{"region":"window"\},"last_rite":\{"verdict":"triumph","at":"[^"]+"\}\}\n$`).Match(slate) {
		t.Fatalf("data-slate %s", slate)
	}
	chronicled := e.chronicleLines()
	line := regexp.MustCompile(`^\{"at":"[^"]+","rite":"capture","edict":true,"inscriptions":\{"region":"(full|window)"\},"verdict":"triumph"\}$`)
	if len(chronicled) != 2 || !line.MatchString(chronicled[0]) || !line.MatchString(chronicled[1]) {
		t.Fatalf("chronicle:\n%s", strings.Join(chronicled, "\n"))
	}
}

func TestEdictII_OnATerminalIsToldAsItHappensWithoutAnAspect(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", edictIIJSON)
	e.terminal(0)

	out := e.mustRun("invoke", "capture")

	got := lines(out)
	if len(got) != 6 || got[0] != "+++ capture · edict +++" || !strings.HasPrefix(got[1], "  ✔ verse 1 ") ||
		!strings.HasPrefix(got[2], "  ⋯ ") || !strings.HasSuffix(got[2], "(1/2)") ||
		!strings.HasPrefix(got[3], "  ✔ verse 2 ") || !strings.HasSuffix(got[4], "(2/2)") ||
		!strings.HasPrefix(got[5], "✠ ") || aspectless.MatchString(out) {
		t.Fatalf("stdout:\n%s", out)
	}
	if e.notified() != "" {
		t.Fatalf("the desktop heard %q", e.notified())
	}
	if out := e.mustRun("invoke", "capture", "-s"); strings.Contains(out, "+++") || strings.Contains(out, "✔") ||
		aspectless.MatchString(out) {
		t.Fatalf("a silenced edict told more than its vox-casts:\n%s", out)
	}
}

func TestEdictII_RefusesEveryWordAfterItsNameAndForeseesNothing(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", edictIIJSON)

	_, errOut, code := e.run("invoke", "capture", "on")
	if code != 2 || !strings.Contains(errOut, `the edict "capture" bears no aspect`) ||
		!strings.Contains(errOut, `yet "on" followed its name`) {
		t.Errorf("exit %d\nstderr: %s", code, errOut)
	}
	grimdark(t, errOut)

	out := e.mustRun("invoke", "capture", "-f", "--region", "window")
	got := lines(out)
	if len(got) < 5 || got[0] != "+++ Foreseen: the edict capture +++" ||
		got[1] != "verse 1 · incantation echo window >> "+homeward(e.target)+".edict" ||
		!strings.Contains(out, "verse 2 · incantation true") || got[len(got)-1] != "Nothing was performed." ||
		aspectless.MatchString(out) {
		t.Fatalf("foresight:\n%s", out)
	}
	if _, err := os.Stat(e.target + ".edict"); err == nil || e.chronicleLines() != nil || e.notified() != "" {
		t.Fatal("a refused or foreseen edict touched the machine, the chronicle or the desktop")
	}
}

func TestEdictII_AFallIsRevertedAndLamented(t *testing.T) {
	for _, tc := range []struct {
		name, reversion, verdict, ending, standing string
		code                                       int
	}{
		{"reverted", "echo undone >> $TARGET.edict", "reverted", "Every deed is undone", "proclaimed", 1},
		{"faltered", "exit 4", "faltered", "the edict lies corrupted", "corrupted", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv(t)
			e.addRite("rites/capture.json", `{"pattern": "Mark II", "edict": true, "liturgy": [
			  {"designation": "incantation", "words": "echo done >> $TARGET.edict", "reversion": "`+tc.reversion+`"},
			  {"designation": "incantation", "words": "exit 3"}]}`)

			_, errOut, code := e.run("invoke", "capture")

			if code != tc.code || !strings.Contains(errOut, `the edict "capture" fell at verse 2 · incantation exit 3`) ||
				!strings.Contains(errOut, tc.ending) || aspectless.MatchString(errOut) {
				t.Fatalf("exit %d\nstderr: %s", code, errOut)
			}
			chronicled := e.chronicleLines()
			if len(chronicled) != 1 || !strings.Contains(chronicled[0], `"edict":true`) ||
				!strings.Contains(chronicled[0], `"verdict":"`+tc.verdict+`"`) || strings.Contains(chronicled[0], `"aspect"`) {
				t.Fatalf("chronicle %q", chronicled)
			}
			if n := e.notified(); !strings.Contains(n, "-u critical") || aspectless.MatchString(n) {
				t.Fatalf("the desktop heard %q", n)
			}
			if out := e.mustRun("augury", "capture", "standing"); out != tc.standing+"\n" {
				t.Fatalf("standing %q, want %q", out, tc.standing)
			}
		})
	}
}

// TestEdictII_AuguryAndCensus: an edict of Mark II stands dormant until
// first proclaimed, then proclaimed; its augury refuses every sign of an
// aspect, and the census shows it with · for aspect and aspects.
func TestEdictII_AuguryAndCensus(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", edictIIJSON)
	if out := e.mustRun("census"); !regexp.MustCompile(`(?m)^capture\s+·\s+dormant\s+·\s+·\s+Capture the visage of the machine$`).MatchString(out) {
		t.Errorf("census of a dormant edict:\n%s", out)
	}
	e.mustRun("invoke", "capture", "-s", "--region", "window")
	if out := e.mustRun("census"); !regexp.MustCompile(`(?m)^capture\s+·\s+proclaimed\s+·\s+triumph \d{4}-\d\d-\d\d \d\d:\d\d\s+Capture the visage of the machine$`).MatchString(out) {
		t.Errorf("census of a proclaimed edict:\n%s", out)
	}
	if raw := e.mustRun("census", "--binharic"); !strings.Contains(raw, "\"rite\": \"capture\",\n    \"edict\": true,\n") ||
		!strings.Contains(raw, "\"aspects\": [],\n") {
		t.Errorf("census --binharic:\n%s", raw)
	}
	proclaimed := regexp.MustCompile(`^\{\n  "rite": "capture",\n  "edict": true,\n  "standing": "proclaimed",\n  "sanction": "",\n` +
		`  "auspex": "",\n  "inscriptions": \{\n    "region": "window"\n  \},\n` + regexp.QuoteMeta(edictCanon) +
		`  "last_rite": \{\n    "verdict": "triumph",\n    "at": "[0-9T:Z-]+"\n  \}\n\}\n$`)
	if out := e.mustRun("augury", "capture"); !proclaimed.MatchString(out) {
		t.Errorf("augury of a proclaimed edict:\n%s", out)
	}
	for _, args := range [][]string{{"augury", "capture", "aspect"}, {"augury", "capture", "former"},
		{"augury", "capture", "--is", "taken"}} {
		out, errOut, code := e.run(args...)
		if code != 2 || out != "" || !strings.Contains(errOut, `the edict "capture" bears no aspect`) {
			t.Errorf("servitor %v: exit %d\nstdout: %s\nstderr: %s", args, code, out, errOut)
		}
	}
}

// TestEdictII_ExaltedEntriesArePerformedByTheExaltedServitor: the exalted
// entry of an edict of Mark II is sanctioned by its master and then
// spoken by the exalted servitor alone, hearing the root's home; never
// sanctioned, it is refused and nothing is spoken.
func TestEdictII_ExaltedEntriesArePerformedByTheExaltedServitor(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", `{"pattern": "Mark II", "edict": true, "vox-cast": ["success"], "liturgy": [
	  {"designation": "incantation", "words": "echo common >> $TARGET.edict"},
	  {"designation": "incantation", "words": "echo \"exalted $HOME\" >> $TARGET.exalted", "exalted": true}]}`)
	a := e.raiseAltar("sudo")
	a.lendSpeech()

	_, errOut, code := e.run("invoke", "capture")
	if code == 0 || !strings.Contains(errOut, "sanction") || e.exaltedSaid() != "" || len(a.called()) != 0 {
		t.Fatalf("an unsanctioned edict: exit %d\nstderr: %s\nsaid %q", code, errOut, e.exaltedSaid())
	}
	if _, err := os.Stat(e.target + ".edict"); err == nil {
		t.Fatal("a refused edict spoke its common entry")
	}

	e.mustRun("sanction", "capture")
	out, errOut, code := e.run("invoke", "capture")
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	if got := e.exaltedSaid(); got != "exalted /root\n" {
		t.Errorf("the exalted entry said %q", got)
	}
	if got, _ := os.ReadFile(e.target + ".edict"); string(got) != "common\n" {
		t.Errorf("the common entry said %q", got)
	}
	if line := e.lastChronicled(); !strings.Contains(line, `"edict":true`) || !strings.Contains(line, `"intercessor":"sudo"`) {
		t.Errorf("chronicled %s", line)
	}
}
