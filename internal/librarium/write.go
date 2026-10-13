package librarium

import (
	"encoding/json/jsontext"
	"fmt"
	"strings"

	"github.com/tailscale/hujson"
)

// Marshal writes r as indented scripture of pattern Mark II, where every
// map is an object and an edict's words are written once (riteII). Keys
// follow the order of the codex (save "$schema", which is kept and written
// first), an edict is written with "edict": true and no "aspects", defaults
// that were never written stay unwritten, the alias "force" is written as
// "zeal", and comments are not preserved. An array or object of scalars
// shares one line when it fits in eighty columns, save scripture written as
// its lines. A rite of pattern Mark I is never written: it is converted.
func Marshal(r *Rite) ([]byte, error) {
	if r.markI {
		return nil, fmt.Errorf("the servitor writes no scripture of pattern %q; the rite %q is converted to "+
			"pattern %q first", PatternMarkI, r.Name, Pattern)
	}
	w := &writer{}
	w.riteII(r)
	return w.layout()
}

// writer gathers the tokens of a scripture until the first failure, and
// lays them out once whole.
type writer struct {
	root  *node
	stack []frame
	err   error
	// notes are lines of comment written before the value at each JSON
	// pointer; scripture bearing them is JSONC.
	notes map[string][]string
}

// valid reports whether text is scripture the codex can read: JSON, or
// JSONC when the writer bears notes.
func (w *writer) valid(text string) bool {
	if w.notes == nil {
		return jsontext.Value(text).IsValid()
	}
	v, err := hujson.Parse([]byte(text))
	if err != nil {
		return false
	}
	v.Standardize()
	return jsontext.Value(v.Pack()).IsValid()
}

func (w *writer) str(s string) { w.tok(jsontext.String(s)) }
func (w *writer) begin()       { w.tok(jsontext.BeginObject) }
func (w *writer) end()         { w.tok(jsontext.EndObject) }
func (w *writer) field(k, v string) {
	w.str(k)
	w.str(v)
}

// optional writes k: v unless v is empty.
func (w *writer) optional(k, v string) {
	if v != "" {
		w.field(k, v)
	}
}

// flag writes k: true when b holds.
func (w *writer) flag(k string, b bool) {
	if b {
		w.str(k)
		w.tok(jsontext.True)
	}
}

func (w *writer) strings(list []string) {
	w.tok(jsontext.BeginArray)
	for _, s := range list {
		w.str(s)
	}
	w.tok(jsontext.EndArray)
}

func (w *writer) scripture(s Scripture) {
	switch {
	case s.Null:
		w.tok(jsontext.Null)
	case s.IsTome():
		w.begin()
		w.field("tome", s.Tome)
		w.flag("illuminate", s.Illuminate)
		w.end()
	case strings.Contains(s.Text, "\n"):
		w.lines(strings.Split(s.Text, "\n"))
	default:
		w.str(s.Text)
	}
}

func (w *writer) anchor(a Anchor) {
	if a.Null {
		w.tok(jsontext.Null)
		return
	}
	w.str(a.Path)
}
