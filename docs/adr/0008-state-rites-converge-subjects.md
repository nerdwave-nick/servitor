# State rites converge subjects

A rite of pattern Mark I performs its whole liturgy on every invocation (ADR 0001). It
cannot tell which of its steps already stand in the target aspect, so it redoes them: a
hotkey that turns gaming on twice rewrites every knob twice. It reads its aspect from
sanctums, tethers, transcriptions and one rite-level auspex (ADR 0002), so a rite of
incantations alone is read through a single command that the faithful must write to
match the whole liturgy, and a half-performed machine reads as `performed`. It rolls back
by hand-written reversions (ADR 0003), which must guess the former aspect. TuneD,
Terraform, Kubernetes controllers, NixOS and Ansible handlers all take the other road:
declare the state each thing should hold, read it, change only what differs, and act on
change (`docs/research/aspect-transitions.md`, lit issue 0987edb9).

We decided — settled by the owner on 2026-10-12 and 2026-10-13, specified as §19 of the
shared specification (lit issue 0aa6022b) — that pattern **Mark II** has two kinds of
rite. A **state rite** declares `aspects` and a list of **subjects**, and has no liturgy.
An **edict** keeps a `liturgy` of actions and has no subjects (ADR 0005 stands, in the
designation form). Mark I scripture is converted once, on first read.

```jsonc
{
  "pattern": "Mark II",
  "purpose": "silence the heralds of the machine",
  "aspects": ["on", "off"],
  "vox-cast": ["success"],
  "subjects": [
    {
      "name": "mako",
      "designation": "machine-spirit",
      "omens": {"on": "do-not-disturb", "*": "default"},
      "auspex": {"scan": "makoctl mode"},
      "enact": {"*": "makoctl mode -s \"$SERVITOR_OMEN\""},
      "introit": ["herald-silence"]
    }
  ],
  "observances": [
    {"name": "herald-silence", "designation": "incantation", "to": ["on"],
     "words": "notify-send -t 2000 'The heralds fall silent'"}
  ]
}
```

- **Subjects** (amends ADR 0001). A state rite is a list of subjects, performed in written
  order. Every subject has a `name` and a **designation**, the key that discriminates every
  object of Mark II that can take several shapes: `sanctum`, `transcription` and `tether`
  read and set themselves as their Mark I steps did; a **machine-spirit** is read by its
  `auspex` and set by its `enact` words. Every per-aspect or per-omen value is always a map,
  with `"*"` serving whatever it does not name; there is no "plain value or map" form.
- **Omens per subject** (amends ADR 0002). An **omen** is the mark an aspect leaves on a
  subject: a sanctum's marker, a tether's anchor, a transcription's scripture, or the words
  a machine-spirit's auspex prints. A subject's `omens` map gives the omen of each aspect;
  a machine-spirit without one has the aspects' own names. There is **no rite-level
  auspex**: a machine-spirit's `auspex` is `{scan, trials}` — `scan` prints the omen,
  `trials` holds one yes/no command per omen (the death-mark 0 = yes), and an omen without
  a trial is what remains when every trial fails.
- **Agreement by intersection** (amends ADR 0002; owner decision U-19.3, 2026-10-13). The
  omen read from a subject is the **set** of aspects it fits — an omen shared by several
  aspects fits all of them. The rite stands in the intersection of every subject's set:
  one aspect is that aspect; several resolve by the data-slate's aspect, else none; an
  empty intersection is `corrupted`. When every omen names one aspect this is ADR 0002's
  rule unchanged; it adds that a subject left in the wrong state makes the rite
  `corrupted` even when its omen is shared by two aspects.
- **Convergence.** An invocation reads every subject, plans each one **unchanged** (its
  omen already the target's) or **changing** (it differs, or could not be read), and sets
  only the changing ones, in written order. Each one is **proven** after it is set: read
  again, it must show the target omen. An invocation in which nothing changes is still a
  triumph, so a hotkey still gets its answer.
- **Rollback restores what was read** (amends ADR 0003). When a setting, a proof or an
  observance falls, everything performed is undone in reverse order, the fallen one
  included: built-in subjects restore the state captured before they were written, as in
  Mark I; a machine-spirit enacts the omen it showed before, and is proven again; one that
  could not be read enacts the omen of the former aspect, and without one its restoration
  falls and the verdict is `faltered`. Hand-written `reversion` remains only for
  observances and edict entries, whose effects cannot be read.
- **Observances.** Actions tied to subjects that changed are named **observances**
  (`incantation` or `litany`): a subject names them as its `introit` (spoken once, before
  the first changing subject naming it) or `recessional` (once, after the last). Their
  `to`/`from` guards name the aspects they admit. They run in place, not deferred.
- **Vox-casts are a rite key** (amends ADR 0004). `vox-cast` is a list of `"progress"` and
  `"success"`; progress tells each subject that changed ("subject x / n", n counting the
  changing subjects). Failures are announced unbidden, and name the subject, the omen
  commanded and the omen found.
- **Recorded omens.** The omens read at an invocation are kept on the data-slate with the
  boot they were read in, and serve as **stale** omens wherever no live one is heard
  (completion, `--spare-auspex`, the first heralding of a vigil, the cogitator's first
  drawing, and an auspex only the exalted servitor may speak). A stale omen alone never
  makes a rite `desecrated`; an invocation never trusts one.
- **Elevation** (amends ADR 0007). Every designation, observance and edict entry may be
  exalted. The sanctioned unit holds, per aspect, each exalted subject resolved — for a
  machine-spirit, its **enact words for every declared omen**, because a rollback enacts
  the omen read before, which belongs to another aspect — and each exalted observance its
  `to` guard admits. The protocol may **skip**: `perform` names its item, which must lie
  after every item already performed, and `revert` of a machine-spirit names the declared
  omen to restore, from the unit's table alone. The exalted servitor is raised only when an
  exalted item changes, so an invocation that changes nothing exalted asks no proof; the
  sanction is still judged in the pre-flight. An exalted subject may carry an **exalted
  auspex** for state only the root may read: the exalted servitor speaks it once, right
  after the handshake (`read`), and answers only the declared omens that fit, never the
  words printed. A rite holding one raises the exalted servitor at every invocation, for
  its plan cannot be known without it — such a subject was changing, and so asked its
  proof, on every invocation before. Protocol 3; the installed servitor must be
  reinstalled.
- **Conversion.** The first ritual that reads a Mark I scripture and may write converts it:
  the original is copied to `rites/.mark-i/` (never overwritten), the converted scripture
  is written in place under the rite's dominion, and one line of tidings tells what
  changed meaning. Completion, foresight, the reading rituals under `sudo` and the exalted
  servitor convert in memory only. A rite with exalted verses lapses its sanction.

## Considered Options

- **Transitions as edges** (`from → to` liturgies, as in Alembic or XState): rejected. They
  need the current aspect known reliably, which the machine does not grant (reboots, bars,
  other hands), and the faithful would write up to n² liturgies. `from` guards on
  observances serve the rare path-dependent action.
- **Layers left by reversion** (each step belongs to aspects; leaving one undoes it, as in
  TuneD's profile switch or gamemode): rejected. It undoes and redoes what the next aspect
  writes again, and still relies on hand-written reversions.
- **A shared omen yields no aspect** (the first reading of the decisions): replaced by the
  intersection, which reads the hidden wrong subject as `corrupted` where "no omen" would
  have read the rite `performed`.
- **A rite-level auspex beside subject auspexes**: rejected. One command for the whole rite
  is what made a half-performed rite read `performed`.
- **Reading Mark I without converting it, or converting back**: rejected; one pattern is
  read and written, and the original is kept for the faithful.
- **Deferred, Ansible-style handlers**: rejected; observances run in place, so the order of
  the liturgy is the order of the machine.

## Consequences

- An invocation reads before it acts: foresight reads every subject but those of an
  exalted auspex, and tells per subject the omen read, the omen commanded and whether it
  changes.
- Known gaps, accepted until a rite needs them: rites that do things and also hold state;
  order that differs by direction (entering versus leaving).
- A machine-spirit without an auspex cannot be read: it is set on every invocation and
  never proven, which the inquisition notes. A watched-only subject (an auspex without
  `enact`) is out of scope.
- The augury names omens by subject (`{"subject": "smt", "designation": "machine-spirit",
  "omen": "off", "aspects": ["on"], "aspect": "on"}`) and gains `fitting`; the census gains
  `stale`; the chronicle gains `changed` and names `fell_at` by subject or observance.
- Exalted built-in subjects the invoker cannot read are decided by the exalted servitor,
  which answers only "changed" or "unchanged"; they are never recorded on the data-slate,
  which is the invoker's to read (spec §19.7, after "Privileged auspexes").
- Sanctions of Mark I rites lapse at conversion, and the exalted servitor of protocol 3
  must be installed before exalted state rites are invoked.
