package invocation

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// exaltedObservances are spoken before the subject x (see spoken): a
// common observance v1, an exalted one v2 that would leave $DATA/marker
// behind were it ever spoken by the invoker, and a common one v3 that falls
// when $DATA/fall stands; each common observance reverts by word.
var exaltedObservances = []string{
	`{"designation": "incantation", "words": "echo 1 >> $DATA/said", "reversion": "echo undo 1 >> $DATA/said"}`,
	`{"designation": "incantation", "words": "touch $DATA/marker", "reversion": "touch $DATA/marker", "exalted": true}`,
	`{"designation": "incantation", "words": "test ! -e $DATA/fall && echo 3 >> $DATA/said",
	  "reversion": "echo undo 3 >> $DATA/said"}`,
}

// altar stands in for the exalted servitor: it performs and reverts the
// exalted verses it is bidden by writing to the scroll the common verses
// write, and never speaks their words.
type altar struct {
	said     string
	fell     map[string]bool // the items that fall, as named
	changed  bool            // a fallen verse changed the machine all the same
	faltered bool            // reversions falter
	unended  error           // the finish is not heard
	calls    []string
	// omens are the omens the machine-spirits of a state rite are set to,
	// by name, written to <data>/<name> as the exalted servitor would.
	omens map[string]string
	data  string
}

// named names the verse, or the subject or observance, of v.
func named(v Verse) string {
	if v.Subject+v.Observance != "" {
		return v.Subject + v.Observance
	}
	return fmt.Sprint(v.Number)
}

func (a *altar) Enact(v Verse, restore string) (string, bool, error) {
	a.note("enact %s restoring %s", v.Subject, restore)
	if omen, ok := a.omens[v.Subject]; ok {
		_ = os.WriteFile(filepath.Join(a.data, v.Subject), []byte(omen+"\n"), 0o644)
	}
	return "enacted", true, nil
}

func (a *altar) Restore(v Verse, omen string) (string, bool, error) {
	a.note("restore %s to %s", v.Subject, omen)
	_ = os.WriteFile(filepath.Join(a.data, v.Subject), []byte(omen+"\n"), 0o644)
	return "restored", true, nil
}

func (a *altar) note(format string, args ...any) {
	a.calls = append(a.calls, fmt.Sprintf(format, args...))
	f, err := os.OpenFile(a.said, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err == nil {
		_, _ = fmt.Fprintf(f, format+"\n", args...)
		_ = f.Close()
	}
}

func (a *altar) Perform(v Verse) (string, bool, error) {
	if a.fell[named(v)] {
		a.note("exalted %s fell", named(v))
		return "fallen words", a.changed, errors.New("the exalted verse fell")
	}
	a.note("exalted %s", named(v))
	return fmt.Sprintf("heard %s", named(v)), true, nil
}

func (a *altar) Revert(v Verse) (string, bool, error) {
	a.note("undo exalted %s", named(v))
	if a.faltered {
		return "", true, errors.New("the exalted reversion fell")
	}
	return fmt.Sprintf("unheard %s", named(v)), true, nil
}

func (a *altar) Finish() error {
	a.calls = append(a.calls, "finish")
	return a.unended
}

// TestPerform_NeverPerformsAnExaltedObservance: the invoker's servitor
// performs no exalted observance itself. An invocation holding one may be
// prepared and foreseen, but performed only through an exalted servitor:
// without one it is refused, and nothing is touched.
func TestPerform_NeverPerformsAnExaltedObservance(t *testing.T) {
	fx := newFixture(t)
	inv, err := Prepare(fx.spoken(t, "", exaltedObservances...), Options{Aspect: "on"})
	if err != nil {
		t.Fatal(err)
	}
	if f := inv.Foresee(); len(f) != 4 || !f[1].Exalted || f[0].Exalted || f[2].Exalted || f[3].Exalted {
		t.Errorf("foresight %+v", f)
	}
	for name, perform := range map[string]func() (Outcome, error){
		"Perform":             inv.Perform,
		"PerformExalted(nil)": func() (Outcome, error) { return inv.PerformExalted(t.Context(), nil) },
	} {
		_, err := perform()
		var hs Heresies
		if !errors.As(err, &hs) || len(hs) != 1 || !strings.Contains(hs[0].Message, `the observance "v2" is exalted`) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, p := range []string{"said", "marker"} {
		if exists(filepath.Join(fx.data, p)) {
			t.Errorf("%s was touched", p)
		}
	}
}

// TestPerformExalted_InWrittenOrder: common and exalted observances are
// performed in written order, the exalted ones by the exalted servitor
// alone, whose words are the deed's; its work is finished once.
func TestPerformExalted_InWrittenOrder(t *testing.T) {
	fx := newFixture(t)
	said := filepath.Join(fx.data, "said")
	a := &altar{said: said}
	inv, err := Prepare(fx.spoken(t, "", exaltedObservances...), Options{Aspect: "on"})
	if err != nil {
		t.Fatal(err)
	}
	out, err := inv.PerformExalted(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	if out.Verdict != Triumph || len(out.Deeds) != 4 || out.Deeds[1].Output != "heard v2" {
		t.Errorf("outcome %+v", out)
	}
	if got := lines(t, said); !slices.Equal(got, []string{"1", "exalted v2", "3"}) {
		t.Errorf("said %q", got)
	}
	if !slices.Equal(a.calls, []string{"exalted v2", "finish"}) || exists(filepath.Join(fx.data, "marker")) {
		t.Errorf("the exalted servitor was bidden %q; marker %v", a.calls, exists(filepath.Join(fx.data, "marker")))
	}
}

// TestPerformExalted_AFallRevertsBothKindsInReverse: when an observance
// falls, the exalted ones performed are reverted by the exalted servitor
// among the common ones, from the fallen one back to the first; an exalted
// observance that fell without changing anything is not reverted.
func TestPerformExalted_AFallRevertsBothKindsInReverse(t *testing.T) {
	for _, c := range []struct {
		name     string
		altar    altar
		fall     bool
		said     []string
		verdict  Verdict
		reverted []string
	}{
		{"a common observance falls", altar{}, true,
			[]string{"1", "exalted v2", "undo 3", "undo exalted v2", "undo 1"}, Reverted, []string{"v3", "v2", "v1"}},
		{"the exalted observance falls having changed", altar{fell: map[string]bool{"v2": true}, changed: true}, false,
			[]string{"1", "exalted v2 fell", "undo exalted v2", "undo 1"}, Reverted, []string{"v2", "v1"}},
		{"the exalted observance falls unchanged", altar{fell: map[string]bool{"v2": true}}, false,
			[]string{"1", "exalted v2 fell", "undo 1"}, Reverted, []string{"v1"}},
		{"the exalted reversion falters", altar{faltered: true}, true,
			[]string{"1", "exalted v2", "undo 3", "undo exalted v2", "undo 1"}, Faltered, []string{"v3", "v2", "v1"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			fx := newFixture(t)
			if c.fall {
				write(t, fx.data, "fall", "", 0o644)
			}
			a := c.altar
			a.said = filepath.Join(fx.data, "said")
			inv, err := Prepare(fx.spoken(t, "", exaltedObservances...), Options{Aspect: "on"})
			if err != nil {
				t.Fatal(err)
			}
			out, err := inv.PerformExalted(t.Context(), &a)
			if err != nil {
				t.Fatal(err)
			}
			var reverted []string
			for _, r := range out.Reversions {
				reverted = append(reverted, named(r.Verse))
			}
			if out.Verdict != c.verdict || !slices.Equal(reverted, c.reverted) {
				t.Errorf("verdict %s, reverted %v", out.Verdict, reverted)
			}
			if got := lines(t, a.said); !slices.Equal(got, c.said) {
				t.Errorf("said %q, want %q", got, c.said)
			}
			if a.calls[len(a.calls)-1] != "finish" || exists(filepath.Join(fx.data, "marker")) {
				t.Errorf("the exalted servitor was bidden %q", a.calls)
			}
		})
	}
}

// TestPerformExalted_AnUnheardFinishRevertsTheWhole: an exalted servitor
// that never hears the finish reverts what it performed; so the invocation
// is reverted whole — the subject x restored by enacting the omen of the
// former aspect — and falls at its last exalted observance.
func TestPerformExalted_AnUnheardFinishRevertsTheWhole(t *testing.T) {
	fx := newFixture(t)
	a := &altar{said: filepath.Join(fx.data, "said"), unended: errors.New("the channel ended")}
	inv, err := Prepare(fx.spoken(t, "", exaltedObservances...), Options{Aspect: "on", Former: "off"})
	if err != nil {
		t.Fatal(err)
	}
	out, err := inv.PerformExalted(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	if out.Verdict != Reverted || out.Fell == nil || out.Fell.Observance != "v2" || !strings.Contains(out.Fell.Heresy.Error(),
		"the channel ended") {
		t.Errorf("outcome %+v", out)
	}
	if got := lines(t, a.said); !slices.Equal(got, []string{"1", "exalted v2", "3", "undo 3", "undo exalted v2", "undo 1"}) {
		t.Errorf("said %q", got)
	}
}

// TestForesee_AVesselHiddenFromTheInvoker: an exalted vessel its invoker may
// not read is foreseen by its owner, seal and size, and by the digest of
// what a transcription shall write: its change is unknown, never
// "unchanged". A common verse may not touch such a vessel at all.
func TestForesee_AVesselHiddenFromTheInvoker(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("the root reads every vessel")
	}
	fx := newFixture(t)
	hidden := write(t, fx.data, "hidden.conf", "secret\n", 0o200)
	veiled := filepath.Join(fx.data, "veiled", "x.conf")
	write(t, fx.data, "veiled/x.conf", "secret\n", 0o600)
	if err := os.Chmod(filepath.Dir(veiled), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Dir(veiled), 0o755) })
	r := fx.rite(t, `
  {"name": "a", "designation": "transcription", "vessel": "`+hidden+`", "omens": {"*": "kernel.x = {{aspect}}"},
   "exalted": true},
  {"name": "b", "designation": "sanctum", "vessel": "`+hidden+`", "omens": {"*": "y = 1"}, "exalted": true},
  {"name": "c", "designation": "transcription", "vessel": "`+veiled+`", "omens": {"*": "z"}, "exalted": true}`)
	inv, err := Prepare(r, Options{Aspect: "on"})
	if err != nil {
		t.Fatal(err)
	}
	me, _ := user.Current()
	group, _ := user.LookupGroupId(strconv.Itoa(os.Getgid()))
	sum, z := sha256.Sum256([]byte("kernel.x = on")), sha256.Sum256([]byte("z"))
	f := inv.Foresee()
	for i, want := range []VesselChange{
		{Path: hidden, Hidden: true, Existed: true, Exists: true, Owner: me.Username + ":" + group.Name, Size: 7,
			BeforeSeal: 0o200, AfterSeal: 0o200, Digest: hex.EncodeToString(sum[:])},
		// the transcription before it leaves its size unknown
		{Path: hidden, Hidden: true, Existed: true, Exists: true, Owner: me.Username + ":" + group.Name, Size: -1,
			BeforeSeal: 0o200, AfterSeal: 0o200},
		{Path: veiled, Hidden: true, Exists: true, Size: -1, Digest: hex.EncodeToString(z[:])},
	} {
		got := f[i].Vessel
		if got == nil || got.Hidden != want.Hidden || got.Path != want.Path || got.Owner != want.Owner ||
			got.Size != want.Size || got.Digest != want.Digest || got.BeforeSeal != want.BeforeSeal ||
			got.AfterSeal != want.AfterSeal || got.Before != "" || got.After != "" || !got.Changed() ||
			len(got.Diff()) != 0 {
			t.Errorf("subject %d foreseen %+v\nwant %+v", i+1, got, want)
		}
	}

	r = fx.rite(t, `{"name": "a", "designation": "transcription", "vessel": "`+hidden+`",
	  "omens": {"*": "kernel.x = {{aspect}}"}}`)
	if _, err := Prepare(r, Options{Aspect: "on"}); err == nil || !strings.Contains(err.Error(), "denies") {
		t.Errorf("a common subject upon a hidden vessel: %v", err)
	}
}

// TestPrepare_ExaltedSpeechKnowsItsTongues: the pre-flight refuses an
// exalted observance spoken in a tongue exalted speech does not know, by
// the name of the program it truly is; a common observance may speak it.
func TestPrepare_ExaltedSpeechKnowsItsTongues(t *testing.T) {
	fx := newFixture(t)
	bin := t.TempDir()
	write(t, bin, "python3", "#!/bin/sh\n", 0o755)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	r := fx.spoken(t, "", `{"designation": "incantation", "words": "print(1)", "tongue": "python3"}`,
		`{"designation": "incantation", "words": "print(2)", "tongue": "python3", "exalted": true}`,
		`{"designation": "incantation", "words": "echo 3", "tongue": "sh", "exalted": true}`)
	_, err := Prepare(r, Options{Aspect: "on"})
	var hs Heresies
	if !errors.As(err, &hs) || len(hs) != 1 || !strings.Contains(hs[0].Message, "python3") ||
		!strings.Contains(hs[0].Message, `the exalted observance "v2"`) || strings.Contains(hs[0].Message, "verse") ||
		r.Locate("/observances/1/exalted") != hs[0].Position {
		t.Fatalf("pre-flight: %v", err)
	}
}

// TestForesee_ANameHiddenFromTheInvoker: an exalted tether whose name its
// invoker may not behold is foreseen with its change unknown; a common
// tether may not bind it.
func TestForesee_ANameHiddenFromTheInvoker(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("the root beholds every name")
	}
	fx := newFixture(t)
	hall := filepath.Join(fx.data, "veiled")
	write(t, fx.data, "veiled/x", "", 0o644)
	if err := os.Chmod(hall, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(hall, 0o755) })
	inv, err := Prepare(fx.rite(t, `{"name": "t", "designation": "tether", "tether": "`+hall+`/current",
	  "omens": {"*": "/usr/share/x"}, "exalted": true}`), Options{Aspect: "on"})
	if err != nil {
		t.Fatal(err)
	}
	if c := inv.Foresee()[0].Tether; c == nil || !c.Hidden || !c.Changed() || c.From != "" || c.To != "/usr/share/x" {
		t.Errorf("foreseen %+v", c)
	}
	_, err = Prepare(fx.rite(t, `{"name": "t", "designation": "tether", "tether": "`+hall+`/current",
	  "omens": {"*": "/usr/share/x"}}`), Options{Aspect: "on"})
	if err == nil || !strings.Contains(err.Error(), "denies") {
		t.Errorf("a common tether upon a hidden name: %v", err)
	}
}
