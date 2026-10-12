package sanction

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// Show is the unit as its master beholds it before sanctioning it, one line
// after another: the rite, the master and the whole sigil; the environment
// its speech hears; then every plan under its own sigil, with the words of
// its spliced inscriptions and every exalted verse with every field of its
// kind. Scripture is shown as words, one line after another, when it is
// words; scripture that is not — or bears any sign a terminal would heed
// rather than show — is told by its size and SHA-256 alone, and so is a
// word of a single line quoted when it bears such a sign.
func Show(u *Unit) []string {
	s := &shower{}
	s.add(0, "rite %s · uid %d · pattern %s", strconv.Quote(u.Rite), u.UID, word(u.Pattern))
	s.add(0, "sigil %s", u.Sigil())
	keys := make([]string, len(u.Environment.Keys))
	for i, k := range u.Environment.Keys {
		keys[i] = word(k)
	}
	s.add(0, "environment %s: %s", word(u.Environment.Pattern), strings.Join(keys, ", "))
	for _, p := range u.Aspects {
		of := "aspect " + strconv.Quote(p.Aspect)
		if u.Edict {
			of = "the edict"
		}
		sigil, _ := u.SigilOf(p.Aspect)
		s.add(1, "✠ %s · sigil %s", of, sigil)
		for _, key := range sortedKeys(p.Inscriptions) {
			s.add(2, "inscription %s: %s", word(key), word(p.Inscriptions[key]))
		}
		for _, v := range p.Liturgy {
			s.verse(v)
		}
	}
	return s.lines
}

// shower gathers the lines of a unit shown.
type shower struct{ lines []string }

// add adds a line at depth, two spaces for each.
func (s *shower) add(depth int, format string, args ...any) {
	s.lines = append(s.lines, strings.Repeat("  ", depth)+fmt.Sprintf(format, args...))
}

// field adds the field name of a verse, holding a word.
func (s *shower) field(name, value string) {
	if strings.Contains(value, "\n") {
		s.scripture(name, []byte(value))
		return
	}
	s.add(3, "%s: %s", name, word(value))
}

// scripture adds the field name holding scripture, as words when it is
// words, else by its size and sigil.
func (s *shower) scripture(name string, b []byte) {
	switch {
	case len(b) == 0:
		s.add(3, "%s: (empty)", name)
	case !isWords(b):
		s.add(3, "%s: %d bytes that are no words · sha256 %s", name, len(b), digest(b))
	default:
		s.add(3, "%s:", name)
		text, closed := strings.CutSuffix(string(b), "\n")
		for _, line := range strings.Split(text, "\n") {
			s.add(4, "│ %s", line)
		}
		if !closed {
			s.add(4, "⌁ no newline ends it")
		}
	}
}

// verse adds the exalted verse v.
func (s *shower) verse(v Verse) {
	s.add(2, "verse %d · %s", v.Verse, v.Kind.Key())
	key := v.Kind.Key()
	switch v.Kind {
	case librarium.KindSanctum:
		s.field(key, v.Target)
		s.field("ward", v.Ward)
		s.field("glyph", v.Glyph)
		s.field("closing-glyph", v.ClosingGlyph)
		s.field("consecrate", strconv.FormatBool(v.Consecrate))
		s.scripture("scripture", v.Scripture)
	case librarium.KindTranscription:
		s.field(key, v.Target)
		if v.Null {
			s.add(3, "scripture: null — the vessel is struck")
		} else {
			s.scripture("scripture", v.Scripture)
		}
		seal := "(unwritten)"
		if v.Seal != nil {
			seal = fmt.Sprintf("%04o", uint32(*v.Seal))
		}
		s.field("seal", seal)
		s.field("zeal", strconv.FormatBool(v.Zeal))
	case librarium.KindTether:
		s.field(key, v.Target)
		if v.Null {
			s.add(3, "anchor: null — the bond is unbound")
		} else {
			s.field("anchor", v.Anchor)
		}
		s.field("zeal", strconv.FormatBool(v.Zeal))
	case librarium.KindIncantation, librarium.KindLitany:
		s.field(key, v.Target)
		if v.Kind == librarium.KindLitany {
			s.scripture("scroll", v.Scroll)
			offerings := make([]string, len(v.Offerings))
			for i, o := range v.Offerings {
				offerings[i] = strconv.Quote(o)
			}
			s.field("offerings", strings.Join(offerings, ", "))
		}
		s.field("tongue", v.Tongue)
		s.field("patience", v.Patience.String())
		if v.Reversion == "" {
			s.add(3, "reversion: (none)")
		} else {
			s.field("reversion", v.Reversion)
		}
		for _, r := range v.Relics {
			s.scripture("relic "+word(r.Relic), r.Scripture)
		}
	}
}

// isWords tells whether b is words a terminal shows as they are: UTF-8,
// every sign printable, a space, a tab or a newline.
func isWords(b []byte) bool {
	if !utf8.Valid(b) {
		return false
	}
	for _, r := range string(b) {
		if r != '\n' && r != '\t' && !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

// word is a word of a single line as it is shown: as it is when it is
// words, else quoted, as an empty word is.
func word(w string) string {
	if w != "" && isWords([]byte(w)) && !strings.Contains(w, "\n") {
		return w
	}
	return strconv.Quote(w)
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// The marks of a line of a difference.
const (
	kept   = "  "
	lost   = "- "
	gained = "+ "
)

// context is how many lines kept are shown around a change.
const context = 3

// Difference is how the unit after differs from before, as their master
// beholds it: the lines of Show that before lost ("- ") and after gained
// ("+ "), amid at most three lines kept ("  ") on either side; "  ⋯" marks
// the kept lines left out. Units that are one differ in nothing.
func Difference(before, after *Unit) []string {
	marked := differ(Show(before), Show(after))
	if !slices.ContainsFunc(marked, func(l string) bool { return !strings.HasPrefix(l, kept) }) {
		return nil
	}
	near := make([]bool, len(marked))
	for i, l := range marked {
		if strings.HasPrefix(l, kept) {
			continue
		}
		for j := max(0, i-context); j <= min(len(marked)-1, i+context); j++ {
			near[j] = true
		}
	}
	var out []string
	for i, l := range marked {
		switch {
		case near[i]:
			out = append(out, l)
		case len(out) == 0 || out[len(out)-1] != kept+"⋯":
			out = append(out, kept+"⋯")
		}
	}
	return out
}

// differCap is the most lines between their first and last difference two
// showings are compared line by line; beyond it, every line between is shown
// lost and gained.
const differCap = 2000

// differ marks every line of a and b: kept, lost from a or gained in b, in
// the shortest such script (Myers), after the lines both begin and end with.
func differ(a, b []string) []string {
	var head, tail []string
	for len(a) > 0 && len(b) > 0 && a[0] == b[0] {
		head = append(head, kept+a[0])
		a, b = a[1:], b[1:]
	}
	for len(a) > 0 && len(b) > 0 && a[len(a)-1] == b[len(b)-1] {
		tail = append([]string{kept + a[len(a)-1]}, tail...)
		a, b = a[:len(a)-1], b[:len(b)-1]
	}
	var middle []string
	if len(a)+len(b) > differCap {
		middle = mark(lost, a)
		middle = append(middle, mark(gained, b)...)
	} else {
		middle = myers(a, b)
	}
	return slices.Concat(head, middle, tail)
}

func mark(how string, lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = how + l
	}
	return out
}

// myers marks the lines of a and b by the shortest script of losses and
// gains that turns a into b.
func myers(a, b []string) []string {
	n, m := len(a), len(b)
	offset := n + m + 1
	v := make([]int, 2*offset+1)
	var trace [][]int
search:
	for d := 0; d <= n+m; d++ {
		trace = append(trace, slices.Clone(v))
		for k := -d; k <= d; k += 2 {
			x := v[offset+k-1] + 1
			if k == -d || (k != d && v[offset+k-1] < v[offset+k+1]) {
				x = v[offset+k+1]
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x, y = x+1, y+1
			}
			v[offset+k] = x
			if x >= n && y >= m {
				break search
			}
		}
	}
	var out []string
	x, y := n, m
	for d := len(trace) - 1; d >= 0; d-- {
		v := trace[d]
		k := x - y
		prev := k - 1
		if k == -d || (k != d && v[offset+k-1] < v[offset+k+1]) {
			prev = k + 1
		}
		px := v[offset+prev]
		py := px - prev
		for x > px && y > py {
			x, y = x-1, y-1
			out = append(out, kept+a[x])
		}
		if d > 0 {
			if x == px {
				out = append(out, gained+b[py])
			} else {
				out = append(out, lost+a[px])
			}
		}
		x, y = px, py
	}
	slices.Reverse(out)
	return out
}
