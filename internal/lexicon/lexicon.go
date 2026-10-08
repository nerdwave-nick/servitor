// Package lexicon holds the two vocabularies of servitor: the liturgy of the
// Adeptus Mechanicus (the default) and plain technical speech for those who
// pass --no-grimdark.
//
// Only human-facing text is translated. Configuration keys, file markers,
// JSON output and exit codes are identical in both modes, so scripts and
// managed files never depend on the vocabulary in use.
package lexicon

import (
	"os"
	"strconv"
	"strings"
)

// EnvNoGrimdark disables the grimdark vocabulary when set to a true value.
const EnvNoGrimdark = "SERVITOR_NO_GRIMDARK"

// FlagNoGrimdark is the command line flag disabling the grimdark vocabulary.
const FlagNoGrimdark = "no-grimdark"

// Commands holds the primary command names of one vocabulary.
type Commands struct {
	Switch, Meta, List, Verify, TUI string
}

// Lexicon is one vocabulary.
type Lexicon struct {
	Grimdark bool

	Switch, Switches     string // rite, rites
	State, States        string // aspect, aspects
	File, Files          string // vessel, vessels
	Block                string // sanctum
	Guard                string // ward
	Meta                 string // inscriptions
	Description          string // purpose
	ConfigDir            string // Librarium
	Error, Warning       string // heresy, impurity
	Errors, Warnings     string // heresies, impurities
	Applied, NotApplied  string // performed, dormant
	Inconsistent, Broken string // corrupted, heretical
	Drift                string // tainted

	Cmd Commands
}

var (
	grimdark = &Lexicon{
		Grimdark: true,
		Switch:   "rite", Switches: "rites",
		State: "aspect", States: "aspects",
		File: "vessel", Files: "vessels",
		Block: "sanctum", Guard: "ward", Meta: "inscriptions",
		Description: "purpose", ConfigDir: "Librarium",
		Error: "heresy", Warning: "impurity", Errors: "heresies", Warnings: "impurities",
		Applied: "performed", NotApplied: "dormant", Inconsistent: "corrupted", Broken: "heretical",
		Drift: "tainted",
		Cmd:   Commands{Switch: "invoke", Meta: "augury", List: "census", Verify: "inquisition", TUI: "cogitator"},
	}
	plain = &Lexicon{
		Switch: "switch", Switches: "switches",
		State: "state", States: "states",
		File: "file", Files: "files",
		Block: "block", Guard: "guard", Meta: "metadata",
		Description: "description", ConfigDir: "config directory",
		Error: "error", Warning: "warning", Errors: "errors", Warnings: "warnings",
		Applied: "applied", NotApplied: "not applied", Inconsistent: "inconsistent", Broken: "invalid",
		Drift: "edited by hand",
		Cmd:   Commands{Switch: "switch", Meta: "meta", List: "list", Verify: "verify", TUI: "tui"},
	}
)

// Get returns the grimdark or the plain vocabulary.
func Get(grim bool) *Lexicon {
	if grim {
		return grimdark
	}
	return plain
}

// Toggle returns the other vocabulary.
func (l *Lexicon) Toggle() *Lexicon { return Get(!l.Grimdark) }

// P picks the grimdark or the plain variant of a phrase.
func (l *Lexicon) P(grim, plain string) string {
	if l.Grimdark {
		return grim
	}
	return plain
}

// Title capitalizes the first letter of s.
func Title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// Detect decides the vocabulary from the command line and environment:
// the last --no-grimdark[=bool] flag wins, then $SERVITOR_NO_GRIMDARK.
// Grimdark is the default.
func Detect(args []string) bool {
	noGrim, set := false, false
	for _, arg := range args {
		if arg == "--" {
			break
		}
		switch {
		case arg == "--"+FlagNoGrimdark:
			noGrim, set = true, true
		case strings.HasPrefix(arg, "--"+FlagNoGrimdark+"="):
			if v, err := strconv.ParseBool(strings.TrimPrefix(arg, "--"+FlagNoGrimdark+"=")); err == nil {
				noGrim, set = v, true
			}
		}
	}
	if !set {
		v, err := strconv.ParseBool(os.Getenv(EnvNoGrimdark))
		noGrim = err == nil && v
	}
	return !noGrim
}

// Thoughts are the Thoughts for the Day shown by the cogitator.
var Thoughts = []string{
	"A file unwarded is a file defiled.",
	"Trust not the hand-edited config.",
	"The diff does not lie. The editor might.",
	"Suffer not the merge conflict to live.",
	"Even a dotfile may serve the Omnissiah.",
	"In the absence of a backup, faith must suffice.",
	"Comment your wards, lest your successors curse your name.",
	"Idle cursors breed heresy.",
	"Every newline is a prayer.",
	"Blessed is the config that parses on the first attempt.",
	"A rite performed twice changes nothing. Such is idempotence; such is grace.",
	"Question not the state. Consult the augury.",
	"A servitor does not ask why. It asks only which aspect.",
	"Only in rollback is there true safety.",
	"The Librarium forgets nothing that git remembers.",
	"Rename in haste, grep at leisure.",
}

// Thought returns the Thought for the Day with index n (any integer).
func Thought(n int) string {
	if n < 0 {
		n = -n
	}
	return Thoughts[n%len(Thoughts)]
}
