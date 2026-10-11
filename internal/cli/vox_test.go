package cli

import (
	"strings"
	"testing"
)

// TestVox_DecidesWhetherTheDesktopHears: "auto" (the default) sends to the
// desktop only without a terminal, "notify-send" always, "off" never; the
// report upon a terminal is told whichever vox is written.
func TestVox_DecidesWhetherTheDesktopHears(t *testing.T) {
	for _, tc := range []struct {
		vox      string // "" writes no settings
		terminal bool
		notified bool
	}{
		{"", true, false},
		{"", false, true},
		{"auto", true, false},
		{"auto", false, true},
		{"notify-send", true, true},
		{"notify-send", false, true},
		{"off", true, false},
		{"off", false, false},
	} {
		name := tc.vox
		if name == "" {
			name = "unwritten"
		}
		t.Run(name, func(t *testing.T) {
			e := newEnv(t)
			if tc.vox != "" {
				e.addRite("servitor.json", `{"pattern": "Mark I", "vox": "`+tc.vox+`"}`)
			}
			if tc.terminal {
				e.terminal(0)
			}
			out := e.mustRun("invoke", "mouse-autohide-toggle", "on")
			if got := e.notified() != ""; got != tc.notified {
				t.Errorf("terminal=%v: desktop heard %v, want %v (%q)", tc.terminal, got, tc.notified, e.notified())
			}
			if !strings.Contains(out, "+++ mouse-autohide-toggle · (dormant) → on +++") || !strings.Contains(out, "✠ ") {
				t.Errorf("terminal=%v: the report is not told: %q", tc.terminal, out)
			}
		})
	}
}
