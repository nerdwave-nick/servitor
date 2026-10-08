package config

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/tailscale/hujson"
)

// SubDirs are scanned (in order) for switch definitions inside the config
// directory. "rites" is canonical; "switches" is accepted as an alias.
var SubDirs = []string{"rites", "switches"}

// Extensions are the accepted switch definition file extensions.
var Extensions = []string{".json", ".jsonc"}

var (
	nameRe      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	hujsonPosRe = regexp.MustCompile(`^hujson: line (\d+), column (\d+): (.*)$`)
)

// Set is the result of loading a configuration directory.
type Set struct {
	Dir      string
	Switches map[string]*Switch  // switches without errors, usable at runtime
	Broken   map[string]bool     // switch names that failed to load or validate
	Files    map[string][]string // definition file paths of every discovered name
	Diags    Diagnostics
}

// Overlay substitutes one definition for validation without touching disk.
type Overlay struct {
	Name    string // name the definition would be saved as
	Data    []byte // definition content
	Replace string // existing switch the overlay supersedes ("" when new)
}

// NewPath returns where a new definition called name is stored in dir.
func NewPath(dir, name string) string { return filepath.Join(dir, SubDirs[0], name+".json") }

// ValidName reports whether name can be used as a switch name.
func ValidName(name string) bool { return nameRe.MatchString(name) }

// Preview loads dir as if the overlay had been saved, for validating edits.
func Preview(dir string, ov Overlay) *Set { return load(dir, &ov) }

// Names returns the sorted names of all usable switches.
func (s *Set) Names() []string {
	names := make([]string, 0, len(s.Switches))
	for n := range s.Switches {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Load reads and validates every switch definition below dir.
func Load(dir string) *Set { return load(dir, nil) }

func load(dir string, ov *Overlay) *Set {
	set := &Set{Dir: dir, Switches: map[string]*Switch{}, Broken: map[string]bool{}}
	byName := map[string][]string{}
	var order []string
	for _, sub := range SubDirs {
		entries, err := os.ReadDir(filepath.Join(dir, sub))
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				set.Diags = append(set.Diags, Diagnostic{File: filepath.Join(dir, sub), Severity: SevError, Message: err.Error()})
			}
			continue
		}
		for _, e := range entries {
			ext := filepath.Ext(e.Name())
			if e.IsDir() || strings.HasPrefix(e.Name(), ".") || !isConfigExt(ext) {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ext)
			if len(byName[name]) == 0 {
				order = append(order, name)
			}
			byName[name] = append(byName[name], filepath.Join(dir, sub, e.Name()))
		}
	}
	overlayPath := ""
	if ov != nil {
		if ov.Replace != "" {
			delete(byName, ov.Replace)
		}
		overlayPath = NewPath(dir, ov.Name)
		if !slices.Contains(order, ov.Name) {
			order = append(order, ov.Name)
		}
		byName[ov.Name] = append(slices.DeleteFunc(byName[ov.Name], func(p string) bool { return p == overlayPath }), overlayPath)
	}
	set.Files = byName
	var loaded []*loadedSwitch
	for _, name := range order {
		if _, ok := byName[name]; !ok {
			continue
		}
		paths := byName[name]
		if len(paths) > 1 {
			for _, p := range paths {
				set.Diags = append(set.Diags, Diagnostic{File: p, Severity: SevError, Switch: name,
					Message: fmt.Sprintf("switch name %q is defined by multiple files: %s", name, strings.Join(paths, ", "))})
			}
			set.Broken[name] = true
			continue
		}
		var data []byte
		if paths[0] == overlayPath {
			data = ov.Data
		}
		ls, diags := loadFile(name, paths[0], dir, data)
		for i := range diags {
			diags[i].Switch = name
		}
		set.Diags = append(set.Diags, diags...)
		if ls == nil || diags.HasErrors() {
			set.Broken[name] = true
			continue
		}
		loaded = append(loaded, ls)
	}
	for _, d := range guardClashes(loaded) {
		set.Diags = append(set.Diags, d)
		set.Broken[d.Switch] = true
	}
	for _, ls := range loaded {
		if !set.Broken[ls.sw.Name] {
			set.Switches[ls.sw.Name] = ls.sw
		}
	}
	set.Diags.Sort()
	return set
}

func isConfigExt(ext string) bool {
	for _, e := range Extensions {
		if ext == e {
			return true
		}
	}
	return false
}

type loadedSwitch struct {
	sw  *Switch
	src *source
}

// loadFile parses one switch definition, reading path unless data is given.
// It returns nil when the definition could not be decoded at all.
func loadFile(name, path, baseDir string, data []byte) (*loadedSwitch, Diagnostics) {
	if data == nil {
		var err error
		if data, err = os.ReadFile(path); err != nil {
			return nil, Diagnostics{{File: path, Severity: SevError, Message: err.Error()}}
		}
	}
	ast, err := hujson.Parse(data)
	if err != nil {
		return nil, Diagnostics{syntaxDiag(path, err)}
	}
	src := &source{path: path, data: data, ast: ast}
	std := ast.Clone()
	std.Standardize()
	sw := &Switch{}
	if err := json.Unmarshal(std.Pack(), sw, json.RejectUnknownMembers(true)); err != nil {
		return nil, Diagnostics{decodeDiag(src, err)}
	}
	sw.Name, sw.Path, sw.baseDir = name, path, baseDir
	var diags Diagnostics
	if !nameRe.MatchString(name) {
		diags = append(diags, src.at(SevError, "", "invalid switch name %q (file name): use letters, digits, '.', '_' and '-' only, starting with a letter or digit", name))
	}
	for i := range sw.Files {
		f := &sw.Files[i]
		if f.Guard == "" {
			f.Guard = name
		}
		if f.Comment == "" && f.CommentEnd == "" {
			f.Comment, f.CommentEnd = DefaultComment(f.File)
		}
	}
	diags = append(diags, validate(src, sw)...)
	return &loadedSwitch{sw: sw, src: src}, diags
}

func syntaxDiag(path string, err error) Diagnostic {
	d := Diagnostic{File: path, Severity: SevError, Message: "syntax error: " + err.Error()}
	if m := hujsonPosRe.FindStringSubmatch(err.Error()); m != nil {
		d.Line, _ = strconv.Atoi(m[1])
		d.Col, _ = strconv.Atoi(m[2])
		d.Message = "syntax error: " + m[3]
	}
	return d
}

func decodeDiag(src *source, err error) Diagnostic {
	var se *json.SemanticError
	if !errors.As(err, &se) {
		return Diagnostic{File: src.path, Severity: SevError, Message: err.Error()}
	}
	ptr := string(se.JSONPointer)
	var msg string
	switch {
	case errors.Is(err, json.ErrUnknownName):
		msg = fmt.Sprintf("unknown field %q", ptr[strings.LastIndex(ptr, "/")+1:])
	case se.Err != nil:
		msg = se.Err.Error()
	case se.GoType != nil:
		msg = fmt.Sprintf("invalid %s value, expected %s", kindName(se.JSONKind), goTypeName(se.GoType.String()))
	default:
		msg = err.Error()
	}
	if ptr != "" {
		msg = ptr + ": " + msg
	}
	at := src.at
	if errors.Is(err, json.ErrUnknownName) {
		at = src.keyAt
	}
	d := at(SevError, ptr, "%s", msg)
	if d.Line == 0 && se.ByteOffset > 0 {
		d.Line, d.Col = src.lineCol(int(se.ByteOffset))
	}
	return d
}

func kindName(k interface{ String() string }) string {
	switch s := k.String(); s {
	case "\"":
		return "string"
	case "0":
		return "number"
	case "{":
		return "object"
	case "[":
		return "array"
	case "t", "f":
		return "boolean"
	case "n":
		return "null"
	default:
		return s
	}
}

func goTypeName(t string) string {
	switch {
	case strings.HasPrefix(t, "[]"):
		return "array of " + goTypeName(t[2:])
	case strings.HasPrefix(t, "map["):
		return "object"
	case t == "bool":
		return "boolean"
	case strings.HasPrefix(t, "config."):
		return "object"
	}
	return t
}
