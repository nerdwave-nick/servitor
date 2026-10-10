package librarium

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeScripture(t *testing.T, dir, rel, body string) string {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func sanctumRite(vessel, ward string) string {
	w := ""
	if ward != "" {
		w = `, "ward": "` + ward + `"`
	}
	return `{"pattern": "Mark I", "aspects": ["on"],
  "liturgy": [{"sanctum": "` + vessel + `"` + w + `, "scripture": ""}]}`
}

func findingsWith(fs Findings, substr string) Findings {
	var out Findings
	for _, f := range fs {
		if strings.Contains(f.Message, substr) {
			out = append(out, f)
		}
	}
	return out
}

func TestLoad_ReadsOnlyRites(t *testing.T) {
	dir := t.TempDir()
	writeScripture(t, dir, "rites/a.json", sanctumRite("~/a.kdl", ""))
	writeScripture(t, dir, "rites/b.jsonc", "// commented\n"+sanctumRite("~/b.kdl", ""))
	writeScripture(t, dir, "switches/c.json", sanctumRite("~/c.kdl", ""))
	writeScripture(t, dir, "rites/.hidden.json", "not even json")
	writeScripture(t, dir, "rites/notes.txt", "not a scripture")
	writeScripture(t, dir, "rites/nested/d.json", sanctumRite("~/d.kdl", ""))
	lib := Load(dir)
	if len(lib.Findings) != 0 {
		t.Fatalf("findings:\n%v", lib.Findings)
	}
	if got := lib.Names(); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("Names() = %v", got)
	}
	if r := lib.Rites["b"]; r == nil || r.Path != filepath.Join(dir, "rites", "b.jsonc") {
		t.Fatalf("rite b: %+v", r)
	}
	if !reflect.DeepEqual(lib.Scriptures["a"], []string{filepath.Join(dir, "rites", "a.json")}) {
		t.Fatalf("Scriptures: %v", lib.Scriptures)
	}
}

func TestLoad_MissingRitesIsEmpty(t *testing.T) {
	lib := Load(t.TempDir())
	if len(lib.Findings) != 0 || len(lib.Names()) != 0 {
		t.Fatalf("lib: %+v", lib)
	}
}

func TestLoad_HereticalRitesAreSetApart(t *testing.T) {
	dir := t.TempDir()
	writeScripture(t, dir, "rites/good.json", sanctumRite("~/a.kdl", ""))
	writeScripture(t, dir, "rites/bad.json", `{"pattern": "Mark I", "aspects": []}`)
	writeScripture(t, dir, "rites/old.json", `{"states": ["on"]}`)
	writeScripture(t, dir, "rites/impure.json", `{"pattern": "Mark I", "aspects": ["on"],
  "liturgy": [{"vox-cast": "success"}, {"incantation": "true"}]}`)
	lib := Load(dir)
	if got := lib.Names(); !reflect.DeepEqual(got, []string{"bad", "good", "impure", "old"}) {
		t.Fatalf("Names() = %v", got)
	}
	if lib.Rites["good"] == nil || lib.Rites["impure"] == nil || lib.Rites["bad"] != nil || lib.Rites["old"] != nil {
		t.Fatalf("usable rites: %v", lib.Rites)
	}
	if !lib.Heretical["bad"] || !lib.Heretical["old"] || lib.Heretical["impure"] || lib.Heretical["good"] {
		t.Fatalf("heretical: %v", lib.Heretical)
	}
	if len(lib.Rites["impure"].Liturgy) != 2 {
		t.Fatalf("a step after a success vox-cast must be kept: %v", lib.Rites["impure"].Liturgy)
	}
}

func TestLoad_TwoScripturesForOneRite(t *testing.T) {
	dir := t.TempDir()
	a := writeScripture(t, dir, "rites/theme.json", sanctumRite("~/a.kdl", ""))
	b := writeScripture(t, dir, "rites/theme.jsonc", sanctumRite("~/a.kdl", ""))
	lib := Load(dir)
	fs := findingsWith(lib.Findings, `the rite "theme" is recorded in more than one scripture`)
	if len(fs) != 2 || fs[0].Scripture != a || fs[1].Scripture != b || fs[0].Position != (Position{1, 1}) || fs[0].Severity != Heresy {
		t.Fatalf("findings:\n%v", lib.Findings)
	}
	if !lib.Heretical["theme"] || lib.Rites["theme"] != nil || len(lib.Scriptures["theme"]) != 2 {
		t.Fatalf("lib: %+v", lib)
	}
}

func TestLoad_WardUsedTwiceInOneVessel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := t.TempDir()
	writeScripture(t, dir, "rites/a.json", sanctumRite("~/x.kdl", "shared"))
	writeScripture(t, dir, "rites/b.json", sanctumRite("$HOME/x.kdl", "shared"))
	writeScripture(t, dir, "rites/c.json", sanctumRite("~/x.kdl", "own"))
	writeScripture(t, dir, "rites/d.json", sanctumRite("~/x.kdl", "e"))
	writeScripture(t, dir, "rites/e.json", sanctumRite("~/x.kdl", ""))
	writeScripture(t, dir, "rites/f.json", `{"pattern": "Mark I", "aspects": ["on"], "liturgy": [
  {"sanctum": "/y.kdl", "scripture": ""},
  {"sanctum": "/y.kdl", "scripture": ""}]}`)
	lib := Load(dir)
	clash := findingsWith(lib.Findings, "is claimed twice")
	var rites []string
	for _, f := range clash {
		rites = append(rites, f.Rite)
	}
	if !reflect.DeepEqual(rites, []string{"a", "b", "d", "e", "f", "f"}) {
		t.Fatalf("clashing rites %v in\n%v", rites, lib.Findings)
	}
	src := sanctumRite("~/x.kdl", "shared")
	if clash[0].Position != posOf(t, src, `"shared"`, 0) {
		t.Fatalf("clash points at %v", clash[0].Position)
	}
	if want := posOf(t, sanctumRite("~/x.kdl", ""), `"~/x.kdl"`, 0); clash[3].Position != want {
		t.Fatalf("clash of a default ward points at %v, want the vessel %v", clash[3].Position, want)
	}
	for _, name := range []string{"a", "b", "d", "e", "f"} {
		if !lib.Heretical[name] {
			t.Errorf("rite %s must be heretical", name)
		}
	}
	if lib.Heretical["c"] {
		t.Error("rite c keeps its own ward")
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	p := writeScripture(t, dir, "elsewhere/dnd.jsonc", sanctumRite("~/a.kdl", ""))
	r, fs := LoadFile(p)
	if len(fs) != 0 || r == nil || r.Name != "dnd" || r.Path != p {
		t.Fatalf("rite %+v, findings %v", r, fs)
	}
	r, fs = LoadFile(filepath.Join(dir, "absent.json"))
	if r != nil || len(fs) != 1 || !strings.Contains(fs[0].Message, "no scripture") || fs[0].Position != (Position{1, 1}) {
		t.Fatalf("rite %+v, findings %v", r, fs)
	}
}

func TestPreview_SubstitutesOneScripture(t *testing.T) {
	dir := t.TempDir()
	writeScripture(t, dir, "rites/a.json", sanctumRite("/x.kdl", "w"))
	writeScripture(t, dir, "rites/old.jsonc", sanctumRite("/y.kdl", "w"))
	lib := Preview(dir, Overlay{Name: "new", Data: []byte(sanctumRite("/x.kdl", "w")), Replace: "old"})
	if lib.Scriptures["old"] != nil || !lib.Heretical["new"] || !lib.Heretical["a"] {
		t.Fatalf("lib: %+v\n%v", lib, lib.Findings)
	}
	if p := lib.Scriptures["new"]; len(p) != 1 || p[0] != filepath.Join(dir, "rites", "new.json") {
		t.Fatalf("overlay path: %v", p)
	}
	if NewPath(dir, "x") != filepath.Join(dir, "rites", "x.json") {
		t.Fatal("NewPath")
	}
}

func TestLoad_DenunciationsSpeakGrimdark(t *testing.T) {
	dir := t.TempDir()
	writeScripture(t, dir, "rites/a.json", sanctumRite("/x.kdl", "w"))
	writeScripture(t, dir, "rites/b.json", sanctumRite("/x.kdl", "w"))
	writeScripture(t, dir, "rites/b.jsonc", sanctumRite("/x.kdl", "w"))
	writeScripture(t, dir, "rites/c.json", sanctumRite("/x.kdl", "w"))
	fs := Load(dir).Findings
	_, missing := LoadFile(filepath.Join(dir, "absent.json"))
	fs = append(fs, missing...)
	fs = append(fs, readHeresy("p", "r", os.ErrPermission), readHeresy("p", "r", os.ErrInvalid))
	if len(fs) != 7 {
		t.Fatalf("findings:\n%v", fs)
	}
	for _, f := range fs {
		if m := plainGlosses.FindString(f.Message); m != "" {
			t.Errorf("%q speaks the plain word %q", f.Message, m)
		}
	}
}
