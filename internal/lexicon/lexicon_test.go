package lexicon

import "testing"

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
	if Thought(-1) == "" || Thought(len(Thoughts)) != Thoughts[0] {
		t.Fatal("Thought indexing")
	}
}
