# Invocations hold dominion

Until now only the chronicle took a lock; invocations took none. Two invocations of
one rite (a hotkey pressed twice, a picker launched while a hotkey's invocation still
runs) could interleave their liturgies: the second read the former aspect and passed
its pre-flight while the first was still writing, and one reversion could undo the
other's deeds, since a reversion restores what it remembered (ADR 0003). Callers that
launch the servitor detached and go their way (logis) make this more likely, not less.

We decided — settled by the owner on 2026-10-11, who chose the word **dominion** — that
every invocation, from the command line as from the cogitator, of a rite as of an edict
(ADR 0005), **holds dominion** over its rite from before its pre-flight (and before the
augury of its former aspect) until its verdict is kept, and over every vessel and
tethered name it writes. A dominion is an exclusive `flock` on an empty file beneath
`$XDG_STATE_HOME/servitor/dominions/`: `rites/<rite>` for the rite, and
`vessels/<sha256 of the resolved path>` for each vessel and tether name. The files are
created on demand and never removed while the servitor runs; the kernel releases a
`flock` when its holder dies, so even a SIGKILL leaves no stale dominion.

- **Order:** the rite first, then the vessels in sorted path order. Each process holds
  at most one rite dominion and takes vessel dominions in one global order, so no
  deadlock is possible.
- **Wait, never refuse:** an invocation first tries without blocking. Only when another
  holds the dominion does it tell a wait notice once, then wait, with no timeout of its
  own: the holder is bounded by the patience of its steps. The later request is
  performed after the earlier one, so the aspect it asks for is the one that remains.
- **The wait notice** is automatic tidings, like the tidings of failure (ADR 0004): not
  a vox-cast step, never told when the dominion is free at once, and drawn at random
  from its own template lists (`dominion.txt`, `edict-dominion.txt`). On a terminal it
  is one line `⋯ <message>` before the header (stderr under `--binharic`, muted by
  `--silence`); without one it is the invocation's single desktop notification,
  replaced in place by what follows.
- **Readers take nothing:** `--foresee`, augury, census and inquisition never take or
  wait on a dominion.

The rite dominion and the wait notice came first; the vessel dominion follows as its
own change, using the same notice and taking its dominions after the rite's and before
the pre-flight.

## Considered Options

- **Refuse when held** — rejected: it drops the later, intended request. A detached
  caller such as logis closes at once, so the refusal would surface only as a desktop
  failure notice.
- **One global lock for all invocations** — rejected: a long edict (a screenshot waiting
  two minutes on `slurp`) would block every other rite.
- **Per-rite dominion only** — rejected: several rites may keep sanctums in one vessel
  (wards). Without vessel dominion, B's pre-flight reads the vessel before A writes it,
  B's write loses A's change, and A's reversion can wipe out B's sanctum.
- **A wait timeout or a refusal rune** — left out; the steps' own patience bounds the
  holder.

## Consequences

- Rites with no vessel in common still run in parallel.
- An interrupt or SIGTERM ends the waiting; nothing was touched, so nothing is reverted
  or chronicled.
- A dominion that cannot be held at all (its directory unwritable) is lamented, and the
  invocation is performed without it, as it is without a data-slate.
