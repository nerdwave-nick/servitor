# Verification: the elevation transport, in a VM

Date: 2026-10-12. Tracker: [Prototype: verify the elevation transport in a VM](lit:issue:3e6a4dca-85e6-40d6-a639-95df46bc26cf)
in lit project `feat/servitor`, a child of the plan "rite format v2", section "Parked:
elevated steps", step 1 of "Not done yet".

## Question

Does the decided elevation transport work on the real tools? The decided transport is one
root helper per invocation: `/usr/bin/servitor` re-executed through sudo, run0 or pkexec,
with a socketpair on fd 0 and 1.

1. Does a socketpair on fd 0/1 reach the helper unchanged through sudo (`use_pty`, and with
   I/O logging), pkexec and run0? Does the helper still learn who called it?
2. Can other processes reuse the authentication while the helper runs or afterwards? The
   tests started each sibling the way servitor starts its steps.
3. Do `sudo -k` and a servitor-specific polkit action at `auth_admin` leave anything
   reusable behind?

## Setup

- **VM:** Arch Linux cloud image, upgraded on 2026-10-12 to the host's versions:
  - sudo 1.9.17p2
  - polkit (`pkaction`) 127
  - systemd 262 (262-1-arch)
  - kernel 7.2.7 (the host runs 7.2.3)

  QEMU/KVM, cloud-init.
- **Session:** a test user `tester` (uid 1500) in `wheel`, auto-logged in on tty1:
  `loginctl` reported `Type=tty Class=user Active=yes`. So polkit applied the
  `allow_active` defaults, as on the desktop. `manage-units` reported `implicit active:
  auth_admin_keep`.
- **Driver:** runs every scenario in its own pseudo-terminal, which becomes the controlling
  tty, and types the password when a prompt appears. Agents: sudo's own prompt, pkexec's
  internal text agent, and the `pkttyagent` that run0 starts. No graphical agent.
- **Prototype:** a disposable static Go binary `proto`, kept in the session scratch
  directory only and never in the repository.
  - `proto parent <tool…>` creates `socketpair(AF_UNIX, SOCK_STREAM|SOCK_CLOEXEC)` and
    starts `<tool…> /usr/local/bin/proto helper` with the child's end as `*os.File` on
    stdin and stdout. It sets `PR_SET_DUMPABLE=0` on itself.
  - `proto helper` reports what it sees, then answers `ping` with `pong` until `finish`.
  - While the helper is alive, the parent runs **siblings** the way servitor runs steps:
    stdin `/dev/null`, `Setpgid` (as `internal/invocation/speak.go:49`), and with
    `Setsid` in one variant.
- **Helper copies:** `/usr/local/bin/{proto,proto-logged,proto-pk}`, root-owned 0755.
- **Configuration:**
  - sudoers: `%wheel ALL=(ALL:ALL) ALL`, plus `Defaults!/usr/local/bin/proto-logged
    log_input,log_output`.
  - A polkit action `org.servitor.proto.perform` with `exec.path=/usr/local/bin/proto-pk`
    and `auth_admin` for any, inactive and active.
- **A first run's run0 results were void.** `polkit-agent-helper.socket` was not started
  after polkit was installed in that boot, so `pkttyagent` failed with "needs to be setuid
  root". On the host the socket is active and the helper is not setuid [local]. The run0
  scenarios were repeated with the socket active; only that run counts.

## Results

### The channel

| Tool | fd 0 / fd 1 in the helper | `SO_PEERCRED` on fd 0 | Identity in env | euid | cwd | controlling tty |
|---|---|---|---|---|---|---|
| `sudo -k -p … --` | **same socket inode** | pid of the parent, uid 1500 | `SUDO_UID=1500`, `SUDO_USER` | 0 | caller's (`/home/tester`) | yes (sudo's pty, `use_pty`) |
| sudo with `log_input,log_output` | **two pipes** (sudo interposes) | fails: "socket operation on non-socket" | `SUDO_UID=1500` | 0 | caller's | yes |
| `pkexec` (stock action) | same socket inode | pid = the helper (pkexec execs in place), uid 1500 | `PKEXEC_UID`, `SUDO_UID` | 0 | `/root` | yes (inherited) |
| `pkexec` (own action, `auth_admin`) | same socket inode | as above | as above | 0 | `/root` | yes |
| `run0` | same socket inode | pid of the parent, uid 1500 | `SUDO_UID=1500`, `SUDO_USER` | 0 | caller's | **none** (`tty_nr 0`) |

The line exchange (`ping`/`pong`, `finish`) worked through every tool, and each tool
needed exactly one typed password.

Under run0 the helper's parent is PID 1. It runs in a new root session scope
(`user-0.slice/session-N.scope`); the prompt names the transient unit, e.g.
"Authentication is required to start transient unit 'run-p874-i4538.service'".

### Reuse by other processes

| Scenario | What tried without a prompt | Result |
|---|---|---|
| `sudo -k` helper alive | sibling `sudo -n true`; `bash -c 'sudo -n true'` | **refused**: "a password is required" |
| sudo **without** `-k`, helper alive | sibling `sudo -n true` | **succeeded** (the tty timestamp) |
| pkexec stock action, helper alive | sibling `pkexec --disable-internal-agent proto noop` | **refused**: "No authentication agent found" (nothing kept) |
| pkexec own action (`auth_admin`), helper alive | the same against `proto-pk` | **refused**, same message |
| run0 helper alive, siblings with `Setpgid` | `run0 --no-ask-password true`; the same through `bash -c '…'` and `bash -c '…; exit $?'`; `systemctl --no-ask-password start proto-dummy.service` | **all refused**: "requires interactive authentication" |
| run0 helper alive, siblings with `Setsid` | the same four | all refused |
| right after the run0 parent exited, a new process | `run0 --no-ask-password true` | refused |
| one shell runs `run0 true` (password typed) then `run0 --no-ask-password true` | the second run0 | **refused** |

## Conclusions

1. **The socketpair channel works through sudo, pkexec and run0**, with one prompt each.
   This settles the open fact "socket stdio through real sudo (`use_pty`) and run0 to the
   system manager".
2. **Identity:**
   - Under sudo and run0, `SO_PEERCRED` names the servitor parent.
   - Under pkexec it names the helper's own pid, since pkexec execs in place, and the uid
     is still the caller's.
   - With sudo I/O logging, sudo interposes pipes and `SO_PEERCRED` fails. `SUDO_UID` is
     still set, so the decided fallback holds. On a pipe the protocol must stay a byte
     stream with newline framing; it worked.
   - Whether this machine's sudoers enables I/O logging is still unknown and needs a root
     read on the host.
3. **`sudo -k` is required, and it works.** Without it a sibling reuses the tty timestamp
   at once; with it, nothing is reusable.
4. **pkexec at `auth_admin`, stock or a servitor action, leaves nothing reusable.**
5. **run0 left nothing reusable in any scenario tested.** This includes the plain "same
   shell, second run0" case.
   - This **contradicts the inference** in `docs/research/servitor-daemon.md` §5.3. That
     inference was drawn from polkit 127's `subject_equal_for_authz()` and its NEWS line
     "auth_keep: skip re-authentication if new process shares same UID/parent/cgroup/tty".
   - Why the rule did not apply to run0's D-Bus subject was not established. polkit builds
     the process from the bus name with a pidfd when it can
     [src polkit 127 `polkitsystembusname.c`], so the cause lies elsewhere.
   - The observation stands for these versions, a tty session and the text agent.
6. **The working directory differs by tool.** sudo and run0 keep the caller's; pkexec uses
   `/root`. The decided design's `chdir("/")` in the helper is therefore needed, as
   recorded.
7. **The controlling tty differs by tool.** sudo and pkexec give the helper one; run0 gives
   none. The decided "terminal signals ignored, SIGTERM = abort" covers both.

## What this changes

- The decided `intercessor` order needs **no change** for reuse:
  - with a terminal: `sudo -k`;
  - without: run0, then pkexec.

  The "run0 gap" raised on 2026-10-11 was not observed. Keeping `-k` for sudo remains
  essential.
- Not covered:
  - the graphical polkit agent (pantheon on the host);
  - the host's sudoers settings;
  - a long-lived parent (the cogitator) calling run0 twice. The same-shell case suggests
    no reuse, but the cogitator was not tested.

## Sources

- **Local VM, 2026-10-12:** the driver output, kept in the scratch directory only. Versions
  as reported by `sudo -V`, `pkaction --version`, `systemctl --version` and `uname -r`
  inside the VM; `loginctl show-session self`; `pkaction --verbose` for
  `org.freedesktop.systemd1.manage-units`.
- **Host:** `/usr/lib/polkit-1/polkit-agent-helper-1` is 0755 root:root (not setuid), and
  `polkit-agent-helper.socket` is active.
- **Source:** systemd v262 `src/run/run.c` (run0's `-k`/`-K`/`-v` and the
  `StartTransientUnit` path); polkit 127 `src/polkit/polkitsystembusname.c`
  (`polkit_system_bus_name_get_process_sync`).
