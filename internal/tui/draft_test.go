package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/config"
)

const sampleRite = `{
  "description": "Hide the cursor",
  "states": ["on", "off"],
  "files": [{
    "file": "/tmp/x.kdl",
    "guard": "cursor",
    "values": [
      {"state": "on", "value": "a\nb", "meta": {"mode": "hide"}},
      {"state": "off", "value": "", "meta": {"mode": "show"}}
    ],
    "meta": {"reason": {"description": "why"}, "mode": {"optional": false}}
  }]
}`

func loadSample(t *testing.T, name, body string) (*config.Set, string) {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "rites", name+".json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	set := config.Load(dir)
	if set.Diags.HasErrors() {
		t.Fatalf("sample invalid: %v", set.Diags)
	}
	return set, dir
}

func TestDraft_RoundTrip(t *testing.T) {
	set, _ := loadSample(t, "mouse", sampleRite)
	sw := set.Switches["mouse"]
	d := draftFrom(sw)
	if d.states != "on, off" || d.vessels[0].guard != "cursor" || d.vessels[0].comment != "" {
		t.Fatalf("draft = %+v", d)
	}
	if d.vessels[0].inscriptions != "mode!, reason" {
		t.Fatalf("inscriptions = %q", d.vessels[0].inscriptions)
	}
	want, _ := config.Marshal(sw)
	got, err := config.Marshal(d.toSwitch())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("round trip differs:\n%s\n---\n%s", got, want)
	}
}

func TestDraft_RenameKeepsDefaultGuard(t *testing.T) {
	d := draft{name: "new", states: "a", vessels: []vessel{newVessel()}}
	d.vessels[0].file = "/f.lua"
	sw := d.toSwitch()
	if sw.Files[0].Guard != "new" || sw.Files[0].Comment != "--" {
		t.Fatalf("defaults not applied: %+v", sw.Files[0])
	}
}

func TestParseStatesAndInscriptions(t *testing.T) {
	if s, err := parseStates(" on,off  auto "); err != nil || len(s) != 3 {
		t.Fatalf("%v %v", s, err)
	}
	for _, bad := range []string{"", "on, on", "b@d"} {
		if _, err := parseStates(bad); err == nil {
			t.Errorf("parseStates(%q) should fail", bad)
		}
	}
	keys, req, err := parseInscriptions("reason, mode!")
	if err != nil || len(keys) != 2 || !req["mode"] || req["reason"] {
		t.Fatalf("%v %v %v", keys, req, err)
	}
	for _, bad := range []string{"state", "a, a", "b@d"} {
		if _, _, err := parseInscriptions(bad); err == nil {
			t.Errorf("parseInscriptions(%q) should fail", bad)
		}
	}
}
