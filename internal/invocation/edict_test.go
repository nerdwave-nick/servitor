package invocation

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// edict writes an edict named capture with the inscriptions and liturgy
// given ($DATA is the vessels' place) into the Librarium and loads it.
func (fx fixture) edict(t *testing.T, inscriptions, liturgy string) *librarium.Rite {
	t.Helper()
	scripture := `{"pattern": "Mark I", "edict": true, "inscriptions": {` + inscriptions + `},
  "liturgy": [` + strings.ReplaceAll(liturgy, "$DATA", fx.data) + `]}`
	path := write(t, fx.lib, "rites/capture.json", scripture, 0o644)
	r, findings := librarium.LoadFile(path)
	if findings.Heretical() || !r.Edict {
		t.Fatalf("the edict is heretical or no edict:\n%v", findings)
	}
	return r
}

func TestEdict_PerformsItsWholeLiturgyAtEveryInvocation(t *testing.T) {
	fx := newFixture(t)
	log := filepath.Join(fx.data, "log")
	r := fx.edict(t, `"region": {"decrees": "full"}`,
		`{"incantation": "echo {{rite.name}}-{{inscription.region}} >> $DATA/log"},
		 {"vox-cast": "progress"},
		 {"incantation": "echo again >> $DATA/log"},
		 {"vox-cast": "success"}`)
	inscriptions, err := ResolveInscriptions(r, "", nil)
	if err != nil || inscriptions["region"] != "full" {
		t.Fatalf("inscriptions %v, %v", inscriptions, err)
	}

	for range 2 {
		out, seen := perform(t, r, Options{Edict: true, Inscriptions: inscriptions})
		if out.Verdict != Triumph || len(out.Deeds) != 4 || len(seen) != 4 || seen[0].Speech == nil {
			t.Fatalf("outcome %+v, foresight %+v", out, seen)
		}
	}
	if want := []string{"capture-full", "again", "capture-full", "again"}; !slices.Equal(lines(t, log), want) {
		t.Fatalf("log %q, want %q", lines(t, log), want)
	}
}

func TestEdict_RefusesAnAspect(t *testing.T) {
	fx := newFixture(t)
	r := fx.edict(t, ``, `{"incantation": "echo spoken >> $DATA/log"}`)

	hs := refused(t, r, Options{Edict: true, Aspect: "on"}, `the edict "capture" bears no aspect`)

	grimdark(t, hs[0].Error())
	if exists(filepath.Join(fx.data, "log")) {
		t.Fatal("an edict refused was performed")
	}
}

func TestEdict_SpeaksNoAspectInTheEnvironment(t *testing.T) {
	fx := newFixture(t)
	t.Setenv("SERVITOR_ASPECT", "stale")
	t.Setenv("SERVITOR_FORMER_ASPECT", "staler")
	r := fx.edict(t, `"region": {}`,
		`{"incantation": "printf '%s|' \"${SERVITOR_ASPECT-none}\" \"${SERVITOR_FORMER_ASPECT-none}\" \"$SERVITOR_RITE\" \"$SERVITOR_INSCRIPTION_REGION\" > $DATA/said"},
		 {"litany": "$DATA/scroll"}`)
	write(t, fx.data, "scroll", "#!/bin/sh\nprintf '%s|' \"${SERVITOR_ASPECT-none}\" \"${SERVITOR_FORMER_ASPECT-none}\" \"$SERVITOR_RITE\" > \"$(dirname \"$0\")/recited\"\n", 0o755)

	out, _ := perform(t, r, Options{Edict: true, Inscriptions: map[string]string{"region": "full"}})

	if out.Verdict != Triumph {
		t.Fatalf("outcome %+v", out)
	}
	if got, want := read(t, filepath.Join(fx.data, "said")), "none|none|capture|full|"; got != want {
		t.Fatalf("the incantation said %q, want %q", got, want)
	}
	if got, want := read(t, filepath.Join(fx.data, "recited")), "none|none|capture|"; got != want {
		t.Fatalf("the litany said %q, want %q", got, want)
	}
}

func TestEdict_RevertsTheVersesBeforeItsFall(t *testing.T) {
	for _, tc := range []struct {
		name, reversion string
		verdict         Verdict
	}{
		{"reverted", "echo unspoke >> $DATA/log", Reverted},
		{"faltered", "echo unspoke >> $DATA/log; exit 4", Faltered},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newFixture(t)
			log := filepath.Join(fx.data, "log")
			r := fx.edict(t, ``, `{"incantation": "echo spoke >> $DATA/log", "reversion": "`+tc.reversion+`"},
			  {"vox-cast": "progress"},
			  {"incantation": "exit 3"},
			  {"incantation": "echo never >> $DATA/log"}`)
			h := &hearer{}

			out, _ := perform(t, r, Options{Edict: true, Herald: h})

			if out.Verdict != tc.verdict || out.Fell == nil || out.Fell.Number != 3 || len(out.Reversions) != 1 {
				t.Fatalf("outcome %+v", out)
			}
			if want := []string{"spoke", "unspoke"}; !slices.Equal(lines(t, log), want) {
				t.Fatalf("log %q, want %q", lines(t, log), want)
			}
			if len(h.heard) != 2 || h.heard[1].Tidings != Failure || h.heard[1].Outcome.Verdict != tc.verdict {
				t.Fatalf("heard %+v", h.heard)
			}
		})
	}
}

func TestEdict_ProclamationsBearNoAspect(t *testing.T) {
	fx := newFixture(t)
	r := fx.edict(t, ``, `{"incantation": "true"}, {"vox-cast": "success"}`)
	h := &hearer{}

	perform(t, r, Options{Edict: true, Herald: h})

	if len(h.heard) != 1 {
		t.Fatalf("heard %+v", h.heard)
	}
	if p := h.heard[0]; !p.Edict || p.Values.Aspect != "" || p.Values.Former != "" || p.Values.Rite != "capture" {
		t.Fatalf("proclaimed %+v", p)
	}
	rite := fx.rite(t, `{"vox-cast": "success"}`)
	h = &hearer{}
	perform(t, rite, Options{Aspect: "on", Herald: h})
	if len(h.heard) != 1 || h.heard[0].Edict {
		t.Fatalf("a rite proclaimed as an edict: %+v", h.heard)
	}
}

func TestEdict_MandatoryInscriptionUnmetSpeaksOfTheEdict(t *testing.T) {
	fx := newFixture(t)
	r := fx.edict(t, `"region": {"mandatory": true}`, `{"incantation": "echo {{inscription.region}} >> $DATA/log"}`)

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
