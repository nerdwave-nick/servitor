package chronicle

import (
	"encoding/json/v2"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// moment is 2026-10-10T13:30:00Z seen from another zone, with a fraction
// the chronicle does not keep.
var moment = time.Date(2026, 10, 10, 15, 30, 0, 123456789, time.FixedZone("CEST", 2*60*60))

func verse(n int, k librarium.Kind, target string) invocation.Verse {
	return invocation.Verse{Number: n, Kind: k, Target: target}
}

func TestNew_WritesTheLineOfTheCodex(t *testing.T) {
	opts := invocation.Options{Aspect: "porpl", Former: "default", Inscriptions: map[string]string{"reason": "gaming remnant"}}
	fell := errors.New("the incantation fell silent")
	cases := []struct {
		name string
		opts invocation.Options
		out  invocation.Outcome
		want string
	}{
		{
			name: "triumph",
			opts: opts,
			out: invocation.Outcome{Verdict: invocation.Triumph, Deeds: []invocation.Deed{
				{Verse: verse(1, librarium.KindTether, "/x/current-theme")},
			}},
			want: `{"at":"2026-10-10T13:30:00Z","rite":"theme","aspect":"porpl","former":"default",` +
				`"inscriptions":{"reason":"gaming remnant"},"verdict":"triumph"}`,
		},
		{
			name: "reverted",
			opts: opts,
			out: invocation.Outcome{
				Verdict: invocation.Reverted,
				Fell:    &invocation.Fall{Verse: verse(3, librarium.KindIncantation, "source x/hooks"), Heresy: fell, Output: "silence"},
				Reversions: []invocation.Reversion{
					{Verse: verse(3, librarium.KindIncantation, "source x/hooks")},
					{Verse: verse(2, librarium.KindSanctum, "/x/hooks")},
				},
			},
			want: `{"at":"2026-10-10T13:30:00Z","rite":"theme","aspect":"porpl","former":"default",` +
				`"inscriptions":{"reason":"gaming remnant"},"verdict":"reverted",` +
				`"fell_at":{"verse":3,"incantation":"source x/hooks"},"heresy":"the incantation fell silent",` +
				`"reversions":[{"verse":3,"verdict":"triumph"},{"verse":2,"verdict":"triumph"}]}`,
		},
		{
			name: "faltered",
			opts: opts,
			out: invocation.Outcome{
				Verdict: invocation.Faltered,
				Fell:    &invocation.Fall{Verse: verse(2, librarium.KindTether, "/x/current-theme"), Heresy: fell},
				Reversions: []invocation.Reversion{
					{Verse: verse(2, librarium.KindTether, "/x/current-theme")},
					{Verse: verse(1, librarium.KindLitany, "/x/hooks.sh"), Heresy: errors.New("the litany choked")},
				},
			},
			want: `{"at":"2026-10-10T13:30:00Z","rite":"theme","aspect":"porpl","former":"default",` +
				`"inscriptions":{"reason":"gaming remnant"},"verdict":"faltered",` +
				`"fell_at":{"verse":2,"tether":"/x/current-theme"},"heresy":"the incantation fell silent",` +
				`"reversions":[{"verse":2,"verdict":"triumph"},{"verse":1,"verdict":"faltered"}]}`,
		},
		{
			name: "first invocation without inscriptions",
			opts: invocation.Options{Aspect: "porpl"},
			out:  invocation.Outcome{Verdict: invocation.Triumph},
			want: `{"at":"2026-10-10T13:30:00Z","rite":"theme","aspect":"porpl","inscriptions":{},"verdict":"triumph"}`,
		},
		{
			name: "edict proclaimed",
			opts: invocation.Options{Edict: true, Inscriptions: map[string]string{"region": "full"}},
			out:  invocation.Outcome{Verdict: invocation.Triumph},
			want: `{"at":"2026-10-10T13:30:00Z","rite":"theme","edict":true,"inscriptions":{"region":"full"},"verdict":"triumph"}`,
		},
		{
			name: "edict faltered",
			opts: invocation.Options{Edict: true},
			out: invocation.Outcome{
				Verdict: invocation.Faltered,
				Fell:    &invocation.Fall{Verse: verse(2, librarium.KindIncantation, "exit 3"), Heresy: fell},
				Reversions: []invocation.Reversion{
					{Verse: verse(1, librarium.KindIncantation, "true"), Heresy: errors.New("the reversion fell")},
				},
			},
			want: `{"at":"2026-10-10T13:30:00Z","rite":"theme","edict":true,"inscriptions":{},"verdict":"faltered",` +
				`"fell_at":{"verse":2,"incantation":"exit 3"},"heresy":"the incantation fell silent",` +
				`"reversions":[{"verse":1,"verdict":"faltered"}]}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := New(moment, "theme", tc.opts, tc.out)
			line, err := json.Marshal(e)
			if err != nil {
				t.Fatal(err)
			}
			if string(line) != tc.want {
				t.Fatalf("line\n got %s\nwant %s", line, tc.want)
			}
			var back Entry
			if err := json.Unmarshal(line, &back); err != nil {
				t.Fatalf("read back: %v", err)
			}
			if !reflect.DeepEqual(back, e) {
				t.Fatalf("read back\n got %#v\nwant %#v", back, e)
			}
		})
	}
}

func TestNew_KeepsItsOwnInscriptions(t *testing.T) {
	ins := map[string]string{"reason": "gaming remnant"}
	e := New(moment, "theme", invocation.Options{Aspect: "porpl", Inscriptions: ins}, invocation.Outcome{Verdict: invocation.Triumph})
	ins["reason"] = "altered afterwards"
	if e.Inscriptions["reason"] != "gaming remnant" {
		t.Fatalf("entry shares the caller's inscriptions: %v", e.Inscriptions)
	}
}

func TestVerse_ReadsWhatItCannotName(t *testing.T) {
	cases := []struct {
		name, line string
		want       Verse
		bad        bool
	}{
		{"key before verse", `{"tether":"/x","verse":2}`, Verse{Number: 2, Key: "tether", Target: "/x"}, false},
		{"unknown key", `{"verse":4,"censer":"thurible"}`, Verse{Number: 4, Key: "censer", Target: "thurible"}, false},
		{"two keys", `{"verse":4,"tether":"/x","litany":"/y"}`, Verse{}, true},
		{"not an object", `[4]`, Verse{}, true},
		{"target not a string", `{"verse":4,"tether":7}`, Verse{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var v Verse
			err := json.Unmarshal([]byte(tc.line), &v)
			if tc.bad {
				if err == nil {
					t.Fatalf("accepted %s as %#v", tc.line, v)
				}
				return
			}
			if err != nil || v != tc.want {
				t.Fatalf("got %#v, %v; want %#v", v, err, tc.want)
			}
		})
	}
}

// TestEntry_NamesTheIntercessorAndTheSigilOfExaltedVerses: the line of an
// invocation with exalted verses names, last, the intercessor that vouched
// for its master and the first twelve letters of the sigil of the plan the
// exalted servitor performed; it reads back whole.
func TestEntry_NamesTheIntercessorAndTheSigilOfExaltedVerses(t *testing.T) {
	e := New(moment, "split-lock", invocation.Options{Aspect: "on", Former: "off"},
		invocation.Outcome{Verdict: invocation.Reverted,
			Fell:       &invocation.Fall{Verse: verse(2, librarium.KindIncantation, "exit 1"), Heresy: errors.New("fell")},
			Reversions: []invocation.Reversion{{Verse: verse(1, librarium.KindIncantation, "sysctl -w x=1")}}})
	e.Intercessor, e.Sigil = "sudo", "4f0c2a9e1b7d"
	line, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"at":"2026-10-10T13:30:00Z","rite":"split-lock","aspect":"on","former":"off","inscriptions":{},` +
		`"verdict":"reverted","fell_at":{"verse":2,"incantation":"exit 1"},"heresy":"fell",` +
		`"reversions":[{"verse":1,"verdict":"triumph"}],"intercessor":"sudo","sigil":"4f0c2a9e1b7d"}`
	if string(line) != want {
		t.Fatalf("line\n got %s\nwant %s", line, want)
	}
	var back Entry
	if err := json.Unmarshal(line, &back); err != nil || !reflect.DeepEqual(back, e) {
		t.Fatalf("read back %#v, %v", back, err)
	}
}
