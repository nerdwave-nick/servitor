package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/block"
	"github.com/nerdwave-nick/servitor/internal/config"
)

// draft is the editable form of a switch definition used by the wizard.
// Defaults (guard = name, comment by extension) are kept empty so that they
// follow renames and path changes.
type draft struct {
	origName, origPath string // empty for a new switch
	name, description  string
	states             string // comma separated
	vessels            []vessel
}

// vessel is the editable form of one file entry.
type vessel struct {
	file, guard, comment, commentEnd string
	create                           bool
	inscriptions                     string            // "reason, mode!" — "!" marks required keys
	descs                            map[string]string // preserved key descriptions
	values                           map[string]string // state → managed content
	meta                             map[string]map[string]string
}

func newVessel() vessel {
	return vessel{descs: map[string]string{}, values: map[string]string{}, meta: map[string]map[string]string{}}
}

func draftFrom(sw *config.Switch) draft {
	d := draft{origName: sw.Name, origPath: sw.Path, name: sw.Name, description: sw.Description,
		states: strings.Join(sw.States, ", ")}
	for _, f := range sw.Files {
		v := newVessel()
		v.file, v.create = f.File, f.Create
		if f.Guard != sw.Name {
			v.guard = f.Guard
		}
		if c, ce := config.DefaultComment(f.File); f.Comment != c || f.CommentEnd != ce {
			v.comment, v.commentEnd = f.Comment, f.CommentEnd
		}
		var keys []string
		for _, k := range sortedKeys(f.Meta) {
			spec := f.Meta[k]
			if spec.Required() {
				k += "!"
			}
			keys = append(keys, k)
			v.descs[strings.TrimSuffix(k, "!")] = spec.Description
		}
		v.inscriptions = strings.Join(keys, ", ")
		for _, val := range f.Values {
			v.values[val.State] = string(val.Value)
			v.meta[val.State] = val.Meta
		}
		d.vessels = append(d.vessels, v)
	}
	return d
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// splitList parses a comma or whitespace separated list, dropping blanks.
func splitList(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' })
}

func parseStates(s string) ([]string, error) {
	states := splitList(s)
	if len(states) == 0 {
		return nil, fmt.Errorf("name at least one")
	}
	seen := map[string]bool{}
	for _, st := range states {
		if !config.ValidName(st) {
			return nil, fmt.Errorf("%q: use letters, digits, '.', '_' and '-'", st)
		}
		if seen[st] {
			return nil, fmt.Errorf("%q appears twice", st)
		}
		seen[st] = true
	}
	return states, nil
}

// parseInscriptions returns the declared keys and which of them are required.
func parseInscriptions(s string) (keys []string, required map[string]bool, err error) {
	required = map[string]bool{}
	for _, k := range splitList(s) {
		req := strings.HasSuffix(k, "!")
		k = strings.TrimSuffix(k, "!")
		switch {
		case !block.ValidKey(k):
			return nil, nil, fmt.Errorf("%q: use letters, digits, '.', '_' and '-'", k)
		case slices.Contains(config.ReservedKeys, k):
			return nil, nil, fmt.Errorf("%q is reserved", k)
		case slices.Contains(keys, k):
			return nil, nil, fmt.Errorf("%q appears twice", k)
		}
		keys = append(keys, k)
		required[k] = req
	}
	return keys, required, nil
}

// toSwitch converts the draft into a definition. Invalid parts are passed
// through as far as possible; config validation reports what is wrong.
func (d draft) toSwitch() *config.Switch {
	states, _ := parseStates(d.states)
	sw := &config.Switch{Name: d.name, Description: strings.TrimSpace(d.description), States: states}
	for _, v := range d.vessels {
		f := config.File{File: v.file, Guard: v.guard, Comment: v.comment, CommentEnd: v.commentEnd, Create: v.create}
		if f.Guard == "" {
			f.Guard = d.name
		}
		if f.Comment == "" && f.CommentEnd == "" {
			f.Comment, f.CommentEnd = config.DefaultComment(f.File)
		}
		keys, required, _ := parseInscriptions(v.inscriptions)
		if len(keys) > 0 {
			f.Meta = map[string]config.MetaSpec{}
		}
		for _, k := range keys {
			spec := config.MetaSpec{Description: v.descs[k]}
			if required[k] {
				no := config.FlexBool(false)
				spec.Optional = &no
			}
			f.Meta[k] = spec
		}
		for _, st := range states {
			val := config.Value{State: st, Value: config.Text(v.values[st])}
			for _, k := range keys {
				if mv := v.meta[st][k]; mv != "" {
					if val.Meta == nil {
						val.Meta = map[string]string{}
					}
					val.Meta[k] = mv
				}
			}
			f.Values = append(f.Values, val)
		}
		sw.Files = append(sw.Files, f)
	}
	return sw
}

// summary is a one-line description of a vessel for lists.
func (v vessel) summary(t *theme, guardWord, name string, width int) string {
	g := v.guard
	if g == "" {
		g = name
	}
	s := t.text.Render(truncateLeft(shortPath(v.file), width*3/5)) + t.dim.Render("  "+guardWord+" "+g)
	if v.inscriptions != "" {
		s += t.dim.Render("  · " + v.inscriptions)
	}
	return s
}
