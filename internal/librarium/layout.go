package librarium

import (
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// columns is the width a joined line of scripture may fill.
const columns = 80

// node is one value of the scripture being written, laid out once whole.
type node struct {
	scalar string   // the value written, when it is no array or object
	object bool     // an object rather than an array
	keys   []string // the names of an object's members, written
	kids   []*node
	lines  bool     // scripture written as its lines: one per line, never joined
	notes  []string // lines of comment written before it
}

// frame is an array or object still being written, and the name of the
// member whose value comes next.
type frame struct {
	n     *node
	ptr   string // the JSON pointer of n
	key   string
	raw   string // the name of the next member, unquoted
	keyed bool
}

// tok adds t to the scripture being written.
func (w *writer) tok(t jsontext.Token) {
	if w.err != nil {
		return
	}
	switch t.Kind() {
	case '{', '[':
		n := &node{object: t.Kind() == '{'}
		ptr := w.next()
		w.attach(n, "")
		w.stack = append(w.stack, frame{n: n, ptr: ptr})
	case '}', ']':
		w.stack = w.stack[:len(w.stack)-1]
	case '"':
		quoted, err := jsontext.AppendQuote(nil, t.String())
		if err != nil {
			w.err = err
			return
		}
		w.attach(&node{scalar: string(quoted)}, t.String())
	default:
		w.attach(&node{scalar: t.String()}, "")
	}
}

// next is the JSON pointer of the value written next; "" while the name of
// an object's member is awaited, and for the whole scripture.
func (w *writer) next() string {
	if len(w.stack) == 0 {
		return ""
	}
	f := w.stack[len(w.stack)-1]
	switch {
	case f.n.object && !f.keyed:
		return ""
	case f.n.object:
		return f.ptr + "/" + strings.NewReplacer("~", "~0", "/", "~1").Replace(f.raw)
	}
	return fmt.Sprintf("%s/%d", f.ptr, len(f.n.kids))
}

// attach places n as the next value of the array or object being written,
// or as the name (raw, unquoted) of an object's next member, or as the
// whole scripture; the notes kept for its place are written before it.
func (w *writer) attach(n *node, raw string) {
	if f := w.top(); f == nil || !f.n.object || f.keyed {
		n.notes = w.notes[w.next()]
	}
	if len(w.stack) == 0 {
		w.root = n
		return
	}
	f := &w.stack[len(w.stack)-1]
	switch {
	case f.n.object && !f.keyed:
		f.key, f.raw, f.keyed = n.scalar, raw, true
		return
	case f.n.object:
		f.n.keys = append(f.n.keys, f.key)
		f.keyed = false
	}
	f.n.kids = append(f.n.kids, n)
}

// top is the array or object being written; nil before the first.
func (w *writer) top() *frame {
	if len(w.stack) == 0 {
		return nil
	}
	return &w.stack[len(w.stack)-1]
}

// lines writes list as the lines of a scripture.
func (w *writer) lines(list []string) {
	w.strings(list)
	if w.err == nil {
		top := w.stack[len(w.stack)-1].n
		top.kids[len(top.kids)-1].lines = true
	}
}

// layout writes the scripture indented by two spaces per depth, joining
// on one line each array or object of scalars that fits in the columns.
func (w *writer) layout() ([]byte, error) {
	if w.err != nil {
		return nil, w.err
	}
	var b strings.Builder
	noted(&b, w.root, "")
	put(&b, w.root, 0, "", "")
	if !w.valid(b.String()) {
		return nil, errors.New("the servitor transcribed scripture the codex cannot read")
	}
	return []byte(b.String()), nil
}

// put writes n at depth after head — the indent and the member's name on
// its first line — and ends its last line in tail.
func put(b *strings.Builder, n *node, depth int, head, tail string) {
	if len(n.kids) == 0 {
		b.WriteString(head + n.written() + tail + "\n")
		return
	}
	if one, ok := n.joined(); ok && utf8.RuneCountInString(head+one+tail) <= columns {
		b.WriteString(head + one + tail + "\n")
		return
	}
	open, closing := n.brackets()
	b.WriteString(head + open + "\n")
	pad := strings.Repeat("  ", depth+1)
	for i, kid := range n.kids {
		h, t := pad, ","
		if n.object {
			h += n.keys[i] + ": "
		}
		if i == len(n.kids)-1 {
			t = ""
		}
		noted(b, kid, pad)
		put(b, kid, depth+1, h, t)
	}
	b.WriteString(strings.Repeat("  ", depth) + closing + tail + "\n")
}

// noted writes the notes of n as lines of comment, each after pad.
func noted(b *strings.Builder, n *node, pad string) {
	for _, l := range n.notes {
		b.WriteString(strings.TrimRight(pad+"// "+l, " ") + "\n")
	}
}

func (n *node) brackets() (string, string) {
	if n.object {
		return "{", "}"
	}
	return "[", "]"
}

// written is a scalar, or an empty array or object, as written.
func (n *node) written() string {
	if n.scalar != "" {
		return n.scalar
	}
	open, closing := n.brackets()
	return open + closing
}

// joined is n written on one line; it is not when n holds an array or
// object that is not empty, or the lines of a scripture.
func (n *node) joined() (string, bool) {
	if n.lines {
		return "", false
	}
	parts := make([]string, len(n.kids))
	for i, kid := range n.kids {
		if len(kid.kids) > 0 || kid.notes != nil {
			return "", false
		}
		parts[i] = kid.written()
		if n.object {
			parts[i] = n.keys[i] + ": " + parts[i]
		}
	}
	open, closing := n.brackets()
	return open + strings.Join(parts, ", ") + closing, true
}
