package tui

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/nerdwave-nick/servitor/internal/chronicle"
	"github.com/nerdwave-nick/servitor/internal/codex"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// Three invocations, oldest first: the first lies in the rotated chronicle,
// the other two in the current one.
var (
	elderMouse = chronicle.Entry{
		At: time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC), Rite: "mouse", Aspect: "on",
		Inscriptions: map[string]string{"reason": "gaming remnant", "mode": "hide"}, Verdict: invocation.Triumph,
	}
	fallenTheme = chronicle.Entry{
		At: time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC), Rite: "theme", Aspect: "porpl", Former: "default",
		Inscriptions: map[string]string{}, Verdict: invocation.Reverted,
		FellAt: &chronicle.Verse{Number: 3, Key: "incantation", Target: "source hooks"},
		Heresy: "it ended bearing the death-mark 1, a sign that its work was not done",
		Reversions: []chronicle.Reversion{
			{Verse: 3, Verdict: invocation.Triumph}, {Verse: 2, Verdict: invocation.Triumph},
		},
	}
	falteredMouse = chronicle.Entry{
		At: time.Date(2026, 10, 10, 11, 0, 0, 0, time.UTC), Rite: "mouse", Aspect: "off", Former: "on",
		Inscriptions: map[string]string{"mode": "show"}, Verdict: invocation.Faltered,
		FellAt: &chronicle.Verse{Number: 2, Key: "sanctum", Target: "/vessels/x.kdl"},
		Heresy: "the vessel is guarded against the servitor's hand",
		Reversions: []chronicle.Reversion{
			{Verse: 2, Verdict: invocation.Faltered}, {Verse: 1, Verdict: invocation.Triumph},
		},
	}
)

// seedChronicle writes entries as lines of the chronicle file at path.
func seedChronicle(t *testing.T, path string, entries ...chronicle.Entry) {
	t.Helper()
	var b strings.Builder
	for _, e := range entries {
		line, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(append(line, '\n'))
	}
	write(t, path, b.String())
}

// seeded is a harness whose chronicle holds the three invocations, the
// eldest in the rotated chronicle.
func seeded(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t)
	path := h.m.s.Orders().Chronicle
	seedChronicle(t, chronicle.Rotated(path), elderMouse)
	seedChronicle(t, path, fallenTheme, falteredMouse)
	return h
}

// when is the moment of e as the chronicle view shows it.
func when(e chronicle.Entry) string { return e.At.Local().Format("2006-01-02 15:04") }

var spaces = regexp.MustCompile(` {2,}`)

// inOrder fails unless every s appears on the screen, each after the one
// before; runs of spaces count as one, so columns may be aligned.
func (h *harness) inOrder(substrs ...string) {
	h.t.Helper()
	out, at := spaces.ReplaceAllString(h.screen(), " "), 0
	for _, s := range substrs {
		i := strings.Index(out[at:], s)
		if i < 0 {
			h.t.Fatalf("screen lacks %q after the preceding lines:\n%s", s, out)
		}
		at += i + len(s)
	}
}

// TestChronicle_NewestFirstPrefilteredToTheChosenRite: h opens the chronicle
// filtered to the chosen rite; casting the filter away shows every entry of
// both the current and the rotated chronicle, newest first.
func TestChronicle_NewestFirstPrefilteredToTheChosenRite(t *testing.T) {
	h := seeded(t)
	h.keys("h")
	h.mustShow("Chronicle (2)", "/ mouse")
	h.inOrder(when(falteredMouse)+" mouse on → off ✖ faltered",
		when(elderMouse)+" mouse dormant → on ✔ triumph")
	h.mustNotShow("porpl", "Rites (1)")

	h.keys("/", "esc")
	h.mustShow("Chronicle (3)")
	h.inOrder(when(falteredMouse)+" mouse on → off ✖ faltered",
		when(fallenTheme)+" theme default → porpl ✖ reverted",
		when(elderMouse)+" mouse dormant → on ✔ triumph")

	h.keys("esc")
	h.mustShow("Rites (1)")
	h.mustNotShow("Chronicle (")
}

// TestChronicle_SlashFilters: / filters by rite, aspect or verdict.
func TestChronicle_SlashFilters(t *testing.T) {
	h := seeded(t)
	h.keys("h", "/")
	for range len("mouse") {
		h.keys("backspace")
	}
	h.typeText("porpl")
	h.mustShow("Chronicle (1)")
	h.inOrder("theme default → porpl")
	h.mustNotShow("on → off")
	h.keys("enter")
	h.mustShow("Chronicle (1)", "/ porpl")

	h.keys("/", "esc", "/")
	h.typeText("TRIUMPH")
	h.keys("enter")
	h.mustShow("Chronicle (1)")
	h.inOrder("dormant → on ✔ triumph")

	h.keys("/", "esc", "/")
	h.typeText("xenos")
	h.mustShow("Chronicle (0)", "No entry of the chronicle answers the filter.")
}

// TestChronicle_EnterReadsTheWholeEntry: enter shows the chosen entry with
// its inscriptions, the verse it fell at, its heresy and every reversion;
// esc returns to the list, then to the overview.
func TestChronicle_EnterReadsTheWholeEntry(t *testing.T) {
	h := seeded(t)
	h.keys("h", "/", "esc", "j", "enter")
	h.mustShow("The rite theme, default → porpl: reverted", when(fallenTheme),
		"INSCRIPTIONS", "none inscribed",
		"It fell at verse 3 · incantation source hooks", "death-mark 1",
		"REVERSIONS", "✔ verse 3 is undone", "✔ verse 2 is undone",
		"Every deed is undone")
	h.keys("esc")
	h.mustShow("Chronicle (3)")

	h.keys("k", "enter")
	h.mustShow("The rite mouse, on → off: faltered", "mode=show",
		"It fell at verse 2 · sanctum /vessels/x.kdl", "guarded against the servitor's hand",
		"✖ verse 2 could not be undone", "✔ verse 1 is undone", "The reversion faltered")
	h.keys("esc", "G", "enter")
	h.mustShow("The rite mouse, dormant → on: triumph", "mode=hide", "reason=gaming remnant",
		"The rite was performed in triumph")
	h.mustNotShow("It fell at", "REVERSIONS")
	h.keys("esc", "esc")
	h.mustShow("Rites (1)")
}

// TestChronicle_SilentAndPlacedByTheRune: an empty chronicle says so; the
// cogitator reads the chronicle its runes place.
func TestChronicle_SilentAndPlacedByTheRune(t *testing.T) {
	h := newHarness(t)
	h.keys("h")
	h.mustShow("Chronicle (0)", "The chronicle is silent")
	h.keys("q")

	elsewhere := filepath.Join(t.TempDir(), "rites.jsonl")
	seedChronicle(t, elsewhere, fallenTheme)
	h.m = newModel(h.dir, librarium.Runes{Chronicle: elsewhere})
	h.send(tea.WindowSizeMsg{Width: 120, Height: 36})
	h.keys("h", "/", "esc")
	h.mustShow("Chronicle (1)", shortPath(elsewhere))
	h.inOrder("theme default → porpl")
}

// TestChronicle_FollowsTheChosenRite: the pre-filter is the rite chosen in
// the overview; a garbled chronicle is told, not hidden.
func TestChronicle_FollowsTheChosenRite(t *testing.T) {
	h := seeded(t)
	h.writeRite("theme", strings.ReplaceAll(markRite, `"ward": "cursor"`, `"ward": "theme"`))
	h.keys("r", "j", "h")
	h.mustShow("Chronicle (1)", "/ theme", "default → porpl")
	h.keys("esc")

	path := h.m.s.Orders().Chronicle
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	h.keys("h")
	h.mustShow("Chronicle (0)", "The chronicle cannot be read")
}

// TestCodex_TheCogitatorPassageNamesEveryKey: the codex's passage on the
// cogitator names every key of its catalogue.
func TestCodex_TheCogitatorPassageNamesEveryKey(t *testing.T) {
	p, ok := codex.Lookup("cogitator")
	if !ok {
		t.Fatal("the codex holds no cogitator")
	}
	lore := p.Body
	for _, b := range overviewBindings {
		for _, k := range strings.FieldsFunc(b.keys, func(r rune) bool { return r == ' ' || r == '/' }) {
			if k == "↑" || k == "↓" {
				continue // "the arrows"
			}
			if !regexp.MustCompile(`(^|[\s(])` + regexp.QuoteMeta(k) + `([\s,.;)]|$)`).MatchString(lore) {
				t.Errorf("the cogitator passage does not name the key %q (%s)", k, b.text)
			}
		}
	}
}
