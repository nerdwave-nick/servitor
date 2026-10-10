package librarium

import (
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"strings"
)

// Marshal writes r as indented Mark I scripture. Keys follow the order of
// the codex (save "$schema", which is kept and written first), defaults that
// were never written stay unwritten, the alias "force" is written as "zeal",
// and comments are not preserved. An array or object of scalars shares one
// line when it fits in eighty columns, save scripture written as its lines.
func Marshal(r *Rite) ([]byte, error) {
	w := &writer{}
	w.rite(r)
	return w.layout()
}

// writer gathers the tokens of a scripture until the first failure, and
// lays them out once whole.
type writer struct {
	root  *node
	stack []frame
	err   error
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

// writeMap writes an aspect map as one value or as an object by aspect.
func writeMap[T any](w *writer, m AspectMap[T], one func(T)) {
	if m.uniform {
		one(m.entries[0].Value)
		return
	}
	w.begin()
	for _, e := range m.entries {
		w.str(e.Aspect)
		one(e.Value)
	}
	w.end()
}

func (w *writer) rite(r *Rite) {
	w.begin()
	w.optional(SchemaKey, r.Schema)
	w.field("pattern", Pattern)
	w.optional("purpose", r.Purpose)
	w.str("aspects")
	w.strings(r.Aspects)
	if len(r.Inscriptions) > 0 {
		w.str("inscriptions")
		w.begin()
		for _, in := range r.Inscriptions {
			w.str(in.Key)
			w.begin()
			w.optional("purpose", in.Purpose)
			w.flag("mandatory", in.Mandatory)
			if !in.Decrees.IsZero() {
				w.str("decrees")
				writeMap(w, in.Decrees, w.str)
			}
			w.end()
		}
		w.end()
	}
	if a := r.Auspex; a != nil {
		w.str("auspex")
		if a.Patience == 0 {
			w.str(a.Rite)
		} else {
			w.begin()
			w.field("rite", a.Rite)
			w.field("patience", a.Patience.String())
			w.end()
		}
	}
	w.optional("tongue", r.Tongue)
	w.str("liturgy")
	w.tok(jsontext.BeginArray)
	for _, s := range r.Liturgy {
		w.step(s)
	}
	w.tok(jsontext.EndArray)
	w.end()
}

func (w *writer) step(s Step) {
	w.begin()
	switch s := s.(type) {
	case *Sanctum:
		w.field("sanctum", s.Vessel)
		w.optional("ward", s.Ward)
		w.optional("glyph", s.Glyph)
		w.optional("closing-glyph", s.ClosingGlyph)
		w.flag("consecrate", s.Consecrate)
		w.str("scripture")
		writeMap(w, s.Scripture, w.scripture)
	case *Transcription:
		w.field("transcription", s.Vessel)
		w.str("scripture")
		writeMap(w, s.Scripture, w.scripture)
		if s.Seal != nil {
			w.field("seal", fmt.Sprintf("%04o", uint32(*s.Seal)))
		}
		w.flag("zeal", s.Zeal)
	case *Tether:
		w.field("tether", s.Name)
		w.str("anchor")
		writeMap(w, s.Anchor, w.anchor)
		w.flag("zeal", s.Zeal)
	case *Incantation:
		w.str("incantation")
		writeMap(w, s.Command, w.str)
		w.utterance(s.Utterance)
	case *Litany:
		w.str("litany")
		writeMap(w, s.Scroll, w.str)
		if s.Offerings != nil {
			w.str("offerings")
			w.tok(jsontext.BeginArray)
			for _, o := range s.Offerings {
				writeMap(w, o, w.str)
			}
			w.tok(jsontext.EndArray)
		}
		w.utterance(s.Utterance)
	case *VoxCast:
		w.field("vox-cast", s.Tidings)
	default:
		if w.err == nil {
			w.err = errors.New("the servitor cannot transcribe a step of a kind the codex does not know")
		}
	}
	w.end()
}

func (w *writer) utterance(u Utterance) {
	w.optional("tongue", u.Tongue)
	if u.Patience != 0 {
		w.field("patience", u.Patience.String())
	}
	if !u.Reversion.IsZero() {
		w.str("reversion")
		writeMap(w, u.Reversion, w.str)
	}
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
