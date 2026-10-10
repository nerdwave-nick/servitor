# servitor

> *+++ Thought for the day: A file unwarded is a file defiled. +++*

**servitor** is a configuration thrall of the Adeptus Mechanicus. It performs
**rites** upon the sacred texts of your machine: each rite knows a fixed set of
**aspects**, and invoking an aspect rewrites one warded **sanctum** inside every
**vessel** (target file) of the rite. It asks not *why*. It asks only *which
aspect*.

```sh
servitor                                                     # awaken the cogitator (TUI)
servitor invoke mouse-autohide-toggle on --reason "gaming remnant"
servitor invoke mouse-autohide-toggle off
servitor augury mouse-autohide-toggle                        # {"state": "off", ...}
servitor augury mouse-autohide-toggle --is on || echo "the cursor walks unveiled"
```

## The Sanctum

```kdl
// begin servitor managed -- mouse-autohide -- state|off reason|"gaming remnant"
cursor {
    // hide-after-inactive-ms 400
}
// end servitor managed -- mouse-autohide
```

* Only the lines between the wards are ever touched. The rest of the vessel
  remains inviolate.
* A sanctum not yet consecrated is appended to the end of its vessel.
* The header bears **inscriptions**. `state` is inscribed by the servitor
  alone and cannot be forged. All others are declared by the rite, take their
  values from the rite, and may be overridden with `--<key> <value>`
  (`--<key> ""` erases one). Empty inscriptions are omitted. Values with
  spaces are quoted.
* Writes are atomic (temp file + rename), keep file permissions, follow
  symlinks (so your dotfiles shrine stays intact), and are rolled back if a
  later vessel refuses the rite.

There are no entry or exit commands. What happens after a rite is your
scripts' business. That is what the augury is for:

```sh
servitor invoke mouse-autohide-toggle on -s && notify-send "The rite is performed"
[ "$(servitor augury mouse-autohide-toggle state)" = on ] && notify-send "The cursor is veiled"
```

## Installation

Requires Go 1.27+.

```sh
go install github.com/nerdwave-nick/servitor@latest
# or, from a checkout:
go install .              # or: go build -o ~/.local/bin/servitor .
```

### Completion Litanies

Completion knows rites, aspects (the current one is marked), inscription
runes (`--<rune-key>`, …) and their prescribed values, and augury keys.

```sh
# bash (requires the bash-completion package)
servitor completion bash > ~/.local/share/bash-completion/completions/servitor
# zsh (any directory on $fpath, compinit enabled)
servitor completion zsh > "${fpath[1]}/_servitor"
# fish
servitor completion fish > ~/.config/fish/completions/servitor.fish
```

## The Cogitator

Invoking `servitor` without a command on a terminal awakens the cogitator, a
keyboard-driven shrine in the manner of lazygit: the rites on the left, the
scripture of the chosen rite on the right (aspects, status, inscriptions,
vessels with their sanctum state), and the Thought for the Day below.

| Key | Rite |
|-----|------|
| `↑/k` `↓/j` `g` `G` | choose a rite |
| `enter` | invoke: choose an aspect (`1-9` for swift choice), `p` for an augury of the diff, then amend inscriptions |
| `space` | cycle to the next aspect at once |
| `n` | consecrate a new rite (guided) |
| `e` | amend the rite (guided) |
| `c` | replicate the rite |
| `d` | excommunicate: `y` strike the definition, `p` also purge its sanctums |
| `o` | open the scripture in `$EDITOR` |
| `i` | the verdict of the Inquisition |
| `/` | filter |
| `r` | re-read the Librarium |
| `?` | lore |
| `q` | retreat |

The consecration wizard walks through three stations: **Rite** (name,
purpose, aspects), **Vessels** (each with its path, ward, comment glyph, an
optional *create* flag and its inscriptions, then one page of scripture per
aspect), and **Seal**, which shows the resulting JSON and the Inquisition's
verdict *before* anything is written. A heretical rite cannot be sealed. In
forms, `tab`/`↑↓` move between fields, `enter` advances, `ctrl+s` seals the
page and `esc` steps back.

## The Librarium

Rites dwell in `$XDG_CONFIG_HOME/servitor` (`~/.config/servitor`). Point
elsewhere with `--librarium <dir>` / `-l <dir>` or `SERVITOR_LIBRARIUM`; the rune
outranks the environment.

Each rite is one file: `<config>/rites/<name>.json` (or `.jsonc`; `switches/`
is searched as well). The file name *is* the rite's name. Comments and
trailing commas are tolerated, for the Omnissiah is merciful.

```jsonc
{
  "description": "Hide the mouse cursor after inactivity", // the rite's purpose
  "states": ["on", "off"],                                  // its aspects
  "files": [                                                // its vessels
    {
      "file": "~/.config/niri/util.kdl", // ~ and $VARS expand; relative = Librarium
      "guard": "mouse-autohide",         // the ward; default: the rite's name
      "comment": "//",                   // default: divined from the extension
      "comment_end": "",                 // for block comments, e.g. "*/"
      "create": false,                   // consecrate the vessel if absent
      "values": [                        // one scripture per aspect
        {"state": "on", "value": "cursor {\n    hide-after-inactive-ms 400\n}",
         "meta": {"example-key": "example-value"}},
        {"state": "off", "value": ["or", "a list of lines"],
         "meta": {"example-key": "other-value", "reason": ""}}
      ],
      "meta": {                          // the inscriptions → --<key> runes
        "reason": {"optional": "true"},  // optional by default; bool or "true"/"false"
        "example-key": {"optional": false, "description": "rune help text"}
      }
    }
  ]
}
```

Comment glyphs divined by extension: `//` for kdl, js/ts, c/go/rust, scss,
qml…; `--` for lua/sql; `/* */` for css; `<!-- -->` for html/xml/svg/md; `;`
for ini; `#` for all else.

See [`examples/rites/mouse-autohide-toggle.json`](examples/rites/mouse-autohide-toggle.json).

## The Rituals

| Ritual | Purpose |
|---|---|
| `servitor invoke <rite> <aspect> [--<key> v] [-f] [-s]` | Perform an aspect. `-f/--foresee` divines the diff without touching a vessel; `-s/--silence` performs in reverent silence. `servitor invoke <rite> --help` reveals its aspects, vessels and runes. |
| `servitor augury <rite>` | The inscriptions as JSON, `state` first. |
| `servitor augury <rite> <key>` | One inscription (an empty line when uninscribed). |
| `servitor augury <rite> --is <aspect>` | Exit 0 if the rite stands in that aspect, 1 if not. |
| `servitor augury <rite> --per-file` | Every vessel's sanctum as JSON (presence, inscriptions, taint). |
| `servitor census [--binharic]` | All rites and the aspects they stand in. |
| `servitor inquisition [rite…] [--binharic] [--spare-vessels]` | Purge heresy from rites and vessels. |
| `servitor cogitator` | The interactive shrine (also the default). |
| `servitor completion bash\|zsh\|fish` | Engrave completion litanies. |

Augury exit codes: `0` success, `1` `--is` did not match, `2` the augury failed
(unknown rite, never performed, or vessels that disagree).

### The Inquisition

Reports each heresy as `file:line:column: heresy: message` and exits 1 when
any is found. Impurities are noted but forgiven. It purges:

* malformed JSON, unknown or mistyped fields
* missing, duplicated or undeclared aspects; aspects without scripture
* undeclared, malformed or reserved inscriptions (`state`, `help`, `librarium`,
  `foresee`, `silence`, `version`)
* rites whose names clash (`rites/x.json` beside `switches/x.jsonc`) and wards
  claimed twice in the same vessel
* vessels: missing (unless `create`), sanctums with broken or duplicated wards
  (heresy), and scripture tainted by unsanctioned hands (impurity)

## For Tech-Priests

```sh
go test -race -cover ./...
golangci-lint run ./...
```

Commits follow the [Liturgy of Commits](CONTRIBUTING.md): Conventional Commits
spoken in the tongue of the Mechanicus (`consecrate`, `purge`, `tithe`, …).

`internal/block` (wards and inscriptions), `internal/config` (the Librarium:
loading, validation, diagnostics, marshalling), `internal/engine` (invoking,
augury, purging, inspecting vessels), `internal/lexicon` (both vocabularies and
the Thoughts for the Day), `internal/cli` (cobra rituals and completion),
`internal/tui` (the cogitator: bubbletea v2, bubbles, lipgloss).

## License

[MIT](LICENSE). The Omnissiah shares freely; so may you.

*The Librarium forgets nothing that git remembers.*
