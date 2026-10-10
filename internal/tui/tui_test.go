package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/nerdwave-nick/servitor/internal/augury"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// harness drives the model with key presses and renders after every step,
// so any panic in update or view fails the test. What an invocation sends
// from its own goroutine waits in the inbox until next or await delivers it.
type harness struct {
	t       *testing.T
	m       *model
	dir     string
	dataDir string
	stub    string // where the notify-send stub records its calls
	inbox   chan tea.Msg
}

// markRite is a rite of pattern Mark I: a progress vox-cast, a sanctum, an
// incantation that utters words and leaves the aspect for the auspex, and a
// success vox-cast. $DATA is the place of its vessels.
const markRite = `{
  "pattern": "Mark I",
  "purpose": "Hide the cursor",
  "aspects": ["on", "off"],
  "inscriptions": {
    "reason": {"purpose": "why"},
    "mode": {"mandatory": true, "decrees": {"on": "hide", "off": "show"}}
  },
  "auspex": "cat $DATA/mode",
  "liturgy": [
    {"vox-cast": "progress"},
    {"sanctum": "$DATA/x.kdl", "ward": "cursor", "scripture": {"on": "a\nb", "off": ""}},
    {"incantation": "echo chanting {{aspect}}; echo {{aspect}} > $DATA/mode", "reversion": "rm -f $DATA/mode"},
    {"vox-cast": "success"}
  ]
}`

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{t: t, dir: t.TempDir(), dataDir: t.TempDir(), inbox: make(chan tea.Msg, 64)}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv(librarium.EnvChronicle, "")
	stubs := t.TempDir()
	h.stub = filepath.Join(stubs, "heard")
	stub := "#!/bin/sh\necho \"$@\" >> " + h.stub + "\necho 7\n"
	if err := os.WriteFile(filepath.Join(stubs, "notify-send"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubs+string(os.PathListSeparator)+os.Getenv("PATH"))
	h.writeRite("mouse", markRite)
	h.writeData("x.kdl", "input {}\n")
	h.m = newModel(h.dir, librarium.Runes{})
	h.m.send = func(msg tea.Msg) { h.inbox <- msg }
	h.send(tea.WindowSizeMsg{Width: 120, Height: 36})
	return h
}

// writeRite writes the scripture of the rite name; $DATA is the vessels' place.
func (h *harness) writeRite(name, body string) string {
	h.t.Helper()
	return write(h.t, filepath.Join(h.dir, "rites", name+".json"), strings.ReplaceAll(body, "$DATA", h.dataDir))
}

func (h *harness) writeData(rel, content string) string {
	h.t.Helper()
	return write(h.t, filepath.Join(h.dataDir, rel), content)
}

func (h *harness) readData(rel string) string {
	h.t.Helper()
	data, err := os.ReadFile(filepath.Join(h.dataDir, rel))
	if err != nil {
		h.t.Fatal(err)
	}
	return string(data)
}

func write(t *testing.T, p, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
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
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	case "ctrl+p":
		return tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl}
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
	case <-time.After(20 * time.Millisecond):
		return nil, false
	}
}

// next delivers the next message of the running invocation.
func (h *harness) next() tea.Msg {
	h.t.Helper()
	select {
	case msg := <-h.inbox:
		h.send(msg)
		return msg
	case <-time.After(10 * time.Second):
		h.t.Fatalf("the invocation sent nothing:\n%s", h.screen())
	}
	return nil
}

// await delivers the invocation's messages until it has ended.
func (h *harness) await() {
	h.t.Helper()
	for {
		if _, ended := h.next().(invokedMsg); ended {
			return
		}
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

func (h *harness) mustNotShow(substrs ...string) {
	h.t.Helper()
	out := h.screen()
	for _, s := range substrs {
		if strings.Contains(out, s) {
			h.t.Fatalf("screen shows %q:\n%s", s, out)
		}
	}
}

// reading is the augury of the rite name, read anew.
func (h *harness) reading(name string) augury.Augury {
	h.t.Helper()
	h.m.reload("")
	for _, r := range h.m.all {
		if r.name == name {
			return r.reading.Augury
		}
	}
	h.t.Fatalf("no rite %q", name)
	return augury.Augury{}
}

// desktopSilent fails when notify-send was called: the cogitator shows its
// vox-casts itself.
func (h *harness) desktopSilent() {
	h.t.Helper()
	if data, err := os.ReadFile(h.stub); err == nil {
		h.t.Fatalf("notify-send was called: %q", data)
	}
}

func TestOverview_ShowsRitesAndDetails(t *testing.T) {
	h := newHarness(t)
	h.writeRite("broken", `{"pattern": "Mark I", "aspects": [}`)
	h.keys("r")
	h.mustShow("SERVITOR", "COGITATOR", "Rites (2)", "mouse", "dormant", "heretical", "Thought for the day")
	h.keys("j")
	h.mustShow("Hide the cursor", "ASPECTS", "◇ on", "◇ off", "STANDING", "dormant",
		"LAST RITE", "never invoked", "INSCRIPTIONS", "mode=—", "reason=—", "LITURGY",
		"· 1 ✉ progress", "· 2 § "+shortPath(filepath.Join(h.dataDir, "x.kdl")),
		"· 3 » echo chanting {{aspect}}", "· 4 ✉ success", "AUSPEX", "· silent",
		"RECORDED IN", "rites/mouse.json")
	h.keys("k")
	h.mustShow("tainted by heresy", "1:")
	h.keys("enter")
	h.mustShow("This rite is heretical")
}

func TestDetail_ShowsTheOmensOfEveryStep(t *testing.T) {
	h := newHarness(t)
	h.keys("space")
	h.await()
	h.mustShow("◆ on", "◇ off", "performed", "LAST RITE", "✔ triumph", "mode=hide",
		"✔ 2 § ", "· 3 » echo chanting {{aspect}}", "AUSPEX", "✔ on")

	// other hands rewrite the sanctum's marker: the omens disagree
	h.writeData("x.kdl", strings.Replace(h.readData("x.kdl"), "aspect|on", "aspect|off", 1))
	h.keys("r")
	h.mustShow("corrupted", "✖ 2 § ", "(off)", "✖ on")
}

func TestOverview_Cycle(t *testing.T) {
	h := newHarness(t)
	h.keys("space")
	h.await()
	if a := h.reading("mouse"); a.Aspect != "on" || a.Standing != augury.Performed {
		t.Fatalf("cycle from dormant: %+v", a)
	}
	h.mustShow("The rite mouse is performed: dormant → on")
	h.keys("space")
	h.await()
	if a := h.reading("mouse"); a.Aspect != "off" || a.Inscriptions["mode"] != "show" {
		t.Fatalf("cycle from on: %+v", a)
	}
	h.desktopSilent()
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

func TestDelete_StrikeAndPurge(t *testing.T) {
	h := newHarness(t)
	h.writeRite("broken", `{"pattern": "Mark I", "aspects": [}`)
	h.writeData("whole.conf", "kept\n")
	h.keys("r", "j", "space") // invoke mouse into "on"
	h.await()
	if !strings.Contains(h.readData("x.kdl"), "begin of sanctum cursor") {
		t.Fatalf("the sanctum was not written:\n%s", h.readData("x.kdl"))
	}
	h.keys("d")
	h.mustShow("Excommunicate the rite mouse?", "purge its sanctums from every vessel",
		"transcribed vessels and tethers")
	h.keys("n")
	h.mustShow("Mercy is shown. The rite endures.")
	h.keys("d", "p")
	h.mustShow("The rite mouse is excommunicated and its sanctums purged from 1 vessel", "Rites (1)")
	if got := h.readData("x.kdl"); got != "input {}\n" {
		t.Fatalf("the sanctum was not purged: %q", got)
	}
	if _, err := os.Stat(filepath.Join(h.dir, "rites", "mouse.json")); !os.IsNotExist(err) {
		t.Fatal("the scripture still stands")
	}

	h.keys("d")
	h.mustShow("Excommunicate the rite broken?")
	h.mustNotShow("purge its sanctums")
	h.keys("p")
	h.mustShow("Excommunicate the rite broken?")
	h.keys("y")
	h.mustShow("The rite broken is excommunicated. Its sanctums remain.", "The Librarium is empty.")
}

func TestFilterHelpAndVerdict(t *testing.T) {
	h := newHarness(t)
	h.writeRite("theme", strings.ReplaceAll(markRite, `"ward": "cursor"`, `"ward": "theme"`))
	h.writeRite("lost", strings.ReplaceAll(markRite, "x.kdl", "absent.kdl"))
	h.keys("r", "/")
	h.typeText("the")
	h.mustShow("Rites (1)")
	h.keys("enter", "esc")
	h.mustShow("Rites (3)")
	h.keys("?")
	h.mustShow("Catalogue of Sacred Keys", "excommunicate", "words of the last invocation")
	h.keys("esc", "i")
	h.mustShow("Verdict of the Inquisition", "impurity", "lost.json", "no vessel stands at")
	h.keys("esc", "q")
}

func TestAmend_MarkIScriptureIsAmendedByHand(t *testing.T) {
	h := newHarness(t)
	h.keys("e")
	h.mustShow("amend it with o")
	h.keys("c")
	h.mustShow("amend it with o")
}

func TestRender_TooSmall(t *testing.T) {
	h := newHarness(t)
	h.send(tea.WindowSizeMsg{Width: 40, Height: 10})
	h.mustShow("The cogitator demands a larger viewscreen")
}
