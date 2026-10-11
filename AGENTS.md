# Notes for coding agents

servitor speaks only grimdark (Warhammer 40k / Adeptus Mechanicus) — in help, messages,
README, `CONTEXT.md`, configuration keys, vessel markers, JSON output and flags. This
file is the **only** place with plain-language equivalents. Use it to understand the
domain and the code; never add plain words or plain glosses to anything a user reads
(help, messages, README, CONTEXT.md, JSON keys, flags).

- Domain glossary (grimdark): `CONTEXT.md`
- The codex (`servitor expound`): one grimdark passage per topic in
  `internal/codex/passages/<topic>.txt`; tests fail when a ritual, rune, step or key
  lacks its passage, or when a passage speaks a plain word of the lookup below
- JSON Schemas of rites and settings: `schema/*.schema.json` (embedded, recited by
  `servitor expound --schema`); tests fail when their keys drift from `internal/librarium`,
  when they judge a scripture otherwise than the librarium, or when they speak a plain word
- Design decisions: `docs/adr/`
- Planning history: lit project `feat/servitor`, map "Plan: rite format v2 (steps,
  elevation, aspect maps)"

## Plain → grimdark lookup

| Plain concept | Grimdark term | Key / flag / ritual |
|---|---|---|
| CLI tool | servitor | `servitor` |
| config directory | Librarium | `--librarium` / `-l`, `SERVITOR_LIBRARIUM` |
| user config file | settings | `<Librarium>/servitor.json` |
| format version | pattern | `"pattern": "Mark I"` |
| switch / profile (named config) | rite | file in `rites/` |
| state | aspect | `aspects` |
| description | purpose | `purpose` |
| ordered step list | liturgy | `liturgy` |
| managed block in a file | sanctum | step key `sanctum` |
| block identifier | ward | `ward` |
| comment prefix / suffix | glyph / closing glyph | `glyph`, `closing-glyph` |
| create file if missing | consecrate | `consecrate` |
| target file | vessel | `vessel` |
| write whole file | transcription | step key `transcription` |
| file content | scripture | `scripture` |
| content from a file | tome | `{"tome": "path"}` |
| fill placeholders in referenced content | illuminate | `"illuminate": true` |
| file mode | seal | `seal` |
| force overwrite | zeal | `zeal` (alias `force`) |
| symlink step / target | tether / anchor | `tether`, `anchor` |
| shell command step | incantation | `incantation` |
| script file step | litany | `litany` |
| script arguments | offerings | `offerings` (list; templated; aspect map; leading `~/` or lone `~` → home; no other shell parsing) |
| shell | tongue | `tongue` (default bash) |
| timeout | patience | `patience` |
| rollback (per step) | reversion | `reversion` |
| notification step | vox-cast | `vox-cast` (`progress`, `success`) |
| desktop notifications (auto / always / never) | vox | settings `vox`: `auto` / `notify-send` / `off` |
| metadata key/value | inscription | `inscriptions` (`purpose`, `mandatory`, `decrees`) |
| per-state default values | decrees | `decrees` |
| per-state value | aspect map | object keyed by aspect, `"*"` fallback |
| state probe command | auspex | `auspex` (`{"rite", "patience"}` long form) |
| observation (evidence of state) | omen | JSON `omens` |
| state ledger file | data-slate | `$XDG_STATE_HOME/servitor/data-slates/<rite>.json` |
| status | standing | JSON `standing` |
| applied / never applied / inconsistent | performed / dormant / corrupted | standing values |
| edited by hand (drift) | tainted | JSON `taint` |
| changed outside servitor | desecrated | JSON `desecrated` |
| invalid definition | heretical | standing value |
| error / warning | heresy / impurity | inquisition output |
| apply a state | invoke | `servitor invoke <rite> <aspect>` |
| read status/metadata | augury | `servitor augury` |
| list | census | `servitor census` |
| validate | inquisition | `servitor inquisition` |
| TUI | cogitator | `servitor cogitator` / `servitor` |
| explain / help command | expound | `servitor expound [topic]`; `help`, `-h`, `--help` are hidden aliases |
| the documentation itself | codex | flavour: "as written in the codex" |
| invocation log | chronicle | `$XDG_STATE_HOME/servitor/chronicle.jsonl`; settings `chronicle`, `--chronicle` (alias `--log`), `SERVITOR_CHRONICLE` |
| step index | verse | JSON `verse` |
| step kind + target in JSON | the step's own key | e.g. `"tether": "<path>"`; auspex omen `"auspex": true` |
| outcome / result | verdict | JSON `verdict`: `triumph` (ok), `reverted` (failed, undone), `faltered` (failed, undo incomplete) |
| timestamp | at | JSON `at` (RFC 3339) |
| failing step | fell_at | chronicle `fell_at` |
| rite file path | recorded_in | census JSON `recorded_in` |
| last outcome | last_rite | JSON `last_rite` |
| dry run | foresee | `--foresee` / `-f` |
| quiet | silence | `--silence` / `-s` |
| JSON output | binharic | `--binharic` (alias `--json`) |
| skip target-file checks | spare vessels | `--spare-vessels` |
| current state placeholder / env | aspect | `{{aspect}}`, `SERVITOR_ASPECT` |
| previous state | former | `{{former}}`, `SERVITOR_FORMER_ASPECT` |
| metadata placeholder / env | inscription | `{{inscription.x}}`, `SERVITOR_INSCRIPTION_X` |
| error text placeholder | heresy | `{{heresy}}` |

Sanctum markers:

```
<glyph> +++ begin of sanctum <ward> -- aspect|<aspect> +++ <closing-glyph>
<glyph> +++ end of sanctum <ward> +++ <closing-glyph>
```
