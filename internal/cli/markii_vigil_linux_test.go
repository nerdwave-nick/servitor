package cli

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

// TestVigil_ADepartedListenerSlaysEveryTrial: a write finding no reader ends
// the vigil at once, slaying every trial of every machine-spirit still
// awake with its process group.
func TestVigil_ADepartedListenerSlaysEveryTrial(t *testing.T) {
	e := newEnv(t)
	pids := e.t.TempDir() + "/pids"
	trial := `sleep 30 & echo $$ $! >> ` + pids + `; wait`
	for i := range 2 {
		e.addRite(fmt.Sprintf("rites/sloth%d.json", i), `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
		  {"name": "s", "designation": "machine-spirit", "auspex": {"trials": {"*": "`+trial+`"}, "patience": "60s"},
		   "enact": {"*": "true"}}]}`)
	}
	e.addRite("rites/hasty.json", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "s", "designation": "machine-spirit", "auspex": {"scan": "sleep 0.3; echo on"}, "enact": {"*": "true"}}]}`)
	l := &listener{t: t, bear: 4} // the first telling only
	var errb bytes.Buffer

	start := time.Now()
	Execute([]string{"--librarium", e.cfgDir, "census", "--vigil"}, l, &errb)

	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("the vigil outlasted its listener by %v", elapsed)
	}
	if errb.Len() > 0 || len(l.lines) != 4 {
		t.Errorf("stderr %q; lines %v", errb.String(), l.lines)
	}
	for _, pid := range awake(t, pids, 8) { // two trials of two rites, each beside a child
		within(t, func() bool { return ended(pid) }, fmt.Sprintf("process %d of a trial outlived the vigil", pid))
	}
}
