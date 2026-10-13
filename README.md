# servitor

> *+++ Thought for the day: A file unwarded is a file defiled. +++*

**servitor** is a thrall of the Adeptus Mechanicus, bound to the scripture of
your machine. You teach it **rites**. A rite knows a fixed set of
**aspects**, or none at all if it is an **edict**, and one **liturgy**: an
ordered list of steps that rewrite warded **sanctums** inside your vessels,
transcribe whole vessels, re-bind **tethers**, speak **incantations**, recite
**litanies** and send **vox-casts**. Invoke an aspect, and the servitor
performs the liturgy verse by verse; proclaim an edict, and it performs the
liturgy whole, every time. Should one verse fall, it undoes every deed
already done. It asks not *why*. It asks only *which aspect*, and of an
edict not even that.

```sh
servitor                                                  # awaken the cogitator
servitor invoke mouse-autohide-toggle on --reason "gaming remnant"
servitor invoke theme "$(printf '%s\n' default finii porpl | fuzzel --dmenu)"
servitor invoke theme porpl --foresee                     # divine, perform nothing
servitor invoke screenshot --region whole                 # an edict, by its name alone
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

### By pacman

Exalted verses are performed only by the servitor at `/usr/bin/servitor`,
owned by root and writable by none but root. On Arch Linux, pacman installs it
so. From a checkout of this repository:

```sh
cd packaging/arch
makepkg -si
servitor --version
```

The PKGBUILD forges the checkout as it stands, with
`-buildmode=pie -trimpath -mod=readonly`, and puts it to its trials
(`go test`) before pacman takes it. The package holds `/usr/bin/servitor` and
the completion litanies for bash, zsh and fish, so the engraving below is not
needed. After each pull, forge and install anew with `makepkg -si`.

### By the Go forge alone

A servitor forged by Go alone performs every rite that holds no exalted verse.
For exalted verses it rouses the servitor installed by pacman, and only while
both speak one protocol; otherwise it bids you install the servitor anew.

```sh
go install github.com/nerdwave-nick/servitor@latest
# or, from a checkout of this repository:
go build -o ~/.local/bin/servitor .
servitor --version
```

### Completion litanies

Completion knows the rituals, every rite, its aspects (the current one is
marked; an edict is offered none), its inscription runes and the values they
decree, augury keys and the topics of the codex. It never awakens an auspex,
so every Tab is answered at once.

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
  "vox": "auto",             // "notify-send": the desktop hears every invocation; "off": none
  "chronicle": "~/.local/state/servitor/chronicle.jsonl",
  "intercessor": "auto",     // who asks the proof for exalted verses: "sudo", "run0" or "pkexec"
}
```

For each order the rune outranks the environment, the environment outranks
the settings, and the settings outrank the servitor's own default. The
`intercessor` knows no rune and no environment: the settings alone name it
(see [Exalted verses](#exalted-verses)).

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
| `aspects` | The aspects the rite can bring the machine into, written by hand: not empty, never repeated. A rite bears either `aspects` or `"edict": true`, never both and never neither. |
| `edict` | `true` makes the rite an [edict](#edicts), which knows no aspects; `false` is as if unwritten. |
| `inscriptions` | Named words carried by one invocation. Each may bear a `purpose`, be `mandatory`, and hold `decrees`: the word each aspect inscribes when none is given (an edict's decree is one plain word). |
| `auspex` | Optional words, spoken in the tongue, that print the aspect the machine bears. Short form `"<words>"`, long form `{"rite": "<words>", "patience": "5s"}`; its patience is 2s unless written. An edict bears none. |
| `tongue` | The tongue of this rite's incantations and litanies, outranking the settings. |
| `liturgy` | The steps, performed in the order written. |
| `$schema` | Read by your editor, passed over by the servitor (see [The codex](#the-codex)). |

### Aspect maps

Any value of a step may be written once for every aspect, or as an object
keyed by aspect, with `"*"` serving every aspect that names no value of its
own. An aspect left with no value and no `"*"` is heresy. An edict knows no
aspect map: each of its values is written once, plainly, and even an object
keyed by `"*"` alone is heresy there.

```jsonc
{"incantation": {"on": "makoctl mode -a do-not-disturb",
                 "off": "makoctl mode -r do-not-disturb"}}
{"tether": "~/.config/foot/colors.ini",
 "anchor": {"dark": "~/.config/foot/night.ini", "*": null}}
```

### Inscriptions

Every inscription becomes a rune of the rite's own ritual,
`servitor invoke <rite> <aspect> --<key> <word>` (of an edict,
`servitor invoke <edict> --<key> <word>`), and the rune outranks the decree;
`--<key> ""` clears it. A `mandatory` inscription that neither rune nor
decree supplies forbids the invocation. The words of an invocation are
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
the rite never declared, is heresy, never silently left empty. An edict has
no aspect to speak: in its words `{{aspect}}` and `{{former}}` are heresy.

Incantations, litanies and the auspex also receive `SERVITOR_ASPECT`,
`SERVITOR_FORMER_ASPECT`, `SERVITOR_RITE` and `SERVITOR_INSCRIPTION_<KEY>`
(the key in capitals, `-` and `.` made `_`); an edict's receive no
`SERVITOR_ASPECT` and no `SERVITOR_FORMER_ASPECT`. **Within the words a tongue
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
| `sanctum` | the vessel whose warded region it keeps | `scripture`, `ward`, `glyph`, `closing-glyph`, `consecrate`, `exalted` |
| `transcription` | the vessel it writes whole, or strikes | `scripture` (`null` strikes the vessel), `seal`, `zeal`, `exalted` |
| `tether` | the name it binds | `anchor` (`null` unbinds), `zeal`, `exalted` |
| `incantation` | one line of words for the tongue | `tongue`, `patience`, `reversion`, `exalted`, `relics` |
| `litany` | the scroll it recites | `offerings`, `tongue`, `patience`, `reversion`, `exalted`, `relics` |
| `vox-cast` | `"progress"` or `"success"` | none |

`"exalted": true` bids the machine's own authority perform the step, as its
master sanctioned it; `relics` names what exalted speech reads beside it.
See [Exalted verses](#exalted-verses).

An edict's liturgy holds only incantations, litanies and vox-casts:
sanctums, transcriptions and tethers keep and reveal an aspect, and an edict
has none.

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

`servitor invoke <rite> <aspect> [--<inscription> <word>…] [-f] [-s]`, or
`servitor invoke <edict> [--<inscription> <word>…] [-f] [-s]` for an
[edict](#edicts)

1. **Pre-flight.** Every placeholder is filled, every tome read, every vessel,
   tether, scroll and tongue weighed and every inscription resolved, without
   touching the machine. Any heresy forbids the invocation, and nothing is
   disturbed.
2. **Performance.** The steps are performed in the order written.
3. **Reversion.** Should verse *k* fall, verses *k* … 1 are undone in reverse
   order: sanctums, transcriptions and tethers are restored exactly as they
   were found (a vessel the rite brought into being is struck); incantations
   and litanies speak their `reversion`. A failing reversion does not halt
   the others. An interrupt from the terminal (Ctrl-C) or SIGTERM halts and
   reverts the invocation the same way; a hangup does not.
4. **Verdict.** `triumph` (every step performed), `reverted` (a step fell and
   all was undone) or `faltered` (a step fell and some reversion fell too;
   the rite then stands corrupted). The verdict is kept on the rite's
   data-slate, `$XDG_STATE_HOME/servitor/data-slates/<rite>.json`, and in
   the chronicle.

Spoken at a terminal, the invocation is told in one report as it is
performed: a header with the rite and its turning (an edict's header names
it alone), a line for every verse
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
the lines the rite's own vox-casts ask for.

`--binharic` is for lesser machines: stdout then bears one line of
binharic alone, the very line the invocation added to the
[chronicle](#the-chronicle), while the report and every lament go to stderr
(the desktop hears as it always would). Should the pre-flight forbid the
invocation, the line names the heresies under `forbidden` instead; an
edict's names no aspect. `--binharic` is not spoken with `--foresee`.

```console
❯ servitor invoke theme porpl --binharic 2>/dev/null
{"at":"2026-10-11T13:30:00Z","rite":"theme","aspect":"porpl","former":"default","inscriptions":{},"verdict":"triumph"}

❯ servitor invoke theme porpl --binharic 2>/dev/null
{"rite":"theme","aspect":"porpl","forbidden":["…/rites/theme.json:9:7: heresy: …"]}
```

The death-mark tells how it ended, with or without `--binharic`:

| Death-mark | The invocation |
|---|---|
| 0 | triumphed |
| 1 | was reverted: a step fell and every deed was undone |
| 2 | was refused before anything was touched: the pre-flight forbade it, the rite or aspect is unknown, words strayed after an edict, a rune was malformed, or — for [exalted verses](#exalted-verses) — the sanction lapsed or was never laid, no intercessor answered, or the master's proof was dismissed |
| 3 | faltered: a step fell and a reversion fell too |

### Invoked by machines that do not wait

A picker, a launcher or a bar that invokes a rite and departs at once
launches the servitor detached, as fuzzel launches its programs: stdin,
stdout and stderr upon `/dev/null`, a session of its own (`setsid`), and,
should the caller's own cgroup or scope be slain whole, a launch prefix
such as `systemd-run --user --scope` or `uwsm app --`. The death of its
caller never slays the servitor, and the [vox](#vox-casts) tells how the
invocation ended.

```console
❯ setsid -f servitor invoke theme porpl </dev/null >/dev/null 2>&1
```

The servitor stands steadfast besides: for the whole of an invocation, from
the ritual invoke as from the cogitator, it heeds no hangup, and words lost
upon a stdout or stderr whose reader is gone never touch the liturgy, its
reversion, the data-slate or the chronicle. An interrupt and SIGTERM still
halt and revert it. SIGKILL, and a slain cgroup without a launch prefix, it
cannot survive. `servitor expound detached` tells the whole of it.

## Edicts

> *+++ Thought for the day: If a task is worth doing, it is worth a servitor. +++*

Some deeds bring the machine into no aspect at all: its visage captured, the
machine spirit bidden to sleep or to wake anew. Written as a rite, such a
deed would demand a hollow aspect on every invocation
(`servitor invoke screenshot taken`), report a standing that says nothing
true, and let the cogitator perform it on a single key. Write it instead as
an **edict**, with `"edict": true` in place of `aspects`. From
[`examples/rites/screenshot.json`](examples/rites/screenshot.json), its
remarks set aside:

```jsonc
{
  "$schema": "https://raw.githubusercontent.com/nerdwave-nick/servitor/main/schema/rite.schema.json",
  "pattern": "Mark I",
  "purpose": "Capture the visage of the machine",
  "edict": true,
  "inscriptions": {
    "region": {"purpose": "\"chosen\" lets slurp mark the region, \"whole\" takes every output", "decrees": "chosen"},
  },
  "liturgy": [
    {
      "incantation": "if [ \"$SERVITOR_INSCRIPTION_REGION\" = whole ]; then grim ~/pictures/visage.png; else grim -g \"$(slurp)\" ~/pictures/visage.png; fi",
      "patience": "2m",
      "reversion": "rm -f ~/pictures/visage.png",
    },
    {"incantation": "wl-copy --type image/png < ~/pictures/visage.png"},
    {"vox-cast": "success"},
  ],
}
```

Choose an edict when the liturgy is a command to be carried out, not an
aspect to be kept: when a second invocation should do the work again, and no
omen could ever tell what the machine bears. Choose a rite of aspects when
the machine is brought into a condition it keeps, and you will later ask
which one.

* **Its scripture.** A rite bears exactly one of `aspects` and
  `"edict": true`; both, or neither, is heresy, and `"edict": false` is as
  if unwritten. An edict's liturgy holds only incantations, litanies and
  vox-casts, with their `reversion`, `patience`, `tongue` and `offerings`.
  It bears no auspex and no aspect map (its decrees are one plain word), and
  never speaks `{{aspect}}` or `{{former}}`.
* **Its proclaiming.** An edict is invoked by its name alone; any word
  after its name is refused before a verse is performed. Every invocation
  performs the whole liturgy anew, with its pre-flight, reversion, patience
  and foresight as for any rite. Should the second verse above fall, the
  first verse's reversion strikes the captured visage.
* **Its standing.** An edict yields no omen and is judged by its last
  verdict alone: `dormant` until first proclaimed, `proclaimed` after a
  triumph or a reversion, `corrupted` after a faltered invocation until it
  is proclaimed again, `heretical` when its scripture is. It is never
  `performed`, `tainted` or `desecrated`.
* **Its augury and census.** The augury speaks its standing, inscriptions
  and last rite, never an aspect, and refuses `--is`; the census shows `·`
  for its aspect and its aspects. Its data-slate and its chronicle lines
  bear no aspect.
* **The cogitator asks first.** `enter` and `space` alike open its invoke
  screen: its inscriptions, should it bear any, then a confirmation. Only a
  second `enter` proclaims it; `p` foresees, `esc` retreats. No single key
  ever proclaims an edict.

```console
❯ servitor invoke screenshot --region whole
+++ screenshot · edict +++
  ✔ verse 1 · incantation if [ "$SERVITOR_INSCRIPTION_REGION" = whole ]; then g…
  ✔ verse 2 · incantation wl-copy --type image/png < ~/pictures/visage.png
✠ Every verse is sung; the edict screenshot is carried out.

❯ servitor augury screenshot standing
proclaimed

❯ servitor invoke screenshot taken
servitor ✠ the edict "screenshot" bears no aspect and is proclaimed by its name alone, yet "taken" followed its name; not one verse of its liturgy is performed
```

`servitor expound edict` recites the rest.

## Exalted verses

> *+++ Thought for the day: The flesh may ask; only the sanctioned may command. +++*

Some steps reach beyond what you may touch: the knobs of the kernel, the
scriptures of `/etc`, the units of the machine itself. Write `"exalted": true`
upon such a step, and it is performed not with your authority but with the
machine's own, by an **exalted servitor** raised for that one invocation, and
only as you **sanctioned** it. Every other verse is performed as ever. From
[`examples/rites/split-lock.json`](examples/rites/split-lock.json), its
remarks set aside:

```jsonc
{
  "$schema": "https://raw.githubusercontent.com/nerdwave-nick/servitor/main/schema/rite.schema.json",
  "pattern": "Mark I",
  "purpose": "Lift the kernel's split-lock penance while the faithful play",
  "aspects": ["on", "off"],
  "auspex": "[ \"$(sysctl -n kernel.split_lock_mitigate)\" = 0 ] && echo on || echo off",
  "liturgy": [
    {
      "incantation": {"on": "sysctl -wq kernel.split_lock_mitigate=0",
                      "*":  "sysctl -wq kernel.split_lock_mitigate=1"},
      "exalted": true,
      "reversion": "if [ \"$SERVITOR_FORMER_ASPECT\" = on ]; then sysctl -wq kernel.split_lock_mitigate=0; else sysctl -wq kernel.split_lock_mitigate=1; fi",
    },
    {
      "transcription": "/etc/sysctl.d/90-split-lock.conf",
      "exalted": true,
      "scripture": {"on": "kernel.split_lock_mitigate = 0\n", "off": null},
    },
    {"vox-cast": "success"},
  ],
}
```

Exalted verses need the servitor installed by pacman at `/usr/bin/servitor`
(see [Installation](#installation)): it alone is ever raised, never the
servitor that asks, and only while it and every hall above it are the root's
and written by none but the root.

### Writing exalted verses

* Any step but a vox-cast may be exalted, in rites and edicts alike (an edict
  exalts its incantations and litanies). `exalted` is true for every aspect or
  for none, never an aspect map; `false` is as if unwritten.
* The words of an exalted verse are fixed when you sanction them. They may
  speak `{{aspect}}`, `{{rite.name}}` and `{{inscription.<key>}}`, the
  inscription then taking its decree alone (its rune is refused for the
  rite). `{{former}}` is heresy there: exalted speech hears the former aspect
  and the words of runes from `"$SERVITOR_FORMER_ASPECT"` and
  `"$SERVITOR_INSCRIPTION_<KEY>"`, each at most 4 KiB, without NUL or new
  line.
* An exalted vessel or bound name must begin at the root once rendered, may
  not lie beneath `/proc`, `/sys` or `/dev` (their knobs are written by
  exalted incantations), and every hall along it must be the root's alone. An
  exalted tether's anchor too: a tether from `/etc` into your home is heresy,
  and an exalted transcription is the remedy.
* Exalted speech is spoken in a fresh hall of the root's own beneath
  `/run/servitor`, with `HOME=/root`, the `PATH` of the machine, and nothing
  of your environment but `LANG`, `LC_*` and the `SERVITOR_` variables. What
  it reads beside its scroll it must name under `relics`, paths within the
  rite's hall, laid beside it under the same names:
  `{"litany": "scrolls/governor", "exalted": true, "relics": ["scrolls/governor.conf"]}`.
* Its tongue must be bash, zsh, fish, sh or dash, the program the root's
  alone, and bidden read nothing of its own. Processes it leaves behind are
  slain when the invocation ends; a root process that must last belongs in a
  unit: `systemd-run --unit=servitor-fan-curve fancontrol`.

The Inquisition denounces what breaks these rules, and notes as impurities
`sudo`, `run0`, `pkexec` or `doas` within words that are not exalted (write
`"exalted": true` instead), exalted speech that reaches into your home, and a
program that asks a proof of its own (`systemctl`, `hostnamectl`, …) in a
verse that is not exalted beside exalted ones.

### Sanction and abjure

```sh
servitor sanction split-lock      # behold the exalted verses, prove yourself once
servitor sanction                 # every sanction, how it stands, its sigil
servitor abjure split-lock        # withdraw it, with one proof (--all: every one)
```

`servitor sanction <rite>` first summons the Inquisition: a rite in heresy,
or one holding no exalted verse, is refused. Then it shows you everything the
exalted verses would do, for every aspect, each under its own **sigil** (the
SHA-256 of the sanction, the first twelve letters naming it), or, when a
sanction stands, only what changed since. You must behold it: without a
terminal (or the cogitator) sanctioning is refused. The intercessor then asks
your proof once, and that proof is the sanction; unchanged, nothing is asked.
The exalted servitor lays it in `/var/lib/servitor/sanctions/<uid>/<rite>.json`,
where only the root writes, and journals it.

A sanction holds the exalted verses alone, with every word, path, tongue,
patience, seal, scroll, relic, tome and spliced decree they hold. Changing a
vox-cast or a common verse changes no sanction; changing an exalted verse, or
a tome or relic it reads, lapses it. Each rite stands towards its sanction:

| Sanction | Glyph | Meaning |
|---|---|---|
| `sanctioned` | `✠` | its exalted verses are those you sanctioned |
| `lapsed` | `☓` | they changed since, and must be sanctioned anew |
| `unsanctioned` | `○` | it holds exalted verses, yet no sanction |
| `forsaken` | `†` | a sanction whose rite is gone, or holds no exalted verse |

The census, the augury (`servitor augury split-lock sanction`) and the
cogitator tell it; the Inquisition notes the lapsed, the unsanctioned and the
forsaken as impurities.

### Invoking exalted verses

```console
❯ servitor invoke split-lock on
⋯ sudo demands the master's proof before the exalted verses of split-lock are performed, as sanctioned under the sigil 3893911c6eed.
servitor: authenticate to perform split-lock (sigil 3893911c6eed):
+++ split-lock · off → on +++
  ✔ verse 1 · incantation sysctl -wq kernel.split_lock_mitigate=0
  ✔ verse 2 · transcription /etc/sysctl.d/90-split-lock.conf
✠ The machine spirit is appeased; it bears the aspect «on».
```

1. **Pre-flight**, with your authority as ever: the exalted verses of the
   aspect are drawn anew and their sigil compared with the sanction. Lapsed
   or unsanctioned, the invocation is refused with the death-mark 2 before
   any proof is asked, and the words name `servitor sanction <rite>`. So is
   a rite whose dominion cannot be held, or whose intercessor or installed
   servitor is missing.
2. **One proof.** The intercessor asks it once and raises one exalted
   servitor; a proof dismissed touches nothing (2, and no chronicle line).
   On a terminal one line of tidings (`⋯`) tells it first, naming the
   intercessor and the sigil; `--silence` withholds it.
3. **Performance.** The verses run in the order written; each exalted verse
   is performed by the exalted servitor from the sanction alone, never from
   the scripture as it stands now. Of a vessel it tells only sizes and
   SHA-256 sums; the words of exalted speech return, at most 64 KiB, for the
   lament and the cogitator's `O`.
4. **Reversion.** Should a verse fall, the exalted verses are reverted among
   the common ones without a second proof. Should the invoking servitor be
   slain, the exalted servitor reverts what it performed by itself, and
   journals it.

Every invocation asks anew: sudo is told `-k`, and the cogitator speaks
`run0 -k` after each act through run0. `--foresee` never exalts: it tells the
sanction standing and marks each exalted verse `✠`, and a vessel you may not
read is foreseen as "contents hidden — change unknown". The chronicle names
the intercessor and the sigil. What the exalted servitor did is in the
machine's journal: `journalctl -t servitor`.

### The intercessor

The settings' `intercessor` names who asks the proof: `"auto"` (the
default), `"sudo"`, `"run0"` or `"pkexec"`. `auto` chooses sudo, else run0,
else pkexec at a terminal, and run0, else pkexec without one (sudo then asks
only through a helper you laid outside the servitor). An intercessor named
but missing refuses the invocation; doas is never called upon. sudo is
called upon thus:

```sh
sudo -k -p "servitor: authenticate to perform <rite> (sigil <sigil>): " \
  -- /usr/bin/servitor __exalt <rite> <sigil> <aspect>
```

In the cogitator, sudo is lent the terminal while it asks; run0 and pkexec
are called upon apart from it, so the graphical agent of the desktop asks,
and without one they cannot ask at all. pkexec asks through the agent of the
session too: at a bare terminal, polkit 127 refuses pkexec's own agent
("No session for cookie"), whatever it is bidden run.

### Roused as root

The servitor performs nothing as root: `sudo servitor invoke …`, sanction,
abjure and the cogitator are refused with 2. The reading rituals still serve
under sudo — `sudo servitor census`, augury, inquisition, `--foresee`,
expound — and read the Librarium, the sanctions and the data-slates of the
one who spoke sudo, writing nothing.

`servitor expound exalted`, `sanction`, `abjure`, `sigil`, `relics` and
`intercessor` recite the rest.

## Rituals and runes

| Ritual | What it does |
|---|---|
| `servitor invoke <rite> <aspect> [--<key> <word>] [-f] [-s] [--binharic]` | Perform an aspect of a rite. |
| `servitor invoke <edict> [--<key> <word>] [-f] [-s] [--binharic]` | Proclaim an edict, its whole liturgy anew. |
| `servitor augury <rite> [key] [--is <aspect>] [--spare-auspex]` | Read how a rite stands, the aspect it bears (an edict bears none) and its inscriptions. |
| `servitor census [--binharic] [--spare-auspex] [--vigil]` | Every rite with its aspect, standing, aspects, last rite and purpose. |
| `servitor inquisition [rite…] [--binharic] [--spare-vessels]` | Denounce heresy in rites, settings and vessels. |
| `servitor sanction [<rite>] [--binharic]` | Sanction a rite's exalted verses with one proof, or list every sanction and how it stands. |
| `servitor abjure <rite> \| --all` | Withdraw a sanction, or every sanction, with one proof. |
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
{"rite": "theme", "edict": false, "aspect": "porpl", "standing": "performed",
 "desecrated": false,
 "inscriptions": {"reason": "the night watch begins"},
 "canon": [{"key": "reason", "purpose": "why the visage was changed",
            "mandatory": false, "decrees": {}, "decree": null}],
 "last_rite": {"verdict": "triumph", "at": "2026-10-10T13:30:00Z"},
 "omens": [{"verse": 2, "tether": "~/.local/share/nfluff/current-theme", "aspect": "porpl"}],
 "taint": []}
```

With a key it speaks one value: `aspect`, `standing`, `desecrated`, `former`,
`sanction`, or any inscription the rite declares (an empty line when
uninscribed). `sanction` tells how a rite stands towards the sanction of its
[exalted verses](#exalted-verses): `sanctioned`, `lapsed`, `unsanctioned`, or
`""` when it holds none.
`--is <aspect>` answers by exit alone. The augury exits 0 when it was read,
1 when `--is` did not match, and 2 when it could not be read.

`inscriptions` holds the words the data-slate keeps; `canon` holds what the
rite declares, in written order: each inscription's `key`, `purpose` (`""`
when unwritten), `mandatory`, its `decrees` per aspect (a `"*"` resolved for
every aspect it serves, an aspect without a decree absent) and `decree`, an
edict's one plain decree or `null`. An edict's `decrees` is `{}`, and a
heretical rite's canon is `[]`.

The augury of an edict bears only its standing, inscriptions, canon and last
rite:

```json
{"rite": "screenshot", "edict": true, "standing": "proclaimed",
 "inscriptions": {"region": "whole"},
 "canon": [{"key": "region", "purpose": "the slurp geometry",
            "mandatory": false, "decrees": {}, "decree": "full"}],
 "last_rite": {"verdict": "triumph", "at": "2026-10-11T01:42:00Z"}}
```

Asked for `aspect`, `desecrated` or `former`, or given `--is`, an edict is
refused, and the augury exits 2: it bears no aspect. Ask after its
`standing` instead.

```sh
servitor augury theme reason
[ "$(servitor augury mouse-autohide-toggle aspect)" = on ] && echo "the cursor is veiled"
```

The servitor does not trust its memory alone: it reads **omens** from the
machine. A sanctum bears witness through the aspect in its marker, a tether
through the anchor it leads to, a transcription through the scripture its
vessel holds, and the auspex through what it prints. Incantations and
litanies leave nothing to be read, and so an edict yields no omen at all.
From the omens the **standing** is judged:

| Standing | Meaning |
|---|---|
| `performed` | the omens agree on one aspect, or none speaks and the data-slate names it |
| `dormant` | neither omen nor data-slate names an aspect; an edict never proclaimed |
| `corrupted` | the omens disagree, or the last invocation faltered |
| `tainted` | other hands altered the scripture within a sanctum |
| `desecrated` | the omens agree on an aspect the data-slate did not record; `former` names the remembered one |
| `heretical` | the rite's scripture is invalid |
| `proclaimed` | an edict alone: its last invocation ended in triumph or was reverted |

Desecration is no heresy. When do-not-disturb is raised from your bar, an
auspex lets the augury report the true aspect and mark the rite desecrated;
the next invocation through the servitor cleanses it.

```jsonc
"auspex": "makoctl mode | grep -q do-not-disturb && echo on || echo off"
```

### Census

```sh
servitor census
servitor census --binharic | jq -r '.[] | select(.standing | IN("performed", "proclaimed") | not) | .rite'
```

In binharic every rite bears `rite`, `edict`, `aspect`, `standing`,
`sanction`, `auspex`, `aspects`, `purpose`, `desecrated`, `last_rite`, `recorded_in`
(the scripture it was read from) and `canon`, as the augury speaks it, so
one census tells what every rite may be inscribed with. An edict bears
`"edict": true`, an empty `aspect` and `"aspects": []`. The table marks a
rite with exalted verses by the glyph of its sanction: `✠` sanctioned, `☓`
lapsed, `○` unsanctioned; a forsaken sanction is lamented apart.

`auspex`, in the census and the augury alike, tells what the rite's auspex
gave: `""` when it keeps none (an edict or a heretical rite neither),
`"heeded"` when it named an aspect, `"mute"` when it fell, outlasted its
patience or named no declared aspect, and `"forgone"` under `--spare-auspex`,
which awakens no auspex and answers at once from the other omens and the
data-slate:

```sh
servitor census --binharic --spare-auspex
servitor augury do-not-disturb --spare-auspex | jq -r .aspect
```

A lesser machine that must show every rite at once keeps vigil instead:
`--vigil` speaks one binharic object per line, each named by its `herald`.
First a `"rite"` line for every rite by name, read without its auspex: a
rite whose auspex is yet to answer bears `"auspex": "awaited"`, every other
rite is told whole at once. Then one more `"rite"` line for every awaited
rite as its auspex settles, `"heeded"` or `"mute"`, each replacing the
earlier line entirely; then a single `{"herald": "concluded"}`. A data-slate
that cannot be read is told as `{"herald": "lament", "lament": …, "rite": …}`.
Once concluded, every rite's last line, its herald set aside, is its entry
of `census --binharic`. Should the reader depart early, the servitor slays
the auspexes still labouring and departs in silence.

```sh
servitor census --vigil | jq -c 'select(.herald == "rite") | {rite, aspect, auspex}'
servitor census --vigil --spare-auspex      # every rite once, forgone, then concluded
```

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
headed `theme → porpl` (an edict's by its name alone) and replaced in place
by the next: progress tells "step x / n" and lingers until the rite ends,
triumph fades as the desktop wills, and a fall is proclaimed with critical
urgency, unbidden; the report follows on stdout once the rite has triumphed.
The words are drawn at random from the servitor's own litanies of progress,
triumph and lament, and speak of "the aspect «porpl»", never of a bare
aspect; an edict's speak of "the edict screenshot", never of an aspect.
Should `notify-send` be absent, the servitor stays silent rather than fail.
The settings' `"vox"` decides which invocations the desktop hears: `"auto"`
(the default) only those roused without a terminal, `"notify-send"` every
one — a terminal's invocation then tells its report *and* sends its missive
— and `"off"` none. In the cogitator the tidings are shown in the cogitator
itself.

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

The line of an edict names no aspect and no former aspect, and bears
`"edict": true` in their stead:

```json
{"at": "2026-10-11T01:42:00Z", "rite": "screenshot", "edict": true,
 "inscriptions": {"region": "whole"}, "verdict": "triumph"}
```

The line of an invocation with exalted verses names last the intercessor
that asked the proof and the first twelve letters of the sigil performed,
`"intercessor": "sudo", "sigil": "4f0c2a9e1b7d"`; a proof dismissed performed
nothing, and is not chronicled.

It is kept at the first of: the rune `--chronicle`, `SERVITOR_CHRONICLE`, the
settings' `chronicle`, `$XDG_STATE_HOME/servitor/chronicle.jsonl`. Past one
mebibyte it is renamed with the suffix `.1` and a fresh chronicle begins. An
invocation forbidden by its pre-flight, or only foreseen, is not chronicled.

```sh
tail -n 3 ~/.local/state/servitor/chronicle.jsonl | jq -r '.rite + " → " + (.aspect // "edict") + ": " + .verdict'
```

## The cogitator

> *+++ Thought for the day: Every unsaved buffer is a soul unshriven. +++*

`servitor` alone at a terminal, or `servitor cogitator`, awakens the
cogitator, a keyboard-driven shrine: the rites on the left with the glyph of
their standing; on the right the chosen rite's purpose, aspects (the current
one marked), standing, last rite, inscriptions, its liturgy with one omen per
step, and where its scripture is recorded; a Thought for the Day below. An
edict bears the mark `❢` beside its standing, shows its nature in place of
aspects, and its liturgy without omens. A rite with exalted verses bears the
mark of its sanction beside its aspect — `✠` sanctioned, `☓` lapsed, `○`
unsanctioned, `†` a forsaken sanction — tells it beneath its standing, and
marks each exalted verse `✠`.

| Key | Rite |
|---|---|
| `↑`/`k` `↓`/`j`, `g` `G` | choose a rite; the first, the last |
| `enter` | invoke: choose the aspect, amend the inscriptions, `p` to foresee; an edict: amend its inscriptions, then confirm, and a second `enter` proclaims it |
| `space` | cycle the rite to its next aspect at once; upon an edict, as `enter`: it asks first, and never proclaims at once |
| `n` | consecrate a new rite |
| `e` | amend the rite |
| `c` | replicate the rite |
| `d` | excommunicate the rite: `y` strikes its scripture, `p` its sanctums as well |
| `s` | the sanction: the exalted verses as they would be sanctioned, or what changed since; `enter` sanctions them with one proof, `a` abjures the sanction |
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

Exalted verses are performed by the exalted servitor alone, raised with one
proof of the master after the pre-flight, as at a terminal. sudo is lent the
terminal while it asks and gives it back once the exalted servitor answers;
run0 and pkexec are called upon apart from the terminal, so the graphical
agent of the desktop asks, and after each act through run0 the cogitator
speaks `run0 -k`. A refusal before any verse — a sanction lapsed or never
laid, no intercessor, the servitor not installed, a proof dismissed — is shown
as a heresy with the words the intercessor spoke, and nothing is touched.

The consecration wizard (`n`, `e`, `c`) leads a rite through three stations:
the **Rite** (name, purpose, its nature, a rite of aspects or an edict, then
aspects, inscriptions, auspex), the **Liturgy** (`a` adds a step of any
kind, `J`/`K` reorder, a page for each aspect where values differ, "same for
every aspect" writes `"*"`, rarer keys rest under the further rites) and the
**Seal**, where the scripture stands as it will be written, judged by the
Inquisition before anything is written. A heretical rite cannot be sealed.
An edict is asked no aspects and no auspex, each of its inscriptions takes
one plain decree, its steps are only incantations, litanies and vox-casts,
and their values are written once, on a single page; amended or replicated,
an edict stays an edict. Every step but a vox-cast may be declared an
exalted verse on its first page, where an exalted incantation or litany
names its relics, one per line; amended or replicated, a rite keeps its
exalted verses and their relics. In forms, `tab`/`↑↓` move between fields, `enter`
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
* [`rites/screenshot.json`](examples/rites/screenshot.json) is an edict:
  it captures the visage of the machine through grim, the region marked by
  slurp unless `--region whole` is inscribed, and lays it upon the
  clipboard, striking the visage should that fall.
* [`rites/split-lock.json`](examples/rites/split-lock.json) holds exalted
  verses: it lifts the kernel's split-lock penance through `sysctl` and keeps
  that aspect in `/etc/sysctl.d`, once its master has sanctioned it.

```sh
servitor --librarium examples inquisition --spare-vessels
servitor --librarium examples invoke theme porpl --foresee
servitor --librarium examples invoke screenshot --foresee
servitor --librarium examples invoke split-lock on --foresee
```

## For Tech-Priests

```sh
go test -race ./...
go vet ./...
golangci-lint run ./...
```

Exalted verses are judged upon the real intercessors only inside a
disposable Arch VM (KVM/QEMU and cloud-init): `test/vm/run.sh [<rev>]`
installs the commit through its PKGBUILD, sanctions, invokes through sudo,
run0 and pkexec with one proof each, and tells its verdict. Nothing is
exalted upon the forge itself; the trials of `go test` raise the exalted
servitor in-process, its authority feigned.

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
| `internal/dominion` | the dominions that keep two invocations apart |
| `internal/sanction` | the exalted unit, its sigils, the store of sanctions and their standing |
| `internal/hallowed` | the trusted path and the acts upon vessels the root may perform |
| `internal/exalted` | the exalted servitor: its session, its acts and its exalted speech |
| `internal/intercessor` | finding sudo, run0 or pkexec and the installed servitor |
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
