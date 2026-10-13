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
	rite  *librarium.Rite
	slate *Slate
	// target and inscriptions are those of the invocation reading the
	// rite: they render the omens of target first.
	target       string
	inscriptions map[string]string
	booted       *string // the boot of the machine, once read
	variants     map[string][]placeholder.Values
	scrolls      map[string]scroll
	bonds        map[string]bond
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

// fitting tells whether the observation of a subject fits aspect when its
// fields are rendered with v, and what its vessel or bound name holds
// rendered so.
type fitting func(aspect string, v placeholder.Values) (target string, fits bool)

// sighting is the one aspect an observation fits, and how it was rendered.
type sighting struct {
	aspect, target string
	v              placeholder.Values
}

// sightings returns every aspect the observation fits, in the order of the
// rite's aspects.
func (s *seer) sightings(fits fitting) []sighting {
	var found []sighting
	for _, a := range s.rite.Aspects {
		for _, v := range s.valuesOf(a) {
			if target, ok := fits(a, v); ok {
				found = append(found, sighting{aspect: a, target: target, v: v})
				break
			}
		}
	}
	return found
}

// sight returns the aspect the observation fits; ok is false when it fits
// none or several.
func (s *seer) sight(fits fitting) (seen sighting, ok bool) {
	return one(s.sightings(fits))
}

func one(found []sighting) (sighting, bool) {
	if len(found) != 1 {
		return sighting{}, false
	}
	return found[0], true
}

// sanctum reads the aspect the marker of a sanctum subject records, and
// whether its scripture is still that of the aspect; ok is false when no
// marker of its ward names one aspect. A tainted sanctum names its vessel,
// rendered for that aspect.
func (s *seer) sanctum(st *librarium.Sanctum) (aspect string, taint *string, ok bool) {
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
		return "", nil, false
	}
	_, found, _ := held(seen.v)
	sc, _ := st.Scripture.For(seen.aspect)
	judged := false
	for _, v := range s.valuesOf(seen.aspect) {
		text, ok := invocation.Illuminate(s.rite, sc, v)
		if !ok {
			continue
		}
		if strings.TrimSuffix(text, "\n") == found.Content {
			return seen.aspect, nil, true
		}
		judged = true
	}
	if !judged {
		return seen.aspect, nil, true
	}
	return seen.aspect, &seen.target, true
}

// transcribed returns every aspect whose scripture a transcribed vessel
// holds.
func (s *seer) transcribed(st *librarium.Transcription) []sighting {
	return s.sightings(func(aspect string, v placeholder.Values) (string, bool) {
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
}

// bound returns every aspect whose anchor a tether's name leads to.
func (s *seer) bound(st *librarium.Tether) []sighting {
	return s.sightings(func(aspect string, v placeholder.Values) (string, bool) {
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
}

// valuesOf returns every set of placeholder values an invocation into
// aspect may have rendered with: those of the invocation reading the rite
// into aspect, the data-slate's inscriptions when it recorded aspect,
// and the decrees of aspect, with any former aspect.
func (s *seer) valuesOf(aspect string) []placeholder.Values {
	if vs, ok := s.variants[aspect]; ok {
		return vs
	}
	var sets []map[string]string
	if s.inscriptions != nil && aspect == s.target {
		sets = append(sets, s.inscriptions)
	}
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
