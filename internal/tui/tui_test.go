package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/nerdwave-nick/servitor/internal/engine"
)

// harness drives the model with key presses and renders after every step,
// so any panic in update or view fails the test.
type harness struct {
	t       *testing.T
	m       *model
	dir     string
	dataDir string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{t: t, dir: t.TempDir(), dataDir: t.TempDir()}
	h.writeRite("mouse", strings.ReplaceAll(sampleRite, "/tmp/x.kdl", filepath.Join(h.dataDir, "x.kdl")))
	if err := os.WriteFile(filepath.Join(h.dataDir, "x.kdl"), []byte("input {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h.m = newModel(h.dir)
	h.send(tea.WindowSizeMsg{Width: 120, Height: 36})
	return h
}

func (h *harness) writeRite(name, body string) {
	h.t.Helper()
	p := filepath.Join(h.dir, "rites", name+".json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		h.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		h.t.Fatal(err)
	}
}

func keyMsg(k string) tea.KeyPressMsg {
	switch k {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "ctrl+s":
		return tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}
	case "ctrl+u":
		return tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}
	}
	r := []rune(k)[0]
	return tea.KeyPressMsg{Code: r, Text: k}
}

func (h *harness) send(msgs ...tea.Msg) {
	h.t.Helper()
	for _, msg := range msgs {
		_, cmd := h.m.Update(msg)
		h.runCmd(cmd)
		_ = h.m.View()
	}
}

// runCmd executes synchronous follow-up commands (reloads), skipping timers.
func (h *harness) runCmd(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	if msg, ok := runQuick(cmd); ok {
		if _, isReload := msg.(reloadMsg); isReload {
			h.send(msg)
		}
	}
}

func runQuick(cmd tea.Cmd) (tea.Msg, bool) {
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		return msg, true
	default:
		return nil, false
	}
}

func (h *harness) keys(keys ...string) {
	h.t.Helper()
	for _, k := range keys {
		h.send(keyMsg(k))
	}
}

func (h *harness) typeText(s string) {
	h.t.Helper()
	for _, r := range s {
		h.send(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func (h *harness) screen() string { return ansi.Strip(h.m.render()) }

func (h *harness) mustShow(substrs ...string) {
	h.t.Helper()
	out := h.screen()
	for _, s := range substrs {
		if !strings.Contains(out, s) {
			h.t.Fatalf("screen lacks %q:\n%s", s, out)
		}
	}
}

func (h *harness) state(name string) string {
	h.t.Helper()
	h.m.reload("")
	for _, r := range h.m.all {
		if r.name == name && r.sw != nil {
			return engine.ReadStatus(r.sw).State()
		}
	}
	return ""
}

func TestOverview_ShowsRitesAndDetails(t *testing.T) {
	h := newHarness(t)
	h.writeRite("broken", `{"states": [}`)
	h.keys("r")
	h.mustShow("SERVITOR", "COGITATOR", "Rites (2)", "mouse", "dormant", "heretical", "Thought for the day")
	h.keys("j")
	h.mustShow("Hide the cursor", "ASPECTS", "VESSELS", "ward cursor", "INSCRIPTIONS")
	h.keys("k")
	h.mustShow("tainted by heresy", "syntax error")
	h.keys("e")
	h.mustShow("This rite is heretical")
}

func TestOverview_Cycle(t *testing.T) {
	h := newHarness(t)
	h.keys("space")
	if got := h.state("mouse"); got != "on" {
		t.Fatalf("cycle from dormant: %q", got)
	}
	h.mustShow("Rite mouse performed", "→ on")
	h.keys("space")
	if got := h.state("mouse"); got != "off" {
		t.Fatalf("cycle from on: %q", got)
	}
}

// TestOverview_TKeyIsSilent: the cogitator knows one liturgy; t neither
// changes the screen nor appears in the catalogue of keys.
func TestOverview_TKeyIsSilent(t *testing.T) {
	h := newHarness(t)
	before := h.screen()
	h.keys("t")
	if after := h.screen(); after != before {
		t.Fatalf("t changed the cogitator:\n%s", after)
	}
	h.mustShow("Rites (1)", "COGITATOR", "Thought for the day")
	h.keys("?")
	h.mustShow("Catalogue of Sacred Keys")
	for _, line := range strings.Split(h.screen(), "\n") {
		if f := strings.Fields(strings.Trim(line, " │|")); len(f) > 0 && f[0] == "t" {
			t.Fatalf("catalogue still names t: %q", line)
		}
	}
	if strings.Contains(h.screen(), "toggle") {
		t.Fatalf("catalogue still offers a vocabulary toggle:\n%s", h.screen())
	}
}

func TestInvoke_PickStateFillMetaAndPreview(t *testing.T) {
	h := newHarness(t)
	h.keys("enter")
	h.mustShow("Invoke mouse", "Choose the aspect to invoke")
	h.keys("p")
	h.mustShow("Augury of aspect on", "+ // begin servitor managed -- cursor -- state|on mode|hide")
	h.keys("esc")
	h.mustShow("Choose the aspect to invoke")
	h.keys("1")
	h.mustShow("Inscriptions for aspect", "mode *", "reason")
	h.keys("tab")
	h.typeText("gaming remnant")
	h.keys("enter")
	if got := h.state("mouse"); got != "on" {
		t.Fatalf("state after invoke: %q", got)
	}
	st := engine.ReadStatus(h.m.set.Switches["mouse"])
	if st.Meta["reason"] != "gaming remnant" || st.Meta["mode"] != "hide" {
		t.Fatalf("meta = %v", st.Meta)
	}
	h.mustShow("Rite mouse performed: dormant → on")
}

func TestInvoke_RequiredMetaClearedShowsError(t *testing.T) {
	h := newHarness(t)
	h.keys("enter", "1")
	h.keys("ctrl+u", "enter", "enter")
	h.mustShow("The rite falters:", `"mode" is required`)
}

func TestWizard_CreateRite(t *testing.T) {
	h := newHarness(t)
	target := filepath.Join(h.dataDir, "new.conf")
	h.keys("n")
	h.mustShow("Consecration of a new rite", "Name of the rite")
	h.keys("enter")
	h.mustShow("a rite's name must be")
	h.typeText("theme")
	h.keys("tab")
	h.typeText("Colour scheme")
	h.keys("tab")
	for range len("on, off") {
		h.keys("backspace")
	}
	h.typeText("dark, light")
	h.keys("enter") // to the first vessel page
	h.mustShow("New vessel", "Vessel (target file)")
	h.typeText(filepath.Join(h.dataDir, "x.lua"))
	if ph := h.m.screen.(*wizard).form.fields[2].input.Placeholder; ph != "--" {
		t.Fatalf("comment placeholder for .lua = %q", ph)
	}
	h.keys("ctrl+u")
	h.typeText(target)
	h.keys("tab", "tab", "tab", "tab", "space", "tab") // guard, comment, comment_end, create (on), inscriptions
	h.typeText("reason")
	h.keys("enter")
	h.mustShow(`Scripture for aspect "dark"`)
	h.typeText("scheme = dark")
	h.keys("ctrl+s")
	h.mustShow(`Scripture for aspect "light"`)
	h.typeText("scheme = light")
	h.keys("tab")
	h.typeText("sunny")
	h.keys("enter")
	h.mustShow(target, "reason")
	h.keys("tab")
	h.mustShow("The rite is pure", `"scheme = light"`)
	h.keys("enter")
	h.mustShow("The rite theme is consecrated")
	if _, err := os.Stat(filepath.Join(h.dir, "rites", "theme.json")); err != nil {
		t.Fatal(err)
	}
	sw := h.m.set.Switches["theme"]
	if sw == nil || !sw.Files[0].Create || sw.Files[0].Values[1].Meta["reason"] != "sunny" {
		t.Fatalf("saved rite = %+v", sw)
	}
	if _, err := engine.Apply(sw, "light", nil, engine.Options{}); err != nil {
		t.Fatal(err)
	}
}

func TestWizard_EditRenameAndValidation(t *testing.T) {
	h := newHarness(t)
	h.keys("e")
	h.mustShow("Amendment of the rite mouse", "Speak the name, purpose and aspects")
	for range len("mouse") {
		h.keys("backspace")
	}
	h.typeText("cursor")
	h.keys("enter", "enter", "enter")
	h.mustShow("The vessels whose sanctums this rite keeps", "x.kdl")
	h.keys("enter") // edit the vessel
	h.mustShow("Vessel 1", "settings")
	h.keys("esc")
	h.keys("tab")
	h.mustShow("The rite is pure. Press enter to seal it", "rites/cursor.json")
	h.keys("enter")
	h.mustShow("The rite cursor is amended.")
	if _, err := os.Stat(filepath.Join(h.dir, "rites", "mouse.json")); !os.IsNotExist(err) {
		t.Fatal("old definition not removed after rename")
	}
	if h.m.set.Switches["cursor"] == nil {
		t.Fatal("renamed rite not loaded")
	}
}

func TestWizard_ReviewBlocksInvalidDefinition(t *testing.T) {
	h := newHarness(t)
	h.writeRite("other", `{"states":["a"],"files":[{"file":"/other.conf","guard":"cursor","values":[{"state":"a","value":""}]}]}`)
	h.keys("r", "c") // clone the selected rite ("mouse" sorts before "other")
	h.mustShow("Consecration of a new rite", "mouse-copy")
	h.keys("enter", "enter", "enter", "tab")
	h.mustShow("Heresy detected", `guard "cursor"`)
	h.keys("enter")
	h.mustShow("Heresy remains. The rite cannot be sealed.")
	if _, err := os.Stat(filepath.Join(h.dir, "rites", "mouse-copy.json")); !os.IsNotExist(err) {
		t.Fatal("invalid definition was saved")
	}
	h.keys("esc", "d") // remove the only vessel, then try to continue
	h.keys("tab")
	h.mustShow("A rite without vessels is an empty prayer")
	h.keys("esc", "esc")
	h.mustShow("The consecration is abandoned.")
}

func TestDelete_DefinitionAndPurge(t *testing.T) {
	h := newHarness(t)
	h.keys("space") // apply "on"
	h.keys("d")
	h.mustShow("Excommunicate the rite mouse?", "purge its sanctums from every vessel")
	h.keys("n")
	h.mustShow("Mercy is shown. The rite endures.")
	h.keys("d", "p")
	h.mustShow("The rite mouse is excommunicated and its sanctums purged.", "The Librarium is empty.")
	data, _ := os.ReadFile(filepath.Join(h.dataDir, "x.kdl"))
	if string(data) != "input {}\n" {
		t.Fatalf("block not purged: %q", data)
	}
}

func TestFilterHelpAndVerdict(t *testing.T) {
	h := newHarness(t)
	h.writeRite("theme", strings.ReplaceAll(sampleRite, `"guard": "cursor"`, `"guard": "theme"`))
	h.keys("r", "/")
	h.typeText("the")
	h.mustShow("Rites (1)")
	h.keys("enter", "esc")
	h.mustShow("Rites (2)")
	h.keys("?")
	h.mustShow("Catalogue of Sacred Keys", "excommunicate", "ctrl+s seal")
	h.keys("esc", "i")
	h.mustShow("Verdict of the Inquisition")
	h.keys("esc", "q")
}

func TestRender_TooSmall(t *testing.T) {
	h := newHarness(t)
	h.send(tea.WindowSizeMsg{Width: 40, Height: 10})
	h.mustShow("The cogitator demands a larger viewscreen")
}
