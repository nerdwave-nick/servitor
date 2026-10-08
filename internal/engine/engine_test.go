package engine

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/config"
)

type fixture struct {
	cfgDir, dataDir string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	return fixture{cfgDir: t.TempDir(), dataDir: t.TempDir()}
}

// load writes a switch definition (with $DATA replaced by the data dir) and loads it.
func (fx fixture) load(t *testing.T, name, body string) *config.Switch {
	t.Helper()
	body = strings.ReplaceAll(body, "$DATA", fx.dataDir)
	p := filepath.Join(fx.cfgDir, "switches", name+".json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	set := config.Load(fx.cfgDir)
	if set.Diags.HasErrors() {
		t.Fatalf("config errors: %v", set.Diags)
	}
	return set.Switches[name]
}

func (fx fixture) write(t *testing.T, rel, content string) string {
	t.Helper()
	p := filepath.Join(fx.dataDir, rel)
	if err := os.WriteFile(p, []byte(content), 0o640); err != nil {
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

const mouseSwitch = `{
  "states": ["on", "off"],
  "files": [{
    "file": "$DATA/util.kdl",
    "values": [
      {"state": "on", "value": "cursor {\n    hide-after-inactive-ms 400\n}", "meta": {"example-key": "example-value"}},
      {"state": "off", "value": "", "meta": {"example-key": "other-value", "reason": ""}}
    ],
    "meta": {"reason": {}, "example-key": {"optional": "false"}}
  }]
}`

func TestApply_OnThenOffRoundTrip(t *testing.T) {
	fx := newFixture(t)
	sw := fx.load(t, "mouse", mouseSwitch)
	p := fx.write(t, "util.kdl", "input {\n}\n")

	res, err := Apply(sw, "on", map[string]string{"reason": "gaming remnant"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := "input {\n}\n\n// begin servitor managed -- mouse -- state|on example-key|example-value reason|\"gaming remnant\"\ncursor {\n    hide-after-inactive-ms 400\n}\n// end servitor managed -- mouse\n"
	if got := read(t, p); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	if res.Previous != "" || len(res.Changes) != 1 || !res.Changes[0].Changed() {
		t.Fatalf("unexpected result %+v", res)
	}
	st := ReadStatus(sw)
	if st.Err != nil || st.State() != "on" || st.Meta["reason"] != "gaming remnant" || st.Files[0].Drift {
		t.Fatalf("status = %+v", st)
	}

	res, err = Apply(sw, "off", nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	want = "input {\n}\n\n// begin servitor managed -- mouse -- state|off example-key|other-value\n// end servitor managed -- mouse\n"
	if got := read(t, p); got != want || res.Previous != "on" {
		t.Fatalf("got\n%s\nprevious=%q", got, res.Previous)
	}
	if info, _ := os.Stat(p); info.Mode().Perm() != 0o640 {
		t.Fatalf("mode not preserved: %v", info.Mode())
	}
}

func TestApply_IsIdempotent(t *testing.T) {
	fx := newFixture(t)
	sw := fx.load(t, "mouse", mouseSwitch)
	fx.write(t, "util.kdl", "")
	if _, err := Apply(sw, "on", nil, Options{}); err != nil {
		t.Fatal(err)
	}
	res, err := Apply(sw, "on", nil, Options{})
	if err != nil || res.Changes[0].Changed() {
		t.Fatalf("second apply should be a no-op: %+v %v", res, err)
	}
}

func TestApply_Validation(t *testing.T) {
	fx := newFixture(t)
	sw := fx.load(t, "mouse", mouseSwitch)
	fx.write(t, "util.kdl", "")
	cases := map[string]struct {
		state string
		over  map[string]string
		want  string
	}{
		"unknown state":    {"maybe", nil, `unknown state "maybe"`},
		"unknown key":      {"on", map[string]string{"nope": "x"}, `does not declare metadata key "nope"`},
		"required cleared": {"on", map[string]string{"example-key": ""}, `"example-key" is required`},
		"multiline":        {"on", map[string]string{"reason": "a\nb"}, "single-line"},
	}
	for name, tc := range cases {
		if _, err := Apply(sw, tc.state, tc.over, Options{}); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if got := read(t, filepath.Join(fx.dataDir, "util.kdl")); got != "" {
		t.Fatalf("file modified on validation error: %q", got)
	}
}

func TestApply_MissingFileAndCreate(t *testing.T) {
	fx := newFixture(t)
	sw := fx.load(t, "mouse", mouseSwitch)
	if _, err := Apply(sw, "on", nil, Options{}); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("err = %v", err)
	}
	sw = fx.load(t, "mouse", strings.Replace(mouseSwitch, `"file":`, `"create": true, "file":`, 1))
	if _, err := Apply(sw, "on", nil, Options{}); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(read(t, filepath.Join(fx.dataDir, "util.kdl")), "// begin servitor managed -- mouse -- state|on") {
		t.Fatal("file not created with block")
	}
}

func TestApply_FollowsSymlinks(t *testing.T) {
	fx := newFixture(t)
	sw := fx.load(t, "mouse", mouseSwitch)
	real := fx.write(t, "real.kdl", "")
	link := filepath.Join(fx.dataDir, "util.kdl")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(sw, "on", nil, Options{}); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Lstat(link); fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symlink was replaced by a regular file")
	}
	if !strings.Contains(read(t, real), "state|on") {
		t.Fatal("symlink target not updated")
	}
}

func TestApply_MultipleBlocksInSameFileAndRollback(t *testing.T) {
	fx := newFixture(t)
	ro := filepath.Join(fx.dataDir, "ro")
	if err := os.Mkdir(ro, 0o755); err != nil {
		t.Fatal(err)
	}
	sw := fx.load(t, "multi", `{"states": ["a", "b"], "files": [
	  {"file": "$DATA/one.conf", "guard": "g1", "values": [{"state": "a", "value": "1a"}, {"state": "b", "value": "1b"}]},
	  {"file": "$DATA/one.conf", "guard": "g2", "values": [{"state": "a", "value": "2a"}, {"state": "b", "value": "2b"}]},
	  {"file": "$DATA/ro/two.conf", "values": [{"state": "a", "value": "x"}, {"state": "b", "value": "y"}]}
	]}`)
	one := fx.write(t, "one.conf", "")
	fx.write(t, "ro/two.conf", "")
	if _, err := Apply(sw, "a", nil, Options{}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, one); strings.Count(got, "begin servitor") != 2 || !strings.Contains(got, "\n1a\n") || !strings.Contains(got, "\n2a\n") {
		t.Fatalf("one.conf:\n%s", got)
	}
	before := read(t, one)
	if err := os.Chmod(ro, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ro, 0o755) })
	if _, err := Apply(sw, "b", nil, Options{}); err == nil {
		t.Skip("directory permissions not enforced (running as root?)")
	}
	if read(t, one) != before {
		t.Fatal("one.conf was not rolled back")
	}
}

func TestReadStatus_NotAppliedInconsistentAndDrift(t *testing.T) {
	fx := newFixture(t)
	sw := fx.load(t, "multi", `{"states": ["a", "b"], "files": [
	  {"file": "$DATA/x.conf", "values": [{"state": "a", "value": "xa"}, {"state": "b", "value": "xb"}]},
	  {"file": "$DATA/y.conf", "values": [{"state": "a", "value": "ya"}, {"state": "b", "value": "yb"}]}
	]}`)
	x := fx.write(t, "x.conf", "")
	fx.write(t, "y.conf", "")
	if st := ReadStatus(sw); !errors.Is(st.Err, ErrNotApplied) || st.State() != "" {
		t.Fatalf("expected not applied, got %+v", st)
	}
	if _, err := Apply(sw, "a", nil, Options{}); err != nil {
		t.Fatal(err)
	}
	if st := ReadStatus(sw); st.State() != "a" {
		t.Fatalf("status %+v", st)
	}
	fx.write(t, "x.conf", strings.Replace(read(t, x), "xa", "edited", 1))
	if st := ReadStatus(sw); st.Err != nil || !st.Files[0].Drift || st.Files[1].Drift {
		t.Fatalf("expected drift on first file only: %+v", st)
	}
	fx.write(t, "x.conf", strings.Replace(read(t, x), "state|a", "state|b", 1))
	if st := ReadStatus(sw); !errors.Is(st.Err, ErrInconsistent) {
		t.Fatalf("expected inconsistent, got %+v", st)
	}
	fx.write(t, "x.conf", "# begin servitor managed -- multi\n")
	if st := ReadStatus(sw); !errors.Is(st.Err, ErrInconsistent) || st.Files[0].Error == "" {
		t.Fatalf("expected malformed error, got %+v", st)
	}
	fx.write(t, "x.conf", "")
	if st := ReadStatus(sw); !errors.Is(st.Err, ErrInconsistent) {
		t.Fatalf("expected partially applied to be inconsistent, got %+v", st)
	}
}

func TestApply_DryRunWritesNothing(t *testing.T) {
	fx := newFixture(t)
	sw := fx.load(t, "mouse", mouseSwitch)
	p := fx.write(t, "util.kdl", "a\n")
	res, err := Apply(sw, "on", nil, Options{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if read(t, p) != "a\n" {
		t.Fatal("dry run modified file")
	}
	diff := DiffLines(res.Changes[0].Before, res.Changes[0].After)
	if !slices.Contains(diff, "+ // begin servitor managed -- mouse -- state|on example-key|example-value") {
		t.Fatalf("diff = %q", diff)
	}
}

func TestPurge_RemovesBlocksAndSkipsMissingFiles(t *testing.T) {
	fx := newFixture(t)
	sw := fx.load(t, "multi", `{"states": ["a"], "files": [
	  {"file": "$DATA/x.conf", "values": [{"state": "a", "value": "xa"}]},
	  {"file": "$DATA/gone.conf", "create": true, "values": [{"state": "a", "value": "ga"}]}
	]}`)
	x := fx.write(t, "x.conf", "keep\n")
	if _, err := Apply(sw, "a", nil, Options{}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(fx.dataDir, "gone.conf")); err != nil {
		t.Fatal(err)
	}
	if ch, err := Purge(sw, Options{DryRun: true}); err != nil || len(ch) != 1 || !strings.Contains(read(t, x), "begin servitor") {
		t.Fatalf("dry purge: %v %v", ch, err)
	}
	if _, err := Purge(sw, Options{}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, x); got != "keep\n" {
		t.Fatalf("after purge: %q", got)
	}
	if st := ReadStatus(sw); !errors.Is(st.Err, ErrNotApplied) {
		t.Fatalf("status after purge: %+v", st)
	}
}
