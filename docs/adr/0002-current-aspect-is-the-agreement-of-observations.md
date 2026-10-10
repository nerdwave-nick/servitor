# The current aspect is the agreement of all omens

With steps beyond sanctums, servitor can no longer read a rite's current aspect from
sanctum headers alone. We decided that every observable step contributes an **omen**
— a sanctum's header records `aspect|<aspect>`, a tether is compared with each
aspect's rendered anchor, a transcribed vessel with each aspect's scripture, and an
optional rite-level `auspex` command prints the aspect — and that **all omens must
agree**. Agreement names the aspect; any disagreement makes the rite *corrupted*.
Steps that cannot be observed (incantations, litanies, unreadable vessels) contribute
nothing.

The **data-slate** (`$XDG_STATE_HOME/servitor/data-slates/<rite>.json`) records what
servitor last did: aspect, inscriptions and verdict. It names the aspect only when no omen
exists; when it disagrees with the omens, the rite is *desecrated* (changed outside
servitor).

## Considered Options

- **First source wins** in a fixed order — rejected: it silently reports a wrong
  aspect when, say, a tether and a sanctum disagree; disagreement was already an error
  for multi-file rites.
- **Detect a sanctum's aspect from its scripture** instead of its header — rejected:
  aspects can share identical or empty scripture, and tomes can change after an
  invocation.
- **Inscriptions in sanctum headers** (the original `key|value` marker line) —
  rejected: inscriptions are declared once per rite and live only on the data-slate,
  so rites without sanctums have them too. Headers carry `aspect|<aspect>` and nothing
  else:

  ```
  // +++ begin of sanctum <ward> -- aspect|<aspect> +++
  // +++ end of sanctum <ward> +++
  ```

## Consequences

- The auspex has a short patience (2s by default, `{"rite": …, "patience": …}` per
  rite) and is skipped by shell completion so that Tab stays instant; completion marks
  the current aspect from the other omens. During an invocation the auspex sees the
  target aspect; outside one it is empty.
- A rite never invoked through servitor has no inscriptions.
- Augury prints one detailed JSON shape: `aspect`, `standing`, `inscriptions`,
  `last_rite`, every omen, `taint`, `desecrated` and — when desecrated — `former`.
  Neither desecration nor a failed last invocation is an error.
- Hand-converted rites need their sanctum markers rewritten too, because the marker
  text changed along with everything else.
