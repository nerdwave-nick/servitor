package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/config"
	"github.com/nerdwave-nick/servitor/internal/engine"
)

// The consecration wizard still speaks the elder scripture until it learns
// pattern Mark I; these trials keep it whole until then.

// newElderHarness is a harness whose rite mouse is elder scripture.
func newElderHarness(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t)
	h.writeRite("mouse", strings.ReplaceAll(sampleRite, "/tmp/x.kdl", filepath.Join(h.dataDir, "x.kdl")))
	h.keys("r")
	return h
}

func TestWizard_CreateRite(t *testing.T) {
	h := newElderHarness(t)
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
	sw := config.Load(h.dir).Switches["theme"]
	if sw == nil || !sw.Files[0].Create || sw.Files[0].Values[1].Meta["reason"] != "sunny" {
		t.Fatalf("saved rite = %+v", sw)
	}
	if _, err := engine.Apply(sw, "light", nil, engine.Options{}); err != nil {
		t.Fatal(err)
	}
}

func TestWizard_EditRenameAndValidation(t *testing.T) {
	h := newElderHarness(t)
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
	if config.Load(h.dir).Switches["cursor"] == nil {
		t.Fatal("renamed rite not loaded")
	}
}

func TestWizard_ReviewBlocksInvalidDefinition(t *testing.T) {
	h := newElderHarness(t)
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
