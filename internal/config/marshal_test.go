package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestMarshal_RoundTripsAndOmitsDefaults(t *testing.T) {
	dir := t.TempDir()
	writeSwitch(t, dir, "rites/mouse-autohide-toggle.json", userExample)
	sw := Load(dir).Switches["mouse-autohide-toggle"]
	data, err := Marshal(sw)
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)
	for _, unwanted := range []string{`"guard"`, `"comment"`, `"create"`, `"description"`} {
		if strings.Contains(out, unwanted) {
			t.Errorf("default %s should be omitted:\n%s", unwanted, out)
		}
	}
	if !strings.Contains(out, "\"cursor {\",\n") || !strings.Contains(out, `"optional": false`) {
		t.Errorf("unexpected output:\n%s", out)
	}
	set := Preview(dir, Overlay{Name: sw.Name, Data: data, Replace: sw.Name})
	again := set.Switches[sw.Name]
	if again == nil {
		t.Fatalf("marshalled definition does not load: %v", set.Diags)
	}
	data2, _ := Marshal(again)
	if string(data2) != out {
		t.Fatalf("not stable:\n%s\n---\n%s", out, data2)
	}
}

func TestPreview_RenameAndClashDetection(t *testing.T) {
	dir := t.TempDir()
	body := `{"states":["a"],"files":[{"file":"/same","guard":"g","values":[{"state":"a","value":""}]}]}`
	writeSwitch(t, dir, "rites/old.json", body)
	set := Preview(dir, Overlay{Name: "new", Data: []byte(body), Replace: "old"})
	if set.Switches["new"] == nil || set.Switches["old"] != nil {
		t.Fatalf("rename preview: %v %v", set.Names(), set.Diags)
	}
	if got := set.Files["new"]; len(got) != 1 || got[0] != filepath.Join(dir, "rites", "new.json") {
		t.Fatalf("files = %v", got)
	}
	set = Preview(dir, Overlay{Name: "other", Data: []byte(body)})
	findDiag(t, set, `guard "g" for /same is used by both`)
	set = Preview(dir, Overlay{Name: "x", Data: []byte(`{`)})
	if !set.Broken["x"] || set.Switches["old"] == nil {
		t.Fatal("broken overlay must not affect other switches")
	}
}

func TestValidNameAndNewPath(t *testing.T) {
	if !ValidName("a-b.c_d") || ValidName("a b") || ValidName("") {
		t.Fatal("ValidName mismatch")
	}
	if NewPath("/c", "x") != "/c/rites/x.json" {
		t.Fatal(NewPath("/c", "x"))
	}
}
