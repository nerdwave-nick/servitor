package rituals

import (
	"errors"
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

// TestInquire_TheScripturesOfRitesAndSettings: the inquisition denounces
// heresy in the scripture of a rite and in the settings, each at its
// place; examining one rite by name still examines the settings beside it,
// and a rite the Librarium does not record is no rite to examine.
func TestInquire_TheScripturesOfRitesAndSettings(t *testing.T) {
	fx := newFixture(t)
	fx.vessel("util.kdl", "")
	fx.rite("mouse", mouse)
	fx.rite("bad", "{\n  \"pattern\": \"Mark II\",\n  \"aspects\": [\"on\"],\n  \"colour\": 1,\n"+
		"  \"subjects\": [{\"name\": \"s\", \"designation\": \"machine-spirit\", \"enact\": {\"*\": \"true\"}}]\n}")
	fx.write(fx.lib, "servitor.json", `{"pattern": "Mark II", "vox": "loud"}`)
	s := fx.servitor()

	fs := inquire(t, s, true)

	if got := denounced(fs, "bad", librarium.Heresy, `"colour"`); len(got) != 1 || got[0].Line != 4 {
		t.Errorf("unknown key: %v", got)
	}
	settings := denounced(fs, "", librarium.Heresy, "loud")
	if len(settings) != 1 || settings[0].Scripture != filepath.Join(fx.lib, "servitor.json") {
		t.Errorf("settings: %v", settings)
	}
	if !fs.Heretical() {
		t.Error("heresy found, yet the findings are not heretical")
	}

	named := inquire(t, s, true, "mouse")
	if len(denounced(named, "bad", librarium.Heresy)) != 0 || len(denounced(named, "", librarium.Heresy, "loud")) != 1 {
		t.Errorf("examining only mouse:\n%v", named)
	}
	var unrecorded *Unrecorded
	if _, err := s.Inquire([]string{"mouse", "nope"}, false); !errors.As(err, &unrecorded) || unrecorded.Name != "nope" {
		t.Errorf("unknown rite: %v", err)
	}
}

// TestInquire_TheMachineAnEdictActsUponIsExaminedOnce: an edict brings the
// machine into no aspect, so what it acts upon is examined once, with the
// decrees of its inscriptions: a scroll nowhere to be found, and tongues
// not upon the path, each named by its verse. A scroll that stands but may
// not be executed is recited through its tongue, and is not denounced.
func TestInquire_TheMachineAnEdictActsUponIsExaminedOnce(t *testing.T) {
	fx := newFixture(t)
	fx.vessel("plain.sh", "true\n") // stands, but may not be executed: recited through its tongue
	capture := fx.rite("capture", `{"pattern": "Mark II", "edict": true,
	  "inscriptions": {"place": {"purpose": "where", "decrees": "lost"}},
	  "vox-cast": ["success"],
	  "liturgy": [
	  {"designation": "litany", "scroll": "$DATA/{{inscription.place}}.sh"},
	  {"designation": "litany", "scroll": "$DATA/plain.sh", "tongue": "no-such-tongue-of-edicts"},
	  {"designation": "incantation", "words": "true", "tongue": "no-such-tongue-of-edicts"}]}`)
	fx.rite("whole", `{"pattern": "Mark II", "edict": true, "liturgy": [{"designation": "litany", "scroll": "$DATA/plain.sh"}]}`)

	fs := inquire(t, fx.servitor(), false)

	expectFindings(t, fs, []sought{
		{"capture", librarium.Impurity, []string{filepath.Join(fx.data, "lost.sh"), "scroll", "verse 1"}, capture,
			`"` + fx.data + `/{{inscription.place}}.sh"`},
	})
	tongues := denounced(fs, "capture", librarium.Impurity, `"no-such-tongue-of-edicts"`)
	if len(tongues) != 2 || len(denounced(tongues, "capture", librarium.Impurity, "verse 2")) != 1 ||
		len(denounced(tongues, "capture", librarium.Impurity, "verse 3")) != 1 {
		t.Errorf("the missing tongues of an edict: %v\nin\n%v", tongues, fs)
	}
	if len(fs) != 3 {
		t.Errorf("found %d, expected 3:\n%v", len(fs), fs)
	}
	if spared := inquire(t, fx.servitor(), true); len(spared) != 0 {
		t.Fatalf("sparing the vessels:\n%v", spared)
	}
}
