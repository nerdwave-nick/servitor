package lexicon

import (
	"math"
	"slices"
	"testing"
	"unicode/utf8"
)

func TestDetect(t *testing.T) {
	cases := []struct {
		args []string
		env  string
		want bool
	}{
		{nil, "", true},
		{[]string{"census", "--no-grimdark"}, "", false},
		{[]string{"--no-grimdark=false"}, "1", true},
		{[]string{"--no-grimdark=true"}, "", false},
		{[]string{"--no-grimdark", "--no-grimdark=false"}, "", true},
		{[]string{"--", "--no-grimdark"}, "", true},
		{nil, "true", false},
		{nil, "nonsense", true},
	}
	for _, c := range cases {
		t.Setenv(EnvNoGrimdark, c.env)
		if got := Detect(c.args); got != c.want {
			t.Errorf("Detect(%v) with env %q = %v, want %v", c.args, c.env, got, c.want)
		}
	}
}

func TestVocabularies(t *testing.T) {
	g, p := Get(true), Get(false)
	if g.Switch != "rite" || p.Switch != "switch" || g.Toggle() != p || p.Toggle() != g {
		t.Fatal("vocabulary mismatch")
	}
	if g.P("a", "b") != "a" || p.P("a", "b") != "b" {
		t.Fatal("P picks the wrong variant")
	}
	if Title("rite") != "Rite" || Title("") != "" {
		t.Fatal("Title")
	}
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
