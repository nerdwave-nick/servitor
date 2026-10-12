# Exalted steps are performed from a sanction

Some steps need the machine's own authority, not the invoker's: writing
`/sys/devices/system/cpu/smt/control`, `sysctl -w`, files under `/etc`, starting a
system service. Under ADR 0001 such steps were out of scope, so rites wrote `sudo`
into their incantations. That has three flaws:

- it cannot prompt without a terminal, so a hotkey or picker invocation fails;
- sudo's cached timestamp lets anything on the same terminal act as root afterwards;
- root runs whatever the rite, a file the invoker can write, says at that moment.

We decided — settled by the owner on 2026-10-11 and 2026-10-12, the transport verified
in a VM on 2026-10-12 (`docs/research/elevation-verification.md`) — that a step may be **exalted**,
and that exalted steps are performed by a short-lived **exalted servitor** (the helper below) from a
root-owned copy the master has **sanctioned**. The evidence and the threat model live
in the elevation research: the snapshot on lit issue 0190b605, the decision on lit
issue 12d5c7fe, and `docs/research/elevation-tools.md`.

```jsonc
{
  "pattern": "Mark I",
  "purpose": "enable or disable gaming optimization",
  "aspects": ["on", "off"],
  "auspex": "[ \"$(sysctl -n kernel.split_lock_mitigate)\" = 0 ] && echo on || echo off",
  "liturgy": [
    {"incantation": {"on": "echo off > /sys/devices/system/cpu/smt/control",
                     "*":  "echo on > /sys/devices/system/cpu/smt/control"}, "exalted": true},
    {"incantation": {"on": "sysctl -wq kernel.split_lock_mitigate=0",
                     "*":  "sysctl -wq kernel.split_lock_mitigate=1"}, "exalted": true},
    {"vox-cast": "success"}
  ]
}
```

- **Marking:** `"exalted": true` on any step but a vox-cast, in rites and edicts alike.
  Every other step is performed as today, with the invoker's authority.
- **Sanction** (ritual `servitor sanction <rite>`, one prompt):
  - The servitor shows the rite's exalted verses as a canonical unit, or its difference
    from the standing sanction. The unit is resolved for every aspect: aspect maps, the
    absolute tongue, and the bytes of every scroll, tome and named sibling file.
  - It then installs a root-owned copy with its digest, strict JSON at
    `/var/lib/servitor/sanctions/<uid>/<rite>.json`.
  - `servitor sanction` alone lists every sanction. `servitor abjure <rite>` (or
    `--all`) withdraws one.
  - Census and augury show each rite's standing towards its sanction:
    - *sanctioned*: its exalted verses match the sanction;
    - *lapsed*: they changed since;
    - *unsanctioned*: it has exalted verses but no sanction;
    - *forsaken*: a sanction whose rite is gone or holds no exalted verse.

    *(Owner decision 2026-10-12: lapsed and forsaken replace the earlier "drifted" and
    "orphaned", because drift is the plain gloss for tainted and desecrated. The path
    replaces the earlier `approved/<uid>/<rite>.plan`.)*
- **Invocation:**
  - The pre-flight stays with the invoker. It compares the rite's exalted verses with the
    sanction and refuses **before any prompt** when they differ or no sanction exists.
  - After the pre-flight and before verse 1, **one** exalted helper is started for the
    whole invocation: one prompt, however many exalted verses the rite holds.
  - The helper performs the exalted verses **from the sanctioned copy**. The invoker's
    servitor only drives it (perform the next verse, revert, finish, abort), so the
    written order of exalted and common verses is kept.
  - Every invocation prompts anew. No authentication is reused across invocations
    (owner requirement).
  - **Dominion fails closed for exalted rites.** An invocation with exalted verses that
    cannot seize its dominion refuses instead of proceeding. The dominions of exalted rites
    and their vessels lie where only root writes. Common rites keep ADR 0006 unchanged.
    *(Owner decision 2026-10-12.)*
- **What crosses the boundary:**
  - Words fixed at sanction may be spliced into exalted text: aspects, and decrees of
    inscriptions whose rune is forbidden under exaltation.
  - Words given at invocation reach exalted verses only as environment variables:
    runes, and `{{former}}` (`SERVITOR_INSCRIPTION_<KEY>`, `SERVITOR_FORMER_ASPECT`). Their
    placeholder in exalted text, reversions, paths, offerings or illuminated scripture is
    heresy.
- **Intercessor** (settings key `intercessor`: `auto` | `sudo` | `run0` | `pkexec`). The
  helper is always the fixed, root-owned `/usr/bin/servitor`, never the running binary.
  - `auto` with a terminal: `sudo -k` with a prompt naming the rite and digest, else run0,
    else pkexec.
  - `auto` without a terminal: run0 (`--unit=servitor-<rite>-<digest8>`), else pkexec.
  - No doas.
  - The cogitator gives its terminal to sudo while it asks.
- **Installation:**
  - A PKGBUILD in the repository, installed with `makepkg -si`.
  - The servitor refuses to exalt when `/usr/bin/servitor` or a directory above it is not
    root-owned, or is writable by others.
  - A protocol version is exchanged, so a development build can drive the installed
    helper.
- **The helper:**
  - **Channel:** a socketpair on fd 0/1, never a pipe; the parent sets
    `PR_SET_DUMPABLE=0`. Strict, size-capped JSON lines.
  - **Identity:** taken from `SO_PEERCRED`, or from `SUDO_UID` when sudo's I/O logging
    interposes pipes.
  - **Start-up:** the helper changes to `/`, sets umask 022, resets its rlimits, and
    ignores the terminal's signals; SIGTERM means abort.
  - **Paths:** exalted paths must pass the trusted-path rule: every directory of the
    resolved path is root-owned and not writable by others. Writes are fd-relative,
    atomic, and keep owner and mode. ACL or SELinux attributes fail closed. `/proc`,
    `/sys` and `/dev` are no vessels; incantations write them.
  - **What leaves the helper:** no vessel content, only verdicts and digests. It never
    writes the invoker's data-slate, chronicle or state, and journals each act as root.
- **Exalted speech:**
  - **Where:** a fresh root-owned `/run/servitor/<digest>/`, holding only the sibling
    files a step names, whose bytes are part of the sanction.
  - **How:** an allowlisted environment, `bash --noprofile --norc`, a new session, stdin
    from `/dev/null`, its own process group.
  - **Leftovers:** processes it leaves behind are slain when the invocation ends. A
    lasting root process belongs in a unit, e.g. `systemd-run --unit=…`.
- **Exalted tethers** may resolve only through root-owned directories:
  `/etc/x → ~/dotfiles/x` is heresy whose words name an exalted transcription as the
  remedy.
- **When the invoker's servitor dies:** the helper reads the end of its channel before
  `finish` as a fall. It reverts its performed exalted verses in reverse order, journals,
  and ends.
- **Foresight and readers:**
  - `--foresee` never exalts. A vessel only root may read shows its stat and the planned
    digest as "contents hidden — change unknown".
  - Under `sudo servitor …` the reading rituals (foresight, augury, census, inquisition)
    read the invoker's Librarium via `SUDO_USER`.
  - Performing invocations, sanction, abjure and the cogitator refuse to run as root.
- **The pattern stays `"Mark I"`:** the change is additive. A rite without exalted verses
  keeps its meaning, and ADR 0001's "out of scope" consequence is superseded.

## Considered Options

- **`sudo` inside incantations** (the status quo): rejected for the three flaws above.
- **Root re-reads the rite and checks a hash sent by the invoker:** rejected. The hash is
  unauthenticated, and root would parse files the invoker controls (the class of
  CVE-2025-32463). Root executes only its own sanctioned copy.
- **A pipe as the channel:** rejected. Another process of the same user can reopen it
  through `/proc/<pid>/fd` and inject; a socketpair cannot be reopened (reproduced
  2026-10-11).
- **A resident root service** that performs exalted verses on request, authorised by
  polkit: deferred. It would remove the intercessor matrix, at the cost of a root
  endpoint that is always listening. A servitor daemon is not planned
  (research/servitor-daemon, 2026-10-11); both shapes can share one executor.
- **A prompt per exalted verse:** rejected; one per invocation.
- **Passwordless performing of sanctioned rites:** not in this decision. It would let any
  process of the invoker reach root without a prompt.
- **doas:** rejected; unmaintained since 2022, and it needs `/etc/doas.conf`.

## Consequences

- Two installation steps, and a development build needs a matching installed servitor.
- Exalted scripts must name their sibling files. Editing an exalted verse needs a new
  sanction before the next invocation, which shows the difference; common verses need
  none.
- Inscription values were spliced into `bash -c` verbatim. That is harmless for common
  verses, but it is why runtime values reach exalted verses only through the
  environment. A sanctioned script that `eval`s such a variable still runs it as code, so
  the sanction review covers that.
- **Verified in a VM** (sudo 1.9.17p2, polkit 127, systemd 262):
  - the socketpair reaches the helper unchanged through `sudo -k`, pkexec and run0;
  - `sudo -k` is essential, because without it a sibling reuses the timestamp;
  - pkexec and run0 left nothing reusable.

  Not verified: the graphical polkit agent, and a long-lived cogitator invoking run0
  twice.
