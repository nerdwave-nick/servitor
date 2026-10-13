package cli

import (
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// TestForeseenPlan_APrivilegedAuspexIsNeverSpoken: the foresight of a
// machine-spirit whose privileged auspex was not heard tells the omen the
// data-slate recorded, stale, with the hour it was read; else that it is
// read at invocation.
func TestForeseenPlan_APrivilegedAuspexIsNeverSpoken(t *testing.T) {
	at := time.Date(2026, 10, 13, 20, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		plan invocation.SubjectPlan
		want string
	}{
		{invocation.SubjectPlan{Privileged: true, Target: "gaming", Changing: true},
			`(read at invocation) → "gaming", changing`},
		{invocation.SubjectPlan{Privileged: true, Stale: true, Read: "none", ReadAt: at, Target: "gaming", Changing: true},
			`"none" (stale, read 2026-10-13T20:00:00Z) → "gaming", changing`},
		{invocation.SubjectPlan{Read: "gaming", Target: "gaming"}, `"gaming" → "gaming", unchanged`},
	} {
		if got := foreseenPlan(c.plan, true); got != c.want {
			t.Errorf("%+v: %s, want %s", c.plan, got, c.want)
		}
		grimdark(t, c.want)
	}
}
