package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/tailscale/hujson"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// compile compiles the schema named name as the faithful's editors would.
func compile(t *testing.T, name string) *jsonschema.Schema {
	t.Helper()
	data, ok := For(name)
	if !ok {
		t.Fatalf("no schema %q", name)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("schema %s is no JSON: %v", name, err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource(URL(name), doc); err != nil {
		t.Fatalf("schema %s: %v", name, err)
	}
	s, err := c.Compile(URL(name))
	if err != nil {
		t.Fatalf("schema %s does not compile: %v", name, err)
	}
	return s
}

// validate judges JSONC scripture against the schema.
func validate(t *testing.T, s *jsonschema.Schema, scripture []byte) error {
	t.Helper()
	std, err := hujson.Standardize(scripture)
	if err != nil {
		t.Fatalf("scripture is no JSONC: %v\n%s", err, scripture)
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(std))
	if err != nil {
		t.Fatalf("scripture is no JSON: %v", err)
	}
	return s.Validate(inst)
}

// tree reads the schema named name as plain JSON.
func tree(t *testing.T, name string) map[string]any {
	t.Helper()
	data, _ := For(name)
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("schema %s: %v", name, err)
	}
	return m
}

func obj(t *testing.T, v any, path ...string) map[string]any {
	t.Helper()
	for _, p := range path {
		m, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("no object at %v", path)
		}
		v = m[p]
	}
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("no object at %v", path)
	}
	return m
}

func sameKeys(t *testing.T, what string, props map[string]any, want []string) {
	t.Helper()
	got := slices.Sorted(maps.Keys(props))
	if w := slices.Sorted(slices.Values(want)); !slices.Equal(got, w) {
		t.Errorf("%s: the schema knows %v, the librarium %v", what, got, w)
	}
}

func TestSchemas_AreDraft2020AndNamedByTheirAddress(t *testing.T) {
	for _, name := range Names {
		compile(t, name)
		m := tree(t, name)
		if m["$schema"] != "https://json-schema.org/draft/2020-12/schema" || m["$id"] != URL(name) {
			t.Errorf("schema %s: $schema %v, $id %v", name, m["$schema"], m["$id"])
		}
		data, _ := For(name)
		disk, err := os.ReadFile(File(name))
		if err != nil || !bytes.Equal(data, disk) {
			t.Errorf("schema %s differs from %s (%v)", name, File(name), err)
		}
	}
	if _, ok := For("nonesuch"); ok {
		t.Error("For(nonesuch) found a schema")
	}
}

// TestRiteSchema_KnowsEveryKeyOfTheLibrarium: every key the librarium reads
// stands in the schema, and the schema knows no key the librarium does not.
func TestRiteSchema_KnowsEveryKeyOfTheLibrarium(t *testing.T) {
	root := tree(t, Rite)
	defs := obj(t, root, "$defs")
	m := obj(t, defs, "mark-i")
	sameKeys(t, "a rite", obj(t, m, "properties"), librarium.RiteKeys)
	sameKeys(t, "an inscription", obj(t, defs, "inscription", "properties"), librarium.InscriptionKeys)
	sameKeys(t, "the auspex", obj(t, defs, "auspex", "properties"), librarium.AuspexKeys)
	sameKeys(t, "a tome", obj(t, defs, "tome", "properties"), librarium.TomeKeys)
	objects := append([]string{"mark-i", "inscription", "auspex", "tome"}, markIIObjects(t, defs)...)
	var refs []string
	for _, k := range librarium.Kinds {
		sameKeys(t, "the step "+k.Key(), obj(t, defs, k.Key(), "properties"), librarium.StepKeys(k))
		objects = append(objects, k.Key())
		refs = append(refs, "#/$defs/"+k.Key())
	}
	var oneOf []string
	for _, b := range obj(t, defs, "step")["oneOf"].([]any) {
		oneOf = append(oneOf, b.(map[string]any)["$ref"].(string))
	}
	if !slices.Equal(oneOf, refs) {
		t.Errorf("a step is one of %v, want %v", oneOf, refs)
	}
	for name, def := range defs {
		if _, ok := def.(map[string]any)["properties"]; ok && !slices.Contains(objects, name) {
			t.Errorf("the schema's %q holds keys the librarium does not know", name)
		}
	}
	reserved := obj(t, m, "properties", "inscriptions", "propertyNames", "not")["enum"].([]any)
	if !slices.Equal(strs(reserved), librarium.ReservedRunes) {
		t.Errorf("reserved runes %v, want %v", reserved, librarium.ReservedRunes)
	}
	if got := strs(obj(t, defs, "vox-cast", "properties", "vox-cast")["enum"].([]any)); !slices.Equal(got,
		[]string{librarium.VoxProgress, librarium.VoxSuccess}) {
		t.Errorf("vox-cast tidings %v", got)
	}
	if got := obj(t, m, "properties", "pattern")["const"]; got != librarium.Pattern {
		t.Errorf("pattern %v", got)
	}
	if got := strs(obj(t, root, "properties", "pattern")["enum"].([]any)); !slices.Equal(got, librarium.Patterns) {
		t.Errorf("the patterns of a rite %v, want %v", got, librarium.Patterns)
	}
}

// markIIObjects holds the definitions of pattern Mark II to the keys the
// librarium reads, and returns the names of those that hold keys: each
// union of a subject, an observance and an edict's entry is told apart by
// its "designation" alone.
func markIIObjects(t *testing.T, defs map[string]any) []string {
	t.Helper()
	m := obj(t, defs, "mark-ii")
	sameKeys(t, "a rite of Mark II", obj(t, m, "properties"), librarium.MarkIIRiteKeys)
	if got := obj(t, m, "properties", "pattern")["const"]; got != librarium.PatternMarkII {
		t.Errorf("pattern %v", got)
	}
	if got := strs(obj(t, m, "properties", "vox-cast", "items")["enum"].([]any)); !slices.Equal(got,
		[]string{librarium.VoxProgress, librarium.VoxSuccess}) {
		t.Errorf("vox-cast tidings %v", got)
	}
	if reserved := obj(t, m, "properties", "inscriptions", "propertyNames", "not")["enum"].([]any); !slices.Equal(strs(reserved),
		librarium.ReservedRunes) {
		t.Errorf("reserved runes of Mark II %v, want %v", reserved, librarium.ReservedRunes)
	}
	sameKeys(t, "the auspex of a machine-spirit", obj(t, defs, "spiritAuspex", "properties"), librarium.SpiritAuspexKeys)
	objects := []string{"mark-ii", "subject", "observance", "entry", "spiritAuspex", "exaltedRelics"}
	unions := []struct {
		union, prefix string
		designations  []string
		common        []string
	}{
		{"subject", "subject-", librarium.SubjectDesignations, librarium.SubjectKeys},
		{"observance", "observance-", librarium.SpeechDesignations, librarium.ObservanceKeys},
		{"entry", "entry-", librarium.SpeechDesignations, []string{"designation"}},
	}
	for _, u := range unions {
		var refs, oneOf []string
		for _, d := range u.designations {
			name := u.prefix + d
			sameKeys(t, name, obj(t, defs, name, "properties"), slices.Concat(u.common, librarium.DesignationKeys(d)))
			if got := obj(t, defs, name, "properties", "designation")["const"]; got != d {
				t.Errorf("%s is told by the designation %v, want %q", name, got, d)
			}
			refs, objects = append(refs, "#/$defs/"+name), append(objects, name)
		}
		for _, b := range obj(t, defs, u.union)["oneOf"].([]any) {
			oneOf = append(oneOf, b.(map[string]any)["$ref"].(string))
		}
		if !slices.Equal(oneOf, refs) {
			t.Errorf("%s is one of %v, want %v", u.union, oneOf, refs)
		}
		if got := strs(obj(t, defs, u.union, "properties", "designation")["enum"].([]any)); !slices.Equal(got, u.designations) {
			t.Errorf("the designations of %s %v, want %v", u.union, got, u.designations)
		}
	}
	return objects
}

func TestSettingsSchema_KnowsEveryKeyOfTheLibrarium(t *testing.T) {
	m := tree(t, Settings)
	sameKeys(t, "the settings", obj(t, m, "properties"), librarium.SettingsKeys)
	if got := strs(obj(t, m, "properties", "vox")["enum"].([]any)); !slices.Equal(got, librarium.Voxes) {
		t.Errorf("vox %v, want %v", got, librarium.Voxes)
	}
	if got := strs(obj(t, m, "properties", "intercessor")["enum"].([]any)); !slices.Equal(got, librarium.Intercessors) {
		t.Errorf("intercessor %v, want %v", got, librarium.Intercessors)
	}
	if got := strs(obj(t, m, "properties", "pattern")["enum"].([]any)); !slices.Equal(got, librarium.Patterns) {
		t.Errorf("pattern %v, want %v", got, librarium.Patterns)
	}
}

func strs(vs []any) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i], _ = v.(string)
	}
	return out
}

// TestSchemas_EveryKeyBearsItsLore: every key and every definition carries
// a description for the faithful's editors to recite.
func TestSchemas_EveryKeyBearsItsLore(t *testing.T) {
	for _, name := range Names {
		m := tree(t, name)
		if d, _ := m["description"].(string); strings.TrimSpace(d) == "" {
			t.Errorf("schema %s bears no lore of its own", name)
		}
		for path, d := range lore(m, "") {
			if strings.TrimSpace(d) == "" {
				t.Errorf("schema %s: %s bears no lore", name, path)
			}
		}
	}
}

// lore gathers the description of every key and definition beneath v, by
// its path; a key without one yields "".
func lore(v any, path string) map[string]string {
	out := map[string]string{}
	switch v := v.(type) {
	case map[string]any:
		for k, child := range v {
			if k == "properties" || k == "$defs" {
				for name, s := range child.(map[string]any) {
					d, _ := s.(map[string]any)["description"].(string)
					out[path+"/"+k+"/"+name] = d
				}
			}
			maps.Copy(out, lore(child, path+"/"+k))
		}
	case []any:
		for i, child := range v {
			maps.Copy(out, lore(child, fmt.Sprintf("%s/%d", path, i)))
		}
	}
	return out
}

// TestExampleRites_AreFaithfulToSchemaAndLibrarium: the example rites — the
// ones shown to the faithful in examples/ and the ones kept for the trials —
// are pure in the eyes of the librarium and valid against the schema.
func TestExampleRites_AreFaithfulToSchemaAndLibrarium(t *testing.T) {
	s := compile(t, Rite)
	shown, _ := filepath.Glob("../examples/rites/*.json")
	kept, _ := filepath.Glob("testdata/rites/*.json")
	if len(shown) < 2 || len(kept) < 2 {
		t.Fatalf("too few example rites: %v, %v", shown, kept)
	}
	for _, path := range slices.Concat(shown, kept) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			r, fs := librarium.LoadFile(path)
			if r == nil || len(fs) != 0 {
				t.Fatalf("the librarium denounces the example:\n%v", fs)
			}
			if slices.Contains(shown, path) && r.Schema != URL(Rite) {
				t.Errorf("the example names the schema %q, not %q", r.Schema, URL(Rite))
			}
			data, _ := os.ReadFile(path)
			if err := validate(t, s, data); err != nil {
				t.Fatalf("the schema denounces the example: %v", err)
			}
		})
	}
	data, _ := os.ReadFile("testdata/servitor.json")
	if _, fs := librarium.ParseSettings("testdata/servitor.json", data); len(fs) != 0 {
		t.Fatalf("the librarium denounces the example settings:\n%v", fs)
	}
	if err := validate(t, compile(t, Settings), data); err != nil {
		t.Fatalf("the schema denounces the example settings: %v", err)
	}
}
