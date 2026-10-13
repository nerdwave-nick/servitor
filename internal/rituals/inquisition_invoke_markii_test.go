package rituals

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanctum"
)

// TestInvokeII_DoesAsTheInquisitionForetells: what the inquisition foretells
// of the invocation of a state rite of pattern Mark II is what the
// invocation does, upon the same reading of its subjects — a refusal where
// it foretells one, nothing touched; a tainted sanctum subject overwritten;
// a machine-spirit without an auspex enacted anew each time; a desecrated
// rite read in the aspect its subjects agree upon.
func TestInvokeII_DoesAsTheInquisitionForetells(t *testing.T) {
	refused := func(t *testing.T, err error, res Result) {
		t.Helper()
		var hs invocation.Heresies
		if !errors.As(err, &hs) || res.Outcome != nil {
			t.Fatalf("the invocation was not refused: %v %+v", err, res.Outcome)
		}
	}
	for name, tc := range map[string]struct {
		subjects string
		lay      func(fx *fixture)
		sev      librarium.Severity
		words    []string
		aspect   string
		then     func(t *testing.T, fx *fixture, s *Servitor, err error, res Result)
	}{
		"a vessel that may not be consecrated": {
			subjects: `{"name": "absent", "designation": "sanctum", "vessel": "$DATA/absent.kdl", "omens": {"on": "x", "*": ""}}`,
			sev:      librarium.Impurity, words: []string{`the sanctum subject "absent"`, "will be refused"}, aspect: "on",
			then: func(t *testing.T, fx *fixture, _ *Servitor, err error, res Result) {
				refused(t, err, res)
				if _, err := os.Stat(filepath.Join(fx.data, "absent.kdl")); err == nil {
					t.Fatal("a vessel was consecrated")
				}
			},
		},
		"a defiled vessel": {
			subjects: `{"name": "util", "designation": "sanctum", "vessel": "$DATA/broken.kdl", "omens": {"on": "x", "*": ""}}`,
			lay: func(fx *fixture) {
				fx.vessel("broken.kdl", "// +++ begin of sanctum gaming -- aspect|on +++\nx\n")
			},
			sev: librarium.Heresy, words: []string{"defiled", "no invocation can rewrite"}, aspect: "off",
			then: func(t *testing.T, fx *fixture, _ *Servitor, err error, res Result) {
				refused(t, err, res)
				if got := fx.read(filepath.Join(fx.data, "broken.kdl")); got != "// +++ begin of sanctum gaming -- aspect|on +++\nx\n" {
					t.Fatalf("the defiled vessel was touched: %q", got)
				}
			},
		},
		"a tome nowhere to be found": {
			subjects: `{"name": "lost", "designation": "transcription", "vessel": "$DATA/lost.kdl",
			  "omens": {"on": {"tome": "../tomes/missing.kdl"}, "off": null}}`,
			sev: librarium.Heresy, words: []string{"missing.kdl", "would be refused"}, aspect: "on",
			then: func(t *testing.T, fx *fixture, _ *Servitor, err error, res Result) {
				refused(t, err, res)
				if _, err := os.Stat(filepath.Join(fx.data, "lost.kdl")); err == nil {
					t.Fatal("a vessel was written")
				}
			},
		},
		"a tongue spoken nowhere": {
			subjects: `{"name": "smt", "designation": "machine-spirit", "tongue": "no-such-tongue-of-spirits",
			  "auspex": {"scan": "cat $DATA/smt"}, "enact": {"*": "echo {{omen}} > $DATA/smt"}}`,
			lay: func(fx *fixture) { fx.vessel("smt", "off\n") },
			sev: librarium.Impurity, words: []string{`the subject "smt"`, "could never be set"}, aspect: "on",
			then: func(t *testing.T, fx *fixture, _ *Servitor, err error, res Result) {
				refused(t, err, res)
				if got := fx.read(filepath.Join(fx.data, "smt")); got != "off\n" {
					t.Fatalf("the subject was set: %q", got)
				}
			},
		},
		"a tainted sanctum subject": {
			subjects: `{"name": "niri", "designation": "sanctum", "vessel": "$DATA/niri.kdl", "ward": "gaming", "glyph": "#",
			  "omens": {"on": "fast", "*": "slow"}}`,
			lay: func(fx *fixture) {
				fx.vessel("niri.kdl", sanctum.Render(sanctum.Marker{Glyph: "#", Ward: "gaming"}, "", "on", "altered"))
			},
			sev: librarium.Impurity, words: []string{`the sanctum subject "niri"`, "tainted", "will overwrite"}, aspect: "on",
			then: func(t *testing.T, fx *fixture, _ *Servitor, err error, res Result) {
				if err != nil || res.Outcome == nil || !slices.Equal(res.Outcome.Changed, []string{"niri"}) {
					t.Fatalf("the tainted subject was not overwritten: %v %+v", err, res.Outcome)
				}
				want := sanctum.Render(sanctum.Marker{Glyph: "#", Ward: "gaming"}, "", "on", "fast")
				if got := fx.read(filepath.Join(fx.data, "niri.kdl")); got != want {
					t.Fatalf("vessel %q, want %q", got, want)
				}
			},
		},
		"a machine-spirit without an auspex": {
			subjects: `{"name": "blind", "designation": "machine-spirit", "omens": {"on": "1", "*": "0"},
			  "enact": {"*": "echo {{omen}} >> $DATA/log"}}`,
			lay: func(fx *fixture) { fx.vessel("log", "") },
			sev: librarium.Impurity, words: []string{`"blind"`, "enacted anew on every invocation"}, aspect: "on",
			then: func(t *testing.T, fx *fixture, s *Servitor, err error, res Result) {
				again, errAgain := s.Invoke(context.Background(), Petition{Rite: "gaming", Aspect: "on"})
				for _, r := range []Result{res, again} {
					if err != nil || errAgain != nil || r.Outcome == nil || !slices.Equal(r.Outcome.Changed, []string{"blind"}) {
						t.Fatalf("the blind subject was not enacted anew: %v %v %+v", err, errAgain, r.Outcome)
					}
				}
				if got := fx.read(filepath.Join(fx.data, "log")); got != "1\n1\n" {
					t.Fatalf("log %q", got)
				}
			},
		},
		"a desecrated rite": {
			subjects: `{"name": "theme", "designation": "tether", "tether": "$DATA/current", "omens": {"*": "$DATA/themes/{{aspect}}"}}`,
			lay: func(fx *fixture) {
				for _, a := range []string{"on", "off"} {
					if err := os.MkdirAll(filepath.Join(fx.data, "themes", a), 0o755); err != nil {
						fx.t.Fatal(err)
					}
				}
				if err := os.Symlink(filepath.Join(fx.data, "themes", "off"), filepath.Join(fx.data, "current")); err != nil {
					fx.t.Fatal(err)
				}
				if err := augury.WriteSlate(os.LookupEnv, "gaming", augury.Slate{Aspect: "on", LastRite: &augury.LastRite{
					Verdict: invocation.Triumph, At: time.Date(2026, 10, 10, 13, 30, 0, 0, time.UTC)}}); err != nil {
					fx.t.Fatal(err)
				}
			},
			sev: librarium.Impurity, words: []string{"desecrated", `"off"`, `"on"`}, aspect: "off",
			then: func(t *testing.T, _ *fixture, _ *Servitor, err error, res Result) {
				if err != nil || res.Outcome == nil || res.Outcome.Verdict != invocation.Triumph ||
					len(res.Outcome.Changed) != 0 || res.Options.Former != "off" || res.Former != augury.Desecrated {
					t.Fatalf("the desecrated rite was not read in the aspect its subjects agree upon: %v %+v %+v",
						err, res.Former, res.Outcome)
				}
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			fx := newFixture(t)
			fx.rite("gaming", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [`+tc.subjects+`]}`)
			if tc.lay != nil {
				tc.lay(fx)
			}
			s := fx.servitor()

			if got := denounced(inquire(t, s, false, "gaming"), "gaming", tc.sev, tc.words...); len(got) != 1 {
				t.Fatalf("the inquisition foretold %d of %q", len(got), tc.words)
			}
			res, err := s.Invoke(context.Background(), Petition{Rite: "gaming", Aspect: tc.aspect})
			tc.then(t, fx, s, err, res)
		})
	}
}
