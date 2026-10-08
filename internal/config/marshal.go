package config

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"
)

// MarshalJSONTo writes multi-line text as an array of lines for readability.
func (t Text) MarshalJSONTo(enc *jsontext.Encoder) error {
	if !strings.Contains(string(t), "\n") {
		return enc.WriteToken(jsontext.String(string(t)))
	}
	return json.MarshalEncode(enc, strings.Split(string(t), "\n"))
}

// Marshal renders sw as an indented JSON definition. Values equal to their
// defaults (guard = switch name, comment inferred from the extension) are
// omitted so that the file stays minimal.
func Marshal(sw *Switch) ([]byte, error) {
	out := *sw
	out.Files = make([]File, len(sw.Files))
	for i, f := range sw.Files {
		if f.Guard == sw.Name {
			f.Guard = ""
		}
		if c, ce := DefaultComment(f.File); f.Comment == c && f.CommentEnd == ce {
			f.Comment, f.CommentEnd = "", ""
		}
		out.Files[i] = f
	}
	data, err := json.Marshal(&out, jsontext.WithIndent("  "), json.Deterministic(true))
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
