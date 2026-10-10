// Package sanctum finds, renders and rewrites the sanctums of pattern Mark I
// within the scripture of a vessel.
//
// A sanctum with glyph "//" and ward "mouse" reads:
//
//	// +++ begin of sanctum mouse -- aspect|on +++
//	<scripture>
//	// +++ end of sanctum mouse +++
//
// A closing glyph, when there is one, follows the final "+++". Of the
// key|value pairs after "--" only aspect is read; every other pair is ignored.
package sanctum

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	beginWords = "+++ begin of sanctum "
	endWords   = "+++ end of sanctum "
	seal       = "+++"
	aspectKey  = "aspect"
)

// Marker names one sanctum within a vessel.
type Marker struct {
	Glyph        string
	ClosingGlyph string // "" when the glyph needs no closing
	Ward         string
}

// Sanctum is a sanctum found within a vessel's scripture.
type Sanctum struct {
	BeginLine int // line of the begin marker, counted from zero
	EndLine   int // line of the end marker, counted from zero
	Indent    string
	Aspect    string // the aspect recorded in the begin marker; "" when none
	Content   string
}

func (m Marker) beginRe() *regexp.Regexp {
	return regexp.MustCompile(`^(\s*)` + regexp.QuoteMeta(m.Glyph) + `\s*\+\+\+\s+begin of sanctum\s+` +
		regexp.QuoteMeta(m.Ward) + `(?:\s+--\s+(.*?))?\s+\+\+\+\s*` + regexp.QuoteMeta(m.ClosingGlyph) + `\s*$`)
}

func (m Marker) endRe() *regexp.Regexp {
	return regexp.MustCompile(`^\s*` + regexp.QuoteMeta(m.Glyph) + `\s*\+\+\+\s+end of sanctum\s+` +
		regexp.QuoteMeta(m.Ward) + `\s+\+\+\+\s*` + regexp.QuoteMeta(m.ClosingGlyph) + `\s*$`)
}

// IsMarkerLine reports whether line is a begin or end marker of m.
func (m Marker) IsMarkerLine(line string) bool {
	return m.beginRe().MatchString(line) || m.endRe().MatchString(line)
}

// Find locates the sanctum of m in text; found is false when there is none.
// A sanctum begun twice, never sealed, or sealed without beginning is
// denounced.
func Find(text string, m Marker) (s Sanctum, found bool, err error) {
	lines := strings.Split(text, "\n")
	beginRe, endRe := m.beginRe(), m.endRe()
	s.BeginLine, s.EndLine = -1, -1
	for i, line := range lines {
		if sm := beginRe.FindStringSubmatch(line); sm != nil {
			if s.BeginLine >= 0 {
				if s.EndLine < 0 {
					return Sanctum{}, false, fmt.Errorf("line %d: the sanctum %q is begun anew within itself, "+
						"though its first beginning on line %d was never sealed", i+1, m.Ward, s.BeginLine+1)
				}
				return Sanctum{}, false, fmt.Errorf("line %d: the sanctum %q is kept twice in one vessel "+
					"(first on line %d); the servitor cannot know which of them is true", i+1, m.Ward, s.BeginLine+1)
			}
			s.BeginLine, s.Indent, s.Aspect = i, sm[1], aspectOf(sm[2])
			continue
		}
		if endRe.MatchString(line) {
			if s.BeginLine < 0 || s.EndLine >= 0 {
				return Sanctum{}, false, fmt.Errorf("line %d: the sanctum %q is sealed where it was never begun",
					i+1, m.Ward)
			}
			s.EndLine = i
		}
	}
	if s.BeginLine < 0 {
		return Sanctum{}, false, nil
	}
	if s.EndLine < 0 {
		return Sanctum{}, false, fmt.Errorf("line %d: the sanctum %q is begun but never sealed by its end marker",
			s.BeginLine+1, m.Ward)
	}
	s.Content = strings.Join(lines[s.BeginLine+1:s.EndLine], "\n")
	return s, true, nil
}

// aspectOf reads the aspect from the key|value pairs of a begin marker,
// ignoring every other pair and every word that is no pair.
func aspectOf(pairs string) string {
	rest := strings.TrimSpace(pairs)
	for rest != "" {
		token := rest
		if end := strings.IndexAny(rest, " \t"); end >= 0 {
			token = rest[:end]
		}
		key, val, isPair := strings.Cut(token, "|")
		if isPair && strings.HasPrefix(val, `"`) {
			if quoted, err := strconv.QuotedPrefix(rest[len(key)+1:]); err == nil {
				val, _ = strconv.Unquote(quoted)
				token = rest[:len(key)+1+len(quoted)]
			}
		}
		if isPair && key == aspectKey {
			return val
		}
		rest = strings.TrimSpace(rest[len(token):])
	}
	return ""
}

// encodeAspect writes an aspect bare when it can be read back bare.
func encodeAspect(aspect string) string {
	if aspect == "" || strings.ContainsAny(aspect, " \t\"|") {
		return strconv.Quote(aspect)
	}
	return aspect
}

// Render returns the sanctum with its markers, without a final newline. One
// final newline of content is dropped; empty content keeps an empty sanctum.
func Render(m Marker, indent, aspect, content string) string {
	closing := ""
	if m.ClosingGlyph != "" {
		closing = " " + m.ClosingGlyph
	}
	parts := []string{indent + m.Glyph + " " + beginWords + m.Ward + " -- " + aspectKey + "|" + encodeAspect(aspect) +
		" " + seal + closing}
	if content = strings.TrimSuffix(content, "\n"); content != "" {
		parts = append(parts, content)
	}
	parts = append(parts, indent+m.Glyph+" "+endWords+m.Ward+" "+seal+closing)
	return strings.Join(parts, "\n")
}

// Upsert rewrites the sanctum of m in text, keeping its indent, or appends it
// after a blank line when the vessel holds none.
func Upsert(text string, m Marker, aspect, content string) (string, error) {
	s, found, err := Find(text, m)
	if err != nil {
		return "", err
	}
	if !found {
		if text != "" && !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		if text != "" {
			text += "\n"
		}
		return text + Render(m, "", aspect, content) + "\n", nil
	}
	lines := strings.Split(text, "\n")
	out := append([]string{}, lines[:s.BeginLine]...)
	out = append(out, Render(m, s.Indent, aspect, content))
	out = append(out, lines[s.EndLine+1:]...)
	return strings.Join(out, "\n"), nil
}
