package cli

import (
	"os"
	"strings"
	"testing"
)

// exaltedSmtJSON is a state rite whose exalted machine-spirit smt keeps its omen
// in $TARGET.smt; its enact words tell what omen they set, and in whose
// home, into $TARGET.exalted.
const exaltedSmtJSON = `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
  {"name": "smt", "designation": "machine-spirit", "exalted": true, "omens": {"on": "off", "*": "on"},
   "auspex": {"scan": "cat $TARGET.smt"},
   "enact": {"*": "echo {{omen}} > $TARGET.smt; echo \"{{omen}} $SERVITOR_OMEN $HOME\" >> $TARGET.exalted"}}]}`

// TestInvokeII_AnExaltedSubjectIsSetWithOneProof: the exalted subject of a
// state rite is sanctioned by its master, the unit shown by subject, and
// then set by the exalted servitor alone with one proof; invoked again
// into the aspect it already stands in, the rite asks no proof at all.
func TestInvokeII_AnExaltedSubjectIsSetWithOneProof(t *testing.T) {
	e := newEnv(t)
	e.addRite("rites/smt.json", exaltedSmtJSON)
	if err := os.WriteFile(e.target+".smt", []byte("on\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := e.raiseAltar("sudo")
	a.lendSpeech()

	_, errOut, code := e.run("invoke", "smt", "on")
	if code != markRefused || !strings.Contains(errOut, `the subject "smt" is exalted`) ||
		!strings.Contains(errOut, "servitor sanction smt") || len(a.called()) != 0 {
		t.Fatalf("unsanctioned: exit %d\n%s", code, errOut)
	}
	out := e.mustRun("sanction", "smt")
	for _, want := range []string{"The exalted subjects of \"smt\"", "subject smt · machine-spirit",
		"enact on: echo on >", "omen: off"} {
		if !strings.Contains(out, want) {
			t.Errorf("the sanction shown lacks %q:\n%s", want, out)
		}
	}
	grimdark(t, out)

	out, errOut, code = e.run("invoke", "smt", "on")
	if code != 0 || e.exaltedSaid() != "off off /root\n" || len(a.called()) != 2 {
		t.Fatalf("exit %d, said %q, called %q\nstdout: %s\nstderr: %s", code, e.exaltedSaid(), a.called(), out, errOut)
	}
	if line := e.lastChronicled(); !strings.Contains(line, `"changed":["smt"]`) ||
		!strings.Contains(line, `"intercessor":"sudo"`) {
		t.Errorf("chronicled %s", line)
	}

	out, errOut, code = e.run("invoke", "smt", "on")
	if code != 0 || len(a.called()) != 2 || e.exaltedSaid() != "off off /root\n" {
		t.Fatalf("again: exit %d, called %q\nstdout: %s\nstderr: %s", code, a.called(), out, errOut)
	}
	if line := e.lastChronicled(); !strings.Contains(line, `"changed":[]`) || strings.Contains(line, "intercessor") {
		t.Errorf("chronicled %s", line)
	}
	out = e.mustRun("invoke", "smt", "off", "--foresee")
	if !strings.Contains(out, "subject smt · machine-spirit ✠") || !strings.Contains(out, "The exalted subjects stand "+
		"sanctioned") {
		t.Errorf("foreseen:\n%s", out)
	}
}
