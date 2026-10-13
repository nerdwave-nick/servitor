package augury

import (
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// TestAugur_TheAuspexHearsTheSlatesInscriptions: a scan hears the
// inscriptions the data-slate recorded, as variables and as placeholders.
func TestAugur_TheAuspexHearsTheSlatesInscriptions(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off"`, "", spirit("chant", "",
		`{"scan": "echo ${SERVITOR_INSCRIPTION_PLACE}{{inscription.reason}}"}`))
	fx.slate(t, r, "off", "triumph", map[string]string{"place": "o", "reason": "n"})
	if got := readings(fx.augur(t, r, Options{})); got != "chant=on(on)" {
		t.Errorf("readings %q, want chant=on(on)", got)
	}
}

// TestAugur_TheAuspexWaitsTwoSecondsByDefault: an auspex that names no
// patience labours two seconds before it is slain.
func TestAugur_TheAuspexWaitsTwoSecondsByDefault(t *testing.T) {
	fx := newFixture(t)
	r := fx.rite(t, `"on", "off"`, "", spirit("chant", "", `{"scan": "sleep 1; echo on"}`))
	wait := r.Subjects[0].Matter.(*librarium.MachineSpirit).Auspex.Wait()
	if wait != librarium.DefaultAuspexPatience || librarium.DefaultAuspexPatience != 2*time.Second {
		t.Fatalf("default patience %v", wait)
	}
	if got := readings(fx.augur(t, r, Options{})); got != "chant=on(on)" {
		t.Errorf("readings %q, want chant=on(on)", got)
	}
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
