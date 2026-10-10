package augury

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

func TestAugur_TheAuspexGivesAnOmen(t *testing.T) {
	for _, tc := range []struct {
		name   string
		auspex string
		want   string
	}{
		{"short form", `"echo off"`, "auspex=off"},
		{"long form", `{"rite": "printf '  on\n'"}`, "auspex=on"},
		{"output of no declared aspect", `"echo porpl"`, ""},
		{"several lines", `"echo on; echo off"`, ""},
		{"failure", `"echo on; exit 1"`, ""},
		{"a tongue spoken nowhere", `"echo on"`, ""},
		{"only its standard output is heard", `"echo off >&2; echo on"`, "auspex=on"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newFixture(t)
			keys := `
  "auspex": ` + tc.auspex + `,`
			if tc.name == "a tongue spoken nowhere" {
				keys += `
  "tongue": "no-such-tongue-of-the-machine",`
			}
			r := fx.rite(t, `"on", "off"`, keys, `{"vox-cast": "success"}`)
			a := fx.augur(t, r, Options{})
			if got := omens(a); got != tc.want {
				t.Errorf("omens %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAugur_TheAuspexIsLastAndMustAgree(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off"`, `
  "auspex": "cat $SERVITOR_RITE.omen",`, `{"tether": "$DATA/current", "anchor": "$DATA/themes/{{aspect}}"}`)
	auspexSays := func(aspect string) { write(t, filepath.Join(fx.lib, "rites", "mouse.omen"), aspect+"\n") }
	fx.invoke(t, r, "on", nil)
	auspexSays("on")
	a := fx.augur(t, r, Options{})
	if got := omens(a); got != "1:tether=on auspex=on" {
		t.Errorf("omens %q", got)
	}
	standing(t, a, "on", Performed)
	auspexSays("off")
	standing(t, fx.augur(t, r, Options{}), "", Corrupted)
}

func TestAugur_TheAuspexSeesTheTargetAspectOnlyDuringAnInvocation(t *testing.T) {
	for name, auspex := range map[string]string{
		"environment": `[ -z "$SERVITOR_ASPECT" ] && echo off || echo "$SERVITOR_ASPECT"`,
		"placeholder": `[ -z "{{aspect}}" ] && echo off || echo "{{aspect}}"`,
	} {
		t.Run(name, func(t *testing.T) {
			fx := newFixture(t)
			r := fx.rite(t, `"on", "off", "auto"`, `
  "auspex": `+quote(auspex)+`,`, `{"vox-cast": "success"}`)
			if got := omens(fx.augur(t, r, Options{})); got != "auspex=off" {
				t.Errorf("outside an invocation: omens %q, want auspex=off", got)
			}
			if got := omens(fx.augur(t, r, Options{Target: "auto"})); got != "auspex=auto" {
				t.Errorf("during an invocation into auto: omens %q, want auspex=auto", got)
			}
		})
	}
}

func TestAugur_TheAuspexHearsTheSlatesInscriptions(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off"`, `
  "auspex": "echo ${SERVITOR_INSCRIPTION_PLACE}{{inscription.reason}}",`, `{"vox-cast": "success"}`)
	fx.invoke(t, r, "off", map[string]string{"place": "o", "reason": "n"})
	if got := omens(fx.augur(t, r, Options{})); got != "auspex=on" {
		t.Errorf("omens %q, want auspex=on", got)
	}
}

func TestAugur_AnImpatientAuspexIsSlainAndGivesNoOmen(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off"`, `
  "auspex": {"rite": "sleep 3 & sleep 3; echo on", "patience": "100ms"},`, `{"vox-cast": "success"}`)
	start := time.Now()
	a := fx.augur(t, r, Options{})
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("the augury waited %v for an impatient auspex", elapsed)
	}
	if got := omens(a); got != "" {
		t.Errorf("omens %q, want none", got)
	}
}

func TestAugur_TheAuspexWaitsTwoSecondsByDefault(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off"`, `
  "auspex": "sleep 1; echo on",`, `{"vox-cast": "success"}`)
	if r.Auspex.Wait() != librarium.DefaultAuspexPatience || librarium.DefaultAuspexPatience != 2*time.Second {
		t.Fatalf("default patience %v", r.Auspex.Wait())
	}
	if got := omens(fx.augur(t, r, Options{})); got != "auspex=on" {
		t.Errorf("omens %q, want auspex=on", got)
	}
}

func TestAugur_TheAuspexMayBeForgone(t *testing.T) {
	fx := newFixture(t)
	mark := filepath.Join(fx.data, "awakened")
	r := fx.rite(t, `"on", "off"`, `
  "auspex": "touch `+mark+`; echo on",`, `{"vox-cast": "success"}`)
	a := fx.augur(t, r, Options{ForgoAuspex: true})
	if _, err := os.Stat(mark); err == nil {
		t.Error("a forgone auspex was awakened")
	}
	if got := omens(a); got != "" {
		t.Errorf("omens %q, want none", got)
	}
	standing(t, a, "", Dormant)
}

func quote(s string) string {
	out := `"`
	for _, c := range s {
		if c == '"' || c == '\\' {
			out += `\`
		}
		out += string(c)
	}
	return out + `"`
}
