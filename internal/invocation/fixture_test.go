package invocation

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// fixture is a Librarium and a separate place for vessels, both temporary.
type fixture struct {
	lib, data string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	return fixture{lib: t.TempDir(), data: t.TempDir()}
}

// rite writes a rite of aspects on and off with the given liturgy ($DATA is
// the vessels' place) into the Librarium and loads it.
func (fx fixture) rite(t *testing.T, liturgy string) *librarium.Rite {
	t.Helper()
	return fx.riteWith(t, "", liturgy)
}

// riteWith is rite with further keys of the rite itself (e.g. `"tongue":
// "zsh",`) written before its liturgy.
func (fx fixture) riteWith(t *testing.T, keys, liturgy string) *librarium.Rite {
	t.Helper()
	scripture := fmt.Sprintf(`{
  "pattern": "Mark I",%s
  "aspects": ["on", "off"],
  "inscriptions": {"reason": {"purpose": "why"}},
  "liturgy": [%s]
}`, keys, strings.ReplaceAll(liturgy, "$DATA", fx.data))
	path := filepath.Join(fx.lib, "rites", "mouse.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(scripture), 0o644); err != nil {
		t.Fatal(err)
	}
	r, findings := librarium.LoadFile(path)
	if findings.Heretical() {
		t.Fatalf("rite is heretical:\n%v", findings)
	}
	return r
}

// write writes content with mode to rel within dir, creating directories.
func write(t *testing.T, dir, rel, content string, mode fs.FileMode) string {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, mode); err != nil {
		t.Fatal(err)
	}
	return p
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func sealOf(t *testing.T, p string) fs.FileMode {
	t.Helper()
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

func link(t *testing.T, anchor, name string) {
	t.Helper()
	if err := os.Symlink(anchor, name); err != nil {
		t.Fatal(err)
	}
}

func anchorOf(t *testing.T, name string) string {
	t.Helper()
	a, err := os.Readlink(name)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// invoke prepares and performs r into aspect and expects triumph.
func invoke(t *testing.T, r *librarium.Rite, aspect string) []Foresight {
	t.Helper()
	inv, err := Prepare(r, Options{Aspect: aspect})
	if err != nil {
		t.Fatalf("pre-flight of %q:\n%v", aspect, err)
	}
	out, err := inv.Perform()
	if err != nil || out.Fell != nil {
		t.Fatalf("performing %q: err=%v fell=%+v", aspect, err, out.Fell)
	}
	return inv.Foresee()
}

// refused prepares r into aspect, expects a heresy containing want, and
// checks that performing does nothing.
func refused(t *testing.T, r *librarium.Rite, opts Options, want string) []Heresy {
	t.Helper()
	inv, err := Prepare(r, opts)
	var hs Heresies
	if !errors.As(err, &hs) || !strings.Contains(err.Error(), want) {
		t.Fatalf("expected a heresy containing %q, got %v", want, err)
	}
	if out, err := inv.Perform(); err == nil || out.Fell != nil {
		t.Fatalf("a heretical invocation was performed: %+v %v", out, err)
	}
	return hs
}

// tree records everything below dir: scripture and seal of vessels, anchors
// of links, and directories.
func tree(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(p)
		if err != nil {
			return err
		}
		switch {
		case info.Mode()&fs.ModeSymlink != 0:
			a, _ := os.Readlink(p)
			out[p] = "-> " + a
		case info.IsDir():
			out[p] = "dir " + info.Mode().Perm().String()
		default:
			b, _ := os.ReadFile(p)
			out[p] = info.Mode().Perm().String() + " " + string(b)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func sameTree(t *testing.T, before, after map[string]string) {
	t.Helper()
	for p, v := range before {
		if after[p] != v {
			t.Errorf("%s changed: %q -> %q", p, v, after[p])
		}
	}
	for p, v := range after {
		if _, ok := before[p]; !ok {
			t.Errorf("%s appeared: %q", p, v)
		}
	}
}

// plainGlosses are plain words no denunciation may speak.
var plainGlosses = regexp.MustCompile(`(?i)\b(error|warning|switch|states?|metadata|config|invalid|required|description|file|field|value is|timeout|symlink|command line|directory|permission denied|no such|fail(ed|s|ure)?|status)\b`)

func grimdark(t *testing.T, words string) {
	t.Helper()
	if m := plainGlosses.FindString(words); m != "" {
		t.Errorf("%q speaks the plain word %q", words, m)
	}
}
