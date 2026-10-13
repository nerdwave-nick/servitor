# servitor

> *+++ Thought for the day: A file unwarded is a file defiled. +++*

**servitor** is a thrall of the Adeptus Mechanicus, bound to the scripture of
your machine. You teach it **rites**. A rite knows a fixed set of
**aspects** and the **subjects** it holds in each: warded **sanctums**
inside your vessels, whole vessels it **transcribes**, **tethers** it binds,
and **machine-spirits** it reads and commands by words of your own. Invoke
an aspect, and the servitor reads every subject, sets only those that do not
yet bear the aspect, proves each one after setting it, and speaks the
**observances** you placed around them. Should one subject fall, it restores
everything it changed to what it read before. An **edict** knows no aspect
at all: proclaim it, and its **liturgy** is performed whole, every time. It
asks not *why*. It asks only *which aspect*, and of an edict not even that.

```sh
servitor                                                  # awaken the cogitator
servitor invoke mouse-autohide-toggle on --reason "gaming remnant"
servitor invoke theme "$(printf '%s\n' default finii porpl | fuzzel --dmenu)"
servitor invoke theme porpl --foresee                     # divine, perform nothing
servitor invoke screenshot --region whole                 # an edict, by its name alone
servitor augury theme | jq -r .aspect                     # porpl
servitor augury do-not-disturb --is on || echo "the heralds still speak"
servitor expound machine-spirit                           # as written in the codex
```

The servitor speaks only the tongue of the Mechanicus: its runes, keys,
markers, binharic and laments alike. Where a word is strange to you, ask the
codex: `servitor expound` lists every passage, and `servitor expound <topic>`
recites one at length, with exempla. This scripture is the short catechism;
the codex is the whole of the law.

## Installation

The forge requires Go 1.27 or newer.

### By pacman

Exalted subjects and observances are performed only by the servitor at
`/usr/bin/servitor`, owned by root and writable by none but root. On Arch
Linux, pacman installs it so. From a checkout of this repository:

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

A servitor forged by Go alone performs every rite that holds nothing
exalted. For exalted subjects and observances it rouses the servitor
installed by pacman, and only while both speak one protocol; otherwise it
bids you install the servitor anew. Pattern Mark II speaks a newer protocol
than the servitor that knew only Mark I: install it anew once.

```sh
go install github.com/nerdwave-nick/servitor@latest
# or, from a checkout of this repository:
go build -o ~/.local/bin/servitor .
servitor --version
```

### Completion litanies

Completion knows the rituals, every rite, its aspects (the current one is
marked; an edict is offered none), its inscription runes and the values they
decree, augury keys and the topics of the codex. It never awakens an auspex:
it marks the current aspect from the omens it can read at once and from the
omens the data-slate recorded, so every Tab is answered at once.

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
    ├── do-not-disturb.json
    ├── scripts/recite-hooks  whatever your rites read or recite
    └── .mark-i/              scriptures of the elder pattern, kept once converted
```

Each rite is one scripture of JSON in `rites/`, named `<rite>.json` (or
`.jsonc`): the name of the scripture *is* the name of the rite. Remarks
(`//`, `/* */`) and trailing commas are tolerated, for the Omnissiah is
merciful. Only `rites/` is read; its inner halls and every name beginning
with `.` are passed over.

### The settings

`servitor.json` beside `rites/` holds the standing orders of the servitor.
Every key is optional except the pattern:

```jsonc
{
  "$schema": "https://raw.githubusercontent.com/nerdwave-nick/servitor/main/schema/settings.schema.json",
  "pattern": "Mark II",
  "tongue": "bash",          // the tongue of enact words, observances and auspexes
  "patience": "30s",         // how long words may labour
  "vox": "auto",             // "notify-send": the desktop hears every invocation; "off": none
  "chronicle": "~/.local/state/servitor/chronicle.jsonl",
  "intercessor": "auto",     // who asks the proof for exalted words: "sudo", "run0" or "pkexec"
}
```

For each order the rune outranks the environment, the environment outranks
the settings, and the settings outrank the servitor's own default. The
`intercessor` knows no rune and no environment: the settings alone name it
(see [Exalted subjects and observances](#exalted-subjects-and-observances)).

## Anatomy of a rite

> *+++ Thought for the day: Heresy grows from changes no Magos has examined. +++*

The rite of the machine's visage, from [`examples/rites/theme.json`](examples/rites/theme.json),
with its remarks set aside:

```jsonc
{
  "$schema": "https://raw.githubusercontent.com/nerdwave-nick/servitor/main/schema/rite.schema.json",
  "pattern": "Mark II",
  "purpose": "The visage of the machine",
  "aspects": ["default", "finii", "porpl"],
  "inscriptions": {
    "reason": {"purpose": "why the visage was changed"},
  },
  "vox-cast": ["progress", "success"],
  "subjects": [
    {
      "name": "visage",
      "designation": "tether",
      "tether": "~/.local/share/nfluff/current-theme",
      "omens": {"*": "~/.config/nfluff/themes/{{aspect}}"},
      "introit": ["veil"],
      "recessional": ["awaken"],
    },
  ],
  "observances": [
    {"name": "veil", "designation": "incantation", "words": "niri msg action do-screen-transition --delay-ms 200 || true"},
    {
      "name": "awaken",
      "designation": "litany",
      "scroll": "scripts/recite-hooks",
      "offerings": ["~/.config/nfluff/themes/{{aspect}}"],
      "patience": "1m",
      "reversion": "test -z \"$SERVITOR_FORMER_ASPECT\" || scripts/recite-hooks ~/.config/nfluff/themes/\"$SERVITOR_FORMER_ASPECT\"",
    },
  ],
}
```

| Key | What it holds |
|---|---|
| `pattern` | The Mark of the form the scripture is written in: `"Mark II"`. Required. Scripture of the elder `"Mark I"` is [converted](#conversion-of-the-elder-pattern) once, on first read; scripture without a pattern, or of a Mark this servitor was not forged for, is heresy. |
| `purpose` | Why the rite exists, in words for the faithful. |
| `aspects` | The aspects the rite can bring the machine into, written by hand: not empty, never repeated. A rite bears either `aspects` or `"edict": true`, never both and never neither. |
| `edict` | `true` makes the rite an [edict](#edicts), which knows no aspects and keeps a `liturgy` in place of subjects; `false` is as if unwritten. |
| `inscriptions` | Named words carried by one invocation. Each may bear a `purpose`, be `mandatory`, and hold `decrees`: the word each aspect inscribes when none is given, always keyed by aspect (`{"*": "<word>"}` for one word for every aspect; an edict's decree is one plain word). |
| `tongue` | The tongue of this rite's enact words, auspexes and observances, outranking the settings. |
| `vox-cast` | Which tidings the desktop hears: `["success"]`, `["progress", "success"]`, or none. See [Vox-casts](#vox-casts). |
| `subjects` | What the rite reads and sets, each by its `name`, set in the order written. Not empty. |
| `observances` | Named words spoken around the subjects that change. |
| `$schema` | Read by your editor, passed over by the servitor (see [The codex](#the-codex)). |

A rite holds no auspex of its own and no liturgy: each subject is read by
itself, and a key of the elder pattern written here is heresy whose words
name its new home. Every key the codex does not know is heresy, so a
misspelt key never passes in silence.

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

Every word a rite writes (vessels, bound names, omens, enact words, trials,
words and scrolls of observances, offerings, reversions, inline scripture,
illuminated tomes) may speak `{{aspect}}`, `{{former}}` (the aspect before;
empty when unknown), `{{rite.name}}` and `{{inscription.<key>}}` (empty when
uninscribed). Within an omen, `{{aspect}}` names the aspect whose omen it is.
`{{omen}}` is the omen being set, and may be spoken only in enact words and
trials. Write `{{{{` for a literal `{{`. A mark the codex does not know, or an
inscription the rite never declared, is heresy, never silently left empty.
An edict has no aspect to speak: in its words `{{aspect}}` and `{{former}}`
are heresy.

Enact words, auspexes and observances also receive `SERVITOR_ASPECT`,
`SERVITOR_FORMER_ASPECT`, `SERVITOR_RITE` and `SERVITOR_INSCRIPTION_<KEY>`
(the key in capitals, `-` and `.` made `_`); enact words and trials receive
`SERVITOR_OMEN` besides, and nothing else does. An auspex hears
`SERVITOR_ASPECT` only during an invocation; outside one it is empty. An
edict's words receive no `SERVITOR_ASPECT` and no `SERVITOR_FORMER_ASPECT`.
**Within the words a tongue will speak, prefer these:** a placeholder is
pasted into the words before the tongue hears them, so a quote, a `$` or a
`;` inside an inscription would be spoken as a command.
`"$SERVITOR_INSCRIPTION_REASON"` stays one word, whatever it holds, and so
does `"$SERVITOR_OMEN"` when an omen splices an inscription.

```jsonc
// heeded: the reason reaches the tongue as one word
{"name": "remnant", "designation": "incantation", "words": "echo \"$SERVITOR_INSCRIPTION_REASON\" >> ~/.cache/remnants"}
// perilous: a reason of  it's done; reboot  is spoken as commands
{"name": "remnant", "designation": "incantation", "words": "echo '{{inscription.reason}}' >> ~/.cache/remnants"}
```

## Subjects

> *+++ Thought for the day: Idle cursors breed heresy. +++*

A **subject** is one thing a rite reads and sets. Every subject bears a
`name` (unique within the rite: a letter or digit, then letters, digits,
`_`, `.` or `-`), a `designation` telling what kind of thing it is, and its
`omens`: the mark each aspect leaves upon it. It may bear a `purpose`,
`exalted`, and the observances spoken around it (`introit`, `recessional`).

| Designation | Its own keys | An omen is | Read by | Set by |
|---|---|---|---|---|
| `sanctum` | `vessel`, `ward`, `glyph`, `closing-glyph`, `consecrate` | scripture | the aspect its marker names | rewriting its warded region and marker |
| `transcription` | `vessel`, `seal`, `zeal` | scripture, or `null` (no vessel) | the vessel's scripture | writing or striking the vessel |
| `tether` | `tether` (the bound name), `zeal` | an anchor, or `null` (unbound) | where the name leads | binding, rebinding or unbinding it |
| `machine-spirit` | `auspex`, `enact`, `tongue`, `patience`, `relics` | words | its `auspex` | its `enact` words |

Paths of vessels, tethers, anchors, scrolls and tomes expand a leading `~`
and `$NAMES`, and a path not rooted at `/` is reckoned from the `rites/` hall.
Missing halls above a vessel or tether are never raised: the pre-flight
denounces them.

### Omens

`omens` is always an object keyed by aspect, with `"*"` serving every aspect
that names no omen of its own; an aspect left with no omen and no `"*"` is
heresy. An omen shared by several aspects fits all of them, and `{{aspect}}`
within an omen names the aspect whose omen it is, so one `"*"` can give every
aspect its own:

```jsonc
"omens": {"*": "~/.config/nfluff/themes/{{aspect}}"}    // a tether: one anchor per aspect
"omens": {"on": "0", "*": "1"}                          // a machine-spirit: "1" for every aspect but on
"omens": {"on": ["[Desktop Entry]", "Exec=gamemoded"], "off": null}   // a transcription: struck when off
```

`omens` is required of a sanctum, transcription and tether. A machine-spirit
without `omens` has one omen per aspect, the aspect's own name, so its
auspex may simply print the name of the aspect.

### Sanctum

A sanctum rewrites only the region between its two markers; every line
outside them is inviolate. A vessel that bears no sanctum of its ward yet
receives one at its end. From
[`examples/rites/mouse-autohide-toggle.json`](examples/rites/mouse-autohide-toggle.json):

```jsonc
{"name": "cursor", "designation": "sanctum", "vessel": "~/.config/niri/util.kdl",
 "ward": "mouse-autohide", "consecrate": true,
 "omens": {"on": ["cursor {", "    hide-after-inactive-ms 400", "}"], "off": ""},
 "recessional": ["reread"]}
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
for html, xml, svg and md, `#` for all else. `consecrate: true` brings a
missing vessel into being. Empty scripture keeps an empty sanctum; `null`
scripture is heresy. A sanctum bears the aspect its marker names, and is
**tainted** when other hands altered the scripture within it.

### Scripture and tomes

Scripture is written as one string, as a list of lines, or drawn from a tome,
a scripture-file beside the rite: `{"tome": "snippets/porpl.kdl"}`. A tome is
read afresh in every pre-flight and placed verbatim, unless
`"illuminate": true` bids its placeholders be filled.

### Transcription

A transcription writes the whole vessel. Because it casts down everything, it
replaces a vessel only when that vessel holds the scripture of one of the
rite's aspects, its own work. Anything else halts the pre-flight unless the
subject burns with `"zeal": true`. A new vessel bears the seal `0644` unless
`seal` decrees another; a `null` omen strikes the vessel.

```jsonc
{"name": "gamemode", "designation": "transcription", "vessel": "~/.config/autostart/gamemode.desktop",
 "omens": {"on": ["[Desktop Entry]", "Exec=gamemoded"], "off": null}, "seal": "0600"}
```

### Tether

A tether binds a name to the anchor of the invoked aspect, and a `null`
anchor unbinds it. It rebinds only a tether: should a scripture-file stand
in its place, the pre-flight refuses unless the subject burns with `zeal`.

```jsonc
{"name": "visage", "designation": "tether", "tether": "~/.local/share/nfluff/current-theme",
 "omens": {"*": "~/.config/nfluff/themes/{{aspect}}"}}
```

### Machine-spirit

A machine-spirit is whatever no vessel holds: a mode kept in the memory of a
daemon, a knob of the kernel, a unit of the machine. Its `auspex` reads it,
and its `enact` words set it. From
[`examples/rites/do-not-disturb.json`](examples/rites/do-not-disturb.json):

```jsonc
{
  "name": "mako",
  "designation": "machine-spirit",
  "omens": {"on": "do-not-disturb", "*": "default"},
  "auspex": {"scan": "makoctl mode"},
  "enact": {"*": "makoctl mode -s \"$SERVITOR_OMEN\""},
  "introit": ["herald-silence"],
}
```

* **The auspex** is an object of `scan`, `trials` or both, with an optional
  `patience` (2s unless written) and `exalted` (see
  [The exalted auspex](#the-exalted-auspex)).
  * `scan` is words, spoken once, that print the subject's omen. What they
    print upon stdout, trimmed, at most 4 KiB, is the omen read; the aspects
    whose omen it equals are the aspects it fits. A machine-spirit without
    `omens` may print several names of aspects, parted by emptiness.
  * `trials` holds one word of judgement per omen, keyed by omen (`"*"`
    allowed): all are spoken at once, and an omen fits when its trial ends
    with the death-mark 0. When no trial holds, the omens that bear no trial
    fit. Each trial hears its own omen as `{{omen}}` and `SERVITOR_OMEN`.
  * With both, the scan reads and the trials are not spoken while reading.
  * An auspex that falls, outlasts its patience, prints nothing or an omen
    that is not declared, or whose trial is slain, is **mute**: it gives no
    omen, and is never mistaken for an aspect. One slow trial makes the
    whole auspex mute.
* **`enact`** is keyed by omen, with `"*"`: the words that set the subject
  to the omen of the aspect invoked, which they hear as `{{omen}}` and
  `SERVITOR_OMEN`. An omen with neither its own words nor `"*"` is heresy, and
  so is a machine-spirit without `enact`.
* **No auspex** may be written: such a subject cannot be read, so it is set
  at every invocation and never proven. The Inquisition notes it.

```jsonc
// tried, for no word tells which scheduler reigns: "cosmos" holds while its
// unit is active, and "eevdf", which bears no trial, when every trial fails
"auspex": {"trials": {"cosmos": "systemctl is-active -q scx-cosmos"}, "patience": "1s"},
"enact": {"cosmos": "systemd-run --unit=scx-cosmos --collect scx_cosmos -p 0 -c 0",
          "*": "systemctl stop scx-cosmos"}
```

Enact words and auspexes are spoken in the rite's hall (`rites/`), with
nothing upon their input, in the subject's `tongue`, else the rite's, else
the settings', else bash. A death-mark other than 0 makes enact words fall.
`patience` (30s unless the settings or the subject say otherwise) is how
long enact words may labour; the auspex keeps its own. When patience runs
out, the words are slain together with every process they summoned. What
they utter is captured, never printed; the lament of a fallen subject shows
its last words.

## Observances

Some words belong to no subject: the transition of a screen, the litany a
theme keeps for its awakening, a herald sent before the heralds fall silent.
They are **observances**: a list of named words, spoken around the subjects
that name them.

```jsonc
"observances": [
  {"name": "herald-silence", "designation": "incantation", "to": ["on"],
   "words": "notify-send -t 2000 'The heralds fall silent'"},
]
```

| Key | What it holds |
|---|---|
| `name` | Required; unique among the observances, named as subjects are. |
| `designation` | `incantation` (one line of `words` for the tongue) or `litany` (a whole `scroll`). |
| `words` | An incantation's command, plain words that may hold placeholders. |
| `scroll`, `offerings` | A litany's scroll, and what is laid before it: each offering handed over exactly as written, never split, never interpreted by a tongue, its placeholders filled and a leading `~/` borne to the invoker's home. |
| `tongue`, `patience` | As for a machine-spirit. |
| `reversion` | Words spoken should the invocation fall after this observance was spoken. |
| `exalted`, `relics` | See [Exalted subjects and observances](#exalted-subjects-and-observances). |
| `to`, `from` | Guards: lists of declared aspects. `to` admits an invocation into an aspect it names, `from` one out of an aspect it names; an omitted guard admits every aspect, an empty one none, and an unknown former aspect admits no `from`. |

A subject names observances under `introit` (spoken once, before the first
changing subject that names it) and `recessional` (spoken once, after the
last changing subject that names it). An observance is spoken in its place,
never deferred; one named by no changing subject is not spoken at all, so
invoking an aspect the machine already bears speaks nothing. An unknown name
in `introit` or `recessional` is heresy, and so is one observance named both
as an introit and as a recessional; an observance no subject names is an
impurity.

Words that differ per aspect are written as several observances, each
guarded by `to`. A litany is recited by its own shebang when the scroll may
be executed, else through its tongue. Children sent into the background
outlive their observance, but the servitor listens to them only a moment
longer: send their words elsewhere,
`swaybg -i ~/walls/{{aspect}}.png >/dev/null 2>&1 &`. Interactive commands are
unsupported: no terminal is lent to them.

## Vox-casts

> *+++ Thought for the day: Death to the false positive! +++*

`"vox-cast"` is a key of the rite: `["success"]` proclaims the triumph,
`["progress", "success"]` also tells each subject that changed, and none
leaves the desktop silent but for failure, which is always announced. Order
does not matter, and each word may stand once.

Invoked from a terminal, every vox-cast is told in its place within the
report of the invocation (see [Invocation](#invocation)): a progress as
`⋯ <words>  (x/n)`, the triumph as the closing `✠ <words>`. Invoked from a
hotkey or a launcher, with no controlling terminal, the vox-casts of one
invocation become **one** missive upon the desktop through `notify-send`,
headed `theme → porpl` (an edict's by its name alone) and replaced in place
by the next: progress tells "subject x / n", counting the subjects that
change (an edict's tells "step x / n"), and lingers until the rite ends;
triumph fades as the desktop wills, and a fall is proclaimed with critical
urgency, unbidden, naming the subject, the omen commanded and the omen found;
the report follows on stdout once the rite has triumphed. The words are
drawn at random from the servitor's own litanies of progress, triumph and
lament, and speak of "the aspect «porpl»", never of a bare aspect; an
edict's speak of "the edict screenshot". Should `notify-send` be absent, the
servitor stays silent rather than fail. The settings' `"vox"` decides which
invocations the desktop hears: `"auto"` (the default) only those roused
without a terminal, `"notify-send"` every one — a terminal's invocation then
tells its report *and* sends its missive — and `"off"` none. In the
cogitator the tidings are shown in the cogitator itself.

## Invocation

> *+++ Thought for the day: Excuses are the refuge of the untested. +++*

`servitor invoke <rite> <aspect> [--<inscription> <word>…] [-f] [-s]`, or
`servitor invoke <edict> [--<inscription> <word>…] [-f] [-s]` for an
[edict](#edicts)

1. **Pre-flight.** Every omen, enact word, observance and tome is rendered
   for the aspect invoked, every vessel, tether, scroll and tongue weighed
   and every inscription resolved, without touching the machine. Any heresy
   forbids the invocation, and nothing is disturbed.
2. **Reading.** Every subject is read. The **former aspect** is the aspect
   its omens agree on (see [Reading the machine](#reading-the-machine)),
   else the one the data-slate remembers among several that fit, else
   unknown.
3. **The plan.** A subject is *unchanged* when it already bears the omen of
   the aspect invoked, and *changing* when it does not, or when it could not
   be read. **Nothing changing** is still a triumph: the data-slate and the
   chronicle are written, and the success vox-cast still answers.
4. **Performance**, the subjects in the order written. For each changing
   subject: its introits not yet spoken, then the subject is set, then it is
   **proven**: read again, it must bear the omen commanded (a mute reading
   is a fall); then its recessionals, should it be their last changing
   subject. Unchanged subjects speak nothing.
5. **Restoration.** Should a setting, a proof or an observance fall,
   everything performed is undone in reverse order, the fallen one included:
   a sanctum, transcription or tether is restored exactly as it was found (a
   vessel the rite brought into being is struck); a machine-spirit is set
   anew to the omen read before, and proven again (one that could not be
   read is set to the omen of the former aspect, and with no former aspect
   it cannot be restored); an observance speaks its `reversion`. A failing
   restoration does not halt the others. An interrupt from the terminal
   (Ctrl-C) or SIGTERM halts and reverts the invocation the same way; a
   hangup does not.
6. **Verdict.** `triumph` (every change performed and proven), `reverted`
   (something fell and all was restored) or `faltered` (something fell and
   a restoration fell too; the rite then stands corrupted). The verdict and
   the omens read are kept on the rite's data-slate,
   `$XDG_STATE_HOME/servitor/data-slates/<rite>.json`, and in the chronicle.

Spoken at a terminal, the invocation is told in one report as it is
performed: a header with the rite and its turning, a line for every subject
set and every observance spoken (the home written `~`, a long line cut short
with `…` to the terminal's width), every progress vox-cast in its place, and
one closing line of triumph, drawn from the rite's success vox-cast or,
should it have none, from the same litany:

```console
❯ servitor invoke theme porpl
+++ theme · (dormant) → porpl +++
  ✔ observance veil · incantation
  ✔ subject visage · tether ~/.local/share/nfluff/current-theme
  ⋯ Cogs turn: visage now bears the aspect «porpl».  (1/1)
  ✔ observance awaken · litany
✠ Every rune turned, every litany sung: the aspect «porpl» prevails.

❯ servitor invoke theme porpl
+++ theme · porpl → porpl +++
✠ The tech-priests rejoice; the machine wears the aspect «porpl».
```

Should something fall, the lament names the subject, the omen commanded and
the omen found, and tells every restoration:

```console
❯ servitor invoke do-not-disturb on
+++ do-not-disturb · off → on +++
  ✔ observance herald-silence · incantation
servitor ✠ the rite "do-not-disturb" fell at subject mako · machine-spirit: the subject "mako" shows the omen "default"; "do-not-disturb" was commanded
  subject mako · machine-spirit is undone
Every deed is undone; the machine stands as it stood before.
```

`--foresee` (`-f`) reads every subject, as an invocation would, and shows for
each the omen read and the omen commanded, unchanged or changing, every
sanctum and transcription as it would change, every tether rebound, every
word as it would be spoken and every observance that would be spoken; it
sets nothing and never exalts. `--silence` (`-s`) withholds the report, save
the lines the rite's own vox-casts ask for.

```console
❯ servitor invoke do-not-disturb on --foresee
+++ Foreseen: the rite do-not-disturb, off → on +++
observance herald-silence · incantation
  spoken in the tongue bash, with the patience of 30s: bash -c "notify-send -t 2000 'The heralds fall silent'"
subject mako · machine-spirit: "default" → "do-not-disturb", changing
  spoken in the tongue bash, with the patience of 30s: bash -c "makoctl mode -s \"$SERVITOR_OMEN\""
Nothing was performed.
```

`--binharic` is for lesser machines: stdout then bears one line of
binharic alone, the very line the invocation added to the
[chronicle](#the-chronicle), while the report and every lament go to stderr
(the desktop hears as it always would). Should the pre-flight forbid the
invocation, the line names the heresies under `forbidden` instead; an
edict's names no aspect. `--binharic` is not spoken with `--foresee`.

```console
❯ servitor invoke theme default --binharic 2>/dev/null
{"at":"2026-10-13T16:27:42Z","rite":"theme","aspect":"default","former":"finii","inscriptions":{},"verdict":"triumph","changed":["visage"]}

❯ servitor invoke theme porpl --binharic 2>/dev/null
{"rite":"theme","aspect":"porpl","forbidden":["…/rites/theme.json:…: heresy: …"]}
```

The death-mark tells how it ended, with or without `--binharic`:

| Death-mark | The invocation |
|---|---|
| 0 | triumphed, or found nothing to change |
| 1 | was reverted: something fell and every change was restored |
| 2 | was refused before anything was touched: the pre-flight forbade it, the rite or aspect is unknown, words strayed after an edict, a rune was malformed, or — for [exalted subjects and observances](#exalted-subjects-and-observances) — the sanction lapsed or was never laid, no intercessor answered, or the master's proof was dismissed |
| 3 | faltered: something fell and a restoration fell too |

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
upon a stdout or stderr whose reader is gone never touch the subjects, their
restoration, the data-slate or the chronicle. An interrupt and SIGTERM still
halt and revert it. SIGKILL, and a slain cgroup without a launch prefix, it
cannot survive. `servitor expound detached` tells the whole of it.

## Edicts

> *+++ Thought for the day: If a task is worth doing, it is worth a servitor. +++*

Some deeds bring the machine into no aspect at all: its visage captured, the
machine spirit bidden to sleep or to wake anew. Written as a rite of
aspects, such a deed would demand a hollow aspect on every invocation
(`servitor invoke screenshot taken`), report a standing that says nothing
true, and let the cogitator perform it on a single key. Write it instead as
an **edict**, with `"edict": true` in place of `aspects`, and a `liturgy` of
words in place of subjects. From
[`examples/rites/screenshot.json`](examples/rites/screenshot.json), its
remarks set aside:

```jsonc
{
  "$schema": "https://raw.githubusercontent.com/nerdwave-nick/servitor/main/schema/rite.schema.json",
  "pattern": "Mark II",
  "purpose": "Capture the visage of the machine",
  "edict": true,
  "inscriptions": {
    "region": {"purpose": "\"chosen\" lets slurp mark the region, \"whole\" takes every output", "decrees": "chosen"},
  },
  "vox-cast": ["success"],
  "liturgy": [
    {
      "designation": "incantation",
      "words": "if [ \"$SERVITOR_INSCRIPTION_REGION\" = whole ]; then grim ~/pictures/visage.png; else grim -g \"$(slurp)\" ~/pictures/visage.png; fi",
      "patience": "2m",
      "reversion": "rm -f ~/pictures/visage.png",
    },
    {"designation": "incantation", "words": "wl-copy --type image/png < ~/pictures/visage.png"},
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
  if unwritten. An edict bears no `subjects`, `observances` or `aspects`.
  Each entry of its liturgy is shaped as an observance without `name`, `to`
  and `from`: `"designation": "incantation"` with `words`, or `"litany"`
  with `scroll` and `offerings`, and their `reversion`, `patience`, `tongue`,
  `exalted` and `relics`. Entries are named by their **verse**, their place
  counted from one. It bears no aspect map (its decrees are one plain word),
  and never speaks `{{aspect}}` or `{{former}}`.
* **Its proclaiming.** An edict is invoked by its name alone; any word
  after its name is refused before a verse is performed. Every invocation
  performs the whole liturgy anew, with its pre-flight, reversion, patience
  and foresight as for any rite. Should a verse fall, every verse before it
  speaks its reversion in reverse order: should the second verse above
  fall, the first verse's reversion strikes the captured visage.
* **Its standing.** An edict yields no omen and is judged by its last
  verdict alone: `dormant` until first proclaimed, `proclaimed` after a
  triumph or a reversion, `corrupted` after a faltered invocation until it
  is proclaimed again, `heretical` when its scripture is. It is never
  `performed`, `tainted` or `desecrated`.
* **Its augury and census.** The augury speaks its standing, inscriptions
  and last rite, never an aspect, and refuses `--is`; the census shows `·`
  for its aspect and its aspects. Its data-slate and its chronicle lines
  bear no aspect and no omens.
* **The cogitator asks first.** `enter` and `space` alike open its invoke
  screen: its inscriptions, should it bear any, then a confirmation. Only a
  second `enter` proclaims it; `p` foresees, `esc` retreats. No single key
  ever proclaims an edict.

```console
❯ servitor invoke screenshot --region whole
+++ screenshot · edict +++
  ✔ verse 1 · incantation if [ "$SERVITOR_INSCRIPTION_REGION" = whole ]; then g…
  ✔ verse 2 · incantation wl-copy --type image/png < ~/pictures/visage.png
✠ The tech-priests rejoice; the edict screenshot is fulfilled.

❯ servitor augury screenshot standing
proclaimed

❯ servitor invoke screenshot taken
servitor ✠ the edict "screenshot" bears no aspect and is proclaimed by its name alone, yet "taken" followed its name; not one verse of its liturgy is performed
```

`servitor expound edict` recites the rest.

## Exalted subjects and observances

> *+++ Thought for the day: The flesh may ask; only the sanctioned may command. +++*

Some subjects reach beyond what you may touch: the knobs of the kernel, the
scriptures of `/etc`, the units of the machine itself. Write `"exalted": true`
upon such a subject or observance (or upon an edict's verse), and it is set
not with your authority but with the machine's own, by an **exalted
servitor** raised for that one invocation, and only as you **sanctioned** it.
Everything else is performed as ever. From
[`examples/rites/split-lock.json`](examples/rites/split-lock.json), its
remarks set aside:

```jsonc
{
  "$schema": "https://raw.githubusercontent.com/nerdwave-nick/servitor/main/schema/rite.schema.json",
  "pattern": "Mark II",
  "purpose": "Lift the kernel's split-lock penance while the faithful play",
  "aspects": ["on", "off"],
  "vox-cast": ["success"],
  "subjects": [
    {
      "name": "penance",
      "designation": "machine-spirit",
      "exalted": true,
      "omens": {"on": "0", "*": "1"},
      "auspex": {"scan": "sysctl -n kernel.split_lock_mitigate"},
      "enact": {"*": "sysctl -wq kernel.split_lock_mitigate={{omen}}"},
    },
    {
      "name": "remembrance",
      "designation": "transcription",
      "exalted": true,
      "vessel": "/etc/sysctl.d/90-split-lock.conf",
      "omens": {"on": "kernel.split_lock_mitigate = 0\n", "off": null},
    },
  ],
}
```

Exalted subjects and observances need the servitor installed by pacman at
`/usr/bin/servitor` (see [Installation](#installation)): it alone is ever
raised, never the servitor that asks, and only while it and every hall above
it are the root's and written by none but the root.

### Writing exalted words

* Every designation of subject, every observance and every verse of an
  edict may be exalted. `exalted` sets the subject (the vessel written, the
  name bound, the enact words spoken) and speaks the observance with the
  machine's authority; a subject is still read by its invoker, unless its
  auspex is itself exalted. `exalted` is true for every aspect or for none;
  `false` is as if unwritten.
* The words of exalted subjects and observances are fixed when you sanction
  them. They may speak `{{aspect}}`, `{{omen}}`, `{{rite.name}}` and
  `{{inscription.<key>}}`, the inscription then taking its decree alone (its
  rune is refused for the rite). `{{former}}` is heresy there: exalted speech
  hears the former aspect and the words of runes from
  `"$SERVITOR_FORMER_ASPECT"` and `"$SERVITOR_INSCRIPTION_<KEY>"`, each at
  most 4 KiB, without NUL or new line, and its omen from `"$SERVITOR_OMEN"`.
* An exalted vessel or bound name must begin at the root once rendered, may
  not lie beneath `/proc`, `/sys` or `/dev` (their knobs are written by
  exalted machine-spirits), and every hall along it must be the root's alone.
  An exalted tether's anchor too: a tether from `/etc` into your home is
  heresy, and an exalted transcription is the remedy.
* Exalted speech is spoken in a fresh hall of the root's own beneath
  `/run/servitor`, with `HOME=/root`, the `PATH` of the machine, and nothing
  of your environment but `LANG`, `LC_*` and the `SERVITOR_` variables. What
  it reads beside its scroll it must name under `relics`, paths within the
  rite's hall, laid beside it under the same names:
  `{"name": "governor", "designation": "litany", "scroll": "scrolls/governor", "exalted": true, "relics": ["scrolls/governor.conf"]}`.
* Its tongue must be bash, zsh, fish, sh or dash, the program the root's
  alone, and bidden read nothing of its own. Processes it leaves behind are
  slain when the invocation ends; a root process that must last belongs in a
  unit: `systemd-run --unit=scx-cosmos --collect scx_cosmos -p 0 -c 0`.

The Inquisition denounces what breaks these rules, and notes as impurities
`sudo`, `run0`, `pkexec` or `doas` within words that are not exalted (write
`"exalted": true` instead), exalted speech that reaches into your home, a
program that asks a proof of its own (`systemctl`, `hostnamectl`, …) in
words that are not exalted beside exalted ones, and an exalted
machine-spirit without an auspex, set as root at every invocation.

### Sanction and abjure

```sh
servitor sanction split-lock      # behold the exalted subjects, prove yourself once
servitor sanction                 # every sanction, how it stands, its sigil
servitor abjure split-lock        # withdraw it, with one proof (--all: every one)
```

`servitor sanction <rite>` first summons the Inquisition: a rite in heresy,
or one holding nothing exalted, is refused. Then it shows you everything the
exalted subjects and observances would do, for every aspect, each aspect
under its own **sigil** (the SHA-256 of the sanction, the first twelve
letters naming it), or, when a sanction stands, only what changed since. For
each aspect it shows, in the order written, every exalted subject (`subject
penance · machine-spirit · place 1`) with what it would write or bind, and
for a machine-spirit its omen and the enact words of **every** omen it
declares, for a restoration enacts the omen read before, which belongs to
another aspect; then every exalted observance the aspect admits by `to`. You
must behold it: without a terminal (or the cogitator) sanctioning is
refused. The intercessor then asks your proof once, and that proof is the
sanction; unchanged, nothing is asked. The exalted servitor lays it in
`/var/lib/servitor/sanctions/<uid>/<rite>.json`, where only the root writes,
and journals it.

A sanction holds the exalted subjects and observances alone, with every
word, path, tongue, patience, seal, scroll, relic, tome and spliced decree
they hold, and the words of an exalted auspex. Changing a vox-cast, a common
subject or observance, or the common auspex of an exalted subject changes no
sanction; changing an enact word of an exalted machine-spirit lapses every
aspect's sigil, and changing a `to`-guarded observance lapses only the
aspects it admits. Each rite stands towards its sanction:

| Sanction | Glyph | Meaning |
|---|---|---|
| `sanctioned` | `✠` | its exalted subjects and observances are those you sanctioned |
| `lapsed` | `☓` | they changed since, and must be sanctioned anew |
| `unsanctioned` | `○` | it holds exalted subjects or observances, yet no sanction |
| `forsaken` | `†` | a sanction whose rite is gone, or holds nothing exalted |

The census, the augury (`servitor augury split-lock sanction`) and the
cogitator tell it; the Inquisition notes the lapsed, the unsanctioned and the
forsaken as impurities. A rite [converted](#conversion-of-the-elder-pattern)
from the elder pattern lapses its sanction, and must be sanctioned anew.

### Invoking exalted subjects

```console
❯ servitor invoke split-lock on --foresee
+++ Foreseen: the rite split-lock, off → on +++
✠ The exalted subjects and observances stand unsanctioned, and the invocation would be refused — sanction them: servitor sanction split-lock
subject penance · machine-spirit ✠: "1" → "0", changing
  spoken in the tongue bash, with the patience of 30s: bash -c "sysctl -wq kernel.split_lock_mitigate=0"
subject remembrance · transcription /etc/sysctl.d/90-split-lock.conf ✠: changing
  the vessel is consecrated, sealed 0644
  + kernel.split_lock_mitigate = 0
Nothing was performed.
```

1. **Pre-flight**, with your authority as ever: the exalted subjects and
   observances of the aspect are drawn anew and their sigil compared with
   the sanction. Lapsed or unsanctioned, the invocation is refused with the
   death-mark 2 before any proof is asked, and the words name
   `servitor sanction <rite>`, even when nothing exalted would change. So is
   a rite whose dominion cannot be held, or whose intercessor or installed
   servitor is missing.
2. **Reading**, with your authority: every subject is read and the plan
   drawn.
3. **One proof, and only when needed.** When an exalted subject or
   observance is about to change, the intercessor asks the proof once and
   raises one exalted servitor; a proof dismissed touches nothing (2, and no
   chronicle line). **When nothing exalted changes, no proof is asked**: an
   idempotent hotkey asks nothing. On a terminal one line of tidings (`⋯`)
   tells the proof first, naming the intercessor and the sigil; `--silence`
   withholds it.
4. **Performance.** Subjects and observances run in the order written; each
   exalted one is performed by the exalted servitor from the sanction alone,
   never from the scripture as it stands now, and only after every item
   already performed: it may pass over what is unchanged, never go back.
   Your servitor proves an exalted machine-spirit by its common auspex. Of a
   vessel the exalted servitor tells only sizes and SHA-256 sums; the words
   of exalted speech return, at most 64 KiB, for the lament and the
   cogitator's `O`.
5. **Restoration.** Should something fall, the exalted subjects are restored
   among the common ones without a second proof: a machine-spirit is enacted
   to the omen read before, and only to an omen it declares. Should the
   invoking servitor be slain, the exalted servitor restores what it
   performed by itself, and journals it.

Every invocation that asks, asks anew: sudo is told `-k`, and the cogitator
speaks `run0 -k` after each act through run0. `--foresee` never exalts: it
tells the sanction standing and marks each exalted subject and observance
`✠`, and a vessel you may not read is foreseen as "contents hidden — change
unknown". The chronicle names the intercessor and the sigil, when one was
raised. What the exalted servitor did is in the machine's journal:
`journalctl -t servitor`.

### The exalted auspex

Some subjects only the root may read: `nft list ruleset`, `/sys/kernel/debug`,
`/dev/cpu/*/msr`, a scripture-file sealed `0600`. Mark the auspex of an
exalted machine-spirit `"exalted": true`, and the exalted servitor speaks it
for you, at invocation alone:

```jsonc
{
  "name": "gates",
  "designation": "machine-spirit",
  "exalted": true,
  "omens": {"on": "open", "*": "shut"},
  "auspex": {"scan": "nft list table inet gaming >/dev/null 2>&1 && echo open || echo shut", "exalted": true},
  "enact": {"open": "nft -f /etc/nftables.d/gaming.nft", "*": "nft delete table inet gaming"},
}
```

* It is allowed only upon an exalted subject. Its words, tongue, patience
  and relics are part of the sanction: editing them lapses it.
* A rite holding one raises the exalted servitor at **every** invocation,
  for its plan cannot be known without it: right after the proof and before
  anything is performed, your servitor asks it to read every exalted auspex
  at once, and only then is the plan complete. A subject found unchanged is
  passed over; a changed one is set, proven and, should something fall,
  restored and proven by the exalted servitor.
* The exalted servitor matches what the auspex printed against the omens
  the subject declares, and answers only which declared omens fit, or mute:
  **never the words printed**. A sanctioned auspex can thus never serve as
  an oracle of what else the root keeps. An exalted sanctum, transcription
  or tether you cannot read is judged the same way, unchanged or changing,
  never its scripture.
* It is never spoken by the augury, the census, the vigil, completion or
  `--foresee`, which never ask a proof. Outside an invocation such a subject
  shows the omen last recorded on the data-slate, marked stale (see
  [Recorded omens](#recorded-omens)), or none; the foresight of a subject
  never recorded reads `(read at invocation)`:

```console
❯ servitor invoke firewall on --foresee
+++ Foreseen: the rite firewall, (dormant) → on +++
✠ The exalted subjects and observances stand unsanctioned, and the invocation would be refused — sanction them: servitor sanction firewall
subject gates · machine-spirit ✠: (read at invocation) → "open", changing
  spoken in the tongue bash, with the patience of 30s: bash -c "nft -f /etc/nftables.d/gaming.nft"
Nothing was performed.
```

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
one who spoke sudo, writing nothing, converting nothing upon disk.

`servitor expound exalted`, `sanction`, `abjure`, `sigil`, `relics`,
`exalted-servitor` and `intercessor` recite the rest.

## Rituals and runes

| Ritual | What it does |
|---|---|
| `servitor invoke <rite> <aspect> [--<key> <word>] [-f] [-s] [--binharic]` | Bring a rite into an aspect, changing only what differs. |
| `servitor invoke <edict> [--<key> <word>] [-f] [-s] [--binharic]` | Proclaim an edict, its whole liturgy anew. |
| `servitor augury <rite> [key] [--is <aspect>] [--spare-auspex]` | Read how a rite stands, the aspect it bears (an edict bears none), its omens and its inscriptions. |
| `servitor census [--binharic] [--spare-auspex] [--vigil]` | Every rite with its aspect, standing, aspects, last rite and purpose. |
| `servitor inquisition [rite…] [--binharic] [--spare-vessels]` | Denounce heresy in rites, settings and vessels. |
| `servitor sanction [<rite>] [--binharic]` | Sanction a rite's exalted subjects and observances with one proof, or list every sanction and how it stands. |
| `servitor abjure <rite> \| --all` | Withdraw a sanction, or every sanction, with one proof. |
| `servitor cogitator` | Awaken the interactive shrine; `servitor` alone does so at a terminal. |
| `servitor expound [topic] [--schema [rite\|settings]]` | Recite a passage of the codex, or its index. |
| `servitor completion bash\|zsh\|fish\|powershell` | Engrave the completion litanies. |

Universal runes: `--librarium <hall>` / `-l` (env `SERVITOR_LIBRARIUM`),
`--chronicle <path>` (env `SERVITOR_CHRONICLE`), and `--version` / `-v` for
the servitor alone.

## Reading the machine

> *+++ Thought for the day: The Omnissiah knows all. Lesser beings must consult the manual. +++*

The servitor does not trust its memory alone: it reads **omens** from the
machine. Every subject is read by itself: a sanctum through the aspect in
its marker, a tether through the anchor it leads to, a transcription through
the scripture its vessel holds, a machine-spirit through its auspex. The
**omen** of a subject is the set of aspects it fits: one aspect, several
when they share its omen, or none when it is mute. Observances and edicts
leave nothing to be read.

The omens agree by **intersection**: each subject narrows the aspects the
machine may stand in, and what remains is the rite's **fitting** aspects.
From them the **standing** is judged:

| Standing | Meaning |
|---|---|
| `performed` | one aspect fits, or several and the data-slate's among them, or no subject yields an omen and the data-slate names the aspect |
| `dormant` | neither omen nor data-slate names an aspect (several fit and no data-slate chooses); an edict never proclaimed |
| `corrupted` | the omens leave no aspect at all, or the last invocation faltered |
| `tainted` | other hands altered the scripture within a sanctum |
| `desecrated` | the omens fit an aspect the data-slate did not record, or several that exclude it; `former` names the remembered one |
| `heretical` | the rite's scripture is invalid |
| `proclaimed` | an edict alone: its last invocation ended in triumph or was reverted |

So a subject left in the wrong omen is never hidden by a shared one: should
`smt` read `on` (the omen of `powersave` and `balanced` alike) while the
governor reads `performance`, no aspect remains, and the rite is corrupted.
The next invocation cleanses it, setting only what is wrong:

```console
❯ servitor augury power standing
corrupted
❯ servitor invoke power performance --foresee
+++ Foreseen: the rite power, (corrupted) → performance +++
subject smt · machine-spirit: "on" → "off", changing
  spoken in the tongue bash, with the patience of 30s: bash -c "echo off > ~/smt"
subject governor · machine-spirit: "performance" → "performance", unchanged
Nothing was performed.
```

Desecration is no heresy. When do-not-disturb is raised from your bar, the
auspex of mako lets the augury report the true aspect and mark the rite
desecrated; the next invocation through the servitor cleanses it.

### Augury

Without a key the augury speaks one binharic object:

```json
{"rite": "do-not-disturb", "edict": false, "aspect": "off", "standing": "performed",
 "sanction": "", "auspex": "heeded", "desecrated": false,
 "inscriptions": {}, "canon": [],
 "last_rite": {"verdict": "triumph", "at": "2026-10-13T16:26:40Z"},
 "fitting": ["off"],
 "omens": [{"subject": "mako", "designation": "machine-spirit", "omen": "default",
            "aspects": ["off"], "aspect": "off"}],
 "taint": []}
```

Each omen names its `subject` and `designation`, `aspects` it fits (always
present, possibly one), `aspect` only when exactly one fits, and `omen`, what
a machine-spirit's auspex showed (a built-in subject's omen is its scripture
or anchor, and `aspects` tells what it fits). A mute subject yields no omen.
`fitting` is the intersection, `[]` when none remains or no subject yields
an omen. A taint names the sanctum subject. An omen taken from the
data-slate's record bears `"stale": true` and `"read_at"` (see
[Recorded omens](#recorded-omens)).

With a key it speaks one value: `aspect`, `standing`, `desecrated`, `former`,
`sanction`, or any inscription the rite declares (an empty line when
uninscribed). `sanction` tells how a rite stands towards the sanction of its
[exalted subjects and observances](#exalted-subjects-and-observances):
`sanctioned`, `lapsed`, `unsanctioned`, or `""` when it holds none.
`--is <aspect>` answers by exit alone. The augury exits 0 when it was read,
1 when `--is` did not match, and 2 when it could not be read.

`inscriptions` holds the words the data-slate keeps; `canon` holds what the
rite declares, in written order: each inscription's `key`, `purpose` (`""`
when unwritten), `mandatory`, its `decrees` per aspect (a `"*"` resolved for
every aspect it serves, an aspect without a decree absent) and `decree`, an
edict's one plain decree or `null`. An edict's `decrees` is `{}`, and a
heretical rite's canon is `[]`.

The augury of an edict bears only its standing, sanction, inscriptions,
canon and last rite:

```json
{"rite": "screenshot", "edict": true, "standing": "proclaimed", "sanction": "", "auspex": "",
 "inscriptions": {"region": "whole"},
 "canon": [{"key": "region", "purpose": "\"chosen\" lets slurp mark the region, \"whole\" takes every output",
            "mandatory": false, "decrees": {}, "decree": "chosen"}],
 "last_rite": {"verdict": "triumph", "at": "2026-10-13T16:27:42Z"}}
```

Asked for `aspect`, `desecrated` or `former`, or given `--is`, an edict is
refused, and the augury exits 2: it bears no aspect. Ask after its
`standing` instead.

```sh
servitor augury theme reason
[ "$(servitor augury mouse-autohide-toggle aspect)" = on ] && echo "the cursor is veiled"
```

`auspex`, in the augury and the census alike, tells what the auspexes of a
rite's machine-spirits gave: `""` when none keeps one the augury may speak
(a rite of built-in subjects alone, one whose only auspexes are exalted, an
edict or a heretical rite), `"heeded"` when every one fitted an aspect,
`"mute"` when any fell, outlasted its patience or showed no declared omen,
and `"forgone"` under `--spare-auspex`, which awakens no auspex and answers
at once from the other omens, the recorded ones and the data-slate. The
census bound counts auspex processes: at most sixteen labour at once, a
trials auspex of three omens taking three.

### Recorded omens

At the end of every invocation of a rite of aspects, whatever its verdict,
the data-slate keeps the omen of every machine-spirit last read by an
auspex: its reading when unchanged, its proof when set, its restoration's
proof when restored. A subject whose setting or restoration was not proven
keeps none. Sanctums, transcriptions and tethers are read in an instant and
never recorded.

```json
{"aspect": "off", "inscriptions": {}, "last_rite": {"verdict": "triumph", "at": "2026-10-13T16:26:46Z"},
 "boot": "a74aeb41-cd68-4760-b796-138592d15e93",
 "omens": {"mako": {"omen": "default", "at": "2026-10-13T16:26:46Z"}}}
```

`exalted` marks an omen the exalted servitor read. Recorded omens of another
boot are never used, nor an omen the scripture no longer declares. They
serve wherever no live omen is heard: an exalted auspex outside an
invocation, an auspex forgone (`--spare-auspex`, completion), or one still
labouring (the vigil, the cogitator). Such an omen fits aspects as a live one
does, bears `"stale": true` and `"read_at"` in the augury, and a stale omen
alone never makes a rite desecrated. An invocation never trusts them: it
reads every subject anew.

```console
❯ servitor census --spare-auspex
RITE                   ASPECT   STANDING    ASPECTS              LAST RITE                 PURPOSE
do-not-disturb         ~on      performed   on|off               triumph 2026-10-13 18:27  Silence the heralds of the machine
gaming ○               ·        dormant     on|off               ·                         Enable or disable the gaming optimizations of the machine
mouse-autohide-toggle  on       performed   on|off               triumph 2026-10-13 18:26  Hide the cursor after 400ms of stillness
screenshot             ·        proclaimed  ·                    triumph 2026-10-13 18:27  Capture the visage of the machine
split-lock ○           off      performed   on|off               ·                         Lift the kernel's split-lock penance while the faithful play
theme                  default  performed   default|finii|porpl  triumph 2026-10-13 18:27  The visage of the machine
```

### Census

```sh
servitor census
servitor census --binharic | jq -r '.[] | select(.standing | IN("performed", "proclaimed") | not) | .rite'
```

In binharic every rite bears `rite`, `edict`, `aspect`, `standing`,
`sanction`, `auspex`, `aspects`, `purpose`, `desecrated`, `stale`,
`last_rite`, `recorded_in` (the scripture it was read from) and `canon`, as
the augury speaks it, so one census tells what every rite may be inscribed
with. `stale` is true when the aspect or standing rests on any recorded
omen, and the table marks such an aspect with `~`. An edict bears
`"edict": true`, an empty `aspect` and `"aspects": []`. The table marks a
rite holding exalted subjects or observances by the glyph of its sanction:
`✠` sanctioned, `☓` lapsed, `○` unsanctioned; a forsaken sanction is
lamented apart.

A lesser machine that must show every rite at once keeps vigil instead:
`--vigil` speaks one binharic object per line, each named by its `herald`.
First a `"rite"` line for every rite by name, read without its auspexes and
from the omens recorded: a rite whose auspexes are yet to answer bears
`"auspex": "awaited"` (and `"stale": true` when its recorded omens already
tell its likely aspect), every other rite is told whole at once. Then one
more `"rite"` line for every awaited rite as its auspexes settle, `"heeded"`
or `"mute"`, each replacing the earlier line entirely; then a single
`{"herald": "concluded"}`. A data-slate that cannot be read is told as
`{"herald": "lament", "lament": …, "rite": …}`. Once concluded, every rite's
last line, its herald set aside, is its entry of `census --binharic`. Should
the reader depart early, the servitor slays the auspexes still labouring and
departs in silence.

```sh
servitor census --vigil | jq -c 'select(.herald == "rite") | {rite, aspect, auspex, stale}'
servitor census --vigil --spare-auspex      # every rite once, forgone, then concluded
```

### Inquisition

The Inquisition denounces every **heresy** (the rite cannot be invoked) and
notes every **impurity** (forgiven), each with its place as
`scripture:line:column` and a verbose account of the sin and its penance. It
exits 1 when any heresy was found. Beside the scripture it weighs the
machine: vessels, tomes, anchors, the tongues of enact words, auspexes and
observances, and the sanctions of exalted subjects. Among its impurities:
omens alike for every aspect (such a subject never moves), an observance no
subject names, and a machine-spirit without an auspex.

```sh
servitor inquisition
servitor inquisition theme --spare-vessels     # judge the scriptures alone
servitor inquisition --binharic | jq -r '.[] | select(.judgement == "heresy") | .denunciation'
```

`servitor expound heresy` lists every sin it knows.

## The chronicle

Every performed invocation adds one line of binharic to the chronicle,
whatever its verdict. `changed` names the subjects set and proven, in
order; a fall names the subject or observance under `fell_at`, and each
restoration likewise:

```json
{"at": "2026-10-13T16:26:46Z", "rite": "do-not-disturb", "aspect": "on", "former": "off",
 "inscriptions": {}, "verdict": "reverted", "fell_at": {"subject": "mako"},
 "heresy": "the subject \"mako\" shows the omen \"default\"; \"do-not-disturb\" was commanded",
 "reversions": [{"subject": "mako", "verdict": "triumph"}], "changed": []}
```

An invocation that changes nothing is chronicled too, with `"changed": []`.
The line of an edict names no aspect and no former aspect, bears
`"edict": true` in their stead, and names a fallen verse by its number:

```json
{"at": "2026-10-11T01:42:00Z", "rite": "screenshot", "edict": true,
 "inscriptions": {"region": "whole"}, "verdict": "triumph"}
```

The line of an invocation that raised an exalted servitor names last the
intercessor that asked the proof and the first twelve letters of the sigil
performed, `"intercessor": "sudo", "sigil": "4f0c2a9e1b7d"`; a proof
dismissed performed nothing, and is not chronicled.

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
one marked), standing, last rite, inscriptions, its subjects, observances and
vox-casts, and where its scripture is recorded; a Thought for the Day below.
It draws at once, from the omens it can read in an instant and the omens the
data-slates recorded, and draws again as each rite's auspexes answer.

* **Subjects**, one line each: the glyph of its designation (`⚙` for a
  machine-spirit; a built-in subject shows its vessel or bound name), its
  name and the omen read, marked `✔` when it bears the
  rite's aspect, `✖` when it does not, `·` with the reason when it yields
  none (`mute`, `unread`, `awaited`, `forgone`). When no aspect remains, the
  subjects that left none are marked `· breaks the agreement`. A recorded
  omen is shown dimmed with its age, `stale · read 2h ago`, and the list
  marks such an aspect `~`.
* **Observances**, with their guards (`» veil · to on`), then the vox-cast
  list.
* An edict bears the mark `❢` beside its standing, shows its nature in place
  of aspects, and its liturgy. A rite holding exalted subjects or observances
  bears the mark of its sanction beside its aspect — `✠` sanctioned, `☓`
  lapsed, `○` unsanctioned, `†` a forsaken sanction — tells it beneath its
  standing, and marks each exalted item `✠`.

```
✔ ⚙ ✠ smt "off"
✔ ⚙ ✠ scheduler "cosmos" stale · read 2h ago
✔ ↣ visage ~/.local/share/nfluff/current-theme
· ⚙ boost · unread
```

| Key | Rite |
|---|---|
| `↑`/`k` `↓`/`j`, `g` `G` | choose a rite; the first, the last |
| `enter` | invoke: choose the aspect, amend the inscriptions, `p` to foresee; an edict: amend its inscriptions, then confirm, and a second `enter` proclaims it |
| `space` | cycle the rite to its next aspect at once; upon an edict, as `enter`: it asks first, and never proclaims at once |
| `n` | consecrate a new rite |
| `e` | amend the rite |
| `c` | replicate the rite |
| `d` | excommunicate the rite: `y` strikes its scripture, `p` its sanctums as well |
| `s` | the sanction: the exalted subjects and observances as they would be sanctioned, or what changed since; `enter` sanctions them with one proof, `a` abjures the sanction |
| `o` | open the scripture in `$EDITOR` |
| `i` | summon the Inquisition |
| `O` | the words of the last invocation |
| `h` | read the chronicle, filtered to the chosen rite |
| `/` | filter |
| `r` | re-read the Librarium |
| `?` | expound the codex |
| `q` | retreat, slaying every auspex still labouring |

The preview of an invocation (`p`) is its foresight: each subject's omen
read and commanded, unchanged or changing, and the observances that would
be spoken. While a rite is invoked, a panel shows its vox-casts, "subject
x / n" (counting the subjects that change), the subject being set with the
omen it is set to, and the observances spoken; the words are captured,
never printed. A fall opens its lament: the subject or observance that
fell, the omen commanded, the omen read before, every restoration and the
last words of the fallen one.

Exalted subjects and observances are performed by the exalted servitor
alone, raised with one proof of the master when something exalted changes
or an exalted auspex must be read, as at a terminal. sudo is lent the
terminal while it asks and gives it back once the exalted servitor answers;
run0 and pkexec are called upon apart from the terminal, so the graphical
agent of the desktop asks, and after each act through run0 the cogitator
speaks `run0 -k`. A refusal before anything is performed — a sanction
lapsed or never laid, no intercessor, the servitor not installed, a proof
dismissed — is shown as a heresy with the words the intercessor spoke, and
nothing is touched.

The consecration wizard (`n`, `e`, `c`) leads a rite through its stations:
the **Rite** (name, purpose, its nature, a rite of aspects or an edict, then
aspects, inscriptions and the vox-cast list), then its **Subjects** (`a`
adds one by designation, `J`/`K` reorder, `d` strikes; an omens grid gives
each aspect its omen, "every other aspect" writing `"*"`) or, for an edict,
its **Liturgy**, and the **Seal**, where the scripture stands as it will be
written, judged by the Inquisition before anything is written. A heretical
rite cannot be sealed. Amended or replicated, a rite keeps its pattern. In
forms, `tab`/`↑↓` move between fields, `enter`
advances, `ctrl+s` seals the page and `esc` steps back.

## Conversion of the elder pattern

> *+++ Thought for the day: The old forms are honoured, then remade. +++*

Scripture of the elder pattern `"Mark I"` — a rite whose `liturgy` of steps
the servitor performed whole at every invocation, read by one auspex of the
whole rite — is converted to pattern Mark II **once**, by the first ritual
that reads it and may write:

1. The original is copied to `rites/.mark-i/<rite>.json` (never
   overwriting a copy that stands; the Librarium never reads `.mark-i/`).
2. The converted scripture is written in its place, under the rite's
   dominion; a scripture tethered into `rites/` is converted where it lies.
3. One line of tidings names the rite, the copy and anything that changed
   meaning, upon stderr (even under `--binharic`; `--silence` mutes it), or
   as a toast in the cogitator:

```console
⋯ the rite "dnd" is converted to pattern "Mark II"; its scripture of pattern "Mark I" is kept at ~/.config/servitor/rites/.mark-i/dnd.json; changed in meaning, as the comments of its scripture tell: the auspex of the rite is struck
```

Completion, `--foresee`, the reading rituals under sudo and the exalted
servitor never write: they read the elder scripture converted in memory
alone, until a writing ritual reads it. Heretical elder scripture is not
converted, and its words say it will be once its heresies are purged. The
settings have only their `pattern` rewritten, their remarks kept, the copy
laid at `.mark-i/servitor.json` beside them.

What becomes what:

| Mark I | Mark II |
|---|---|
| a sanctum, transcription or tether step | a subject of that designation, named `<kind>-<verse>`, its scripture or anchor become its `omens`; its markers unchanged, so no vessel is rewritten |
| an incantation or litany beside such steps | an observance named `<kind>-<verse>`, the introit of the next subject after it, else the recessional of the last before it; words that differ per aspect part into observances guarded by `to` |
| an incantation or litany in a rite of none | a machine-spirit without omens, its words become `enact` keyed by aspect; its reversion is struck, for a machine-spirit restores by enacting |
| the auspex of the rite | the `scan` of the only machine-spirit, should there be exactly one; otherwise struck |
| vox-cast steps | the rite's `vox-cast` list |
| decrees | always keyed by aspect |
| an edict's steps | entries of the designation form |

Whatever is struck or changes meaning is kept as a remark at its new place,
quoting the elder words; the original's own remarks live on in `.mark-i/`. A
converted rite with exalted steps must be sanctioned anew, and its tidings
say `servitor sanction <rite>`. A converted machine-spirit has no auspex:
write one, so that it is read, skipped when unchanged and proven. Data-slates
and the chronicle need no conversion.

`servitor expound conversion` and `servitor expound mark-i` recite the rest.

## The codex

```sh
servitor expound                  # the index of every passage
servitor expound machine-spirit   # one passage, as written in the codex
servitor expound --schema         # the schema of a rite
servitor expound --schema settings
```

Every ritual, rune, designation and key has its passage, with exempla. The
JSON Schemas of a rite and of the settings dwell in [`schema/`](schema/);
name them under `"$schema"` and your editor completes and judges every key,
`designation` choosing the shape of each subject, observance and verse, with
the codex's own lore:

```jsonc
"$schema": "https://raw.githubusercontent.com/nerdwave-nick/servitor/main/schema/rite.schema.json"
```

## Exempla

[`examples/`](examples/) is a Librarium of its own:

* [`rites/theme.json`](examples/rites/theme.json) re-tethers the nfluff
  theme, veils the screen in a transition before and recites the theme's
  litany of awakening after, through
  [`rites/scripts/recite-hooks`](examples/rites/scripts/recite-hooks), and
  vox-casts its progress and triumph.
* [`rites/mouse-autohide-toggle.json`](examples/rites/mouse-autohide-toggle.json)
  keeps a sanctum in niri's scripture, bids niri read it anew once it
  changed and proclaims the triumph.
* [`rites/do-not-disturb.json`](examples/rites/do-not-disturb.json) asks
  mako which mode it keeps, sets it by `$SERVITOR_OMEN`, and sends a herald
  before the heralds fall silent, only when entering `on`.
* [`rites/split-lock.json`](examples/rites/split-lock.json) holds exalted
  subjects: it lifts the kernel's split-lock penance through `sysctl` and
  keeps that aspect in `/etc/sysctl.d`, once its master has sanctioned it;
  invoked again into the aspect it bears, it asks no proof.
* [`rites/gaming.json`](examples/rites/gaming.json) silences the twin
  threads, lifts the split-lock penance and raises the cosmos scheduler,
  three exalted machine-spirits read by scan and by trial.
* [`rites/screenshot.json`](examples/rites/screenshot.json) is an edict:
  it captures the visage of the machine through grim, the region marked by
  slurp unless `--region whole` is inscribed, and lays it upon the
  clipboard, striking the visage should that fall.

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

Exalted subjects and observances are judged upon the real intercessors only
inside a disposable Arch VM (KVM/QEMU and cloud-init): `test/vm/run.sh
[<rev>]` installs the commit through its PKGBUILD, sanctions, invokes through
sudo, run0 and pkexec with one proof each, and tells its verdict. It also
converts a rite of the elder pattern holding exalted steps, gives it an
exalted auspex and invokes it; invokes a rite whose exalted subjects already
bear the aspect, which asks no proof; and lets a proof fall, restored
without a second proof. `SERVITOR_VM_DIR` names where its images dwell.
Nothing is exalted upon the forge itself; the trials of `go test` raise the
exalted servitor in-process, its authority feigned.

Commits follow the [Liturgy of Commits](CONTRIBUTING.md): Conventional
Commits spoken in the tongue of the Mechanicus (`consecrate`, `purge`,
`transcribe`, …). The lexicon of the domain is [`CONTEXT.md`](CONTEXT.md);
the decisions are recorded in [`docs/adr/`](docs/adr/).

| Package | Its charge |
|---|---|
| `internal/librarium` | reading, examining, converting and writing rites and the settings |
| `internal/placeholder` | illuminating and judging the `{{…}}` marks |
| `internal/sanctum` | finding and rewriting sanctums within a vessel |
| `internal/invocation` | the pre-flight, the plan, the performance, the proofs and the restoration |
| `internal/dominion` | the dominions that keep two invocations apart |
| `internal/sanction` | the exalted unit, its sigils, the store of sanctions and their standing |
| `internal/hallowed` | the trusted path and the acts upon vessels the root may perform |
| `internal/exalted` | the exalted servitor: its session, its acts, its readings and its exalted speech |
| `internal/intercessor` | finding sudo, run0 or pkexec and the installed servitor |
| `internal/augury` | omens, their agreement, standing, the data-slates and their recorded omens |
| `internal/chronicle` | the chronicle |
| `internal/vox` | the herald of vox-casts |
| `internal/rituals` | every ritual upon one Librarium, for the rituals and the cogitator alike |
| `internal/cli` | the rituals, runes and completion litanies |
| `internal/tui` | the cogitator |
| `internal/codex` | the passages of `servitor expound` |
| `internal/lexicon` | the Thoughts for the Day |
| `schema` | the JSON Schemas of pattern Mark II, beside the elder Mark I until it is retired |

## License

[MIT](LICENSE). The Omnissiah shares freely; so may you.

*The Librarium forgets nothing that git remembers.*
