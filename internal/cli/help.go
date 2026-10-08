package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/nerdwave-nick/servitor/internal/lexicon"
)

// exampleRite is the sample definition shown in the root help.
const exampleRite = `  {
    "description": "Hide the mouse cursor after inactivity",
    "states": ["on", "off"],
    "files": [{
      "file": "~/.config/niri/util.kdl",
      "guard": "mouse-autohide",          // default: the name
      "comment": "//",                    // default: inferred from extension
      "values": [
        {"state": "on",  "value": "cursor {\n    hide-after-inactive-ms 400\n}"},
        {"state": "off", "value": ""}
      ],
      "meta": {"reason": {"optional": true, "description": "why it was switched"}}
    }]
  }`

func rootLong(l *lexicon.Lexicon) string {
	if l.Grimdark {
		return `+++ SERVITOR · Configuration Thrall of the Adeptus Mechanicus +++

The servitor performs RITES upon the sacred texts of your machine. Each rite
knows a fixed set of ASPECTS. Invoking an aspect rewrites one warded SANCTUM in
every VESSEL (target file) of the rite:

  // begin servitor managed -- <ward> -- state|on reason|"gaming remnant"
  <scripture kept by the servitor>
  // end servitor managed -- <ward>

Only the lines between the wards are ever touched; the rest of the vessel
remains inviolate. A sanctum not yet consecrated is appended to the end of its
vessel. The header bears INSCRIPTIONS as key|value pairs: "state" is inscribed
by the servitor alone, all others are declared by the rite and may be
overridden with --<key> runes.

Rites are kept in the Librarium, one per file:
  <config>/rites/<name>.json      (JSON with comments; "switches/" is also searched)

` + exampleRite + `

Invoked without a command (and attached to a terminal), the servitor awakens
the COGITATOR, an interactive shrine for performing, consecrating, amending and
excommunicating rites.

Those whose spirits are too weak for the liturgy may pass --no-grimdark or set
SERVITOR_NO_GRIMDARK=1. The Omnissiah judges no one. Mostly.`
	}
	return `servitor switches managed blocks inside files between configured states.

A switch is a named set of file changes with a fixed list of states. Applying
a state rewrites one guarded block in each file of the switch:

  // begin servitor managed -- <guard> -- state|on reason|"gaming remnant"
  <content managed by servitor>
  // end servitor managed -- <guard>

Only the lines between the markers are ever modified. A block that does not
exist yet is appended to the end of the file. The header stores metadata as
key|value pairs: "state" is set automatically, other keys are declared by the
switch and can be overridden with --<key> flags.

Switches are defined one per file in:
  <config>/rites/<name>.json      (JSON with comments; "switches/" is also searched)

` + exampleRite + `

Without a command (on a terminal), servitor opens an interactive TUI for
applying, creating, editing and deleting switches.

The default vocabulary is Warhammer 40k themed; --no-grimdark or
SERVITOR_NO_GRIMDARK=1 selects the plain vocabulary used here.`
}

func rootExample(l *lexicon.Lexicon) string {
	c := l.Cmd
	return strings.Join([]string{
		"  servitor                                   # " + l.P("awaken the cogitator", "open the TUI"),
		"  servitor " + c.Switch + ` mouse-autohide-toggle on --reason "gaming remnant"`,
		"  servitor " + c.Switch + " mouse-autohide-toggle off",
		"  servitor " + c.Meta + " mouse-autohide-toggle",
		"  servitor " + c.Meta + " mouse-autohide-toggle state",
		"  servitor " + c.Meta + " mouse-autohide-toggle --is on && echo " + l.P(`"the cursor is veiled"`, `"cursor hiding is on"`),
		"  servitor " + c.Verify,
		"  source <(servitor completion bash)",
	}, "\n")
}

func switchLongHelp(l *lexicon.Lexicon) string {
	if l.Grimdark {
		return `Invoke an aspect of a rite: every sanctum of the rite is rewritten with the
scripture its aspect prescribes.

Every inscription declared by the rite is available as a --<key> rune that
overrides the prescribed value; pass an empty string to erase it. Consult
"servitor ` + l.Cmd.Switch + ` <rite> --help" to learn the aspects and runes of one rite.`
	}
	return `Apply a state of a switch: rewrite the managed block of every file of the
switch with the value configured for that state.

Every metadata key declared by the switch is available as a --<key> flag that
overrides the configured value; pass an empty string to clear it. Run
"servitor ` + l.Cmd.Switch + ` <name> --help" to see the states and flags of one switch.`
}

func metaLongHelp(l *lexicon.Lexicon) string {
	if l.Grimdark {
		return `Perform an augury: read the inscriptions from the sanctum headers of a rite.

Without a key, the augury speaks a JSON object with "state" and every inscribed
value. With a key, it speaks only that value (an empty line when uninscribed).
With --is <aspect>, it speaks nothing and exits 0 when the rite stands in that
aspect and 1 otherwise, for shell conditions.

Exit codes: 0 the augury succeeded, 1 --is did not match, 2 the augury failed
(unknown rite, never performed, or vessels that disagree; consult --per-file).`
	}
	return `Read the metadata from the managed block headers of a switch.

Without a key, prints a JSON object with "state" and every non-empty metadata
value. With a key, prints only that value (an empty line when it is unset).
With --is <state>, prints nothing and exits 0 when the switch is in that state
and 1 otherwise, for use in shell conditions.

Exit codes: 0 success, 1 --is did not match, 2 error (unknown switch, not
applied yet, or files that disagree; use --per-file to inspect them).`
}

func verifyLongHelp(l *lexicon.Lexicon) string {
	if l.Grimdark {
		return `Summon the Inquisition to examine every rite (or only the named ones). Each
heresy is reported with its location as file:line:column.

The Inquisition purges: malformed JSON, unknown or mistyped fields, undeclared
or missing aspects, undeclared or reserved inscriptions, rites whose names
clash (one name recorded in several files), wards claimed twice for the same
vessel, and — unless --no-files — vessels that are missing, sanctums with
broken or duplicated wards, and scripture tainted by unsanctioned hands.

Exits 1 when any heresy was found. Impurities are noted but forgiven.`
	}
	return `Check every switch definition (or only the named ones) and report problems
with their location as file:line:column.

Checks include JSON syntax, unknown or mistyped fields, undeclared or missing
states, undeclared or reserved metadata keys, switch name clashes (the same
name defined in several files), guard clashes (two entries managing the same
guard in the same target file), and — unless --no-files is given — missing
target files, malformed or duplicated markers, and managed content that was
edited by hand.

Exits 1 when any error was found; warnings do not fail.`
}

func init() {
	// Usage lines are generated by cobra with a literal " [flags]" suffix.
	cobra.AddTemplateFunc("servitorRunes", func(s string) string { return strings.Replace(s, "[flags]", "[runes]", 1) })
}

// flavorUsage rewrites cobra's usage template headings into liturgy.
func flavorUsage(l *lexicon.Lexicon, tmpl string) string {
	if !l.Grimdark {
		return tmpl
	}
	return strings.NewReplacer(
		"{{.UseLine}}", "{{servitorRunes .UseLine}}",
		`Use "{{.CommandPath}}`, `Consult "{{.CommandPath}}`,
		"Usage:", "Invocation:",
		"Aliases:", "Also Known As:",
		"Examples:", "Exempla:",
		"Available Commands:", "Sanctioned Rituals:",
		"Additional Commands:", "Lesser Rituals:",
		"Global Flags:", "Universal Runes:",
		"Flags:", "Runes:",
		"Additional help topics:", "Further Lore:",
		"for more information about a command.", "for deeper lore on a ritual.",
		"[command]", "[ritual]",
	).Replace(tmpl)
}
