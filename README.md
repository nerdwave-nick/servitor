# servitor

> *+++ Thought for the day: A file unwarded is a file defiled. +++*

**servitor** is a thrall of the Adeptus Mechanicus, bound to the scripture of
your machine. You teach it **rites**. Each rite knows a fixed set of
**aspects** and one **liturgy**: an ordered list of steps that rewrite warded
**sanctums** inside your vessels, transcribe whole vessels, re-bind
**tethers**, speak **incantations**, recite **litanies** and send
**vox-casts**. Invoke an aspect, and the servitor performs the liturgy verse
by verse. Should one verse fall, it undoes every deed already done. It asks
not *why*. It asks only *which aspect*.

```sh
servitor                                                  # awaken the cogitator
servitor invoke mouse-autohide-toggle on --reason "gaming remnant"
servitor invoke theme "$(printf '%s\n' default finii porpl | fuzzel --dmenu)"
servitor invoke theme porpl --foresee                     # divine, perform nothing
servitor augury theme | jq -r .aspect                     # porpl
servitor augury mouse-autohide-toggle --is on || echo "the cursor walks unveiled"
servitor expound tether                                   # as written in the codex
```

The servitor speaks only the tongue of the Mechanicus: its runes, keys,
markers, binharic and laments alike. Where a word is strange to you, ask the
codex: `servitor expound` lists every passage, and `servitor expound <topic>`
recites one at length, with exempla. This scripture is the short catechism;
the codex is the whole of the law.

## Installation

The forge requires Go 1.27 or newer.

```sh
go install github.com/nerdwave-nick/servitor@latest
# or, from a checkout of this repository:
go build -o ~/.local/bin/servitor .
servitor --version
```

### Completion litanies

Completion knows the rituals, every rite, its aspects (the current one is
marked), its inscription runes and the values they decree, augury keys and
the topics of the codex. It never awakens an auspex, so every Tab is answered
at once.

```sh
# bash (the bash-completion package must be installed)
servitor completion bash > ~/.local/share/bash-completion/completions/servitor
# zsh (any hall on $fpath, with compinit called)
servitor completion zsh > "${fpath[1]}/_servitor"
# fish
servitor completion fish > ~/.config/fish/completions/servitor.fish
```

`servitor expound completion` recites the rest.

## The Librarium

> *+++ Thought for the day: The Librarium forgets nothing that git remembers. +++*

The Librarium is the hall where your rites are kept:
`$XDG_CONFIG_HOME/servitor` (`~/.config/servitor`) unless the rune
`--librarium <hall>` (`-l`) or `SERVITOR_LIBRARIUM` names another; the rune
outranks the environment.

```
~/.config/servitor/
├── servitor.json             the settings (optional)
└── rites/
    ├── theme.json            the rite "theme"
    ├── mouse-autohide-toggle.json
    └── scripts/recite-hooks  whatever your rites read or recite
```

Each rite is one scripture of JSON in `rites/`, named `<rite>.json` (or
`.jsonc`): the name of the scripture *is* the name of the rite. Remarks
(`//`, `/* */`) and trailing commas are tolerated, for the Omnissiah is
merciful. Only `rites/` is read.

### The settings

`servitor.json` beside `rites/` holds the standing orders of the servitor.
Every key is optional except the pattern:

```jsonc
{
  "$schema": "https://raw.githubusercontent.com/nerdwave-nick/servitor/main/schema/settings.schema.json",
  "pattern": "Mark I",
  "tongue": "bash",          // the tongue of incantations and litanies
  "patience": "30s",         // how long a command may labour
  "vox": "notify-send",      // or "off": the desktop hears no vox-cast
  "chronicle": "~/.local/state/servitor/chronicle.jsonl",
}
```

For each order the rune outranks the environment, the environment outranks
the settings, and the settings outrank the servitor's own default.

## Anatomy of a rite

> *+++ Thought for the day: Heresy grows from changes no Magos has examined. +++*

The rite of the machine's visage, from [`examples/rites/theme.json`](examples/rites/theme.json),
with its remarks set aside:

```jsonc
{
  "$schema": "https://raw.githubusercontent.com/nerdwave-nick/servitor/main/schema/rite.schema.json",
  "pattern": "Mark I",
  "purpose": "The visage of the machine",
  "aspects": ["default", "finii", "porpl"],
  "inscriptions": {
    "reason": {"purpose": "why the visage was changed"},
  },
  "liturgy": [
    {"incantation": "niri msg action do-screen-transition --delay-ms 200 || true"},
    {"tether": "~/.local/share/nfluff/current-theme", "anchor": "~/.config/nfluff/themes/{{aspect}}"},
    {"vox-cast": "progress"},
    {
      "litany": "scripts/recite-hooks",
      "offerings": ["~/.config/nfluff/themes/{{aspect}}"],
      "patience": "1m",
      "reversion": "test -z \"$SERVITOR_FORMER_ASPECT\" || scripts/recite-hooks ~/.config/nfluff/themes/\"$SERVITOR_FORMER_ASPECT\"",
    },
    {"vox-cast": "success"},
  ],
}
```

| Key | What it holds |
|---|---|
| `pattern` | The Mark of the form the scripture is written in: `"Mark I"`. Required. Scripture without a pattern, or of a Mark this servitor was not forged for, is heresy; elder scripture is never read nor converted. |
| `purpose` | Why the rite exists, in words for the faithful. |
| `aspects` | The aspects the rite can bring the machine into, written by hand: not empty, never repeated. |
| `inscriptions` | Named words carried by one invocation. Each may bear a `purpose`, be `mandatory`, and hold `decrees`: the word each aspect inscribes when none is given. |
| `auspex` | Optional words, spoken in the tongue, that print the aspect the machine bears. Short form `"<words>"`, long form `{"rite": "<words>", "patience": "5s"}`; its patience is 2s unless written. |
| `tongue` | The tongue of this rite's incantations and litanies, outranking the settings. |
| `liturgy` | The steps, performed in the order written. |
| `$schema` | Read by your editor, passed over by the servitor (see [The codex](#the-codex)). |

### Aspect maps

Any value of a step may be written once for every aspect, or as an object
keyed by aspect, with `"*"` serving every aspect that names no value of its
own. An aspect left with no value and no `"*"` is heresy.

```jsonc
{"incantation": {"on": "makoctl mode -a do-not-disturb",
                 "off": "makoctl mode -r do-not-disturb"}}
{"tether": "~/.config/foot/colors.ini",
 "anchor": {"dark": "~/.config/foot/night.ini", "*": null}}
```

### Inscriptions

Every inscription becomes a rune of the rite's own ritual,
`servitor invoke <rite> <aspect> --<key> <word>`, and the rune outranks the
decree; `--<key> ""` clears it. A `mandatory` inscription that neither rune
nor decree supplies forbids the invocation. The words of an invocation are
kept on the rite's data-slate (never in its scripture), read back by the
augury and kept in the chronicle.

```jsonc
"inscriptions": {
  "reason": {"purpose": "why the cursor's bearing was changed"},
  "mode": {"mandatory": true, "decrees": {"on": "auto-hide", "off": "always-visible"}},
}
```

```sh
servitor invoke mouse-autohide-toggle on --reason "gaming remnant"
servitor augury mouse-autohide-toggle mode        # auto-hide
```

### Placeholders and the environment

Every word a rite writes for its steps (paths, anchors, commands, scrolls,
offerings, reversions, inline scripture, illuminated tomes) and its auspex may
speak `{{aspect}}`, `{{former}}` (the aspect before; empty when unknown),
`{{rite.name}}` and `{{inscription.<key>}}` (empty when uninscribed). Write
`{{{{` for a literal `{{`. A mark the codex does not know, or an inscription
the rite never declared, is heresy, never silently left empty.

Incantations, litanies and the auspex also receive `SERVITOR_ASPECT`,
`SERVITOR_FORMER_ASPECT`, `SERVITOR_RITE` and `SERVITOR_INSCRIPTION_<KEY>`
(the key in capitals, `-` and `.` made `_`). **Within the words a tongue
will speak, prefer these:** a placeholder is pasted into the words before the
tongue hears them, so a quote, a `$` or a `;` inside an inscription would be
spoken as a command. `"$SERVITOR_INSCRIPTION_REASON"` stays one word,
whatever it holds.

```jsonc
// heeded: the reason reaches the tongue as one word
{"incantation": "echo \"$SERVITOR_INSCRIPTION_REASON\" >> ~/.cache/remnants"}
// perilous: a reason of  it's done; reboot  is spoken as commands
{"incantation": "echo '{{inscription.reason}}' >> ~/.cache/remnants"}
```

## The liturgy

> *+++ Thought for the day: Idle cursors breed heresy. +++*

Six kinds of step are known to the codex. A step is named by its own key,
which holds its vessel, its bound name, its words or its scroll.

| Step | Its key holds | Further keys |
|---|---|---|
| `sanctum` | the vessel whose warded region it keeps | `scripture`, `ward`, `glyph`, `closing-glyph`, `consecrate` |
| `transcription` | the vessel it writes whole, or strikes | `scripture` (`null` strikes the vessel), `seal`, `zeal` |
| `tether` | the name it binds | `anchor` (`null` unbinds), `zeal` |
| `incantation` | one line of words for the tongue | `tongue`, `patience`, `reversion` |
| `litany` | the scroll it recites | `offerings`, `tongue`, `patience`, `reversion` |
| `vox-cast` | `"progress"` or `"success"` | none |

Paths of vessels, tethers, anchors, scrolls and tomes expand a leading `~`
and `$NAMES`, and a path not rooted at `/` is reckoned from the `rites/` hall.
Missing halls above a vessel or tether are never raised: the pre-flight
denounces them.

### Sanctum

A sanctum rewrites only the region between its two markers; every line
outside them is inviolate. A vessel that bears no sanctum of its ward yet
receives one at its end.

```jsonc
{"sanctum": "~/.config/niri/util.kdl", "ward": "mouse-autohide", "consecrate": true,
 "scripture": {"on": ["cursor {", "    hide-after-inactive-ms 400", "}"], "off": ""}}
```

```kdl
// +++ begin of sanctum mouse-autohide -- aspect|on +++
cursor {
    hide-after-inactive-ms 400
}
// +++ end of sanctum mouse-autohide +++
```

`ward` defaults to the rite's name; `glyph` and `closing-glyph` are divined
from the vessel's extension unless written: `//` for kdl and the tongues of
C, `--` for lua and sql, `;` for ini, `/*` and `*/` for css, `<!--` and `-->`
for html, xml, svg and md, `#` for all else. `consecrate: true` brings a missing vessel into being. Empty
scripture keeps an empty sanctum; `null` scripture is heresy.

### Scripture and tomes

Scripture is written as one string, as a list of lines, or drawn from a tome,
a scripture-file beside the rite: `{"tome": "snippets/porpl.kdl"}`. A tome is
read afresh in every pre-flight and placed verbatim, unless
`"illuminate": true` bids its placeholders be filled.

### Transcription

A transcription writes the whole vessel. Because it casts down everything, it
replaces a vessel only when that vessel holds the scripture of one of the
rite's aspects, its own work. Anything else halts the pre-flight unless the
step burns with `"zeal": true`. A new vessel bears the seal `0644` unless
`seal` decrees another.

```jsonc
{"transcription": "~/.config/autostart/gamemode.desktop",
 "scripture": {"on": ["[Desktop Entry]", "Exec=gamemoded"], "off": null},
 "seal": "0600"}
```

### Tether

A tether binds a name to the anchor of the invoked aspect. It rebinds only a
tether: should a scripture-file stand in its place, the pre-flight refuses
unless the step burns with `zeal`.

```jsonc
{"tether": "~/.local/share/nfluff/current-theme", "anchor": "~/.config/nfluff/themes/{{aspect}}"}
```

### Incantation and litany

An incantation hands one line of words to its tongue (`<tongue> -c <words>`);
a litany recites a whole scroll, by its own shebang when the scroll may be
executed, else through its tongue. The tongue is bash unless the step, the
rite or the settings say otherwise. A litany receives nothing it is not
offered: its `offerings` are handed over one word each, exactly as written:
never split, never interpreted by a tongue, no `$NAME` expanded. Only their
placeholders are filled, and a leading `~/` is borne to the invoker's home.

```jsonc
{"incantation": "niri msg action load-config-file", "patience": "5s"}
{"litany": "scripts/recite-hooks", "offerings": ["~/.config/nfluff/themes/{{aspect}}"],
 "reversion": "test -z \"$SERVITOR_FORMER_ASPECT\" || scripts/recite-hooks ~/.config/nfluff/themes/\"$SERVITOR_FORMER_ASPECT\""}
```

* Both are spoken in the `rites/` hall, with nothing upon their input.
* A death-mark other than 0 makes the step fall. A failure you mean to
  tolerate is written into the words: `cmd || true`.
* `patience` (30s unless the settings or the step say otherwise) is how long
  the step may labour; when it runs out, the step is slain together with
  every process it summoned, and it falls.
* What they utter is captured, never printed; the lament of a fallen step
  shows its last words.
* Children sent into the background outlive their step, but the servitor
  listens to them only a moment longer. Send their words elsewhere:
  `swaybg -i ~/walls/{{aspect}}.png >/dev/null 2>&1 &`.
* Interactive commands are unsupported: no terminal is lent to them.
* `reversion` is one line of words, spoken in the tongue should the
  invocation fall, the fallen step's own reversion among them.

### Vox-cast

`{"vox-cast": "progress"}` tells of the kind of step performed last and how
far the liturgy has come; `{"vox-cast": "success"}` proclaims the
triumph, and belongs last. Tidings of a fall need no step. See
[Vox-casts](#vox-casts).

## Invocation

> *+++ Thought for the day: Excuses are the refuge of the untested. +++*

`servitor invoke <rite> <aspect> [--<inscription> <word>…] [-f] [-s]`

1. **Pre-flight.** Every placeholder is filled, every tome read, every vessel,
   tether, scroll and tongue weighed and every inscription resolved, without
   touching the machine. Any heresy forbids the invocation, and nothing is
   disturbed.
2. **Performance.** The steps are performed in the order written.
3. **Reversion.** Should verse *k* fall, verses *k* … 1 are undone in reverse
   order: sanctums, transcriptions and tethers are restored exactly as they
   were found (a vessel the rite brought into being is struck); incantations
   and litanies speak their `reversion`. A failing reversion does not halt
   the others. An interrupt from the terminal (Ctrl-C) halts and reverts the
   invocation the same way.
4. **Verdict.** `triumph` (every step performed), `reverted` (a step fell and
   all was undone) or `faltered` (a step fell and some reversion fell too;
   the rite then stands corrupted). The verdict is kept on the rite's
   data-slate, `$XDG_STATE_HOME/servitor/data-slates/<rite>.json`, and in
   the chronicle.

Spoken at a terminal, the invocation is told in one report as it is
performed: a header with the rite and its turning, a line for every verse
once it is performed (the home written `~`, a long verse cut short with `…`
to the terminal's width), every progress vox-cast in its place with the
count of steps done, and one closing line of triumph, drawn from the rite's
success vox-cast or, should it have none, from the same litany:

```console
❯ servitor invoke mouse-autohide-toggle on
+++ mouse-autohide-toggle · off → on +++
  ✔ verse 1 · sanctum ~/.config/niri/util.kdl
  ✔ verse 2 · incantation niri msg action load-config-file
✠ The Omnissiah is pleased: the rite now bears the aspect «on».

❯ servitor invoke theme porpl
+++ theme · default → porpl +++
  ✔ verse 1 · incantation niri msg action do-screen-transition --delay-ms 200 |…
  ✔ verse 2 · tether ~/.local/share/nfluff/current-theme
  ⋯ Incense burns; the tether is complete.  (2/3)
  ✔ verse 4 · litany scripts/recite-hooks
✠ Binharic hymns resound: the aspect «porpl» is attained.
```

Should a verse fall, the report ends with the last verse performed and the
lament alone tells the fall, its last words and every reversion.

`--foresee` (`-f`) shows every sanctum and transcription as it would change,
every tether rebound and every command as it would be spoken, and performs
nothing, not even the auspex. `--silence` (`-s`) withholds the report, save
the lines the rite's own vox-casts ask for. The ritual exits 0 upon triumph
and 1 otherwise.

## Rituals and runes

| Ritual | What it does |
|---|---|
| `servitor invoke <rite> <aspect> [--<key> <word>] [-f] [-s]` | Perform an aspect of a rite. |
| `servitor augury <rite> [key] [--is <aspect>]` | Read the aspect a rite stands in, its standing and its inscriptions. |
| `servitor census [--binharic]` | Every rite with its aspect, standing, aspects, last rite and purpose. |
| `servitor inquisition [rite…] [--binharic] [--spare-vessels]` | Denounce heresy in rites, settings and vessels. |
| `servitor cogitator` | Awaken the interactive shrine; `servitor` alone does so at a terminal. |
| `servitor expound [topic] [--schema [rite\|settings]]` | Recite a passage of the codex, or its index. |
| `servitor completion bash\|zsh\|fish\|powershell` | Engrave the completion litanies. |

Universal runes: `--librarium <hall>` / `-l` (env `SERVITOR_LIBRARIUM`),
`--chronicle <path>` (env `SERVITOR_CHRONICLE`), and `--version` / `-v` for
the servitor alone.

### Augury

> *+++ Thought for the day: The Omnissiah knows all. Lesser beings must consult the manual. +++*

Without a key the augury speaks one binharic object:

```json
{"rite": "theme", "aspect": "porpl", "standing": "performed",
 "desecrated": false,
 "inscriptions": {"reason": "the night watch begins"},
 "last_rite": {"verdict": "triumph", "at": "2026-10-10T13:30:00Z"},
 "omens": [{"verse": 2, "tether": "~/.local/share/nfluff/current-theme", "aspect": "porpl"}],
 "taint": []}
```

With a key it speaks one value: `aspect`, `standing`, `desecrated`, `former`,
or any inscription the rite declares (an empty line when uninscribed).
`--is <aspect>` answers by exit alone. The augury exits 0 when it was read,
1 when `--is` did not match, and 2 when it could not be read.

```sh
servitor augury theme reason
[ "$(servitor augury mouse-autohide-toggle aspect)" = on ] && echo "the cursor is veiled"
```

The servitor does not trust its memory alone: it reads **omens** from the
machine. A sanctum bears witness through the aspect in its marker, a tether
through the anchor it leads to, a transcription through the scripture its
vessel holds, and the auspex through what it prints. Incantations and
litanies leave nothing to be read. From the omens the **standing** is judged:

| Standing | Meaning |
|---|---|
| `performed` | the omens agree on one aspect, or none speaks and the data-slate names it |
| `dormant` | neither omen nor data-slate names an aspect |
| `corrupted` | the omens disagree, or the last invocation faltered |
| `tainted` | other hands altered the scripture within a sanctum |
| `desecrated` | the omens agree on an aspect the data-slate did not record; `former` names the remembered one |
| `heretical` | the rite's scripture is invalid |

Desecration is no heresy. When do-not-disturb is raised from your bar, an
auspex lets the augury report the true aspect and mark the rite desecrated;
the next invocation through the servitor cleanses it.

```jsonc
"auspex": "makoctl mode | grep -q do-not-disturb && echo on || echo off"
```

### Census

```sh
servitor census
servitor census --binharic | jq -r '.[] | select(.standing != "performed") | .rite'
```

In binharic every rite bears `rite`, `aspect`, `standing`, `aspects`,
`purpose`, `desecrated`, `last_rite` and `recorded_in`, the scripture it was
read from.

### Inquisition

The Inquisition denounces every **heresy** (the rite cannot be invoked) and
notes every **impurity** (forgiven), each with its place as
`scripture:line:column` and a verbose account of the sin and its penance. It
exits 1 when any heresy was found.

```sh
servitor inquisition
servitor inquisition theme --spare-vessels     # judge the scriptures alone
servitor inquisition --binharic | jq -r '.[] | select(.judgement == "heresy") | .denunciation'
```

`servitor expound heresy` lists every sin it knows.

## Vox-casts

> *+++ Thought for the day: Death to the false positive! +++*

Invoked from a terminal, every vox-cast is told in its place within the
report of the invocation (see [Invocation](#invocation)): a progress as
`⋯ <words>  (x/n)`, the triumph as the closing `✠ <words>`. Invoked from a
hotkey or a launcher, with no controlling terminal, the vox-casts of one
invocation become **one** missive upon the desktop through `notify-send`,
headed `theme → porpl` and replaced in place by the next: progress tells
"step x / n" and lingers until the rite ends, triumph fades as the desktop
wills, and a fall is proclaimed with critical urgency, unbidden; the report
follows on stdout once the rite has triumphed. The words are drawn at
random from the servitor's own litanies of progress, triumph and lament,
and speak of "the aspect «porpl»", never of a bare aspect. Should `notify-send`
be absent, the servitor stays silent rather than fail; `"vox": "off"` in the
settings silences the desktop altogether. In the cogitator the tidings are
shown in the cogitator itself.

## The chronicle

Every performed invocation adds one line of binharic to the chronicle,
whatever its verdict:

```json
{"at": "2026-10-10T13:30:00Z", "rite": "theme", "aspect": "porpl", "former": "default",
 "inscriptions": {"reason": "rain"}, "verdict": "reverted",
 "fell_at": {"verse": 4, "litany": "/home/you/.config/servitor/rites/scripts/recite-hooks"},
 "heresy": "it ended bearing the death-mark 1, a sign that its work was not done",
 "reversions": [{"verse": 4, "verdict": "triumph"}, {"verse": 2, "verdict": "triumph"}]}
```

It is kept at the first of: the rune `--chronicle`, `SERVITOR_CHRONICLE`, the
settings' `chronicle`, `$XDG_STATE_HOME/servitor/chronicle.jsonl`. Past one
mebibyte it is renamed with the suffix `.1` and a fresh chronicle begins. An
invocation forbidden by its pre-flight, or only foreseen, is not chronicled.

```sh
tail -n 3 ~/.local/state/servitor/chronicle.jsonl | jq -r '.rite + " → " + .aspect + ": " + .verdict'
```

## The cogitator

> *+++ Thought for the day: Every unsaved buffer is a soul unshriven. +++*

`servitor` alone at a terminal, or `servitor cogitator`, awakens the
cogitator, a keyboard-driven shrine: the rites on the left with the glyph of
their standing; on the right the chosen rite's purpose, aspects (the current
one marked), standing, last rite, inscriptions, its liturgy with one omen per
step, and where its scripture is recorded; a Thought for the Day below.

| Key | Rite |
|---|---|
| `↑`/`k` `↓`/`j`, `g` `G` | choose a rite; the first, the last |
| `enter` | invoke: choose the aspect, amend the inscriptions, `p` to foresee |
| `space` | cycle the rite to its next aspect at once |
| `n` | consecrate a new rite |
| `e` | amend the rite |
| `c` | replicate the rite |
| `d` | excommunicate the rite: `y` strikes its scripture, `p` its sanctums as well |
| `o` | open the scripture in `$EDITOR` |
| `i` | summon the Inquisition |
| `O` | the words of the last invocation |
| `h` | read the chronicle, filtered to the chosen rite |
| `/` | filter |
| `r` | re-read the Librarium |
| `?` | expound the codex |
| `q` | retreat |

While a rite is invoked, a panel shows its vox-casts, "step x / n" and the
verse being performed; the words of incantations and litanies are captured,
never printed, and a fall opens its lament with every reversion and the last
words of the fallen step.

The consecration wizard (`n`, `e`, `c`) leads a rite through three stations:
the **Rite** (name, purpose, aspects, inscriptions, auspex), the **Liturgy**
(`a` adds a step of any kind, `J`/`K` reorder, a page for each aspect where
values differ, "same for every aspect" writes `"*"`, rarer keys rest under
the further rites) and the **Seal**, where the scripture stands as it will be
written, judged by the Inquisition before anything is written. A heretical
rite cannot be sealed. In forms, `tab`/`↑↓` move between fields, `enter`
advances, `ctrl+s` seals the page and `esc` steps back.

## The codex

```sh
servitor expound                  # the index of every passage
servitor expound litany           # one passage, as written in the codex
servitor expound --schema         # the schema of a rite
servitor expound --schema settings
```

Every ritual, rune, step and key has its passage, with exempla. The JSON
Schemas of a rite and of the settings dwell in [`schema/`](schema/); name
them under `"$schema"` and your editor completes and judges every key, with
the codex's own lore:

```jsonc
"$schema": "https://raw.githubusercontent.com/nerdwave-nick/servitor/main/schema/rite.schema.json"
```

## Exempla

[`examples/`](examples/) is a Librarium of its own:

* [`rites/mouse-autohide-toggle.json`](examples/rites/mouse-autohide-toggle.json)
  keeps a sanctum in niri's scripture, bids niri read it anew and proclaims
  the triumph.
* [`rites/theme.json`](examples/rites/theme.json) veils the screen in a
  transition, re-tethers the nfluff theme, recites the theme's hooks through
  [`rites/scripts/recite-hooks`](examples/rites/scripts/recite-hooks) and
  vox-casts its progress and triumph.

```sh
servitor --librarium examples inquisition --spare-vessels
servitor --librarium examples invoke theme porpl --foresee
```

## For Tech-Priests

```sh
go test -race ./...
go vet ./...
golangci-lint run ./...
```

Commits follow the [Liturgy of Commits](CONTRIBUTING.md): Conventional
Commits spoken in the tongue of the Mechanicus (`consecrate`, `purge`,
`transcribe`, …). The lexicon of the domain is [`CONTEXT.md`](CONTEXT.md);
the decisions are recorded in [`docs/adr/`](docs/adr/).

| Package | Its charge |
|---|---|
| `internal/librarium` | reading, examining and writing rites and the settings |
| `internal/placeholder` | illuminating and judging the `{{…}}` marks |
| `internal/sanctum` | finding and rewriting sanctums within a vessel |
| `internal/invocation` | the pre-flight, the performance and the reversion |
| `internal/augury` | omens, standing and the data-slates |
| `internal/chronicle` | the chronicle |
| `internal/vox` | the herald of vox-casts |
| `internal/rituals` | every ritual upon one Librarium, for the rituals and the cogitator alike |
| `internal/cli` | the rituals, runes and completion litanies |
| `internal/tui` | the cogitator |
| `internal/codex` | the passages of `servitor expound` |
| `internal/lexicon` | the Thoughts for the Day |
| `schema` | the JSON Schemas of pattern Mark I |

## License

[MIT](LICENSE). The Omnissiah shares freely; so may you.

*The Librarium forgets nothing that git remembers.*
