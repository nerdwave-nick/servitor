package librarium

import (
	"slices"
	"testing"
)

// TestRite_ExaltedSubjectsAndObservancesExaltTheRite: a state rite of
// pattern Mark II is exalted by its exalted subjects and observances, each
// named with the place of its "exalted"; it holds no exalted verse.
func TestRite_ExaltedSubjectsAndObservancesExaltTheRite(t *testing.T) {
	r, fs := Parse("dnd", "/lib/rites/dnd.json", []byte(markII(``,
		`{"name": "mako", "designation": "machine-spirit", "enact": {"*": "true"}, "recessional": ["herald"]},
		 {"name": "conf", "designation": "transcription", "vessel": "/etc/dnd.conf", "omens": {"*": "x"},
		  "exalted": true}`,
		`{"name": "quiet", "designation": "incantation", "words": "true", "to": ["on"]},
		 {"name": "herald", "designation": "incantation", "words": "true", "exalted": true}`)))
	if r == nil {
		t.Fatalf("heretical: %v", fs)
	}
	var got []string
	for _, item := range r.ExaltedItems() {
		got = append(got, item.String()+" "+item.Pointer)
	}
	want := []string{`the subject "conf" /subjects/1/exalted`, `the observance "herald" /observances/1/exalted`}
	if !slices.Equal(got, want) || !r.Exalted() || r.ExaltedVerses() != nil {
		t.Fatalf("exalted items %q, want %q; exalted %v, verses %v", got, want, r.Exalted(), r.ExaltedVerses())
	}
	common, _ := Parse("dnd", "/lib/rites/dnd.json", []byte(dnd))
	if common.Exalted() || common.ExaltedItems() != nil {
		t.Fatal("a state rite without exalted subjects or observances is told exalted")
	}
}

// TestRite_ExaltedVersesAreItemsToo: the exalted verses of a liturgy are
// its exalted items, named by their verse.
func TestRite_ExaltedVersesAreItemsToo(t *testing.T) {
	r, _ := Parse("split-lock", "/lib/rites/split-lock.json", []byte(splitLock))
	var got []string
	for _, item := range r.ExaltedItems() {
		got = append(got, item.String()+" "+item.Pointer)
	}
	want := []string{"verse 1 /liturgy/0/exalted", "verse 3 /liturgy/2/exalted", "verse 4 /liturgy/3/exalted",
		"verse 5 /liturgy/4/exalted"}
	if !slices.Equal(got, want) {
		t.Fatalf("exalted items %q, want %q", got, want)
	}
}

// TestRite_ExaltedNounNamesWhatIsExalted: laments name what of a rite is
// exalted as it holds it: its exalted subjects, its exalted observances, or
// both; an edict's exalted verses.
func TestRite_ExaltedNounNamesWhatIsExalted(t *testing.T) {
	spirit := func(exalted string) string {
		return `{"name": "s", "designation": "machine-spirit", "enact": {"*": "true"}, "introit": ["o"]` + exalted + `}`
	}
	observance := func(exalted string) string {
		return `{"name": "o", "designation": "incantation", "words": "true"` + exalted + `}`
	}
	for _, c := range []struct{ scripture, want string }{
		{markII(``, spirit(`, "exalted": true`), observance(``)), "exalted subjects"},
		{markII(``, spirit(``), observance(`, "exalted": true`)), "exalted observances"},
		{markII(``, spirit(`, "exalted": true`), observance(`, "exalted": true`)), "exalted subjects and observances"},
		{`{"pattern": "Mark II", "edict": true, "liturgy": [{"designation": "incantation", "words": "true",
		  "exalted": true}]}`, "exalted verses"},
	} {
		r, fs := Parse("r", "/lib/rites/r.json", []byte(c.scripture))
		if r == nil || fs.Heretical() {
			t.Fatalf("heretical: %v", fs)
		}
		if got := r.ExaltedNoun(); got != c.want {
			t.Errorf("%s: %q, want %q", c.scripture, got, c.want)
		}
	}
}
