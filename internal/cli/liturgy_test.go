package cli

import (
	"os"
	"strings"
	"testing"
)

// TestRunes_NameTheirWordsInTheLiturgy: a rune that takes a word names that
// word in the liturgy, never by the type the machine stores it in.
func TestRunes_NameTheirWordsInTheLiturgy(t *testing.T) {
	e := newEnv(t)
	pages := map[string]string{
		"invoke":    e.mustRun("expound", "invoke"),
		"augury":    e.mustRun("expound", "augury"),
		"rite page": e.mustRun("invoke", "mouse-autohide-toggle", "--help"),
	}
	for name, page := range pages {
		if strings.Contains(page, " string ") {
			t.Errorf("%s names a rune's word by its type:\n%s", name, page)
		}
	}
	for _, want := range []string{"--librarium hall", "--chronicle scroll"} {
		if !strings.Contains(pages["invoke"], want) {
			t.Errorf("the invoke page lacks %q:\n%s", want, pages["invoke"])
		}
	}
	if !strings.Contains(pages["augury"], "--is aspect") {
		t.Errorf("the augury page lacks %q:\n%s", "--is aspect", pages["augury"])
	}
	if !strings.Contains(pages["rite page"], "--reason word") {
		t.Errorf("the rite page lacks %q:\n%s", "--reason word", pages["rite page"])
	}
}

// TestRituals_SpeakNoPlainWordsOfTheirOwn: the short words of the rituals
// themselves are grimdark.
func TestRituals_SpeakNoPlainWordsOfTheirOwn(t *testing.T) {
	e := newEnv(t)
	for _, args := range [][]string{{"expound"}, {"expound", "completion"}, {"__complete", ""}, {"__complete", "completion", ""}} {
		out := e.mustRun(args...)
		for _, plain := range []string{"config files", "your shell", "unknown command"} {
			if strings.Contains(out, plain) {
				t.Errorf("%v speaks %q:\n%s", args, plain, out)
			}
		}
	}
}

// TestStrayWords_AreUnknownRituals: a word no ritual expects is denounced as
// an unknown ritual, as at the root.
func TestStrayWords_AreUnknownRituals(t *testing.T) {
	e := newEnv(t)
	for _, ritual := range []string{"census", "cogitator"} {
		_, errOut, code := e.run(ritual, "extra")
		if code == 0 || !strings.Contains(errOut, `unknown ritual "extra"`) || strings.Contains(errOut, "command") {
			t.Errorf("%s extra: %d %q", ritual, code, errOut)
		}
	}
}

// TestInquisition_CountsInTheLiturgy: the verdict counts rites, heresies and
// impurities as words, not as riddles of brackets.
func TestInquisition_CountsInTheLiturgy(t *testing.T) {
	e := newEnv(t)
	out, errOut, code := e.run("inquisition")
	out += errOut
	if code != 0 || !strings.Contains(out, "examined 1 rite in") || !strings.Contains(out, "0 heresies, 0 impurities") {
		t.Fatalf("one clean rite: %d %q", code, out)
	}
	e.addRite("rites/heretic.json", `{"pattern": "Mark I", "aspects": []}`)
	out, errOut, code = e.run("inquisition")
	out += errOut
	if code != 1 || !strings.Contains(out, "examined 2 rites in") || !strings.Contains(out, "1 heresy,") {
		t.Fatalf("one heretic: %d %q", code, out)
	}
	if strings.Contains(out, "(y/ies)") || strings.Contains(out, "rite(s)") {
		t.Fatalf("the verdict speaks in brackets: %q", out)
	}
}

// TestInvoke_NamesAFormerStandingThatIsNotDormancy: a rite whose omens
// disagree is not dormant, and the summary does not call it so.
func TestInvoke_NamesAFormerStandingThatIsNotDormancy(t *testing.T) {
	e := newEnv(t)
	if out := e.mustRun("invoke", "mouse-autohide-toggle", "on", "--foresee"); !strings.Contains(out, "(dormant) → on") {
		t.Fatalf("a dormant rite: %q", out)
	}
	// Two vessels whose scriptures name different aspects: the omens
	// disagree, and the rite stands corrupted.
	dir := t.TempDir()
	first, second := dir+"/first", dir+"/second"
	for path, word := range map[string]string{first: "dawn", second: "dusk"} {
		if err := os.WriteFile(path, []byte(word), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	e.addRite("rites/twain.json", `{"pattern": "Mark I", "aspects": ["dawn", "dusk"], "liturgy": [
	  {"transcription": "`+first+`", "scripture": {"dawn": "dawn", "dusk": "dusk"}},
	  {"transcription": "`+second+`", "scripture": {"dawn": "dawn", "dusk": "dusk"}}]}`)
	out := e.mustRun("invoke", "twain", "dawn", "--foresee")
	if !strings.Contains(out, "(corrupted) → dawn") {
		t.Fatalf("a corrupted rite is not named so: %q", out)
	}
}
