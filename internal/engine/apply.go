package engine

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/block"
	"github.com/nerdwave-nick/servitor/internal/config"
)

// Options control how a switch is applied.
type Options struct {
	DryRun bool // compute the changes without writing anything
}

// Change describes the effect of applying a switch on one file.
type Change struct {
	Path    string // real path written (symlinks resolved)
	Before  string
	After   string
	Created bool
	mode    fs.FileMode
}

// Changed reports whether the file content differs.
func (c Change) Changed() bool { return c.Created || c.Before != c.After }

// Result summarizes an Apply call.
type Result struct {
	Previous string // state before applying, "" when unknown
	Changes  []Change
}

// ResolveMeta merges the configured metadata of f for state with the given
// overrides and checks required keys. Overrides for keys the file does not
// declare are ignored.
func ResolveMeta(f *config.File, state string, overrides map[string]string) (map[string]string, error) {
	meta := map[string]string{}
	if v, ok := f.ValueFor(state); ok {
		for k, val := range v.Meta {
			meta[k] = val
		}
	}
	for k, val := range overrides {
		if _, declared := f.Meta[k]; declared {
			meta[k] = val
		}
	}
	for k, spec := range f.Meta {
		if spec.Required() && meta[k] == "" {
			return nil, fmt.Errorf("metadata key %q is required for %s (pass --%s)", k, f.File, k)
		}
	}
	for k, v := range meta {
		if v == "" {
			delete(meta, k)
		}
	}
	meta[block.StateKey] = state
	return meta, nil
}

// Apply switches sw to state. Files are only written when every block could
// be rendered; a failed write rolls back files written earlier in the same run.
func Apply(sw *config.Switch, state string, overrides map[string]string, opt Options) (Result, error) {
	if !sw.HasState(state) {
		return Result{}, fmt.Errorf("unknown state %q for switch %q (valid: %s)", state, sw.Name, strings.Join(sw.States, ", "))
	}
	declared := sw.MetaKeys()
	for k, v := range overrides {
		if _, ok := declared[k]; !ok {
			return Result{}, fmt.Errorf("switch %q does not declare metadata key %q", sw.Name, k)
		}
		if strings.ContainsAny(v, "\r\n") {
			return Result{}, fmt.Errorf("metadata value for %q must be single-line", k)
		}
	}
	metas := make([]map[string]string, len(sw.Files))
	for i := range sw.Files {
		m, err := ResolveMeta(&sw.Files[i], state, overrides)
		if err != nil {
			return Result{}, err
		}
		metas[i] = m
	}
	res := Result{Previous: ReadStatus(sw).State()}
	changes, err := plan(sw, false, func(i int, f *config.File, text string) (string, error) {
		v, _ := f.ValueFor(state)
		return block.Upsert(text, marker(f), metas[i], string(v.Value))
	})
	res.Changes = changes
	if err != nil || opt.DryRun {
		return res, err
	}
	return res, commit(changes)
}

// Purge removes the managed blocks of sw from all existing target files.
func Purge(sw *config.Switch, opt Options) ([]Change, error) {
	changes, err := plan(sw, true, func(_ int, f *config.File, text string) (string, error) {
		return block.Remove(text, marker(f))
	})
	if err != nil || opt.DryRun {
		return changes, err
	}
	return changes, commit(changes)
}

type editFunc func(i int, f *config.File, text string) (string, error)

// plan computes the new content of every target file without writing.
// Missing files are skipped when skipMissing is set, created when the entry
// allows it, and an error otherwise.
func plan(sw *config.Switch, skipMissing bool, edit editFunc) ([]Change, error) {
	byPath := map[string]*Change{}
	var order []string
	for i := range sw.Files {
		f := &sw.Files[i]
		path := realPath(sw.Target(f))
		c, ok := byPath[path]
		if !ok {
			c = &Change{Path: path, mode: 0o644}
			info, err := os.Stat(path)
			switch {
			case err == nil:
				data, rerr := os.ReadFile(path)
				if rerr != nil {
					return nil, rerr
				}
				c.Before, c.mode = string(data), info.Mode().Perm()
			case errors.Is(err, os.ErrNotExist) && skipMissing:
				continue
			case errors.Is(err, os.ErrNotExist) && f.Create:
				c.Created = true
			case errors.Is(err, os.ErrNotExist):
				return nil, fmt.Errorf("%s does not exist (set \"create\": true to create it)", path)
			default:
				return nil, err
			}
			c.After = c.Before
			byPath[path] = c
			order = append(order, path)
		}
		after, err := edit(i, f, c.After)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		c.After = after
	}
	changes := make([]Change, 0, len(order))
	for _, p := range order {
		changes = append(changes, *byPath[p])
	}
	return changes, nil
}

// commit writes all changed files atomically, rolling back on failure.
func commit(changes []Change) error {
	var done []Change
	for _, c := range changes {
		if !c.Changed() {
			continue
		}
		if err := writeAtomic(c.Path, []byte(c.After), c.mode); err != nil {
			rollback(done)
			return fmt.Errorf("writing %s: %w", c.Path, err)
		}
		done = append(done, c)
	}
	return nil
}

func rollback(done []Change) {
	for _, c := range done {
		if c.Created {
			_ = os.Remove(c.Path)
		} else {
			_ = writeAtomic(c.Path, []byte(c.Before), c.mode)
		}
	}
}

func writeAtomic(path string, data []byte, mode fs.FileMode) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".servitor-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.Remove(tmp.Name())
		}
	}()
	_, err = tmp.Write(data)
	if err == nil {
		err = tmp.Chmod(mode)
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// DiffLines returns the lines that differ between before and after, prefixed
// with "- " and "+ ", using the common prefix and suffix as boundaries.
func DiffLines(before, after string) []string {
	a, b := strings.Split(before, "\n"), strings.Split(after, "\n")
	p := 0
	for p < len(a) && p < len(b) && a[p] == b[p] {
		p++
	}
	s := 0
	for s < len(a)-p && s < len(b)-p && a[len(a)-1-s] == b[len(b)-1-s] {
		s++
	}
	var out []string
	for _, l := range a[p : len(a)-s] {
		out = append(out, "- "+l)
	}
	for _, l := range b[p : len(b)-s] {
		out = append(out, "+ "+l)
	}
	return out
}
