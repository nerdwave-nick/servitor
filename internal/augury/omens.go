package augury

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
	"github.com/nerdwave-nick/servitor/internal/sanctum"
)

// seer reads the omens of one rite, reading every place of the machine at
// most once.
type seer struct {
	rite     *librarium.Rite
	slate    *Slate
	variants map[string][]placeholder.Values
	scrolls  map[string]scroll
	bonds    map[string]bond
}

// scroll is what stands at a vessel's place, its symbolic links followed.
type scroll struct {
	readable bool // false when the machine would not tell
	exists   bool
	content  string
}

// bond is what stands at a tether's name, which is never followed.
type bond struct {
	readable bool
	bound    bool   // a symbolic link stands there
	absent   bool   // nothing stands there
	anchor   string // where the link leads, resolved against its hall and cleaned
}

func newSeer(r *librarium.Rite, slate *Slate) *seer {
	return &seer{rite: r, slate: slate, variants: map[string][]placeholder.Values{},
		scrolls: map[string]scroll{}, bonds: map[string]bond{}}
}

// observe reads the omen of the step at index i of the liturgy, and its
// taint when it is a sanctum altered by other hands.
func (s *seer) observe(i int, step librarium.Step) (*Omen, *Taint) {
	switch st := step.(type) {
	case *librarium.Sanctum:
		return s.sanctum(i, st)
	case *librarium.Transcription:
		return s.transcription(i, st), nil
	case *librarium.Tether:
		return s.tether(i, st), nil
	}
	return nil, nil
}

// fitting tells whether the observation of a step fits aspect when its
// fields are rendered with v, and what its own key holds rendered so.
type fitting func(aspect string, v placeholder.Values) (target string, fits bool)

// sighting is the one aspect an observation fits, and how it was rendered.
type sighting struct {
	aspect, target string
	v              placeholder.Values
}

// sight returns the aspect the observation fits; ok is false when it fits
// none or several.
func (s *seer) sight(fits fitting) (seen sighting, ok bool) {
	var found []sighting
	for _, a := range s.rite.Aspects {
		for _, v := range s.valuesOf(a) {
			if target, ok := fits(a, v); ok {
				found = append(found, sighting{aspect: a, target: target, v: v})
				break
			}
		}
	}
	if len(found) != 1 {
		return sighting{}, false
	}
	return found[0], true
}

func (s *seer) omen(i int, k librarium.Kind, seen sighting) *Omen {
	return &Omen{Verse: invocation.Verse{Number: i + 1, Kind: k, Target: seen.target}, Aspect: seen.aspect}
}

// sanctum reads the aspect a sanctum's marker records and whether its
// scripture is still that of the aspect.
func (s *seer) sanctum(i int, st *librarium.Sanctum) (*Omen, *Taint) {
	glyph, closing := st.Glyphs()
	m := sanctum.Marker{Glyph: glyph, ClosingGlyph: closing, Ward: st.WardFor(s.rite.Name)}
	held := func(v placeholder.Values) (string, sanctum.Sanctum, bool) {
		target, ok := s.render(st.Vessel, v)
		if !ok {
			return "", sanctum.Sanctum{}, false
		}
		f := s.scroll(s.rite.ResolvePath(target))
		if !f.readable || !f.exists {
			return "", sanctum.Sanctum{}, false
		}
		found, has, err := sanctum.Find(f.content, m)
		return target, found, has && err == nil
	}
	seen, ok := s.sight(func(aspect string, v placeholder.Values) (string, bool) {
		target, found, ok := held(v)
		return target, ok && found.Aspect == aspect
	})
	if !ok {
		return nil, nil
	}
	omen := s.omen(i, st.Kind(), seen)
	_, found, _ := held(seen.v)
	sc, _ := st.Scripture.For(seen.aspect)
	judged := false
	for _, v := range s.valuesOf(seen.aspect) {
		text, ok := invocation.Illuminate(s.rite, sc, v)
		if !ok {
			continue
		}
		if strings.TrimSuffix(text, "\n") == found.Content {
			return omen, nil
		}
		judged = true
	}
	if !judged {
		return omen, nil
	}
	return omen, &Taint{Verse: omen.Verse}
}

// transcription reads which aspect's scripture a transcribed vessel holds;
// an absent vessel is that of a null scripture.
func (s *seer) transcription(i int, st *librarium.Transcription) *Omen {
	seen, ok := s.sight(func(aspect string, v placeholder.Values) (string, bool) {
		sc, has := st.Scripture.For(aspect)
		target, ok := s.render(st.Vessel, v)
		if !has || !ok {
			return "", false
		}
		f := s.scroll(s.rite.ResolvePath(target))
		switch {
		case !f.readable:
			return "", false
		case sc.Null:
			return target, !f.exists
		case !f.exists:
			return "", false
		}
		text, ok := invocation.Illuminate(s.rite, sc, v)
		return target, ok && strings.TrimSuffix(text, "\n") == strings.TrimSuffix(f.content, "\n")
	})
	if !ok {
		return nil
	}
	return s.omen(i, st.Kind(), seen)
}

// tether reads which aspect's anchor a tether's name leads to; an unbound
// name is that of a null anchor.
func (s *seer) tether(i int, st *librarium.Tether) *Omen {
	seen, ok := s.sight(func(aspect string, v placeholder.Values) (string, bool) {
		anchor, has := st.Anchor.For(aspect)
		target, ok := s.render(st.Name, v)
		if !has || !ok {
			return "", false
		}
		b := s.bond(s.rite.ResolvePath(target))
		switch {
		case !b.readable:
			return "", false
		case anchor.Null:
			return target, b.absent
		case !b.bound:
			return "", false
		}
		want, ok := s.render(anchor.Path, v)
		return target, ok && s.rite.ResolvePath(want) == b.anchor
	})
	if !ok {
		return nil
	}
	return s.omen(i, st.Kind(), seen)
}

// valuesOf returns every set of placeholder values an invocation into
// aspect may have rendered with: the data-slate's inscriptions when it
// recorded aspect, else the decrees of aspect, with any former aspect.
func (s *seer) valuesOf(aspect string) []placeholder.Values {
	if vs, ok := s.variants[aspect]; ok {
		return vs
	}
	var sets []map[string]string
	if s.slate != nil && s.slate.Aspect == aspect {
		sets = append(sets, s.slate.Inscriptions)
	}
	decreed, _ := invocation.ResolveInscriptions(s.rite, aspect, nil)
	sets = append(sets, decreed)
	formers := append([]string{""}, s.rite.Aspects...)
	var vs []placeholder.Values
	for _, ins := range sets {
		for _, former := range formers {
			vs = append(vs, placeholder.Values{Aspect: aspect, Former: former, Rite: s.rite.Name, Inscriptions: ins})
		}
	}
	s.variants[aspect] = vs
	return vs
}

func (s *seer) render(text string, v placeholder.Values) (string, bool) {
	out, err := s.rite.Scope().Render(text, v)
	return out, err == nil
}

func (s *seer) scroll(path string) scroll {
	if f, ok := s.scrolls[path]; ok {
		return f
	}
	data, err := os.ReadFile(path)
	f := scroll{readable: err == nil || errors.Is(err, fs.ErrNotExist), exists: err == nil, content: string(data)}
	s.scrolls[path] = f
	return f
}

func (s *seer) bond(name string) bond {
	if b, ok := s.bonds[name]; ok {
		return b
	}
	var b bond
	info, err := os.Lstat(name)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		b = bond{readable: true, absent: true}
	case err != nil:
	case info.Mode()&fs.ModeSymlink != 0:
		if anchor, err := os.Readlink(name); err == nil {
			if !filepath.IsAbs(anchor) {
				anchor = filepath.Join(filepath.Dir(name), anchor)
			}
			b = bond{readable: true, bound: true, anchor: filepath.Clean(anchor)}
		}
	default:
		b = bond{readable: true}
	}
	s.bonds[name] = b
	return b
}
