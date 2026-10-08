package block

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// StateKey is the automatic, non-overridable metadata key holding the state.
const StateKey = "state"

var (
	keyRe       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	bareValueRe = regexp.MustCompile(`^[^\s"|\\]+$`)
)

// ValidKey reports whether k can be used as a metadata key.
func ValidKey(k string) bool { return keyRe.MatchString(k) }

// SortedKeys returns the keys of meta with "state" first and the rest sorted.
func SortedKeys(meta map[string]string) []string {
	keys := make([]string, 0, len(meta))
	for k := range meta {
		if k != StateKey {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	if _, ok := meta[StateKey]; ok {
		keys = append([]string{StateKey}, keys...)
	}
	return keys
}

// EncodeMeta renders meta as space separated key|value pairs. Empty values
// are omitted; values containing whitespace or special characters are quoted.
func EncodeMeta(meta map[string]string) string {
	var parts []string
	for _, k := range SortedKeys(meta) {
		v := meta[k]
		if v == "" {
			continue
		}
		if !bareValueRe.MatchString(v) {
			v = strconv.Quote(v)
		}
		parts = append(parts, k+"|"+v)
	}
	return strings.Join(parts, " ")
}

// DecodeMeta parses the key|value pairs produced by EncodeMeta.
func DecodeMeta(s string) (map[string]string, error) {
	meta := map[string]string{}
	rest := strings.TrimSpace(s)
	for rest != "" {
		key, after, ok := strings.Cut(rest, "|")
		if !ok || !ValidKey(key) {
			return nil, fmt.Errorf("malformed metadata %q: expected key|value", rest)
		}
		var val string
		if strings.HasPrefix(after, `"`) {
			quoted, err := strconv.QuotedPrefix(after)
			if err != nil {
				return nil, fmt.Errorf("malformed quoted value for key %q: %w", key, err)
			}
			val, _ = strconv.Unquote(quoted)
			after = after[len(quoted):]
			if after != "" && !strings.HasPrefix(after, " ") && !strings.HasPrefix(after, "\t") {
				return nil, fmt.Errorf("malformed metadata after key %q: missing separator", key)
			}
		} else {
			end := strings.IndexAny(after, " \t")
			if end < 0 {
				end = len(after)
			}
			val, after = after[:end], after[end:]
		}
		if _, dup := meta[key]; dup {
			return nil, fmt.Errorf("duplicate metadata key %q", key)
		}
		meta[key] = val
		rest = strings.TrimSpace(after)
	}
	return meta, nil
}
