package invocation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEdict_RefusesAnAspect: an edict is proclaimed by its name alone; an
// aspect invoked upon it is refused, and nothing is spoken.
func TestEdict_RefusesAnAspect(t *testing.T) {
	fx := newFixture(t)
	r := fx.edictII(t, ``, ``, `{"designation": "incantation", "words": "echo spoken >> $DATA/log"}`)

	hs := refused(t, r, Options{Edict: true, Aspect: "on"}, `the edict "capture" bears no aspect`)

	grimdark(t, hs[0].Error())
	if exists(filepath.Join(fx.data, "log")) {
		t.Fatal("an edict refused was performed")
	}
}

// TestEdict_ProclamationsBearNoAspect: an edict's proclamations are marked
// so and bear no aspect; a rite of aspects is never proclaimed as an edict.
func TestEdict_ProclamationsBearNoAspect(t *testing.T) {
	fx := newFixture(t)
	r := fx.edictII(t, `"success"`, ``, `{"designation": "incantation", "words": "true"}`)
	h := &hearer{}

	perform(t, r, Options{Edict: true, Herald: h})

	if len(h.heard) != 1 {
		t.Fatalf("heard %+v", h.heard)
	}
	if p := h.heard[0]; !p.Edict || p.Values.Aspect != "" || p.Values.Former != "" || p.Values.Rite != "capture" {
		t.Fatalf("proclaimed %+v", p)
	}
	write(t, fx.data, "util.kdl", "", 0o644)
	rite := fx.riteWith(t, `"vox-cast": ["success"],`, mouseSanctum)
	h = &hearer{}
	perform(t, rite, Options{Aspect: "on", Herald: h})
	if len(h.heard) != 1 || h.heard[0].Edict {
		t.Fatalf("a rite proclaimed as an edict: %+v", h.heard)
	}
}

// TestEdict_MandatoryInscriptionUnmetSpeaksOfTheEdict: a mandatory
// inscription left without value is lamented of the edict, never of an
// aspect, and refuses it.
func TestEdict_MandatoryInscriptionUnmetSpeaksOfTheEdict(t *testing.T) {
	fx := newFixture(t)
	r := fx.edictII(t, ``, `"region": {"mandatory": true}`,
		`{"designation": "incantation", "words": "echo {{inscription.region}} >> $DATA/log"}`)

	_, err := ResolveInscriptions(r, "", nil)

	if err == nil || !strings.Contains(err.Error(), `the edict "capture"`) || strings.Contains(err.Error(), "aspect") {
		t.Fatalf("lament %v", err)
	}
	grimdark(t, err.Error())
	refused(t, r, Options{Edict: true}, `the edict "capture"`)
	if _, err := os.Stat(filepath.Join(fx.data, "log")); err == nil {
		t.Fatal("an edict refused was performed")
	}
}
