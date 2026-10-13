package exalted

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
	sanctums "github.com/nerdwave-nick/servitor/internal/sanctum"
)

// privilegedShrine is the world of a trial of a state rite of pattern Mark
// II whose subjects only the root can read, "hallow": the machine-spirit
// nft (place 1), read by a privileged scan that prints what <hall>/nft
// holds, whispers on its stderr and tells it was spoken in <hall>/marker,
// set by enact words that write the omen to <hall>/nft unless
// <hall>/stubborn stands; msr (place 2), which declares no omens and is
// read by a privileged trial of "on" alone; the transcription conf (place
// 3) and the sanctum lock (place 4), whose scriptures hold the secret; the
// tether theme (place 5); and smt (place 6), whose auspex is common.
func privilegedShrine(t *testing.T) (*shrine, string) {
	t.Helper()
	bash, hall := tongue(t), t.TempDir()
	at := func(name string) string { return filepath.Join(hall, name) }
	enact := func(file string, omens ...string) map[string]string {
		m := map[string]string{}
		for _, omen := range omens {
			m[omen] = "[ -e " + at("stubborn") + " ] || echo " + omen + " > " + at(file)
		}
		return m
	}
	plan := func(aspect, nft string) sanction.Plan {
		return sanction.Plan{Aspect: aspect, Inscriptions: map[string]string{}, Observances: []sanction.Verse{},
			Subjects: []sanction.Verse{
				{Subject: "nft", Place: 0, Omen: nft, Enact: enact("nft", "gaming", "none"), Tongue: bash,
					Patience: patience, Relics: []sanction.Relic{}, Auspex: &sanction.Auspex{
						Scan:     "echo read >> " + at("marker") + "; cat " + at("nft") + "; echo whisper >&2",
						Trials:   map[string]string{},
						Patience: patience}},
				{Subject: "msr", Place: 1, Omen: aspect, Enact: enact("msr", "on", "off"), Tongue: bash,
					Patience: patience, Relics: []sanction.Relic{}, Auspex: &sanction.Auspex{
						Trials:   map[string]string{"on": "grep -qx on " + at("msr")},
						Patience: patience, NamesAspects: true}},
				{Subject: "conf", Place: 2, Kind: librarium.KindTranscription, Target: "/etc/firewall.conf",
					Scripture: []byte("rules = " + aspect + " # " + secret + "\n")},
				{Subject: "lock", Place: 3, Kind: librarium.KindSanctum, Target: "/etc/hallow.conf", Ward: "hallow",
					Glyph: "#", Scripture: []byte("mode = " + aspect + " " + secret)},
				{Subject: "theme", Place: 4, Kind: librarium.KindTether, Target: "/etc/hallow-theme",
					Anchor: "/usr/share/hallow/" + aspect},
				{Subject: "smt", Place: 5, Omen: aspect, Enact: enact("smt", "on", "off"), Tongue: bash,
					Patience: patience, Relics: []sanction.Relic{}},
			}}
	}
	u := &sanction.Unit{Pattern: sanction.PatternMarkII, UID: me, Rite: "hallow",
		Environment: sanction.Environment{Pattern: sanction.PatternMarkII, Keys: []string{"SERVITOR_ASPECT",
			"SERVITOR_FORMER_ASPECT", "SERVITOR_INSCRIPTION_WHO", "SERVITOR_OMEN", "SERVITOR_RITE"}},
		Aspects: []sanction.Plan{plan("on", "gaming"), plan("off", "none")}}
	return speakingShrine(t, u), hall
}

func reading(names ...string) Request { return Request{Read: names} }

// sightsOf tells each sight as "name omens…" or "name unchanged/changing".
func sightsOf(r Reply) []string {
	var out []string
	for _, s := range r.Sights {
		told := strings.Join(s.Omens, ",")
		switch {
		case s.Omens != nil:
		case s.Unchanged:
			told = "unchanged"
		default:
			told = "-"
		}
		out = append(out, s.Subject+" "+told)
	}
	return out
}

// TestReadII_AnswersTheFittingOmensAndNeverTheWords: a read names the
// privileged subjects and the built-in subjects the invoker cannot read,
// all spoken at once before the first item. The exalted servitor matches
// what a privileged auspex printed against the declared omens itself and
// answers with those that fit, or none (mute), never the words printed;
// of a built-in subject it tells only whether it stands as the plan
// demands, never its scripture.
func TestReadII_AnswersTheFittingOmensAndNeverTheWords(t *testing.T) {
	sh, hall := privilegedShrine(t)
	must(t, os.WriteFile(filepath.Join(hall, "nft"), []byte("gaming\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(hall, "msr"), []byte("on\n"), 0o644))
	sh.scroll("/etc/firewall.conf", "rules = on # "+secret+"\n", 0o644)
	sealed, err := sanctums.Upsert(original, sanctums.Marker{Glyph: "#", Ward: "hallow"}, "on", "mode = on "+secret)
	must(t, err)
	sh.scroll("/etc/hallow.conf", sealed, 0o600)
	inv := sh.accepted("on")
	r := inv.say(reading("nft", "msr", "conf", "lock", "theme"))
	want := []string{"nft gaming", "msr on", "conf unchanged", "lock unchanged", "theme -"}
	if r.To != "read" || r.Verdict != Beheld || !slices.Equal(sightsOf(r), want) {
		t.Fatalf("read %+v, want %q", r, want)
	}
	if got := lines(t, filepath.Join(hall, "marker")); !slices.Equal(got, []string{"read"}) {
		t.Errorf("the scan was spoken %q", got)
	}
	inv.say(Request{Finish: new(true)})
	inv.ended()

	for what, c := range map[string]struct {
		nft, msr, conf string
		want           []string
	}{
		"words no omen fits": {nft: secret + "\n", msr: "off\n", conf: "rules = off # " + secret + "\n",
			want: []string{"nft -", "msr off", "conf -"}},
		"two omens at once": {nft: "gaming none\n", msr: "\n", conf: "rules = on # " + secret,
			want: []string{"nft -", "msr off", "conf -"}},
		"nothing printed": {nft: "", msr: "on\n", conf: "",
			want: []string{"nft -", "msr on", "conf -"}},
	} {
		t.Run(what, func(t *testing.T) {
			sh, hall := privilegedShrine(t)
			must(t, os.WriteFile(filepath.Join(hall, "nft"), []byte(c.nft), 0o644))
			must(t, os.WriteFile(filepath.Join(hall, "msr"), []byte(c.msr), 0o644))
			if c.conf != "" {
				sh.scroll("/etc/firewall.conf", c.conf, 0o644)
			}
			inv := sh.accepted("on")
			if r := inv.say(reading("nft", "msr", "conf")); r.Verdict != Beheld || !slices.Equal(sightsOf(r), c.want) {
				t.Fatalf("read %+v, want %q", r, c.want)
			}
			inv.say(Request{Finish: new(true)})
			inv.ended()
			for _, words := range []string{secret, "whisper", "gaming none"} {
				if strings.Contains(inv.heard.String(), words) {
					t.Errorf("the channel carried %q: %s", words, inv.heard)
				}
			}
			for _, e := range sh.journal.all() {
				if strings.Contains(e.String(), secret) {
					t.Errorf("the journal keeps the secret: %s", e)
				}
			}
		})
	}
}

// TestReadII_AnImpatientAuspexIsMute: a privileged auspex that outlasts its
// patience is slain and mute, and the read is answered at once.
func TestReadII_AnImpatientAuspexIsMute(t *testing.T) {
	for what, patient := range map[string]func(a *sanction.Auspex){
		"a scan":  func(a *sanction.Auspex) { a.Scan = "sleep 30; echo gaming" },
		"a trial": func(a *sanction.Auspex) { a.Scan, a.Trials = "", map[string]string{"gaming": "sleep 30"} },
	} {
		t.Run(what, func(t *testing.T) {
			sh, _ := privilegedShrine(t)
			for i := range sh.unit.Aspects {
				a := sh.unit.Aspects[i].Subjects[0].Auspex
				patient(a)
				a.Patience = 200 * time.Millisecond
			}
			sh.lay(sh.cfg.Store, sh.unit)
			inv := sh.accepted("on")
			began := time.Now()
			if r := inv.say(reading("nft")); r.Verdict != Beheld || !slices.Equal(sightsOf(r), []string{"nft -"}) {
				t.Fatalf("read %+v", r)
			}
			if d := time.Since(began); d > 5*time.Second {
				t.Errorf("the read was answered after %v", d)
			}
			inv.say(Request{Finish: new(true)})
			inv.ended()
		})
	}
}

// TestReadII_IsSpokenOnceBeforeTheFirstItem: a read is heard only from the
// invoker of a state rite, once, before any item is performed, and names
// only privileged and built-in subjects of the plan, each once; any other
// read is refused and ends the session.
func TestReadII_IsSpokenOnceBeforeTheFirstItem(t *testing.T) {
	for what, c := range map[string]struct {
		before []Request
		read   Request
		lament string
	}{
		"after an item":             {before: []Request{{Perform: subject("theme")}}, read: reading("nft"), lament: "before"},
		"twice":                     {before: []Request{reading("nft")}, read: reading("msr"), lament: "once"},
		"a common auspex":           {read: reading("nft", "smt"), lament: `"smt"`},
		"a subject no plan holds":   {read: reading("ghost"), lament: `"ghost"`},
		"a subject named twice":     {read: reading("nft", "nft"), lament: "twice"},
		"no subject":                {read: Request{Read: []string{}}, lament: "names"},
		"with another word at once": {read: Request{Read: []string{"nft"}, Finish: new(true)}, lament: "exactly one"},
	} {
		t.Run(what, func(t *testing.T) {
			sh, _ := privilegedShrine(t)
			inv := sh.accepted("on")
			for _, req := range c.before {
				if r := inv.say(req); r.Verdict == Refused {
					t.Fatalf("%+v: %+v", req, r)
				}
			}
			r := inv.say(c.read)
			if r.Verdict != Refused || !strings.Contains(r.Lament, c.lament) {
				t.Fatalf("heard %+v, want a refusal naming %q", r, c.lament)
			}
			if m := inv.ended(); m != 1 {
				t.Errorf("ended with the death-mark %d, want 1", m)
			}
			sh.holds(map[string]string{"/etc/hallow-theme": "absent"})
		})
	}
	sh := newShrine(t, nil)
	inv := sh.accepted("on")
	if r := inv.say(reading("verse")); r.Verdict != Refused || r.To != "read" {
		t.Fatalf("a read of a liturgy was heard: %+v", r)
	}
	inv.ended()
}

// TestPerformII_TheExaltedServitorVerifiesAPrivilegedSubject: a privileged
// subject set by the exalted servitor is read anew by it, and the perform
// triumphs only when it shows the omen commanded, answering the omens it
// fits; otherwise it falls, naming the omen found and the omen commanded,
// yet stands performed. Its restoration is verified alike: a revert that
// leaves it showing another omen falters.
func TestPerformII_TheExaltedServitorVerifiesAPrivilegedSubject(t *testing.T) {
	sh, hall := privilegedShrine(t)
	nft := filepath.Join(hall, "nft")
	must(t, os.WriteFile(nft, []byte("none\n"), 0o644))
	inv := sh.accepted("on")
	r := inv.say(Request{Perform: spirit("nft", "none")})
	if r.Verdict != Triumph || !r.Changed || !slices.Equal(r.Omens, []string{"gaming"}) {
		t.Fatalf("perform nft: %+v", r)
	}
	if r := inv.say(Request{Revert: spirit("nft", "")}); r.Verdict != Reverted || !slices.Equal(r.Omens, []string{"none"}) {
		t.Fatalf("revert nft: %+v", r)
	}
	inv.say(Request{Abort: new(true)})
	inv.ended()

	// The enact words take no effect: the perform falls, and its revert
	// finds the omen it restores.
	sh, hall = privilegedShrine(t)
	must(t, os.WriteFile(filepath.Join(hall, "nft"), []byte("none\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(hall, "stubborn"), nil, 0o644))
	inv = sh.accepted("on")
	r = inv.say(Request{Perform: spirit("nft", "none")})
	if r.Verdict != Fell || !r.Changed || !strings.Contains(r.Lament, `the subject "nft" shows the omen "none"; `+
		`"gaming" was commanded`) || r.Omens != nil {
		t.Fatalf("perform nft: %+v", r)
	}
	if r := inv.say(Request{Revert: spirit("nft", "")}); r.Verdict != Reverted {
		t.Fatalf("revert nft: %+v", r)
	}
	inv.say(Request{Abort: new(true)})
	inv.ended()

	// The restoration takes no effect: the revert falters, naming what the
	// subject shows.
	sh, hall = privilegedShrine(t)
	must(t, os.WriteFile(filepath.Join(hall, "nft"), []byte("none\n"), 0o644))
	inv = sh.accepted("on")
	if r := inv.say(Request{Perform: spirit("nft", "none")}); r.Verdict != Triumph {
		t.Fatalf("perform nft: %+v", r)
	}
	must(t, os.WriteFile(filepath.Join(hall, "stubborn"), nil, 0o644))
	r = inv.say(Request{Revert: spirit("nft", "")})
	if r.Verdict != Faltered || !strings.Contains(r.Lament, `shows the omen "gaming"; "none" was commanded`) {
		t.Fatalf("revert nft: %+v", r)
	}
	inv.say(Request{Abort: new(true)})
	inv.ended()
	if strings.Contains(inv.heard.String(), "whisper") {
		t.Errorf("the channel carried what the scan whispered: %s", inv.heard)
	}
}

// TestPerformII_TheEndOfTheChannelVerifiesTheRestoration: when the invoker
// falls, the exalted servitor restores a privileged subject by itself and
// verifies it; one that does not show the omen restored falters the
// session.
func TestPerformII_TheEndOfTheChannelVerifiesTheRestoration(t *testing.T) {
	for _, stubborn := range []bool{false, true} {
		sh, hall := privilegedShrine(t)
		must(t, os.WriteFile(filepath.Join(hall, "nft"), []byte("none\n"), 0o644))
		inv := sh.accepted("on")
		if r := inv.say(Request{Perform: spirit("nft", "none")}); r.Verdict != Triumph {
			t.Fatalf("perform nft: %+v", r)
		}
		if stubborn {
			must(t, os.WriteFile(filepath.Join(hall, "stubborn"), nil, 0o644))
		}
		must(t, inv.conn.Close())
		want := map[bool]int{false: 1, true: 3}[stubborn]
		if m := inv.ended(); m != want {
			t.Errorf("stubborn %v: ended with the death-mark %d, want %d", stubborn, m, want)
		}
	}
}

// TestSanctionII_APrivilegedAuspexMustBeSpeakable: a unit whose privileged
// auspex holds neither scan nor trials, a trial of an omen its spirit does
// not declare, or no patience is refused, and nothing is written; a true
// one is laid.
func TestSanctionII_APrivilegedAuspexMustBeSpeakable(t *testing.T) {
	unit := func(t *testing.T) (*shrine, *sanction.Unit) {
		state, _ := privilegedShrine(t)
		sh, _ := keepingShrine(t)
		sh.cfg.Voice, sh.cfg.Weigh = state.cfg.Voice, sh.speech().Weigh
		return sh, state.unit
	}
	sh, u := unit(t)
	inv := sh.sanctionFor(u, "")
	if r := inv.answer(); r.Verdict != Sanctioned {
		t.Fatalf("answered %+v", r)
	}
	inv.ended()
	for what, c := range map[string]struct {
		mar    func(a *sanction.Auspex)
		lament string
	}{
		"neither scan nor trials": {func(a *sanction.Auspex) { a.Scan = "" }, "neither"},
		"an undeclared omen tried": {func(a *sanction.Auspex) { a.Trials = map[string]string{"auto": "true"} },
			`"auto"`},
		"no patience": {func(a *sanction.Auspex) { a.Patience = 0 }, "patience"},
	} {
		t.Run(what, func(t *testing.T) {
			sh, u := unit(t)
			c.mar(u.Aspects[1].Subjects[0].Auspex)
			inv := sh.sanctionFor(u, "")
			if r := inv.answer(); r.Verdict != Refused || !strings.Contains(r.Lament, c.lament) {
				t.Fatalf("answered %+v, want a refusal naming %q", r, c.lament)
			}
			if m := inv.ended(); m != 2 {
				t.Errorf("refused with the death-mark %d", m)
			}
			if _, err := os.Lstat(sh.store().Path(u.Rite)); err == nil {
				t.Error("a sanction was laid")
			}
		})
	}
}

// TestPerformII_TrialsProveWhatTheScanReads: a privileged auspex holding a
// scan and trials is read by its scan, yet what was set or restored is
// proven by its trials alone.
func TestPerformII_TrialsProveWhatTheScanReads(t *testing.T) {
	sh, hall := privilegedShrine(t)
	for i := range sh.unit.Aspects {
		sh.unit.Aspects[i].Subjects[0].Auspex.Trials = map[string]string{"gaming": "false"}
	}
	sh.lay(sh.cfg.Store, sh.unit)
	must(t, os.WriteFile(filepath.Join(hall, "nft"), []byte("gaming\n"), 0o644))
	inv := sh.accepted("on")
	if r := inv.say(reading("nft")); !slices.Equal(sightsOf(r), []string{"nft gaming"}) {
		t.Fatalf("read %+v", r)
	}
	r := inv.say(Request{Perform: spirit("nft", "none")})
	if r.Verdict != Fell || !strings.Contains(r.Lament, `shows the omen "none"; "gaming" was commanded`) {
		t.Fatalf("perform nft: %+v", r)
	}
	if r := inv.say(Request{Revert: spirit("nft", "")}); r.Verdict != Reverted || !slices.Equal(r.Omens, []string{"none"}) {
		t.Fatalf("revert nft: %+v", r)
	}
	inv.say(Request{Abort: new(true)})
	inv.ended()
}
