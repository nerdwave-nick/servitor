package config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tailscale/hujson"
)

// Severity of a diagnostic.
type Severity string

const (
	SevError   Severity = "error"
	SevWarning Severity = "warning"
)

// Diagnostic is a problem found in a configuration or a managed file.
type Diagnostic struct {
	File     string   `json:"file"`
	Line     int      `json:"line,omitempty"`
	Col      int      `json:"column,omitempty"`
	Severity Severity `json:"severity"`
	Switch   string   `json:"switch,omitempty"`
	Message  string   `json:"message"`
}

func (d Diagnostic) String() string {
	loc := d.File
	if d.Line > 0 {
		loc = fmt.Sprintf("%s:%d:%d", d.File, d.Line, d.Col)
	}
	return fmt.Sprintf("%s: %s: %s", loc, d.Severity, d.Message)
}

// Diagnostics is a list of diagnostics.
type Diagnostics []Diagnostic

// HasErrors reports whether any diagnostic is an error.
func (ds Diagnostics) HasErrors() bool {
	for _, d := range ds {
		if d.Severity == SevError {
			return true
		}
	}
	return false
}

// Sort orders diagnostics by file, line and column.
func (ds Diagnostics) Sort() {
	sort.SliceStable(ds, func(i, j int) bool {
		a, b := ds[i], ds[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Col < b.Col
	})
}

// source is a parsed config file used to map JSON pointers to positions.
type source struct {
	path string
	data []byte
	ast  hujson.Value
}

// lineCol converts a byte offset into 1-based line and column numbers.
func (s *source) lineCol(off int) (int, int) {
	if off < 0 || off > len(s.data) {
		return 0, 0
	}
	line, col := 1, 1
	for _, c := range s.data[:off] {
		if c == '\n' {
			line, col = line+1, 1
		} else {
			col++
		}
	}
	return line, col
}

// at creates a diagnostic positioned at the JSON pointer ptr. When ptr does
// not exist the closest existing parent is used.
func (s *source) at(sev Severity, ptr, format string, args ...any) Diagnostic {
	d := Diagnostic{File: s.path, Severity: sev, Message: fmt.Sprintf(format, args...)}
	for {
		if v := s.ast.Find(ptr); v != nil {
			d.Line, d.Col = s.lineCol(v.StartOffset)
			return d
		}
		if ptr == "" {
			return d
		}
		ptr = parentPointer(ptr)
	}
}

// keyAt is like at but points at the member name of ptr instead of its value.
func (s *source) keyAt(sev Severity, ptr, format string, args ...any) Diagnostic {
	d := s.at(sev, ptr, format, args...)
	parent := s.ast.Find(parentPointer(ptr))
	if parent == nil {
		return d
	}
	if obj, ok := parent.Value.(*hujson.Object); ok {
		name := strings.NewReplacer("~1", "/", "~0", "~").Replace(ptr[len(parentPointer(ptr))+1:])
		for _, m := range obj.Members {
			if lit, ok := m.Name.Value.(hujson.Literal); ok && lit.String() == name {
				d.Line, d.Col = s.lineCol(m.Name.StartOffset)
			}
		}
	}
	return d
}

func parentPointer(ptr string) string {
	for i := len(ptr) - 1; i >= 0; i-- {
		if ptr[i] == '/' {
			return ptr[:i]
		}
	}
	return ""
}
