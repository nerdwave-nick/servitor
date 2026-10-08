// Package config loads and validates servitor switch definitions.
//
// A switch is a named set of file changes with discrete states. Each switch
// lives in its own JSON (or JSONC: comments and trailing commas allowed) file
// whose base name is the switch name.
package config

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"strings"
)

// Switch is one configured set of file changes.
type Switch struct {
	Name string `json:"-"`
	Path string `json:"-"` // path of the config file defining this switch

	Description string   `json:"description,omitzero"`
	States      []string `json:"states"`
	Files       []File   `json:"files"`

	baseDir string // directory relative target paths are resolved against
}

// File describes one managed block in one target file.
type File struct {
	File       string              `json:"file"`
	Guard      string              `json:"guard,omitzero"`       // defaults to the switch name
	Comment    string              `json:"comment,omitzero"`     // defaults by file extension
	CommentEnd string              `json:"comment_end,omitzero"` // for block comments like /* */
	Create     bool                `json:"create,omitzero"`      // create the target file if missing
	Values     []Value             `json:"values"`
	Meta       map[string]MetaSpec `json:"meta,omitempty"`
}

// Value is the managed content for one state.
type Value struct {
	State string            `json:"state"`
	Value Text              `json:"value"`
	Meta  map[string]string `json:"meta,omitempty"`
}

// MetaSpec declares a metadata key that can be stored in the block header.
type MetaSpec struct {
	Optional    *FlexBool `json:"optional,omitzero"` // defaults to true
	Description string    `json:"description,omitzero"`
}

// Required reports whether the key must have a non-empty value.
func (m MetaSpec) Required() bool { return m.Optional != nil && !bool(*m.Optional) }

// Text is a string that may also be written as an array of lines.
type Text string

// FlexBool is a bool that also accepts the strings "true" and "false".
type FlexBool bool

func semErr(msg string) error { return &json.SemanticError{Err: errors.New(msg)} }

func readStringOrList(dec *jsontext.Decoder, what string) ([]string, error) {
	switch dec.PeekKind() {
	case '"':
		var s string
		err := json.UnmarshalDecode(dec, &s)
		return []string{s}, err
	case '[':
		var list []string
		if err := json.UnmarshalDecode(dec, &list); err != nil {
			return nil, err
		}
		return list, nil
	}
	_ = dec.SkipValue()
	return nil, semErr(what + " must be a string or an array of strings")
}

// UnmarshalJSONFrom accepts a string or an array of lines joined with "\n".
func (t *Text) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	lines, err := readStringOrList(dec, "value")
	*t = Text(strings.Join(lines, "\n"))
	return err
}

// UnmarshalJSONFrom accepts true, false, "true" and "false".
func (b *FlexBool) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	switch {
	case tok.Kind() == 't', tok.Kind() == '"' && tok.String() == "true":
		*b = true
	case tok.Kind() == 'f', tok.Kind() == '"' && tok.String() == "false":
		*b = false
	default:
		return semErr(`expected true, false, "true" or "false"`)
	}
	return nil
}

// MetaKeys returns all metadata keys declared by any file of the switch, with
// the spec of the first declaration.
func (s *Switch) MetaKeys() map[string]MetaSpec {
	keys := map[string]MetaSpec{}
	for _, f := range s.Files {
		for k, spec := range f.Meta {
			if _, ok := keys[k]; !ok {
				keys[k] = spec
			}
		}
	}
	return keys
}

// MetaValues returns the configured values for key across all states.
func (s *Switch) MetaValues(key string) []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range s.Files {
		for _, v := range f.Values {
			if val, ok := v.Meta[key]; ok && val != "" && !seen[val] {
				seen[val] = true
				out = append(out, val)
			}
		}
	}
	return out
}

// HasState reports whether state is declared by the switch.
func (s *Switch) HasState(state string) bool {
	for _, st := range s.States {
		if st == state {
			return true
		}
	}
	return false
}

// ValueFor returns the configured value for state, if any.
func (f *File) ValueFor(state string) (Value, bool) {
	for _, v := range f.Values {
		if v.State == state {
			return v, true
		}
	}
	return Value{}, false
}
