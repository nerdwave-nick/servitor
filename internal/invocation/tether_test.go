package invocation

import (
	"os"
	"path/filepath"
	"testing"
)

const theme = `{"tether": "$DATA/current-theme", "anchor": {"off": null, "*": "$DATA/themes/{{aspect}}"}`

func TestTether_BindsRebindsAndUnbinds(t *testing.T) {
	fx := newFixture(t)
	name := filepath.Join(fx.data, "current-theme")
	for _, a := range []string{"on", "porpl"} {
		if err := os.MkdirAll(filepath.Join(fx.data, "themes", a), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	r := fx.rite(t, theme+`}`)

	seen := invoke(t, r, "on")
	on := filepath.Join(fx.data, "themes", "on")
	if anchorOf(t, name) != on {
		t.Fatalf("bound to %q", anchorOf(t, name))
	}
	if c := seen[0].Tether; c == nil || c.From != "" || c.To != on || !c.Changed() || seen[0].Target != name {
		t.Fatalf("foresight %+v", seen[0])
	}

	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	link(t, "themes/porpl", name)
	seen = invoke(t, r, "on")
	if anchorOf(t, name) != on || seen[0].Tether.From != "themes/porpl" {
		t.Fatalf("not rebound: %q, foresight %+v", anchorOf(t, name), seen[0].Tether)
	}

	if seen := invoke(t, r, "on"); seen[0].Tether.Changed() {
		t.Fatalf("binding the same anchor again changed %+v", seen[0].Tether)
	}

	seen = invoke(t, r, "off")
	if exists(name) || seen[0].Tether.To != "" || seen[0].Tether.From != on {
		t.Fatalf("not unbound; foresight %+v", seen[0].Tether)
	}
	if seen := invoke(t, r, "off"); seen[0].Tether.Changed() {
		t.Fatal("unbinding an absent tether changed something")
	}
}

func TestTether_ResolvesRelativeAnchorsAgainstTheRite(t *testing.T) {
	fx := newFixture(t)
	name := filepath.Join(fx.data, "current-theme")
	invoke(t, fx.rite(t, `{"tether": "$DATA/current-theme", "anchor": "themes/{{aspect}}"}`), "on")
	if want := filepath.Join(fx.lib, "rites", "themes", "on"); anchorOf(t, name) != want {
		t.Fatalf("bound to %q, want %q", anchorOf(t, name), want)
	}
}

func TestTether_ReplacesOnlyBondsUnlessZealous(t *testing.T) {
	fx := newFixture(t)
	name := write(t, fx.data, "current-theme", "hand-written", 0o600)

	hs := refused(t, fx.rite(t, theme+`}`), Options{Aspect: "on"}, "zeal")
	grimdark(t, hs[0].Message)
	refused(t, fx.rite(t, theme+`}`), Options{Aspect: "off"}, "zeal")
	if read(t, name) != "hand-written" {
		t.Fatal("the vessel was touched")
	}

	seen := invoke(t, fx.rite(t, theme+`, "zeal": true}`), "on")
	if anchorOf(t, name) != filepath.Join(fx.data, "themes", "on") || !seen[0].Tether.Displaced {
		t.Fatalf("not replaced with zeal; foresight %+v", seen[0].Tether)
	}

	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	write(t, fx.data, "current-theme", "again", 0o600)
	invoke(t, fx.rite(t, theme+`, "force": true}`), "off")
	if exists(name) {
		t.Fatal("not struck with force")
	}
}

func TestTether_Refusals(t *testing.T) {
	cases := map[string]struct {
		hall    string // a directory to raise within the vessels' place
		liturgy string
		want    string
	}{
		"hall at the name":         {"current-theme/inner", theme + `, "zeal": true}`, "hall stands"},
		"absent hall for the name": {"", `{"tether": "$DATA/absent/current-theme", "anchor": "x"}`, "is absent"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			fx := newFixture(t)
			if tc.hall != "" {
				if err := os.MkdirAll(filepath.Join(fx.data, tc.hall), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			before := tree(t, fx.data)
			hs := refused(t, fx.rite(t, tc.liturgy), Options{Aspect: "on"}, tc.want)
			sameTree(t, before, tree(t, fx.data))
			grimdark(t, hs[0].Message)
		})
	}
}
