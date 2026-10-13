package invocation

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// edictII writes an edict of pattern Mark II named capture with the
// vox-casts, inscriptions and liturgy given ($DATA is the vessels' place)
// into the Librarium and loads it.
func (fx fixture) edictII(t *testing.T, voxCasts, inscriptions, liturgy string) *librarium.Rite {
	t.Helper()
	scripture := `{"pattern": "Mark II", "edict": true, "vox-cast": [` + voxCasts + `],
  "inscriptions": {` + inscriptions + `},
  "liturgy": [` + strings.ReplaceAll(liturgy, "$DATA", fx.data) + `]}`
	path := write(t, fx.lib, "rites/capture.json", scripture, 0o644)
	r, findings := librarium.LoadFile(path)
	if findings.Heretical() || !r.Edict || !r.MarkII {
		t.Fatalf("the edict is heretical or no edict of Mark II:\n%v", findings)
	}
	return r
}

// TestEdictII_PerformsItsWholeLiturgyAtEveryInvocation: an edict of Mark II
// speaks every entry of its liturgy, named by its verse, each time it is
// proclaimed; its vox-casts are no verses.
func TestEdictII_PerformsItsWholeLiturgyAtEveryInvocation(t *testing.T) {
	fx := newFixture(t)
	log := filepath.Join(fx.data, "log")
	write(t, fx.data, "scroll", "#!/bin/sh\necho \"recited $1\" >> \"$(dirname \"$0\")/log\"\n", 0o755)
	r := fx.edictII(t, `"progress", "success"`, `"region": {"decrees": "full"}`,
		`{"designation": "incantation", "words": "echo {{rite.name}}-{{inscription.region}} >> $DATA/log"},
		 {"designation": "litany", "scroll": "$DATA/scroll", "offerings": ["{{inscription.region}}"]}`)
	inscriptions, err := ResolveInscriptions(r, "", nil)
	if err != nil || inscriptions["region"] != "full" {
		t.Fatalf("inscriptions %v, %v", inscriptions, err)
	}

	for range 2 {
		out, seen := perform(t, r, Options{Edict: true, Inscriptions: inscriptions})
		if out.Verdict != Triumph || len(out.Deeds) != 2 || len(seen) != 2 || seen[0].Speech == nil ||
			seen[1].Number != 2 || seen[1].Kind != librarium.KindLitany {
			t.Fatalf("outcome %+v, foresight %+v", out, seen)
		}
	}
	want := []string{"capture-full", "recited full", "capture-full", "recited full"}
	if !slices.Equal(lines(t, log), want) {
		t.Fatalf("log %q, want %q", lines(t, log), want)
	}
}

// TestEdictII_VoxCastsOfTheRite: the rite's "vox-cast" list proclaims
// progress once every verse is performed, and success once the whole
// liturgy is; a fall is proclaimed unbidden, and no list proclaims nothing
// but the fall.
func TestEdictII_VoxCastsOfTheRite(t *testing.T) {
	for _, tc := range []struct {
		name, voxCasts, liturgy string
		heard                   []Tidings
		steps                   []int // Values.Step of each proclamation
	}{
		{"progress and success", `"progress", "success"`, `{"designation": "incantation", "words": "true"},
		  {"designation": "incantation", "words": "true"}`,
			[]Tidings{Progress, Progress, Success}, []int{1, 2, 2}},
		{"success written first", `"success", "progress"`, `{"designation": "incantation", "words": "true"}`,
			[]Tidings{Progress, Success}, []int{1, 1}},
		{"success alone", `"success"`, `{"designation": "incantation", "words": "true"},
		  {"designation": "incantation", "words": "true"}`,
			[]Tidings{Success}, []int{2}},
		{"none", ``, `{"designation": "incantation", "words": "true"}`, nil, nil},
		{"a fall", `"progress", "success"`, `{"designation": "incantation", "words": "true"},
		  {"designation": "incantation", "words": "exit 3"},
		  {"designation": "incantation", "words": "true"}`,
			[]Tidings{Progress, Failure}, []int{1, 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newFixture(t)
			r := fx.edictII(t, tc.voxCasts, ``, tc.liturgy)
			h := &hearer{}

			perform(t, r, Options{Edict: true, Herald: h})

			var heard []Tidings
			var steps []int
			for _, p := range h.heard {
				heard, steps = append(heard, p.Tidings), append(steps, p.Values.Tidings.Step)
				if !p.Edict || p.Values.Aspect != "" || p.Values.Former != "" || p.Values.Rite != "capture" ||
					p.Values.Tidings.Steps != strings.Count(tc.liturgy, "designation") ||
					p.Values.Tidings.StepKind != "incantation" {
					t.Errorf("proclaimed %+v", p)
				}
			}
			if !slices.Equal(heard, tc.heard) || !slices.Equal(steps, tc.steps) {
				t.Fatalf("heard %v at steps %v, want %v at %v", heard, steps, tc.heard, tc.steps)
			}
		})
	}
}

// TestEdictII_SpeaksNoAspectInTheEnvironment: an edict's incantations and
// litanies hear its rite and inscriptions, never an aspect.
func TestEdictII_SpeaksNoAspectInTheEnvironment(t *testing.T) {
	fx := newFixture(t)
	t.Setenv("SERVITOR_ASPECT", "stale")
	t.Setenv("SERVITOR_FORMER_ASPECT", "staler")
	r := fx.edictII(t, ``, `"region": {}`,
		`{"designation": "incantation", "words": "printf '%s|' \"${SERVITOR_ASPECT-none}\" \"${SERVITOR_FORMER_ASPECT-none}\" \"$SERVITOR_RITE\" \"$SERVITOR_INSCRIPTION_REGION\" > $DATA/said"},
		 {"designation": "litany", "scroll": "$DATA/scroll"}`)
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

// TestEdictII_RevertsTheVersesBeforeItsFallInReverseOrder: when verse k
// falls, verses k … 1 are reverted in reverse order, a litany's reversion
// as an incantation's, and a reversion that falls leaves the edict
// faltered.
func TestEdictII_RevertsTheVersesBeforeItsFallInReverseOrder(t *testing.T) {
	for _, tc := range []struct {
		name, reversion string
		verdict         Verdict
	}{
		{"reverted", "echo unspoke-1 >> $DATA/log", Reverted},
		{"faltered", "echo unspoke-1 >> $DATA/log; exit 4", Faltered},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newFixture(t)
			log := filepath.Join(fx.data, "log")
			write(t, fx.data, "scroll", "#!/bin/sh\necho spoke-2 >> \"$(dirname \"$0\")/log\"\n", 0o755)
			r := fx.edictII(t, `"progress"`, ``,
				`{"designation": "incantation", "words": "echo spoke-1 >> $DATA/log", "reversion": "`+tc.reversion+`"},
				 {"designation": "litany", "scroll": "$DATA/scroll", "reversion": "echo unspoke-2 >> $DATA/log"},
				 {"designation": "incantation", "words": "echo spoke-3 >> $DATA/log; exit 3", "reversion": "echo unspoke-3 >> $DATA/log"},
				 {"designation": "incantation", "words": "echo never >> $DATA/log"}`)
			h := &hearer{}

			out, _ := perform(t, r, Options{Edict: true, Herald: h})

			if out.Verdict != tc.verdict || out.Fell == nil || out.Fell.Number != 3 || len(out.Reversions) != 3 {
				t.Fatalf("outcome %+v", out)
			}
			for i, number := range []int{3, 2, 1} {
				if out.Reversions[i].Number != number {
					t.Errorf("reversion %d undid verse %d, want %d", i, out.Reversions[i].Number, number)
				}
			}
			want := []string{"spoke-1", "spoke-2", "spoke-3", "unspoke-3", "unspoke-2", "unspoke-1"}
			if !slices.Equal(lines(t, log), want) {
				t.Fatalf("log %q, want %q", lines(t, log), want)
			}
			if len(h.heard) != 3 || h.heard[2].Tidings != Failure || h.heard[2].Outcome.Verdict != tc.verdict ||
				h.heard[2].Verse.Number != 3 {
				t.Fatalf("heard %+v", h.heard)
			}
		})
	}
}

// TestEdictII_PreflightNamesTheKeysOfTheDesignationForm: a heresy found by
// the pre-flight is located at the key the entry is written with: its
// "scroll" or its "tongue".
func TestEdictII_PreflightNamesTheKeysOfTheDesignationForm(t *testing.T) {
	fx := newFixture(t)
	r := fx.edictII(t, ``, ``, `{"designation": "incantation", "words": "true"},
	{"designation": "litany",
	 "scroll": "$DATA/absent"},
	{"designation": "incantation", "words": "true",
	 "tongue": "no-such-tongue-of-the-machine"}`)

	hs := refused(t, r, Options{Edict: true}, "absent")

	if len(hs) != 2 {
		t.Fatalf("heresies %v", hs)
	}
	for i, want := range []struct {
		verse int
		ptr   string
	}{{2, "/liturgy/1/scroll"}, {3, "/liturgy/2/tongue"}} {
		at := r.Locate(want.ptr)
		if h := hs[i]; h.Verse != want.verse || h.Position != at || at == r.Locate(want.ptr[:len("/liturgy/1")]) {
			t.Errorf("heresy %d at verse %d, %+v; want verse %d, %s at %+v", i, h.Verse, h.Position, want.verse,
				want.ptr, at)
		}
	}
	if exists(filepath.Join(fx.data, "log")) {
		t.Fatal("a forbidden edict was performed")
	}
}
