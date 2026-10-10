package librarium

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/tailscale/hujson"

	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// decoder walks a scripture's syntax tree, filling a rite and gathering
// every finding with its position.
type decoder struct {
	src      *source
	rite     *Rite
	findings Findings
	aspects  []string // valid aspects declared once, in written order
	scope    placeholder.Scope
}

// member is one key of an object and its value.
type member struct {
	name string
	key  *hujson.Value
	val  *hujson.Value
}

// fields are the known, unique members of an object in written order.
type fields []member

func (fs fields) get(name string) (member, bool) {
	for _, m := range fs {
		if m.name == name {
			return m, true
		}
	}
	return member{}, false
}

func (d *decoder) finding(sev Severity, off int, format string, args ...any) {
	d.findings = append(d.findings, Finding{
		Severity: sev, Scripture: d.src.path, Rite: d.rite.Name,
		Position: d.src.position(off), Message: fmt.Sprintf(format, args...),
	})
}

func (d *decoder) heresy(v *hujson.Value, format string, args ...any) {
	d.finding(Heresy, v.StartOffset, format, args...)
}

func (d *decoder) impurity(v *hujson.Value, format string, args ...any) {
	d.finding(Impurity, v.StartOffset, format, args...)
}

// object returns the members of v, denouncing keys outside known (nil allows
// any key), keys written twice, and a v that is no object.
func (d *decoder) object(v *hujson.Value, what string, known []string) (fields, bool) {
	obj, ok := v.Value.(*hujson.Object)
	if !ok {
		d.wrongForm(v, what, "an object of keys between { and }")
		return nil, false
	}
	var fs fields
	for i := range obj.Members {
		m := member{key: &obj.Members[i].Name, val: &obj.Members[i].Value}
		m.name = m.key.Value.(hujson.Literal).String()
		switch {
		case known != nil && !slices.Contains(known, m.name):
			d.heresy(m.key, "the key %q is not written in the codex for %s; strike it, or write one of "+
				"the keys the codex knows there: %s", m.name, what, quoteAll(withoutAliases(known)))
		case slices.ContainsFunc(fs, func(o member) bool { return o.name == m.name }):
			d.heresy(m.key, "the key %q is written twice in %s; the servitor will not choose between two "+
				"commandments of one name — keep only one", m.name, what)
		default:
			fs = append(fs, m)
		}
	}
	return fs, true
}

// wrongForm denounces a value written in a form its place does not permit.
func (d *decoder) wrongForm(v *hujson.Value, what, want string) {
	d.heresy(v, "%s must be written as %s, yet here stands %s", what, want, formOf(v))
}

func formOf(v *hujson.Value) string {
	switch v.Value.Kind() {
	case '"':
		return "a string"
	case '0':
		return "a number"
	case 't', 'f':
		return "true or false"
	case 'n':
		return "null"
	case '[':
		return "a list"
	}
	return "an object"
}

// withoutAliases drops alias keys, which no listing names.
func withoutAliases(keys []string) []string {
	return slices.DeleteFunc(slices.Clone(keys), func(k string) bool { _, alias := Aliases[k]; return alias })
}

func quoteAll(names []string) string {
	q := make([]string, len(names))
	for i, n := range names {
		q[i] = fmt.Sprintf("%q", n)
	}
	return strings.Join(q, ", ")
}

func literal(v *hujson.Value) (hujson.Literal, bool) {
	lit, ok := v.Value.(hujson.Literal)
	return lit, ok
}

// str decodes a string; the second result is false (and the heresy spoken)
// when v holds something else.
func (d *decoder) str(v *hujson.Value, what string) (string, bool) {
	if lit, ok := literal(v); ok && lit.Kind() == '"' {
		return lit.String(), true
	}
	d.wrongForm(v, what, "a string")
	return "", false
}

// word decodes a string that must not be empty.
func (d *decoder) word(v *hujson.Value, what string) (string, bool) {
	s, ok := d.str(v, what)
	if ok && strings.TrimSpace(s) == "" {
		d.heresy(v, "%s may not be empty, for the servitor cannot perform what names nothing", what)
		return s, false
	}
	return s, ok
}

func (d *decoder) boolean(v *hujson.Value, what string) bool {
	if lit, ok := literal(v); ok && (lit.Kind() == 't' || lit.Kind() == 'f') {
		return lit.Bool()
	}
	d.wrongForm(v, what, "true or false")
	return false
}

func (d *decoder) patience(v *hujson.Value, what string) time.Duration {
	s, ok := d.str(v, what)
	if !ok {
		return 0
	}
	p, err := time.ParseDuration(s)
	if err != nil || p <= 0 {
		d.heresy(v, "%q is no measure of patience the servitor understands; %s must be a span greater "+
			"than nothing, written such as \"500ms\", \"5s\" or \"2m\"", s, what)
		return 0
	}
	return p
}

// marks examines the placeholders of the string literal v, whose decoded
// text is text.
func (d *decoder) marks(v *hujson.Value, text string) {
	d.lineMarks([]*hujson.Value{v}, []string{text})
}

// lineMarks examines the placeholders of string literals vs, decoded as
// texts, which are illuminated together as lines joined with "\n".
func (d *decoder) lineMarks(vs []*hujson.Value, texts []string) {
	for _, h := range d.scope.Check(strings.Join(texts, "\n")) {
		i, off := 0, h.Pos.Offset
		for i < len(texts)-1 && off > len(texts[i]) {
			off -= len(texts[i]) + 1
			i++
		}
		raw := d.src.data[vs[i].StartOffset:vs[i].EndOffset]
		d.finding(Heresy, vs[i].StartOffset+rawOffset(raw, off), "%s", h.Message())
	}
}

// templated decodes a string whose placeholders are examined; unless
// mayBeEmpty it must not be empty.
func (d *decoder) templated(v *hujson.Value, what string, mayBeEmpty bool) (string, bool) {
	read := d.word
	if mayBeEmpty {
		read = d.str
	}
	s, ok := read(v, what)
	if ok {
		d.marks(v, s)
	}
	return s, ok
}
