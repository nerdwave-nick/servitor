package librarium

import (
	"encoding/json/jsontext"
	"fmt"
)

// riteII writes r as scripture of pattern Mark II: every map an object,
// even one keyed by Fallback alone, and an edict's words written once.
func (w *writer) riteII(r *Rite) {
	w.begin()
	w.optional(SchemaKey, r.Schema)
	w.field("pattern", Pattern)
	w.optional("purpose", r.Purpose)
	if r.Edict {
		w.flag("edict", true)
	} else {
		w.str("aspects")
		w.strings(r.Aspects)
	}
	if len(r.Inscriptions) > 0 {
		w.str("inscriptions")
		w.begin()
		for _, in := range r.Inscriptions {
			w.str(in.Key)
			w.begin()
			w.optional("purpose", in.Purpose)
			w.flag("mandatory", in.Mandatory)
			switch {
			case in.Decrees.IsZero():
			case r.Edict:
				w.str("decrees")
				w.plain(in.Decrees)
			default:
				w.str("decrees")
				writeObject(w, in.Decrees, w.str)
			}
			w.end()
		}
		w.end()
	}
	w.optional("tongue", r.Tongue)
	if r.VoxCasts != nil {
		w.str("vox-cast")
		w.strings(r.VoxCasts)
	}
	if r.Edict {
		w.str("liturgy")
		w.tok(jsontext.BeginArray)
		for _, s := range r.Liturgy {
			w.begin()
			w.speech(s)
			w.end()
		}
		w.tok(jsontext.EndArray)
	} else {
		w.str("subjects")
		w.tok(jsontext.BeginArray)
		for _, s := range r.Subjects {
			w.subject(s)
		}
		w.tok(jsontext.EndArray)
		if len(r.Observances) > 0 {
			w.str("observances")
			w.tok(jsontext.BeginArray)
			for _, o := range r.Observances {
				w.observance(o)
			}
			w.tok(jsontext.EndArray)
		}
	}
	w.end()
}

// writeObject writes a map of pattern Mark II: always an object.
func writeObject[T any](w *writer, m AspectMap[T], one func(T)) {
	w.begin()
	for _, e := range m.entries {
		w.str(e.Aspect)
		one(e.Value)
	}
	w.end()
}

// fail notes the first thing the writer cannot transcribe.
func (w *writer) fail(format string, args ...any) {
	if w.err == nil {
		w.err = fmt.Errorf(format, args...)
	}
}

// plain writes words written once for every aspect.
func (w *writer) plain(m AspectMap[string]) {
	if len(m.entries) != 1 || m.entries[0].Aspect != Fallback {
		w.fail("the servitor cannot transcribe words keyed by aspect where pattern %q writes them once", Pattern)
		w.str("")
		return
	}
	w.str(m.entries[0].Value)
}

func (w *writer) subject(s Subject) {
	w.begin()
	w.field("name", s.Name)
	if s.Matter == nil {
		w.fail("the servitor cannot transcribe the subject %q, which bears no designation", s.Name)
		w.end()
		return
	}
	w.field("designation", s.Designation())
	w.optional("purpose", s.Purpose)
	w.flag("exalted", s.Exalted())
	switch m := s.Matter.(type) {
	case *Sanctum:
		w.field("vessel", m.Vessel)
		w.optional("ward", m.Ward)
		w.optional("glyph", m.Glyph)
		w.optional("closing-glyph", m.ClosingGlyph)
		w.flag("consecrate", m.Consecrate)
		w.str("omens")
		writeObject(w, m.Scripture, w.scripture)
	case *Transcription:
		w.field("vessel", m.Vessel)
		w.str("omens")
		writeObject(w, m.Scripture, w.scripture)
		if m.Seal != nil {
			w.field("seal", fmt.Sprintf("%04o", uint32(*m.Seal)))
		}
		w.flag("zeal", m.Zeal)
	case *Tether:
		w.field("tether", m.Name)
		w.str("omens")
		writeObject(w, m.Anchor, w.anchor)
		w.flag("zeal", m.Zeal)
	case *MachineSpirit:
		w.machineSpirit(m)
	default:
		w.fail("the servitor cannot transcribe a subject of a designation the codex does not know")
	}
	w.references("introit", s.Introit)
	w.references("recessional", s.Recessional)
	w.end()
}

func (w *writer) machineSpirit(m *MachineSpirit) {
	if !m.Omens.IsZero() {
		w.str("omens")
		writeObject(w, m.Omens, w.str)
	}
	if a := m.Auspex; a != nil {
		w.str("auspex")
		w.begin()
		w.optional("scan", a.Scan)
		if !a.Trials.IsZero() {
			w.str("trials")
			writeObject(w, a.Trials, w.str)
		}
		if a.Patience != 0 {
			w.field("patience", a.Patience.String())
		}
		w.flag("exalted", a.Exalted)
		w.end()
	}
	w.str("enact")
	writeObject(w, m.Enact, w.str)
	w.optional("tongue", m.Tongue)
	if m.Patience != 0 {
		w.field("patience", m.Patience.String())
	}
	if m.Relics != nil {
		w.str("relics")
		w.strings(m.Relics)
	}
}

// references writes the observances a subject names under key, when
// written.
func (w *writer) references(key string, names []string) {
	if names != nil {
		w.str(key)
		w.strings(names)
	}
}

func (w *writer) observance(o Observance) {
	w.begin()
	w.field("name", o.Name)
	w.speech(o.Speech, func() {
		if o.To != nil {
			w.str("to")
			w.strings(o.To)
		}
		if o.From != nil {
			w.str("from")
			w.strings(o.From)
		}
	})
	w.end()
}

// speech writes the designation and words of an observance or an edict's
// entry; guards, when given, write what follows the designation.
func (w *writer) speech(s Step, guards ...func()) {
	var u Utterance
	switch s := s.(type) {
	case *Incantation:
		w.field("designation", DesignationIncantation)
		for _, g := range guards {
			g()
		}
		w.str("words")
		w.plain(s.Command)
		u = s.Utterance
	case *Litany:
		w.field("designation", DesignationLitany)
		for _, g := range guards {
			g()
		}
		w.str("scroll")
		w.plain(s.Scroll)
		if s.Offerings != nil {
			w.str("offerings")
			w.tok(jsontext.BeginArray)
			for _, o := range s.Offerings {
				w.plain(o)
			}
			w.tok(jsontext.EndArray)
		}
		u = s.Utterance
	default:
		w.fail("the servitor cannot transcribe into pattern %q speech that is neither an incantation nor a litany",
			Pattern)
		return
	}
	w.optional("tongue", u.Tongue)
	if u.Patience != 0 {
		w.field("patience", u.Patience.String())
	}
	if !u.Reversion.IsZero() {
		w.str("reversion")
		w.plain(u.Reversion)
	}
	w.flag("exalted", u.Exalted)
	if u.Relics != nil {
		w.str("relics")
		w.strings(u.Relics)
	}
}
