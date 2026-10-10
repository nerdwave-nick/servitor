package librarium

import (
	"path/filepath"
	"strings"
)

// glyphs maps lowercase vessel extensions to glyph and closing glyph.
var glyphs = map[string][2]string{
	".kdl": {"//"}, ".js": {"//"}, ".ts": {"//"}, ".jsonc": {"//"}, ".json5": {"//"},
	".c": {"//"}, ".h": {"//"}, ".cpp": {"//"}, ".hpp": {"//"}, ".go": {"//"},
	".rs": {"//"}, ".java": {"//"}, ".kt": {"//"}, ".swift": {"//"}, ".zig": {"//"},
	".scss": {"//"}, ".less": {"//"}, ".qml": {"//"}, ".dart": {"//"},
	".lua": {"--"}, ".sql": {"--"}, ".hs": {"--"},
	".vim": {`"`}, ".tex": {"%"}, ".el": {";;"}, ".scm": {";;"}, ".lisp": {";;"},
	".ini":  {";"},
	".css":  {"/*", "*/"},
	".html": {"<!--", "-->"}, ".xml": {"<!--", "-->"}, ".svg": {"<!--", "-->"}, ".md": {"<!--", "-->"},
}

// DefaultGlyphs returns the glyph and closing glyph inferred from a vessel's
// extension; unknown extensions use "#" and no closing glyph.
func DefaultGlyphs(vessel string) (glyph, closing string) {
	if g, ok := glyphs[strings.ToLower(filepath.Ext(vessel))]; ok {
		return g[0], g[1]
	}
	return "#", ""
}
