package cli

import (
	"encoding/json/v2"
	"os"
	"strings"
	"testing"
)

// falteringJSON is a rite whose recessional falls once its subject
// changed, and whose introit cannot be undone.
const falteringJSON = `{"pattern": "Mark II", "aspects": ["on"], "subjects": [
  {"name": "veil", "designation": "transcription", "vessel": "$TARGET.veil", "omens": {"*": "x"},
   "introit": ["kept"], "recessional": ["dying"]}], "observances": [
  {"name": "kept", "designation": "incantation", "words": "true", "reversion": "exit 4"},
  {"name": "dying", "designation": "incantation", "words": "exit 3"}]}`

// doubleHeresyJSON is a rite whose pre-flight finds two heresies: a scroll
// that is not there and a tongue the machine does not speak.
const doubleHeresyJSON = `{"pattern": "Mark II", "aspects": ["on"], "subjects": [
  {"name": "veil", "designation": "transcription", "vessel": "$TARGET.veil", "omens": {"*": "x"},
   "introit": ["scroll", "tongue"]}], "observances": [
  {"name": "scroll", "designation": "litany", "scroll": "no-such-scroll"},
  {"name": "tongue", "designation": "incantation", "words": "true", "tongue": "no-such-tongue-of-the-machine"}]}`

// lastChronicled is the chronicle's last line, as written.
func (e *env) lastChronicled() string {
	e.t.Helper()
	lines := e.chronicleLines()
	if len(lines) == 0 {
		e.t.Fatal("the chronicle holds no line")
	}
	return lines[len(lines)-1]
}

// TestInvoke_BinharicPrintsTheChronicledEntryAndItsVerdictIsTheExit: every
// verdict prints the very line the chronicle recorded, alone upon stdout,
// and exits 0, 1 or 3.
func TestInvoke_BinharicPrintsTheChronicledEntryAndItsVerdictIsTheExit(t *testing.T) {
	for _, tc := range []struct {
		name, rite, body string
		args             []string
		verdict          string
		code             int
	}{
		{"triumph", "mouse-autohide-toggle", "", []string{"on", "--reason", "gaming remnant"}, "triumph", 0},
		{"reverted", "falling", fallingJSON, []string{"on"}, "reverted", 1},
		{"faltered", "faltering", falteringJSON, []string{"on"}, "faltered", 3},
	} {
		for _, flag := range []string{"--binharic", "--json"} {
			t.Run(tc.name+flag, func(t *testing.T) {
				e := newEnv(t)
				if tc.body != "" {
					e.addRite("rites/"+tc.rite+".json", tc.body)
				}

				out, errOut, code := e.run(append(append([]string{"invoke", tc.rite}, tc.args...), flag)...)

				if code != tc.code {
					t.Fatalf("exit %d, want %d\nstdout: %s\nstderr: %s", code, tc.code, out, errOut)
				}
				if out != e.lastChronicled()+"\n" {
					t.Fatalf("stdout is not the chronicle's last line:\nstdout:    %q\nchronicle: %q", out, e.lastChronicled())
				}
				var entry map[string]any
				if err := json.Unmarshal([]byte(out), &entry); err != nil {
					t.Fatal(err)
				}
				if entry["rite"] != tc.rite || entry["aspect"] != "on" || entry["verdict"] != tc.verdict {
					t.Fatalf("entry %v", entry)
				}
				if tc.verdict != "triumph" && (entry["fell_at"] == nil || !strings.Contains(errOut, "fell at observance dying")) {
					t.Fatalf("the fall is neither recorded nor lamented: %v\nstderr: %s", entry, errOut)
				}
			})
		}
	}
}

// TestInvoke_WithoutBinharicTheVerdictIsTheExit: the exit tells the verdict
// with or without --binharic, and the narration is as before.
func TestInvoke_WithoutBinharicTheVerdictIsTheExit(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/faltering.json", falteringJSON)

	out, errOut, code := e.run("invoke", "faltering", "on")

	if code != 3 || out != "" || !strings.Contains(errOut, "The reversion faltered") {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
}

// TestInvoke_BinharicEdictBearsNoAspect: an edict's entry names no aspect
// and no former aspect, as its chronicle line does not.
func TestInvoke_BinharicEdictBearsNoAspect(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", edictJSON)

	out, errOut, code := e.run("invoke", "capture", "--binharic", "--region", "window")

	if code != 0 || out != e.lastChronicled()+"\n" {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s\nchronicle: %s", code, out, errOut, e.lastChronicled())
	}
	var entry map[string]any
	if err := json.Unmarshal([]byte(out), &entry); err != nil {
		t.Fatal(err)
	}
	_, aspect := entry["aspect"]
	_, former := entry["former"]
	if aspect || former || entry["edict"] != true || entry["verdict"] != "triumph" {
		t.Fatalf("entry %v", entry)
	}
}

// TestInvoke_BinharicForbiddenRecitesEveryHeresy: a pre-flight refusal
// prints the rite, its aspect and every heresy under "forbidden", exits 2
// and chronicles nothing.
func TestInvoke_BinharicForbiddenRecitesEveryHeresy(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/doubly.json", doubleHeresyJSON)

	out, errOut, code := e.run("invoke", "doubly", "on", "--binharic")

	if code != 2 || e.chronicleLines() != nil {
		t.Fatalf("exit %d, chronicle %q\nstdout: %s\nstderr: %s", code, e.chronicleLines(), out, errOut)
	}
	var refusal map[string]any
	if err := json.Unmarshal([]byte(out), &refusal); err != nil {
		t.Fatalf("%v: %q", err, out)
	}
	forbidden, _ := refusal["forbidden"].([]any)
	if len(refusal) != 3 || refusal["rite"] != "doubly" || refusal["aspect"] != "on" || len(forbidden) != 2 ||
		!strings.Contains(forbidden[0].(string), "no-such-scroll") ||
		!strings.Contains(forbidden[1].(string), "no-such-tongue-of-the-machine") {
		t.Fatalf("refusal %v", refusal)
	}
	if strings.Count(out, "\n") != 1 || !strings.Contains(errOut, "the pre-flight forbids") {
		t.Fatalf("stdout %q\nstderr: %s", out, errOut)
	}
	if out, _, code := e.run("invoke", "doubly", "on"); code != 2 || out != "" {
		t.Fatalf("without binharic: exit %d, stdout %q", code, out)
	}
}

// TestInvoke_BinharicForbiddenEdictNamesNoAspect: an edict's refusal names
// the edict and its heresies alone.
func TestInvoke_BinharicForbiddenEdictNamesNoAspect(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", `{"pattern": "Mark II", "edict": true,
	  "inscriptions": {"region": {"mandatory": true}},
	  "liturgy": [{"designation": "incantation", "words": "echo {{inscription.region}} >> $TARGET.edict"}]}`)

	out, errOut, code := e.run("invoke", "capture", "--json")

	var refusal map[string]any
	if err := json.Unmarshal([]byte(out), &refusal); err != nil || code != 2 {
		t.Fatalf("exit %d, %v\nstdout: %s\nstderr: %s", code, err, out, errOut)
	}
	if _, aspect := refusal["aspect"]; aspect || refusal["rite"] != "capture" || len(refusal["forbidden"].([]any)) != 1 {
		t.Fatalf("refusal %v", refusal)
	}
	if _, err := os.Stat(e.target + ".edict"); err == nil || e.chronicleLines() != nil {
		t.Fatal("a forbidden edict touched the machine or the chronicle")
	}
}

// TestInvoke_RefusalsBeforeThePreflightSpeakOnlyToStderr: an unknown rite
// or aspect, stray words, an unknown rune and foresight in binharic are
// refused with exit 2, and stdout stays empty.
func TestInvoke_RefusalsBeforeThePreflightSpeakOnlyToStderr(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", edictJSON)
	for _, tc := range []struct {
		args []string
		err  string
	}{
		{[]string{"invoke", "nonesuch", "on", "--binharic"}, `no rite named "nonesuch"`},
		{[]string{"invoke", "mouse-autohide-toggle", "maybe", "--binharic"}, `knows no aspect "maybe"`},
		{[]string{"invoke", "mouse-autohide-toggle", "--binharic"}, "the rite demands exactly one aspect"},
		{[]string{"invoke", "capture", "taken", "--binharic"}, `yet "taken" followed its name`},
		{[]string{"invoke", "mouse-autohide-toggle", "on", "--binharic", "--bogus"}, "unknown rune: --bogus"},
		{[]string{"invoke", "mouse-autohide-toggle", "on", "--binharic", "--foresee"}, "--foresee"},
		{[]string{"invoke", "capture", "-f", "--json"}, "--foresee"},
	} {
		out, errOut, code := e.run(tc.args...)
		if code != 2 || out != "" || !strings.Contains(errOut, tc.err) {
			t.Errorf("servitor %v: exit %d\nstdout: %q\nstderr: %s", tc.args, code, out, errOut)
		}
		grimdark(t, errOut)
	}
	if e.targetContent() != "input {}\n" || e.chronicleLines() != nil || e.notified() != "" {
		t.Fatal("a refused invocation touched the machine, the chronicle or the desktop")
	}
}

// TestInvoke_BinharicNarratesUponStderr: stdout holds the entry alone, with
// a terminal (the live report goes to stderr) and without one (the report
// told at the end goes to stderr); the desktop hears as it always did.
func TestInvoke_BinharicNarratesUponStderr(t *testing.T) {
	for _, tc := range []struct {
		name string
		cols int
	}{{"terminal", 80}, {"no terminal", -1}} {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv(t)
			e.terminal(tc.cols)

			out, errOut, code := e.run("invoke", "mouse-autohide-toggle", "on", "--binharic")

			if code != 0 || out != e.lastChronicled()+"\n" {
				t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
			}
			for _, want := range []string{"+++ mouse-autohide-toggle · (dormant) → on +++\n", "  ✔ subject cursor · sanctum ", "\n✠ "} {
				if !strings.Contains(errOut, want) {
					t.Errorf("stderr lacks %q:\n%s", want, errOut)
				}
			}
			if tc.cols < 0 && !strings.Contains(e.notified(), "mouse-autohide-toggle → on") {
				t.Errorf("the desktop was not told: %q", e.notified())
			}
		})
	}
}
