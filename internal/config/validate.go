package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/nerdwave-nick/servitor/internal/block"
)

// ReservedKeys cannot be declared as metadata keys: "state" is managed
// automatically and the others collide with built-in command flags.
var ReservedKeys = []string{block.StateKey, "help", "config", "dry-run", "quiet", "version"}

// validate checks a decoded switch for semantic errors.
func validate(src *source, sw *Switch) Diagnostics {
	var ds Diagnostics
	errAt := func(ptr, format string, args ...any) {
		ds = append(ds, src.at(SevError, ptr, format, args...))
	}

	if len(sw.States) == 0 {
		errAt("/states", "at least one state must be declared in \"states\"")
	}
	seenState := map[string]bool{}
	for i, st := range sw.States {
		ptr := fmt.Sprintf("/states/%d", i)
		switch {
		case !nameRe.MatchString(st):
			errAt(ptr, "invalid state %q: use letters, digits, '.', '_' and '-' only", st)
		case seenState[st]:
			errAt(ptr, "duplicate state %q", st)
		}
		seenState[st] = true
	}
	if len(sw.Files) == 0 {
		errAt("/files", "at least one entry must be declared in \"files\"")
	}
	for i := range sw.Files {
		ds = append(ds, validateFile(src, sw, i, seenState)...)
	}
	return ds
}

func validateFile(src *source, sw *Switch, i int, states map[string]bool) Diagnostics {
	var ds Diagnostics
	f := &sw.Files[i]
	base := fmt.Sprintf("/files/%d", i)
	errAt := func(ptr, format string, args ...any) {
		ds = append(ds, src.at(SevError, base+ptr, format, args...))
	}

	if strings.TrimSpace(f.File) == "" {
		errAt("/file", "\"file\" must not be empty")
	} else if p := os.ExpandEnv(f.File); !filepath.IsAbs(p) && p != "~" && !strings.HasPrefix(p, "~/") {
		ds = append(ds, src.at(SevWarning, base+"/file", "relative path %q is resolved against the config directory: %s", f.File, sw.Target(f)))
	}
	if f.Guard == "" || strings.IndexFunc(f.Guard, unicode.IsSpace) >= 0 {
		errAt("/guard", "invalid guard %q: must be non-empty and contain no whitespace", f.Guard)
	}
	if strings.TrimSpace(f.Comment) == "" || strings.ContainsAny(f.Comment, "\r\n") {
		errAt("/comment", "\"comment\" must be a non-empty single-line comment prefix")
	}
	if strings.ContainsAny(f.CommentEnd, "\r\n") {
		errAt("/comment_end", "\"comment_end\" must be single-line")
	}
	if f.CommentEnd != "" && strings.Contains(f.Guard, f.CommentEnd) {
		errAt("/guard", "guard %q contains the comment terminator %q", f.Guard, f.CommentEnd)
	}

	for key := range f.Meta {
		ptr := "/meta/" + escapePointer(key)
		switch {
		case !block.ValidKey(key):
			errAt(ptr, "invalid metadata key %q: use letters, digits, '.', '_' and '-' only", key)
		case slices.Contains(ReservedKeys, key):
			errAt(ptr, "metadata key %q is reserved", key)
		}
	}

	marker := block.Marker{Comment: f.Comment, CommentEnd: f.CommentEnd, Guard: f.Guard}
	covered := map[string]bool{}
	for j, v := range f.Values {
		vptr := fmt.Sprintf("/values/%d", j)
		switch {
		case !states[v.State]:
			errAt(vptr+"/state", "state %q is not declared in \"states\" (%s)", v.State, strings.Join(sw.States, ", "))
		case covered[v.State]:
			errAt(vptr+"/state", "duplicate value for state %q", v.State)
		}
		covered[v.State] = true
		for key, val := range v.Meta {
			kptr := vptr + "/meta/" + escapePointer(key)
			if key == block.StateKey {
				errAt(kptr, "metadata key \"state\" is set automatically and cannot be configured")
			} else if _, ok := f.Meta[key]; !ok {
				errAt(kptr, "metadata key %q is not declared in this file's \"meta\"", key)
			}
			if strings.ContainsAny(val, "\r\n") {
				errAt(kptr, "metadata values must be single-line")
			}
		}
		for _, line := range strings.Split(string(v.Value), "\n") {
			if marker.IsMarkerLine(line) {
				errAt(vptr+"/value", "value contains a servitor marker line for its own guard: %q", line)
				break
			}
		}
	}
	for _, st := range sw.States {
		if !covered[st] {
			errAt("/values", "no value configured for state %q", st)
		}
	}
	for key, spec := range f.Meta {
		if !spec.Required() {
			continue
		}
		for _, st := range sw.States {
			if v, ok := f.ValueFor(st); ok && v.Meta[key] == "" {
				ds = append(ds, src.at(SevWarning, base+"/meta/"+escapePointer(key),
					"required metadata key %q has no value for state %q; it must be passed as --%s", key, st, key))
			}
		}
	}
	return ds
}

// guardClashes reports file entries that manage the same guard in the same
// target file, across all switches.
func guardClashes(loaded []*loadedSwitch) Diagnostics {
	type owner struct {
		ls  *loadedSwitch
		idx int
	}
	seen := map[[2]string]owner{}
	var ds Diagnostics
	for _, ls := range loaded {
		for i := range ls.sw.Files {
			f := &ls.sw.Files[i]
			key := [2]string{ls.sw.Target(f), f.Guard}
			first, dup := seen[key]
			if !dup {
				seen[key] = owner{ls, i}
				continue
			}
			for _, o := range []owner{first, {ls, i}} {
				d := o.ls.src.at(SevError, fmt.Sprintf("/files/%d/guard", o.idx),
					"guard %q for %s is used by both switch %q (%s) and switch %q (%s)",
					f.Guard, key[0], first.ls.sw.Name, first.ls.src.path, ls.sw.Name, ls.src.path)
				d.Switch = o.ls.sw.Name
				ds = append(ds, d)
			}
		}
	}
	return ds
}

func escapePointer(s string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(s)
}
