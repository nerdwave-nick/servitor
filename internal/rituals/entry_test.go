package rituals

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

// TestInvoke_TheResultBearsTheChronicledEntry: a performance hands back the
// very entry it chronicled, whatever its verdict; an invocation refused or
// foreseen chronicles nothing and bears none.
func TestInvoke_TheResultBearsTheChronicledEntry(t *testing.T) {
	fx := newFixture(t)
	fx.vessel("util.kdl", "input {}\n")
	fx.rite("mouse", mouse)
	fx.rite("falling", strings.Replace(mouse, `{"vox-cast": "success"}`, `{"incantation": "exit 3"}`, 1))
	s := fx.servitor()

	for i, p := range []Petition{
		{Rite: "mouse", Aspect: "on", Runes: map[string]string{"reason": "gaming remnant"}},
		{Rite: "falling", Aspect: "off"},
	} {
		res, err := s.Invoke(context.Background(), p)
		if err != nil {
			t.Fatal(err)
		}
		entries := fx.chronicled()
		if len(entries) != i+1 || res.Entry == nil || !reflect.DeepEqual(*res.Entry, entries[0]) {
			t.Fatalf("%s: the result bears %+v, the chronicle %+v", p.Rite, res.Entry, entries)
		}
	}
	for _, p := range []Petition{
		{Rite: "mouse", Aspect: "maybe"},
		{Rite: "mouse", Aspect: "off", Foresee: true},
	} {
		if res, _ := s.Invoke(context.Background(), p); res.Entry != nil {
			t.Fatalf("%+v bears the entry %+v", p, res.Entry)
		}
	}
}
