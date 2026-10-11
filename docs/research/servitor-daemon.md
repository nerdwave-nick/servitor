# Research: a servitor daemon, a root-owned rite store, and who performs invocations

Date: 2026-10-11. Machine: Arch Linux, kernel 7.2.3, niri, systemd 262, polkit 127
(graphical agent `io.elementary.desktop.agent-polkit` running), sudo 1.9.17p2, Go 1.27.1.
servitor at commit `56a34319afc2ac1e5df881a21947f0170cb5e687`.

Tracker (lit project `research/servitor-daemon`):
[Root daemon as sole executor and a root-owned rite store](lit:issue:5e19ed60-f002-4f49-b859-a8a75cf7ecab)
(this document's question), with first-pass answers for the six open research tickets
in §8. The owner decides on
[Decide: build the daemon or not, and its name](lit:issue:4c95eb6c-4f30-4c9b-8789-abf73fdbb8f1).

**Builds on** the elevation research (the only copy is the snapshot comment on
[Research: secure design for elevated steps](lit:issue:0190b605-476c-4bb2-bd18-6a3726c12590))
and the elevation decision of 2026-10-11 on
[elevated steps re-execute servitor through an elevation tool](lit:issue:12d5c7fe-0390-4412-ae9a-2e9116ff5024).
Its threat model (A1 = any process running as the same user, A2 = a careless rite, A3 = a
swapped binary, A4 = other local users, A5 = servitor bugs) and goals G1–G4 are used
unchanged here.

**Source tags:** **[local]** observed on this machine; **[src]** source at the named
version or commit; **[man]** local man page; **[doc]** upstream documentation;
**[consult]** the independent consultation (below), used only where re-checked or marked.
*Inference* marks reasoning that has not been verified.

**Consultation:** Codex CLI, model `gpt-6-astra`, effort high, read-only sandbox, session
`01a12bd9-b30c-7cd0-b495-bf08f7cfa64d`, neutral briefing without a verdict. Record and
paths: comment on the tracker ticket above. Its three code and polkit claims that
mattered were re-checked locally (§3.4, §5.3, §6).

**Outcome (owner, 2026-10-11):** the daemon is **not planned** for now, and the start-up fix
of §2 is taken up in feat/servitor
([Process start: bump go-runewidth to v0.0.30](lit:issue:31e466b7-0aa4-475b-94c5-6499ccba3ff7)).
If a daemon is revisited, the reason would be event-triggered rites or live state push,
as an optional `systemd --user` service.

## Question

The owner's proposal (2026-10-11):

> perhaps it would make sense to store rites in a root-owned sqlite or something similar,
> instead of filesystem json files. They're bothersome to create manually anyway. The user
> owned filesystem can be the source of imports and target for exports of rites, so that
> there's a non-authorative copy there, which is portable. Or the export could be done
> explicitly. But the daemon could have a root-owned authority (sqlite or
> root-owned-directory) for the actual authority of rites. The root-daemon could also do
> ALL executions, with user-owned servitor invocations just communicating to the daemon.
> This could speed things up and simplify the whole elevated command thing.

So, five questions:

1. Should a root daemon perform every invocation, with the CLI as a thin client?
2. Should rites live in a root-owned authoritative store (SQLite or a root-owned
   directory), with the user's files as import/export copies?
3. Would a privileged service make exalted steps simpler than the decided per-invocation
   helper?
4. Where would a daemon actually save time?
5. Would a store make rites easier to create?

## Verdict

The proposal bundles separate changes. They should be judged one by one:

| Part | Verdict |
|---|---|
| A root daemon that performs **every** step as root | **Reject.** Theme switches, `niri msg`, `notify-send`, `gsettings` and user files need the user's session. Running them as root gains no security and creates confused-deputy risks (§3). |
| A root daemon that coordinates everything but spawns unprivileged workers for ordinary steps | Defensible (cron works this way), but far more machinery than servitor needs. It also makes the daemon mandatory, which conflicts with the daemon project's "optional by design" (§3.5). |
| A root-owned authoritative store **for exalted content** | **Keep.** This is the decided `sanction` design: `/var/lib/servitor/approved/<uid>/<rite>.plan`. Root executes only what root holds. |
| A root-owned authoritative store **for every rite** | **Not needed for security**, and costly for the workflow. Against A1 it protects nothing an unprivileged rite could not do directly. Git/dotfiles editing becomes import/export with drift (§4). |
| SQLite instead of files | **Not justified at this scale.** Tens of small documents, one writer, and plain files are diffable, auditable and need no new dependency (§4.4). |
| Speed | **A daemon is not the lever.** 22 of servitor's 23 ms process start is one dependency's eager initialisation; with that fixed, an invoke takes about 6 ms (§2). Only auspex time remains, and a root daemon does not help with it. |
| Simpler elevation | **Partly true, but for a narrow service only.** A socket-activated root service that executes only sanctioned exalted plans and authorises each invocation through polkit `auth_admin` removes the sudo/run0/pkexec matrix. It also closes a gap found here: under polkit 127, an authentication through run0 is kept, and other children of the same servitor process on a terminal can use it (§5.3). It costs a listening root endpoint and its own polkit integration. |
| Easier creation | **A separate problem.** The cogitator already has a consecration wizard for creating, amending and replicating rites (§6). Storage location does not change how hard authoring is. |

**Recommended shape** (inference, ranked):

1. Ordinary rites stay user files in the Librarium, run by the user's servitor process,
   as today.
2. Exalted content stays under the sanctioned root-owned store, already decided.
3. For the root side, choose between the decided per-invocation helper and a small
   socket-activated **exalted service** (polkit mechanism). §5.4 compares them. Either way
   the run0 backend needs a fix: its kept authorisation is shareable by sibling processes (§5.3).
4. Drop the dependency's start-up cost first. Then decide on any daemon from measurements.
   If live state or event push is still wanted, an optional **user** service
   (`systemd --user`) is the right home, not a root one.

## 1. The proposal, split into its parts

There are three kinds of authority, and the proposal moves all three at once
[consult; inference]:

- **Definition authority:** which revision of a rite is in force.
- **Execution authority:** who performs which operation, as which user.
- **Observational authority:** what establishes the current aspect. ADR 0002 gives this
  to agreeing omens, not to a record of what was done.

A store can hold the first. A daemon can hold the second. Neither changes the third: a
database that records "invoked porpl" is the data-slate again, and ADR 0002 already
ranks it below every omen.

## 2. Where the time goes [local]

Benchmark: a synthetic Librarium of 31 rites in the scratch directory. 30 rites each have
a sanctum and a tether on scratch vessels; one rite has an auspex `sleep 0.3; echo on`.
Wall-clock time of whole processes, current build (`go build`, commit above):

| Command | Time |
|---|---|
| `servitor --version` | 23 ms |
| `servitor census --binharic --spare-auspex` (31 rites) | 25–30 ms |
| `servitor census --binharic` (one 0.3 s auspex) | 330–357 ms |
| `servitor invoke r01 on` (sanctum + tether) | 22–27 ms |
| `servitor invoke probe on` (rite with the auspex) | 327 ms |

`GODEBUG=inittrace=1 servitor --version` attributes **22 ms** of package initialisation
to `github.com/mattn/go-runewidth` v0.0.27. Every other package takes under 0.3 ms. Its
`init()` builds full lookup tables eagerly (`eastAsianWidth0`, `initStrictWidthLUT`)
[src go-runewidth v0.0.27 `runewidth.go:44-53`]. It arrives indirectly:
`internal/tui` → `charm.land/bubbles/v2/textarea` → go-runewidth (`go mod why`). Because
the TUI is linked into the one binary, every ritual pays the cost.

go-runewidth v0.0.30 initialises only the table below U+0300 and builds the rest lazily
[src go-runewidth v0.0.30 `runewidth.go`, `initStrictWidthLUTLow`, `initTables`]. Built
from a scratch copy with only that bump (`go get github.com/mattn/go-runewidth@v0.0.30`;
the repository is unchanged):

| Command | Time |
|---|---|
| `servitor --version` | 2.8–3.6 ms |
| `census --binharic --spare-auspex` (31 rites) | 4.6–6.6 ms |
| `census --binharic` (one 0.3 s auspex) | 308 ms |
| `invoke r01 on/off` | 5.5–7.8 ms |
| `invoke probe on` | 309 ms |
| `augury r05 --spare-auspex` | 5.3 ms |

(Tests were not run against the bumped copy; it is a measurement only.)

**Consequences:**
- A thin client would still be a process: about 3 ms to start, plus a socket round trip.
  A daemon can save at most the 2–4 ms of real work left. *Inference:* that is not worth
  a second way of computing state.
- What remains is **auspex time**. Census already runs auspexes in parallel, at most 16 at
  once [src `internal/rituals/gather.go`], and `census --vigil` streams results before
  the auspexes settle (feat/servitor-caller-interface S3/S4, implemented). A cache of
  observations could serve auspex results instantly. That is the original daemon
  question, and it needs no root (§8).
- `invoke` always runs its own pre-flight observation, auspex included (caller-interface
  principle: "Pending or stale information never drives an invocation"). A daemon cannot
  shorten `invoke probe on` without breaking that principle.

## 3. A root daemon that performs every invocation

### 3.1 What ordinary rites need from their environment

- Every incantation and litany inherits servitor's own environment, plus the
  invocation's variables (`environ()`, [src `internal/invocation/speak.go:123-140`]).
- On this machine the variables user rites rely on (`WAYLAND_DISPLAY`, `NIRI_SOCKET`,
  `DISPLAY`, `DBUS_SESSION_BUS_ADDRESS`, `XDG_RUNTIME_DIR`, `XDG_CURRENT_DESKTOP`) are set
  in the systemd **user** manager [local `systemctl --user show-environment`]. niri
  imports them there. The system manager has no per-user session.
- `~` in anchors and offerings is the invoker's home [src `internal/librarium/model.go:164`].
  Data-slates, chronicle and dominions live in the user's `$XDG_STATE_HOME`
  [src `internal/librarium/orders.go:72-80`, `internal/dominion/dominion.go`].
- The vox chooses notify-send by whether the invocation has a terminal
  [src `internal/librarium/settings.go:23`]. A daemon has none, so the client would have to
  tell it.

Context that matters per operation [consult; niri IPC docs, Wayland client API, GIO docs]:

| Operation | Needs |
|---|---|
| `niri msg` | the compositor's IPC socket (`NIRI_SOCKET`) |
| Wayland clients | the display socket and the runtime directory |
| `notify-send`, `gsettings` | the session D-Bus and the user's configuration |
| `systemctl --user` | the right user manager |
| writes to user files | the user's credentials, ownership and path resolution |

### 3.2 What breaks or becomes risky as root

- **Wrong identity for user files.** Root writing into `~/.config` creates root-owned
  files unless it chowns, and follows symlinks the user controls. Those are the confused
  deputy and CWE-59/CWE-367 classes, already found in `world.go` by the elevation
  research (§3.4 there). `fs.protected_symlinks` does not cover home directories.
- **Environment import is not a fix.** Copying the client's environment into a root
  process lets the caller choose code (`LD_*`, `BASH_ENV`, `GIO_MODULE_DIR`, `PATH`). That
  is the class of Flatpak CVE-2021-21261, where caller-controlled environment reached a
  more privileged execution context [doc GHSA-4ppf-fxf6-vxg2; consult].
- **Root gains nothing.** Against A1, anything an ordinary rite does, A1 can already do as
  the user. Performing it as root adds privilege without adding protection.
- **Prior art does not do it.** Daemons that run user-defined jobs drop to the job's
  owner before executing: cronie changes to the job owner in `do_command.c`
  [src cronie 1.7.2; consult]. launchd separates per-user agents from system daemons
  [doc Apple launchd guide]. Flatpak keeps applications in the user's process and sends
  only installation operations to its root helper [src flatpak 1.16.1
  `system-helper/flatpak-system-helper.c`; consult].

### 3.3 The coordinator variant

A root daemon could coordinate while forking user workers (setresuid to the user, a
reconstructed session environment). That is how cron works. For servitor it would have to
own (inference):

- identity and session lookup per client;
- which compositor and session an invocation belongs to;
- worker lifetimes;
- rollback across mixed privileges;
- locks, outcomes and the chronicle written as the user.

Most rites are graphical-session rites, so almost every step would make the round trip
root → worker → user session. The existing design already has the user process as
coordinator and root as the narrow exception, which is the cheaper direction.

### 3.4 Nested invocations and serialisation

- A step may itself run `servitor invoke …`. Inscriptions of an "enclosing invocation"
  are cast out [src `internal/invocation/speak.go:117-122`], so this is anticipated.
- With one daemon queue, a nested invocation would wait on the queue its own parent holds:
  a deadlock (*inference*). The per-rite and per-vessel dominion (ADR 0006) does not have
  this problem.
- Today dominion is **best effort**. If a dominion cannot be seized, the invocation laments
  and proceeds without it [src `internal/rituals/dominion.go:96-104`, `covers` at `:126-145`;
  re-checked after the consultation pointed it out]. That is acceptable for user files. For
  exalted work it must fail closed, and the locks must live in a root-controlled namespace
  that the user cannot redirect through `$XDG_STATE_HOME` [consult; inference].

### 3.5 Conflicts with recorded constraints

- The daemon project says "optional by design: every ritual keeps working without it". A
  thin client that cannot invoke without the daemon breaks this. Socket activation makes
  the dependency convenient; it does not make it optional [consult].
- Caller-interface S7: invocations survive their **caller**. With a daemon owning
  invocations, the daemon's own restart or crash becomes a new failure mode. Undo data held
  in memory would be lost, and arbitrary commands cannot be replayed [consult; ADR 0003].
- **Never retry through a fallback.** If a client loses an ambiguous reply from a daemon
  that performs invocations, it must not re-run the invocation directly. The edict may
  already be proclaimed. Invocation IDs and outcome lookup would be needed [consult;
  inference].

## 4. A root-owned authority for rites

### 4.1 What it protects, and against whom

- **Exalted content:** a root-owned copy that root executes is the strong form. The
  elevation research (§8) already settled this as `sanction`.
- **Ordinary rites:** a root-owned copy protects nothing against A1. A1 can do whatever the
  rite would do, directly, as the user. It protects only against *unnoticed edits*, which a
  user-owned copy plus `git diff` already shows.
- **Mutation policy decides everything.** If any same-user process can replace a stored
  rite through an unauthenticated daemon API, root ownership protects nothing. If every
  edit needs administrator authentication, a theme tweak becomes an admin operation
  [consult]. The sanction design avoids the dilemma by gating only exalted content.

### 4.2 Precedents for "user-authored, root-held"

- **crontab** is the closest precedent for the proposal: users edit through `crontab -e`
  or import with `crontab <file>`, the spool is not user-writable, and crond runs each job
  as its owner. Its spool is root-held so that the **root daemon** trusts what it reads.
  The user gains nothing over a home-directory file. *(Details: §7.)*
- **visudo, `systemctl edit`:** edit a copy, validate, install. The consumer reads only the
  installed copy (elevation research §8.1).
- **Nix:** the store is root-owned and written only by the daemon; users write their
  configuration in their home and *build* into the store. home-manager follows this model
  for user configuration [elevation research §2 and §8.1].
- **systemd user units** live in `~/.config/systemd/user` and are run by the user's own
  manager. No root store is involved for per-user automation.

### 4.3 What moving every rite into a store would cost

- **ADR 0001 and the Librarium change.** Rites are files named by their file name; tomes
  and litany scripts are read by relative path, fresh, at invocation. "The Librarium
  forgets nothing that git remembers" is in the README's own voice. An import model needs
  an ADR covering activation, revisions and fallback [consult].
- **Two copies drift.** An edit to the exported file does not change the active rite until
  it is imported, and a later export can overwrite local edits. This is the
  sanctioned/drifted/unsanctioned/orphaned problem, spread over every rite instead of only
  exalted ones.
- **Comments are lost.** The librarium serializer writes plain JSON and drops JSONC
  remarks [src `internal/librarium/write.go:10`; consult]. An export would not round-trip
  the hand-written file unless the original text is stored as well.
- **Portability needs rules:** relative paths, `~`, and the bytes of tomes and scripts that
  a rite reads [consult].
- **The fallback must use the same revision.** If the daemon is down and the CLI reads the
  user's export instead, two different definitions can be in force [consult].

### 4.4 SQLite or a root-owned directory

| | Root-owned directory of canonical files | Root-owned SQLite |
|---|---|---|
| Authority against A1 | Path ownership plus mutation policy | The same |
| Inspection | `cat`, `diff`, `sudo ls` | needs `sqlite3` or an export |
| Atomic change of one plan | write temp, fsync, rename, fsync dir | a transaction |
| Atomic change of several | needs generations or a manifest | a transaction |
| Queries, history | the application builds them | natural |
| Go dependency | none new | `mattn/go-sqlite3` needs cgo; `modernc.org/sqlite` is pure Go but large |
| Backup | copy a coherent generation | the backup API or `VACUUM INTO`, plus the `-wal`/`-shm` files |
| Fit with git/dotfiles | exported files are the files | export needed for any review |

[consult; doc sqlite.org `whentouse.html`, `wal.html`, `backup.html`; doc mattn/go-sqlite3
README; pkg.go.dev modernc.org/sqlite]

*Inference:* at tens of rites, with one writer and no relational queries, files win. The
sanctioned store is already designed as files: strict JSON, one plan per rite, with
digests. SQLite becomes interesting only with history or multi-plan transactions, and
neither is needed yet. Whichever store, root never imports a user-supplied database
wholesale; it validates domain documents into its own format [consult].

## 5. Elevation: per-invocation helper or an exalted service

### 5.1 The two shapes

- **Decided:** one root helper per invocation. It re-executes `/usr/bin/servitor` through
  sudo, run0 or pkexec (setting `intercessor`) over a socketpair, and executes the
  sanctioned copy.
- **Alternative** (a narrow form of the owner's idea): a root **exalted service**.
  - socket-activated, exits when idle;
  - socket under `/run`;
  - executes only sanctioned plans;
  - asks polkit for each invocation with its own action set to `auth_admin`;
  - talks to the user's servitor with the same perform/revert/finish/abort protocol.

  Ordinary steps stay in the user's process.

### 5.2 A modern precedent on this machine [local]

systemd 262 ships root varlink services built exactly this way. `systemd-mountfsd.socket`
listens on `/run/systemd/io.systemd.MountFileSystem` with `SocketMode=0666`, is
socket-activated, and runs `NoNewPrivileges=yes`. Each call is authorised through polkit
(`io.systemd.mount-file-system.*`). The access is open to all; authorisation is
per-request. `systemd-nsresourced` (`io.systemd.namespace-resource.policy`) follows the
same pattern. systemd's `src/shared/bus-polkit.c` contains the varlink peer check plus
asynchronous polkit call [src systemd v262; consult].

Older mechanisms (hostnamed, PackageKit, udisks2, fwupd) use the system D-Bus with typed
methods and polkit (elevation research §2).

### 5.3 Correction: an authentication through run0 is kept, and siblings can use it

- run0 starts a transient unit through PID 1. That is gated by
  `org.freedesktop.systemd1.manage-units`, which is **`auth_admin_keep`** for active
  sessions [local `pkaction --verbose`; systemd v262 `org.freedesktop.systemd1.policy`].
  The kept authorisation lasts `ExpirationSeconds`, 300 s by default
  [local `/usr/share/polkit-1/polkitd.conf`; src polkit 127 interactiveauthority.c].
- systemd authorises D-Bus calls with a `system-bus-name` subject
  [src systemd v262 `src/shared/bus-polkit.c:98`]. polkit stores kept authorisations for
  bus names under the **process** behind the name
  [src polkit 127 `convert_temporary_authorization_subject()`].
- polkit 127 (installed here) added "auth_keep: skip re-authentication if new process
  shares same UID/parent/cgroup/tty" [doc polkit `NEWS.md`, "polkit 127"].
  `subject_equal_for_authz()` treats two processes as one subject when all of these hold
  [src polkit 127 `polkitbackendinteractiveauthority.c`, read 2026-10-11]:
  - both have safe pidfds and the same uid;
  - they have the same parent, which is still alive and is not PID 1;
  - all three share one cgroup;
  - all three share one controlling tty, and the tty is older than both processes.
- **What that means for the decided helper** (*inference* from the source; not yet
  observed, because observing it needs a real authentication):
  - **Across separate CLI invocations: no reuse.** Each run0 has a different servitor
    process as its parent, and that parent exits. Without a terminal (hotkeys) there is no
    ctty, so nothing is ever shared.
  - **Within one invocation on a terminal: siblings can use it.** servitor starts step
    commands with `Setpgid` only, so they keep the ctty and the cgroup
    [src `internal/invocation/speak.go:49`]. An *unexalted* step that servitor starts while
    the run0 helper's authorisation is kept would pass `manage-units` without a prompt if it
    is a direct child of the same servitor process. Examples:
    - a litany, which servitor executes directly;
    - an incantation whose `bash -c` execs its last command. Bash usually does this for a
      single simple command; not verified here.

    `manage-units` includes `run0 <anything>`, so that step gets root that was never
    sanctioned. This breaks G2 ("one authentication grants only the plan that was shown").
  - **The cogitator** is a long-lived parent on a terminal. It reuses authorisation across
    invocations unless it revokes. The decided `run0 -k` addresses that, but not the
    sibling case above.
  - sudo is the `auto` choice with a terminal, so the gap applies when run0 is chosen
    explicitly, or when sudo is missing.
- `sudo -k <cmd>` ignores and does not refresh sudo's cache, as recorded. It still
  honours `NOPASSWD` if the local sudoers grants it [man sudo 1.9.17p2; consult].
- `run0 -k` / `-K` (reset / remove temporary authorisations) exist in systemd 262
  [src systemd v262 `src/run/run.c`, options `-v`, `-k`, `-K`].

*Consequence*, with or without a daemon:
- run0 needs either steps started with `Setsid` (no ctty) while an exalted helper runs, or
  the helper started outside servitor's direct children.
- Alternatively, prefer an action servitor owns at plain `auth_admin` (no keep):
  - pkexec with a dedicated servitor action, or
  - the exalted service.

  Neither leaves a kept authorisation behind.

### 5.4 Comparison

| Concern | Per-invocation helper (decided) | Exalted service (alternative) |
|---|---|---|
| Root lifetime | one invocation | while active; socket-activated, idle exit |
| Who may connect | whoever got through sudo/run0/pkexec | anyone (`0666`) — authorisation per request |
| Authentication | delegated to three tools with different prompts, caching and TTY rules | one polkit action, `auth_admin`, no keep; servitor controls the policy |
| Prompt text | sudo `-p`, run0 unit name, pkexec cmdline | `polkit.message` with details: rite, aspect, digest |
| No-reuse guarantee | sudo `-k` yes; run0 kept for 5 min and shareable by sibling processes on a tty (§5.3); pkexec stock action yes | yes, by the shipped action (admins can still override with rules) |
| Terminal-only sessions | sudo on the tty | client forks `pkttyagent --notify-fd N --fallback`, as systemd's tools do [src systemd v262 `src/shared/polkit-agent.c`] |
| Hotkeys, no terminal | run0/pkexec via the graphical agent | the graphical agent |
| Identity | `SO_PEERCRED` on socketpair, `SUDO_UID` | `SO_PEERCRED` + `SO_PEERPIDFD`; polkit `unix-process` subject with pidfd |
| New code | helper mode, tool selection, socketpair through tools | listener, polkit client (godbus), unit files, action file, lifecycle/upgrade |
| Root-safe file operations, plan validation, trusted-path rule | required | equally required |
| Verification still open | socket through real sudo/run0 (one authentication) | polkit call with pidfd subject; pkttyagent handover in the cogitator |

Identity details for the service [consult; doc man-pages `unix.7`; src polkit 127
`polkitsubject.c`, `polkitunixprocess.c`]:
- `SO_PEERCRED` describes the peer at connect time.
- `SO_PEERPIDFD` (Linux 6.5) gives a stable process handle.
- polkit's `unix-process` subject accepts a pidfd as a D-Bus Unix-fd, plus the uid. The old
  pid+start-time identity has known races (CVE-2013-4288 class).
- Never accept a client-nominated pid. Fail closed when the lookup fails or the client
  disappears (CVE-2021-3560).
- Keep the kernel-supplied credentials as typed fields end to end. snapd's
  CVE-2019-7304 ("dirty_sock") parsed a uid out of attacker-controlled socket-address text.

*Inference:* an exalted service does not make the root side smaller. It still needs the
typed plan, the trusted-path rule, fd-relative writes, the sanctioned store and runtime
values only through the environment. What it removes is the **launcher matrix** (sudo,
run0, pkexec, TTY detection, socketpair through each tool) and the run0 caching problem.
What it adds is a **listening root endpoint** and an unconditional polkit/D-Bus
dependency. Both shapes can share one plan executor, so the choice can be deferred until
the helper's tool verification is done [consult].

### 5.5 What neither shape gives

- **One prompt for a mixed rite.** A rite that also runs `systemctl restart` unprivileged
  gets systemd's own `auth_admin_keep` prompt (or reuses one), separate from servitor's.
  "One authentication per invocation" needs a rule for already polkit-gated commands
  [consult; local `pkaction`].
- **A trusted path against A1.** A same-user process can register its own polkit agent,
  which polkit prefers (elevation research §7.7). A daemon does not change that.
- **Safety of runtime values by environment alone.** An approved `eval "$SERVITOR_INSCRIPTION_X"`
  still runs the value as code. The rule is necessary, not sufficient: sanctioned scripts
  must be reviewed for how they use these values. `BASH_ENV` must be absent, since
  `--noprofile --norc` does not stop it for non-interactive shells
  [man bash 5.3; consult].

## 6. Making rites easier to create

- The cogitator already has a **consecration wizard**: Rite → Liturgy → Seal, for
  "consecrating, amending and replicating rites of pattern Mark I"
  [src `internal/tui/wizard.go:13-36`; re-checked].
- Editors validate against `schema/rite.schema.json` through `$schema`, and the
  inquisition checks rites.
- Authoring and storage are independent. A store would need the same wizard, plus an
  export path for git.

*Inference:* if creation still feels bothersome, the levers are in the wizard and the CLI.
Examples:
- a non-interactive `consecrate`-style ritual from flags or a template;
- a `replicate <rite> <new>` ritual;
- starting a rite from an example;
- and, later, a sanction step offered at the end of the wizard for exalted steps.

## 7. Prior art

Gathered by a background researcher from primary sources (upstream source at the named
tags, man pages, vendor documentation; local files where marked). The full report, with a
line citation for each claim, was in the session scratch directory. Its key facts are kept
here.

| Program | Where definitions live, who may change them | Who executes, as whom | Lesson for servitor (*inference*) |
|---|---|---|---|
| **cron** (cronie 1.7.2 / @59a3d15; Debian trixie) | Spool `/var/spool/cron/<user>`: "not intended to be edited directly" [man crontab.1]. The gate is the small privileged `crontab` (setuid root on Arch [PKGBUILD]; setgid `crontab` on Debian). It installs by temp file in the spool → fsync → `check_syntax()` → fchown → rename → `poke_daemon()` [src `crontab.c` `replace_cmd()`]. Dotfile users keep `~/.crontab` and run `crontab ~/.crontab`. | crond forks per job: `pam_start("crond", user)`, PAM session, `initgroups`, env from PAM, `setreuid`, `chdir(HOME)`, then exec [src `security.c`]. It must *manufacture* a session (it forwards `XDG_SESSION_CLASS` for pam_systemd). | The closest precedent for the proposal. The store is root-held so the **root daemon** can trust what it reads, and jobs still run as the user. Debian had to fix `crontab -l \| crontab -`, which added another copy of the header on each round trip [man crontab(1) Debian]. Import/export fidelity is a real requirement. |
| **systemd** (262) | User units in `~/.config/systemd/user`, run by the user's own manager. No root copy, because nothing privileged reads them [man systemd.unit]. | PID 1 starts `user@UID.service` (`User=%i`, `PAMName=systemd-user`) to own all user-level work [local unit file]. "Processes started by the system service manager generally do not inherit environment variables set for the service manager itself", whereas user-manager processes do [man systemd.exec]. `graphical-session.target` exists only in the user manager [man systemd.special]. | Session work belongs to the user manager. A root service is the wrong home for it. |
| **systemd services** (hostnamed, mountfsd, nsresourced; polkit's agent helper) | Plain files in `/etc` (hostnamed). | Socket-activated, exit when idle, sandboxed (`ProtectSystem=strict`, `ProtectHome=yes`, `NoNewPrivileges=yes`), sockets `0666`, authorisation per call [local unit files]. The varlink peer is identified by SO_PEERCRED plus a **pidfd** and refused without one ("Failed to get peer pidfd, cannot securely authenticate"). The polkit subject is `unix-process{pidfd, uid}`, and interaction only happens if the request asks for it [src systemd v262 `bus-polkit.c:755-840`]. polkit 127 replaced its setuid helper with a per-connection (`Accept=yes`) socket-activated root helper. | "Root service" no longer means "always-on daemon". If servitor gets a privileged service, this is the shape to copy. |
| **snapd** (2.78) | All state in one `/var/lib/snapd/state.json`, rewritten atomically (0600) on every unlock; not SQLite [src `overlord/state/state.go`]. | A root daemon serves everything over `/run/snapd.socket` (0666). The uid comes from SO_PEERCRED; root is allowed; polkit (`io.snapcraft.snapd.manage`, `auth_admin_keep` active) is asked only if the client sends `X-Allow-Interaction` [src `daemon/access.go`]. **Yet it runs a separate per-user session agent** for notifications and user services [src `usersession/agent`]. | Even an "everything through the root daemon" design needs a user-side process for session work. CVE-2019-7304 (dirty_sock): the uid was string-concatenated with the client's socket address and re-parsed, and a socket named `;uid=0;` became root. |
| **Flatpak** (1.18.4) | `--user` installations never touch root. The system helper is used only for system-wide installs by non-root callers [src `flatpak-dir.c:3004-3019`]. | `flatpak-system-helper`: D-Bus activated, exits after 10 min idle. Typed methods (`Deploy`, `Uninstall`, …) with 15 polkit actions. The subject is `system-bus-name`. The unprivileged side downloads; the helper **re-pulls and re-verifies** (GPG), refusing what it cannot verify [src `flatpak-system-helper.c`, `flatpak-dir.c:7445-7493`]. | The closest structural analogue to the decided design. The user does everything it can, and the root side handles only the system-scope step, re-verifying its input. |
| **Nix** (2.35.2) | Root-owned store plus SQLite `/nix/var/nix/db/db.sqlite` (WAL, `SQLITE_FCNTL_PERSIST_WAL` so that read-only clients work) [src `local-store.cc:515-590`]. | The daemon performs store operations; builds run as throwaway `nixbld` users, never root and never the caller. `trusted-users` is "essentially equivalent to giving that user root access" [doc conf-file]. home-manager under NixOS runs activation with `User=<user>` even when root orchestrates [src home-manager `nixos/default.nix`]. | Root owns only immutable, content-addressed outputs; writes into `$HOME` run as the user. Its recurring CVE class (CVE-2024-27297; 2025 CVE-2025-46415/52991/52993) is a root daemon racing on paths others can influence. |
| **Docker / Podman** | — | Docker: "the docker group grants root-level privileges to the user" [doc post-install]. Podman uses fork/exec. Dan Walsh: with Docker's client/server model, `loginuid` is unset and audit shows `auid=unset`, "the identity was lost". Podman keeps it [opensource.com 2018]. | Access to an execution socket is root-equivalent. A daemon also loses audit attribution and session context. |
| **tailscaled** (1.104.1) | — | A root daemon; the CLI talks over a local socket. Read/write access for root, the daemon's uid or one `--operator` user; read-only for everyone else [src `ipn/ipnauth/ipnauth.go:166-208`]. | The "operator" pattern is a standing grant, which conflicts with fresh authentication. |
| **launchd / SMAppService** | LaunchDaemons "must be owned by root"; user LaunchAgents are owned by the user. "Allowing writability to a launchd configuration file allows one to specify which executable will be launched" [man launchctl]. SMAppService LaunchDaemons start only after admin approval, which can be revoked [doc Apple]. | Agents run in the user's `gui/<uid>` domain; "there are no distinct sessions in the privileged system context" [man launchd.plist]. | Root-held only for what root runs; session work as user agents. Approve once, then parameters over IPC: the shape of `sanction`. |
| **fwupd, udisks2, PackageKit** | fwupd `pending.db` and PackageKit `transactions.db` are SQLite (history and records) [local `file`; src `pk-transaction-db.c`]. | Narrow typed D-Bus verbs that take **fds, not paths** (fwupd `Install(h …)`, udisks `LoopSetup(h …)`). The polkit action is chosen by object trust (`*-trusted` vs untrusted) [local interface XML and policy files]. None accepts "run this script as root". | PackageKit CVE-2020-16121 (the root daemon as a file-presence oracle) and CVE-2020-16122 (trusting a user-supplied package). |

**Storage across these programs:**
- SQLite holds **history and records**: Nix's metadata, fwupd, PackageKit, dnf's
  `history.sqlite`.
- Definitions people read or edit stay **plain files**: crontabs, units, plists,
  `/etc/hostname`. snapd uses one JSON file.
- Measured by the researcher (Go 1.27.2, stripped binaries): `modernc.org/sqlite` adds about
  **4.7 MB** without cgo; `mattn/go-sqlite3` adds about 1.9 MB but needs cgo and gcc.
- A root-owned WAL database cannot be opened read-only by unprivileged readers unless the
  `-wal`/`-shm` files persist and are readable [doc sqlite.org `wal.html`]. Nix sets
  `PERSIST_WAL` for this. The directory must not be user-writable.

**polkit guidance** ["Writing polkit applications", local polkit 127 docs]:
- "DON'T use polkit if your program isn't intended to be used by unprivileged programs …
  it's fine to just require the user to be root. Users can always run your tool through
  e.g. sudo(8), pkexec(1) or write a simple polkit-using mechanism that allows access to a
  (safe) subset of your tool."
- "DON'T use polkit unless you actually have to … a secure program is often one with
  little amount of code and few dependencies."
- DO pass a specific `polkit.message`; its `$(var)` placeholders expand from the details,
  and setting `polkit.*` details requires the checker to be privileged.
- `auth_admin_keep` caches by action and subject "even if the variables passed along with
  the check are different" [man polkit(8)].

**The security record of root daemons acting for users:**
- Identity plumbing bugs: CVE-2019-7304, CVE-2021-3560, CVE-2013-4288 and CVE-2019-6133
  (pid/start-time races).
- Socket access as root equivalence: Docker, Nix `trusted-users`.
- Root writing in trees others influence: CVE-2018-6954 and CVE-2017-18078 (tmpfiles), the
  2025 Nix/Lix/Guix races.
- A single root process as a single point of failure: CVE-2021-33910 (PID 1 crash on a long
  mount path).
- The setuid helpers failed too: PwnKit, Baron Samedit, snap-confine. That is why systemd
  and polkit moved to helpers forked by the service manager or socket-activated, not to
  always-on daemons.

## 8. Answers to the open daemon tickets (first pass)

These are proposals for the owner, not decisions [inference, partly after the
consultation].

| Ticket | Proposed answer |
|---|---|
| [Who performs invocations?](lit:issue:2f0a796f-8ffe-4c57-b26b-aed1ffdf0137) | The user's servitor process, as today. Root only executes sanctioned exalted plans, through the helper or an exalted service (§3, §5). |
| [Truth model](lit:issue:4aa13d93-b12c-476e-b27a-b765a4d5349c) | Cache **omens** with source, time and rite revision; mark them stale, invalid or pending. A recorded invocation never becomes an omen. The data-slate fallback is unchanged. Restart means a full reconciliation. Fits ADR 0002 as written. |
| [Scheduling the auspex](lit:issue:4ca10890-c47c-476f-994d-c9fae72f99fd) | On demand, with duplicate suppression; after an invocation of the rite; optional explicit polling per rite. Never start running existing auspexes periodically without the rite asking for it. |
| [Watching file-based omens](lit:issue:7e48cf1c-5e51-4c46-850a-4e3836afc80a) | inotify on parent directories and names, rebuilt after renames. Overflow and network filesystems mean re-reading. A watch is an invalidation signal, never proof [doc man-pages `inotify.7`; consult]. Cost not measured yet. |
| [Socket protocol and CLI fallback](lit:issue:17e5e02b-8714-4ff1-93a4-5a3909bfd366) | A socket under `$XDG_RUNTIME_DIR` serving the `census --vigil` / augury shapes. Read-only requests fall back to direct computation. Invocations are never retried through a fallback. |
| [Lifecycle and events](lit:issue:2ca7a2ea-2047-4e9e-8404-e1a751eb5e99) | An optional `systemd --user` service, socket-activated. Events "rite changed", "desecrated" and "auspex settled" go to subscribers, with bounded queues. |
| [Decide](lit:issue:4c95eb6c-4f30-4c9b-8789-abf73fdbb8f1) | Fix the start-up cost first. Then decide on an observation daemon from measured demand (live status bars, pickers), not for speed of invocation. |

## 9. Existing decisions this touches

1. **Elevation backends:** under polkit 127, run0's kept authorisation can be used by other
   children of the same servitor process on a terminal. That includes unexalted steps
   (§5.3). run0 needs `Setsid` for steps or a detached helper, or the `intercessor` list
   changes.
2. **Dominion fails open** (§3.4). It must fail closed for exalted work, in a
   root-controlled namespace. For ordinary writes it is the owner's call.
3. **The session digest** H(approved digest ‖ aspect ‖ values) *commits to* the runtime
   values; it does not *record* them. Define an unambiguous encoding, and decide which
   values may be logged [consult].
4. **The daemon project's "optional" constraint** rules out the thin-client design unless
   the owner lifts it.

## 10. Questions for the owner

1. **Workflow:** are rites editable files that take effect directly (today), or drafts
   installed explicitly as revisions?
2. **Scope of protected authority:** only exalted content (the sanction design), or also
   ordinary rites, against unnoticed changes?
3. **Must the daemon stay optional?**
4. **Elevation transport:** the per-invocation helper (with a corrected backend list) or an
   exalted service with its own polkit action?
5. **Is fresh authentication a policy servitor enforces,** for example by refusing sudo
   `NOPASSWD` or polkit rules that keep authorisation, or only its default?
6. **Is there demand for live state** (status bar, picker) strong enough for a user
   observation service, once start-up is fixed?
7. **The go-runewidth bump** (v0.0.27 → v0.0.30) is a one-line change that removes 20 ms
   from every ritual. It needs a ticket in feat/servitor.

## Sources

**Local** (2026-10-11):
- Benchmarks and `GODEBUG=inittrace=1` on a synthetic Librarium, in the session scratch
  directory.
- `go mod why`.
- `systemctl --user show-environment` (names only).
- `pkaction --verbose` for `org.freedesktop.systemd1.manage-units` and
  `io.systemd.mount-file-system.mount-image`.
- `/usr/share/polkit-1/polkitd.conf`.
- `/usr/lib/systemd/system/systemd-mountfsd.{socket,service}`.
- `pgrep` for the polkit agent.

**servitor at `56a3431`:** `internal/invocation/speak.go`, `internal/rituals/dominion.go`,
`internal/rituals/gather.go`, `internal/tui/wizard.go`, `internal/librarium/{orders,settings,model,write}.go`,
`internal/dominion/`, `README.md` (the Librarium), ADR 0001–0006.

**Upstream:**
- go-runewidth v0.0.27 and v0.0.30 `runewidth.go` (module cache).
- polkit `NEWS.md` (main, section 127); polkit tag 127
  `src/polkitbackend/polkitbackendinteractiveauthority.c` (`subject_equal_for_authz`,
  `convert_temporary_authorization_subject`).
- systemd v262 `src/shared/bus-polkit.c`.

**Background researcher** (§7): cronie @59a3d15 and Arch PKGBUILD 1.7.2, Debian trixie
crontab(1), systemd 262 man pages and unit files, snapd 2.78, Flatpak 1.18.4, Nix 2.35.2 and
@651b0cd, home-manager master, tailscale 1.104.1, Apple launchctl/launchd.plist/SMAppService
docs, fwupd 2.1.8, udisks2 2.11.2, PackageKit @40c752e, libdnf 4, sqlite.org (`whentouse`,
`appfileformat`, `wal`) and `os_unix.c`, mattn/go-sqlite3 README, modernc.org/sqlite v1.60.1,
Docker security docs, opensource.com (Walsh 2018), MITRE CVE entries named in §7.

**Consultation** (session above): further citations for cronie 1.7.2, Flatpak 1.16.1,
systemd v262 (`run0.xml`, `run.c`, `bus-polkit.c`, `systemd.exec`, `systemd.socket`),
polkit 127 (`polkitsubject.c`, `polkitunixprocess.c`, Authority interface), man-pages
(`unix.7`, `inotify.7`, `fsync.2`, `flock.2`, `openat2.2`), sqlite.org, Docker security
docs, Podman manual, Nix manual, Tailscale operator docs, Apple launchd/SMAppService, and
advisories CVE-2019-7304, CVE-2021-3560, CVE-2021-21261, CVE-2021-4034. Claims marked
[consult] were not all re-checked.
