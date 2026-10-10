package cli

import (
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
	"github.com/nerdwave-nick/servitor/internal/rituals"
	"github.com/nerdwave-nick/servitor/internal/vox"
)

// controllingTerminal tells whether the servitor has a controlling terminal
// and how many columns it holds; replaced in tests.
var controllingTerminal = vox.ControllingTerminal

// columns is how wide a verse line may grow when the terminal does not say.
const columns = 80

// narration tells one invocation upon the servitor's stdout, in one report:
// a header, a line for every verse performed, the vox-casts the rite asks
// for, and one closing line of triumph.
//
// With a controlling terminal the report is told as the invocation is
// performed and the desktop hears nothing. Without one the vox-casts are
// sent to the desktop, and the report is told once the rite has triumphed.
// Silence withholds all but the vox-casts told upon a terminal. A fall is
// told by the lament alone.
type narration struct {
	w       io.Writer
	live    bool // told as it happens, upon a terminal
	silence bool
	width   int
	pick    func(n int) int
	desktop *vox.Herald // nil when live
	res     rituals.Result
	verses  []invocation.Verse
	closing string // the triumph a success vox-cast proclaimed
}

func newNarration(w io.Writer, voxOrder string, silence bool) *narration {
	cols, live := controllingTerminal()
	n := &narration{w: w, live: live, silence: silence, width: columns, pick: rand.IntN}
	if cols > 0 {
		n.width = cols
	}
	if !live {
		n.desktop = vox.New(vox.Config{Vox: voxOrder, Pick: n.pick})
	}
	return n
}

// telling is whether the report is told as it happens.
func (n *narration) telling() bool { return n.live && !n.silence }

// commence is told once the liturgy is about to be performed.
func (n *narration) commence(res rituals.Result) {
	n.res = res
	if n.telling() {
		n.header()
	}
}

// Performed satisfies invocation.Witness.
func (n *narration) Performed(v invocation.Verse, _, _ int) {
	n.verses = append(n.verses, v)
	if n.telling() {
		n.verse(v)
	}
}

// Proclaim satisfies invocation.Herald. Upon a terminal a progress is told
// with the count of real steps and the first success closes the report;
// the fall is left to the lament.
func (n *narration) Proclaim(p invocation.Proclamation) {
	m := vox.Compose(p, n.pick)
	closed := n.closing != ""
	if p.Tidings == invocation.Success && !closed {
		n.closing = m.Flavour
	}
	if !n.live {
		n.desktop.Send(m)
		return
	}
	switch p.Tidings {
	case invocation.Progress:
		fmt.Fprintf(n.w, "  ⋯ %s  (%d/%d)\n", m.Flavour, p.Values.Tidings.Step, p.Values.Tidings.Steps)
	case invocation.Success:
		if !closed {
			fmt.Fprintln(n.w, "✠ "+m.Flavour)
		}
	}
}

// end lets the desktop rest and closes the report of a triumph: without a
// terminal it is told whole, upon one only the closing line a success
// vox-cast did not already tell.
func (n *narration) end(out *invocation.Outcome) {
	if n.desktop != nil {
		_ = n.desktop.Close()
	}
	if out == nil || out.Verdict != invocation.Triumph || n.silence {
		return
	}
	if n.live && n.closing != "" {
		return
	}
	if !n.live {
		n.header()
		for _, v := range n.verses {
			n.verse(v)
		}
	}
	if n.closing == "" {
		opts := n.res.Options
		n.closing = vox.Compose(invocation.Proclamation{Tidings: invocation.Success,
			Values: placeholder.Values{Rite: n.res.Rite.Name, Aspect: opts.Aspect, Former: opts.Former}}, n.pick).Flavour
	}
	fmt.Fprintln(n.w, "✠ "+n.closing)
}

func (n *narration) header() {
	fmt.Fprintf(n.w, "+++ %s · %s → %s +++\n", n.res.Rite.Name, former(n.res), n.res.Options.Aspect)
}

// verse tells v on one line, its target cut short to fit the width.
func (n *narration) verse(v invocation.Verse) {
	v.Target = strings.Join(strings.Fields(v.Target), " ")
	fmt.Fprintln(n.w, fit("  ✔ "+verseName(v), n.width))
}

// fit cuts line to width characters, ending it in "…" when it was longer.
func fit(line string, width int) string {
	if utf8.RuneCountInString(line) <= width {
		return line
	}
	r := []rune(line)
	return string(r[:max(0, width-1)]) + "…"
}

// homeward speaks a path within the home of the faithful from "~".
func homeward(path string) string {
	home, err := os.UserHomeDir()
	home = strings.TrimRight(home, "/")
	switch {
	case err != nil || home == "":
		return path
	case path == home:
		return "~"
	case strings.HasPrefix(path, home+"/"):
		return "~" + path[len(home):]
	}
	return path
}
