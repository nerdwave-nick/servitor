package sanction

import (
	"bytes"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// firewall is a state rite of exalted machine-spirits only the root can
// read: nft, read by a privileged scan that splices a decree-only
// inscription, with relics of its own; msr, which declares no omens and is
// read by a privileged trial of one omen (the other bears none); and smt,
// whose auspex is common.
const firewall = `{
  "pattern": "Mark II",
  "aspects": ["on", "off"],
  "inscriptions": {"table": {"decrees": {"*": "gaming"}}},
  "subjects": [
    {"name": "nft", "designation": "machine-spirit", "exalted": true, "omens": {"on": "{{inscription.table}}", "*": "none"},
     "auspex": {"scan": "nft list tables | grep -qw {{inscription.table}} && echo {{inscription.table}} || echo none",
                "patience": "3s", "exalted": true},
     "enact": {"*": "nft -f /etc/nftables.d/$SERVITOR_OMEN.nft"}, "relics": ["nft/rules.conf"]},
    {"name": "msr", "designation": "machine-spirit", "exalted": true,
     "auspex": {"trials": {"on": "rdmsr 0x1a4 | grep -qx {{omen}}f"}, "exalted": true},
     "enact": {"*": "wrmsr -a 0x1a4 {{omen}}"}},
    {"name": "smt", "designation": "machine-spirit", "exalted": true, "omens": {"on": "off", "*": "on"},
     "auspex": {"scan": "cat /sys/devices/system/cpu/smt/control"},
     "enact": {"*": "echo {{omen}} > /sys/devices/system/cpu/smt/control"}}
  ]
}`

func firewallFiles() map[string]string {
	return map[string]string{"nft/rules.conf": "table inet gaming\n"}
}

func buildFirewall(t *testing.T, scripture string) *Unit {
	t.Helper()
	u, err := Build(librariumOf(t, "firewall", scripture, firewallFiles()), options)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// TestBuildII_APrivilegedAuspexIsDrawnIntoTheUnit: the privileged auspex of
// an exalted machine-spirit is part of its unit in every plan: its scan, or
// its trials keyed by every declared omen that bears one, each rendered for
// the plan's aspect, its patience, and whether the spirit prints the names
// of aspects. A common auspex is no part of it.
func TestBuildII_APrivilegedAuspexIsDrawnIntoTheUnit(t *testing.T) {
	u := buildFirewall(t, firewall)
	golden(t, "firewall.json", append(u.Canonical(), '\n'))
	golden(t, "firewall.shown", []byte(strings.Join(Show(u), "\n")+"\n"))
	on, _ := u.SigilOf("on")
	off, _ := u.SigilOf("off")
	if got := []string{u.Sigil(), on, off}; !slices.Equal(got, goldenSigilsFirewall) {
		t.Errorf("sigils %q, want %q", got, goldenSigilsFirewall)
	}
	for _, p := range u.Aspects {
		nft, msr, smt := p.Subjects[0], p.Subjects[1], p.Subjects[2]
		if a := nft.Auspex; a == nil ||
			a.Scan != "nft list tables | grep -qw gaming && echo gaming || echo none" || len(a.Trials) != 0 ||
			a.Patience != 3*time.Second || a.NamesAspects {
			t.Errorf("%s: nft %+v", p.Aspect, nft.Auspex)
		}
		if a := msr.Auspex; a == nil || a.Scan != "" || len(a.Trials) != 1 ||
			a.Trials["on"] != "rdmsr 0x1a4 | grep -qx onf" || a.Patience != 2*time.Second || !a.NamesAspects {
			t.Errorf("%s: msr %+v", p.Aspect, msr.Auspex)
		}
		if smt.Auspex != nil {
			t.Errorf("%s: the common auspex of smt is drawn: %+v", p.Aspect, smt.Auspex)
		}
	}
	if !strings.Contains(string(u.Canonical()), `"auspex":{"names-aspects":false,"patience":"3s","scan":`) {
		t.Errorf("the auspex of nft is written %s", u.Canonical())
	}
}

// goldenSigilsFirewall are the sigils of the golden unit of firewall:
// whole, of "on", of "off".
var goldenSigilsFirewall = []string{
	"69de0aa5de297e7f4f1970a3bc789b9cde1f18a43c21659083c25926bdd99ea4",
	"0be49da1cec853b325b905d73f9fd67c12cecbe2e72cb63f3590d9085c8ba083",
	"269222e7fb40699327e1e2153b98fca580fa8ea9609f8ab1206f5f6b5ab70fc5",
}

// TestBuildII_APrivilegedAuspexLapsesEveryAspect: the words, tongue,
// patience and relics of a privileged auspex are part of the unit, and its
// subject lies in every plan: editing any of them lapses the plan of every
// aspect.
func TestBuildII_APrivilegedAuspexLapsesEveryAspect(t *testing.T) {
	base := buildFirewall(t, firewall)
	for what, c := range map[string]struct{ old, new string }{
		"the scan":     {`echo {{inscription.table}} || echo none"`, `echo {{inscription.table}} || echo nothing"`},
		"a trial":      {`"rdmsr 0x1a4 | grep -qx {{omen}}f"`, `"rdmsr 0x1a5 | grep -qx {{omen}}f"`},
		"the patience": {`"patience": "3s"`, `"patience": "4s"`},
		"a new trial":  {`{"on": "rdmsr`, `{"off": "true", "on": "rdmsr`},
		"the tongue":   {`"relics": ["nft/rules.conf"]}`, `"relics": ["nft/rules.conf"], "tongue": "sh"}`},
		"a relic":      {`"relics": ["nft/rules.conf"]}`, `"relics": []}`},
	} {
		edited := strings.Replace(firewall, c.old, c.new, 1)
		if edited == firewall {
			t.Fatalf("%s: %s is not in the rite", what, c.old)
		}
		u := buildFirewall(t, edited)
		for _, aspect := range []string{"on", "off"} {
			if mustSigil(t, base, aspect) == mustSigil(t, u, aspect) {
				t.Errorf("%s was edited, yet the sigil of %q stays", what, aspect)
			}
		}
	}
	// So too the relic's scripture.
	files := firewallFiles()
	files["nft/rules.conf"] = "table inet war\n"
	u, err := Build(librariumOf(t, "firewall", firewall, files), options)
	must(t, err)
	if mustSigil(t, base, "off") == mustSigil(t, u, "off") {
		t.Error("the relic of the privileged auspex was edited, yet the sigil of \"off\" stays")
	}
}

// TestBuildII_ACommonAuspexBesidePrivilegedOnesLapsesNothing: editing the
// common auspex of an exalted subject leaves every sigil as it was, even in
// a rite holding privileged auspexes.
func TestBuildII_ACommonAuspexBesidePrivilegedOnesLapsesNothing(t *testing.T) {
	edited := strings.Replace(firewall, "smt/control\"}", "smt/active\"}", 1)
	if edited == firewall {
		t.Fatal("the common auspex is not in the rite")
	}
	sameSigils(t, "the common auspex", buildFirewall(t, firewall), buildFirewall(t, edited))
}

// TestBuildII_APrivilegedAuspexIsReadBackWhole: the sanction of a rite with
// privileged auspexes is written and read back strictly, its sigil
// unchanged.
func TestBuildII_APrivilegedAuspexIsReadBackWhole(t *testing.T) {
	u := buildFirewall(t, firewall)
	store := Store{Root: t.TempDir(), UID: 1000}
	data, err := json.Marshal(New(u, time.Date(2026, 10, 13, 18, 0, 0, 0, time.UTC)))
	must(t, err)
	must(t, os.MkdirAll(filepath.Dir(store.Path("firewall")), 0o750))
	must(t, os.WriteFile(store.Path("firewall"), data, 0o640))
	kept, err := store.Read("firewall")
	if err != nil || kept == nil || kept.Sigil != u.Sigil() || !bytes.Equal(kept.Canonical(), u.Canonical()) {
		t.Fatalf("read back %v: %v", kept, err)
	}
	if a := kept.Aspects[0].Subjects[1].Auspex; a == nil || a.Trials["on"] == "" || !a.NamesAspects {
		t.Errorf("the auspex of msr is read back as %+v", a)
	}
}
