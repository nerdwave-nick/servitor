package invocation

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestUtter_SpeaksInASessionOfItsOwn: a command uttered in a session of its
// own leads that session and its process group, and whoever asked is told
// the group; its words are captured as every command's, and its patience
// slays it as ever.
func TestUtter_SpeaksInASessionOfItsOwn(t *testing.T) {
	var summoned []int
	u := Utterance{
		// the fields of /proc/self/stat after the name: state, ppid, pgrp, session
		Argv:     []string{"bash", "-c", `s=$(cat /proc/$$/stat); s=${s##*) }; set -- $s; echo "$$ $3 $4"`},
		Dir:      t.TempDir(),
		Env:      []string{"PATH=/usr/local/bin:/usr/bin:/bin"},
		Patience: 10 * time.Second,
		Session:  true,
		Summoned: func(pgid int) { summoned = append(summoned, pgid) },
	}
	said, err := Utter(context.Background(), u)
	if err != nil {
		t.Fatalf("the command fell: %v (%q)", err, said)
	}
	f := strings.Fields(said)
	if len(f) != 3 || f[0] != f[1] || f[0] != f[2] {
		t.Fatalf("pid, group and session are %q, want one number thrice", said)
	}
	if pid, _ := strconv.Atoi(f[0]); len(summoned) != 1 || summoned[0] != pid {
		t.Errorf("told the groups %v, want [%d]", summoned, pid)
	}

	u.Argv, u.Patience = []string{"bash", "-c", "echo begun; sleep 30"}, 200*time.Millisecond
	said, err = Utter(context.Background(), u)
	if err == nil || !strings.Contains(err.Error(), "patience") || said != "begun\n" {
		t.Errorf("an impatient command: %q, %v", said, err)
	}
}
