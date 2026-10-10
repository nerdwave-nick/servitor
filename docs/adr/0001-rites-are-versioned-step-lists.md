# Rites are versioned liturgies of steps

The original rite format (servitor ≤ v0.2.0) could only rewrite sanctums in files
(`files[]`), and its global entry/exit hooks were removed as overkill. Real use cases
need more: a do-not-disturb toggle is a shell command, and switching the nfluff theme
means re-pointing a symlink and running per-theme hooks. We decided that a rite is an
**ordered liturgy of steps** of six kinds — `sanctum`, `transcription`, `tether`,
`incantation`, `litany`, `vox-cast` — performed in the order written, and that every
rite file declares its format as a pattern, `"pattern": "Mark I"`.

```jsonc
{
  "pattern": "Mark I",
  "purpose": "The visage of the machine",
  "aspects": ["default", "finii", "porpl"],
  "inscriptions": {"reason": {"purpose": "why the rite was invoked"}},
  "liturgy": [
    {"incantation": "niri msg action do-screen-transition -d 200"},
    {"tether": "~/.local/share/nfluff/current-theme", "anchor": "~/.config/nfluff/themes/{{aspect}}"},
    {"incantation": "source ~/.config/nfluff/themes/{{aspect}}/hooks", "tongue": "bash",
     "reversion": "source ~/.config/nfluff/themes/{{former}}/hooks"},
    {"vox-cast": "success"}
  ]
}
```

- **One convention for per-aspect values:** any field is either a single value or an
  aspect map (`{"on": …, "off": …, "*": …}`); strings may use `{{…}}` placeholders
  (`{{aspect}}`, `{{former}}`, `{{rite.name}}`, `{{inscription.<key>}}`).
- **Scripture** of sanctums and transcriptions is inline (string or list of lines) or
  a tome (`{"tome": "snippets/x.kdl"}`), resolved against the rite file's directory,
  read freshly during every pre-flight and inserted verbatim unless it opts into
  placeholders with `"illuminate": true`. A literal `{{` is written `{{{{`; unknown
  placeholders are a heresy rather than rendering empty.
- **Deleting per aspect:** `"scripture": null` deletes a transcribed vessel and
  `"anchor": null` removes a tether; a sanctum is never deleted (empty scripture keeps
  its header, which records the aspect).
- **Transcriptions keep no backups** of vessels they replace, so they are careful
  instead: they replace a vessel only if its scripture matches one of the rite's
  aspects, and a tether replaces only symbolic links; anything else needs
  `"zeal": true` (alias `force`). Symlinked vessels are followed so that dotfile links
  survive, and missing parent directories are never created.
- **Incantations and litanies** are spoken in a configurable tongue, bash by default.
- **Sanctums vs. tethers:** sanctums are for small dynamic toggles inside configs that
  are not ours to restructure (mouse auto-hide, do-not-disturb); a tether is for
  swapping something we own wholesale (the nfluff theme directory). The theme stays a
  symlink because its consumers include a JSON file, which cannot hold a sanctum, and
  tracked dotfiles, which sanctums would dirty on every switch.
- **Everything speaks grimdark** — keys, markers, JSON, flags — with plain meanings
  documented in `CONTEXT.md`; `--json`, `--log` and `force` remain as aliases.

## Considered Options

- **Keep `files[]` and add hooks** — rejected: commands need per-aspect variation and
  an order relative to file changes.
- **Discover aspects from a directory** — rejected: adding a theme's name to the list
  once is cheaper than rules for naming, filtering and disappearing aspects. Aspects
  are always a hand-written list.
- **Interactive input** (an input step, or inscriptions that ask when missing) —
  rejected as too much complexity; callers prompt in their own scripts, e.g.
  `servitor invoke dnd on --reason "$(fuzzel --dmenu --prompt-only 'Reason: ')"`.
- **Migrating rites without a pattern** — rejected for now: there is a single user,
  who converts by hand. Such files are a heresy; the pattern keeps future migrations
  possible.
- **Plain configuration keys and a plain vocabulary mode** (`--no-grimdark`) — rejected:
  servitor speaks grimdark only; understandability comes from documentation instead.

## Consequences

- Elevated steps (sudo/pkexec/run0) are **out of scope** for this pattern; the
  research in `docs/research/elevation-tools.md` and the tracker map hold the work
  done so far.
