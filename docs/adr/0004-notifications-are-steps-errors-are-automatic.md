# Vox-casts are steps; heresies are automatic

Rites are often invoked from hotkeys or launchers where no terminal shows their
output. We decided that progress and success are reported by **vox-cast steps**
(`{"vox-cast": "progress"}`, `{"vox-cast": "success"}`) placed where the rite author
wants them, while **failures are reported automatically**. Each invocation shows at
most one desktop notification and updates it in place (`notify-send -p`, then
`-r <id>`). With a terminal attached, the same messages are printed instead of sent.
Messages are drawn at random from built-in grimdark template lists. Every invocation
is also appended as one JSON line to the chronicle (default
`$XDG_STATE_HOME/servitor/chronicle.jsonl`).

## Considered Options

- **A per-rite notifications object** (error/success switches) — rejected as an
  overloaded key; placing steps is more flexible and clearer.
- **A failure vox-cast step** — rejected: a failure aborts the liturgy, so a failure
  step after the failing step never runs, and one before it fires on every run.

## Consequences

- A `success` vox-cast that is not the last step is an impurity.
- Progress messages describe the previous step and count real steps ("step x / n"),
  which mako also renders as a progress bar.
- The settings file can silence desktop notifications globally (`"vox": "off"`).
