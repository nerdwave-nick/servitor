package placeholder

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
)

var (
	rite = Scope{Mode: RiteMode, Inscriptions: []string{"reason", "mode"}}
	vox  = Scope{Mode: VoxCastMode, Inscriptions: []string{"reason", "mode"}}
	vals = Values{
		Aspect: "porpl", Former: "default", Rite: "theme",
		Inscriptions: map[string]string{"reason": "gaming remnant"},
		Tidings:      Tidings{Heresy: "the vessel is sealed", Step: 2, Steps: 5, StepKind: "tether", StepTarget: "~/current-theme"},
	}
)

func TestRender_RiteMode(t *testing.T) {
	cases := []struct {
		name, text string
		v          Values
		want       string
	}{
		{"empty", "", vals, ""},
		{"no marks", "niri msg action do-screen-transition", vals, "niri msg action do-screen-transition"},
		{"aspect", "{{aspect}}", vals, "porpl"},
		{"aspect inside a path", "~/.config/nfluff/themes/{{aspect}}/hooks", vals, "~/.config/nfluff/themes/porpl/hooks"},
		{"former", "{{former}}", vals, "default"},
		{"former unknown is empty", "[{{former}}]", Values{Aspect: "on"}, "[]"},
		{"rite name", "{{rite.name}}", vals, "theme"},
		{"inscription set", "{{inscription.reason}}", vals, "gaming remnant"},
		{"declared inscription unset is empty", "<{{inscription.mode}}>", vals, "<>"},
		{"inscriptions nil map", "<{{inscription.reason}}>", Values{}, "<>"},
		{"several marks", "{{rite.name}}: {{former}} -> {{aspect}}", vals, "theme: default -> porpl"},
		{"adjacent marks", "{{aspect}}{{aspect}}", vals, "porplporpl"},
		{"escaped mark", "{{{{aspect}}", vals, "{{aspect}}"},
		{"escape alone", "{{{{", vals, "{{"},
		{"two escapes", "{{{{{{{{", vals, "{{{{"},
		{"escape then lone brace", "{{{{{", vals, "{{{"},
		{"lone closing", "a }} b", vals, "a }} b"},
		{"closing after mark", "{{aspect}}}}", vals, "porpl}}"},
		{"single braces", "{ {aspect} }", vals, "{ {aspect} }"},
		{"multibyte around mark", "→ {{aspect}} ←", vals, "→ porpl ←"},
		{"multiline", "a\n{{aspect}}\nb", vals, "a\nporpl\nb"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := rite.Render(c.text, c.v)
			if err != nil {
				t.Fatalf("Render(%q): %v", c.text, err)
			}
			if got != c.want {
				t.Fatalf("Render(%q) = %q, want %q", c.text, got, c.want)
			}
		})
	}
}

func TestRender_VoxCastMode(t *testing.T) {
	cases := []struct{ text, want string }{
		{"{{heresy}}", "the vessel is sealed"},
		{"step {{step}} / {{steps}}", "step 2 / 5"},
		{"{{step.kind}} {{step.target}}", "tether ~/current-theme"},
		{"{{rite.name}} took the aspect {{aspect}}", "theme took the aspect porpl"},
		{"{{inscription.reason}}", "gaming remnant"},
		{"{{{{heresy}}", "{{heresy}}"},
	}
	for _, c := range cases {
		t.Run(c.text, func(t *testing.T) {
			got, err := vox.Render(c.text, vals)
			if err != nil {
				t.Fatalf("Render(%q): %v", c.text, err)
			}
			if got != c.want {
				t.Fatalf("Render(%q) = %q, want %q", c.text, got, c.want)
			}
		})
	}
}

func TestCheck_Heresies(t *testing.T) {
	cases := []struct {
		name  string
		scope Scope
		text  string
		want  Heresies
	}{
		{"clean", rite, "{{aspect}} {{former}} {{rite.name}} {{inscription.reason}} {{{{x}} }}", nil},
		{"clean vox", vox, "{{heresy}} {{step}} {{steps}} {{step.kind}} {{step.target}}", nil},
		{"unknown", rite, "{{stat}}", Heresies{
			{Kind: Unknown, Pos: Position{0, 1, 1}, Mark: "{{stat}}", Name: "stat", Mode: RiteMode}}},
		{"unknown after text", rite, "x {{rite}}", Heresies{
			{Kind: Unknown, Pos: Position{2, 1, 3}, Mark: "{{rite}}", Name: "rite", Mode: RiteMode}}},
		{"heresy outside vox-cast", rite, "{{heresy}}", Heresies{
			{Kind: Unknown, Pos: Position{0, 1, 1}, Mark: "{{heresy}}", Name: "heresy", Mode: RiteMode}}},
		{"step kind outside vox-cast", rite, "{{step.kind}}", Heresies{
			{Kind: Unknown, Pos: Position{0, 1, 1}, Mark: "{{step.kind}}", Name: "step.kind", Mode: RiteMode}}},
		{"unknown in vox-cast", vox, "{{step.goal}}", Heresies{
			{Kind: Unknown, Pos: Position{0, 1, 1}, Mark: "{{step.goal}}", Name: "step.goal", Mode: VoxCastMode}}},
		{"spaces are not forgiven", rite, "{{ aspect }}", Heresies{
			{Kind: Unknown, Pos: Position{0, 1, 1}, Mark: "{{ aspect }}", Name: " aspect ", Mode: RiteMode}}},
		{"empty mark", rite, "{{}}", Heresies{
			{Kind: Unknown, Pos: Position{0, 1, 1}, Mark: "{{}}", Name: "", Mode: RiteMode}}},
		{"inscription without key", rite, "{{inscription.}}", Heresies{
			{Kind: Unknown, Pos: Position{0, 1, 1}, Mark: "{{inscription.}}", Name: "inscription.", Mode: RiteMode}}},
		{"undeclared inscription on second line", rite, "a\nbc {{inscription.reasn}}", Heresies{
			{Kind: Undeclared, Pos: Position{5, 2, 4}, Mark: "{{inscription.reasn}}", Name: "inscription.reasn", Mode: RiteMode}}},
		{"no inscriptions declared", Scope{}, "{{inscription.reason}}", Heresies{
			{Kind: Undeclared, Pos: Position{0, 1, 1}, Mark: "{{inscription.reason}}", Name: "inscription.reason", Mode: RiteMode}}},
		{"unclosed", rite, "ok {{aspect", Heresies{
			{Kind: Unclosed, Pos: Position{3, 1, 4}, Mark: "{{", Name: "", Mode: RiteMode}}},
		{"three braces", rite, "{{{", Heresies{
			{Kind: Unclosed, Pos: Position{0, 1, 1}, Mark: "{{", Name: "", Mode: RiteMode}}},
		{"byte column after multibyte", rite, "→{{x}}", Heresies{
			{Kind: Unknown, Pos: Position{3, 1, 4}, Mark: "{{x}}", Name: "x", Mode: RiteMode}}},
		{"several in order", rite, "{{a}} {{aspect}}\n{{b}} {{c", Heresies{
			{Kind: Unknown, Pos: Position{0, 1, 1}, Mark: "{{a}}", Name: "a", Mode: RiteMode},
			{Kind: Unknown, Pos: Position{17, 2, 1}, Mark: "{{b}}", Name: "b", Mode: RiteMode},
			{Kind: Unclosed, Pos: Position{23, 2, 7}, Mark: "{{", Name: "", Mode: RiteMode}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.scope.Check(c.text)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("Check(%q)\n got  %+v\n want %+v", c.text, got, c.want)
			}
		})
	}
}

func TestRender_HeresyIsNeverRenderedEmpty(t *testing.T) {
	for _, text := range []string{"{{stat}}", "a {{inscription.reasn}} b", "{{heresy}}", "x {{aspect"} {
		t.Run(text, func(t *testing.T) {
			got, err := rite.Render(text, vals)
			if err == nil {
				t.Fatalf("Render(%q) = %q, want heresy", text, got)
			}
			if got != "" {
				t.Fatalf("Render(%q) rendered %q despite heresy", text, got)
			}
			var hs Heresies
			if !errors.As(err, &hs) || !reflect.DeepEqual(hs, rite.Check(text)) {
				t.Fatalf("Render(%q) error %v is not the heresies of Check %v", text, err, rite.Check(text))
			}
		})
	}
}

// profane words that must never reach the faithful.
var profane = []string{"error", "invalid", "variable", "template", "state", "undefined", "metadata", "placeholder name"}

func TestHeresy_Denunciations(t *testing.T) {
	cases := []struct {
		name  string
		scope Scope
		text  string
		must  []string
	}{
		{"unknown", rite, "{{stat}}", []string{"{{stat}}", "{{aspect}}", "{{former}}", "{{rite.name}}", "{{inscription.<key>}}"}},
		{"unknown in vox-cast names its marks", vox, "{{stat}}", []string{"{{stat}}", "{{heresy}}", "{{step}}", "{{steps}}", "{{step.kind}}", "{{step.target}}"}},
		{"reserved for vox-casts", rite, "{{steps}}", []string{"{{steps}}", "vox-cast"}},
		{"undeclared", rite, "{{inscription.reasn}}", []string{"{{inscription.reasn}}", `"reasn"`, "inscriptions"}},
		{"unclosed", rite, "{{aspect", []string{"{{", "}}", "{{{{"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			hs := c.scope.Check(c.text)
			if len(hs) != 1 {
				t.Fatalf("Check(%q) = %v, want one heresy", c.text, hs)
			}
			msg := hs[0].Message()
			for _, m := range c.must {
				if !strings.Contains(msg, m) {
					t.Errorf("denunciation %q lacks %q", msg, m)
				}
			}
			for _, p := range profane {
				if strings.Contains(strings.ToLower(msg), p) {
					t.Errorf("denunciation %q speaks the profane %q", msg, p)
				}
			}
			if want := "1:1: " + msg; hs[0].Error() != want {
				t.Errorf("Error() = %q, want %q", hs[0].Error(), want)
			}
		})
	}
}

func TestHeresies_ErrorJoinsEveryDenunciation(t *testing.T) {
	hs := rite.Check("{{a}}\n{{b}}")
	want := hs[0].Error() + "\n" + hs[1].Error()
	if hs.Error() != want {
		t.Fatalf("Error() = %q, want %q", hs.Error(), want)
	}
	if !strings.HasPrefix(hs[1].Error(), "2:1: ") {
		t.Fatalf("second heresy %q not placed at 2:1", hs[1].Error())
	}
}

// braced builds words from fragments rich in braces, so that properties are
// tested where the heresies dwell rather than in random runes alone.
func braced(picks []uint8, tail string) string {
	frags := []string{"{", "}", "{{", "}}", "a", "é", "\n", " ", "aspect", "inscription.reason", "{{aspect}}", "{{x}}"}
	var b strings.Builder
	for _, p := range picks {
		b.WriteString(frags[int(p)%len(frags)])
	}
	return b.String() + tail
}

func TestRender_WithoutOpeningIsIdentityProperty(t *testing.T) {
	prop := func(picks []uint8, tail string, v Values, voxMode bool) bool {
		text := braced(picks, tail)
		for strings.Contains(text, "{{") {
			text = strings.ReplaceAll(text, "{{", "{")
		}
		s := rite
		if voxMode {
			s = vox
		}
		got, err := s.Render(text, v)
		return err == nil && got == text
	}
	if err := quick.Check(prop, &quick.Config{MaxCount: 500}); err != nil {
		t.Fatal(err)
	}
}

func TestRender_EscapedWordsReturnUnchangedProperty(t *testing.T) {
	prop := func(picks []uint8, tail string) bool {
		text := braced(picks, tail)
		got, err := rite.Render(strings.ReplaceAll(text, "{{", "{{{{"), vals)
		return err == nil && got == text
	}
	if err := quick.Check(prop, &quick.Config{MaxCount: 500}); err != nil {
		t.Fatal(err)
	}
}

func TestCheck_AgreesWithRenderProperty(t *testing.T) {
	prop := func(picks []uint8, tail string) bool {
		text := braced(picks, tail)
		hs := rite.Check(text)
		_, err := rite.Render(text, vals)
		if len(hs) == 0 {
			return err == nil
		}
		var got Heresies
		return errors.As(err, &got) && reflect.DeepEqual(got, hs)
	}
	if err := quick.Check(prop, &quick.Config{MaxCount: 500}); err != nil {
		t.Fatal(err)
	}
}
