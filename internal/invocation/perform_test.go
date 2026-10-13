package invocation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// TestPerform_RevertsEveryFileStepWhenOneFalls: when a subject falls,
// every subject set before it is restored in reverse order, the machine as
// it stood.
func TestPerform_RevertsEveryFileStepWhenOneFalls(t *testing.T) {
	fx := newFixture(t)
	util := write(t, fx.data, "util.kdl", "input {}\n", 0o640)
	created := filepath.Join(fx.data, "created.kdl")
	theme := filepath.Join(fx.data, "current-theme")
	link(t, "themes/porpl", theme)
	vis := write(t, fx.data, "visage.conf", "visage off", 0o600)
	ro := write(t, fx.data, "ro/x.conf", "x off", 0o644)
	r := fx.rite(t, `{"name": "util", "designation": "sanctum", "vessel": "$DATA/util.kdl", "omens": {"*": "1"}},
	  {"name": "created", "designation": "sanctum", "vessel": "$DATA/created.kdl", "consecrate": true, "omens": {"*": "2"}},
	  {"name": "theme", "designation": "tether", "tether": "$DATA/current-theme", "omens": {"*": "$DATA/themes/{{aspect}}"}},
	  {"name": "visage", "designation": "transcription", "vessel": "$DATA/visage.conf",
	   "omens": {"on": null, "*": "visage {{aspect}}"}},
	  {"name": "ro", "designation": "transcription", "vessel": "$DATA/ro/x.conf", "omens": {"*": "x {{aspect}}"}}`)
	before := tree(t, fx.data)

	inv, err := Prepare(r, Options{Aspect: "on"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Dir(ro), 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Dir(ro), 0o755) })
	before[filepath.Dir(ro)] = "dir " + os.FileMode(0o555).String()

	out, err := inv.Perform()
	if err != nil {
		t.Fatal(err)
	}
	if out.Fell == nil {
		t.Skip("the hall's seal is not enforced (performing as root?)")
	}
	if out.Fell.Subject != "ro" || out.Fell.Kind != librarium.KindTranscription || out.Fell.Target != ro {
		t.Fatalf("fell at %+v", out.Fell)
	}
	grimdark(t, out.Fell.Heresy.Error())
	var subjects []string
	for _, rv := range out.Reversions {
		if rv.Heresy != nil {
			t.Errorf("reversion of the subject %s failed: %v", rv.Subject, rv.Heresy)
		}
		subjects = append(subjects, rv.Subject)
	}
	if want := []string{"visage", "theme", "created", "util"}; !equalStrings(subjects, want) {
		t.Fatalf("restored subjects %v, want %v", subjects, want)
	}
	sameTree(t, before, tree(t, fx.data))
	if out.Verdict != Reverted {
		t.Fatalf("verdict %q", out.Verdict)
	}
	if exists(created) || sealOf(t, util) != 0o640 || sealOf(t, vis) != 0o600 || anchorOf(t, theme) != "themes/porpl" {
		t.Fatal("the machine was not restored")
	}
}

// TestPerform_RestoresAScrollCastDownByATether: a vessel a zealous tether
// cast down is raised again when a later subject falls.
func TestPerform_RestoresAScrollCastDownByATether(t *testing.T) {
	fx := newFixture(t)
	theme := write(t, fx.data, "current-theme", "hand-written", 0o600)
	write(t, fx.data, "ro/x.conf", "x off", 0o644)
	r := fx.rite(t, `{"name": "theme", "designation": "tether", "tether": "$DATA/current-theme",
	   "omens": {"*": "$DATA/themes/{{aspect}}"}, "zeal": true},
	  {"name": "ro", "designation": "transcription", "vessel": "$DATA/ro/x.conf", "omens": {"*": "x {{aspect}}"}}`)
	inv, err := Prepare(r, Options{Aspect: "on"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(fx.data, "ro"), 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(fx.data, "ro"), 0o755) })
	out, _ := inv.Perform()
	if out.Fell == nil {
		t.Skip("the hall's seal is not enforced (performing as root?)")
	}
	if len(out.Reversions) != 1 || out.Reversions[0].Heresy != nil {
		t.Fatalf("reversions %+v", out.Reversions)
	}
	if read(t, theme) != "hand-written" || sealOf(t, theme) != 0o600 {
		t.Fatal("the vessel cast down by the tether was not raised again")
	}
}

// deed is a performer that records what was asked of it.
type deed struct {
	n          int
	fails      bool  // perform falls
	changes    bool  // perform changes something to revert
	revertErr  error // revert falls
	log        *[]string
	onPerform  func() // called as perform begins
	performed  bool
	wasRevered bool
}

func (d *deed) verse() Verse       { return Verse{Number: d.n, Kind: librarium.KindIncantation, Target: "x"} }
func (d *deed) foresee() Foresight { return Foresight{Verse: d.verse()} }
func (d *deed) perform(context.Context) error {
	*d.log = append(*d.log, "perform")
	d.performed = true
	if d.onPerform != nil {
		d.onPerform()
	}
	if d.fails {
		return errors.New("the deed falls")
	}
	return nil
}

func (d *deed) revert() (bool, error) {
	*d.log = append(*d.log, "revert")
	d.wasRevered = true
	return d.changes, d.revertErr
}

func TestPerformAll_RevertsFromTheFallenStepBackToTheFirst(t *testing.T) {
	var log []string
	falter := errors.New("the reversion falters")
	steps := []*deed{
		{n: 1, changes: true},
		{n: 2, revertErr: falter},
		{n: 3},
		{n: 4, fails: true, changes: true},
		{n: 5, changes: true},
	}
	performers := make([]performer, len(steps))
	for i, s := range steps {
		s.log = &log
		performers[i] = s
	}

	out := performAll(context.Background(), performers, nil, nil)

	if out.Fell == nil || out.Fell.Number != 4 || out.Fell.Heresy == nil {
		t.Fatalf("fell %+v", out.Fell)
	}
	if out.Verdict != Faltered {
		t.Fatalf("verdict %q", out.Verdict)
	}
	if len(out.Deeds) != 4 || out.Deeds[3].Number != 4 || out.Deeds[3].Heresy == nil || out.Deeds[2].Heresy != nil {
		t.Fatalf("deeds %+v", out.Deeds)
	}
	if steps[4].performed || steps[4].wasRevered {
		t.Fatal("a step after the fallen one was touched")
	}
	for _, s := range steps[:4] {
		if !s.wasRevered {
			t.Errorf("verse %d was not reverted", s.n)
		}
	}
	var verses []int
	for _, rv := range out.Reversions {
		verses = append(verses, rv.Number)
	}
	if !equalInts(verses, []int{4, 2, 1}) {
		t.Fatalf("reported reversions %v; steps with nothing to revert are left out", verses)
	}
	if out.Reversions[0].Heresy != nil || !errors.Is(out.Reversions[1].Heresy, falter) || out.Reversions[2].Heresy != nil {
		t.Fatalf("reversion outcomes %+v", out.Reversions)
	}
	if want := []string{"perform", "perform", "perform", "perform", "revert", "revert", "revert", "revert"}; !equalStrings(log, want) {
		t.Fatalf("order %v", log)
	}
}

func TestPerformAll_Triumphs(t *testing.T) {
	var log []string
	out := performAll(context.Background(), []performer{&deed{n: 1, log: &log, changes: true}, &deed{n: 2, log: &log}}, nil, nil)
	if out.Fell != nil || len(out.Reversions) != 0 || len(log) != 2 || out.Verdict != Triumph || len(out.Deeds) != 2 {
		t.Fatalf("outcome %+v, log %v", out, log)
	}
}

func TestPerform_OnlyOnce(t *testing.T) {
	fx := newFixture(t)
	write(t, fx.data, "util.kdl", "", 0o644)
	inv, err := Prepare(fx.rite(t, mouseSanctum), Options{Aspect: "on"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := inv.Perform(); err != nil {
		t.Fatal(err)
	}
	if _, err := inv.Perform(); !errors.Is(err, ErrPerformed) {
		t.Fatalf("second performance: %v", err)
	}
}

// TestForesee_TouchesNothing: the foresight of every subject, named in
// written order, tells what it would change and touches nothing.
func TestForesee_TouchesNothing(t *testing.T) {
	fx := newFixture(t)
	write(t, fx.data, "util.kdl", "input {}\n", 0o640)
	write(t, fx.data, "visage.conf", "visage off", 0o600)
	link(t, "themes/porpl", filepath.Join(fx.data, "current-theme"))
	r := fx.rite(t, `{"name": "util", "designation": "sanctum", "vessel": "$DATA/util.kdl", "omens": {"*": "1"}},
	  {"name": "created", "designation": "sanctum", "vessel": "$DATA/created.kdl", "consecrate": true, "omens": {"*": "2"}},
	  {"name": "theme", "designation": "tether", "tether": "$DATA/current-theme", "omens": {"*": "$DATA/themes/{{aspect}}"}},
	  {"name": "visage", "designation": "transcription", "vessel": "$DATA/visage.conf", "omens": {"*": "visage {{aspect}}"},
	   "seal": "0644"}`)
	before := tree(t, fx.data)

	inv, err := Prepare(r, Options{Aspect: "on"})
	if err != nil {
		t.Fatal(err)
	}
	seen := inv.Foresee()
	sameTree(t, before, tree(t, fx.data))

	if len(seen) != 4 {
		t.Fatalf("foresaw %d subjects", len(seen))
	}
	for i, name := range []string{"util", "created", "theme", "visage"} {
		if seen[i].Subject != name {
			t.Errorf("foresight %d names the subject %q, want %q", i, seen[i].Subject, name)
		}
	}
	if d := seen[0].Vessel.Diff(); !equalStrings(d, []string{"+ // +++ begin of sanctum mouse -- aspect|on +++",
		"+ 1", "+ // +++ end of sanctum mouse +++", "+ "}) {
		t.Errorf("sanctum diff %q", d)
	}
	if c := seen[1].Vessel; c.Existed || !c.Exists {
		t.Errorf("consecration foreseen as %+v", c)
	}
	if c := seen[2].Tether; c.From != "themes/porpl" || c.To != filepath.Join(fx.data, "themes", "on") {
		t.Errorf("tether foreseen as %+v", c)
	}
	if c := seen[3].Vessel; c.BeforeSeal != 0o600 || c.AfterSeal != 0o644 || !equalStrings(c.Diff(), []string{"- visage off", "+ visage on"}) {
		t.Errorf("transcription foreseen as %+v", c)
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestPerformAll_HaltedBetweenStepsBeginsNoMore(t *testing.T) {
	var log []string
	ctx, halt := context.WithCancel(context.Background())
	defer halt()
	first := &deed{n: 1, changes: true, log: &log, onPerform: halt}
	second := &deed{n: 2, changes: true, log: &log}
	third := &deed{n: 3, log: &log}

	out := performAll(ctx, []performer{first, second, third}, nil, nil)

	if second.performed || second.wasRevered || third.performed {
		t.Fatalf("a step was begun or reverted after the halt: log %v", log)
	}
	if out.Fell == nil || out.Fell.Number != 2 || out.Fell.Heresy == nil || out.Verdict != Reverted {
		t.Fatalf("outcome %+v", out)
	}
	grimdark(t, out.Fell.Heresy.Error())
	if len(out.Reversions) != 1 || out.Reversions[0].Number != 1 || !first.wasRevered {
		t.Fatalf("reversions %+v", out.Reversions)
	}
	if len(out.Deeds) != 2 || out.Deeds[1].Heresy == nil {
		t.Fatalf("deeds %+v", out.Deeds)
	}
}
