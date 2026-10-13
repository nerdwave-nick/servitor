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
| format version | pattern | `"pattern": "Mark II"`; `"Mark I"` is read only to be converted |
| format migration (Mark I → Mark II) | conversion | automatic, once, on first read by a ritual that may write; the original kept in `rites/.mark-i/` (settings: `.mark-i/servitor.json`) |
| switch / profile (named config) | rite | file in `rites/` |
| state | aspect | `aspects` |
| one-shot action (no state) | edict | `"edict": true` (instead of `aspects`) |
| description | purpose | `purpose` |
| ordered step list | liturgy | `liturgy` (an edict's alone) |
| managed block in a file | sanctum | `"designation": "sanctum"` |
| block identifier | ward | `ward` |
| comment prefix / suffix | glyph / closing glyph | `glyph`, `closing-glyph` |
| create file if missing | consecrate | `consecrate` |
| target file | vessel | `vessel` |
| write whole file | transcription | `"designation": "transcription"` |
| file content | scripture | a sanctum's or transcription's omen (Mark I key `scripture`) |
| content from a file | tome | `{"tome": "path"}` |
| fill placeholders in referenced content | illuminate | `"illuminate": true` |
| file mode | seal | `seal` |
| force overwrite | zeal | `zeal` (alias `force`) |
| symlink / target | tether / anchor | `"designation": "tether"`, `tether` (bound name); the anchor is its omen |
| shell command | incantation | `"designation": "incantation"`, `words` |
| elevated item (runs as root) | exalted | `"exalted": true` on a subject, observance or edict entry |
| root-only read of a state item | exalted auspex | `"auspex": {"scan": …, "exalted": true}`; spoken by the exalted servitor alone (`read`, protocol 3) |
| root helper process (one per invocation) | exalted servitor | hidden mode of the root-owned `/usr/bin/servitor` |
| approve elevated steps (root-owned copy) | sanction | `servitor sanction [<rite>]`; `/var/lib/servitor/sanctions/<uid>/<rite>.json` |
| files an elevated script reads (pinned in the approval) | relics | `relics` (list of paths in the rite's directory) |
| approval digest | sigil | JSON `sigil`; shown in prompts, the sanction list and the chronicle |
| revoke an approval | abjure | `servitor abjure <rite>` / `--all` |
| approval matches / changed / missing / leftover | sanctioned / lapsed / unsanctioned / forsaken | sanction standing in census and augury |
| elevation tool (sudo / run0 / pkexec) | intercessor | settings `intercessor`: `auto` / `sudo` / `run0` / `pkexec` |
| script file | litany | `"designation": "litany"`, `scroll` |
| script arguments | offerings | `offerings` (list; templated; leading `~/` or lone `~` → home; no other shell parsing) |
| shell | tongue | `tongue` (default bash) |
| timeout | patience | `patience` |
| rollback | reversion | restores what was read; `reversion` words only on observances and edict entries |
| notifications | vox-cast | rite key `vox-cast`, a list (`progress`, `success`) |
| state item | subject | `subjects` (list; each `name`, `designation`, `omens`) |
| type discriminator | designation | `designation` (`sanctum`, `transcription`, `tether`, `machine-spirit`; `incantation`, `litany`) |
| item read and set by commands | machine-spirit | `"designation": "machine-spirit"` |
| read command | scan | `auspex.scan` |
| per-state check, exit 0 = yes | trials | `auspex.trials` (keyed by omen) |
| set command | enact | `enact` (keyed by omen) |
| named hook | observance | `observances` (list; each `name`, `designation`) |
| before / after hooks | introit / recessional | `introit`, `recessional` (observance names) |
| hook guards: target / previous state | to / from | `to`, `from` |
| hook command / script | words / scroll | `words`, `scroll` |
| value being set or checked | omen | `{{omen}}`, `SERVITOR_OMEN` (enact words and trials only) |
| desktop notifications (auto / always / never) | vox | settings `vox`: `auto` / `notify-send` / `off` |
| metadata key/value | inscription | `inscriptions` (`purpose`, `mandatory`, `decrees`) |
| per-state default values | decrees | `decrees` |
| per-state value | aspect map | object keyed by aspect, `"*"` fallback |
| state probe command | auspex | a machine-spirit's `auspex` (`scan`, `trials`, `patience`, `exalted`) |
| observation (evidence of state) | omen | JSON `omens`; the mark an aspect leaves on a subject |
| desired state per item | omens | `omens` (aspect map: the omen of each aspect) |
| states fitting every item (intersection) | fitting | augury JSON `fitting` |
| cached / last-read value | recorded omen, stale | data-slate `omens` (with `boot`); JSON `stale`, `read_at`; census table `~` |
| items changed by an invocation | changed | chronicle `changed` |
| state ledger file | data-slate | `$XDG_STATE_HOME/servitor/data-slates/<rite>.json` |
| status | standing | JSON `standing` |
| applied / never applied / inconsistent | performed / dormant / corrupted | standing values |
| last run done (no state) | proclaimed | standing value (edicts only) |
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
| lock (exclusive, per rite / vessel) | dominion | `$XDG_STATE_HOME/servitor/dominions/` |
| step index (edicts) | verse | JSON `verse` |
| item named in JSON | subject / observance / verse | `{"subject": "<name>", "designation": …}` / `{"observance": "<name>"}`; an edict's step by its `verse` |
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

The one-shot action is an **edict**, not a "deed": `invocation.Deed` already names
one performed step of an invocation in the code (`Outcome.Deeds`). See
`docs/adr/0005-edicts-are-rites-without-aspects.md`.

Sanctum markers:

```
<glyph> +++ begin of sanctum <ward> -- aspect|<aspect> +++ <closing-glyph>
<glyph> +++ end of sanctum <ward> +++ <closing-glyph>
```
