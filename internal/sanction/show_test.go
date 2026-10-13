package sanction

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

func mustSigil(t *testing.T, u *Unit, aspect string) string {
	t.Helper()
	s, ok := u.SigilOf(aspect)
	if !ok {
		t.Fatalf("no plan of %q", aspect)
	}
	return s
}

// TestShow_ScriptureThatIsNoWords: scripture that is not words is told by
// its size and SHA-256, never written out.
func TestShow_ScriptureThatIsNoWords(t *testing.T) {
	blob := []byte{0, 1, 2, 0xff}
	u := &Unit{Pattern: Pattern, UID: 1000, Rite: "blob", Aspects: []Plan{{Aspect: "on", Subjects: []Verse{
		{Subject: "blob", Kind: librarium.KindTranscription, Target: "/etc/blob", Scripture: blob}}}}}
	shown := strings.Join(Show(u), "\n")
	if want := "4 bytes that are no words · sha256 " + digest(blob); !strings.Contains(shown, want) {
		t.Errorf("shown:\n%s\nwant a line holding %q", shown, want)
	}
	if strings.Contains(shown, "\x00") {
		t.Errorf("the scripture was written out:\n%q", shown)
	}
}

// TestDifference_ShowsWhatChanged: the difference of a unit from the one
// sanctioned shows the lines it lost and gained, with the sigils that
// changed, amid a little of what stayed; a unit unchanged differs in
// nothing.
func TestDifference_ShowsWhatChanged(t *testing.T) {
	before := buildGaming(t, gaming, gamingFiles())
	after := buildGaming(t, strings.Replace(gaming, `"logger war"`, `"logger wars"`, 1), gamingFiles())
	diff := Difference(before, after)
	var lost, gained []string
	for _, line := range diff {
		switch {
		case strings.HasPrefix(line, "- "):
			lost = append(lost, line)
		case strings.HasPrefix(line, "+ "):
			gained = append(gained, line)
		case !strings.HasPrefix(line, "  "):
			t.Errorf("a line of the difference is unmarked: %q", line)
		}
	}
	wantLost := []string{"- sigil " + before.Sigil(), "-   ✠ aspect \"on\" · sigil " + mustSigil(t, before, "on"),
		"-       words: logger war"}
	wantGained := []string{"+ sigil " + after.Sigil(), "+   ✠ aspect \"on\" · sigil " + mustSigil(t, after, "on"),
		"+       words: logger wars"}
	if !slices.Equal(lost, wantLost) || !slices.Equal(gained, wantGained) {
		t.Errorf("lost %q\ngained %q\nin\n%s", lost, gained, strings.Join(diff, "\n"))
	}
	if len(diff) >= len(Show(after)) {
		t.Errorf("the difference shows everything:\n%s", strings.Join(diff, "\n"))
	}
	if !slices.Contains(diff, "  ⋯") {
		t.Errorf("the difference marks no lines left out:\n%s", strings.Join(diff, "\n"))
	}
	if d := Difference(before, before); len(d) != 0 {
		t.Errorf("an unchanged unit differs:\n%s", strings.Join(d, "\n"))
	}
}

// TestDiffer_KeepsBothShowings: whatever two showings are compared, the
// lines kept and lost are the first, those kept and gained the second, and
// a line is never both lost and gained where it could be kept.
func TestDiffer_KeepsBothShowings(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 17))
	words := []string{"a", "b", "c", "d"}
	lines := func() []string {
		out := make([]string, r.IntN(12))
		for i := range out {
			out[i] = words[r.IntN(len(words))]
		}
		return out
	}
	for range 500 {
		a, b := lines(), lines()
		var gotA, gotB []string
		marked := differ(a, b)
		for _, l := range marked {
			how, line := l[:2], l[2:]
			if how != gained {
				gotA = append(gotA, line)
			}
			if how != lost {
				gotB = append(gotB, line)
			}
		}
		if !slices.Equal(gotA, a) || !slices.Equal(gotB, b) {
			t.Fatalf("differ(%q, %q) = %q", a, b, marked)
		}
		if same := slices.Equal(a, b); same != !slices.ContainsFunc(marked, func(l string) bool {
			return !strings.HasPrefix(l, kept)
		}) {
			t.Fatalf("differ(%q, %q) = %q", a, b, marked)
		}
	}
}
