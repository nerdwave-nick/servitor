package librarium

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	markISettings = `// the standing orders of the faithful
{
  "pattern": "Mark I", // the elder pattern
  "vox": "off",
}
`
	markIRite = `{"pattern": "Mark I", "aspects": ["on", "off"], "liturgy": [{"incantation": "echo {{aspect}}"}]}`
)

// librariumOf writes files (relative path → scripture) into a new
// Librarium.
func librariumOf(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestLoad_ConvertsInMemory: the Librarium read converting holds every
// pure scripture of pattern Mark I converted, rites and settings alike,
// yet writes nothing; heretical scripture stays as read, and its heresy
// tells that it is converted once purged.
func TestLoad_ConvertsInMemory(t *testing.T) {
	dir := librariumOf(t, map[string]string{
		SettingsName:         markISettings,
		"rites/plain.json":   markIRite,
		"rites/fallen.jsonc": `{"pattern": "Mark I", "aspects": ["on"], "liturgy": [{"incantation": 3}]}`,
		"rites/new.json":     `{"pattern": "Mark II", "edict": true, "liturgy": [{"designation": "incantation", "words": "x"}]}`,
	})
	lib := Load(dir)
	if r := lib.Rites["plain"]; r == nil || !r.MarkII || r.Subjects[0].Name != "incantation-1" {
		t.Fatalf("the rite of Mark I is not converted in memory: %+v %v", r, lib.Findings)
	}
	if !lib.Settings.MarkII || lib.Settings.Vox != VoxOff {
		t.Errorf("the settings are not converted in memory: %+v", lib.Settings)
	}
	if !lib.Heretical["fallen"] || !strings.Contains(fmt.Sprint(lib.Findings), "is converted once its heresies are purged") {
		t.Errorf("the heretical rite does not await its purging: %v", lib.Findings)
	}
	var converted []string
	for _, c := range lib.Conversions {
		converted = append(converted, c.Rite)
	}
	if fmt.Sprint(converted) != "[ plain]" {
		t.Errorf("conversions %q, want the settings and plain", converted)
	}
	if read(t, filepath.Join(dir, "rites/plain.json")) != markIRite || read(t, SettingsPath(dir)) != markISettings {
		t.Error("a conversion in memory wrote the Librarium")
	}
	if _, err := os.Stat(filepath.Join(dir, RitesDir, MarkIDir)); err == nil {
		t.Error("a conversion in memory kept a copy")
	}
}

// TestConversion_Write: writing a conversion keeps the original in the
// .mark-i directory beside it and writes the converted scripture in its
// place, its mode kept; the settings keep their comments, only their
// pattern rewritten.
func TestConversion_Write(t *testing.T) {
	dir := librariumOf(t, map[string]string{SettingsName: markISettings, "rites/plain.json": markIRite})
	lib := Load(dir)
	for _, c := range lib.Conversions {
		if err := c.Write(); err != nil {
			t.Fatalf("%q: %v", c.Rite, err)
		}
		if read(t, c.Copy()) != string(c.Original) || read(t, c.Path) != string(c.Data) {
			t.Errorf("%q: the copy or the converted scripture is not what was read and converted", c.Rite)
		}
		if fi, err := os.Stat(c.Path); err != nil || fi.Mode().Perm() != 0o640 {
			t.Errorf("%q: the mode of the scripture is not kept: %v %v", c.Rite, fi.Mode(), err)
		}
	}
	if got, want := read(t, SettingsPath(dir)), strings.Replace(markISettings, `"Mark I"`, `"Mark II"`, 1); got != want {
		t.Errorf("the settings are rewritten as:\n%s", got)
	}
	if read(t, filepath.Join(dir, MarkIDir, SettingsName)) != markISettings {
		t.Error("the copy of the settings is not kept beside them")
	}
	again := Load(dir)
	if len(again.Conversions) != 0 || !again.Rites["plain"].MarkII || !again.Settings.MarkII {
		t.Errorf("the Librarium converted is converted anew: %v", again.Findings)
	}
	entries, _ := os.ReadDir(filepath.Join(dir, RitesDir))
	if len(entries) != 2 {
		t.Errorf("the rites hold more than the scripture and its copy: %v", entries)
	}
}

// TestConversion_NeverOverwritesACopy: a copy that differs from the
// original is never overwritten, and then nothing is written; a copy
// already holding the original lets the conversion proceed. A scripture
// changed since it was read is not written over.
func TestConversion_NeverOverwritesACopy(t *testing.T) {
	dir := librariumOf(t, map[string]string{"rites/plain.json": markIRite, "rites/.mark-i/plain.json": "elder"})
	c := Load(dir).Conversions[0]
	if err := c.Write(); err == nil || !strings.Contains(err.Error(), "never overwritten") {
		t.Errorf("the copy was overwritten: %v", err)
	}
	if read(t, c.Copy()) != "elder" || read(t, c.Path) != markIRite {
		t.Error("a refused conversion touched the Librarium")
	}
	if err := os.WriteFile(c.Copy(), []byte(markIRite), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := c.Write(); err != nil || read(t, c.Path) != string(c.Data) {
		t.Errorf("a copy holding the original stayed the conversion: %v", err)
	}
	dir = librariumOf(t, map[string]string{"rites/plain.json": markIRite})
	c = Load(dir).Conversions[0]
	if err := os.WriteFile(c.Path, []byte(markIRite+" "), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := c.Write(); err == nil || read(t, c.Path) != markIRite+" " {
		t.Errorf("a scripture changed since its reading was written over: %v", err)
	}
}

// TestConversion_WritesThroughATether: a scripture tethered into the
// Librarium is converted where it lies, and the tether stays.
func TestConversion_WritesThroughATether(t *testing.T) {
	dir := librariumOf(t, map[string]string{})
	elsewhere := filepath.Join(t.TempDir(), "plain.json")
	if err := os.WriteFile(elsewhere, []byte(markIRite), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, RitesDir, "plain.json")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, link); err != nil {
		t.Fatal(err)
	}
	c := Load(dir).Conversions[0]
	if err := c.Write(); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the tether was broken: %v", err)
	}
	if !bytes.Equal([]byte(read(t, elsewhere)), c.Data) || read(t, c.Copy()) != markIRite {
		t.Error("the scripture is not converted where it lies, or its copy is not kept")
	}
}

// TestLoad_HereticalSettingsStayUnconverted: heretical settings of pattern
// Mark I are not converted, and their heresy tells that they are once
// purged.
func TestLoad_HereticalSettingsStayUnconverted(t *testing.T) {
	dir := librariumOf(t, map[string]string{SettingsName: `{"pattern": "Mark I", "vox": "loud"}`})
	lib := Load(dir)
	if lib.Settings.MarkII || len(lib.Conversions) != 0 {
		t.Errorf("heretical settings are converted: %+v", lib.Settings)
	}
	if !strings.Contains(fmt.Sprint(lib.Findings), "converted once their heresies are purged") {
		t.Errorf("the heresy does not tell the conversion awaited: %v", lib.Findings)
	}
}

// TestPreview_ConvertsInMemory: the preview of a scripture reads every
// other scripture of pattern Mark I converted, and writes nothing.
func TestPreview_ConvertsInMemory(t *testing.T) {
	dir := librariumOf(t, map[string]string{"rites/plain.json": markIRite})
	lib := Preview(dir, Overlay{Name: "new", Data: []byte(`{"pattern": "Mark II", "edict": true,
		"liturgy": [{"designation": "incantation", "words": "x"}]}`)})
	if r := lib.Rites["plain"]; r == nil || !r.MarkII || r.Subjects[0].Name != "incantation-1" {
		t.Fatalf("the rite beside the preview is not converted: %+v\n%v", r, lib.Findings)
	}
	if got := read(t, filepath.Join(dir, "rites", "plain.json")); got != markIRite {
		t.Errorf("the preview wrote the conversion:\n%s", got)
	}
}
