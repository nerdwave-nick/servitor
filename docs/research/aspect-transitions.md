# Research: how a rite of Mark II moves between aspects

Date: 2026-10-12. Machine: Arch Linux, systemd 262. Upstream sources read on that date;
TuneD at commit `278644ed9650` (master, 2026-09-10).

## Question

A rite may have to do different things depending on where it comes from:
`servitor invoke computer-profile balanced` must turn off the optimisations of
`performance` when it comes from there, and must not touch them when it comes from
`powersave`. How do comparable tools model "moving between named states", and which
model should Mark II adopt?

The owner's candidate models (discussion of 2026-10-12):

- **A. Transitions:** `aspects`, a `subjects` map with each subject's state per aspect,
  and a `liturgy` made of transitions `from → to`, with `"*"` meaning from anywhere.
  n² transitions is an upper bound; most rites would only write `"*" → X`.
- **B. Layers left by reversion:** each step belongs to some aspects (`for`). Steps of the
  former aspect are undone in reverse, and steps of the target are performed. A `from`
  guard serves the rare path-dependent step. The owner's objection: this does extra work
  (undoing deletes a file, and the next aspect writes it again, perhaps with the same
  content).

Settled constraints from the same discussion: each stateful step carries its own optional
undoing and its own optional state check (signs: "which aspect is it?"; trials: "is it
aspect X?"). There is no rite-level auspex. A step whose state is already the target's is
skipped. A failed check after performing a step fails the step and reverts the invocation.
Mark I scripture is converted automatically, once.

Source tags: **[man]** = local man page of the installed version. **[src]** = upstream
source or official documentation, read on the date above. **[local]** = this repository.
Statements marked *Inference* are not verified facts.

## Verdict

None of the comparable tools that manage machine state with a drifting or unknown
current state models it as a transition table. They declare the **desired state per
resource**, read the **current state**, and act only on the **difference** (Terraform,
Kubernetes, systemd `isolate`, NixOS `switch-to-configuration`, TuneD `verify`). They
order the work explicitly, and tie side effects to **what actually changed** (Ansible
handlers, NixOS restarting only the units whose definition changed).

Per-state entry and exit actions (XState, gamemode, TuneD's script plugin) and
per-edge transitions (Alembic, XState transition actions) appear where the current state
is **known reliably**, because the tool holds it in memory or in its own ledger. Servitor
cannot rely on that: Mark I already treats the data-slate as the last resort behind the
omens (ADR 0002), because the machine drifts (reboots, bars, other hands).

**Recommendation (C):** keep the owner's `subjects`, but give each subject a **value per
aspect** and let servitor **converge** each subject from its read value to the target
value. Keep **one ordered liturgy** that places the subjects and the actions. Tie actions
to subjects that actually changed (`when-changed`, after Ansible's handlers). Keep `from`
guards and `{{former}}` for the rare truly path-dependent action. Explicit transition
liturgies stay out until a real rite needs one (§6). This answers the owner's objection
to B: a subject is written once, straight from its current to its target value, and is
not touched when both are equal.

## 1. Desired state and convergence

**Terraform [src].** `terraform plan` "reads the current state of any already-existing
remote objects", "compares the current configuration to the prior state and noting any
differences", and "proposes a set of change actions that should, if applied, make the
remote objects match the configuration". With no difference it reports "that no actions
need to be taken". `-refresh=false` "causes Terraform to ignore external changes, which
could result in an incomplete or incorrect plan".
(<https://developer.hashicorp.com/terraform/cli/commands/plan>)

**Kubernetes [src].** A controller is a control loop like a thermostat: the desired state
is set, "the actual room temperature is the current state", and "each controller tries to
move the current cluster state closer to the desired state". The design expects drift:
"your cluster could be changing at any point".
(<https://kubernetes.io/docs/concepts/architecture/controller/>)

**systemd `isolate` [man, systemd 262].** "Start the unit specified on the command line
and its dependencies and stop all others". Switching targets is computed from the target,
not from the former target: there is no `multi-user → graphical` transition definition.
(`systemctl(1)`)

**NixOS `switch-to-configuration` [src].** "The current systemd state is inspected, the
difference between the current system and the desired configuration is calculated and
actions are generated to get to this state." The actions always run in one fixed order:
stop units, run the activation script, reload systemd, then reload, restart, and start
units. (`nixos/doc/manual/development/what-happens-during-a-system-switch.chapter.md`)

**TuneD `verify` [src].** `tuned-adm verify` "verifies current profile against system
settings … whether system settings match current profile or not (e.g. somebody modified
a sysfs/sysctl value by hand)" (`man/tuned-adm.8`). The script plugin passes `verify` to
profile scripts (`tuned/plugins/plugin_script.py`).

What this means for servitor (*Inference*):
- Mark I's sanctums, transcriptions and tethers are already convergent: they write the
  target aspect's content, whatever was there before.
- The gap is incantations and litanies: they are imperative and cannot be read. Signs and
  trials close that gap; the decided "skip what is already in the target" is exactly
  Terraform's empty plan for that subject.
- `--foresee` becomes a plan: for every subject, its read value, its target value and
  whether it will be set.

## 2. Ordering and change-triggered actions

**Ansible handlers [src].** "Sometimes you want a task to run only when a change is made
on a machine. For example, you may want to restart a service if a task updates the
configuration of that service, but not if the configuration is unchanged. Ansible uses
handlers." A handler notified several times runs once. Handlers run after the tasks of
their section, or earlier with `meta: flush_handlers`. `changed_when` decides what counts
as a change. Tasks themselves run in written order.
(<https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_handlers.html>)

**NixOS [src]** restarts or reloads only the units whose definition changed (same source
as §1).

Puppet's `notify`/`subscribe` relationships serve the same purpose, but its documentation
pages did not render for this research and are not cited as verified.

What this means for servitor, checked against the repository's examples [local]:
- `examples/rites/mouse-autohide-toggle.json` writes a sanctum and then runs
  `niri msg action load-config-file`.
- `examples/rites/theme.json` runs a screen transition, re-tethers
  `current-theme`, and recites the theme's hooks.

In both, the actions only make sense when the subject actually changes. Today they run on
every invocation. A `when-changed` tie would make them run only when needed, and would
keep them in their written place (unlike Ansible's end-of-play default) so that the
screen transition still runs before the tether. *Inference.*

## 3. Entry and exit per state

**XState [src].** "Entry actions are actions that occur on any transition that enters a
state node. Exit actions are actions that occur on any transition that exits a state
node." Actions can also be defined on the transition itself.
(<https://stately.ai/docs/actions>)

**gamemode [src].** Settings come in desired/default pairs: "The desired governor is used
when entering GameMode", "The default governor is used when leaving GameMode instead of
restoring the original value"; `[custom]` holds `start=` and `end=` scripts.
(`example/gamemode.ini`)

**darkman [src].** Scripts in `$XDG_DATA_HOME/darkman` receive the new mode (`dark` or
`light`) as `$1`; the legacy `dark-mode.d` and `light-mode.d` directories hold per-mode
scripts. There is only an entry per mode, with no exit and no from-mode.
(<https://darkman.whynothugo.nl/darkman.1>)

What this means for servitor (*Inference*):
- Entry/exit scales with 2n instead of n², and path dependence emerges from
  `exit(A)` + `entry(B)`.
- Exit needs a known former state. XState and gamemode hold it in memory; servitor does
  not after a reboot or desecration. darkman avoids the problem by having no exit at all:
  each script sets everything for the new mode, which is convergence in disguise.

## 4. Leave by restoring the original value

**TuneD [src].** On a profile switch the daemon stops the old profile with a full
rollback, then applies the new one (`tuned/daemon/daemon.py`, `_thread_code`: "if
terminating due to profile switch: rollback = ROLLBACK_FULL"). Plugins store the current
value before setting their own (`tuned/plugins/base.py`, `_storage_set` of the current
value) and restore it on unapply. `tuned-main.conf`: "rollbacks are always performed on a
profile switch". The script plugin is called with `start` and `stop`, and "you need to
correctly implement the `stop` action … Otherwise, the roll-back step after changing your
TuneD profile will not work" (`plugin_script.py`).

What this means for servitor (*Inference*):
- This is design B. TuneD accepts the double write (restore, then set again) that the
  owner objects to.
- TuneD's profiles are overlays on one baseline, the machine "before TuneD". That fits
  on/off rites, but not peers like `powersave`/`balanced`/`performance`, where no aspect
  is the baseline.
- TuneD's storage of the value read before setting is worth taking. It is how a subject
  can be rolled back on failure (ADR 0003) without a hand-written `reversion`.

## 5. Transitions as edges

**Alembic [src].** Migrations are edges (`upgrade`/`downgrade`) of a revision graph. The
current revision is kept in the database's own `alembic_version` table. Upgrading to a
revision runs "all revisions upon which this version depends … and nothing more": the
path is computed through the graph.
(<https://alembic.sqlalchemy.org/en/latest/branches.html>)

**XState** transition actions (§3) are the per-edge form inside a state machine.

What this means for servitor (*Inference*):
- Edges suit history-like states, where the current state is authoritative (Alembic's
  own table) and every move is deliberate and ordered.
- Servitor's state is not authoritative. When the former aspect is unknown (dormant,
  corrupted, first run), only `"*" → X` can apply. So `"*" → X` must always be complete,
  and a specific `A → X` either copies it (and replaces it) or adds to it (which is a
  `from` guard).

## 6. Comparison

| | A. Transitions (`subjects` + `from → to` liturgies) | B. Layers left by reversion | C. Subjects converged, one liturgy (recommended) |
|---|---|---|---|
| Comparable tools | Alembic, XState transition actions | TuneD profile switch, gamemode | Terraform, Kubernetes, systemd isolate, NixOS, Ansible handlers |
| What the owner writes | n aspect liturgies at least (`"*" → X`), up to n² | one liturgy with `for` layers | one subjects table, one ordered liturgy |
| Unknown or drifted current state | falls back to `"*" → X`, which must be complete | each step's check tells whether it is active | each subject is read; no former aspect is needed |
| Extra work | none, if the transition is written well | undo, then redo (owner's objection) | a subject is set only when its value differs |
| Rollback on failure (ADR 0003) | hand-written reversions | hand-written reversions | read value restored; reversion only for subjects that cannot be read and for actions |
| Path-dependent actions | native | `from` guard | `from` guard, `{{former}}`, `when-changed` |
| Order differing by direction | native | reverse order on leave | not native (§8) |
| Reading the aspect (omens) | checks must live in `subjects`, separate from the transitions that change them | per step | per subject, ADR 0002 unchanged |
| Error message "verse/subject X shows `off`, `on` expected" | per subject | per step | per subject |
| Exalted verses and sanctions (§17) | a sigil per transition | per aspect | per aspect, as decided |
| Mark I automatic conversion | each aspect-mapped step copied into each `"*" → X` | steps become layers | each aspect-mapped observable step becomes a subject in place |
| Cogitator wizard | a transition matrix | per-step aspect toggles | a subject × aspect grid, plus the liturgy list |

## 7. The recommendation, sketched

The key shapes are provisional; the discriminator word and the aspect-map form are still
open. The idea: a subject holds a **value per aspect**; signs read its current value,
trials test a value, and `set` writes one. The decided signs/trials semantics carry over
unchanged, keyed by value rather than by aspect: a value shared by several aspects
naturally fits several aspects.

```jsonc
{
  "pattern": "Mark II",
  "aspects": ["powersave", "balanced", "performance"],
  "subjects": {
    "power-profile": {
      "values": {"powersave": "power-saver", "balanced": "balanced", "performance": "performance"},
      "signs":  {"read": "powerprofilesctl get"},                // what it prints is matched against the values
      "set":    "powerprofilesctl set \"$SERVITOR_VALUE\""
    },
    "smt": {
      "values": {"performance": "off", "*": "on"},
      "trials": {"off": "grep -qx 0 /sys/devices/system/cpu/smt/active"},  // fails → "on"
      "set":    "echo \"$SERVITOR_VALUE\" > /sys/devices/system/cpu/smt/control",
      "exalted": true
    }
  },
  "liturgy": [
    {"office": "subject", "subject": "smt"},
    {"office": "subject", "subject": "power-profile"},
    {"office": "incantation", "words": "notify-send \"profile: $SERVITOR_ASPECT\"",
     "when-changed": ["power-profile"]}
  ]
}
```

| Invocation | smt | power-profile | Action |
|---|---|---|---|
| performance → balanced | `off` → `on`: set | set | runs |
| powersave → balanced | `on` = `on`: skipped | set | runs |
| balanced → balanced | skipped | skipped | skipped |
| corrupted → balanced | read, set if it differs | read, set if it differs | runs if the profile changed |

An invocation then runs (*Inference*, to be specified):
1. **Read** every subject (this is the augury; omens per subject, ADR 0002).
2. **Plan:** for each subject, its read value and its target value (this is `--foresee`).
3. **Perform** the liturgy in order. A subject whose value differs is set and then verified
   by its check; an equal one is skipped. An action runs unless it is tied by
   `when-changed` to subjects that all stayed unchanged, or its `from` guard excludes the
   former aspect.
4. **Roll back** on failure in reverse order: subjects by setting back the value read in
   step 1 (TuneD's storage), actions by their `reversion`.

Sanctums, transcriptions and tethers fit as subjects whose value is a scripture or an
anchor and whose reading and setting are built in. Incantations, litanies and vox-casts
remain actions.

## 8. Limits and open questions

- **Direction-dependent order** (stop a service before editing its config when entering,
  start it after when leaving) is not native to C. The owner's decision on liturgies per
  aspect asked for "undo the changes in reverse order" for complex rites. No current rite
  needs it [local: `examples/rites/`]. If one does, the candidates are an explicit
  `transitions` override (A, with a rule for how it combines with the convergent
  liturgy) or a per-direction order on liturgy entries.
- **Subjects that cannot be read** (no signs, no trials) cannot be skipped or verified.
  They are set on every invocation, as Mark I does, and need a `reversion` to roll back.
- **One subject, several liturgy entries:** should a subject be placed exactly once
  (heresy otherwise), or may it be set twice (e.g. a temporary value)? *Recommend: once.*
- **Subjects inline in the liturgy or in a separate map?** A separate map gives the
  subject × aspect overview and the grid in the cogitator. Inline needs no reference and
  no "placed exactly once" rule.
- **Values per subject versus per-aspect words:** values keep signs and trials independent
  of aspects (a value shared by aspects fits them all). Subjects whose setting cannot be
  one command with `$SERVITOR_VALUE` would need words per value.
- **`when-changed` naming and timing:** run in place (recommended, keeps `theme.json`'s
  order) or deferred to the end like Ansible handlers?

## 9. Owner decisions, 2026-10-12 (Mark II direction)

Decided in discussion after this research. Mark II follows elevation (§17 of the shared
specification), and is checked again once elevation is done.

Specified on 2026-10-13, after elevation landed, as §19 of the shared specification (lit
issue 0aa6022b, revision 37). The elevation check is its §19.10, recorded omens its §19.7,
and its open items §19.14.

**Two kinds of rite**
- A **state rite** declares `aspects` and `subjects`, and has no liturgy. Servitor reads
  every subject, sets only those whose omen differs from the target's, verifies each one
  after setting it, and rolls back by restoring what it read.
- An **edict** ("do things", e.g. a restart) keeps a `liturgy` of actions and has no
  subjects.

**Structure**
- Every value that can take several forms is a discriminated union. The discriminator key
  is `designation`.
- `subjects` is a list. Each entry has a `name` and is performed in written order.
- There is no rite-level auspex. A subject carries its own optional `auspex` and its own
  undoing.
- Every per-aspect or per-omen map is always a map, with `"*"` serving whatever it does not
  name. There is no "plain value or map" form.

**Omens**
- An omen is the mark an aspect leaves on a subject: a value, a scripture, an anchor. A
  subject's `omens` map gives the omen of each aspect.
- Reading a subject finds whose omen it shows. ADR 0002's agreement rule is unchanged.
- An omen shared by several aspects fits all of them, and so yields no single aspect.
- Words see the omen as `{{omen}}` and as `$SERVITOR_OMEN`. The codex prefers the variable
  when the omen holds an inscription.

**Reading and setting a subject**
- **`auspex`** is an object with two optional keys:
  - `scan`: words that print the subject's current omen. Without `omens`, they print
    aspect names directly.
  - `trials`: one yes/no command per aspect, where the death-mark 0 means yes. An aspect
    without a trial is what remains when every trial fails.
- With both keys, `scan` reads and `trials` verifies. With one, that one does both jobs.
  One slow trial makes the whole auspex mute.
- **`enact`** is a map of words per omen, with `"*"`, that sets the subject.

**Actions and feedback**
- **`observances`** is a list of named actions.
- A subject references them in `introit` (before) or `recessional` (after).
  - An introit runs once, before the first changing subject that references it.
  - A recessional runs once, after the last changing subject that references it.
  - To run an action after each subject, write a duplicate under another name.
  - `to`/`from` guards live on the observance.
  - An action that runs when its subject changes runs in place, not deferred.
- **`vox-cast`** on the rite is a list: `["success"]`, `["progress", "success"]`, or none.
  Progress reports each subject that changed. Failures are always announced.
- The cogitator wizard creates and edits every form, so that hand editing is not needed.

**Carried over from §18** (step auspexes): one slow trial mutes the auspex; the
inquisition warns about omens that are identical for every aspect; the census limit only
bounds how many auspexes run at once.

**Conversion:** Mark I scripture is converted automatically, once, on first read. The
original is kept under `rites/.mark-i/`, and the user is told once.

**Known gaps, accepted until a rite needs them**
- Rites that "do things" and also hold state.
- Order that differs by direction: entering versus leaving.

## Sources

- systemd 262 `systemctl(1)`, section `isolate` [man].
- TuneD, <https://github.com/redhat-performance/tuned> at `278644ed9650`:
  `tuned-main.conf`, `man/tuned-main.conf.5`, `man/tuned.conf.5`, `man/tuned-adm.8`,
  `tuned/daemon/daemon.py`, `tuned/plugins/base.py`, `tuned/plugins/plugin_script.py`
  [src].
- NixOS manual, `nixos/doc/manual/development/what-happens-during-a-system-switch.chapter.md`
  (nixpkgs master) [src].
- Ansible, "Handlers: running operations on change" (latest) [src].
- Terraform, `terraform plan` command reference [src].
- Kubernetes, "Controllers" [src].
- XState (Stately), "Actions" [src].
- Alembic 1.20.0, "Working with Branches" [src].
- gamemode, `example/gamemode.ini` (master) [src].
- darkman, `darkman(1)` at <https://darkman.whynothugo.nl/darkman.1> [src].
- Not verified: Puppet relationships (documentation did not render); the W3C SCXML
  recommendation (blocked by a bot challenge).
- This repository: `examples/rites/*.json`, `docs/adr/0002`, `docs/adr/0003` [local].
