package placeholder

import (
	"strings"
	"testing"
)

// TestRender_OmenIsSpokenWhereAnOmenIsSet: {{omen}} names the omen being
// enacted or tried, in the words that enact or try it and nowhere else.
func TestRender_OmenIsSpokenWhereAnOmenIsSet(t *testing.T) {
	omens := Scope{Mode: RiteMode, Omen: true}
	got, err := omens.Render("echo {{omen}} > /sys/x {{{{omen}}", Values{Omen: "off"})
	if err != nil || got != "echo off > /sys/x {{omen}}" {
		t.Fatalf("Render = %q, %v", got, err)
	}
	for _, sc := range []Scope{{Mode: RiteMode}, {Mode: RiteMode, Edict: true}, {Mode: RiteMode, Exalted: true}, {Mode: VoxCastMode}} {
		hs := sc.Check("echo {{omen}}")
		if len(hs) != 1 || hs[0].Kind != Omenless || hs[0].Pos.Offset != 5 {
			t.Fatalf("scope %+v: %v", sc, hs)
		}
		msg := hs[0].Message()
		for _, want := range []string{"{{omen}}", "enact", "trials"} {
			if !strings.Contains(msg, want) {
				t.Errorf("the denunciation lacks %q: %s", want, msg)
			}
		}
	}
	exaltedOmens := Scope{Mode: RiteMode, Exalted: true, Omen: true}
	if hs := exaltedOmens.Check("echo {{omen}}"); len(hs) != 0 {
		t.Errorf("exalted enact words may splice the omen, fixed at sanction: %v", hs)
	}
}
