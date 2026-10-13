package librarium

import (
	"strings"
	"testing"
)

// splitLock is the exalted rite of the codex: kernel knobs written by
// exalted incantations, a vessel of /etc transcribed, a scroll reciting its
// relic.
const splitLock = `{
  "pattern": "Mark I",
  "aspects": ["on", "off"],
  "inscriptions": {"level": {"mandatory": true, "decrees": {"on": "0", "*": "1"}}},
  "liturgy": [
    {"incantation": {"on": "echo off > /sys/devices/system/cpu/smt/control",
                     "*":  "echo on > /sys/devices/system/cpu/smt/control"}, "exalted": true},
    {"vox-cast": "progress"},
    {"incantation": "sysctl -wq kernel.split_lock_mitigate={{inscription.level}}", "exalted": true,
     "reversion": "sysctl -wq kernel.split_lock_mitigate=$SERVITOR_FORMER_ASPECT"},
    {"transcription": "/etc/sysctl.d/90-{{rite.name}}.conf", "scripture": "kernel.split_lock_mitigate = {{inscription.level}}",
     "exalted": true},
    {"litany": "scripts/governor", "offerings": ["{{aspect}}"], "exalted": true, "relics": ["scripts/governor.conf"]},
    {"incantation": "notify-send {{former}}", "exalted": false},
    {"vox-cast": "success"}
  ]
}`

func TestParse_ExaltedVersesAreRead(t *testing.T) {
	r, fs := parse("split-lock", "/lib/rites/split-lock.json", []byte(splitLock))
	if len(fs) != 0 {
		t.Fatalf("findings:\n%v", fs)
	}
	var exalted []bool
	for _, s := range r.Liturgy {
		exalted = append(exalted, IsExalted(s))
	}
	if want := []bool{true, false, true, true, true, false, false}; !equalBools(exalted, want) {
		t.Fatalf("exalted verses %v, want %v", exalted, want)
	}
	if l := r.Liturgy[4].(*Litany); len(l.Relics) != 1 || l.Relics[0] != "scripts/governor.conf" {
		t.Fatalf("relics %v", l.Relics)
	}
	if got := r.ExaltedVerses(); len(got) != 4 || got[0] != 1 || got[3] != 5 || !r.Exalted() {
		t.Fatalf("exalted verses %v", got)
	}
	if plain, _ := parse("x", "/x.json", []byte(mark(``, `{"incantation": "a"}`))); plain.Exalted() || plain.ExaltedVerses() != nil {
		t.Fatal("a rite without exalted verses is told exalted")
	}
}

func equalBools(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestParse_EdictMayExaltIncantationsAndLitanies(t *testing.T) {
	scripture := edict(``, `{"incantation": "systemctl reboot", "exalted": true},
		{"litany": "scripts/prune", "exalted": true, "relics": ["scripts/prune.list"]}`)
	r, fs := parse("reboot", "/lib/rites/reboot.json", []byte(scripture))
	if r == nil || len(fs) != 0 || !r.Edict || !r.Exalted() {
		t.Fatalf("rite %v, findings:\n%v", r, fs)
	}
}

var exaltedHeresyCases = []heresyCase{
	{"exalted vox-cast", mark(``, `{"vox-cast": "progress", "exalted": true}`), Heresy,
		`a vox-cast is never exalted`, `"exalted"`, 0},
	{"exalted no boolean", mark(``, `{"incantation": "a", "exalted": "yes"}`), Heresy,
		`the "exalted" of the incantation of verse 1 must be written as true or false, yet here stands a string`, `"yes"`, 0},
	{"exalted aspect map", mark(``, `{"incantation": "a", "exalted": {"on": true, "*": false}}`), Heresy,
		`a verse is exalted for every aspect or for none`, `{"on": true`, 0},
	{"relics not exalted", mark(``, `{"litany": "s", "relics": ["s.conf"]}`), Heresy,
		`the litany of verse 1 names "relics", yet it is not exalted`, `"relics"`, 0},
	{"relics exalted false", mark(``, `{"incantation": "a", "exalted": false, "relics": []}`), Heresy,
		`the incantation of verse 1 names "relics", yet it is not exalted`, `"relics"`, 0},
	{"relics on a sanctum", mark(``, `{"sanctum": "/v.kdl", "scripture": "", "exalted": true, "relics": []}`), Heresy,
		`the key "relics" is not written in the codex for the sanctum of verse 1`, `"relics"`, 0},
	{"relics no list", mark(``, `{"litany": "s", "exalted": true, "relics": "s.conf"}`), Heresy,
		`the "relics" of the litany of verse 1 must be written as a list of paths`, `"s.conf"`, 0},
	{"relic no string", mark(``, `{"litany": "s", "exalted": true, "relics": [1]}`), Heresy,
		`relic 1 of the litany of verse 1 must be written as a string`, `1]`, 0},
	{"relic empty", mark(``, `{"litany": "s", "exalted": true, "relics": [" "]}`), Heresy,
		`relic 1 of the litany of verse 1 may not be empty`, `" "`, 0},
	{"relic from the root", mark(``, `{"litany": "s", "exalted": true, "relics": ["a", "/etc/s.conf"]}`), Heresy,
		`relic 2 of the litany of verse 1, "/etc/s.conf", lies outside the hall of the rite`, `"/etc/s.conf"`, 0},
	{"relic above the hall", mark(``, `{"incantation": "a", "exalted": true, "relics": ["x/../../s.conf"]}`), Heresy,
		`lies outside the hall of the rite`, `"x/../../s.conf"`, 0},
	{"relic the hall itself", mark(``, `{"incantation": "a", "exalted": true, "relics": ["x/.."]}`), Heresy,
		`lies outside the hall of the rite`, `"x/.."`, 0},

	{"exalted former", mark(``, `{"incantation": "echo {{former}}", "exalted": true}`), Heresy,
		`speak it as "$SERVITOR_FORMER_ASPECT"`, `{{former}}`, 0},
	{"exalted former in reversion", mark(``, `{"incantation": "a", "exalted": true, "reversion": {"on": "b {{former}}", "*": ""}}`), Heresy,
		`{{former}} names the aspect the machine stood in before`, `{{former}}`, 0},
	{"exalted former in vessel", mark(``, `{"transcription": "/etc/{{former}}", "scripture": null, "exalted": true}`), Heresy,
		`"$SERVITOR_FORMER_ASPECT"`, `{{former}}`, 0},
	{"exalted former in anchor", mark(``, `{"tether": "/etc/t", "anchor": "/usr/{{former}}", "exalted": true}`), Heresy,
		`"$SERVITOR_FORMER_ASPECT"`, `{{former}}`, 0},
	{"exalted former in offering", mark(``, `{"litany": "s", "offerings": ["{{former}}"], "exalted": true}`), Heresy,
		`"$SERVITOR_FORMER_ASPECT"`, `{{former}}`, 0},
	{"exalted former in scripture", mark(``, `{"sanctum": "/etc/v.conf", "scripture": ["a", "{{former}}"], "exalted": true}`), Heresy,
		`"$SERVITOR_FORMER_ASPECT"`, `{{former}}`, 0},
	{"exalted former in tome", mark(``, `{"sanctum": "/etc/v.conf", "scripture": {"tome": "{{former}}.conf"}, "exalted": true}`), Heresy,
		`"$SERVITOR_FORMER_ASPECT"`, `{{former}}`, 0},
	{"exalted fleeting inscription", `{"pattern": "Mark I", "aspects": ["on", "off"],
  "inscriptions": {"cpu-mode": {"mandatory": true, "decrees": {"on": "a"}}},
  "liturgy": [{"incantation": "x {{inscription.cpu-mode}}", "exalted": true}]}`, Heresy,
		`speak it as "$SERVITOR_INSCRIPTION_CPU_MODE"`, `{{inscription.cpu-mode}}`, 0},
	{"exalted fleeting inscription of an edict", `{"pattern": "Mark I", "edict": true,
  "inscriptions": {"region": {"mandatory": true}},
  "liturgy": [{"incantation": "x {{inscription.region}}", "exalted": true}]}`, Heresy,
		`"$SERVITOR_INSCRIPTION_REGION"`, `{{inscription.region}}`, 0},

	{"exalted vessel not rooted", mark(``, `{"sanctum": "v.conf", "scripture": "", "exalted": true}`), Heresy,
		`the vessel v.conf of the exalted sanctum of verse 1 does not begin at the root`, `"v.conf"`, 0},
	{"exalted vessel in the home", mark(``, `{"transcription": "~/.bashrc", "scripture": null, "exalted": true}`), Heresy,
		`does not begin at the root`, `"~/.bashrc"`, 0},
	{"exalted vessel rendered unrooted", mark(``, `{"transcription": "{{aspect}}/x", "scripture": null, "exalted": true}`), Heresy,
		`the vessel on/x of the exalted transcription of verse 1 does not begin at the root`, `"{{aspect}}/x"`, 0},
	{"exalted tether not rooted", mark(``, `{"tether": "$HOME/t", "anchor": "/usr/a", "exalted": true}`), Heresy,
		`the bound name $HOME/t of the exalted tether of verse 1 does not begin at the root`, `"$HOME/t"`, 0},
	{"exalted vessel of the kernel", mark(``, `{"transcription": "/sys/kernel/mm/x", "scripture": "1", "exalted": true}`), Heresy,
		`/sys/kernel/mm/x lies beneath /sys, where no vessel is kept`, `"/sys/kernel/mm/x"`, 0},
	{"exalted tether of the devices", mark(``, `{"tether": "/dev/../dev/x", "anchor": "/usr/a", "exalted": true}`), Heresy,
		`write it by an exalted incantation`, `"/dev/../dev/x"`, 0},
	{"exalted vessel of the processes", mark(``, `{"sanctum": "/proc/sys/x", "scripture": "", "exalted": true}`), Heresy,
		`beneath /proc`, `"/proc/sys/x"`, 0},

	{"intercessor in a common verse", mark(``, `{"incantation": "sudo sysctl -w a=1"}`), Impurity,
		`verse 1 calls upon the intercessor "sudo" within words that are not exalted`, `"sudo sysctl`, 0},
	{"intercessor in a reversion", mark(``, `{"incantation": "a", "reversion": {"on": "b", "*": "x && /usr/bin/run0 c"}}`), Impurity,
		`"run0"`, `"x && /usr`, 0},
	{"intercessor pkexec", mark(``, `{"incantation": "pkexec tee /etc/x"}`), Impurity, `write "exalted": true`, `"pkexec`, 0},
	{"intercessor doas", mark(``, `{"litany": "s", "reversion": "(doas x)"}`), Impurity, `"doas"`, `"(doas x)"`, 0},
	{"exalted speech names the home", mark(``, `{"incantation": "cp $HOME/a /etc/a", "exalted": true}`), Impurity,
		`the exalted speech of verse 1 reaches into the home`, `"cp $HOME`, 0},
	{"exalted speech names the tilde", mark(``, `{"incantation": "a", "exalted": true, "reversion": "cat ~/b"}`), Impurity,
		`reaches into the home`, `"cat ~/b"`, 0},
	{"exalted speech names a stray", mark(``, `{"incantation": "./tune.sh", "exalted": true, "relics": ["tuned.sh"]}`), Impurity,
		`the exalted speech of verse 1 reaches for tune.sh, which is not among its relics`, `"./tune.sh"`, 0},
	{"exalted speech sources a stray", mark(``, `{"incantation": "source lib.sh; go", "exalted": true}`), Impurity,
		`reaches for lib.sh, which is not among its relics`, `"source lib.sh`, 0},
	{"exalted speech sources from the root", mark(``, `{"incantation": "a; . /etc/os-release", "exalted": true}`), Impurity,
		`reaches for /etc/os-release, which is not among its relics`, `"a; .`, 0},
	{"guarded call beside exalted verses", mark(``, `{"incantation": "sysctl -w x=1", "exalted": true},
		{"incantation": {"on": "systemctl restart tuned", "*": "true"}}`), Impurity,
		`verse 2 calls "systemctl", which the machine guards by an intercessor of its own`, `"systemctl restart`, 0},
	{"guarded reversion beside exalted verses", mark(``, `{"litany": "s", "reversion": "hostnamectl hostname x"},
		{"incantation": "a", "exalted": true}`), Impurity,
		`verse 1 calls "hostnamectl"`, `"hostnamectl`, 0},
}

func init() { heresyCases = append(heresyCases, exaltedHeresyCases...) }

func TestParse_ExaltedScriptureIsPure(t *testing.T) {
	cases := map[string]string{
		"exalted false": mark(``, `{"incantation": "echo {{former}}", "exalted": false}`),
		"fixed marks": mark(``, `{"incantation": "x {{aspect}} {{rite.name}} {{inscription.reason}}", "exalted": true,
			"reversion": "y $SERVITOR_FORMER_ASPECT \"$SERVITOR_INSCRIPTION_REASON\""}`),
		"decreed mandatory": `{"pattern": "Mark I", "aspects": ["on", "off"],
			"inscriptions": {"m": {"mandatory": true, "decrees": {"on": "a", "*": "b"}}},
			"liturgy": [{"incantation": "x {{inscription.m}}", "exalted": true}]}`,
		"rooted vessels": mark(``, `{"sanctum": "/etc/{{aspect}}.conf", "scripture": "", "exalted": true},
			{"transcription": "/proc/../etc/x", "scripture": null, "exalted": true},
			{"tether": "/etc/{{rite.name}}", "anchor": {"on": "/usr/a", "off": null}, "exalted": true}`),
		"common vessels anywhere": mark(``, `{"transcription": "/sys/x", "scripture": "1"}, {"tether": "t", "anchor": "a"}`),
		"relics named": mark(``, `{"litany": "s", "exalted": true, "relics": ["s.conf", "lib/./a.sh", "x/../b"]},
			{"incantation": "source ./lib/a.sh && . b && ./s.conf", "exalted": true, "relics": ["lib/a.sh", "b", "s.conf"]}`),
		"intercessor exalted":                 mark(``, `{"incantation": "sudo -k", "exalted": true}`),
		"guarded call without exalted verses": mark(``, `{"incantation": "systemctl restart tuned"}`),
		"guarded call of the master":          mark(``, `{"incantation": "systemctl --user restart x"}, {"incantation": "a", "exalted": true}`),
		"guarded call exalted":                mark(``, `{"incantation": "systemctl restart x", "exalted": true}`),
		"common speech names the home":        mark(``, `{"incantation": "cat ~/a $HOME/b ./c; . d"}, {"incantation": "a", "exalted": true}`),
		"plain words":                         mark(``, `{"incantation": "pseudo sudoku ./. ~user", "exalted": false}, {"incantation": "echo sudoers", "exalted": true}`),
	}
	for name, scripture := range cases {
		t.Run(name, func(t *testing.T) {
			if r, fs := parse("rite", "/lib/rites/rite.json", []byte(scripture)); r == nil || len(fs) > 0 {
				t.Fatalf("rite %v, findings:\n%v", r, fs)
			}
		})
	}
}

// TestParse_FleetingWordsStayInCommonVerses: words given only at invocation
// may still be spliced into the verses that are not exalted.
func TestParse_FleetingWordsStayInCommonVerses(t *testing.T) {
	_, fs := parse("rite", "/r.json", []byte(`{"pattern": "Mark I", "aspects": ["on", "off"],
		"inscriptions": {"m": {"mandatory": true, "decrees": {"on": "a"}}},
		"liturgy": [{"incantation": "x {{inscription.m}} {{former}}"}, {"incantation": "y", "exalted": true}]}`))
	if fs.Heretical() {
		t.Fatalf("findings:\n%v", fs)
	}
}

// TestParse_ExaltedHeresyStandsAlone: a vessel of the kernel is denounced
// once, though it renders alike for every aspect.
func TestParse_ExaltedHeresyStandsAlone(t *testing.T) {
	_, fs := parse("rite", "/r.json", []byte(mark(``, `{"transcription": "/sys/x", "scripture": "1", "exalted": true}`)))
	if len(fs) != 1 || !strings.Contains(fs[0].Message, "/sys") {
		t.Fatalf("findings:\n%v", fs)
	}
}

func TestMarshal_ExaltedRitesRoundTrip(t *testing.T) {
	forged := &Rite{Name: "reboot", Edict: true, Liturgy: []Step{
		&Incantation{Command: Uniform("systemctl reboot"), Utterance: Utterance{Exalted: true, Relics: []string{}}},
		&Litany{Scroll: Uniform("s"), Utterance: Utterance{Exalted: true, Relics: []string{"a", "b"}}},
	}}
	roundTrip(t, forged)
}
