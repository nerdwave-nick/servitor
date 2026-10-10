package rituals

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// denounced returns the findings about rite of severity sev whose message
// holds every one of words.
func denounced(fs librarium.Findings, rite string, sev librarium.Severity, words ...string) librarium.Findings {
	var out librarium.Findings
next:
	for _, f := range fs {
		if f.Rite != rite || f.Severity != sev {
			continue
		}
		for _, w := range words {
			if !strings.Contains(f.Message, w) {
				continue next
			}
		}
		out = append(out, f)
	}
	return out
}

func inquire(t *testing.T, s *Servitor, spare bool, names ...string) librarium.Findings {
	t.Helper()
	fs, err := s.Inquire(names, spare)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		grimdark(t, f.Message)
		if f.Scripture == "" || f.Line < 1 || f.Column < 1 {
			t.Errorf("finding without place: %+v", f)
		}
	}
	return fs
}

func TestInquire_APureLibrariumIsAbsolved(t *testing.T) {
	fx := newFixture(t)
	fx.vessel("util.kdl", "input {}\n")
	fx.rite("mouse", mouse)
	if fs := inquire(t, fx.servitor(), false); len(fs) != 0 {
		t.Fatalf("findings:\n%v", fs)
	}
}

func TestInquire_TheScripturesOfRitesAndSettings(t *testing.T) {
	fx := newFixture(t)
	fx.vessel("util.kdl", "")
	fx.rite("mouse", mouse)
	fx.rite("bad", "{\n  \"pattern\": \"Mark I\",\n  \"aspects\": [\"on\"],\n  \"colour\": 1,\n  \"liturgy\": []\n}")
	fx.write(fx.lib, "servitor.json", `{"pattern": "Mark I", "vox": "loud"}`)
	fx.write(fx.lib, "tomes/hide.kdl", "hide {{aspect}}\nafter {{inscription.mood}}\n")
	fx.rite("tomes", `{"pattern": "Mark I", "aspects": ["on", "off"], "liturgy": [
	  {"sanctum": "$DATA/util.kdl", "ward": "lost", "scripture": {"on": {"tome": "../tomes/missing.kdl"}, "off": ""}},
	  {"sanctum": "$DATA/util.kdl", "ward": "lit", "scripture": {"tome": "../tomes/hide.kdl", "illuminate": true}}
	]}`)
	s := fx.servitor()

	fs := inquire(t, s, true)

	if got := denounced(fs, "bad", librarium.Heresy, `"colour"`); len(got) != 1 || got[0].Line != 4 {
		t.Errorf("unknown key: %v", got)
	}
	settings := denounced(fs, "", librarium.Heresy, "loud")
	if len(settings) != 1 || settings[0].Scripture != filepath.Join(fx.lib, "servitor.json") {
		t.Errorf("settings: %v", settings)
	}
	missing := denounced(fs, "tomes", librarium.Heresy, "missing.kdl")
	if len(missing) != 1 || missing[0].Line != 2 {
		t.Errorf("missing tome: %v", missing)
	}
	lit := denounced(fs, "tomes", librarium.Heresy, "mood")
	if len(lit) != 1 || lit[0].Scripture != filepath.Join(fx.lib, "tomes", "hide.kdl") || lit[0].Line != 2 || lit[0].Column != 7 {
		t.Errorf("placeholder within the tome: %v", lit)
	}
	if !fs.Heretical() {
		t.Error("heresy found, yet the findings are not heretical")
	}

	named := inquire(t, s, true, "mouse")
	if len(denounced(named, "bad", librarium.Heresy)) != 0 || len(denounced(named, "tomes", librarium.Heresy)) != 0 ||
		len(denounced(named, "", librarium.Heresy, "loud")) != 1 {
		t.Errorf("examining only mouse:\n%v", named)
	}
	var unrecorded *Unrecorded
	if _, err := s.Inquire([]string{"mouse", "nope"}, false); !errors.As(err, &unrecorded) || unrecorded.Name != "nope" {
		t.Errorf("unknown rite: %v", err)
	}
}

func TestInquire_TheMachineTheLiturgiesActUpon(t *testing.T) {
	fx := newFixture(t)
	fx.vessel("util.kdl", "input {}\n")
	if err := os.Chmod(fx.vessel("run.sh", "#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	fx.vessel("broken.kdl", "// +++ begin of sanctum broken -- aspect|on +++\nx\n")
	if err := os.MkdirAll(filepath.Join(fx.data, "themes", "dark"), 0o755); err != nil {
		t.Fatal(err)
	}
	fx.rite("mouse", mouse)
	fx.rite("missing", `{"pattern": "Mark I", "aspects": ["on"], "liturgy": [
	  {"sanctum": "$DATA/absent.kdl", "scripture": "x"},
	  {"sanctum": "$DATA/consecrated.kdl", "scripture": "x", "consecrate": true}]}`)
	fx.rite("broken", `{"pattern": "Mark I", "aspects": ["on"], "liturgy": [{"sanctum": "$DATA/broken.kdl", "scripture": "x"}]}`)
	fx.rite("tongues", `{"pattern": "Mark I", "aspects": ["on"], "auspex": {"rite": "echo on"}, "tongue": "no-such-tongue-of-men", "liturgy": [
	  {"incantation": "true", "tongue": "bash"},
	  {"incantation": "true"},
	  {"litany": "$DATA/run.sh"},
	  {"litany": "$DATA/run.sh", "reversion": "true"}]}`)
	fx.rite("theme", `{"pattern": "Mark I", "aspects": ["dark", "light"], "liturgy": [
	  {"tether": "$DATA/current", "anchor": "$DATA/themes/{{aspect}}"}]}`)
	s := fx.servitor()
	for _, p := range []Petition{{Rite: "mouse", Aspect: "on"}, {Rite: "theme", Aspect: "dark"}} {
		if res, err := s.Invoke(context.Background(), p); err != nil || res.Outcome.Verdict != "triumph" {
			t.Fatalf("%+v: %v %+v", p, err, res.Outcome)
		}
	}
	util := filepath.Join(fx.data, "util.kdl")
	if err := os.WriteFile(util, []byte(strings.Replace(fx.read(util), "400", "999", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	current := filepath.Join(fx.data, "current")
	if err := os.Remove(current); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(fx.data, "themes", "light"), current); err != nil {
		t.Fatal(err)
	}

	fs := inquire(t, s, false)

	expect := []struct {
		rite  string
		sev   librarium.Severity
		words []string
	}{
		{"missing", librarium.Impurity, []string{filepath.Join(fx.data, "absent.kdl"), `"consecrate"`}},
		{"broken", librarium.Heresy, []string{filepath.Join(fx.data, "broken.kdl")}},
		{"tongues", librarium.Impurity, []string{`"no-such-tongue-of-men"`, "auspex"}},
		{"tongues", librarium.Impurity, []string{`"no-such-tongue-of-men"`, "verse 2"}},
		{"tongues", librarium.Impurity, []string{`"no-such-tongue-of-men"`, "verse 4"}},
		{"theme", librarium.Impurity, []string{filepath.Join(fx.data, "themes", "light"), `"light"`}},
		{"theme", librarium.Impurity, []string{"desecrated", `"light"`, `"dark"`}},
		{"mouse", librarium.Impurity, []string{"tainted", util}},
	}
	for _, e := range expect {
		if got := denounced(fs, e.rite, e.sev, e.words...); len(got) != 1 {
			t.Errorf("%s %v %q: found %d in\n%v", e.rite, e.sev, e.words, len(got), fs)
		}
	}
	if n := len(denounced(fs, "missing", librarium.Impurity)); n != 1 {
		t.Errorf("a consecrating sanctum was denounced: %d", n)
	}
	if got := denounced(fs, "tongues", librarium.Impurity, "verse 2"); len(got) == 1 && got[0].Line != 1 {
		t.Errorf("the rite's tongue is denounced where the rite names it, not at %d:%d", got[0].Line, got[0].Column)
	}
	if n := len(denounced(fs, "tongues", librarium.Impurity)); n != 3 {
		t.Errorf("tongues: %d", n)
	}
	if len(fs) != len(expect) {
		t.Errorf("found %d, expected %d:\n%v", len(fs), len(expect), fs)
	}
	if spared := inquire(t, s, true); len(spared) != 0 {
		t.Fatalf("sparing the vessels:\n%v", spared)
	}
}
