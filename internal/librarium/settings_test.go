package librarium

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const settingsPath = "/lib/servitor.json"

func TestParseSettings_ReadsEveryKeyOfJSONC(t *testing.T) {
	scripture := `// the standing orders of this servitor
{
  "pattern": "Mark I",
  "tongue": "zsh",           // spoken by incantations and litanies
  "chronicle": "~/chronicle.jsonl",
  "vox": "off",
  "patience": "1m30s",       /* for the slow machine spirits */
}`
	s, fs := ParseSettings(settingsPath, []byte(scripture))
	if len(fs) != 0 {
		t.Fatalf("findings:\n%v", fs)
	}
	want := Settings{Path: settingsPath, Tongue: "zsh", Chronicle: "~/chronicle.jsonl", Vox: VoxOff, Patience: 90 * time.Second}
	if *s != want {
		t.Fatalf("settings %+v, want %+v", *s, want)
	}
}

func TestParseSettings_PatternAloneLeavesEverythingUnwritten(t *testing.T) {
	s, fs := ParseSettings(settingsPath, []byte(`{"pattern": "Mark I"}`))
	if len(fs) != 0 || *s != (Settings{Path: settingsPath}) {
		t.Fatalf("settings %+v, findings %v", *s, fs)
	}
}

var settingsHeresies = []struct {
	name, scripture, message, at string
}{
	{"syntax", `{"pattern": "Mark I",, }`, "malformed beyond reading", `, }`},
	{"no object", `["Mark I"]`, "the settings must be written as one object", `[`},
	{"pattern missing", `{"tongue": "zsh"}`, `bears no "pattern"`, `{`},
	{"pattern malformed", `{"pattern": "Mark 1"}`, "numerals of the old tongue", `"Mark 1"`},
	{"pattern no string", `{"pattern": 1}`, "numerals of the old tongue", `1}`},
	{"pattern unknown mark", `{"pattern": "Mark II"}`, `forged to read only "Mark I"`, `"Mark II"`},
	{"unknown key", `{"pattern": "Mark I", "shell": "zsh"}`,
		`the key "shell" is not written in the codex for the settings; strike it, or write one of the keys the codex knows there: "pattern", "tongue", "chronicle", "vox", "patience"`, `"shell"`},
	{"key twice", `{"pattern": "Mark I", "vox": "off", "vox": "off"}`, `the key "vox" is written twice`, `"vox": "off"}`},
	{"patience unreadable", `{"pattern": "Mark I", "patience": "soon"}`, `"soon" is no measure of patience the servitor understands; the "patience" of the settings`, `"soon"`},
	{"patience nothing", `{"pattern": "Mark I", "patience": "0s"}`, `"0s" is no measure of patience`, `"0s"`},
	{"patience negative", `{"pattern": "Mark I", "patience": "-5s"}`, `"-5s" is no measure of patience`, `"-5s"`},
	{"patience number", `{"pattern": "Mark I", "patience": 30}`, `the "patience" of the settings must be written as a string`, `30}`},
	{"vox unknown", `{"pattern": "Mark I", "vox": "loud"}`, `"loud" is no vox the codex knows`, `"loud"`},
	{"vox boolean", `{"pattern": "Mark I", "vox": false}`, `the "vox" of the settings must be written as a string`, `false}`},
	{"tongue empty", `{"pattern": "Mark I", "tongue": " "}`, `the "tongue" of the settings may not be empty`, `" "`},
	{"chronicle empty", `{"pattern": "Mark I", "chronicle": ""}`, `the "chronicle" of the settings may not be empty`, `""`},
	{"chronicle list", `{"pattern": "Mark I", "chronicle": ["a"]}`, `the "chronicle" of the settings must be written as a string`, `["a"]`},
}

func TestParseSettings_DenouncesWithPosition(t *testing.T) {
	for _, c := range settingsHeresies {
		t.Run(c.name, func(t *testing.T) {
			_, fs := ParseSettings(settingsPath, []byte(c.scripture))
			want := posOf(t, c.scripture, c.at, 0)
			for _, f := range fs {
				if strings.Contains(f.Message, c.message) {
					if f.Severity != Heresy || f.Position != want || f.Scripture != settingsPath || f.Rite != "" {
						t.Fatalf("finding %v; want heresy at %v", f, want)
					}
					if got := f.String(); !strings.HasPrefix(got, settingsPath+":") {
						t.Fatalf("finding renders as %q", got)
					}
					return
				}
			}
			t.Fatalf("no finding containing %q in\n%v", c.message, fs)
		})
	}
}

func TestParseSettings_DenunciationsSpeakGrimdark(t *testing.T) {
	for _, c := range settingsHeresies {
		_, fs := ParseSettings(settingsPath, []byte(c.scripture))
		for _, f := range fs {
			if m := plainGlosses.FindString(f.Message); m != "" {
				t.Errorf("%s: %q speaks the plain word %q", c.name, f.Message, m)
			}
		}
	}
}

func TestParseSettings_HereticalOrdersFallBackToDefaults(t *testing.T) {
	s, fs := ParseSettings(settingsPath, []byte(`{"pattern": "Mark I", "tongue": "fish", "vox": "loud", "patience": "soon"}`))
	if len(fs) != 2 || !fs.Heretical() {
		t.Fatalf("findings: %v", fs)
	}
	if *s != (Settings{Path: settingsPath, Tongue: "fish"}) {
		t.Fatalf("settings %+v: heretical orders must stay unwritten, sound ones kept", *s)
	}
	if s, _ := ParseSettings(settingsPath, []byte(`{"pattern": "Mark II", "tongue": "fish"}`)); *s != (Settings{Path: settingsPath}) {
		t.Fatalf("settings %+v: scripture of an unknown pattern must not be heeded at all", *s)
	}
}

func TestLoadSettings_AbsentScriptureYieldsDefaults(t *testing.T) {
	dir := t.TempDir()
	s, fs := LoadSettings(dir)
	if len(fs) != 0 || *s != (Settings{Path: filepath.Join(dir, SettingsName)}) {
		t.Fatalf("settings %+v, findings %v", *s, fs)
	}
	o := s.Resolve(Runes{}, environ(map[string]string{"HOME": "/home/adept"}))
	want := Orders{Tongue: "bash", Chronicle: "/home/adept/.local/state/servitor/chronicle.jsonl", Vox: VoxNotifySend, Patience: 30 * time.Second}
	if o != want {
		t.Fatalf("orders %+v, want %+v", o, want)
	}
}

func TestLoadSettings_UnreadableScriptureIsHeresy(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, SettingsName), 0o755); err != nil {
		t.Fatal(err)
	}
	s, fs := LoadSettings(dir)
	if len(fs) != 1 || fs[0].Severity != Heresy || fs[0].Scripture != filepath.Join(dir, SettingsName) || fs[0].Line != 1 {
		t.Fatalf("findings: %v", fs)
	}
	if *s != (Settings{Path: filepath.Join(dir, SettingsName)}) {
		t.Fatalf("settings %+v", *s)
	}
}

func TestLoad_CarriesTheSettingsOfItsLibrarium(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	writeScripture(t, a, "rites/r.json", sanctumRite("~/a.kdl", ""))
	writeScripture(t, a, SettingsName, `{"pattern": "Mark I", "tongue": "zsh"}`)
	writeScripture(t, b, "rites/r.json", sanctumRite("~/b.kdl", ""))
	writeScripture(t, b, SettingsName, `{"pattern": "Mark I", "tongue": "fish", "vox": "loud"}`)

	la, lb := Load(a), Load(b)
	if la.Settings == nil || la.Settings.Tongue != "zsh" || len(la.Findings) != 0 {
		t.Fatalf("librarium a: settings %+v, findings %v", la.Settings, la.Findings)
	}
	if lb.Settings == nil || lb.Settings.Tongue != "fish" {
		t.Fatalf("librarium b: settings %+v", lb.Settings)
	}
	if got := findingsWith(lb.Findings, `"loud" is no vox`); len(got) != 1 || got[0].Scripture != filepath.Join(b, SettingsName) {
		t.Fatalf("librarium b must denounce its settings: %v", lb.Findings)
	}
	if lb.Rites["r"] == nil || lb.Heretical["r"] {
		t.Fatalf("heretical settings must not make a rite heretical: %v", lb.Heretical)
	}
	if c := Load(t.TempDir()); c.Settings == nil || *c.Settings != (Settings{Path: filepath.Join(c.Dir, SettingsName)}) {
		t.Fatalf("a Librarium without settings: %+v", c.Settings)
	}
}
