package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestCensus_AugursInParallel: the census awakens the auspexes of its rites
// together, never more than sixteen at once, so that n rites whose auspex
// labours a second each are read in ⌈n/16⌉ seconds rather than n.
func TestCensus_AugursInParallel(t *testing.T) {
	for _, n := range []int{20, 12} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			e := newEnv(t)
			awake := filepath.Join(t.TempDir(), "awake")
			if err := os.Mkdir(awake, 0o755); err != nil {
				t.Fatal(err)
			}
			counted := filepath.Join(t.TempDir(), "counted")
			// Each auspex marks itself awake while it labours and counts the
			// auspexes awake beside it.
			auspex := fmt.Sprintf(`touch %[1]s/$$; ls %[1]s | wc -l >> %[2]s; sleep 1; rm %[1]s/$$; echo on`, awake, counted)
			for i := range n {
				e.addRite(fmt.Sprintf("rites/vigil%02d.json", i), `{"pattern": "Mark II", "aspects": ["on", "off"],
  "subjects": [{"name": "spirit", "designation": "machine-spirit", "enact": {"*": "true"},
    "auspex": {"scan": "`+auspex+`", "patience": "10s"}}]}`)
			}

			start := time.Now()
			entries := e.census()
			elapsed := time.Since(start)

			rounds := time.Duration((n+15)/16) * time.Second
			if elapsed < rounds || elapsed > rounds+900*time.Millisecond {
				t.Errorf("the census of %d rites took %v, want about %v", n, elapsed, rounds)
			}
			for i := range n {
				rite := fmt.Sprintf("vigil%02d", i)
				if got := entries[rite]; got["auspex"] != "heeded" || got["aspect"] != "on" {
					t.Errorf("%s: %v", rite, got)
				}
			}
			b, err := os.ReadFile(counted)
			if err != nil {
				t.Fatal(err)
			}
			most, counts := 0, strings.Fields(string(b))
			for _, c := range counts {
				awakeAtOnce, _ := strconv.Atoi(c)
				most = max(most, awakeAtOnce)
			}
			if len(counts) != n || most != min(n, 16) {
				t.Errorf("%d auspexes counted %v beside them; at most %d at once, want %d", n, counts, most, min(n, 16))
			}
		})
	}
}
