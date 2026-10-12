package cli

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// The trials never read the store of the machine: unless a trial lays a
// store of its own, the servitor reads one that is never kept.
func init() { rituals.SanctionRoot = filepath.Join(os.TempDir(), "servitor-trials-keep-no-sanctions") }

// sanctionedAt is the hour of every sanction these trials lay.
var sanctionedAt = time.Date(2026, 10, 12, 18, 0, 0, 0, time.UTC)

// sanctions lays a store of sanctions of its own for the trial, for the uid
// the trial runs as, and returns it.
func (e *env) sanctions() sanction.Store {
	e.t.Helper()
	orig := rituals.SanctionRoot
	e.t.Cleanup(func() { rituals.SanctionRoot = orig })
	rituals.SanctionRoot = e.t.TempDir()
	return sanction.Store{Root: rituals.SanctionRoot, UID: os.Getuid()}
}

// sanctify lays in st the sanction of the rite name as it stands now, under
// the name as.
func (e *env) sanctify(st sanction.Store, name, as string) sanction.Sanction {
	e.t.Helper()
	s := &rituals.Servitor{Librarium: librarium.Load(e.cfgDir)}
	u, err := s.Unit(s.Librarium.Rites[name])
	if err != nil {
		e.t.Fatal(err)
	}
	u.Rite = as
	kept := sanction.New(u, sanctionedAt)
	data, err := json.Marshal(kept)
	if err != nil {
		e.t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(st.Path(as)), 0o750); err != nil {
		e.t.Fatal(err)
	}
	if err := os.WriteFile(st.Path(as), data, 0o640); err != nil {
		e.t.Fatal(err)
	}
	return *kept
}

// sanctuary is a Librarium whose split-lock stands sanctioned, whose
// "lapsing" lapsed, whose "fresh" is unsanctioned, and whose store keeps a
// forsaken sanction of the rite "gone"; mouse-autohide-toggle holds no
// exalted verse.
func sanctuary(t *testing.T) (*env, map[string]sanction.Sanction) {
	t.Helper()
	e := newEnv(t)
	st := e.sanctions()
	e.addRite("rites/split-lock.json", splitLockJSON)
	e.addRite("rites/lapsing.json", splitLockJSON)
	e.addRite("rites/fresh.json", splitLockJSON)
	kept := map[string]sanction.Sanction{}
	for name, as := range map[string]string{"split-lock": "split-lock", "lapsing": "lapsing", "fresh": "gone"} {
		kept[as] = e.sanctify(st, name, as)
	}
	e.addRite("rites/lapsing.json", strings.Replace(splitLockJSON, "echo {{aspect}}", "echo lapsed {{aspect}}", 1))
	return e, kept
}

// TestSanction_ListsEverySanction: `servitor sanction` lists every sanction
// laid, by rite, with its standing, sigil and hour, plainly and in binharic.
func TestSanction_ListsEverySanction(t *testing.T) {
	e, kept := sanctuary(t)

	out, errOut, code := e.run("sanction", "--binharic")
	want := `[
  {
    "rite": "gone",
    "standing": "forsaken",
    "sigil": "` + kept["gone"].Sigil + `",
    "sanctioned_at": "2026-10-12T18:00:00Z"
  },
  {
    "rite": "lapsing",
    "standing": "lapsed",
    "sigil": "` + kept["lapsing"].Sigil + `",
    "sanctioned_at": "2026-10-12T18:00:00Z"
  },
  {
    "rite": "split-lock",
    "standing": "sanctioned",
    "sigil": "` + kept["split-lock"].Sigil + `",
    "sanctioned_at": "2026-10-12T18:00:00Z"
  }
]
`
	if code != 0 || out != want || errOut != "" {
		t.Errorf("sanction --binharic: exit %d\nstdout: %s\nwant: %s\nstderr: %s", code, out, want, errOut)
	}

	out, errOut, code = e.run("sanction")
	hour := sanctionedAt.Local().Format("2006-01-02 15:04")
	plain := "RITE        STANDING      SIGIL         SANCTIONED AT\n" +
		"gone        † forsaken    " + sanction.Short(kept["gone"].Sigil) + "  " + hour + "\n" +
		"lapsing     ☓ lapsed      " + sanction.Short(kept["lapsing"].Sigil) + "  " + hour + "\n" +
		"split-lock  ✠ sanctioned  " + sanction.Short(kept["split-lock"].Sigil) + "  " + hour + "\n"
	if code != 0 || out != plain || errOut != "" {
		t.Errorf("sanction: exit %d\nstdout: %q\nwant:   %q\nstderr: %s", code, out, plain, errOut)
	}
}

// TestSanction_NoneLaid: without a sanction the list is empty, and plainly
// the faithful are told how one is laid.
func TestSanction_NoneLaid(t *testing.T) {
	e := newEnv(t)
	if out, errOut, code := e.run("sanction", "--binharic"); code != 0 || out != "[]\n" || errOut != "" {
		t.Errorf("exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	out, errOut, code := e.run("sanction")
	if code != 0 || out != "" || !strings.Contains(errOut, "servitor sanction <rite>") {
		t.Errorf("exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	grimdark(t, errOut)
}

// TestSanction_AsRootIsRefused: as root, every form of the ritual is
// refused with the death-mark 2, and nothing is written; more than one rite
// is refused as well.
func TestSanction_AsRootIsRefused(t *testing.T) {
	e := newEnv(t)
	st := e.sanctions()
	e.addRite("rites/split-lock.json", splitLockJSON)
	if _, _, code := e.run("sanction", "a", "b"); code != 2 {
		t.Errorf("two rites: exit %d", code)
	}
	asRoot(t)
	for _, args := range [][]string{{"sanction"}, {"sanction", "--binharic"}, {"sanction", "split-lock"}} {
		out, errOut, code := e.run(args...)
		if code != 2 || out != "" || !strings.Contains(errOut, "the machine's own authority") {
			t.Errorf("servitor %v as root: exit %d\nstdout: %s\nstderr: %s", args, code, out, errOut)
		}
	}
	if entries, _ := os.ReadDir(st.Root); len(entries) != 0 {
		t.Errorf("the store was written: %v", entries)
	}
}

// TestCensus_TellsTheSanctionOfEveryRite: the census and the augury tell
// each rite's standing towards its sanction, the table marks it by a glyph,
// the census laments the forsaken sanction, and the vigil settles on the
// binharic census.
func TestCensus_TellsTheSanctionOfEveryRite(t *testing.T) {
	e, _ := sanctuary(t)
	want := map[string]string{"split-lock": "sanctioned", "lapsing": "lapsed", "fresh": "unsanctioned",
		"mouse-autohide-toggle": ""}

	census := e.census("--spare-auspex")
	for rite, standing := range want {
		if got := census[rite]["sanction"]; got != standing {
			t.Errorf("the census tells %s %v, want %q", rite, got, standing)
		}
		if out := e.mustRun("augury", rite, "sanction"); out != standing+"\n" {
			t.Errorf("augury %s sanction = %q, want %q", rite, out, standing)
		}
		var augured map[string]any
		if err := json.Unmarshal([]byte(e.mustRun("augury", rite)), &augured); err != nil || augured["sanction"] != standing {
			t.Errorf("the augury of %s tells %v (%v)", rite, augured["sanction"], err)
		}
	}

	out, errOut, code := e.run("census", "--spare-auspex")
	lament := `servitor ✠ the sanction of "gone" is forsaken: no rite named "gone" is recorded in the Librarium ` +
		"any more, and the sanction serves nothing — abjure it: servitor abjure gone\n"
	if code != 0 || errOut != lament {
		t.Errorf("census: exit %d\nstderr: %q\nwant    %q", code, errOut, lament)
	}
	rows := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n")[1:] {
		fields := strings.Fields(line)
		rows[fields[0]] = strings.Join(fields[1:], " ")
	}
	for rite, glyph := range map[string]string{"split-lock": "✠", "lapsing": "☓", "fresh": "○"} {
		if !strings.HasPrefix(rows[rite], glyph+" ") {
			t.Errorf("the census table marks %s as %q, want %s:\n%s", rite, rows[rite], glyph, out)
		}
	}
	if row := rows["mouse-autohide-toggle"]; !strings.HasPrefix(row, "· dormant") {
		t.Errorf("a rite without exalted verses is marked: %q", row)
	}

	l := &listener{t: t}
	var errb strings.Builder
	if code := Execute([]string{"--librarium", e.cfgDir, "census", "--vigil", "--spare-auspex"}, l, &errb); code != 0 {
		t.Fatalf("vigil: exit %d: %s", code, errb.String())
	}
	if first := l.lines[0].line; first["herald"] != "lament" || first["rite"] != "gone" ||
		!strings.Contains(first["lament"].(string), "forsaken") {
		t.Errorf("the vigil first tells %v", first)
	}
	if last := settle(t, l.lines); !reflect.DeepEqual(last, census) {
		t.Errorf("the vigil settled on\n%v\nwant the census\n%v", last, census)
	}
}

// TestInvoke_TheRefusalNamesTheStanding: the refusal of an exalted rite
// names its standing and the ritual of the sanction; even a sanctioned rite
// is refused, and nothing is touched.
func TestInvoke_TheRefusalNamesTheStanding(t *testing.T) {
	e, _ := sanctuary(t)
	for rite, words := range map[string]string{
		"fresh":      "stands unsanctioned",
		"lapsing":    "has lapsed",
		"split-lock": "stands sanctioned",
	} {
		out, errOut, code := e.run("invoke", rite, "on")
		if code != 2 || out != "" || !strings.Contains(errOut, words) || !strings.Contains(errOut, "nothing was touched") ||
			(rite != "split-lock" && !strings.Contains(errOut, "servitor sanction "+rite)) {
			t.Errorf("invoke %s: exit %d\nstdout: %s\nstderr: %s", rite, code, out, errOut)
		}
		grimdark(t, errOut)
	}
	if got := e.targetContent(); got != "input {}\n" {
		t.Errorf("the vessel was touched:\n%s", got)
	}
	if _, err := os.Stat(e.target + ".exalted"); err == nil {
		t.Error("an exalted verse was spoken")
	}
}
