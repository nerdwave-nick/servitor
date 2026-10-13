# servitor

The servitor is a thrall of the Adeptus Mechanicus that performs rites upon the
machine: it reads every subject a rite names, sets only those that do not yet bear the
aspect commanded — rewriting sacred scripture in vessels, binding names to distant
places, bidding machine-spirits — and proves each one after setting it, or it performs
whole what an edict proclaims. Every word it speaks, every key of its scripture and
every rune of its invocation belongs to the liturgy below.

## Rites and aspects

**Rite**:
A named scripture kept in the Librarium (one file in `rites/`): either a **rite of
aspects**, which declares `aspects` and the subjects that bear them and brings the
machine into one of its aspects, or an **edict**, which proclaims a liturgy and brings
the machine into none.

**Edict**:
A rite that knows no aspects: it declares `"edict": true` instead of `aspects`, and
every invocation proclaims its whole liturgy anew. It holds no subjects and no
observances; its liturgy holds only incantations and litanies, each written with its
designation; it has no aspect map and no `{{aspect}}` or `{{former}}`. Key: `edict`.

**Pattern**:
The Mark of the form a rite or the settings are written in — `"pattern": "Mark II"`,
"Mark" followed by numerals of the old tongue (I, II, III, IV …). Scripture of an
unknown or missing pattern is heresy. The servitor writes Mark II; scripture of the
elder pattern, Mark I, is converted once, on first read.

**Conversion**:
The one rewriting of a scripture of pattern Mark I into Mark II, performed by the first
ritual that reads it and may write: its steps become subjects and observances, its
vox-cast steps the rite's list, and whatever changed meaning is told once and kept as a
remark in the scripture. Completion, foresight, the reading rituals under `sudo` and the
exalted servitor convert in memory alone. A rite with exalted verses lapses its sanction.

**Mark I copy**:
The elder scripture as it stood before its conversion, kept under `rites/.mark-i/` (the
settings under `.mark-i/servitor.json` beside them) and never overwritten; the
Librarium never reads it.

**Aspect**:
One of the named conditions a rite can bring the machine into, written by hand under
`aspects`. A rite that is not an edict always stands in exactly one aspect, or in none
if it lies dormant; an edict never stands in any.

**Purpose**:
The words that tell a reader why a rite or an inscription exists. Key: `purpose`.

**Invocation**:
One performance of a rite, from the pre-flight checks to triumph or reversion — into
an aspect, or, for an edict, without one. Ritual: `servitor invoke <rite> <aspect>`,
or `servitor invoke <edict>` with no aspect.

**Dominion**:
The hold an invocation keeps over its rite and over every vessel and tethered name it
writes, from before its pre-flight to its verdict, so that no other invocation performs
upon them meanwhile; another invocation waits until the dominion is relinquished.

**Reversion**:
When an invocation falls, everything already performed is undone in reverse order, the
fallen one included, so the machine returns to what was read before: a sanctum,
transcription or tether restores what it held, a machine-spirit is enacted into the omen
it showed before and proven again, and an observance or an edict's entry speaks its own
undoing words under `reversion`, if it bears any. A machine-spirit that could not be read
is restored by the omen of the former aspect; without one, its reversion falls.

**Aspect map**:
A value of a rite given per aspect: a table naming one value per aspect, with `"*"`
serving every aspect not named. In pattern Mark II every per-aspect value is always such
a table — `{"*": …}` gives one value to every aspect. An edict writes every value plainly
and knows no aspect map.

**Placeholder**:
A `{{…}}` mark inside a rite's words that the servitor fills at invocation:
`{{aspect}}` (the aspect being invoked; within an omen, the aspect whose omen it is),
`{{former}}` (the aspect before it), `{{omen}}` (the omen being set by enact words, or
tried by a trial), `{{rite.name}}`, `{{inscription.<key>}}`, `{{subject}}` in vox-casts
and `{{heresy}}` in messages of failure. An edict's words never hold `{{aspect}}`,
`{{former}}` or `{{omen}}`.

## Subjects and observances

**Subject**:
One thing of the machine a rite of aspects reads and sets, named by its `name` and performed
in written order: its omen for each aspect, and how it is read and set, told by its
designation. An invocation sets only the subjects whose omen differs from the target's,
and proves each one after setting it. Key: `subjects`.

**Designation**:
The key that tells which form an object of a Mark II scripture takes: a subject is a
`sanctum`, `transcription`, `tether` or `machine-spirit`; an observance or an edict's
entry is an `incantation` or `litany`. Key: `designation`.

**Machine-spirit**:
A subject read by its own auspex and set by its own enact words — a knob of the kernel,
a mode held in the memory of a daemon, a unit that runs or rests. Its omens are words;
without `omens` they are the aspects' own names. One without an auspex cannot be read,
so it is set on every invocation and never proven. Designation `machine-spirit`.

**Enact**:
The words, per omen with `"*"`, that set a machine-spirit to the omen commanded; they
hear it as `{{omen}}` and `SERVITOR_OMEN`. Key: `enact`.

**Scan**:
Words of a machine-spirit's auspex that print the omen it shows. Key: `scan`.

**Trials**:
Words of a machine-spirit's auspex, one per omen, each asking "does it show this omen?"
and answering by its death-mark (0 is yes); an omen without a trial is what remains
when every trial fails. Each hears its own omen as `{{omen}}` and `SERVITOR_OMEN`.
Key: `trials`.

**Observance**:
A named action of a rite of aspects — an incantation or litany — spoken in place when the
subjects that name it change, never otherwise; its `to` and `from` guards name the
aspects it admits, invoked into and out of. Key: `observances`.

**Introit**:
The observances a subject names to be spoken once, before the first changing subject
that names them. Key: `introit`.

**Recessional**:
The observances a subject names to be spoken once, after the last changing subject that
names them. Key: `recessional`.

**Vessel**:
A scripture-file of the machine that a sanctum or transcription writes into, wholly or
within a sanctum. Key: `vessel`.

**Sanctum**:
A subject that keeps a warded region inside a vessel — between a begin and an end
marker — and rewrites only that region; everything outside it is never touched. Its
marker names the aspect it bears. Designation `sanctum`.

**Ward**:
The name that marks one sanctum within its vessel, so that several rites may keep
sanctums in the same vessel. Key: `ward`.

**Transcription**:
A subject that writes a whole vessel anew, or removes it, according to the aspect; its
omen is the scripture the vessel holds, or `null` for none. Designation
`transcription`.

**Tether**:
A subject that binds a name in the machine to an anchor elsewhere, so that whoever reads
the name reaches the anchor instead; it can also rebind or unbind it. Its omen is the
anchor, or `null` for unbound. Designation `tether`; key `tether` (the bound name).

**Incantation**:
Words of command spoken in a tongue, as an observance or an edict's entry. Designation
`incantation`, key `words`.

**Litany**:
A whole written scroll of commands recited, as an observance or an edict's entry.
Designation `litany`, key `scroll`.

## The liturgy of an edict

**Liturgy**:
The ordered entries of an edict, incantations and litanies, performed one after another
exactly as written, whole at every proclaiming. Key: `liturgy`. A rite of aspects has none.

**Verse**:
The place of an entry within an edict's liturgy, counted from one; the foresight, the
chronicle and the laments name an entry by its verse.

## Speech, scripture and exaltation

**Offerings**:
What is laid before a litany for it to work with — each offering handed to the scroll
exactly as written, placeholders filled and a leading `~/` borne to the invoker's home.
A litany receives nothing it is not offered.
Key: `offerings`.

**Exalted**:
A subject set, or an observance or edict entry spoken, not with its invoker's authority
but with the machine's own — and only as its master sanctioned it, by a short-lived
exalted servitor raised for the one invocation, so that a single proof of the master
serves every exalted subject and observance of it and no other. It is raised only when
something exalted changes, or an exalted auspex must be read. An auspex marked
`exalted` reads what only the root may read: the exalted servitor alone speaks it, at
invocation, and tells only which declared omens fit. Key: `exalted`.

**Relics**:
The scriptures exalted words read beside them — of a machine-spirit, an incantation or a
litany — named by their place in the rite's hall. Their words become part of the
sanction, and the exalted servitor lays them beside the words in its own hall, so that it
never reads them from where the invoker may still alter them. Key: `relics`.

**Vox-cast**:
The rite's list of the word it sends to the faithful — `"progress"` (each subject that
changed, "subject x / n") and `"success"` (the triumph); tidings of failure are
vox-cast without being asked, naming the subject, the omen commanded and the omen
found. Key: `vox-cast`.

**Scripture**:
What a sanctum or transcription places into its vessel — its omen for an aspect,
written inline, or drawn from a **tome** (another scripture-file), and **illuminated**
when its placeholders are to be filled. Keys: `tome`, `illuminate`.

**Tongue**:
The language in which enact words, auspexes, incantations and litanies are spoken; bash
unless commanded otherwise. Key: `tongue`.

**Patience**:
How long enact words, an incantation, a litany or an auspex may labour before it is
deemed to have failed. Key: `patience`.

**Zeal**:
Leave to overwrite what the servitor did not itself create. Key: `zeal`.

## Knowing the current aspect

**Omen**:
The mark an aspect leaves on a subject: the aspect a sanctum's marker names, the anchor
a tether leads to, the scripture a transcribed vessel holds, or the words a
machine-spirit's auspex reads. A subject's `omens` give the omen of each aspect; the omen
read from it fits every aspect whose omen it is — one, or several when they share it.
The rite stands in the aspects every subject's omen fits; when none is left, it is
corrupted. Key: `omens`.

**Auspex**:
A machine-spirit's scanning device: its `scan`, its `trials`, or both — the scan then
reads — with a `patience` of its own. It is spoken by the augury, the census and every
invocation; one marked `exalted` is spoken by the exalted servitor alone, at
invocation. Key: `auspex`.

**Data-slate**:
The servitor's record for each rite of its last invocation — aspect, inscriptions and
verdict, and for a rite of aspects the omens known at its end with the boot they were read
in; an edict's record holds no aspect and no omens. Its aspect speaks only when no omen
is read.

**Stale**:
An omen taken from the data-slate's record where no live one is heard — an exalted
auspex outside an invocation, an auspex spared or still labouring — marked with the
moment it was read. A stale omen fits aspects as a live one does, yet alone it never
makes a rite desecrated, and an invocation never trusts one. JSON `stale`, `read_at`.

**Verdict**:
How an invocation ended: *triumph* (every changing subject set and proven, or nothing
needed to change; an edict's whole liturgy performed), *reverted* (something fell and
everything was undone), or *faltered* (something fell and the reversion was
incomplete).

**Inscription**:
A named value carried by one invocation, declared by the rite under `inscriptions`,
given as a `--<key>` rune or taken from the rite's `decrees`, and kept on the
data-slate.

**Standing**:
How a rite stands: *performed* (its omens agree), *dormant* (never invoked, no omen),
*corrupted* (no aspect fits every omen, or its last invocation faltered), *tainted* (a
sanctum was altered by other hands), *desecrated* (its omens agree on an aspect the
data-slate did not record), or
*heretical* (its scripture is invalid). An edict has no omens and stands only as
*dormant* (never proclaimed), *proclaimed* (its last invocation ended in triumph or
was reverted), *corrupted* (its last invocation faltered, until it is proclaimed
again) or *heretical*; it is never *performed*, *tainted* or *desecrated*.

**Proclaimed**:
The standing of an edict whose last invocation ended in triumph or was reverted. Only
an edict can stand proclaimed.

## Places and rituals

**Librarium**:
The place where rites are kept (`rites/`) together with the settings. Rune
`--librarium` / `-l`.

**Settings**:
The servitor's standing orders in the Librarium's `servitor.json`: `pattern`,
`tongue`, `chronicle`, `vox`, `patience`, `intercessor`.

**Sanction**:
The master's leave for a rite's exalted subjects and observances (an edict's exalted
verses): the servitor shows them, resolved for every aspect — an exalted machine-spirit
with its enact words for every omen, and its exalted auspex — the master proves
themselves once, and a sanctioned copy is laid where only the machine itself may write.
They are performed from that copy alone. A rite stands *sanctioned* while they match its
sanction, *lapsed* once they have changed, *unsanctioned* when it has none; a sanction
whose rite is gone or holds nothing exalted is *forsaken*. Ritual:
`servitor sanction <rite>`, or alone to list every sanction.

**Sigil**:
The mark that seals a sanction: drawn from every word of what a rite holds exalted, so
that any change to it yields another. The servitor names it when it asks for the
master's proof, in the chronicle and in the list of sanctions; a rite whose sigil no
longer matches its sanction has lapsed. Key: `sigil`.

**Abjure**:
Withdrawing a sanction, so that the rite's exalted subjects and observances are performed
no more until sanctioned anew. Ritual: `servitor abjure <rite>`, or `--all`.

**Intercessor**:
The one who vouches for the master before an exalted servitor is raised — `sudo`,
`run0` or `pkexec`, or `auto` to let the servitor choose by whether a terminal attends
the invocation. Settings key `intercessor`.

**Chronicle**:
The record of every invocation ever performed — for a rite of aspects, the subjects it
changed — readable in the cogitator. Key and rune `chronicle`.

**Augury**:
The ritual that reads a rite's aspect, inscriptions and the omen of every subject —
or an edict's standing, inscriptions and last rite. Ritual: `servitor augury`.

**Census**:
The ritual that lists every rite and the aspect it stands in — marked `~` when it rests
on a stale omen; an edict stands in none and shows its standing and last rite in place
of an aspect. Ritual: `servitor census`.

**Inquisition**:
The ritual that examines rites, settings and vessels and denounces heresies (which
must be purged) and impurities (which are noted but forgiven). Ritual:
`servitor inquisition`.

**Cogitator**:
The servitor's interactive shrine. Ritual: `servitor cogitator`, or `servitor` alone.

**Codex**:
The servitor's book of instruction: for every ritual, rune, step and key it records at
length what it does, with an example. Its passages are opened by expounding.

**Expound**:
The ritual that recites a passage of the codex — explaining a ritual, rune, step or
key in detail, "as written in the codex". Ritual: `servitor expound [topic]`; without
a topic it lists every passage.

**Foresee**:
Revealing everything an invocation would do without performing any of it: every
subject read, the omen it shows and the omen commanded, whether it would change, the
words or differences it would perform and the observances that would be spoken. Rune
`--foresee` / `-f`.

**Binharic**:
Answers rendered in the machine's own tongue, for other machines to read. Rune
`--binharic`.
