// Package block finds, renders and replaces servitor managed blocks in text.
//
// A managed block looks like this (comment prefix "//", guard "mouse"):
//
//	// begin servitor managed -- mouse -- state|on reason|"gaming remnant"
//	<managed content>
//	// end servitor managed -- mouse
package block

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	beginWord = "begin servitor managed -- "
	endWord   = "end servitor managed -- "
)

// Marker identifies one managed block inside a file.
type Marker struct {
	Comment    string // comment prefix, e.g. "//" or "#"
	CommentEnd string // optional comment suffix, e.g. "*/"
	Guard      string // block identifier, unique per file
}

// Block is a managed block located inside a text.
type Block struct {
	BeginLine int // 0-based line index of the begin marker
	EndLine   int // 0-based line index of the end marker
	Indent    string
	Meta      map[string]string
	Content   string
}

func (m Marker) suffix() string {
	if m.CommentEnd == "" {
		return ""
	}
	return " " + m.CommentEnd
}

func (m Marker) beginRe() *regexp.Regexp {
	return regexp.MustCompile(`^(\s*)` + regexp.QuoteMeta(m.Comment) + `\s*` +
		regexp.QuoteMeta(beginWord+m.Guard) + `(?:\s+--\s+(.*?))?\s*` +
		regexp.QuoteMeta(m.CommentEnd) + `\s*$`)
}

func (m Marker) endRe() *regexp.Regexp {
	return regexp.MustCompile(`^\s*` + regexp.QuoteMeta(m.Comment) + `\s*` +
		regexp.QuoteMeta(endWord+m.Guard) + `\s*` + regexp.QuoteMeta(m.CommentEnd) + `\s*$`)
}

// IsMarkerLine reports whether line is a begin or end marker for m.
func (m Marker) IsMarkerLine(line string) bool {
	return m.beginRe().MatchString(line) || m.endRe().MatchString(line)
}

// Find locates the block for m in text. found is false when no block exists.
// Malformed blocks (duplicate, unterminated, stray end marker) are errors.
func Find(text string, m Marker) (b Block, found bool, err error) {
	lines := strings.Split(text, "\n")
	beginRe, endRe := m.beginRe(), m.endRe()
	b.BeginLine, b.EndLine = -1, -1
	for i, line := range lines {
		if sm := beginRe.FindStringSubmatch(line); sm != nil {
			if b.BeginLine >= 0 {
				if b.EndLine < 0 {
					return Block{}, false, fmt.Errorf("line %d: nested begin marker for guard %q (previous begin on line %d has no end)", i+1, m.Guard, b.BeginLine+1)
				}
				return Block{}, false, fmt.Errorf("line %d: duplicate block for guard %q (first on line %d)", i+1, m.Guard, b.BeginLine+1)
			}
			meta, err := DecodeMeta(sm[2])
			if err != nil {
				return Block{}, false, fmt.Errorf("line %d: %w", i+1, err)
			}
			b.BeginLine, b.Indent, b.Meta = i, sm[1], meta
			continue
		}
		if endRe.MatchString(line) {
			if b.BeginLine < 0 || b.EndLine >= 0 {
				return Block{}, false, fmt.Errorf("line %d: end marker for guard %q without begin marker", i+1, m.Guard)
			}
			b.EndLine = i
		}
	}
	if b.BeginLine < 0 {
		return Block{}, false, nil
	}
	if b.EndLine < 0 {
		return Block{}, false, fmt.Errorf("line %d: begin marker for guard %q has no end marker", b.BeginLine+1, m.Guard)
	}
	b.Content = strings.Join(lines[b.BeginLine+1:b.EndLine], "\n")
	return b, true, nil
}

// Render returns the full block (markers and content) without a trailing newline.
func Render(m Marker, indent string, meta map[string]string, content string) (string, error) {
	enc := EncodeMeta(meta)
	if m.CommentEnd != "" && strings.Contains(enc, m.CommentEnd) {
		return "", fmt.Errorf("metadata %q would terminate the %q comment", enc, m.CommentEnd)
	}
	begin := indent + m.Comment + " " + beginWord + m.Guard
	if enc != "" {
		begin += " -- " + enc
	}
	parts := []string{begin + m.suffix()}
	content = strings.TrimSuffix(content, "\n")
	if content != "" {
		parts = append(parts, content)
	}
	parts = append(parts, indent+m.Comment+" "+endWord+m.Guard+m.suffix())
	return strings.Join(parts, "\n"), nil
}

// Upsert replaces the block for m in text, or appends it when absent.
func Upsert(text string, m Marker, meta map[string]string, content string) (string, error) {
	b, found, err := Find(text, m)
	if err != nil {
		return "", err
	}
	if !found {
		rendered, err := Render(m, "", meta, content)
		if err != nil {
			return "", err
		}
		if text != "" && !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		if text != "" {
			text += "\n"
		}
		return text + rendered + "\n", nil
	}
	rendered, err := Render(m, b.Indent, meta, content)
	if err != nil {
		return "", err
	}
	lines := strings.Split(text, "\n")
	out := append([]string{}, lines[:b.BeginLine]...)
	out = append(out, rendered)
	out = append(out, lines[b.EndLine+1:]...)
	return strings.Join(out, "\n"), nil
}

// Remove deletes the block for m from text. A blank separator line directly
// before the block is removed too when the block ends the file or is followed
// by another blank line. Text without the block is returned unchanged.
func Remove(text string, m Marker) (string, error) {
	b, found, err := Find(text, m)
	if err != nil || !found {
		return text, err
	}
	lines := strings.Split(text, "\n")
	start, end := b.BeginLine, b.EndLine+1
	tail := lines[end:]
	atEOF := len(tail) == 0 || (len(tail) == 1 && tail[0] == "")
	if start > 0 && strings.TrimSpace(lines[start-1]) == "" && (atEOF || strings.TrimSpace(tail[0]) == "") {
		start--
	}
	out := append(append([]string{}, lines[:start]...), tail...)
	return strings.Join(out, "\n"), nil
}
