#!/bin/bash
# Runs inside the VM as root (cloud-init runcmd): installs servitor from the
# PKGBUILD of the commit under trial, rouses the driver on tty1, then judges
# what only the root may read (the journal, the store of sanctions), gives the
# verdict and powers the VM off.
S=/mnt/share
exec > "$S/guest.log" 2>&1
set -x
LOG=$S/trial.log
say() { echo "$*" >> "$LOG"; }
check() { # check <description> <command…>
	local what=$1
	shift
	if "$@" >> "$S/guest-checks.log" 2>&1; then say "ok - $what"; else say "not ok - $what"; fi
}
give_verdict() {
	local failed total
	failed=$(grep -c '^not ok' "$LOG")
	total=$(grep -c -E '^(not )?ok' "$LOG")
	if [ "$failed" = 0 ] && [ "$total" -gt 0 ]; then
		say "verdict: PASS ($total assertions)"
	else
		say "verdict: FAIL ($failed of $total assertions failed)"
	fi
	sync
	poweroff
}
trap give_verdict EXIT
# The driver, as tester, writes its assertions and transcripts here too; the
# share keeps the host's owner, so tester may write only what is laid 0666.
: > "$LOG"
: > "$S/transcripts.log"
chmod 0666 "$LOG" "$S/transcripts.log"

say "# the trial of exalted verses, $(date -u +%FT%TZ)"
pacman-key --init && pacman-key --populate archlinux
# The kernel is held back, so that the running kernel keeps its modules.
pacman -Syu --noconfirm --needed --ignore linux base-devel go git python python-gobject sudo polkit > "$S/pacman.log" 2>&1
check "the forge's tools are installed" pacman -Q base-devel go git python python-gobject sudo polkit
systemctl daemon-reexec
systemctl restart polkit.service
systemctl start polkit-agent-helper.socket || true

rev=$(cat "$S/rev")
runuser -u tester -- git clone -q "$S/servitor.bundle" /home/tester/servitor
runuser -u tester -- git -C /home/tester/servitor checkout -q --detach "$rev"
check "the clone stands at $rev" test "$(runuser -u tester -- git -C /home/tester/servitor rev-parse HEAD)" = "$rev"

# The README's makepkg -si, its two halves apart: tester's sudo asks a
# password, and every proof of this trial is counted.
runuser -u tester -- bash -c 'cd ~/servitor/packaging/arch && makepkg --noconfirm' > "$S/makepkg.log" 2>&1
check "makepkg forges the package (its check() included)" grep -q 'Finished making: servitor' "$S/makepkg.log"
pkg=$(ls /home/tester/servitor/packaging/arch/servitor-[0-9]*.pkg.tar.zst)
pacman -U --noconfirm "$pkg" >> "$S/pacman.log" 2>&1
check "pacman installs /usr/bin/servitor, the root's and 0755" \
	test "$(stat -c '%U:%G %a' /usr/bin/servitor)" = "root:root 755"
say "# $(pacman -Q servitor sudo polkit systemd | tr '\n' ' ')kernel $(uname -r)"

touch /run/servitor-trial-ready
systemctl restart getty@tty1.service
for _ in $(seq 1800); do
	[ -e "$S/driver.done" ] && break
	sleep 1
done
check "the driver ended" test -e "$S/driver.done"

# What the driver cannot behold as tester.
journalctl -t servitor -o short-iso --no-pager > "$S/journal.log"
j() { grep -E "$1" "$S/journal.log"; }
check "the journal tells split-lock sanctioned" j 'rite="split-lock".* former=none deed=sanction verdict=sanctioned'
check "the journal tells split-lock's exalted verses performed" j 'rite="split-lock" aspect="on" verse=1 .*deed=perform verdict=triumph'
check "the journal tells the fallen verse of trial-fall" j 'rite="trial-fall" aspect="on" verse=3 .*deed=perform verdict=fell'
check "the journal tells trial-fall's first verse reverted" j 'rite="trial-fall" aspect="on" verse=1 .*deed=revert verdict=reverted'
check "the journal tells trial-slain's verse reverted by the exalted servitor alone" j 'rite="trial-slain" aspect="on" verse=1 .*deed=revert verdict=reverted'
check "the journal tells trial-slain's fall, its channel lost" j 'rite="trial-slain" aspect="on" .*deed=fall verdict=reverted'
check "the journal tells split-lock abjured" j 'rite="split-lock" .*deed=abjure verdict=abjured'
check "the store's halls stand root:root 0755" \
	test "$(stat -c '%U:%G %a' /var/lib/servitor /var/lib/servitor/sanctions)" = "$(printf 'root:root 755\nroot:root 755')"
check "the master's hall of sanctions stands root:tester 0750" \
	test "$(stat -c '%U:%G %a' /var/lib/servitor/sanctions/1500)" = "root:tester 750"
check "no working hall of exalted speech outlives its invocation" \
	test -z "$(find /run/servitor -mindepth 1 -maxdepth 1 ! -name dominions 2>/dev/null)"
check "no trial vessel remains in /etc" test ! -e /etc/servitor-trial-fall.conf -a ! -e /etc/servitor-trial-slain.conf
