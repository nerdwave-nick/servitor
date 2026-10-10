package librarium

import (
	"regexp"
	"strconv"
	"unicode/utf8"

	"github.com/tailscale/hujson"
)

// source is a scripture as read, for turning byte offsets into positions.
type source struct {
	path string
	data []byte
	ast  hujson.Value
}

// position converts a byte offset of the scripture into line and column.
func (s *source) position(off int) Position {
	off = max(0, min(off, len(s.data)))
	line, col := 1, 1
	for _, c := range s.data[:off] {
		if c == '\n' {
			line, col = line+1, 1
		} else {
			col++
		}
	}
	return Position{Line: line, Column: col}
}

// locate returns the position of the value at the JSON pointer ptr, or of
// its closest existing parent.
func (s *source) locate(ptr string) Position {
	for {
		if v := s.ast.Find(ptr); v != nil {
			return s.position(v.StartOffset)
		}
		if ptr == "" {
			return Position{Line: 1, Column: 1}
		}
		ptr = parentPointer(ptr)
	}
}

func parentPointer(ptr string) string {
	for i := len(ptr) - 1; i >= 0; i-- {
		if ptr[i] == '/' {
			return ptr[:i]
		}
	}
	return ""
}

var syntaxPosRe = regexp.MustCompile(`^hujson: line (\d+), column (\d+): `)

// syntaxPosition extracts the position from a hujson parse error.
func syntaxPosition(err error) Position {
	m := syntaxPosRe.FindStringSubmatch(err.Error())
	if m == nil {
		return Position{Line: 1, Column: 1}
	}
	line, _ := strconv.Atoi(m[1])
	col, _ := strconv.Atoi(m[2])
	return Position{Line: line, Column: col}
}

// rawOffset maps a byte offset within the decoded value of the JSON string
// literal raw (quotes included) to the byte offset within raw where the
// character holding it is written.
func rawOffset(raw []byte, decoded int) int {
	n := 0 // decoded bytes so far
	for i := 1; i < len(raw)-1; {
		if n >= decoded {
			return i
		}
		if raw[i] != '\\' {
			_, size := utf8.DecodeRune(raw[i:])
			n += size
			i += size
			continue
		}
		if raw[i+1] != 'u' {
			n++
			i += 2
			continue
		}
		r, width := unicodeEscape(raw[i:])
		n += utf8.RuneLen(r)
		i += width
	}
	return len(raw) - 1
}

// unicodeEscape decodes \uXXXX (and a following low surrogate) at the start
// of b, returning the rune and the bytes it spans.
func unicodeEscape(b []byte) (rune, int) {
	hex := func(b []byte) rune {
		if len(b) < 6 {
			return utf8.RuneError
		}
		v, err := strconv.ParseUint(string(b[2:6]), 16, 32)
		if err != nil {
			return utf8.RuneError
		}
		return rune(v)
	}
	r := hex(b)
	if r >= 0xD800 && r < 0xDC00 && len(b) >= 12 && b[6] == '\\' && b[7] == 'u' {
		if lo := hex(b[6:]); lo >= 0xDC00 && lo < 0xE000 {
			return (r-0xD800)<<10 + (lo - 0xDC00) + 0x10000, 12
		}
	}
	if r >= 0xD800 && r < 0xE000 {
		r = utf8.RuneError
	}
	return r, 6
}
