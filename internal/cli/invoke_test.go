package cli

import (
	"bytes"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestInvokeAndAugury_UserScenario(t *testing.T) {
	e := newEnv(t)
	out := e.mustRun("invoke", "mouse-autohide-toggle", "on", "--reason", "gaming remnant")
	for _, want := range []string{"+++ The rite mouse-autohide-toggle is performed: (dormant) → on +++",
		"verse 1", "sanctum", e.target, "The Omnissiah is pleased."} {
		if !strings.Contains(out, want) {
			t.Fatalf("summary lacks %q:\n%s", want, out)
		}
	}
	if !strings.Contains(e.targetContent(), "// +++ begin of sanctum mouse-autohide-toggle -- aspect|on +++\ncursor {") {
		t.Fatalf("target:\n%s", e.targetContent())
	}
	if n := strings.Count(e.notified(), "\n"); n != 1 || !strings.Contains(e.notified(), "mouse-autohide-toggle → on") {
		t.Fatalf("notify-send heard:\n%s", e.notified())
	}
	var a struct {
		Aspect       string            `json:"aspect"`
		Standing     string            `json:"standing"`
		Inscriptions map[string]string `json:"inscriptions"`
		LastRite     struct {
			Verdict string `json:"verdict"`
		} `json:"last_rite"`
	}
	if err := json.Unmarshal([]byte(e.mustRun("augury", "mouse-autohide-toggle")), &a); err != nil {
		t.Fatal(err)
	}
	if a.Aspect != "on" || a.Standing != "performed" || a.LastRite.Verdict != "triumph" ||
		a.Inscriptions["reason"] != "gaming remnant" || a.Inscriptions["example-key"] != "example-value" {
		t.Fatalf("augury %+v", a)
	}

	e.mustRun("invoke", "mouse-autohide-toggle", "off", "-s")
	cases := map[string]string{"aspect": "off\n", "standing": "performed\n", "reason": "\n", "example-key": "other-value\n",
		"desecrated": "false\n"}
	for key, want := range cases {
		if got := e.mustRun("augury", "mouse-autohide-toggle", key); got != want {
			t.Errorf("augury %s = %q, want %q", key, got, want)
		}
	}
	lines := e.chronicleLines()
	if len(lines) != 2 || !strings.Contains(lines[1], `"former":"on"`) || !strings.Contains(lines[1], `"verdict":"triumph"`) {
		t.Fatalf("chronicle:\n%s", strings.Join(lines, "\n"))
	}
}

func TestInvoke_Errors(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/broken.json", `{"pattern": "Mark I", "aspects": []}`)
	cases := []struct {
		args []string
		err  string
	}{
		{[]string{"invoke", "mouse-autohide-toggle", "maybe"}, `servitor ✠ the rite "mouse-autohide-toggle" knows no aspect "maybe"`},
		{[]string{"invoke", "mouse-autohide-toggle"}, "the rite demands exactly one aspect (on, off)"},
		{[]string{"invoke", "mouse-autohide", "on"}, "no rite named \"mouse-autohide\" is recorded in the Librarium"},
		{[]string{"invoke", "mouse-autohide", "on"}, "Perhaps you sought:\n\tmouse-autohide-toggle"},
		{[]string{"invoke", "broken", "on"}, `the rite "broken" is tainted by heresy`},
		{[]string{"invoke", "mouse-autohide-toggle", "on", "--example-key", ""}, `the inscription "example-key" is mandatory`},
		{[]string{"invoke", "mouse-autohide-toggle", "on", "--bogus", "x"}, "unknown rune: --bogus"},
	}
	for _, c := range cases {
		if _, errOut, code := e.run(c.args...); code != 1 || !strings.Contains(errOut, c.err) {
			t.Errorf("%v: code=%d stderr=%q", c.args, code, errOut)
		}
	}
	if e.targetContent() != "input {}\n" || e.chronicleLines() != nil || e.notified() != "" {
		t.Fatal("a refused invocation touched the machine, the chronicle or the desktop")
	}
}

func TestInvoke_HelpAndForesee(t *testing.T) {
	e := newEnv(t)
	help := e.mustRun("invoke", "--help")
	if !strings.Contains(help, "mouse-autohide-toggle Hide the cursor after inactivity") {
		t.Fatalf("invoke help lacks rite list:\n%s", help)
	}
	if got := e.mustRun("invoke"); got != help {
		t.Fatal("bare 'invoke' should print help")
	}
	sub := e.mustRun("invoke", "mouse-autohide-toggle", "--help")
	for _, want := range []string{"Aspects: on, off", e.target, "--reason string", "why the rite was invoked",
		"--example-key string", "mandatory", "-f, --foresee", "-s, --silence"} {
		if !strings.Contains(sub, want) {
			t.Errorf("invoke help missing %q:\n%s", want, sub)
		}
	}
	for _, flag := range []string{"--foresee", "-f"} {
		out := e.mustRun("invoke", "mouse-autohide-toggle", "on", flag)
		for _, want := range []string{"verse 1 · sanctum " + e.target, "+ cursor {", "verse 2 · vox-cast success",
			"Nothing was performed."} {
			if !strings.Contains(out, want) {
				t.Fatalf("%s output lacks %q:\n%s", flag, want, out)
			}
		}
	}
	if e.targetContent() != "input {}\n" || e.chronicleLines() != nil || e.notified() != "" {
		t.Fatal("foreseeing touched the machine, the chronicle or the desktop")
	}
}

func TestInvoke_Silence(t *testing.T) {
	e := newEnv(t)
	for _, flag := range []string{"--silence", "-s"} {
		if out := e.mustRun("invoke", "mouse-autohide-toggle", "on", flag); out != "" {
			t.Fatalf("%s should suppress the summary, got %q", flag, out)
		}
	}
}

func TestInvoke_TheTerminalHearsTheVoxInsteadOfTheDesktop(t *testing.T) {
	e := newEnv(t)
	var tty bytes.Buffer
	e.terminal(&tty)
	e.mustRun("invoke", "mouse-autohide-toggle", "on", "-s")
	if !strings.Contains(tty.String(), "+++ mouse-autohide-toggle → on +++") || e.notified() != "" {
		t.Fatalf("terminal %q, desktop %q", tty.String(), e.notified())
	}
}

const fallingJSON = `{
  "pattern": "Mark I",
  "aspects": ["on", "off"],
  "inscriptions": {"reason": {"decrees": {"*": "decreed"}}},
  "liturgy": [
    {"sanctum": "$TARGET", "ward": "falling", "scripture": "x"},
    {"incantation": "echo $SERVITOR_INSCRIPTION_REASON >> $TARGET.said; echo dying words; exit 3"}
  ]
}`

func TestInvoke_AFallIsRevertedAndTold(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/falling.json", fallingJSON)

	out, errOut, code := e.run("invoke", "falling", "on", "--reason", "")

	if code != 1 || out != "" {
		t.Fatalf("code=%d out=%q", code, out)
	}
	for _, want := range []string{`the rite "falling" fell at verse 2`, "death-mark 3", "dying words",
		"verse 1 · sanctum " + e.target + " is undone", "Every deed is undone"} {
		if !strings.Contains(errOut, want) {
			t.Errorf("the lament lacks %q:\n%s", want, errOut)
		}
	}
	if e.targetContent() != "input {}\n" {
		t.Fatalf("not reverted:\n%s", e.targetContent())
	}
	if said, _ := os.ReadFile(e.target + ".said"); string(said) != "\n" {
		t.Fatalf("a rune written empty must clear the decree, the incantation said %q", said)
	}
	if lines := e.chronicleLines(); len(lines) != 1 || !strings.Contains(lines[0], `"verdict":"reverted"`) {
		t.Fatalf("chronicle %q", lines)
	}
	if got := e.mustRun("augury", "falling", "standing"); got != "dormant\n" {
		t.Fatalf("standing %q", got)
	}
	if !strings.Contains(e.notified(), "-u critical") {
		t.Fatalf("the desktop was not told of the fall: %q", e.notified())
	}
}

func TestInvoke_AnInterruptHaltsAndRevertsTheRite(t *testing.T) {
	e := newEnv(t)
	marker := filepath.Join(e.state, "spoken")
	e.addRite("rites/long.json", `{"pattern": "Mark I", "aspects": ["on"], "liturgy": [
	  {"sanctum": "$TARGET", "ward": "long", "scripture": "x"},
	  {"incantation": "touch `+marker+`; sleep 30"}]}`)
	type result struct {
		errOut string
		code   int
	}
	done := make(chan result, 1)
	go func() {
		var out, errb bytes.Buffer
		code := Execute([]string{"-l", e.cfgDir, "invoke", "long", "on"}, &out, &errb)
		done <- result{errb.String(), code}
	}()
	deadline := time.Now().Add(10 * time.Second)
	for _, err := os.Stat(marker); err != nil; _, err = os.Stat(marker) {
		if time.Now().After(deadline) {
			t.Fatal("the incantation was never spoken")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	select {
	case r := <-done:
		if r.code != 1 || !strings.Contains(r.errOut, "halted") || !strings.Contains(r.errOut, "Every deed is undone") {
			t.Fatalf("code=%d stderr=%s", r.code, r.errOut)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the interrupt was not heeded")
	}
	if e.targetContent() != "input {}\n" {
		t.Fatalf("not reverted:\n%s", e.targetContent())
	}
	if lines := e.chronicleLines(); len(lines) != 1 || !strings.Contains(lines[0], `"verdict":"reverted"`) {
		t.Fatalf("chronicle %q", lines)
	}
}

func TestInvoke_TheChronicleRunePlacesTheChronicle(t *testing.T) {
	e := newEnv(t)
	p := filepath.Join(e.state, "elsewhere.jsonl")
	e.mustRun("--chronicle", p, "invoke", "mouse-autohide-toggle", "on", "-s")
	if b, err := os.ReadFile(p); err != nil || !strings.Contains(string(b), `"rite":"mouse-autohide-toggle"`) {
		t.Fatalf("chronicle at the rune: %q %v", b, err)
	}
	if e.chronicleLines() != nil {
		t.Fatal("the default chronicle was written too")
	}
}
