// Package engine applies switch states to files and reads them back.
package engine

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/block"
	"github.com/nerdwave-nick/servitor/internal/config"
)

var (
	// ErrNotApplied means no managed block exists in any target file.
	ErrNotApplied = errors.New("not applied")
	// ErrInconsistent means target files disagree or are partially managed.
	ErrInconsistent = errors.New("inconsistent")
)

// FileStatus is the observed state of one managed block.
type FileStatus struct {
	File    string            `json:"file"`
	Guard   string            `json:"guard"`
	Present bool              `json:"present"`
	Drift   bool              `json:"drift,omitempty"` // content differs from the configured value
	Meta    map[string]string `json:"meta,omitempty"`
	Error   string            `json:"error,omitempty"`
}

// Status is the observed state of a switch across all its files.
type Status struct {
	Files []FileStatus
	Meta  map[string]string // combined metadata, valid when Err is nil
	Err   error
}

// State returns the combined current state, or "" when unknown.
func (s Status) State() string {
	if s.Err != nil {
		return ""
	}
	return s.Meta[block.StateKey]
}

func marker(f *config.File) block.Marker {
	return block.Marker{Comment: f.Comment, CommentEnd: f.CommentEnd, Guard: f.Guard}
}

// realPath resolves symlinks so that dotfile symlinks are edited in place.
func realPath(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

// ReadStatus inspects all target files of sw.
func ReadStatus(sw *config.Switch) Status {
	var st Status
	for i := range sw.Files {
		f := &sw.Files[i]
		fs := FileStatus{File: sw.Target(f), Guard: f.Guard}
		data, err := os.ReadFile(realPath(fs.File))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			fs.Error = err.Error()
		}
		if err == nil {
			b, found, ferr := block.Find(string(data), marker(f))
			switch {
			case ferr != nil:
				fs.Error = ferr.Error()
			case found:
				fs.Present, fs.Meta = true, b.Meta
				v, ok := f.ValueFor(b.Meta[block.StateKey])
				fs.Drift = !ok || strings.TrimSuffix(string(v.Value), "\n") != b.Content
			}
		}
		st.Files = append(st.Files, fs)
	}
	st.Meta, st.Err = combine(st.Files)
	return st
}

func combine(files []FileStatus) (map[string]string, error) {
	merged := map[string]string{}
	present := 0
	for _, f := range files {
		if f.Error != "" {
			return nil, fmt.Errorf("%w: %s: %s", ErrInconsistent, f.File, f.Error)
		}
		if f.Present {
			present++
		}
	}
	if present == 0 {
		return nil, fmt.Errorf("%w: no managed block found in %s", ErrNotApplied, fileList(files))
	}
	for _, f := range files {
		if !f.Present {
			return nil, fmt.Errorf("%w: managed block for guard %q missing in %s", ErrInconsistent, f.Guard, f.File)
		}
		for k, v := range f.Meta {
			if old, ok := merged[k]; ok && old != v {
				return nil, fmt.Errorf("%w: files disagree on %q (%q vs %q)", ErrInconsistent, k, old, v)
			}
			merged[k] = v
		}
	}
	if merged[block.StateKey] == "" {
		return nil, fmt.Errorf("%w: managed block has no state", ErrInconsistent)
	}
	return maps.Clone(merged), nil
}

func fileList(files []FileStatus) string {
	names := make([]string, len(files))
	for i, f := range files {
		names[i] = f.File
	}
	return strings.Join(names, ", ")
}
