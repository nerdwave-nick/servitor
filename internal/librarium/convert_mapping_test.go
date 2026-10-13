package librarium

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

// converted converts the scripture of pattern Mark I and fails unless it
// becomes a pure rite of pattern Mark II.
func converted(t *testing.T, scripture string) (*Rite, *Conversion) {
	t.Helper()
	r, fs, c := parseConverting("rite", "/librarium/rites/rite.json", []byte(scripture))
	if c == nil || fs.Heretical() || r.markI {
		t.Fatalf("not converted to a pure rite of Mark II: %v", fs)
	}
	return r, c
}

// TestConvert_SpeechBesideSubjects: speech in a rite of sanctums,
// transcriptions or tethers becomes observances, each the introit of the
// next subject after it, else the recessional of the last before it, in
// written order; an aspect map splits into one observance per distinct
// words, guarded by "to"; vox-casts become the rite's list.
func TestConvert_SpeechBesideSubjects(t *testing.T) {
	r, c := converted(t, `{"pattern": "Mark I", "aspects": ["a", "b", "c"], "liturgy": [
	  {"incantation": "echo first"},
	  {"vox-cast": "success"},
	  {"incantation": {"a": "echo alpha", "*": "echo other"}, "reversion": "echo undone", "patience": "3s"},
	  {"tether": "/tmp/link", "anchor": {"*": "/tmp/{{aspect}}"}, "zeal": true},
	  {"litany": "scroll.sh", "offerings": ["{{aspect}}"]},
	  {"transcription": "/tmp/vessel", "scripture": {"a": "x", "*": null}},
	  {"incantation": "echo last"},
	  {"vox-cast": "progress"}]}`)
	var names, introits, recessionals []string
	for _, s := range r.Subjects {
		names = append(names, s.Name+":"+s.Designation())
		introits = append(introits, strings.Join(s.Introit, ","))
		recessionals = append(recessionals, strings.Join(s.Recessional, ","))
	}
	if want := []string{"tether-4:tether", "transcription-6:transcription"}; !slices.Equal(names, want) {
		t.Errorf("subjects %v, want %v", names, want)
	}
	if want := []string{"incantation-1,incantation-3-a,incantation-3-b", "litany-5"}; !slices.Equal(introits, want) {
		t.Errorf("introits %q, want %q", introits, want)
	}
	if want := []string{"", "incantation-7"}; !slices.Equal(recessionals, want) {
		t.Errorf("recessionals %q, want %q", recessionals, want)
	}
	split := r.Observances[1:3]
	if split[0].Name != "incantation-3-a" || !slices.Equal(split[0].To, []string{"a"}) ||
		split[1].Name != "incantation-3-b" || !slices.Equal(split[1].To, []string{"b", "c"}) {
		t.Errorf("the aspect map is not split by its words: %+v", split)
	}
	words := split[1].Speech.(*Incantation)
	if w, _ := words.Command.For("b"); w != "echo other" || words.Patience != 3*time.Second {
		t.Errorf("the split observance speaks %q (patience %s)", w, words.Patience)
	}
	if rev, _ := words.Reversion.For("c"); rev != "echo undone" {
		t.Errorf("the observance's reversion is %q", rev)
	}
	if r.Observances[0].To != nil {
		t.Errorf("words alike for every aspect are guarded: %v", r.Observances[0].To)
	}
	if !slices.Equal(r.VoxCasts, []string{VoxProgress, VoxSuccess}) {
		t.Errorf("vox-casts %v", r.VoxCasts)
	}
	if len(c.Changes) != 0 || c.Exalted {
		t.Errorf("nothing changed in meaning, yet the conversion tells %v (exalted %v)", c.Changes, c.Exalted)
	}
}

// TestConvert_SpeechAlone: in a rite without sanctums, transcriptions or
// tethers each verse of speech becomes a machine-spirit without omens,
// enacting its words per aspect; its reversion is struck and told; the
// rite's auspex becomes the scan of the only machine-spirit.
func TestConvert_SpeechAlone(t *testing.T) {
	r, c := converted(t, `{"pattern": "Mark I", "aspects": ["on", "off"], "tongue": "zsh",
	  "auspex": {"rite": "cat /tmp/state", "patience": "300ms"},
	  "liturgy": [{"incantation": {"on": "echo 1 > /tmp/state", "*": "echo 0 > /tmp/state"}, "tongue": "sh",
	    "patience": "4s", "reversion": "echo undone", "exalted": true}]}`)
	m := r.Subjects[0].Matter.(*MachineSpirit)
	if r.Subjects[0].Name != "incantation-1" || !m.Omens.IsZero() || !m.Exalted || m.Tongue != "sh" ||
		m.Patience != 4*time.Second {
		t.Errorf("the machine-spirit is %q %+v", r.Subjects[0].Name, m)
	}
	if on, _ := m.Enact.For("on"); on != "echo 1 > /tmp/state" {
		t.Errorf("enacts %q for on", on)
	}
	if !reflect.DeepEqual(m.Auspex, &SpiritAuspex{Scan: "cat /tmp/state", Patience: 300 * time.Millisecond}) {
		t.Errorf("the auspex of the rite is not the scan of its machine-spirit: %+v", m.Auspex)
	}
	if !slices.Equal(c.Changes, []string{"the reversion of verse 1 is struck"}) || !c.Exalted {
		t.Errorf("changes %v, exalted %v", c.Changes, c.Exalted)
	}
	if !strings.Contains(string(c.Data), `// pattern "Mark I" spoke the reversion "echo undone"`) {
		t.Errorf("the struck reversion is not kept as a comment:\n%s", c.Data)
	}
	if !strings.Contains(c.Tidings(), "servitor sanction rite") {
		t.Errorf("the tidings of an exalted rite name no sanction: %s", c.Tidings())
	}
}

// TestConvert_LitanyAlone: a litany in a rite without sanctums,
// transcriptions or tethers becomes a machine-spirit whose enact words call
// its scroll with its offerings, which is told.
func TestConvert_LitanyAlone(t *testing.T) {
	r, c := converted(t, `{"pattern": "Mark I", "aspects": ["on", "off"], "liturgy": [
	  {"litany": "hooks.sh", "offerings": ["~/themes/{{aspect}}", "it's"]},
	  {"litany": {"on": "/bin/up", "*": "/bin/down"}}]}`)
	first, _ := r.Subjects[0].Matter.(*MachineSpirit).Enact.For("off")
	if first != `./hooks.sh ~/themes/{{aspect}} 'it'\''s'` {
		t.Errorf("the litany is enacted as %q", first)
	}
	second := r.Subjects[1].Matter.(*MachineSpirit).Enact
	if on, _ := second.For("on"); on != "/bin/up" {
		t.Errorf("the litany per aspect is enacted as %q", on)
	}
	if want := []string{"the litany of verse 1 is spoken as enact words", "the litany of verse 2 is spoken as enact words"}; !slices.Equal(c.Changes, want) {
		t.Errorf("changes %v", c.Changes)
	}
}

// TestConvert_KeepsMarkersAndDecrees: subjects keep their wards, so no
// marker changes and no vessel is rewritten; a plain decree becomes an
// aspect map; an edict's decree stays one word.
func TestConvert_KeepsMarkersAndDecrees(t *testing.T) {
	r, _ := converted(t, `{"pattern": "Mark I", "aspects": ["on", "off"],
	  "inscriptions": {"mode": {"decrees": "plain"}},
	  "liturgy": [{"sanctum": "/tmp/v.kdl", "scripture": "x"}]}`)
	if s := r.Subjects[0].Matter.(*Sanctum); s.WardFor(r.Name) != "rite" || s.Ward != "" {
		t.Errorf("the sanctum keeps another ward: %+v", s)
	}
	if e := r.Inscriptions[0].Decrees.Entries(); len(e) != 1 || e[0].Aspect != Fallback || e[0].Value != "plain" {
		t.Errorf("the decree is %+v", e)
	}
	_, c := converted(t, `{"pattern": "Mark I", "edict": true, "inscriptions": {"mode": {"decrees": "plain"}},
	  "liturgy": [{"incantation": "echo"}, {"vox-cast": "success"}]}`)
	if !strings.Contains(string(c.Data), `"decrees": "plain"`) || !strings.Contains(string(c.Data), `"designation": "incantation"`) {
		t.Errorf("the edict is converted as:\n%s", c.Data)
	}
}

// TestConvert_Refused: heretical scripture of Mark I, and scripture that
// would be heretical once converted, is not converted, and a heresy says
// it is converted once purged; scripture of Mark II is read as written.
func TestConvert_Refused(t *testing.T) {
	for name, scripture := range map[string]string{
		"heretical":       `{"pattern": "Mark I", "aspects": ["on"], "liturgy": [{"incantation": 3}]}`,
		"heretical once":  `{"pattern": "Mark I", "aspects": ["on"], "liturgy": [{"vox-cast": "success"}]}`,
		"pattern Mark II": `{"pattern": "Mark II", "edict": true, "liturgy": [{"designation": "incantation", "words": "x"}]}`,
	} {
		r, fs, c := parseConverting("rite", "/librarium/rites/rite.json", []byte(scripture))
		if c != nil {
			t.Errorf("%s: converted", name)
		}
		if name == "pattern Mark II" {
			if fs.Heretical() || r.markI {
				t.Errorf("%s: not read as written: %v", name, fs)
			}
			continue
		}
		if r != nil || !strings.Contains(fmt.Sprint(fs), "is converted once") {
			t.Errorf("%s: the heresy does not tell the conversion awaited: %v", name, fs)
		}
	}
}
