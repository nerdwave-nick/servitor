package sanction

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

var update = flag.Bool("update", false, "write the golden units anew")

// librariumOf writes a Librarium holding the rite name with scripture, and
// the files beside it (relative to rites/), and loads it.
func librariumOf(t *testing.T, name, scripture string, files map[string]string) *librarium.Rite {
	t.Helper()
	dir := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(dir, "rites", rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(name+".json", scripture)
	for rel, content := range files {
		write(rel, content)
	}
	lib := librarium.Load(dir)
	r := lib.Rites[name]
	if r == nil {
		t.Fatalf("the rite %s is heretical: %v", name, lib.Findings)
	}
	return r
}

// options draw a unit for uid 1000, finding every tongue in /usr/bin.
var options = Options{
	UID:      1000,
	Orders:   librarium.Orders{Tongue: "bash", Patience: 30 * time.Second},
	LookPath: func(name string) (string, error) { return "/usr/bin/" + name, nil },
}

func sameSigils(t *testing.T, what string, a, b *Unit) {
	t.Helper()
	if a.Sigil() != b.Sigil() {
		t.Errorf("%s changed the sigil:\n%s\n%s", what, a.Canonical(), b.Canonical())
	}
	for _, aspect := range []string{"on", "off"} {
		x, _ := a.SigilOf(aspect)
		y, _ := b.SigilOf(aspect)
		if x != y {
			t.Errorf("%s changed the sigil of %q", what, aspect)
		}
	}
}

// TestBuild_ExaltedEditsLapseTheSigil: an exalted omen, a tome, a relic or
// a scroll changed changes the sigil; an omen changed for one aspect leaves
// the sigil of the other.
func TestBuild_ExaltedEditsLapseTheSigil(t *testing.T) {
	base := buildGaming(t, gaming, gamingFiles())
	exaltedOmen := func() (string, map[string]string) {
		return strings.Replace(gaming, `"on": "kernel.split_lock_mitigate = {{inscription.level}}"`,
			`"on": "kernel.split_lock_mitigate = 0 # war"`, 1), gamingFiles()
	}
	edits := map[string]func() (string, map[string]string){
		"an exalted omen": exaltedOmen,
		"a tome": func() (string, map[string]string) {
			f := gamingFiles()
			f["tomes/split-lock.conf"] += "# more\n"
			return gaming, f
		},
		"a relic": func() (string, map[string]string) {
			f := gamingFiles()
			f["scrolls/tune.conf"] = "level=4\n"
			return gaming, f
		},
		"a scroll": func() (string, map[string]string) {
			f := gamingFiles()
			f["scrolls/tune"] += "echo more\n"
			return gaming, f
		},
	}
	for what, edit := range edits {
		scripture, files := edit()
		if scripture == gaming && what == "an exalted omen" {
			t.Fatal("the exalted omen is not in the rite")
		}
		if u := buildGaming(t, scripture, files); u.Sigil() == base.Sigil() {
			t.Errorf("%s changed, yet the sigil stays %s", what, u.Sigil())
		}
	}
	scripture, files := exaltedOmen()
	u := buildGaming(t, scripture, files)
	if mustSigil(t, u, "off") != mustSigil(t, base, "off") || mustSigil(t, u, "on") == mustSigil(t, base, "on") {
		t.Errorf("an omen of \"on\" alone: on %s (was %s), off %s (was %s)", mustSigil(t, u, "on"),
			mustSigil(t, base, "on"), mustSigil(t, u, "off"), mustSigil(t, base, "off"))
	}
}

// TestBuild_AnEdictHasOnePlan: an edict is drawn once, without aspect, under
// pattern Mark II; its plan holds its exalted verses by their numbers, and
// it hears neither aspect, former aspect nor omen.
func TestBuild_AnEdictHasOnePlan(t *testing.T) {
	r := librariumOf(t, "reboot", `{"pattern": "Mark II", "edict": true, "vox-cast": ["progress"],
	  "inscriptions": {"when": {"decrees": "now"}},
	  "liturgy": [{"designation": "incantation", "words": "echo common"},
	    {"designation": "incantation", "words": "systemctl reboot --when={{inscription.when}}", "exalted": true}]}`, nil)
	u, err := Build(r, options)
	if err != nil {
		t.Fatal(err)
	}
	if u.Pattern != Pattern || u.Environment.Pattern != Pattern || !u.Edict || len(u.Aspects) != 1 ||
		u.Aspects[0].Aspect != "" || u.Aspects[0].Inscriptions["when"] != "now" || len(u.Aspects[0].Liturgy) != 1 ||
		u.Aspects[0].Liturgy[0].Verse != 2 || u.Aspects[0].Liturgy[0].Target != "systemctl reboot --when=now" ||
		strings.Join(u.Environment.Keys, " ") != "SERVITOR_INSCRIPTION_WHEN SERVITOR_RITE" {
		t.Fatalf("unit %s", u.Canonical())
	}
	if sigil, ok := u.SigilOf(""); !ok || sigil != u.Sigil() {
		t.Errorf("the sigil of the one plan %s is not the sigil of the edict %s", sigil, u.Sigil())
	}
}

// TestBuild_NothingToSanction: a rite without exalted subject or observance
// is no unit; one whose tome cannot be read is none either.
func TestBuild_NothingToSanction(t *testing.T) {
	r := librariumOf(t, "plain", `{"pattern": "Mark II", "aspects": ["on"], "subjects": [
	  {"name": "m", "designation": "machine-spirit", "enact": {"*": "echo"}, "exalted": false}]}`, nil)
	if u, err := Build(r, options); !errors.Is(err, ErrUnexalted) || u != nil {
		t.Errorf("Build = %v, %v; want ErrUnexalted", u, err)
	}
	files := gamingFiles()
	delete(files, "tomes/split-lock.conf")
	if u, err := Build(librariumOf(t, "gaming", gaming, files), options); err == nil ||
		!strings.Contains(err.Error(), "split-lock.conf") || u != nil {
		t.Errorf("Build = %v, %v; want the lament of the tome", u, err)
	}
}

// TestBuild_ARiteOfMarkIIsNeverDrawn: a rite read as pattern Mark I, before
// its conversion, is drawn into no unit: its verses are no form the
// exalted servitor performs.
func TestBuild_ARiteOfMarkIIsNeverDrawn(t *testing.T) {
	r, fs := librarium.Parse("tune", "/lib/rites/tune.json", []byte(`{"pattern": "Mark I", "aspects": ["on"],
	  "liturgy": [{"incantation": "sysctl -wq kernel.split_lock_mitigate=0", "exalted": true}]}`))
	if r == nil || fs.Heretical() || r.MarkII {
		t.Fatalf("the rite of Mark I is not read as written: %v %v", r, fs)
	}
	if u, err := Build(r, options); u != nil || err == nil || !strings.Contains(err.Error(), `"Mark I"`) {
		t.Errorf("Build = %v, %v; want a lament of its pattern", u, err)
	}
}
