package block

import (
	"maps"
	"strings"
	"testing"
	"testing/quick"
)

var mouse = Marker{Comment: "//", Guard: "mouse"}

func TestEncodeMeta_StateFirstAndQuoting(t *testing.T) {
	got := EncodeMeta(map[string]string{"reason": "gaming remnant", "state": "on", "a": "x", "empty": ""})
	want := `state|on a|x reason|"gaming remnant"`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDecodeMeta_RoundTrip(t *testing.T) {
	in := map[string]string{"state": "on", "reason": `say "hi" | there\`, "k.v-x": "plain"}
	out, err := DecodeMeta(EncodeMeta(in))
	if err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(in, out) {
		t.Fatalf("got %v want %v", out, in)
	}
}

func TestDecodeMeta_RoundTripProperty(t *testing.T) {
	prop := func(v1, v2 string) bool {
		in := map[string]string{"state": "s", "a": v1, "b": v2}
		out, err := DecodeMeta(EncodeMeta(in))
		if err != nil {
			return false
		}
		for k, v := range in {
			if v != "" && out[k] != v {
				return false
			}
			if _, ok := out[k]; v == "" && ok {
				return false
			}
		}
		return true
	}
	if err := quick.Check(prop, &quick.Config{MaxCount: 200}); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeMeta_Malformed(t *testing.T) {
	for _, in := range []string{"novalue", `k|"unterminated`, `k|"a"b`, "k|a k|b", "|x"} {
		if _, err := DecodeMeta(in); err == nil {
			t.Errorf("DecodeMeta(%q) should fail", in)
		}
	}
}

func TestRender_WithCommentEnd(t *testing.T) {
	m := Marker{Comment: "/*", CommentEnd: "*/", Guard: "g"}
	got, err := Render(m, "  ", map[string]string{"state": "on"}, "a {}\n")
	if err != nil {
		t.Fatal(err)
	}
	want := "  /* begin servitor managed -- g -- state|on */\na {}\n  /* end servitor managed -- g */"
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	if _, err := Render(m, "", map[string]string{"x": "a*/b"}, ""); err == nil {
		t.Fatal("expected error for metadata containing comment terminator")
	}
}

func TestUpsert_AppendsWhenMissing(t *testing.T) {
	got, err := Upsert("input {}", mouse, map[string]string{"state": "on"}, "cursor {}")
	if err != nil {
		t.Fatal(err)
	}
	want := "input {}\n\n// begin servitor managed -- mouse -- state|on\ncursor {}\n// end servitor managed -- mouse\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestUpsert_AppendsToEmptyFile(t *testing.T) {
	got, _ := Upsert("", mouse, nil, "x")
	if got != "// begin servitor managed -- mouse\nx\n// end servitor managed -- mouse\n" {
		t.Fatalf("got %q", got)
	}
}

func TestUpsert_ReplacesInPlaceKeepingSurroundings(t *testing.T) {
	in := "a\n    // begin servitor managed -- mouse -- state|off\nold\nold2\n    // end servitor managed -- mouse\nb\n"
	got, err := Upsert(in, mouse, map[string]string{"state": "on", "reason": "x y"}, "new")
	if err != nil {
		t.Fatal(err)
	}
	want := "a\n    // begin servitor managed -- mouse -- state|on reason|\"x y\"\nnew\n    // end servitor managed -- mouse\nb\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	again, _ := Upsert(got, mouse, map[string]string{"state": "on", "reason": "x y"}, "new")
	if again != got {
		t.Fatal("upsert is not idempotent")
	}
}

func TestFind_ParsesMetaAndContent(t *testing.T) {
	in := "// begin servitor managed -- mouse -- state|on reason|\"gaming remnant\"\nl1\nl2\n// end servitor managed -- mouse"
	b, found, err := Find(in, mouse)
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	if b.Meta["state"] != "on" || b.Meta["reason"] != "gaming remnant" || b.Content != "l1\nl2" {
		t.Fatalf("unexpected block %+v", b)
	}
}

func TestFind_IgnoresOtherGuardsWithSharedPrefix(t *testing.T) {
	in := "// begin servitor managed -- mouse-extra -- state|on\nx\n// end servitor managed -- mouse-extra"
	if _, found, err := Find(in, mouse); found || err != nil {
		t.Fatalf("found=%v err=%v", found, err)
	}
}

func TestFind_Malformed(t *testing.T) {
	begin := "// begin servitor managed -- mouse"
	end := "// end servitor managed -- mouse"
	cases := map[string]string{
		"unterminated": begin + "\nx",
		"stray end":    "x\n" + end,
		"duplicate":    strings.Join([]string{begin, end, begin, end}, "\n"),
		"nested":       strings.Join([]string{begin, begin, end}, "\n"),
		"bad meta":     begin + " -- broken\n" + end,
	}
	for name, in := range cases {
		if _, _, err := Find(in, mouse); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestIsMarkerLine(t *testing.T) {
	if !mouse.IsMarkerLine("  // end servitor managed -- mouse") || mouse.IsMarkerLine("// end servitor managed -- other") {
		t.Fatal("IsMarkerLine mismatch")
	}
}

func TestRemove_UndoesAppend(t *testing.T) {
	for _, orig := range []string{"input {}\n", "", "a\nb"} {
		added, err := Upsert(orig, mouse, map[string]string{"state": "on"}, "x")
		if err != nil {
			t.Fatal(err)
		}
		got, err := Remove(added, mouse)
		if err != nil {
			t.Fatal(err)
		}
		want := orig
		if want != "" && !strings.HasSuffix(want, "\n") {
			want += "\n"
		}
		if got != want {
			t.Errorf("Remove(Upsert(%q)) = %q, want %q", orig, got, want)
		}
	}
}

func TestRemove_InTheMiddleKeepsNeighbours(t *testing.T) {
	in := "a\n\n// begin servitor managed -- mouse\nx\n// end servitor managed -- mouse\nb\n"
	got, err := Remove(in, mouse)
	if err != nil || got != "a\n\nb\n" {
		t.Fatalf("got %q err %v", got, err)
	}
	if same, _ := Remove("plain", mouse); same != "plain" {
		t.Fatal("text without block must be unchanged")
	}
	if _, err := Remove("// begin servitor managed -- mouse\n", mouse); err == nil {
		t.Fatal("malformed block must error")
	}
}
