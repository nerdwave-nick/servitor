# Research: sudo, pkexec, run0 and doas for elevated step re-execution

Date: 2026-10-09. Machine: Arch Linux, kernel 7.2.3, niri session started by sddm, Go 1.27.1.

## Question

When servitor re-executes itself as `<tool> /abs/servitor __step <rite> <idx> <aspect> [--meta k=v ...] [--config DIR]`, how do sudo, pkexec, run0 and doas behave? This covers environment handling, TTY requirements, credential caching, behaviour with no polkit agent, availability on Arch, exit codes, and how the elevated process finds the binary.

Source tags: **[man]** = local man page for the installed version. **[src]** = upstream source, read on the date above. **[local]** = observed on this machine. **[2nd]** = secondary source. Statements labelled *Inference* are not verified facts.

## Installed versions [local]

| Tool | Package (repo) | Version | Notes |
|---|---|---|---|
| sudo | `sudo` (core) | 1.9.17.p2-6 | `/usr/bin/sudo` setuid. `/etc/sudoers` is 0440, so its local contents were not read. |
| pkexec | `polkit` (extra) | 127-3 | `/usr/bin/pkexec` setuid. Installed as a dependency. |
| run0 | `systemd` (core) | 262-1 | `/usr/bin/run0 -> systemd-run`. PAM stack in `/usr/lib/pam.d/systemd-run0`. |
| doas | `opendoas` (extra, 6.8.2-3) | **not installed** | No `/etc/doas.conf`. |
| polkit agent | `pantheon-polkit-agent` 8.1.0 | running | `io.elementary.desktop.agent-polkit`. polkitd is running. |

Session [local]: niri runs as the systemd user service `niri.service` (`niri-session -l`), and the logind display session is 187. The user is in `wheel`. polkit's `/usr/share/polkit-1/rules.d/50-default.rules` makes `unix-group:wheel` the admin identity. The user manager environment contains `WAYLAND_DISPLAY=wayland-1`, `DISPLAY=:3` and `DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus`.

## Summary table

| | sudo | pkexec | run0 | doas (OpenDoas) |
|---|---|---|---|---|
| Env default | `env_reset`. HOME, MAIL, SHELL, LOGNAME, USER are set for the target user. The keep list is COLORS, DISPLAY, HOSTNAME, KRB5CCNAME, LS_COLORS, PATH, PS1, PS2, XAUTHORITY, XAUTHORIZATION, XDG_CURRENT_DESKTOP. PATH is then replaced by `secure_path`. SUDO_USER, SUDO_UID, SUDO_GID, SUDO_HOME and SUDO_COMMAND are set. | `clearenv()`. Keeps SHELL, LANG*, LC_*, TERM, COLORTERM, all of which are validated. HOME, USER and LOGNAME are set for the target. PATH is fixed. PKEXEC_UID, SUDO_UID and SUDO_GID are set. DISPLAY and XAUTHORITY are dropped. | Nothing comes from the caller. The child gets the system manager env plus TERM, COLORTERM and NO_COLOR (only on a TTY), SUDO_USER, SUDO_UID, SUDO_GID. The unit's `User=` gives HOME, USER, LOGNAME, SHELL. | New env: HOME, LOGNAME, PATH, SHELL, USER for the target, DOAS_USER, plus DISPLAY and TERM. |
| HOME in child | `/root` | `/root` | `/root` | `/root` |
| CWD in child | unchanged | **`/root`** unless `--keep-cwd` | caller's CWD (when target is root) | unchanged |
| Prompt channel | `/dev/tty`, or an askpass helper with `-A` | session polkit agent. Falls back to its own text agent on the controlling TTY. | session polkit agent. Falls back to `pkttyagent --fallback` when there is a controlling TTY. | `/dev/tty` only (`RPP_REQUIRE_TTY`) |
| Works with no TTY | only with `-A` and an askpass helper | yes, if a graphical agent is running | yes, if a graphical agent is running | no |
| Caching default | 5 min sliding window, per tty (per ppid if no tty) | **none**: `policykit.exec` is `auth_admin` | `manage-units` is `auth_admin_keep`, 5 min fixed. Reuse only on the same tty and parent (polkit 127). | `persist` is opt-in. 5 min sliding, keyed on ppid, sid, tty |
| Prompts for N steps | 1 (on a TTY) | **N** | 1 on a TTY, N with no TTY (inference) | 1 with `persist`, N without |
| Non-interactive probe | `sudo -n -v` / `sudo -n true` → 1 + "a password is required" | none built in (`pkcheck` exists) | `run0 -n -v` (v262+) | `doas -n` (exit 1, "Authentication required") |
| Exit: cancelled | 1, or killed by the signal (Ctrl-C) | **126** (dialog dismissed) | 1 ("Access denied") | 1 |
| Exit: auth failed / denied | 1 | 127 | 1 | 1 |
| Exit: child failed | child's code | child's code | child's code. A signal gives 255. | child's code. A signal gives 128+n. |
| Arch availability | core, but not in `base` (pulled in by `base-devel`) | `polkit` (extra), optional dependency of systemd | systemd ≥ 256 (core) | `opendoas` (extra) |

## 1. Environment handling and argument passing

**sudo.** `env_reset` is on by default. It runs the command with a minimal environment: TERM, PATH, HOME, MAIL, SHELL, LOGNAME, USER and SUDO_*. HOME, MAIL, SHELL, LOGNAME and USER are "initialized based on the target user" [man sudoers 1.9.17p2, "Command environment", `env_reset`]. The compiled-in `env_keep` list is COLORS, DISPLAY, HOSTNAME, KRB5CCNAME, LS_COLORS, PATH, PS1, PS2, XAUTHORITY, XAUTHORIZATION and XDG_CURRENT_DESKTOP [src sudo `plugins/sudoers/env.c` `initial_keepenv_table`]. Since 1.9.16 the default sudoers file enables `secure_path` [src sudo NEWS 1.9.16]. Arch builds sudo with `--with-secure-path-value=/usr/local/sbin:/usr/local/bin:/usr/bin` [src Arch `sudo` PKGBUILD 1.9.17.p2-6]. The local `/etc/sudoers` could not be read to confirm that the line is still active. Keeping HOME via env_keep is "strongly discouraged" [man sudoers `env_keep`]. sudo also sets SUDO_HOME, SUDO_USER, SUDO_UID and SUDO_GID [man sudo 1.9.17p2 ENVIRONMENT]. The command runs in the current CWD unless `-D` is used and `runcwd` allows it [man sudo `-D`]. Arguments are passed as an argv vector, with no shell unless `-s` or `-i` is used. Leading `VAR=value` operands are treated as environment assignments, and `--` ends sudo's options [man sudo]. `use_pty` is on by default since 1.9.14, so on a terminal the command gets a new pty [man sudoers `use_pty`].

**pkexec.** pkexec calls `clearenv()`. It then restores only SHELL, LANG, LINGUAS, LANGUAGE, LC_COLLATE, LC_CTYPE, LC_MESSAGES, LC_MONETARY, LC_NUMERIC, LC_TIME, LC_ALL, TERM and COLORTERM. DISPLAY and XAUTHORITY are kept only for actions with `allow_gui` [src polkit `src/programs/pkexec.c`; man pkexec 127]. Each kept value is validated:
- SHELL must be listed in `/etc/shells`.
- Any other value containing `/`, `%` or `..` (XAUTHORITY excepted for `/`) is rejected.

Either failure aborts with "This incident has been reported" and exit 127 [src pkexec.c `validate_environment_variable`]. pkexec then sets:
- `PATH=/usr/sbin:/usr/bin:/sbin:/bin:/root/bin`
- LOGNAME, USER and HOME for the target user
- `PKEXEC_UID`, plus `SUDO_UID` and `SUDO_GID` (new in polkit 127)

[src pkexec.c; polkit NEWS 127]. pkexec changes to the target's home directory (`/root`) unless `--keep-cwd` is given (polkit ≥ 121) [man pkexec]. Arguments go through `execv(path, argv)` unchanged. pkexec only parses `--user/-u`, `--keep-cwd`, `--disable-internal-agent`, `--help` and `--version` before the program, and **does not understand `--`**: it would treat `--` as the program name [src pkexec.c option loop]. The polkit dialog shows the full command line, shortened to 80 characters as `$(cmdline_short)` [src pkexec.c].

**run0.** The command runs as a transient unit started by PID 1, and "no execution or security context credentials are inherited" [man run0 v262]. The session inherits the *system manager* environment. run0 adds TERM (and COLORTERM and NO_COLOR, but only when a stdio fd is a TTY), SUDO_USER, SUDO_UID, SUDO_GID and SHELL_PROMPT_PREFIX. Callers can add more with `--setenv=NAME[=VALUE]` [man run0; src systemd `src/run/run.c`]. Because the unit has `User=`, the service manager sets USER, LOGNAME, HOME and SHELL [man systemd.exec 262 "$USER, $LOGNAME, $HOME, $SHELL"]. PATH is the manager's fixed value [man systemd.exec `$PATH`]. On this machine `systemctl show-environment` reports `/usr/local/sbin:/usr/local/bin:/usr/bin` [local]. The default working directory is the caller's CWD when switching to root [man run0 `--chdir`]. run0 turns off `${VAR}` expansion of command arguments (`arg_expand_environment = false` in sudo mode), so argv reaches the command verbatim [src run.c, systemd main]. Session class and PAM come from the `systemd-run0` PAM stack [man run0].

**doas (OpenDoas).** doas builds a new environment: HOME, LOGNAME, PATH, SHELL and USER for the target, DOAS_USER, and DISPLAY and TERM copied from the caller. `keepenv` and `setenv {}` in doas.conf change this. The working directory is not changed [OpenDoas `doas.1`, `env.c` `createenv`]. PATH handling depends on the rule. If the rule names a `cmd`, the command is looked up with a fixed "safepath". Otherwise doas restores the caller's PATH before `execvpe` [src OpenDoas `doas.c`].

**All four tools.** None of them passes `XDG_CONFIG_HOME`, `XDG_STATE_HOME`, `XDG_RUNTIME_DIR`, `WAYLAND_DISPLAY` or `DBUS_SESSION_BUS_ADDRESS` by default (facts above). HOME is root's home (`/root`) under all four. In Go, `os.UserConfigDir()` reads `$XDG_CONFIG_HOME` or falls back to `$HOME/.config`, so in the elevated child it would resolve to `/root/.config` (stdlib behaviour).

## 2. TTY requirements and detecting whether a terminal prompt is possible

- sudo reads the password from `_PATH_TTY` (`/dev/tty`). If that fails, it errors with "a terminal is required to read the password; either use the -S option to read from standard input or configure an askpass helper" [src sudo `src/tgetpass.c`; the string is present in the installed binary, local]. With `-A`, sudo instead runs `SUDO_ASKPASS` or the `Path askpass` helper from sudo.conf, which may be graphical [man sudo `-A`].
- pkexec first asks the agent registered for the calling process or session. It registers its own text agent only if there is none, and that agent "will fail if we can't find a controlling terminal" ("Error opening current controlling terminal for the process (`/dev/tty')") [src pkexec.c, `polkitagenttextlistener.c`].
- run0 (systemd) forks `pkttyagent --notify-fd N --fallback` only when the process has a controlling terminal (`shall_fork_agent()` → `get_ctty_devnr(0)`) and run0 itself is not root [src systemd `src/shared/polkit-agent.c`, `exec-util.c`]. `--fallback` means the tty agent is used only when no session agent exists. polkit checks the process-scoped agent first, but skips it when it is marked fallback, then uses the session agent [src polkit `polkitbackendinteractiveauthority.c` `get_authentication_agent_for_subject`]. **Consequence (fact from source):** whenever a graphical agent is running in the session, pkexec and run0 prompt with the graphical dialog even from a terminal.
- run0 stdio mode: it allocates a pty only if stdin, stdout and stderr are *all* TTYs. Otherwise it passes the fds through directly (`--pipe`) [man run0 `--pty/--pipe`; src run.c].
- doas uses `readpassphrase(..., RPP_REQUIRE_TTY)`. With no tty, PAM conversation fails and doas prints "Authentication failed" [src OpenDoas `pam.c`].
- polkit's session lookup for a process that is outside any logind session (for example spawned by `niri.service` or fuzzel) falls back to the user's *display* session (`sd_uid_get_display`). So the GUI agent registered for that session is found [src polkit `polkitbackendsessionmonitor-systemd.c`].
- Detection in this sandbox (no terminal): `[ -t 0 ]` was false, and opening `/dev/tty` failed with ENXIO ("No such device or address") [local]. `isatty(stdin)` is not sufficient on its own. sudo, pkexec's text agent and doas need a *controlling terminal* (`/dev/tty`). run0's fallback agent needs a ctty, and its pty mode needs all three stdio fds to be TTYs.
- bubbletea: `tea.ExecProcess` releases the terminal for a child `*exec.Cmd` and restores it afterwards [2nd: pkg.go.dev/github.com/charmbracelet/bubbletea]. A Ctrl-C at a sudo prompt goes to the whole foreground process group. sudo re-raises the signal on itself after restoring the tty (`kill(getpid(), sig)`) [src tgetpass.c].

## 3. Authentication caching

- **sudo:** `timestamp_timeout` defaults to 5 minutes [man sudoers]. `timestamp_type` defaults to `tty`, meaning one record per terminal, and "if no terminal is present, the behavior is the same as ppid" [man sudoers]. Every successful run, including one that used a cached record (TS_CURRENT), calls `timestamp_update`, so the window slides [src sudo `plugins/sudoers/check.c`]. `sudo -v` refreshes the record and `sudo -k` invalidates it [man sudo]. **N steps → 1 prompt** on the same tty within the window.
- **pkexec:** the default action `org.freedesktop.policykit.exec` is `auth_admin` for any, inactive and active sessions [local: `pkaction --verbose`]. The `*_keep` variants are not used, so **N steps → N prompts**. Choosing a different action requires installing a `.policy` file with the `org.freedesktop.policykit.exec.path` annotation [man pkexec "ACTION AND AUTHORIZATIONS"].
- **run0:** it calls `StartTransientUnit`, which is checked against `org.freedesktop.systemd1.manage-units` [src systemd `src/core/dbus*.c`]. That action is `auth_admin_keep` for active sessions and `auth_admin` for inactive or any [local: `pkaction`]. `*_keep` lasts "a brief period (e.g. five minutes)" [man polkit 127]. In polkit the expiry is fixed from the grant time, with no refresh on use. Since polkit 127 it can be configured through `ExpirationSeconds` in `polkitd.conf` (default 300) [src polkitbackendinteractiveauthority.c; local `/usr/share/polkit-1/polkitd.conf`; polkit NEWS 127]. The temporary authorization is stored for the run0 *process*. Since polkit 127, a *new* process reuses it only if it has the same UID, the same parent PID (not 1), the same cgroup and the same controlling tty, the tty number is non-zero, and the tty is older than both processes [src `subject_equal_for_authz`; NEWS 127 "auth_keep: skip re-authentication if new process shares same UID/parent/cgroup/tty"]. From systemd v262, run0 also has `-v/--validate`, `-k`, `-K` and `-n` [man run0 v262; systemd NEWS 262].
- **doas:** `persist` in doas.conf is opt-in. When set, it holds for 5 minutes and is refreshed after each successful run (`timestamp_set(fd, 5*60)`). The record is keyed on `ppid-sid-ttynr-parent_starttime-uid` under `/run/doas` [src OpenDoas `pam.c`, `timestamp.c`]. Arch's OpenDoas build enables it (`--with-timestamp`) [src Arch opendoas PKGBUILD]. `doas -L` clears it.

## 4. When no polkit agent is running

- pkexec with no agent and a ctty: its built-in text agent prompts on the terminal. With no agent and no ctty: "Error creating textual authentication agent: …", exit 127. With `--disable-internal-agent` and no agent: "Error executing command as another user: No authentication agent found.", exit 127 [src pkexec.c; man pkexec].
- run0 with no agent and a ctty: `pkttyagent` prompts on the tty. With no agent and no ctty, polkit returns a challenge and systemd reports SD_BUS_ERROR_INTERACTIVE_AUTHORIZATION_REQUIRED. run0 prints "Failed to start transient service unit: …" and exits 1 [src systemd `bus-polkit.c`, `run.c`]. *Inference:* this path was read in the code but not reproduced.
- niri ships no agent. Its docs recommend starting one, such as `plasma-polkit-agent`, through systemd or `spawn-at-startup` [2nd: yalter.github.io/niri/Important-Software.html]. The Arch Wiki lists hyprpolkitagent, lxqt-policykit, mate-polkit, polkit-gnome, polkit-kde-agent, pantheon-polkit-agent (used here) and soteria [2nd: wiki.archlinux.org/title/Polkit].

## 5. Availability on Arch

- sudo is in core, but is required only by `base-devel`, not by `base` [local `pacman -Qi`].
- polkit is in extra and is an optional dependency of systemd ("allow administration as unprivileged user") [local].
- run0 first appeared in systemd 256 [systemd NEWS 256]. Some flags need later versions: `--pty`/`--pipe` 257, `--pty-late` and `--via-shell` 258, `--empower` 259, `-v/-k/-K/-n` 262 [man run0]. run0 does nothing useful for an unprivileged caller unless polkit is installed.
- OpenDoas 6.8.2 is in extra. The package does not create `/etc/doas.conf`, and without one doas fails with "doas is not enabled, /etc/doas.conf: …" (exit 1) [src doas.c; local file is absent].
- Commands run: `pacman -Qi sudo polkit opendoas`, `systemctl --version` (262), `which` (sudo, pkexec, run0 found; doas not found), `sudo -V` (1.9.17p2). No command was escalated.

## 6. Exit codes

- **sudo:** returns the command's status. If the command dies from a signal, sudo kills itself with the same signal. "Authentication failure, configuration/permission problem, or if the given command cannot be executed" all give exit 1 [man sudo EXIT VALUE]. Messages seen in the binary: "N incorrect password attempts", "no password was provided", "a password is required" (`-n`), "a terminal is required…", "is not in the sudoers file", "Sorry, user … is not allowed to execute" [local strings]. An auth failure **cannot be told apart from a command exit 1** by exit code alone.
- **pkexec:** 126 means the dialog was dismissed. 127 covers "not authorized", authentication failure, no agent, rejected env and other errors. Otherwise pkexec returns the program's status [man pkexec RETURN VALUE; src]. A child that exits 126 or 127 is ambiguous.
- **run0:** 0 on success, the service's exit status for `exit-code` results, 255 (`EXIT_EXCEPTION`) for a signal, and 1 for other failures. Auth denial or dismissal shows up as "Failed to start transient service unit: Access denied" with exit 1 [man run0 EXIT STATUS; src run.c; systemd `exit-status.h`].
- **doas:** returns the child's status, or 128+signal. Not permitted, wrong password, no tty, config errors and command not found are all exit 1 [OpenDoas doas.1 EXIT STATUS; src].
- **pkcheck** (non-interactive polkit probe): 0 authorized, 1 not authorized, 2 needs interaction or no agent, 3 dismissed, 127 error [man pkcheck 127].

## 7. Finding the binary; pkexec program restrictions

- Go `os.Executable()` on Linux is `readlink("/proc/self/exe")` with a trailing " (deleted)" stripped. It returns an absolute, symlink-resolved path, with "no guarantee that the path is still pointing to the correct executable" [src Go 1.27.1 `os/executable_procfs.go`, `os/executable.go`].
- pkexec accepts a relative program name. It resolves it with `g_find_program_in_path`, then `realpath()`, then checks `access(F_OK)`. The `program` variable shown and matched against `exec.path` is the canonical path [src pkexec.c]. Any program may be run under the default `org.freedesktop.policykit.exec` action (`auth_admin`). No custom action is needed [man pkexec].
- pkexec's polkit subject is its *parent* (`getppid()`), which would be servitor. pkexec sets `PR_SET_PDEATHSIG` and refuses to run if the parent is PID 1 ("Refusing to render service to dead parents") [src pkexec.c].
- run0 resolves the command against the caller's PATH to an absolute path before sending it to PID 1 [src run.c `shall_make_executable_absolute`].
- sudo: the sudoers man page warns: "Users should never be granted sudo privileges to execute files that are writable by the user" [man sudo SECURITY NOTES].

## Implications for servitor's design (inference)

1. **Pass everything identity- and path-related as arguments.** The child cannot rely on HOME, XDG_CONFIG_HOME, XDG_STATE_HOME, XDG_RUNTIME_DIR, PATH, WAYLAND_DISPLAY or DBUS_SESSION_BUS_ADDRESS. The parent should resolve `~` and the config dir (`--config /home/u/.config/servitor`) before elevating. It should also pass the invoking user's home, UID, GID and any state dir as explicit arguments. Every path should be absolute, because pkexec changes the CWD to `/root` while the other tools keep the CWD. SUDO_UID, PKEXEC_UID and DOAS_USER can serve as a cross-check only.
2. **Build argv per tool.** Use `sudo -- <exe> …`, `doas -- <exe> …` and `run0 -- <exe> …`. Use `pkexec <exe> …` with **no `--`**. The `<exe>` comes from `os.Executable()`. Running pkexec with a sanitized `cmd.Env` avoids the SHELL check in `/etc/shells` and the `/`/`%`/`..` checks, which abort with exit 127 (for example a `SHELL=~/.cargo/bin/nu` that is not in `/etc/shells`). Long argument lists are cut to 80 characters in the polkit dialog, so the first ~38 characters should be meaningful.
3. **Exit codes cannot separate auth failure from step failure** (sudo, run0 and doas all use 1). The `__step` child should report its outcome on a separate channel, such as a sentinel or structured line on stdout or a dedicated fd. The parent treats a missing report as "elevation failed or cancelled", and maps pkexec 126 to "cancelled". For run0, capturing stdout puts it in `--pipe` mode, which is the deterministic choice.
4. **What `auto` should check.** "Terminal attached" should mean that `/dev/tty` can be opened (a controlling terminal exists), not just `isatty(stdin)`. With no ctty, sudo and doas cannot prompt unless `sudo -A` with an askpass helper is used. pkexec and run0 need a registered agent. Because a graphical agent wins whenever one runs, choosing sudo in a terminal is what keeps the prompt *in* the terminal. The TUI must hand the terminal over via `tea.ExecProcess` for sudo and doas, and also for run0, pkexec and pkttyagent when no GUI agent is running.
5. **Prompt count per apply.** sudo and doas `persist` give 1 prompt per terminal or parent. pkexec gives one prompt **per elevated step**. run0 gives 1 prompt when there is a ctty and the steps start from the same servitor process within 5 minutes, otherwise one per step. To get one prompt per apply in the no-terminal case, the options are to batch all elevated steps of a rite into one elevated invocation, or to keep one elevated child alive for the run. Both are design decisions.
6. **Binary trust.** If servitor lives in a user-writable path (for example `~/go/bin`), any process running as the user can swap the binary between steps and get root at the next prompt. `os.Executable()` does not protect against this.

## Open questions

- Whether this machine's `/etc/sudoers` (unreadable without root) still has `secure_path`, `timestamp_type` and `timestamp_timeout` at their defaults, and whether `/etc/sudoers.d` adds NOPASSWD rules. This can only be checked with root.
- The run0 no-agent/no-ctty error text, and the run0 temporary-auth reuse with no ctty, come from reading source and were not tested. A safe test would need a throwaway user or VM.
- Whether to support sudo with `-A` (askpass) for launcher or hotkey contexts. That would need a configured askpass helper, which is a user decision.
- Whether batching elevated steps (one prompt per apply) is acceptable given the "one elevated step = one re-exec" design. With pkexec, and with run0 lacking a ctty, the current design means one prompt per step.
- Whether `auto` should prefer run0 over pkexec when there is no terminal. Both need an agent. run0 caches auth only when there is a ctty, while pkexec never caches.
