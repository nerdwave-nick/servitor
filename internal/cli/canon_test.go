package cli

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"reflect"
	"testing"
)

// sameJSON tells whether the JSON value got holds what the JSON text want
// holds, the order of object keys aside.
func sameJSON(t *testing.T, got any, want string) bool {
	t.Helper()
	var w any
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatal(err)
	}
	return reflect.DeepEqual(got, w)
}

// TestAugury_TheVisageShowsTheCanonOfItsReason: the augury of the example
// rite theme bears the declaration of its inscription "reason" beside the
// inscriptions its data-slate holds.
func TestAugury_TheVisageShowsTheCanonOfItsReason(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	var out, errb bytes.Buffer
	code := Execute([]string{"--librarium", "../../examples", "augury", "theme"}, &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, out.String(), errb.String())
	}
	var augury map[string]jsontext.Value
	if err := json.Unmarshal(out.Bytes(), &augury); err != nil {
		t.Fatal(err)
	}
	want := `[{"key":"reason","purpose":"why the visage was changed","mandatory":false,"decrees":{},"decree":null}]`
	if got := string(compactValue(t, augury["canon"])); got != want {
		t.Errorf("canon\n got  %s\n want %s", got, want)
	}
	if got := string(compactValue(t, augury["inscriptions"])); got != "{}" {
		t.Errorf("inscriptions %s, want {}", got)
	}
}

func compactValue(t *testing.T, v jsontext.Value) jsontext.Value {
	t.Helper()
	v = v.Clone()
	if err := v.Compact(); err != nil {
		t.Fatalf("%q: %v", v, err)
	}
	return v
}

// TestCensus_EveryEntryBearsItsCanon: every entry of the binharic census
// bears the canon of its rite — decrees expanded per aspect, an edict's one
// plain decree — and a heretical rite bears "canon": [].
func TestCensus_EveryEntryBearsItsCanon(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", edictJSON)
	e.addRite("rites/broken.json", `{`)
	e.addRite("rites/dnd.json", `{"pattern": "Mark II", "aspects": ["on", "off", "idle"],
  "inscriptions": {"mode": {"purpose": "how silent", "decrees": {"on": "deep", "*": "light"}}},
  "subjects": [{"name": "mako", "designation": "machine-spirit", "enact": {"*": "true"}}]}`)

	entries := e.census()

	want := map[string]string{
		"broken":  `[]`,
		"capture": `[{"key":"region","purpose":"the slurp geometry","mandatory":false,"decrees":{},"decree":"full"}]`,
		"dnd": `[{"key":"mode","purpose":"how silent","mandatory":false,` +
			`"decrees":{"idle":"light","off":"light","on":"deep"},"decree":null}]`,
		"mouse-autohide-toggle": `[{"key":"reason","purpose":"why the rite was invoked","mandatory":false,"decrees":{},"decree":null},` +
			`{"key":"example-key","purpose":"","mandatory":true,"decrees":{"off":"other-value","on":"example-value"},"decree":null}]`,
	}
	for rite, canon := range want {
		got, ok := entries[rite]["canon"]
		if !ok {
			t.Errorf("%s bears no canon: %v", rite, entries[rite])
			continue
		}
		if !sameJSON(t, got, canon) {
			t.Errorf("%s: canon\n got  %v\n want %s", rite, got, canon)
		}
	}
}
