package invocation

import (
	"path/filepath"
	"testing"
)

const visage = `{"transcription": "$DATA/visage.conf", "scripture": {"off": null, "*": "visage {{aspect}}"}`

func TestTranscription_WritesRewritesAndStrikes(t *testing.T) {
	fx := newFixture(t)
	p := filepath.Join(fx.data, "visage.conf")
	r := fx.rite(t, visage+`}`)

	seen := invoke(t, r, "on")
	if read(t, p) != "visage on" || sealOf(t, p) != 0o644 {
		t.Fatalf("new vessel holds %q sealed %v", read(t, p), sealOf(t, p))
	}
	if c := seen[0].Vessel; c.Existed || !c.Exists || c.After != "visage on" {
		t.Fatalf("foresight %+v", c)
	}

	seen = invoke(t, r, "off")
	if exists(p) {
		t.Fatal("a null scripture did not strike the vessel")
	}
	if c := seen[0].Vessel; !c.Existed || c.Exists || c.Before != "visage on" {
		t.Fatalf("foresight %+v", c)
	}

	if seen := invoke(t, r, "off"); seen[0].Vessel.Changed() {
		t.Fatal("striking an absent vessel changed something")
	}
}

func TestTranscription_Seals(t *testing.T) {
	fx := newFixture(t)
	p := write(t, fx.data, "visage.conf", "visage off", 0o640)

	invoke(t, fx.rite(t, `{"transcription": "$DATA/visage.conf", "scripture": "visage {{aspect}}"}`), "on")
	if read(t, p) != "visage on" || sealOf(t, p) != 0o640 {
		t.Fatalf("got %q sealed %v; the seal must be kept", read(t, p), sealOf(t, p))
	}

	r := fx.rite(t, `{"transcription": "$DATA/visage.conf", "scripture": "visage {{aspect}}", "seal": "0600"}`)
	seen := invoke(t, r, "on")
	if sealOf(t, p) != 0o600 || !seen[0].Vessel.Changed() {
		t.Fatalf("seal %v, foresight %+v", sealOf(t, p), seen[0].Vessel)
	}

	q := filepath.Join(fx.data, "new.conf")
	invoke(t, fx.rite(t, `{"transcription": "$DATA/new.conf", "scripture": "x", "seal": "600"}`), "on")
	if sealOf(t, q) != 0o600 {
		t.Fatalf("new vessel sealed %v", sealOf(t, q))
	}
}

func TestTranscription_ReplacesOnlyItsOwnVessels(t *testing.T) {
	cases := map[string]struct {
		vessel  string
		liturgy string
		opts    Options
		want    string // "" when the invocation triumphs
	}{
		"scripture of another aspect":        {"visage off", uniform, Options{Aspect: "on"}, ""},
		"with one final newline":             {"visage off\n", uniform, Options{Aspect: "on"}, ""},
		"scripture of no aspect":             {"visage porpl", uniform, Options{Aspect: "on"}, "zeal"},
		"foreign scripture":                  {"hand-written", visage + `}`, Options{Aspect: "on"}, "zeal"},
		"foreign scripture struck":           {"hand-written", visage + `}`, Options{Aspect: "off"}, "zeal"},
		"foreign scripture with zeal":        {"hand-written", visage + `, "zeal": true}`, Options{Aspect: "on"}, ""},
		"foreign scripture with force":       {"hand-written", visage + `, "force": true}`, Options{Aspect: "off"}, ""},
		"recorded inscriptions":              {"why old", reasoned, Options{Aspect: "on", Inscriptions: map[string]string{"reason": "new"}, Recorded: map[string]string{"reason": "old"}}, ""},
		"inscriptions never recorded":        {"why old", reasoned, Options{Aspect: "on", Inscriptions: map[string]string{"reason": "new"}}, "zeal"},
		"inscriptions left empty":            {"why ", reasoned, Options{Aspect: "on", Inscriptions: map[string]string{"reason": "new"}}, "zeal"},
		"scripture of this very invocation":  {"why new", reasoned, Options{Aspect: "off", Inscriptions: map[string]string{"reason": "new"}}, ""},
		"scripture of an aspect from a tome": {"tome on\n", tomed, Options{Aspect: "off"}, ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			fx := newFixture(t)
			write(t, fx.lib, "rites/tomes/on.conf", "tome {{aspect}}\n", 0o644)
			write(t, fx.data, "visage.conf", tc.vessel, 0o644)
			r := fx.rite(t, tc.liturgy)
			if tc.want == "" {
				inv, err := Prepare(r, tc.opts)
				if err != nil {
					t.Fatal(err)
				}
				if out, err := inv.Perform(); err != nil || out.Fell != nil {
					t.Fatalf("%v %+v", err, out.Fell)
				}
				return
			}
			before := tree(t, fx.data)
			hs := refused(t, r, tc.opts, tc.want)
			sameTree(t, before, tree(t, fx.data))
			grimdark(t, hs[0].Message)
		})
	}
}

const (
	uniform  = `{"transcription": "$DATA/visage.conf", "scripture": "visage {{aspect}}"}`
	reasoned = `{"transcription": "$DATA/visage.conf", "scripture": "why {{inscription.reason}}"}`
	tomed    = `{"transcription": "$DATA/visage.conf", "scripture": {"on": {"tome": "tomes/on.conf", "illuminate": true}, "*": "plain"}}`
)

func TestTranscription_FollowsBonds(t *testing.T) {
	fx := newFixture(t)
	real := write(t, fx.data, "dotfiles/visage.conf", "visage off", 0o600)
	name := filepath.Join(fx.data, "visage.conf")
	link(t, real, name)
	r := fx.rite(t, uniform)

	invoke(t, r, "on")
	if anchorOf(t, name) != real || read(t, real) != "visage on" || sealOf(t, real) != 0o600 {
		t.Fatalf("the bond was not followed: %q", read(t, real))
	}
	r = fx.rite(t, visage+`}`)
	invoke(t, r, "off")
	if anchorOf(t, name) != real || exists(real) {
		t.Fatal("striking a bonded vessel must strike what the bond leads to and keep the bond")
	}
	invoke(t, r, "on")
	if read(t, real) != "visage on" {
		t.Fatal("the vessel was not written anew where the bond leads")
	}
}

func TestTranscription_Refusals(t *testing.T) {
	cases := map[string]struct{ liturgy, want string }{
		"absent hall":      {`{"transcription": "$DATA/absent/visage.conf", "scripture": "x"}`, "hall"},
		"vessel is a hall": {`{"transcription": "$DATA", "scripture": "x"}`, "no vessel"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			fx := newFixture(t)
			before := tree(t, fx.data)
			hs := refused(t, fx.rite(t, tc.liturgy), Options{Aspect: "on"}, tc.want)
			sameTree(t, before, tree(t, fx.data))
			grimdark(t, hs[0].Message)
		})
	}
}
