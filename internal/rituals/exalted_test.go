package rituals

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

func TestInvoke_PerformsARiteWithoutExaltedSubjects(t *testing.T) {
	fx := newFixture(t)
	fx.rite("plain", `{"pattern": "Mark II", "aspects": ["on"], "subjects": [
	  {"name": "common", "designation": "machine-spirit", "exalted": false, "enact": {"*": "echo common >> $DATA/said"}}]}`)
	res, err := fx.servitor().Invoke(context.Background(), Petition{Rite: "plain", Aspect: "on"})
	if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Triumph {
		t.Fatalf("%v %+v", err, res)
	}
}

// TestInquire_AnExaltedTomeHearsNoFormerAspect: the illuminated tome of an
// exalted subject is fixed at sanction, so it may not speak {{former}}:
// the former aspect is heard from the environment alone. The heresy is
// found within the tome, at its place.
func TestInquire_AnExaltedTomeHearsNoFormerAspect(t *testing.T) {
	fx := newFixture(t)
	tome := fx.write(fx.lib, "tomes/exalted.conf", "was {{former}}\n")
	fx.rite("tune", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "conf", "designation": "transcription", "vessel": "/etc/tune.conf", "exalted": true,
	   "omens": {"*": {"tome": "../tomes/exalted.conf", "illuminate": true}}}]}`)

	fs := inquire(t, fx.servitor(), true)

	expectFindings(t, fs, []sought{{"tune", librarium.Heresy,
		[]string{"within the illuminated tome", `"$SERVITOR_FORMER_ASPECT"`}, tome, "{{former}}"}})
	if len(fs) != 1 {
		t.Errorf("found %d, expected 1:\n%v", len(fs), fs)
	}
}

// TestInquire_ExaltedSpeechKnowsItsTongues: exalted speech is spoken only in
// the tongues it knows how to bid read no scripture of their own — bash,
// zsh, fish, sh and dash, by the name of the program each truly is. Any
// other tongue of an exalted observance, named by the observance, the rite
// or the settings, is heresy; a common observance may speak it.
func TestInquire_ExaltedSpeechKnowsItsTongues(t *testing.T) {
	fx := newFixture(t)
	bin := t.TempDir()
	for _, name := range []string{"python3", "zsh"} {
		fx.write(bin, name, "#!/bin/sh\n")
		if err := os.Chmod(filepath.Join(bin, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("python3", filepath.Join(bin, "shell")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	tongues := fx.rite("tongues", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "s", "designation": "machine-spirit", "enact": {"*": "true"},
	   "introit": ["one", "two", "three", "four", "five", "six"]}],
	  "observances": [
	  {"name": "one", "designation": "incantation", "words": "print(1)", "tongue": "python3", "exalted": true},
	  {"name": "two", "designation": "incantation", "words": "print(2)", "tongue": "python3"},
	  {"name": "three", "designation": "incantation", "words": "echo 3", "tongue": "zsh", "exalted": true},
	  {"name": "four", "designation": "incantation", "words": "echo 4", "tongue": "shell", "exalted": true},
	  {"name": "five", "designation": "litany", "tongue": "python3", "scroll": "tongues.json", "exalted": true},
	  {"name": "six", "designation": "incantation", "words": "echo 6", "exalted": true}]}`)
	riteTongue := fx.rite("rite-tongue", `{"pattern": "Mark II", "aspects": ["on"], "tongue": "python3",
	  "subjects": [{"name": "s", "designation": "machine-spirit", "enact": {"*": "true"}, "introit": ["one"]}],
	  "observances": [{"name": "one", "designation": "incantation", "words": "print(1)", "exalted": true}]}`)

	fs := inquire(t, fx.servitor(), true)

	expectFindings(t, fs, []sought{
		{"tongues", librarium.Heresy, []string{`the tongue python3 of the observance "one" is python3`,
			librarium.ExaltedTongueNames}, tongues, `"python3", "exalted"`},
		{"tongues", librarium.Heresy, []string{`the tongue shell of the observance "four" is python3`,
			librarium.ExaltedTongueNames}, tongues, `"shell"`},
		{"tongues", librarium.Heresy, []string{`the tongue python3 of the observance "five" is python3`,
			librarium.ExaltedTongueNames}, tongues, `"python3", "scroll"`},
		{"rite-tongue", librarium.Heresy, []string{`the tongue python3 of the observance "one" is python3`,
			librarium.ExaltedTongueNames}, riteTongue, `"python3"`},
	})
	if got := denounced(fs, "", librarium.Heresy); len(got) != 0 {
		t.Errorf("found beyond the rites: %v", got)
	}
	if n := len(denounced(fs, "tongues", librarium.Heresy, "is never spoken")); n != 3 {
		t.Errorf("found %d heresies of tongues in\n%v", n, fs)
	}
}
