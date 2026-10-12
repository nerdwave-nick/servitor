package placeholder

import (
	"strings"
	"testing"
)

// exalted is the scope of the words of an exalted verse, in which the
// mandatory inscription "mode" lacks a decree for some aspect.
var exalted = Scope{Mode: RiteMode, Inscriptions: []string{"reason", "mode"}, Exalted: true, Fleeting: []string{"mode"}}

func TestCheck_ExaltedWordsAreFixedAtSanction(t *testing.T) {
	cases := []struct {
		text, want string // want: a word of the denunciation, "" when pure
	}{
		{"{{aspect}} {{rite.name}} {{inscription.reason}}", ""},
		{"{{{{former}}", ""},
		{"echo {{former}}", `"$SERVITOR_FORMER_ASPECT"`},
		{"x {{inscription.mode}}", `"$SERVITOR_INSCRIPTION_MODE"`},
		{"{{inscription.nonesuch}}", "never declared"},
	}
	for _, c := range cases {
		hs := exalted.Check(c.text)
		switch {
		case c.want == "" && len(hs) != 0:
			t.Errorf("%q: %v", c.text, hs)
		case c.want != "" && (len(hs) != 1 || !strings.Contains(hs[0].Message(), c.want)):
			t.Errorf("%q: %v, want one denunciation naming %s", c.text, hs, c.want)
		}
	}
	if hs := rite.Check("{{former}} {{inscription.mode}}"); len(hs) != 0 {
		t.Errorf("common words may speak what is fleeting: %v", hs)
	}
	if hs := (Scope{Mode: RiteMode, Edict: true, Exalted: true}).Check("{{former}}"); len(hs) != 1 || hs[0].Kind != Aspectless {
		t.Errorf("an edict's exalted words know no former aspect at all: %v", hs)
	}
}

func TestInscriptionVariable_NamesTheKeyInCapitals(t *testing.T) {
	for key, want := range map[string]string{"mode": "SERVITOR_INSCRIPTION_MODE", "cpu-mode.x": "SERVITOR_INSCRIPTION_CPU_MODE_X"} {
		if got := InscriptionVariable(key); got != want {
			t.Errorf("InscriptionVariable(%q) = %q, want %q", key, got, want)
		}
	}
}
