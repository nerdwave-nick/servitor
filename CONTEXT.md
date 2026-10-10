# servitor

The servitor is a thrall of the Adeptus Mechanicus that performs rites upon the
machine: it rewrites sacred scripture in vessels, binds names to distant places,
speaks incantations and recites litanies, so that the machine takes on the aspect a
rite commands. Every word it speaks, every key of its scripture and every rune of its
invocation belongs to the liturgy below.

## Rites and aspects

**Rite**:
A named liturgy kept in the Librarium (one scripture in `rites/`) that brings the
machine into one of a fixed set of aspects.

**Pattern**:
The Mark of the form a rite or the settings are written in — `"pattern": "Mark I"`,
"Mark" followed by numerals of the old tongue (I, II, III, IV …). Scripture of an
unknown or missing pattern is heresy.

**Aspect**:
One of the named conditions a rite can bring the machine into, written by hand under
`aspects`. A rite always stands in exactly one aspect, or in none if it lies dormant.

**Purpose**:
The words that tell a reader why a rite or an inscription exists. Key: `purpose`.

**Invocation**:
One performance of a rite into an aspect, from the pre-flight checks to triumph or
reversion. Ritual: `servitor invoke <rite> <aspect>`.

**Reversion**:
When a step of an invocation fails, the steps already performed are undone in reverse
order, so the machine returns to its former aspect. An incantation or litany may carry
its own undoing words under `reversion`.

**Aspect map**:
A value of a rite that is either the same for every aspect, or a table naming one
value per aspect with `"*"` serving every aspect not named.

**Placeholder**:
A `{{…}}` mark inside a rite's words that the servitor fills at invocation:
`{{aspect}}` (the aspect being invoked), `{{former}}` (the aspect before it),
`{{rite.name}}`, `{{inscription.<key>}}`, and `{{heresy}}` in messages of failure.

## The liturgy and its steps

**Liturgy**:
The ordered steps of a rite, performed one after another exactly as written. Key:
`liturgy`.

**Verse**:
The place of a step within its liturgy, counted from one; omens, taints and the
chronicle name a step by its verse and its own key (e.g. `"verse": 2, "tether": …`).

**Vessel**:
A scripture-file of the machine that a rite writes into, wholly or within a sanctum.
Key: `vessel`.

**Sanctum**:
A step that keeps a warded region inside a vessel — between a begin and an end marker
— and rewrites only that region; everything outside it is never touched. Key:
`sanctum`.

**Ward**:
The name that marks one sanctum within its vessel, so that several rites may keep
sanctums in the same vessel. Key: `ward`.

**Transcription**:
A step that writes a whole vessel anew, or removes it, according to the aspect. Key:
`transcription`.

**Tether**:
A step that binds a name in the machine to an anchor elsewhere, so that whoever reads
the name reaches the anchor instead; it can also rebind or unbind it. Keys: `tether`
(the bound name), `anchor` (where it leads).

**Incantation**:
A step that speaks one line of command in a tongue. Key: `incantation`.

**Litany**:
A step that recites a whole written scroll of commands. Key: `litany`.

**Offerings**:
What is laid before a litany for it to work with — each offering handed to the scroll
exactly as written, placeholders filled and a leading `~/` borne to the invoker's home.
A litany receives nothing it is not offered.
Key: `offerings`.

**Vox-cast**:
A step that sends word of the invocation's progress or triumph to the user; tidings of
failure are vox-cast without being asked. Key: `vox-cast`.

**Scripture**:
What a sanctum or transcription places into its vessel — written inline, or drawn from
a **tome** (another scripture-file), and **illuminated** when its placeholders are to
be filled. Keys: `scripture`, `tome`, `illuminate`.

**Tongue**:
The language in which incantations and litanies are spoken; bash unless commanded
otherwise. Key: `tongue`.

**Patience**:
How long an incantation, litany or auspex may labour before it is deemed to have
failed. Key: `patience`.

**Zeal**:
Leave to overwrite what the servitor did not itself create. Key: `zeal`.

## Knowing the current aspect

**Omen**:
What one step reveals about the aspect the machine stands in: the aspect recorded in
a sanctum's marker, the anchor a tether leads to, or which aspect a transcribed
vessel's scripture matches.

**Auspex**:
A rite's own scanning command that reports which aspect the machine stands in,
yielding an omen where no step can. Key: `auspex`.

**Data-slate**:
The servitor's record for each rite of its last invocation — aspect, inscriptions and
verdict. It names the aspect only when no omen can be read.

**Verdict**:
How an invocation ended: *triumph* (every step performed), *reverted* (a step failed and
everything was undone), or *faltered* (a step failed and the reversion was incomplete).

**Inscription**:
A named value carried by one invocation, declared by the rite under `inscriptions`,
given as a `--<key>` rune or taken from the rite's `decrees`, and kept on the
data-slate.

**Standing**:
How a rite stands: *performed* (its omens agree), *dormant* (never invoked, no omen),
*corrupted* (its omens disagree), *tainted* (a sanctum was altered by other hands),
*desecrated* (its omens agree on an aspect the data-slate did not record), or
*heretical* (its scripture is invalid).

## Places and rituals

**Librarium**:
The place where rites are kept (`rites/`) together with the settings. Rune
`--librarium` / `-l`.

**Settings**:
The servitor's standing orders in the Librarium's `servitor.json`: `pattern`,
`tongue`, `chronicle`, `vox`, `patience`.

**Chronicle**:
The record of every invocation ever performed, readable in the cogitator. Key and rune
`chronicle`.

**Augury**:
The ritual that reads a rite's aspect, inscriptions and omens. Ritual:
`servitor augury`.

**Census**:
The ritual that lists every rite and the aspect it stands in. Ritual:
`servitor census`.

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
Revealing everything an invocation would do without performing any of it. Rune
`--foresee` / `-f`.

**Binharic**:
Answers rendered in the machine's own tongue, for other machines to read. Rune
`--binharic`.
