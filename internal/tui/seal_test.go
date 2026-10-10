package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestHighlightJSON_ColorsEveryNameAndWordOfAJoinedLine(t *testing.T) {
	th := newTheme()
	for _, line := range []string{
		`      {"vox-cast": "success"},`,
		`    "decrees": {"on": "a \"quoted\" word", "*": null}`,
		`  "aspects": ["on", "off"],`,
		`        "a {",`,
	} {
		got := highlightJSON(th, line)
		if ansi.Strip(got) != line {
			t.Errorf("highlighting changed the words: %q became %q", line, ansi.Strip(got))
		}
		for _, name := range []string{`"vox-cast"`, `"decrees"`, `"on"`, `"*"`, `"aspects"`} {
			if strings.Contains(line, name+":") && !strings.Contains(got, th.accent.Render(name)) {
				t.Errorf("the name %s of %q is not lit: %q", name, line, got)
			}
		}
		for _, word := range []string{`"success"`, `"a \"quoted\" word"`, `"off"`, `"a {"`} {
			if strings.Contains(line, word) && !strings.Contains(line, word+":") && !strings.Contains(got, th.text.Render(word)) {
				t.Errorf("the word %s of %q is not lit: %q", word, line, got)
			}
		}
	}
}
