package librarium

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "write the golden conversions anew")

// TestConvert_Golden: every example rite and every rite of the owner,
// written in pattern Mark I, converts to the golden scripture of pattern
// Mark II beside it (its comments naming what was struck), which reads as
// the rite converted, free of heresy; the tidings of each are golden too.
func TestConvert_Golden(t *testing.T) {
	scriptures, err := filepath.Glob(filepath.Join("testdata", "conversion", "*", "*.json"))
	if err != nil || len(scriptures) < 7 {
		t.Fatalf("the fixtures of conversion are missing: %v %v", scriptures, err)
	}
	for _, path := range scriptures {
		t.Run(strings.TrimPrefix(path, filepath.Join("testdata", "conversion")+"/"), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			name := strings.TrimSuffix(filepath.Base(path), ".json")
			at := filepath.Join("/librarium", RitesDir, name+".json")
			r, fs, c := parseConverting(name, at, data)
			if c == nil {
				t.Fatalf("the rite was not converted: %v", fs)
			}
			if fs.Heretical() || r == nil || r.markI {
				t.Fatalf("the converted rite is not a pure rite of Mark II: %v", fs)
			}
			if string(c.Original) != string(data) || c.Path != at || c.Rite != name {
				t.Errorf("the conversion forgets what it converted: %q %q", c.Path, c.Rite)
			}
			golden(t, strings.TrimSuffix(path, ".json")+".mark-ii", c.Data)
			golden(t, strings.TrimSuffix(path, ".json")+".tidings", []byte(c.Tidings()+"\n"))
			back, fs := parse(name, at, c.Data)
			if fs.Heretical() || !back.Equal(r) {
				t.Errorf("the converted scripture does not read back as the rite converted: %v", fs)
			}
		})
	}
}

// golden compares got with the golden file path, or writes it under -update.
func golden(t *testing.T, path string, got []byte) {
	t.Helper()
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("%s drifts:\n--- got\n%s\n--- want\n%s", path, got, want)
	}
}
