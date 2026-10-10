package lexicon

import (
	"math"
	"slices"
	"testing"
	"unicode/utf8"
)

func TestThought_IndexWrapsAround(t *testing.T) {
	n := len(Thoughts)
	if Thought(-1) != Thoughts[n-1] || Thought(n) != Thoughts[0] || Thought(math.MinInt) == "" {
		t.Fatal("Thought indexing")
	}
}

func TestParseThoughts_SkipsBlankAndCommentLines(t *testing.T) {
	got := parseThoughts("# header\r\n\n  first  \r\n\n\n# group\nsecond\n\n")
	if !slices.Equal(got, []string{"first", "second"}) {
		t.Fatalf("got %q", got)
	}
}

func TestParseThoughts_PanicsWithoutThoughts(t *testing.T) {
	for _, in := range []string{"", "\n\n", "# only a comment\n"} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("parseThoughts(%q) should panic", in)
				}
			}()
			parseThoughts(in)
		}()
	}
}

// TestThoughts_EmbeddedFile guards the shipped thoughts: they must exist, be
// unique, and fit the cogitator's footer.
func TestThoughts_EmbeddedFile(t *testing.T) {
	if len(Thoughts) < 30 {
		t.Fatalf("only %d thoughts embedded", len(Thoughts))
	}
	seen := map[string]bool{}
	for _, th := range Thoughts {
		if utf8.RuneCountInString(th) > 80 {
			t.Errorf("thought longer than 80 characters: %q", th)
		}
		if seen[th] {
			t.Errorf("duplicate thought: %q", th)
		}
		seen[th] = true
	}
}
