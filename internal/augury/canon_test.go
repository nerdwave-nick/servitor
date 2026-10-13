package augury

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// scripture writes the rite name of the given scripture into the Librarium
// and loads it.
func (fx fixture) scripture(t *testing.T, name, body string) *librarium.Rite {
	t.Helper()
	path := filepath.Join(fx.lib, "rites", name+".json")
	write(t, path, body)
	r, findings := librarium.LoadFile(path)
	if findings.Heretical() {
		t.Fatalf("rite is heretical:\n%v", findings)
	}
	return r
}

// canon is the binharic "canon" of the augury a, compacted; "" when the
// augury bears none.
func canon(t *testing.T, a Augury) string {
	t.Helper()
	var keys map[string]json.RawMessage
	if err := json.Unmarshal([]byte(marshal(t, a)), &keys); err != nil {
		t.Fatal(err)
	}
	raw, ok := keys["canon"]
	if !ok {
		return ""
	}
	return compact(t, string(raw))
}

// TestAugur_TheCanonDeclaresEveryInscriptionAsWritten: the augury of a rite
// bears its canon — every inscription it declares, in written order, with
// its purpose ("" when unwritten), whether it is mandatory, its decrees
// expanded for every aspect ("*" serving each aspect not named, aspects
// without a decree absent) and a decree that is always null.
func TestAugur_TheCanonDeclaresEveryInscriptionAsWritten(t *testing.T) {
	fx := newFixture(t)
	r := fx.scripture(t, "focus", `{
  "pattern": "Mark II",
  "aspects": ["on", "off", "idle"],
  "inscriptions": {
    "reason": {"purpose": "why the rite was invoked", "mandatory": true, "decrees": {"on": "focus", "*": "rest"}},
    "mode": {"decrees": {"on": "auto"}},
    "place": {"purpose": "where", "decrees": {"*": "home"}},
    "note": {},
  },
  "subjects": [`+chant+`]
}`)
	want := compact(t, `[
  {"key": "reason", "purpose": "why the rite was invoked", "mandatory": true,
   "decrees": {"idle": "rest", "off": "rest", "on": "focus"}, "decree": null},
  {"key": "mode", "purpose": "", "mandatory": false, "decrees": {"on": "auto"}, "decree": null},
  {"key": "place", "purpose": "where", "mandatory": false,
   "decrees": {"idle": "home", "off": "home", "on": "home"}, "decree": null},
  {"key": "note", "purpose": "", "mandatory": false, "decrees": {}, "decree": null}]`)
	for _, opts := range []Options{{}, {ForgoAuspex: true}} {
		if got := canon(t, fx.augur(t, r, opts)); got != want {
			t.Errorf("%+v: canon\n got  %s\n want %s", opts, got, want)
		}
	}
}

// TestAugur_TheCanonOfARiteWithoutInscriptionsIsEmpty: a rite that
// declares no inscription bears "canon": [], never null.
func TestAugur_TheCanonOfARiteWithoutInscriptionsIsEmpty(t *testing.T) {
	fx := newFixture(t)
	r := fx.scripture(t, "bare", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [`+chant+`]}`)
	if got := canon(t, fx.augur(t, r, Options{})); got != "[]" {
		t.Errorf("canon %q, want []", got)
	}
}

// TestAugur_AnEdictsCanonBearsItsPlainDecree: an edict's canon holds no
// decrees per aspect ({}) and its decree as the one plain word, or null
// when it decrees nothing.
func TestAugur_AnEdictsCanonBearsItsPlainDecree(t *testing.T) {
	fx := newFixture(t)
	r := fx.scripture(t, "capture", `{
  "pattern": "Mark II",
  "edict": true,
  "inscriptions": {
    "region": {"purpose": "the slurp geometry", "decrees": "full"},
    "name": {"mandatory": true},
  },
  "liturgy": [{"designation": "incantation", "words": "true"}]
}`)
	want := compact(t, `[
  {"key": "region", "purpose": "the slurp geometry", "mandatory": false, "decrees": {}, "decree": "full"},
  {"key": "name", "purpose": "", "mandatory": true, "decrees": {}, "decree": null}]`)
	if got := canon(t, fx.augur(t, r, Options{})); got != want {
		t.Errorf("canon\n got  %s\n want %s", got, want)
	}
}

// TestHeretic_TheCanonOfAHereticalRiteIsEmpty: a heretical rite's
// scripture is not kept, so its augury bears "canon": [].
func TestHeretic_TheCanonOfAHereticalRiteIsEmpty(t *testing.T) {
	fx := newFixture(t)
	r := fx.edict(t, `{"designation": "incantation", "words": "true"}`)
	fx.proclaim(t, r, map[string]string{"region": "window"})
	a, err := Heretic(r.Name, Options{Env: fx.env})
	if err != nil {
		t.Fatal(err)
	}
	if got := canon(t, a); got != "[]" {
		t.Errorf("canon %q, want []", got)
	}
}
