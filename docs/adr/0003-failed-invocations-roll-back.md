# Failed invocations revert

A rite mixes file changes, which servitor can undo, with shell commands, which it
cannot. We decided that an invocation first runs a **pre-flight** that performs every
check without side effects (rendering scripture, reading tomes, resolving
placeholders, tether and transcription safety, tongue availability) and aborts with
nothing changed on any problem. When a step then fails, servitor performs a
**reversion** of steps *k … 1* in reverse order: sanctums, transcriptions and tethers
restore their previous state from memory captured during the invocation;
incantations and litanies run their optional `reversion` command — including the step
that failed, since it may have done part of its work. If a reversion itself fails,
servitor continues with the remaining ones, reports every outcome, and the rite shows
as corrupted until an aspect is invoked again.

## Considered Options

- **Stop and report** without reverting — recommended during design and rejected by
  the owner. The concern that undoing files after a command already acted on them
  leaves the system half-switched (e.g. kitty reloaded from the new theme while the
  tether points back to the old one) is answered by command reversions such as
  re-running the former aspect's hooks.
- **`allow_failure` per step** — rejected as muddying the model: a step that may fail
  says so in its own command (`pkill -USR1 kitty || true`).

## Consequences

- Incantations and litanies have a `patience` (default 30s, set in the settings); when
  it runs out their process group is killed and the step fails. Background children
  of successful steps keep running.
- Reversion commands may use `{{former}}` and should be safe to run even if their step
  changed nothing.
- `--foresee` renders everything (diffs, tether changes, commands) and performs
  nothing, not even the auspex.
