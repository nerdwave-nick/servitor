package config

import (
	"os"
	"path/filepath"
	"strings"
)

// commentStyles maps lowercase file extensions to comment prefix/suffix.
var commentStyles = map[string][2]string{
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

// DefaultComment returns the comment prefix and suffix used for path when the
// configuration does not specify one. Unknown extensions use "#".
func DefaultComment(path string) (prefix, suffix string) {
	if style, ok := commentStyles[strings.ToLower(filepath.Ext(path))]; ok {
		return style[0], style[1]
	}
	return "#", ""
}

// ExpandPath expands a leading "~", environment variables, and resolves
// relative paths against baseDir.
func ExpandPath(p, baseDir string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			p = home + p[1:]
		}
	}
	p = os.ExpandEnv(p)
	if !filepath.IsAbs(p) {
		p = filepath.Join(baseDir, p)
	}
	return filepath.Clean(p)
}

// Target returns the resolved absolute path of the managed file.
func (s *Switch) Target(f *File) string { return ExpandPath(f.File, s.baseDir) }

// DefaultDir returns the default configuration directory
// ($XDG_CONFIG_HOME/servitor, falling back to ~/.config/servitor).
func DefaultDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "servitor")
}
