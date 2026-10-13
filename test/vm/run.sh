#!/bin/bash
# The trial of exalted verses upon the real intercessors, in a disposable Arch
# Linux VM (KVM/QEMU and cloud-init), grown from the harness of 2026-10-12 in
# docs/research/elevation-verification.md (spec §17.9).
#
#   test/vm/run.sh [<rev>]        # <rev> defaults to HEAD
#
# The guest installs servitor from packaging/arch/PKGBUILD of <rev>. A driver
# then runs every scenario as the user "tester" in the session auto-logged-in
# on tty1 (so polkit treats it as active, as on a desktop), each in its own
# pseudo-terminal: it types the password whenever a proof is asked and counts
# the proofs. Its log, one "ok"/"not ok" line per assertion and a verdict last,
# lands in $SERVITOR_VM_DIR/share/trial.log and is printed at the end.
#
# The scenarios: refusals of the unsanctioned and the lapsed before any proof;
# sanction; invoke on and off with exactly one proof each through sudo, run0
# and pkexec (through an agent of the session, agent.py, for polkit 127 refuses
# pkexec's own agent); an exalted verse that falls, reverted with no second
# proof; the invoker slain mid-liturgy, the exalted servitor reverting alone;
# sudo servitor census; the cogitator lending sudo its terminal, and asking
# through the session's agent for run0 (with run0 -k) and pkexec; abjure; and,
# as root, the journal and the store of sanctions.
#
# SERVITOR_VM_DIR keeps the cloud image, the overlay and the guest's share
# (default ${XDG_CACHE_HOME:-~/.cache}/servitor-vm); nothing is written
# elsewhere, and nothing is elevated on the host. A missing image is fetched
# and checked against the mirror's SHA-256.
#
# Exit: 0 the verdict is PASS, 1 FAIL, 2 the VM gave no verdict.
# Needs: qemu-system-x86_64 with /dev/kvm, qemu-img, xorriso, git, curl.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
checkout=$(git -C "$here" rev-parse --show-toplevel)
rev=$(git -C "$checkout" rev-parse --verify "${1:-HEAD}^{commit}")
vm=${SERVITOR_VM_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/servitor-vm}
image_url=https://geo.mirror.pkgbuild.com/images/latest/Arch-Linux-x86_64-cloudimg.qcow2

mkdir -p "$vm/seed" "$vm/share"
cd "$vm"
if [ ! -e arch.qcow2 ]; then
	curl -fL -o arch.qcow2.part "$image_url"
	want=$(curl -fsL "$image_url.SHA256" | awk '{print $1}')
	echo "$want  arch.qcow2.part" | sha256sum -c -
	mv arch.qcow2.part arch.qcow2
fi

# A fresh share: the checkout as a bundle (every branch and tag, so that the
# PKGBUILD's git describe finds its version), the commit under trial, and the
# guest's own scrolls.
rm -rf share/* overlay.qcow2 seed.iso serial.log
chmod 0777 share
git -C "$checkout" bundle create -q "$vm/share/servitor.bundle" HEAD --branches --tags
if ! git -C "$checkout" merge-base --is-ancestor "$rev" HEAD &&
	[ -z "$(git -C "$checkout" for-each-ref --contains "$rev" refs/heads refs/tags)" ]; then
	echo "the commit $rev is reachable from neither HEAD nor a branch or tag of $checkout" >&2
	exit 2
fi
echo "$rev" > share/rev
install -m 0755 "$here/guest.sh" "$here/driver.py" "$here/agent.py" share/

cp "$here/user-data" seed/user-data
printf 'instance-id: servitor-trial-%s\nlocal-hostname: servitor-trial\n' "$(date +%s)" > seed/meta-data
xorriso -as genisoimage -output seed.iso -volid cidata -joliet -rock seed/ 2>/dev/null
qemu-img create -q -f qcow2 -b arch.qcow2 -F qcow2 overlay.qcow2 16G

echo "the trial of $rev begins in $vm (about ten minutes)" >&2
timeout 60m qemu-system-x86_64 -enable-kvm -cpu host -m 4096 -smp 4 \
	-drive file=overlay.qcow2,if=virtio -drive file=seed.iso,media=cdrom \
	-virtfs local,path="$vm/share",mount_tag=host0,security_model=none,id=host0 \
	-nic user,model=virtio-net-pci -display none -serial file:serial.log -monitor none || true

if [ ! -s share/trial.log ]; then
	echo "the VM gave no verdict; see $vm/serial.log and $vm/share/guest.log" >&2
	exit 2
fi
cat share/trial.log
case $(tail -n 1 share/trial.log) in
"verdict: PASS"*) exit 0 ;;
"verdict: FAIL"*) exit 1 ;;
*) exit 2 ;;
esac
