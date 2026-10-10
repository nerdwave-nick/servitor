# The Liturgy of Commits

> *+++ Thought for the day: A commit without a type is a prayer without a
> recipient. +++*

Commits to servitor follow [Conventional Commits](https://www.conventionalcommits.org/)
in structure, but speak the tongue of the Adeptus Mechanicus. Every message is
a small rite and must be performed correctly:

```
<type>(<scope>)!: <description>

<body>

<footers>
```

* **type**: one of the liturgical types below (required)
* **scope**: the part of the machine you touched (optional)
* **!**: marks a schism, a breaking change (optional)
* **description**: imperative mood, lower case, no trailing period, the whole
  header within 72 characters (*"purge the nested ward"*, not *"Purged the
  nested ward."*)
* **body**: why the rite was needed (optional, wrapped at 72)
* **footers**: references, absolutions, schisms (optional)

## Quick reference: types

| Liturgical   | Standard      | When to use                                                 | Version bump |
|--------------|---------------|-------------------------------------------------------------|--------------|
| `consecrate` | `feat`        | a new capability, rite, flag or key binding                 | minor        |
| `purge`      | `fix`         | heresy (a bug) is cleansed                                  | patch        |
| `augment`    | `perf`        | the machine spirit runs faster or leaner                    | patch        |
| `ward`       | `security`    | defences are raised against the xenos                       | patch        |
| `reforge`    | `refactor`    | same behaviour, new shape                                   | none         |
| `anoint`     | `style`       | formatting only; sacred oils, no change in behaviour        | none         |
| `ordeal`     | `test`        | adding or amending the trials (tests)                       | none         |
| `transcribe` | `docs`        | README, help texts, comments, this very scripture           | none         |
| `fabricate`  | `build`       | the forge itself: build system, Go version, tooling         | none         |
| `tithe`      | `build(deps)` | dependencies are paid their due (bumps, additions, removals) | none         |
| `vigil`      | `ci`          | the pipeline servitors' ceaseless watch                     | none         |
| `maintain`   | `chore`       | the Rite of Maintenance: anything that fits nowhere else    | none         |
| `abjure`     | `revert`      | a previous commit is formally recanted                      | as reverted  |

A **schism** (breaking change) always bumps the major version, whatever the
type.

## Reverse lookup: standard → liturgical

| Standard | Liturgical |
|---|---|
| `feat` | `consecrate` |
| `fix` | `purge` |
| `perf` | `augment` |
| `security` | `ward` |
| `refactor` | `reforge` |
| `style` | `anoint` |
| `test` | `ordeal` |
| `docs` | `transcribe` |
| `build` | `fabricate` |
| `build(deps)` / `deps` | `tithe` |
| `ci` | `vigil` |
| `chore` | `maintain` |
| `revert` | `abjure` |
| `BREAKING CHANGE:` | `SCHISM:` |
| `Refs:` | `Decree:` |
| `Closes:` / `Fixes:` | `Absolves:` |
| `Reviewed-by:` | `Sanctioned-by:` |

## Choosing the type

Ask the questions in order; the first *yes* names your rite.

1. Does it undo an earlier commit? → `abjure`
2. Can users do something they could not before? → `consecrate`
3. Did something behave wrongly and now behaves rightly? → `purge`
4. Is it a security hardening? → `ward`
5. Is it only faster or lighter? → `augment`
6. Does it only touch tests? → `ordeal`
7. Does it only touch documentation or help text? → `transcribe`
8. Does it only bump dependencies? → `tithe`
9. Does it only touch the build or the Go toolchain? → `fabricate`
10. Does it only touch CI? → `vigil`
11. Does it only change formatting? → `anoint`
12. Does it restructure code without changing behaviour? → `reforge`
13. Otherwise → `maintain`

## Scopes

Scopes name the part of the machine. They follow the lexicon of servitor
itself.

| Scope         | Covers                                              | Code                          |
|---------------|-----------------------------------------------------|-------------------------------|
| `librarium`   | loading and validating rite definitions             | `internal/librarium`          |
| `sanctum`     | wards, inscriptions, marker parsing and rendering   | `internal/sanctum`            |
| `invoke`      | applying aspects, purging sanctums                  | `internal/invocation`, `invoke` |
| `augury`      | reading inscriptions back                           | `augury` command              |
| `census`      | listing rites                                       | `census` command              |
| `inquisition` | verification and diagnostics                        | `inquisition` command         |
| `cogitator`   | the TUI                                             | `internal/tui`                |
| `lexicon`     | vocabularies, help texts, Thoughts for the Day      | `internal/lexicon`, CLI help  |
| `completion`  | shell completion litanies                           | completion callbacks          |
| `codex`       | README and other documentation                      | `*.md`                        |

Omit the scope when a change touches the whole machine.

## Schisms (breaking changes)

A schism breaks faith with existing users: config keys, file markers, CLI
names or flags, JSON output or exit codes that change incompatibly. Mark it
twice: a `!` before the colon, and a `SCHISM:` footer explaining how the
faithful must adapt.

```
consecrate(librarium)!: read rites only from rites/

SCHISM: definitions kept in switches/ are no longer read. Move them to
rites/ or face the Inquisition.
```

## Footers

| Footer            | Purpose                                                     |
|-------------------|-------------------------------------------------------------|
| `SCHISM:`         | describes a breaking change and the path of atonement       |
| `Decree: #12`     | references an issue or discussion                           |
| `Absolves: #7`    | closes an issue when merged                                 |
| `Sanctioned-by:`  | the reviewer who blessed the change                         |
| `Co-authored-by:` | unchanged, as GitHub's machine spirits read it literally   |

## Exempla

```
consecrate(cogitator): add replicate binding to clone rites
purge(sanctum): stop nested wards from devouring the vessel
augment(augury): read each vessel only once per augury
ward(librarium): refuse rite names that escape the Librarium
reforge(invoke): share the planning step between invoke and purge
anoint: perform gofmt upon the faithless files
ordeal(cogitator): drive every wizard path through the key harness
transcribe(codex): record the lexicon of sacred and profane names
fabricate: raise the forge to Go 1.27
tithe: bump bubbletea to v2.0.10
vigil: run the inquisition on every pull request
maintain: perform the Rite of Maintenance on go.sum
abjure: consecrate(lexicon): add Thought for the Day about merge conflicts
```

A full rite with body and footers:

```
purge(sanctum): keep the blank line when a sanctum ends the vessel

Removing a sanctum appended to an empty vessel left a stray newline,
so purge followed by invoke drifted by one line per cycle.

Absolves: #7
Sanctioned-by: Magos Reviewer <magos@example.com>
```

## Before invoking automation

Release and changelog tools (semantic-release, release-please, commitlint,
git-cliff) only know the standard types out of the box. Until they are taught
this liturgy, they will reject or misfile these commits. Map the types and
footers above in their configuration before relying on them for versioning.

*Glory to the Omnissiah. Squash with reverence.*
