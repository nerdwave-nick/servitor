package cli

import (
	"bytes"
	stdjson "encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// powerJSON is a state rite of three aspects whose smt shares its omen
// between two of them and whose profile names its aspect; $DATA holds the
// machine's state files.
const powerJSON = `{"pattern": "Mark II", "purpose": "the power of the machine",
  "aspects": ["performance", "balanced", "powersave"],
  "subjects": [
    {"name": "smt", "designation": "machine-spirit", "omens": {"performance": "off", "*": "on"},
     "auspex": {"scan": "cat $DATA/smt"}, "enact": {"*": "true"}},
    {"name": "profile", "designation": "machine-spirit",
     "auspex": {"trials": {"*": "grep -qx {{omen}} $DATA/profile"}}, "enact": {"*": "true"}},
    {"name": "flag", "designation": "transcription", "vessel": "$DATA/flag",
     "omens": {"performance": "raised", "*": null}}]}`

// firewallJSON is a state rite whose privileged auspex, were it ever
// spoken, would leave the mark $DATA/spoken.
const firewallJSON = `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
    {"name": "nft", "designation": "machine-spirit", "exalted": true,
     "auspex": {"exalted": true, "scan": "touch $DATA/spoken; echo on", "trials": {"*": "touch $DATA/spoken"}},
     "enact": {"*": "true"}}]}`

// power writes the rites "power" and "firewall", whose machine lies in the
// returned place.
func (e *env) power() (data string) {
	e.t.Helper()
	data = e.t.TempDir()
	e.addRite("rites/power.json", strings.ReplaceAll(powerJSON, "$DATA", data))
	e.addRite("rites/firewall.json", strings.ReplaceAll(firewallJSON, "$DATA", data))
	return data
}

// shows sets what the machine shows of the state file name in data.
func (e *env) shows(data, name, content string) {
	e.t.Helper()
	if err := os.WriteFile(filepath.Join(data, name), []byte(content), 0o644); err != nil {
		e.t.Fatal(err)
	}
}

// recorded writes the data-slate of rite as an invocation into aspect
// that ended in triumph at noon would have left it.
func (e *env) recorded(rite, aspect string) {
	e.t.Helper()
	at := time.Date(2026, 10, 10, 13, 30, 0, 0, time.UTC)
	if err := augury.WriteSlate(os.LookupEnv, rite, augury.Slate{Aspect: aspect,
		LastRite: &augury.LastRite{Verdict: invocation.Triumph, At: at}}); err != nil {
		e.t.Fatal(err)
	}
}

func compactJSON(t *testing.T, s string) string {
	t.Helper()
	var b bytes.Buffer
	if err := stdjson.Compact(&b, []byte(s)); err != nil {
		t.Fatalf("%v: %s", err, s)
	}
	return b.String()
}

// TestAugury_ReadsTheSubjectsOfAStateRite: the binharic augury of a state
// rite names each omen by its subject, tells the rite's fitting aspects and
// agrees by intersection; its signs answer alike, and its privileged
// auspex is never spoken.
func TestAugury_ReadsTheSubjectsOfAStateRite(t *testing.T) {
	e := newEnv(t)
	data := e.power()
	e.shows(data, "smt", "on\n")
	e.shows(data, "profile", "balanced\n")
	e.recorded("power", "powersave")

	got := compactJSON(t, e.mustRun("augury", "power"))

	want := compactJSON(t, `{"rite": "power", "edict": false, "aspect": "balanced", "standing": "desecrated",
	  "sanction": "", "auspex": "heeded", "desecrated": true, "former": "powersave", "inscriptions": {},
	  "canon": [], "last_rite": {"verdict": "triumph", "at": "2026-10-10T13:30:00Z"}, "fitting": ["balanced"],
	  "omens": [
	    {"subject": "smt", "designation": "machine-spirit", "omen": "on", "aspects": ["balanced", "powersave"]},
	    {"subject": "profile", "designation": "machine-spirit", "omen": "balanced", "aspects": ["balanced"],
	     "aspect": "balanced"},
	    {"subject": "flag", "designation": "transcription", "aspects": ["balanced", "powersave"]}],
	  "taint": []}`)
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	if got := e.mustRun("augury", "power", "aspect"); got != "balanced\n" {
		t.Errorf("augury power aspect = %q", got)
	}
	if _, _, code := e.run("augury", "power", "--is", "balanced"); code != 0 {
		t.Errorf("augury power --is balanced: exit %d", code)
	}

	// the hidden wrong subject: smt shows the omen of two aspects, the
	// profile the third
	e.shows(data, "profile", "performance\n")
	if got := e.mustRun("augury", "power", "standing"); got != "corrupted\n" {
		t.Errorf("augury power standing = %q, want corrupted", got)
	}
	e.recorded("firewall", "on")
	if got := compactJSON(t, e.mustRun("augury", "firewall")); !strings.Contains(got, `"aspect":"on","standing":"performed",`) ||
		!strings.Contains(got, `"auspex":"",`) || !strings.Contains(got, `"fitting":[],"omens":[]`) {
		t.Errorf("augury firewall: %s", got)
	}
	if _, err := os.Stat(filepath.Join(data, "spoken")); err == nil {
		t.Fatal("a privileged auspex was spoken")
	}
}

// TestCensus_ReadsStateRites: the census of a state rite keeps the keys of
// every entry; spared, it awakens no auspex and answers from the built-in
// subjects and the data-slate, and completion does likewise.
func TestCensus_ReadsStateRites(t *testing.T) {
	e := newEnv(t)
	data := e.power()
	e.shows(data, "smt", "on\n")
	e.shows(data, "profile", "powersave\n")

	var entries []map[string]any
	if err := stdjson.Unmarshal([]byte(e.mustRun("census", "--binharic")), &entries); err != nil {
		t.Fatal(err)
	}
	got, _ := stdjson.Marshal(entries[2])
	want := compactJSON(t, `{"aspect": "powersave", "aspects": ["performance", "balanced", "powersave"], "auspex": "heeded",
	  "canon": [], "desecrated": false, "edict": false, "last_rite": null, "purpose": "the power of the machine",
	  "recorded_in": "`+filepath.Join(e.cfgDir, "rites", "power.json")+`", "rite": "power", "sanction": "",
	  "standing": "performed"}`)
	if string(got) != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}

	if err := os.Remove(filepath.Join(data, "smt")); err != nil {
		t.Fatal(err)
	}
	e.recorded("power", "balanced")
	data2 := e.census("--spare-auspex")["power"]
	if data2["auspex"] != "forgone" || data2["aspect"] != "balanced" || data2["standing"] != "performed" {
		t.Errorf("spared: %v", data2)
	}
	values, _ := e.complete("invoke", "power", "")
	if !reflect.DeepEqual(values, []string{"performance", "balanced", "powersave"}) {
		t.Errorf("completion %v", values)
	}
	out := e.mustRun("__complete", "invoke", "power", "")
	if !strings.Contains(out, "balanced\tcurrent aspect") {
		t.Errorf("completion marks no current aspect:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(data, "spoken")); err == nil {
		t.Fatal("a privileged auspex was spoken")
	}
	if e.census()["power"]["auspex"] != "mute" {
		t.Error("an unspared census of a vanished smt is not mute")
	}
}

// TestCensus_BoundCountsAuspexProcesses: the census keeps at most sixteen
// processes of auspexes awake at once, every trial counting as one, so six
// rites of three trials labouring a second each are read in two seconds.
func TestCensus_BoundCountsAuspexProcesses(t *testing.T) {
	e := newEnv(t)
	awake := filepath.Join(t.TempDir(), "awake")
	if err := os.Mkdir(awake, 0o755); err != nil {
		t.Fatal(err)
	}
	counted := filepath.Join(t.TempDir(), "counted")
	trial := fmt.Sprintf(`touch %[1]s/$$; ls %[1]s | wc -l >> %[2]s; sleep 1; rm %[1]s/$$; [ {{omen}} = on ]`, awake, counted)
	const n = 6
	for i := range n {
		e.addRite(fmt.Sprintf("rites/trial%d.json", i), `{"pattern": "Mark II", "aspects": ["on", "off", "auto"],
  "subjects": [{"name": "s", "designation": "machine-spirit", "auspex": {"trials": {"*": "`+trial+`"}, "patience": "10s"},
                "enact": {"*": "true"}}]}`)
	}

	start := time.Now()
	entries := e.census()
	elapsed := time.Since(start)

	if elapsed < 2*time.Second || elapsed > 2900*time.Millisecond {
		t.Errorf("the census of %d trials took %v, want about two seconds", 3*n, elapsed)
	}
	for i := range n {
		if got := entries[fmt.Sprintf("trial%d", i)]; got["auspex"] != "heeded" || got["aspect"] != "on" {
			t.Errorf("trial%d: %v", i, got)
		}
	}
	b, err := os.ReadFile(counted)
	if err != nil {
		t.Fatal(err)
	}
	most, counts := 0, strings.Fields(string(b))
	for _, c := range counts {
		awakeAtOnce, _ := strconv.Atoi(c)
		most = max(most, awakeAtOnce)
	}
	if len(counts) != 3*n || most != 16 {
		t.Errorf("%d trials counted %v beside them; at most %d at once, want 16", 3*n, counts, most)
	}
}

// TestVigil_TellsStateRitesAsTheirAuspexesSettle: the vigil tells a state
// rite awaited at once, then again once every auspex of it settled; its
// last words are the census, and spared it is told once, forgone.
func TestVigil_TellsStateRitesAsTheirAuspexesSettle(t *testing.T) {
	e := newEnv(t)
	data := e.power()
	e.shows(data, "smt", "on\n")
	e.shows(data, "profile", "balanced\n")
	e.addRite("rites/builtin.json", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "flag", "designation": "transcription", "vessel": "`+data+`/flag", "omens": {"on": "raised", "*": null}}]}`)
	e.addRite("rites/capture.json", `{"pattern": "Mark II", "edict": true, "liturgy": [
	  {"designation": "incantation", "words": "true"}]}`)
	census := e.census()

	lines := e.vigil()
	var told []string
	for _, h := range lines[:len(lines)-1] {
		told = append(told, fmt.Sprintf("%s:%s", h.line["rite"], h.line["auspex"]))
	}
	if want := []string{"builtin:", "capture:", "firewall:", "mouse-autohide-toggle:", "power:awaited", "power:heeded"}; !reflect.DeepEqual(told, want) {
		t.Errorf("told %v, want %v", told, want)
	}
	if last := settle(t, lines); !reflect.DeepEqual(last, census) {
		t.Errorf("the vigil settled on\n%v\nwant the census\n%v", last, census)
	}
	spared := e.vigil("--spare-auspex")
	if last := settle(t, spared); len(spared) != 6 || last["power"]["auspex"] != "forgone" ||
		!reflect.DeepEqual(last, e.census("--spare-auspex")) {
		t.Errorf("the spared vigil: %v", spared)
	}
	if _, err := os.Stat(filepath.Join(data, "spoken")); err == nil {
		t.Fatal("a privileged auspex was spoken")
	}
}
