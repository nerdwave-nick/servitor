package cli

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/augury"
)

// vigilLine is one line of a vigil and the moment it was heard.
type vigilLine struct {
	line map[string]any
	at   time.Time
}

// listener hears what the servitor writes, line by line, as it writes it;
// once it has heard bear lines it answers every write with a broken pipe.
type listener struct {
	t     *testing.T
	mu    sync.Mutex
	buf   bytes.Buffer
	lines []vigilLine
	bear  int // lines heard before the listener departs; 0 never departs
}

func (l *listener) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.bear > 0 && len(l.lines) >= l.bear {
		return 0, syscall.EPIPE
	}
	now := time.Now()
	l.buf.Write(p)
	for {
		line, err := l.buf.ReadBytes('\n')
		if err != nil {
			l.buf.Write(line) // not yet whole
			return len(p), nil
		}
		var v map[string]any
		if err := json.Unmarshal(line, &v); err != nil {
			l.t.Errorf("the vigil spoke a line that is no binharic object: %q: %v", line, err)
		}
		l.lines = append(l.lines, vigilLine{line: v, at: now})
	}
}

// vigil keeps `census --vigil` with the further runes and returns its
// lines; stderr must stay empty and the death-mark 0.
func (e *env) vigil(runes ...string) []vigilLine {
	e.t.Helper()
	l := &listener{t: e.t}
	var errb bytes.Buffer
	code := Execute(append([]string{"--librarium", e.cfgDir, "census", "--vigil"}, runes...), l, &errb)
	if code != 0 || errb.Len() > 0 || l.buf.Len() > 0 {
		e.t.Fatalf("census --vigil %v: exit %d\nstderr: %s\nunended: %q", runes, code, errb.String(), l.buf.String())
	}
	return l.lines
}

// settle replays a vigil: the last line of every rite, without its herald,
// by rite; it fails unless "concluded" is told once, last.
func settle(t *testing.T, lines []vigilLine) map[string]map[string]any {
	t.Helper()
	if len(lines) == 0 || !reflect.DeepEqual(lines[len(lines)-1].line, map[string]any{"herald": "concluded"}) {
		t.Fatalf("the vigil did not end with its conclusion: %v", lines)
	}
	last := map[string]map[string]any{}
	for _, h := range lines[:len(lines)-1] {
		switch h.line["herald"] {
		case "rite":
			entry := maps.Clone(h.line)
			delete(entry, "herald")
			last[entry["rite"].(string)] = entry
		case "lament":
		default:
			t.Errorf("a herald unknown before the conclusion: %v", h.line)
		}
	}
	return last
}

// TestVigil_TellsTheCensusAsItIsLearnt: the vigil tells every rite by name
// at once, an auspex still awaited marked so, then once more every awaited
// rite as its auspex settles — heeded or mute — and concludes; the last
// word of every rite is its entry of the binharic census, with or without
// --binharic.
func TestVigil_TellsTheCensusAsItIsLearnt(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/capture.json", edictJSON)
	e.addRite("rites/broken.json", `{`)
	e.addRite("rites/dnd.json", `{"pattern": "Mark I", "aspects": ["on", "off"], "auspex": "sleep 0.3; echo off", "liturgy": []}`)
	e.addRite("rites/fallen.json", `{"pattern": "Mark I", "aspects": ["on", "off"], "auspex": "exit 1", "liturgy": []}`)
	e.addRite("rites/nonsense.json", `{"pattern": "Mark I", "aspects": ["on", "off"], "auspex": "echo porpl", "liturgy": []}`)
	e.addRite("rites/sluggard.json", `{"pattern": "Mark I", "aspects": ["on", "off"],
	  "auspex": {"rite": "sleep 5; echo on", "patience": "0.2s"}, "liturgy": []}`)
	e.mustRun("invoke", "mouse-autohide-toggle", "on", "-s")
	slates := filepath.Join(e.state, "servitor", augury.SlatesDir)
	for _, rite := range []string{"fallen", "broken"} {
		if err := os.WriteFile(filepath.Join(slates, rite+".json"), []byte("{"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	census := e.census()
	names := []string{"broken", "capture", "dnd", "fallen", "mouse-autohide-toggle", "nonsense", "sluggard"}
	awaited := map[string]string{"dnd": "heeded", "fallen": "mute", "nonsense": "mute", "sluggard": "mute"}

	for _, runes := range [][]string{nil, {"--binharic"}} {
		lines := e.vigil(runes...)

		var rites, laments []map[string]any
		for _, h := range lines {
			switch h.line["herald"] {
			case "rite":
				rites = append(rites, h.line)
			case "lament":
				laments = append(laments, h.line)
			}
		}
		if len(rites) != len(names)+len(awaited) {
			t.Fatalf("%v: %d rite lines, want %d: %v", runes, len(rites), len(names)+len(awaited), rites)
		}
		for i, name := range names {
			want := ""
			if _, ok := awaited[name]; ok {
				want = "awaited"
			}
			if rites[i]["rite"] != name || rites[i]["auspex"] != want {
				t.Errorf("%v: first line %d is %s with auspex %v, want %s with %q", runes, i, rites[i]["rite"], rites[i]["auspex"], name, want)
			}
		}
		settled := map[string]string{}
		for _, r := range rites[len(names):] {
			settled[r["rite"].(string)] = r["auspex"].(string)
		}
		if !reflect.DeepEqual(settled, awaited) {
			t.Errorf("%v: settled %v, want %v", runes, settled, awaited)
		}
		var lamented []string
		for _, l := range laments {
			lamented = append(lamented, l["rite"].(string))
			if !strings.Contains(l["lament"].(string), "data-slate") || len(l) != 3 {
				t.Errorf("%v: lament %v", runes, l)
			}
		}
		if !slices.Equal(lamented, []string{"broken", "fallen"}) {
			t.Errorf("%v: lamented %v, want broken and fallen", runes, lamented)
		}
		if last := settle(t, lines); !reflect.DeepEqual(last, census) {
			t.Errorf("%v: the vigil settled on\n%v\nwant the census\n%v", runes, last, census)
		}
	}
}

// TestVigil_DoesNotWaitForOneAuspexAfterAnother: n rites whose auspex labours
// a second are all told at once, and concluded about a second later.
func TestVigil_DoesNotWaitForOneAuspexAfterAnother(t *testing.T) {
	e := newEnv(t)
	const n = 12
	for i := range n {
		e.addRite(fmt.Sprintf("rites/vigil%02d.json", i), `{"pattern": "Mark I", "aspects": ["on", "off"],
  "auspex": {"rite": "sleep 1; echo on", "patience": "10s"}, "liturgy": []}`)
	}

	start := time.Now()
	lines := e.vigil()

	if len(lines) != 2*n+2 {
		t.Fatalf("%d lines, want %d", len(lines), 2*n+2)
	}
	if first := lines[n].at.Sub(start); first > 300*time.Millisecond {
		t.Errorf("the first %d lines took %v", n+1, first)
	}
	if concluded := lines[len(lines)-1].at.Sub(start); concluded < time.Second || concluded > 1900*time.Millisecond {
		t.Errorf("concluded after %v, want about a second", concluded)
	}
	for _, entry := range settle(t, lines) {
		if entry["auspex"] != "heeded" && entry["rite"] != "mouse-autohide-toggle" {
			t.Errorf("%v", entry)
		}
	}
}

// TestVigil_SparedTellsEveryRiteOnce: with --spare-auspex every rite is told
// once, final, forgone where it keeps an auspex, and the vigil concludes
// without awakening one.
func TestVigil_SparedTellsEveryRiteOnce(t *testing.T) {
	e := newEnv(t)
	mark := e.sloth()
	e.addRite("rites/capture.json", edictJSON)

	start := time.Now()
	lines := e.vigil("--spare-auspex")

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("the spared vigil took %v", elapsed)
	}
	if len(lines) != 4 {
		t.Fatalf("lines %v, want three rites and the conclusion", lines)
	}
	want := map[string]string{"capture": "", "mouse-autohide-toggle": "", "sloth": "forgone"}
	for _, h := range lines[:3] {
		if rite := h.line["rite"].(string); h.line["herald"] != "rite" || h.line["auspex"] != want[rite] {
			t.Errorf("%v", h.line)
		}
	}
	if last := settle(t, lines); !reflect.DeepEqual(last, e.census("--spare-auspex")) {
		t.Errorf("the spared vigil settled on %v", last)
	}
	if _, err := os.Stat(mark); err == nil {
		t.Fatal("the spared vigil awakened an auspex")
	}
}
