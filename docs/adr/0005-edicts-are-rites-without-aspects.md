# Edicts are rites without aspects

ADR 0001 says that a rite brings the machine into one of a fixed set of aspects, and
ADR 0002 reads the aspect it stands in from agreeing omens. Some rites have no aspect
at all: rebooting the machine or capturing its visage is done once and leaves nothing
behind to stand in. Under pattern Mark I they can only be written as a rite of a single
dummy aspect, which forces a meaningless aspect word on every invocation
(`servitor invoke screenshot taken`), reports a standing that says nothing true
(`screenshot` *performed* in aspect `taken` forever), and lets the cogitator's `space`
perform a destructive rite with a single key.

We decided — settled by the owner on 2026-10-11 — that such a rite is an **edict**: a
rite that declares `"edict": true` instead of `aspects`. An edict is proclaimed, not
brought into an aspect; every invocation performs its whole liturgy anew.

```jsonc
{
  "pattern": "Mark I",
  "purpose": "Capture the visage of the machine",
  "edict": true,
  "inscriptions": {"region": {"purpose": "the slurp geometry", "decrees": "full"}},
  "liturgy": [
    {"incantation": "grim -g \"$(slurp)\" ~/pictures/{{inscription.region}}.png", "patience": "2m"},
    {"vox-cast": "success"}
  ]
}
```

- **The marker:** a rite has exactly one of `aspects` and `"edict": true`; both, or
  neither, is heresy. `"edict": false` is the same as leaving the key out, and any
  value that is not a boolean is heresy. *(Owner decision: the concept is named
  edict, and it is declared by a separate marker key, not by omitting `aspects`.)*
- **The liturgy:** an edict holds only incantations, litanies and vox-casts, with
  their `reversion`, `patience`, `tongue` and `offerings`. Sanctums, transcriptions
  and tethers are heresy in an edict, because they exist to hold and reveal an aspect.
  *(Owner decision.)*
- **No aspect anywhere:** an edict has no `auspex`, no aspect map in any field (not
  even one keyed only by `"*"`; decrees are one plain word), and no `{{aspect}}` or
  `{{former}}` placeholder. `{{rite.name}}` and `{{inscription.<key>}}` remain; its
  incantations and litanies receive `SERVITOR_RITE` and `SERVITOR_INSCRIPTION_<KEY>`,
  but not `SERVITOR_ASPECT` or `SERVITOR_FORMER_ASPECT`.
- **Proclaiming:** `servitor invoke <edict> [--<inscription> v] [-f] [-s]` takes no
  aspect, and any further word is refused with a heresy. Pre-flight, performing,
  reversion, patience and foresight are unchanged. The data-slate keeps inscriptions
  and `last_rite` but no `aspect`; its inscriptions follow the rule for rites (after
  triumph, those the edict was proclaimed with; after a fall, those recorded before).
  A chronicle line omits `aspect` and `former` and carries `"edict": true`. Vox-casts
  and tidings never speak an aspect; edicts have their own grimdark template lists,
  each naming `the edict {{rite.name}}`.
- **Standing:** an edict is *dormant* until first proclaimed, *proclaimed* after a
  last invocation that ended in triumph or was reverted, *corrupted* when the last
  invocation faltered (until it is proclaimed again), and *heretical* when its
  scripture is invalid. *Proclaimed* is a new standing value, valid only for edicts;
  an edict is never *performed*, *desecrated* or *tainted*.
- **Augury and census:** augury of an edict is `{"rite", "edict": true, "standing",
  "inscriptions", "last_rite"}`, without `aspect`, `desecrated`, `former`, `omens` or
  `taint`; rites gain `"edict": false`. Asking an edict for `aspect`, `desecrated` or
  `former`, or `--is`, is refused with a heresy (exit 2). Census binharic gains
  `"edict"` for every rite, and an edict has `"aspects": []`, never `null`; unlike
  the augury, the census keeps the same keys for every entry, so an edict's also
  carries `"aspect": ""` and `"desecrated": false`. Heretical scripture is read as a
  heretical rite (`"edict": false`), because the Librarium keeps nothing of it.
- **The cogitator always asks first:** `enter` and `space` on an edict both open the
  invoke screen, without an aspect chooser: the inscription form when the edict has
  inscriptions, then a confirmation; a second `enter` proclaims, `p` foresees, `esc`
  retreats. A single key never proclaims an edict; rites keep today's `space`
  cycling. *(Owner decision.)*
- **The pattern stays `"Mark I"`:** the change is additive, and every existing rite
  keeps its meaning.

This **amends ADR 0001 and ADR 0002 for edicts only**: an edict brings the machine into
no aspect, and its standing comes from its last verdict rather than from omens. For
every rite with `aspects`, both records stand unchanged.

## Considered Options

- **A single-aspect rite by convention** (`"aspects": ["taken"]`) — rejected: it is
  the very burden described above — a meaningless aspect word on every invocation, a
  standing that says nothing true, and a one-key `space` that performs a destructive
  rite.
- **Omitting `aspects`** to make an edict — rejected; the owner chose a separate
  marker key. A missing key cannot tell an edict from a rite whose aspects were
  forgotten, and today's "aspects missing" heresy would fall silent. An edict is
  declared by its own key, and a rite with neither is still heresy, now naming both
  ways.
- **An explicit `"aspects": []`** — rejected: an empty list of aspects is heresy for
  every rite today, and reading it as "edict" would make a slip of the hand mean
  something.
- **Allowing tethers and transcriptions in edicts** — rejected; the owner chose a
  liturgy of incantations, litanies and vox-casts only. Tethers and transcriptions
  exist to hold and reveal an aspect, and an edict has none. Edicts that write
  vessels or bind names are out of scope.
- **Keeping one-shot rites out of the servitor** (hotkeys calling commands directly)
  — rejected: such rites want the same inscriptions, reversion, vox-casts, chronicle
  and cogitator as any rite.
- **Letting `space` proclaim an edict at once** — rejected; the owner chose that the
  cogitator never proclaims an edict on a single key. An edict may reboot the
  machine, and one stray key should not do that.
- **Confirmation only where a rite opts into it** — rejected; the owner chose that the
  cogitator always confirms an edict. With an opt-in, the danger would remain the
  default for every edict whose author forgot to ask.

## Consequences

- `CONTEXT.md` gains **Edict** and the standing *proclaimed*; the definitions of Rite,
  Aspect, Invocation and Standing hold for edicts too.
- Inquisition examines an edict's scrolls, tongues and tomes once instead of once per
  aspect. A litany scroll that stands nowhere is an impurity in an edict, since no
  step of an edict can write it. Excommunication with purge has no sanctums to remove
  from an edict.
- The JSON Schema declares `edict` xor `aspects`, restricts an edict's steps, and
  forbids its `auspex` and aspect maps; what the schema cannot declare stays beyond
  the schema in the judgement trials, as with other whole-rite rules.
- Writing scripture back (consecration, amending, replicating) keeps `"edict": true`
  and never writes `aspects` for an edict.
- `servitor invoke` never asks for confirmation; callers confirm before they invoke,
  as ADR 0001 already decided for interactive input. Scheduling or repeating edicts,
  turning existing single-aspect rites into edicts, and elevation stay out of scope.
