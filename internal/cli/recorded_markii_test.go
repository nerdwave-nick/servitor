package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// rememberedJSON is a state rite whose smt is read by its scan from $DATA.
const rememberedJSON = `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
    {"name": "smt", "designation": "machine-spirit", "omens": {"on": "off", "*": "on"},
     "auspex": {"scan": "cat $DATA/smt"}, "enact": {"*": "echo {{omen}} > $DATA/smt"}}]}`

// remembered writes the rites "gaming" (rememberedJSON) and "firewall"
// (firewallJSON), whose machine lies in the returned place, and invokes
// gaming into on, so that its data-slate records the omen of smt.
func (e *env) remembered() (data string) {
	e.t.Helper()
	data = e.t.TempDir()
	e.addRite("rites/gaming.json", strings.ReplaceAll(rememberedJSON, "$DATA", data))
	e.addRite("rites/firewall.json", strings.ReplaceAll(firewallJSON, "$DATA", data))
	e.shows(data, "smt", "on\n")
	e.mustRun("invoke", "gaming", "on")
	return data
}

// recordedOmen writes the data-slate of rite as an invocation reverted in
// boot would have left it: in aspect ("" when unknown), its subject having
// shown omen at noon.
func (e *env) recordedOmen(rite, aspect, boot, subject, omen string) {
	e.t.Helper()
	noon := time.Date(2026, 10, 10, 13, 30, 0, 0, time.UTC)
	if err := augury.WriteSlate(os.LookupEnv, rite, augury.Slate{Aspect: aspect,
		LastRite: &augury.LastRite{Verdict: invocation.Reverted, At: noon}, Boot: boot,
		Omens: map[string]augury.Recorded{subject: {Omen: omen, At: noon}}}); err != nil {
		e.t.Fatal(err)
	}
}

// TestAugury_TellsTheRecordedOmenStale: where no live omen is heard, the
// augury tells the omen the data-slate recorded, marked stale with the
// moment it was read: under --spare-auspex, and always for a privileged
// auspex, which is never spoken. Recorded in another boot it is ignored.
func TestAugury_TellsTheRecordedOmenStale(t *testing.T) {
	e := newEnv(t)
	data := e.remembered()
	slate := e.slateOf("gaming")
	at := slate.Omens["smt"].At.Format(time.RFC3339)

	got := compactJSON(t, e.mustRun("augury", "gaming", "--spare-auspex"))

	want := compactJSON(t, `{"rite": "gaming", "edict": false, "aspect": "on", "standing": "performed",
	  "sanction": "", "auspex": "forgone", "desecrated": false, "inscriptions": {}, "canon": [],
	  "last_rite": {"verdict": "triumph", "at": "`+slate.LastRite.At.Format(time.RFC3339)+`"}, "fitting": ["on"],
	  "omens": [{"subject": "smt", "designation": "machine-spirit", "omen": "off", "aspects": ["on"], "aspect": "on",
	             "stale": true, "read_at": "`+at+`"}],
	  "taint": []}`)
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	if got := compactJSON(t, e.mustRun("augury", "gaming")); strings.Contains(got, "stale") ||
		!strings.Contains(got, `"omen":"off","aspects":["on"],"aspect":"on"}`) {
		t.Errorf("a live augury: %s", got)
	}

	e.recordedOmen("firewall", "on", augury.Boot(), "nft", "on")
	if got := compactJSON(t, e.mustRun("augury", "firewall")); !strings.Contains(got, `"auspex":"",`) ||
		!strings.Contains(got, `"fitting":["on"],"omens":[{"subject":"nft","designation":"machine-spirit","omen":"on",`+
			`"aspects":["on"],"aspect":"on","stale":true,"read_at":"2026-10-10T13:30:00Z"}]`) {
		t.Errorf("augury firewall: %s", got)
	}
	e.recordedOmen("firewall", "on", "another-boot", "nft", "on")
	if got := compactJSON(t, e.mustRun("augury", "firewall")); !strings.Contains(got, `"fitting":[],"omens":[]`) {
		t.Errorf("augury firewall, recorded in another boot: %s", got)
	}
	e.recordedOmen("firewall", "on", augury.Boot(), "nft", "auto")
	if got := compactJSON(t, e.mustRun("augury", "firewall")); !strings.Contains(got, `"fitting":[],"omens":[]`) {
		t.Errorf("augury firewall, an omen no longer declared: %s", got)
	}
	if _, err := os.Stat(filepath.Join(data, "spoken")); err == nil {
		t.Fatal("a privileged auspex was spoken")
	}
}

// TestCensus_MarksWhatRestsOnStaleOmens: every census entry bears "stale",
// true when its aspect or standing rests on a stale omen; the table marks
// such an aspect with "~".
func TestCensus_MarksWhatRestsOnStaleOmens(t *testing.T) {
	e := newEnv(t)
	e.remembered()
	e.recordedOmen("firewall", "on", augury.Boot(), "nft", "on")

	spared, live := e.census("--spare-auspex"), e.census()

	for rite, want := range map[string]bool{"gaming": true, "firewall": true, "mouse-autohide-toggle": false} {
		if got, ok := spared[rite]["stale"]; !ok || got != want {
			t.Errorf("spared census of %s: stale %v (%v), want %v", rite, got, ok, want)
		}
	}
	for rite, want := range map[string]bool{"gaming": false, "firewall": true, "mouse-autohide-toggle": false} {
		if got, ok := live[rite]["stale"]; !ok || got != want {
			t.Errorf("census of %s: stale %v (%v), want %v", rite, got, ok, want)
		}
	}
	rows := map[string][]string{}
	for _, line := range lines(e.mustRun("census", "--spare-auspex")) {
		fields := strings.Fields(line)
		if fields[0] == "firewall" && fields[1] == sanctionGlyph(sanction.Unsanctioned) {
			// the exalted subject of firewall stands unsanctioned
			fields = slices.Delete(fields, 1, 2)
		}
		rows[fields[0]] = fields
	}
	if rows["gaming"][1] != "~on" || rows["firewall"][1] != "~on" || rows["mouse-autohide-toggle"][1] != "·" {
		t.Errorf("the table: %v", rows)
	}
	for _, line := range lines(e.mustRun("census")) {
		if fields := strings.Fields(line); fields[0] == "gaming" && fields[1] != "on" {
			t.Errorf("a live aspect is marked: %v", fields)
		}
	}
}

// TestVigil_FirstTellsTheRecordedOmens: the first heralding of a state rite
// tells the aspect its recorded omens give, awaited and stale; the second
// tells it from the live omens, and a privileged auspex stays stale in
// both.
func TestVigil_FirstTellsTheRecordedOmens(t *testing.T) {
	e := newEnv(t)
	e.shows(e.remembered(), "smt", "on\n") // other hands: smt now fits off
	e.recordedOmen("firewall", "", augury.Boot(), "nft", "on")
	census := e.census()

	lines := e.vigil()

	var told []string
	for _, h := range lines[:len(lines)-1] {
		if r := h.line["rite"]; r == "gaming" || r == "firewall" {
			told = append(told, strings.Join([]string{r.(string), h.line["auspex"].(string), h.line["aspect"].(string),
				h.line["standing"].(string), map[bool]string{true: "stale", false: "live"}[h.line["stale"].(bool)]}, " "))
		}
	}
	want := []string{"firewall  on performed stale", "gaming awaited on performed stale",
		"gaming heeded off desecrated live"}
	if strings.Join(told, "\n") != strings.Join(want, "\n") {
		t.Errorf("told\n%s\nwant\n%s", strings.Join(told, "\n"), strings.Join(want, "\n"))
	}
	if last := settle(t, lines); last["gaming"]["stale"] != false || last["firewall"]["stale"] != true ||
		last["gaming"]["auspex"] != census["gaming"]["auspex"] || last["firewall"]["aspect"] != census["firewall"]["aspect"] {
		t.Errorf("the vigil settled on %v, the census %v", last, census)
	}
}

// TestCompletion_MarksTheCurrentAspectFromRecordedOmens: completion
// awakens no auspex, yet marks the current aspect the recorded omens give.
func TestCompletion_MarksTheCurrentAspectFromRecordedOmens(t *testing.T) {
	e := newEnv(t)
	data := e.remembered()
	e.recordedOmen("gaming", "", augury.Boot(), "smt", "off") // the aspect is not recorded: the omen alone tells it
	e.recordedOmen("firewall", "", augury.Boot(), "nft", "off")

	for rite, current := range map[string]string{"gaming": "on", "firewall": "off"} {
		out := e.mustRun("__complete", "invoke", rite, "")
		if !strings.Contains(out, current+"\tcurrent aspect") {
			t.Errorf("completion of %s marks no current aspect %s:\n%s", rite, current, out)
		}
	}
	e.recordedOmen("firewall", "", "another-boot", "nft", "off")
	if out := e.mustRun("__complete", "invoke", "firewall", ""); strings.Contains(out, "current aspect") {
		t.Errorf("completion took an omen of another boot:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(data, "spoken")); err == nil {
		t.Fatal("a privileged auspex was spoken")
	}
}

// slateOf reads the data-slate of rite.
func (e *env) slateOf(rite string) *augury.Slate {
	e.t.Helper()
	s, err := augury.ReadSlate(os.LookupEnv, rite)
	if err != nil || s == nil {
		e.t.Fatalf("the data-slate of %s: %+v %v", rite, s, err)
	}
	return s
}
