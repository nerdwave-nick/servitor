package augury

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// fixture is a Librarium, a place for vessels and a state directory, all
// temporary.
type fixture struct {
	lib, data string
	env       librarium.Environment
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	env, _ := stateEnv(t)
	return fixture{lib: t.TempDir(), data: t.TempDir(), env: env}
}

// rite writes the rite "mouse" of the given aspects, further keys of the
// rite (e.g. `"auspex": "…",`) and liturgy ($DATA is the vessels' place)
// into the Librarium and loads it.
func (fx fixture) rite(t *testing.T, aspects, keys, liturgy string) *librarium.Rite {
	t.Helper()
	scripture := fmt.Sprintf(`{
  "pattern": "Mark I",
  "aspects": [%s],%s
  "inscriptions": {"place": {"purpose": "where"}, "reason": {"purpose": "why"}},
  "liturgy": [%s]
}`, aspects, keys, liturgy)
	scripture = strings.ReplaceAll(scripture, "$DATA", fx.data)
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

// invoke performs r into aspect with inscriptions and records its slate,
// as the invoking servitor does.
func (fx fixture) invoke(t *testing.T, r *librarium.Rite, aspect string, inscriptions map[string]string) {
	t.Helper()
	opts := invocation.Options{Aspect: aspect, Inscriptions: inscriptions}
	inv, err := invocation.Prepare(r, opts)
	if err != nil {
		t.Fatalf("pre-flight of %q:\n%v", aspect, err)
	}
	out, err := inv.Perform()
	if err != nil || out.Verdict != invocation.Triumph {
		t.Fatalf("performing %q: err=%v outcome=%+v", aspect, err, out)
	}
	if err := Record(fx.env, r.Name, out, opts, noon); err != nil {
		t.Fatal(err)
	}
}

// slate writes the data-slate of r directly.
func (fx fixture) slate(t *testing.T, r *librarium.Rite, aspect string, verdict invocation.Verdict, ins map[string]string) {
	t.Helper()
	if err := WriteSlate(fx.env, r.Name, Slate{Aspect: aspect, Inscriptions: ins,
		LastRite: &LastRite{Verdict: verdict, At: noon}}); err != nil {
		t.Fatal(err)
	}
}

// augur reads the augury of r with opts (the fixture's environment) and
// expects no lament.
func (fx fixture) augur(t *testing.T, r *librarium.Rite, opts Options) Augury {
	t.Helper()
	opts.Env = fx.env
	a, err := Augur(r, opts)
	if err != nil {
		t.Fatalf("augury: %v", err)
	}
	return a
}

func write(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func link(t *testing.T, anchor, name string) {
	t.Helper()
	_ = os.Remove(name)
	if err := os.Symlink(anchor, name); err != nil {
		t.Fatal(err)
	}
}

// omens renders the omens compactly: "<verse>:<key>=<aspect>" and
// "auspex=<aspect>".
func omens(a Augury) string {
	var parts []string
	for _, o := range a.Omens {
		if o.Auspex {
			parts = append(parts, "auspex="+o.Aspect)
			continue
		}
		parts = append(parts, fmt.Sprintf("%d:%s=%s", o.Number, o.Kind.Key(), o.Aspect))
	}
	return strings.Join(parts, " ")
}

// standing expects aspect and standing of a.
func standing(t *testing.T, a Augury, aspect string, s Standing) {
	t.Helper()
	if a.Aspect != aspect || a.Standing != s {
		t.Errorf("aspect %q standing %q, want %q %q (omens %s)", a.Aspect, a.Standing, aspect, s, omens(a))
	}
}
