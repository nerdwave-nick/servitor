package librarium

import (
	"io/fs"
	"testing"
	"time"
)

// gaming is the owner's gaming rite in pattern Mark II: three exalted
// machine-spirits, read by scan and by trials.
const gaming = `{
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

// dnd is the do-not-disturb rite of the codex: one machine-spirit and one
// observance spoken after it, only into "on".
const dnd = `{
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

// visageII is a rite of every built-in designation and every key of a
// subject and an observance.
const visageII = `{
  "$schema": "https://example.org/rite.json",
  "pattern": "Mark II", // the Mark of its form
  "purpose": "The visage of the machine",
  "aspects": ["default", "porpl"],
  "inscriptions": {
    "reason": {"purpose": "why"},
    "mode": {"mandatory": true, "decrees": {"*": "plain"}},
  },
  "tongue": "zsh",
  "subjects": [
    {"name": "util", "designation": "sanctum", "purpose": "the niri visage",
     "vessel": "~/.config/niri/util.kdl", "ward": "visage", "glyph": "/*", "closing-glyph": "*/", "consecrate": true,
     "omens": {"default": "", "porpl": ["a {", "  b", "}"]}, "introit": ["transition"]},
    {"name": "conf", "designation": "transcription", "vessel": "~/.config/x/{{rite.name}}.conf",
     "omens": {"default": null, "*": {"tome": "snippets/{{aspect}}.conf", "illuminate": true}}, "seal": "0600", "zeal": true},
    {"name": "current", "designation": "tether", "tether": "~/.local/share/nfluff/current-theme",
     "omens": {"default": null, "*": "~/.config/nfluff/themes/{{aspect}}"}, "force": true,
     "recessional": ["hooks", "recite"]},
    {"name": "bar", "designation": "machine-spirit",
     "auspex": {"scan": "cat ~/.cache/bar", "trials": {"porpl": "test -e ~/.cache/{{omen}}", "*": "false"}},
     "enact": {"default": "bar-set default", "*": "bar-set {{omen}} {{inscription.mode}}"},
     "tongue": "bash", "patience": "5s"}
  ],
  "observances": [
    {"name": "transition", "designation": "incantation", "words": "niri msg action do-screen-transition -d 200",
     "tongue": "sh", "patience": "1s", "reversion": "", "from": ["default"]},
    {"name": "hooks", "designation": "incantation", "words": "source ~/.config/nfluff/themes/{{aspect}}/hooks",
     "reversion": "source ~/.config/nfluff/themes/{{former}}/hooks", "to": ["porpl", "default"]},
    {"name": "recite", "designation": "litany", "scroll": "scripts/change-theme",
     "offerings": ["~/.config/nfluff/themes/{{aspect}}", "{{inscription.reason}}", ""]}
  ],
}`

// screenshot is the edict of the codex in the designation form.
const screenshot = `{
  "pattern": "Mark II",
  "purpose": "Capture the visage of the machine",
  "edict": true,
  "inscriptions": {"region": {"purpose": "\"chosen\" or \"whole\"", "decrees": "chosen"}},
  "vox-cast": ["success"],
  "liturgy": [
    {"designation": "incantation", "patience": "2m",
     "words": "grim -g \"$(slurp)\" ~/pictures/visage.png",
     "reversion": "rm -f ~/pictures/visage.png"},
    {"designation": "litany", "scroll": "scripts/copy", "offerings": ["{{rite.name}}"], "tongue": "sh"},
    {"designation": "incantation", "words": "systemctl reboot", "exalted": true, "relics": []}
  ]
}`

func TestParse_MarkIIRitesYieldEveryField(t *testing.T) {
	seal := fs.FileMode(0o600)
	cases := []struct {
		name, scripture string
		want            *Rite
	}{
		{"gaming", gaming, &Rite{Purpose: "enable or disable gaming optimization",
			Aspects: []string{"on", "off"}, VoxCasts: []string{VoxProgress, VoxSuccess},
			Subjects: []Subject{
				{Name: "smt", Matter: &MachineSpirit{Exalted: true,
					Omens:  PerAspect(Entry[string]{"on", "off"}, Entry[string]{"*", "on"}),
					Auspex: &SpiritAuspex{Scan: "cat /sys/devices/system/cpu/smt/control"},
					Enact:  PerAspect(Entry[string]{"*", "echo {{omen}} > /sys/devices/system/cpu/smt/control"})}},
				{Name: "split-lock", Matter: &MachineSpirit{Exalted: true,
					Omens:  PerAspect(Entry[string]{"on", "0"}, Entry[string]{"*", "1"}),
					Auspex: &SpiritAuspex{Scan: "sysctl -n kernel.split_lock_mitigate"},
					Enact:  PerAspect(Entry[string]{"*", "sysctl -wq kernel.split_lock_mitigate={{omen}}"})}},
				{Name: "scheduler", Matter: &MachineSpirit{Exalted: true,
					Omens: PerAspect(Entry[string]{"on", "cosmos"}, Entry[string]{"*", "eevdf"}),
					Auspex: &SpiritAuspex{Trials: PerAspect(Entry[string]{"cosmos", "systemctl is-active -q scx-cosmos"}),
						Patience: time.Second},
					Enact: PerAspect(Entry[string]{"cosmos", "systemd-run --unit=scx-cosmos --collect scx_cosmos -p 0 -c 0"},
						Entry[string]{"*", "systemctl stop scx-cosmos"})}},
			}}},
		{"dnd", dnd, &Rite{Purpose: "silence the heralds of the machine",
			Aspects: []string{"on", "off"}, VoxCasts: []string{VoxSuccess},
			Subjects: []Subject{{Name: "mako", Recessional: []string{"herald-silence"}, Matter: &MachineSpirit{
				Omens:  PerAspect(Entry[string]{"on", "do-not-disturb"}, Entry[string]{"*", "default"}),
				Auspex: &SpiritAuspex{Scan: "makoctl mode"},
				Enact:  PerAspect(Entry[string]{"*", `makoctl mode -s "$SERVITOR_OMEN"`})}}},
			Observances: []Observance{{Name: "herald-silence", To: []string{"on"},
				Speech: &Incantation{Command: Uniform("notify-send -t 2000 'The heralds fall silent'")}}},
		}},
		{"visage", visageII, &Rite{Schema: "https://example.org/rite.json", Purpose: "The visage of the machine",
			Aspects: []string{"default", "porpl"},
			Inscriptions: []Inscription{{Key: "reason", Purpose: "why"},
				{Key: "mode", Mandatory: true, Decrees: PerAspect(Entry[string]{"*", "plain"})}},
			Tongue: "zsh",
			Subjects: []Subject{
				{Name: "util", Purpose: "the niri visage", Introit: []string{"transition"}, Matter: &Sanctum{
					Vessel: "~/.config/niri/util.kdl", Ward: "visage", Glyph: "/*", ClosingGlyph: "*/", Consecrate: true,
					Scripture: PerAspect(Entry[Scripture]{"default", Inline("")}, Entry[Scripture]{"porpl", Inline("a {\n  b\n}")})}},
				{Name: "conf", Matter: &Transcription{Vessel: "~/.config/x/{{rite.name}}.conf", Seal: &seal, Zeal: true,
					Scripture: PerAspect(Entry[Scripture]{"default", Scripture{Null: true}},
						Entry[Scripture]{"*", Tome("snippets/{{aspect}}.conf", true)})}},
				{Name: "current", Recessional: []string{"hooks", "recite"}, Matter: &Tether{
					Name: "~/.local/share/nfluff/current-theme", Zeal: true,
					Anchor: PerAspect(Entry[Anchor]{"default", Anchor{Null: true}},
						Entry[Anchor]{"*", Anchor{Path: "~/.config/nfluff/themes/{{aspect}}"}})}},
				{Name: "bar", Matter: &MachineSpirit{Tongue: "bash", Patience: 5 * time.Second,
					Auspex: &SpiritAuspex{Scan: "cat ~/.cache/bar",
						Trials: PerAspect(Entry[string]{"porpl", "test -e ~/.cache/{{omen}}"}, Entry[string]{"*", "false"})},
					Enact: PerAspect(Entry[string]{"default", "bar-set default"}, Entry[string]{"*", "bar-set {{omen}} {{inscription.mode}}"})}},
			},
			Observances: []Observance{
				{Name: "transition", From: []string{"default"}, Speech: &Incantation{
					Command:   Uniform("niri msg action do-screen-transition -d 200"),
					Utterance: Utterance{Tongue: "sh", Patience: time.Second, Reversion: Uniform("")}}},
				{Name: "hooks", To: []string{"porpl", "default"}, Speech: &Incantation{
					Command:   Uniform("source ~/.config/nfluff/themes/{{aspect}}/hooks"),
					Utterance: Utterance{Reversion: Uniform("source ~/.config/nfluff/themes/{{former}}/hooks")}}},
				{Name: "recite", Speech: &Litany{Scroll: Uniform("scripts/change-theme"),
					Offerings: []AspectMap[string]{Uniform("~/.config/nfluff/themes/{{aspect}}"), Uniform("{{inscription.reason}}"), Uniform("")}}},
			},
		}},
		{"screenshot", screenshot, &Rite{Edict: true, Purpose: "Capture the visage of the machine",
			Inscriptions: []Inscription{{Key: "region", Purpose: `"chosen" or "whole"`, Decrees: Uniform("chosen")}},
			VoxCasts:     []string{VoxSuccess},
			Liturgy: []Step{
				&Incantation{Command: Uniform(`grim -g "$(slurp)" ~/pictures/visage.png`),
					Utterance: Utterance{Patience: 2 * time.Minute, Reversion: Uniform("rm -f ~/pictures/visage.png")}},
				&Litany{Scroll: Uniform("scripts/copy"), Offerings: []AspectMap[string]{Uniform("{{rite.name}}")},
					Utterance: Utterance{Tongue: "sh"}},
				&Incantation{Command: Uniform("systemctl reboot"), Utterance: Utterance{Exalted: true, Relics: []string{}}},
			}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := "/lib/rites/" + c.name + ".json"
			r, fs := parse(c.name, path, []byte(c.scripture))
			if r == nil || len(fs) != 0 {
				t.Fatalf("rite %v, findings:\n%v", r, fs)
			}
			c.want.Name, c.want.Path = c.name, path
			if !r.Equal(c.want) {
				t.Fatalf("got  %#v\nwant %#v", r, c.want)
			}
			if r.Path != path || r.Locate("/subjects/0/name") == (Position{}) {
				t.Fatalf("the rite does not know where it was read: %q", r.Path)
			}
		})
	}
}

func TestRite_SubjectsAndObservancesAreFoundByName(t *testing.T) {
	r, fs := parse("visage", "/lib/rites/visage.json", []byte(visageII))
	if len(fs) != 0 {
		t.Fatalf("findings:\n%v", fs)
	}
	s, ok := r.Subject("current")
	if !ok || s.Designation() != DesignationTether || s.Exalted() {
		t.Fatalf("subject %+v, %v", s, ok)
	}
	o, ok := r.Observance("recite")
	if !ok || o.Designation() != DesignationLitany || o.Exalted() {
		t.Fatalf("observance %+v, %v", o, ok)
	}
	if _, ok := r.Subject("nonesuch"); ok {
		t.Fatal("found a subject never declared")
	}
	if _, ok := r.Observance("nonesuch"); ok {
		t.Fatal("found an observance never declared")
	}
	g, _ := parse("gaming", "/lib/rites/gaming.json", []byte(gaming))
	if s, _ := g.Subject("smt"); !s.Exalted() || s.Designation() != DesignationMachineSpirit {
		t.Fatalf("smt %+v", s)
	}
}

// TestParse_MarkIIRiteMayBeBareOfOptions: a rite of one subject, and an
// edict of nothing, bear no key they need not.
func TestParse_MarkIIRiteMayBeBareOfOptions(t *testing.T) {
	cases := map[string]string{
		"spirit named by its aspects": `{"pattern": "Mark II", "aspects": ["on", "off"],
			"subjects": [{"name": "x", "designation": "machine-spirit", "auspex": {"scan": "cat x"}, "enact": {"*": "echo {{omen}} > x"}}]}`,
		"spirit without auspex": `{"pattern": "Mark II", "aspects": ["on"], "subjects": [{"name": "x", "designation": "machine-spirit",
			"enact": {"on": "a"}}]}`,
		"empty vox-cast":    `{"pattern": "Mark II", "edict": true, "vox-cast": []}`,
		"bare edict":        `{"pattern": "Mark II", "edict": true}`,
		"edict false":       `{"pattern": "Mark II", "edict": false, "aspects": ["on"], "subjects": [{"name": "t", "designation": "tether", "tether": "t", "omens": {"*": null}}]}`,
		"omens of one word": `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [{"name": "x", "designation": "machine-spirit", "omens": {"*": "{{aspect}}-mode"}, "enact": {"on-mode": "a", "off-mode": "b"}}]}`,
	}
	for name, scripture := range cases {
		t.Run(name, func(t *testing.T) {
			if r, fs := parse("rite", "/r.json", []byte(scripture)); r == nil || r.markI || fs.Heretical() {
				t.Fatalf("rite %v, findings:\n%v", r, fs)
			}
		})
	}
}
