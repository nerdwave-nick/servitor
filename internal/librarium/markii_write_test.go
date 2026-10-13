package librarium

import (
	"io/fs"
	"strings"
	"testing"
	"time"
)

func TestMarshal_MarkIIScripturesRoundTrip(t *testing.T) {
	for name, scripture := range map[string]string{"gaming": gaming, "dnd": dnd, "visage": visageII, "screenshot": screenshot} {
		t.Run(name, func(t *testing.T) {
			r, fs := Parse(name, "/lib/rites/"+name+".json", []byte(scripture))
			if len(fs) != 0 {
				t.Fatalf("findings:\n%v", fs)
			}
			data := string(roundTrip(t, r))
			if !strings.Contains(data, `"pattern": "Mark II"`) {
				t.Errorf("scripture lacks its pattern:\n%s", data)
			}
		})
	}
}

func TestMarshal_ForgedMarkIIRitesRoundTrip(t *testing.T) {
	seal := fs.FileMode(0o640)
	spirit := func(name string) Subject {
		return Subject{Name: name, Matter: &MachineSpirit{Auspex: &SpiritAuspex{Scan: "cat x"}, Enact: PerAspect(Entry[string]{"*", "x"})}}
	}
	cases := map[string]*Rite{
		"bare": {Name: "bare", MarkII: true, Aspects: []string{"on"}, Subjects: []Subject{spirit("x")}},
		"every designation": {Name: "x", MarkII: true, Purpose: "p", Aspects: []string{"on", "off"}, Tongue: "fish",
			VoxCasts: []string{VoxSuccess, VoxProgress},
			Inscriptions: []Inscription{{Key: "m", Purpose: "why", Mandatory: true, Decrees: PerAspect(Entry[string]{"on", "a"}, Entry[string]{"*", "b"})},
				{Key: "e", Decrees: PerAspect[string]()}, {Key: "z"}},
			Subjects: []Subject{
				{Name: "s", Purpose: "a sanctum", Introit: []string{"i"}, Matter: &Sanctum{Vessel: "/v.css", Ward: "w", Glyph: "/*",
					ClosingGlyph: "*/", Consecrate: true, Exalted: true,
					Scripture: PerAspect(Entry[Scripture]{"on", Inline("a\nb")}, Entry[Scripture]{"*", Tome("t/{{aspect}}", true)})}},
				{Name: "c", Recessional: []string{"r", "l"}, Matter: &Transcription{Vessel: "/c", Seal: &seal, Zeal: true,
					Scripture: PerAspect(Entry[Scripture]{"on", Scripture{Null: true}}, Entry[Scripture]{"off", Tome("t", false)})}},
				{Name: "t", Introit: []string{}, Matter: &Tether{Name: "/t", Zeal: true,
					Anchor: PerAspect(Entry[Anchor]{"on", Anchor{Path: "/a"}}, Entry[Anchor]{"*", Anchor{Null: true}})}},
				{Name: "m", Matter: &MachineSpirit{Exalted: true, Tongue: "sh", Patience: 3 * time.Second, Relics: []string{"r.conf"},
					Omens: PerAspect(Entry[string]{"on", "1"}, Entry[string]{"*", "0"}),
					Auspex: &SpiritAuspex{Scan: "cat /m", Trials: PerAspect(Entry[string]{"1", "test -e /m1"}), Patience: time.Second, Exalted: true},
					Enact:  PerAspect(Entry[string]{"1", "echo 1 > /m"}, Entry[string]{"*", "echo {{omen}} > /m"})}},
				{Name: "n", Matter: &MachineSpirit{Auspex: &SpiritAuspex{Trials: PerAspect(Entry[string]{"on", "true"}, Entry[string]{"*", "false"})},
					Enact: PerAspect(Entry[string]{"on", "a"}, Entry[string]{"off", "b"})}},
			},
			Observances: []Observance{
				{Name: "i", From: []string{"off"}, Speech: &Incantation{Command: Uniform("a"),
					Utterance: Utterance{Tongue: "sh", Patience: time.Minute, Reversion: Uniform("")}}},
				{Name: "r", To: []string{}, Speech: &Incantation{Command: Uniform("sysctl -w a={{aspect}}"),
					Utterance: Utterance{Exalted: true, Relics: []string{}, Reversion: Uniform("sysctl -w a=$SERVITOR_FORMER_ASPECT")}}},
				{Name: "l", To: []string{"on"}, From: []string{"off"}, Speech: &Litany{Scroll: Uniform("s"),
					Offerings: []AspectMap[string]{Uniform("~/{{aspect}}"), Uniform("")}}},
			}},
		"edict": {Name: "e", MarkII: true, Edict: true, Purpose: "p", VoxCasts: []string{},
			Inscriptions: []Inscription{{Key: "region", Decrees: Uniform("whole")}},
			Liturgy: []Step{
				&Incantation{Command: Uniform("a"), Utterance: Utterance{Patience: 2 * time.Minute, Reversion: Uniform("b")}},
				&Litany{Scroll: Uniform("s"), Offerings: []AspectMap[string]{}, Utterance: Utterance{Tongue: "sh"}},
				&Litany{Scroll: Uniform("s"), Utterance: Utterance{Exalted: true, Relics: []string{"s.list"}}},
			}},
		"empty edict": {Name: "e", MarkII: true, Edict: true},
	}
	for name, r := range cases {
		t.Run(name, func(t *testing.T) {
			data := string(roundTrip(t, r))
			if r.Edict && strings.Contains(data, `"aspects"`) {
				t.Errorf("an edict gained aspects:\n%s", data)
			}
		})
	}
}

// TestMarshal_MarkIIWritesEveryMapAsAnObject: pattern Mark II knows no
// single value in place of a map, and no aspect map in an edict's words.
func TestMarshal_MarkIIWritesEveryMapAsAnObject(t *testing.T) {
	r := &Rite{Name: "x", MarkII: true, Aspects: []string{"on"},
		Inscriptions: []Inscription{{Key: "m", Decrees: Uniform("all")}},
		Subjects: []Subject{{Name: "t", Matter: &Tether{Name: "/t", Anchor: Uniform(Anchor{Path: "/a"})}},
			{Name: "x", Matter: &MachineSpirit{Enact: Uniform("a")}}}}
	data, err := Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"decrees": {"*": "all"}`, `"omens": {"*": "/a"}`, `"enact": {"*": "a"}`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("scripture lacks %s:\n%s", want, data)
		}
	}
	if _, fs := Parse("x", "/x.json", data); fs.Heretical() {
		t.Errorf("written scripture is heretical:\n%v\n%s", fs, data)
	}
}

func TestMarshal_RefusesWhatMarkIICannotWrite(t *testing.T) {
	cases := map[string]*Rite{
		"words keyed by aspect": {Name: "x", MarkII: true, Edict: true, Liturgy: []Step{
			&Incantation{Command: PerAspect(Entry[string]{"on", "a"}, Entry[string]{"*", "b"})}}},
		"a sanctum in an edict": {Name: "x", MarkII: true, Edict: true, Liturgy: []Step{&Sanctum{Vessel: "/v"}}},
		"a subject of no matter": {Name: "x", MarkII: true, Aspects: []string{"on"}, Subjects: []Subject{{Name: "s"}}},
		"a vox-cast step":        {Name: "x", MarkII: true, Edict: true, Liturgy: []Step{&VoxCast{Tidings: VoxSuccess}}},
	}
	for name, r := range cases {
		if data, err := Marshal(r); err == nil {
			t.Errorf("%s: written as\n%s", name, data)
		}
	}
}
