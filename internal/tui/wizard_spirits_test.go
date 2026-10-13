package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// The owner's gaming and do-not-disturb rites, as the specification shows
// them in pattern Mark II (§19.1).
const (
	gamingII = `{
  "pattern": "Mark II",
  "purpose": "enable or disable gaming optimization",
  "aspects": ["on", "off"],
  "vox-cast": ["progress", "success"],
  "subjects": [
    {
      "name": "smt",
      "designation": "machine-spirit",
      "exalted": true,
      "omens": {"on": "off", "*": "on"},
      "auspex": {"scan": "cat /sys/devices/system/cpu/smt/control"},
      "enact": {"*": "echo {{omen}} > /sys/devices/system/cpu/smt/control"}
    },
    {
      "name": "split-lock",
      "designation": "machine-spirit",
      "exalted": true,
      "omens": {"on": "0", "*": "1"},
      "auspex": {"scan": "sysctl -n kernel.split_lock_mitigate"},
      "enact": {"*": "sysctl -wq kernel.split_lock_mitigate={{omen}}"}
    },
    {
      "name": "scheduler",
      "designation": "machine-spirit",
      "exalted": true,
      "omens": {"on": "cosmos", "*": "eevdf"},
      "auspex": {"trials": {"cosmos": "systemctl is-active -q scx-cosmos"}, "patience": "1s"},
      "enact": {"cosmos": "systemd-run --unit=scx-cosmos --collect scx_cosmos -p 0 -c 0",
                "*":      "systemctl stop scx-cosmos"}
    }
  ]
}`
	dndII = `{
  "pattern": "Mark II",
  "purpose": "silence the heralds of the machine",
  "aspects": ["on", "off"],
  "vox-cast": ["success"],
  "subjects": [
    {
      "name": "mako",
      "designation": "machine-spirit",
      "omens": {"on": "do-not-disturb", "*": "default"},
      "auspex": {"scan": "makoctl mode"},
      "enact": {"*": "makoctl mode -s \"$SERVITOR_OMEN\""},
      "recessional": ["herald-silence"]
    }
  ],
  "observances": [
    {"name": "herald-silence", "designation": "incantation", "to": ["on"],
     "words": "notify-send -t 2000 'The heralds fall silent'"}
  ]
}`
)

// sealedAs holds the scripture the wizard sealed for the rite name to the
// scripture want: it must read as the very rite want is, and be written as
// the servitor writes that rite.
func (h *harness) sealedAs(name, want string) {
	h.t.Helper()
	path := filepath.Join(h.dir, "rites", name+".json")
	rite, found := librarium.Parse(name, path, []byte(want))
	if found.Heretical() {
		h.t.Fatalf("the scripture wanted is heretical: %v", found)
	}
	canon, err := librarium.Marshal(rite)
	if err != nil {
		h.t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != string(canon) {
		h.t.Fatalf("the scripture sealed (%v):\n%s\nwant:\n%s", err, data, canon)
	}
	if got := h.loadRite(name); !got.Equal(rite) {
		h.t.Fatalf("the rite sealed reads otherwise than the scripture wanted:\n%s", data)
	}
}

// beginMarkII opens the Rite page of a new rite of pattern Mark II named
// name with purpose, and chooses its vox-casts.
func (h *harness) beginMarkII(name, purpose string, progress, success bool) {
	h.t.Helper()
	h.keys("n")
	h.typeText(name)
	h.beholdTheRitePage()
	h.keys("down")
	h.typeText(purpose)
	h.focusOn("vox:progress")
	if progress {
		h.keys("space")
	}
	h.keys("tab")
	if success {
		h.keys("space")
	}
	h.keys("ctrl+s")
	h.mustShow("✔ Rite", "2 Subjects", "3 Observances", "4 Seal")
}

// spirit writes a new machine-spirit, exalted or not: its name, its omen
// for the aspect on and the omen of every other aspect, its scan, and the
// enact words of every omen.
func (h *harness) spirit(name string, exalted bool, on, other, scan, enact string) {
	h.t.Helper()
	h.keys("a", "4")
	h.mustShow("New subject", "⚙ machine-spirit", "Name of the subject", "Exalted subject")
	h.typeText(name)
	if exalted {
		h.focusOn("exalted")
		h.keys("space")
		h.mustShow("Relics, one per line")
	}
	h.keys("ctrl+s")
	h.omens(on, other)
	h.mustShow("auspex", "Scan")
	h.typeText(scan)
	h.keys("ctrl+s")
	h.mustShow("enact", "Words that enact every other omen (*)")
	h.focusOn("omen:*")
	h.typeText(enact)
	h.keys("ctrl+s")
}

// omens writes the omens of a machine-spirit of the aspects on and off:
// on bears its own, off the omen of every other aspect.
func (h *harness) omens(on, other string) {
	h.t.Helper()
	h.mustShow("omens", "Each aspect is its own omen", "[✔] yes")
	h.mustNotShow(`Omen of aspect "on"`)
	h.keys("space")
	h.mustShow(`Omen of aspect "on"`, `Omen of aspect "off"`)
	h.focusOn("omen:on")
	h.typeText(on)
	h.focusOn("same:off")
	h.keys("space")
	h.focusOn("omen:*")
	h.typeText(other)
	h.keys("ctrl+s")
}

// TestWizard_ConsecrateTheGamingRite: the owner's gaming rite — three
// exalted machine-spirits, two read by their scan and one by a trial for
// one omen, the other left without a trial — is consecrated by key presses
// alone and sealed as the specification writes it.
func TestWizard_ConsecrateTheGamingRite(t *testing.T) {
	h := newHarness(t)
	h.beginMarkII("gaming", "enable or disable gaming optimization", true, true)
	h.keys("a")
	h.mustShow("1 § sanctum", "2 ¶ transcription", "3 ↣ tether", "4 ⚙ machine-spirit")
	h.keys("esc")
	h.spirit("smt", true, "off", "on", "cat /sys/devices/system/cpu/smt/control",
		"echo {{omen}} > /sys/devices/system/cpu/smt/control")
	h.mustShow("1 ⚙ smt")
	h.spirit("split-lock", true, "0", "1", "sysctl -n kernel.split_lock_mitigate",
		"sysctl -wq kernel.split_lock_mitigate={{omen}}")
	h.mustShow("2 ⚙ split-lock")

	h.keys("a", "4")
	h.typeText("scheduler")
	h.focusOn("exalted")
	h.keys("space", "ctrl+s")
	h.omens("cosmos", "eevdf")
	h.mustShow(`Trial of omen "cosmos"`, `Trial of omen "eevdf"`, "Patience of the auspex", "Exalted auspex")
	h.focusOn("omen:cosmos")
	h.typeText("systemctl is-active -q scx-cosmos")
	h.focusOn("patience")
	h.typeText("1s")
	h.keys("ctrl+s")
	h.mustShow(`Omen "cosmos" is enacted by the words of every other omen`)
	h.focusOn("same:cosmos")
	h.keys("space")
	h.mustShow(`Words that enact omen "cosmos"`)
	h.focusOn("omen:cosmos")
	h.typeText("systemd-run --unit=scx-cosmos --collect scx_cosmos -p 0 -c 0")
	h.focusOn("omen:*")
	h.typeText("systemctl stop scx-cosmos")
	h.keys("ctrl+s")
	h.mustShow("1 ⚙ smt", "2 ⚙ split-lock", "3 ⚙ scheduler")

	h.keys("tab")
	h.mustShow("✔ Subjects", "The observances of the rite gaming", "No observance is declared yet")
	h.keys("tab")
	h.mustShow("The rite is pure", `"designation": "machine-spirit"`)
	h.keys("enter")
	h.mustShow("The rite gaming is consecrated")
	h.sealedAs("gaming", gamingII)
}

// TestWizard_ConsecrateTheDoNotDisturbRite: the owner's do-not-disturb
// rite — a machine-spirit whose recessional is an observance spoken only
// into the aspect on — is consecrated by key presses alone: the observance
// is added at its own station, then chosen as the subject's recessional.
func TestWizard_ConsecrateTheDoNotDisturbRite(t *testing.T) {
	h := newHarness(t)
	h.beginMarkII("dnd", "silence the heralds of the machine", false, true)
	h.spirit("mako", false, "do-not-disturb", "default", "makoctl mode", `makoctl mode -s "$SERVITOR_OMEN"`)
	h.mustShow("1 ⚙ mako")

	h.keys("tab", "a")
	h.mustShow("Choose the designation of the new observance", "1 » incantation", "2 ≡ litany")
	h.keys("1")
	h.mustShow("New observance", "» incantation", "Name of the observance", "Exalted observance", "Further rites")
	h.typeText("herald-silence")
	h.keys("ctrl+s")
	h.mustShow("Words")
	h.typeText("notify-send -t 2000 'The heralds fall silent'")
	h.keys("ctrl+s")
	h.mustShow("guards", `Spoken into aspect "on"`, `Spoken out of aspect "off"`)
	h.focusOn("to:on")
	h.keys("space", "ctrl+s")
	h.mustShow("1 » herald-silence", "to on")

	h.keys("esc")
	h.mustShow("The subjects of the rite dnd")
	h.keys("enter")
	for range 4 {
		h.keys("ctrl+s")
	}
	h.mustShow(`Observance "herald-silence"`, "◆ unheeded")
	h.keys("space", "space")
	h.mustShow("◆ recessional")
	h.keys("ctrl+s")
	h.mustShow("1 ⚙ mako")
	h.keys("tab", "tab")
	h.mustShow("The rite is pure")
	h.keys("enter")
	h.mustShow("The rite dnd is consecrated")
	h.sealedAs("dnd", dndII)
}

// everyFormII is a rite of pattern Mark II bearing every form the wizard
// writes beside those of the owner's rites: a machine-spirit whose omens
// are its aspects, one never read, with omens rendered from {{aspect}};
// one tried by a trial under "*"; one exalted, with relics and an exalted
// auspex; a transcription; introits written out of their order, and
// recessionals; observances of both designations, guarded to and from,
// with tongue, patience, reversion, offerings, exaltation and relics.
const everyFormII = `{
  "pattern": "Mark II",
  "aspects": ["dark", "light", "dusk"],
  "inscriptions": {"mood": {"decrees": {"*": "calm"}}},
  "vox-cast": ["progress"],
  "subjects": [
    {"name": "plain", "designation": "machine-spirit", "purpose": "its aspects are its omens",
     "auspex": {"scan": "cat $DATA/plain"}, "enact": {"dark": "echo dark > $DATA/plain", "*": "echo {{omen}} > $DATA/plain"},
     "tongue": "sh", "patience": "5s", "introit": ["lamp", "veil"]},
    {"name": "blind", "designation": "machine-spirit", "omens": {"dark": "0", "*": "{{aspect}}-1"},
     "enact": {"light-1": "echo light", "*": "true"}},
    {"name": "tried", "designation": "machine-spirit", "omens": {"dark": "a", "light": "b", "dusk": "c"},
     "auspex": {"trials": {"a": "test -e $DATA/a", "*": "false"}, "patience": "3s"},
     "enact": {"a": "touch $DATA/a", "b": "rm -f $DATA/a", "c": "rm -f $DATA/a"}, "recessional": ["hooks"]},
    {"name": "root", "designation": "machine-spirit", "exalted": true, "omens": {"dark": "on", "*": "off"},
     "auspex": {"scan": "cat /proc/sys/kernel/sysrq", "exalted": true},
     "enact": {"*": "scripts/root {{omen}}"}, "relics": ["scripts/root"]},
    {"name": "palette", "designation": "transcription", "vessel": "$DATA/palette.conf",
     "omens": {"dark": "dusk", "*": null}, "recessional": ["mark"]}
  ],
  "observances": [
    {"name": "veil", "designation": "incantation", "to": ["dark"], "from": ["light", "dusk"],
     "words": "echo veil", "tongue": "sh", "patience": "2s", "reversion": "echo unveil"},
    {"name": "lamp", "designation": "litany", "scroll": "scripts/lamp", "offerings": ["{{aspect}}", "x"],
     "exalted": true, "relics": ["scripts/lamp"]},
    {"name": "hooks", "designation": "incantation", "from": ["dark"], "words": "echo hooks"},
    {"name": "mark", "designation": "incantation", "words": "echo mark"}
  ]
}`

// throughSubject seals page after page of the subject or observance being
// amended until its list is shown again, gathering what its pages showed.
func (h *harness) throughSubject(list string, seen *strings.Builder) {
	h.t.Helper()
	for range 10 {
		seen.WriteString(h.screen())
		h.keys("ctrl+s")
		if strings.Contains(h.screen(), list) {
			return
		}
	}
	h.t.Fatalf("the pages would not be left:\n%s", h.screen())
}

// TestWizard_AmendEveryFormOfMarkII: the owner's gaming and do-not-disturb
// rites, and a rite of every other form, are amended through every page of
// every subject and observance, and each reads back as the very rite it
// was — a trial left unwritten for an omen stays unwritten.
func TestWizard_AmendEveryFormOfMarkII(t *testing.T) {
	cases := []struct {
		name, scripture string
		subjects        int
		observances     int
		shown           []string // shown upon the pages amended
	}{
		{"gaming", gamingII, 3, 0, []string{`Trial of omen "eevdf"`, "Exalted auspex", "Words that enact every other omen (*)"}},
		{"dnd", dndII, 1, 1, []string{`Observance "herald-silence"`, "◆ recessional", `Spoken into aspect "on"`}},
		{"forms", everyFormII, 5, 4, []string{`Trial of omen "a"`, `Omen "b" is tried by the trial of every other omen`,
			`Omen "light-1" is enacted by the words of every other omen`, "[✔] yes", "Offerings, one per line",
			`Spoken out of aspect "light"`, "◆ introit", "Exalted observance"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness(t)
			h.writeHall("scripts/root", "#!/bin/sh\n", 0o755)
			h.writeHall("scripts/lamp", "#!/bin/sh\n", 0o755)
			h.writeRite(c.name, c.scripture)
			before := *h.loadRite(c.name)
			h.keys("r", "/")
			h.typeText(c.name)
			h.keys("enter", "e")
			h.mustShow("Amendment of the rite "+c.name, "pattern Mark II")
			var seen strings.Builder
			h.throughSubject("The subjects of the rite", &seen)
			for range c.subjects {
				h.keys("enter")
				h.throughSubject("The subjects of the rite", &seen)
				h.keys("j")
			}
			h.keys("tab")
			h.mustShow("The observances of the rite " + c.name)
			for range c.observances {
				h.keys("enter")
				h.throughSubject("The observances of the rite", &seen)
				h.keys("j")
			}
			for _, s := range c.shown {
				if !strings.Contains(seen.String(), s) {
					t.Errorf("no page showed %q", s)
				}
			}
			h.keys("tab")
			h.mustShow("is pure", `"pattern": "Mark II"`)
			h.keys("enter")
			h.mustShow("is amended")
			if got := h.loadRite(c.name); !got.Equal(&before) {
				data, _ := librarium.Marshal(got)
				t.Fatalf("the amended rite differs:\n%s", data)
			}
		})
	}
}

// TestWizard_TheExaltedAuspexFollowsItsSubject: the auspex of an exalted
// machine-spirit is exalted upon its page, and an exalted auspex and relics
// are laid down with the exaltation of their subject, for they are borne
// by an exalted subject alone.
func TestWizard_TheExaltedAuspexFollowsItsSubject(t *testing.T) {
	h := newHarness(t)
	h.writeHall("scripts/root", "#!/bin/sh\n", 0o755)
	h.writeRite("root", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "root", "designation": "machine-spirit", "exalted": true, "auspex": {"scan": "cat /proc/sys/kernel/sysrq"},
	   "enact": {"*": "scripts/root {{omen}}"}, "relics": ["scripts/root"]}]}`)
	h.keys("r", "/")
	h.typeText("root")
	h.keys("enter")
	// amend turns the field of the subject's page that shows label.
	amend := func(label, field string) *librarium.MachineSpirit {
		t.Helper()
		h.keys("e", "ctrl+s", "enter")
		for range 6 {
			if strings.Contains(h.screen(), label) {
				break
			}
			h.keys("ctrl+s")
		}
		h.focusOn(field)
		h.keys("space")
		var seen strings.Builder
		h.throughSubject("The subjects of the rite", &seen)
		h.keys("tab", "tab")
		h.mustShow("is pure")
		h.keys("enter")
		h.mustShow("is amended")
		return h.loadRite("root").Subjects[0].Matter.(*librarium.MachineSpirit)
	}
	ms := amend("Exalted auspex", "privileged")
	if !ms.Exalted || !ms.Auspex.Exalted || len(ms.Relics) != 1 {
		t.Fatalf("the auspex was not exalted: %+v %+v", ms, ms.Auspex)
	}
	ms = amend("Exalted subject", "exalted")
	if ms.Exalted || ms.Auspex.Exalted || ms.Relics != nil {
		t.Fatalf("the exaltation was not laid down whole: %+v %+v", ms, ms.Auspex)
	}
}
