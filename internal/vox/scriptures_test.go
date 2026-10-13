package vox

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// spoken matches what a template may illuminate into.
func spoken(tmpl string) *regexp.Regexp {
	re := regexp.QuoteMeta(tmpl)
	re = strings.ReplaceAll(re, regexp.QuoteMeta("{{aspect}}"), "«[^»]+»")
	re = strings.ReplaceAll(re, regexp.QuoteMeta("{{step.kind}}"), "(sanctum|transcription|tether|incantation|litany)")
	re = strings.ReplaceAll(re, regexp.QuoteMeta("{{rite.name}}"), `[\w.-]+`)
	re = strings.ReplaceAll(re, regexp.QuoteMeta("{{subject}}"), `[\w.-]+`)
	re = strings.ReplaceAll(re, regexp.QuoteMeta("{{intercessor}}"), "(sudo|run0|pkexec)")
	re = strings.ReplaceAll(re, regexp.QuoteMeta("{{sigil}}"), "[0-9a-f]{12}")
	return regexp.MustCompile("^" + re + "$")
}

func drawn(tid invocation.Tidings, words string) bool {
	for _, tmpl := range slices.Concat(templates[tid], edictTemplates[tid], subjectTemplates[tid]) {
		if spoken(tmpl).MatchString(words) {
			return true
		}
	}
	return false
}

// toldElsewhere are lines the README and the codex show that rituals other
// than the herald tell: the sanction standing of a foresight or of servitor
// sanction, and the tidings of a conversion.
var toldElsewhere = regexp.MustCompile(`^✠ The (sanction of the )?exalted (verses|subjects and observances) |` +
	`^⋯ the rite "[^"]+" is converted to pattern "Mark II"; `)

// TestScriptures_ShowTheVoxCastsAsTheyAreTold: every closing, progress and
// awaiting line the README and the codex show is one the servitor could
// tell: an awaiting of dominion, or of the master's proof.
func TestScriptures_ShowTheVoxCastsAsTheyAreTold(t *testing.T) {
	paths, _ := filepath.Glob("../codex/passages/*.txt")
	paths = append(paths, "../../README.md")
	shown := 0
	progress := regexp.MustCompile(`^⋯ (.+)  \(\d+/\d+\)$`)
	awaiting := regexp.MustCompile(`^⋯ (.+)$`)
	awaited := 0
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.Lines(string(data)) {
			line = strings.TrimSpace(line)
			if toldElsewhere.MatchString(line) {
				continue
			}
			if words, ok := strings.CutPrefix(line, "✠ "); ok {
				shown++
				if !drawn(invocation.Success, words) {
					t.Errorf("%s shows a triumph never told: %q", path, line)
				}
			}
			if m := progress.FindStringSubmatch(line); m != nil {
				shown++
				if !drawn(invocation.Progress, m[1]) {
					t.Errorf("%s shows a progress never told: %q", path, line)
				}
			} else if m := awaiting.FindStringSubmatch(line); m != nil {
				awaited++
				if !drawn(invocation.Dominion, m[1]) && !drawn(invocation.Exaltation, m[1]) {
					t.Errorf("%s shows an awaiting of dominion never told: %q", path, line)
				}
			}
		}
	}
	if shown < 4 {
		t.Fatalf("the scriptures show only %d vox-casts", shown)
	}
	if awaited < 2 {
		t.Fatalf("the scriptures show only %d awaitings of dominion", awaited)
	}
}
