#!/usr/bin/env python3
"""The driver of test/vm/run.sh: the trial of exalted subjects upon sudo, run0 and pkexec.

It runs inside the VM as "tester", in the session auto-logged-in on tty1, so
that polkit judges it active, as on a desktop. Every scenario runs in a
pseudo-terminal of its own, its controlling terminal; the driver types the
password whenever a proof is asked and counts the proofs. Each assertion is one
line of /mnt/share/trial.log, "ok - …" or "not ok - …", with what was seen
beneath a failure; guest.sh adds the root's assertions and the verdict.
Every transcript of servitor at a terminal is kept in /mnt/share/transcripts.log.

Every rite is of pattern Mark II but one: trial-gaming is laid in pattern
Mark I with exalted verses, converted as the servitor converts (test/vm/convert),
and then given auspexes by its master as the faithful would, one of them an
exalted auspex the invoker cannot read.

The cogitator is driven by its keys: sudo is lent its terminal (and Ctrl-C at
sudo's prompt is tried), while run0 and pkexec, called upon apart from any
terminal, ask through agent.py, which stands in for the desktop's agent.
"""
import fcntl
import json
import os
import pty
import re
import select
import shutil
import signal
import struct
import subprocess
import termios
import time

PW = "servitor-trial"
SHARE = "/mnt/share"
HOME = os.path.expanduser("~")
LIB = f"{HOME}/.config/servitor"
RITES = f"{LIB}/rites"
CHRONICLE = f"{HOME}/.local/state/servitor/chronicle.jsonl"
SLATES = f"{HOME}/.local/state/servitor/data-slates"
FORGE = f"{HOME}/servitor"
KNOB = "/proc/sys/kernel/split_lock_mitigate"
KEPT = "/etc/sysctl.d/90-split-lock.conf"
# What only the root may read: the omen of trial-gaming's subject
# incantation-3, read by its exalted auspex alone.
HIDDEN = "/etc/servitor-trial-hidden"
# sudo asks "servitor: authenticate to perform <rite> (sigil <sigil>): ", or
# "[sudo] password for tester: "; the agents of polkit ask "Password: ".
PROMPT = re.compile(rb"(password[^\n]*:|authenticate to [^\n]*\):)\s*$", re.I)
ANSI = re.compile(r"\x1b\[[0-9;?<>=]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[()][0-9A-Za-z]|\x1b[=>78]")

LOG = open(f"{SHARE}/trial.log", "a", buffering=1)


def say(line):
    LOG.write(line + "\n")


def check(cond, what, seen=""):
    say(("ok - " if cond else "not ok - ") + what)
    if not cond and seen:
        for line in str(seen).strip().splitlines()[-25:]:
            say("    | " + line)
    return cond


class Term:
    """A program in a pseudo-terminal of its own, its proofs answered and counted."""

    def __init__(self, argv, env=None, answer=True):
        self.out, self.mark, self.proofs, self.answer, self.code = b"", 0, 0, answer, None
        self.pid, self.fd = pty.fork()
        if self.pid == 0:
            fcntl.ioctl(0, termios.TIOCSWINSZ, struct.pack("HHHH", 40, 120, 0, 0))
            e = dict(os.environ, TERM="xterm-256color")
            e.update(env or {})
            os.execvpe(argv[0], argv, e)

    def pump(self, wait=0.1):
        r, _, _ = select.select([self.fd], [], [], wait)
        if r:
            try:
                chunk = os.read(self.fd, 65536)
            except OSError:
                chunk = b""
            self.out += chunk
        if self.answer and PROMPT.search(self.out[self.mark:]):
            time.sleep(0.3)
            os.write(self.fd, (PW + "\n").encode())
            self.proofs += 1
            self.mark = len(self.out)
        return self

    def asking(self):
        return PROMPT.search(self.out[self.mark:]) is not None

    def alive(self):
        if self.code is None:
            pid, status = os.waitpid(self.pid, os.WNOHANG)
            if pid:
                self.code = os.waitstatus_to_exitcode(status)
        return self.code is None

    def send(self, keys, settle=0.4):
        os.write(self.fd, keys.encode())
        pump_for(settle, self)

    def text(self):
        return ANSI.sub("", self.out.decode("utf8", "replace")).replace("\r", "")

    def wait(self, timeout=120):
        end = time.time() + timeout
        while self.alive() and time.time() < end:
            self.pump()
        if self.alive():
            os.kill(self.pid, signal.SIGKILL)
            os.waitpid(self.pid, 0)
            self.code = "slain after %ss" % timeout
        pump_for(0.2, self)
        return self.code


def pump_for(seconds, *terms):
    end = time.time() + seconds
    while time.time() < end:
        for t in terms:
            t.pump(0.05)


def pump_until(cond, timeout, *terms):
    end = time.time() + timeout
    while time.time() < end:
        if cond():
            return True
        for t in terms:
            t.pump(0.05)
    return cond()


def servitor(*args, timeout=120, agent=None):
    """Runs servitor in a terminal of its own until it ends; the proofs an
    agent asked meanwhile count among its own."""
    t = Term(["servitor", *args])
    asked = agent.proofs if agent else 0
    end = time.time() + timeout
    while t.alive() and time.time() < end:
        pump_for(0.1, t, *([agent] if agent else []))
    t.wait(1)
    if agent:
        t.proofs += agent.proofs - asked
    with open(f"{SHARE}/transcripts.log", "a") as f:
        f.write(f"\n❯ servitor {' '.join(args)}   [exit {t.code}, proofs {t.proofs}]\n{t.text()}")
    return t


def session_agent():
    """The stand-in for the desktop's agent (agent.py), registered for the
    whole tty1 session."""
    agent = Term(["python3", f"{SHARE}/agent.py"], answer=True)
    pump_until(lambda: "the agent is registered" in agent.text() or not agent.alive(), 15, agent)
    check("the agent is registered" in agent.text(), "a polkit agent stands for the whole session", agent.text())
    return agent


def dismiss(agent):
    os.kill(agent.pid, signal.SIGTERM)
    agent.wait(5)


def knob():
    with open(KNOB) as f:
        return f.read().strip()


def last_chronicle():
    try:
        with open(CHRONICLE) as f:
            lines = f.read().splitlines()
        return json.loads(lines[-1]) if lines else {}
    except FileNotFoundError:
        return {}


def chronicle_count():
    try:
        with open(CHRONICLE) as f:
            return len(f.read().splitlines())
    except FileNotFoundError:
        return 0


def slate(name):
    try:
        with open(f"{SLATES}/{name}.json") as f:
            return json.load(f)
    except FileNotFoundError:
        return {}


def settings(intercessor=None):
    s = {"pattern": "Mark II"}
    if intercessor:
        s["intercessor"] = intercessor
    with open(f"{LIB}/servitor.json", "w") as f:
        json.dump(s, f)


def rite(name, subjects, aspects=("on", "off")):
    with open(f"{RITES}/{name}.json", "w") as f:
        json.dump({"pattern": "Mark II", "purpose": f"the trial {name}", "aspects": list(aspects),
                   "subjects": list(subjects)}, f, indent=1)


def lay_the_librarium():
    os.makedirs(RITES, exist_ok=True)
    shutil.copy(f"{FORGE}/examples/rites/split-lock.json", f"{RITES}/split-lock.json")
    rite("trial-unsanctioned", [
        {"name": "nothing", "designation": "machine-spirit", "exalted": True, "enact": {"*": "true"}},
    ])
    # The exalted subject "wrong" never shows the omen "on": its proof fails,
    # and everything set before it is restored to what was read.
    rite("trial-fall", [
        {"name": "vessel", "designation": "transcription", "exalted": True,
         "vessel": "/etc/servitor-trial-fall.conf", "omens": {"on": "fallen on\n", "*": None}},
        {"name": "mark", "designation": "machine-spirit", "omens": {"on": "performed", "*": "reverted"},
         "auspex": {"scan": "cat fall.mark 2>/dev/null || echo reverted"}, "enact": {"*": "echo {{omen}} > fall.mark"}},
        {"name": "wrong", "designation": "machine-spirit", "exalted": True,
         "auspex": {"scan": "cat /etc/servitor-trial-wrong 2>/dev/null || echo off"},
         "enact": {"*": "rm -f /etc/servitor-trial-wrong"}},
    ])
    rite("trial-slain", [
        {"name": "vessel", "designation": "transcription", "exalted": True,
         "vessel": "/etc/servitor-trial-slain.conf", "omens": {"on": "slain on\n", "*": None}},
        {"name": "sleeper", "designation": "machine-spirit", "enact": {"*": "touch slain.mark; sleep 120"},
         "patience": "5m"},
    ])
    settings()


def convert_trial_gaming():
    """trial-gaming, laid in pattern Mark I with exalted verses, is converted
    as the servitor converts on its first reading, then given its auspexes
    by its master: the knob is read by the invoker, the hidden omen by an
    exalted auspex alone."""
    path = f"{RITES}/trial-gaming.json"
    with open(path, "w") as f:
        json.dump({"pattern": "Mark I", "purpose": "the trial trial-gaming", "aspects": ["on", "off"], "liturgy": [
            {"incantation": {"on": "sysctl -wq kernel.split_lock_mitigate=0",
                             "*": "sysctl -wq kernel.split_lock_mitigate=1"}, "exalted": True},
            {"vox-cast": "progress"},
            {"incantation": {"on": f"umask 077; echo on > {HIDDEN}", "*": f"umask 077; echo off > {HIDDEN}"},
             "exalted": True},
            {"vox-cast": "success"},
        ]}, f, indent=1)
    env = dict(os.environ, GOTOOLCHAIN="local", GOFLAGS="-mod=readonly -modcacherw")
    gopath = f"{FORGE}/packaging/arch/src/gopath"
    if os.path.isdir(gopath):
        env["GOPATH"] = gopath
    p = subprocess.run(["go", "run", "./test/vm/convert", LIB], cwd=FORGE, env=env, capture_output=True,
                       text=True, timeout=900)
    with open(path) as f:
        converted = json.load(f)
    check(p.returncode == 0 and 'the rite "trial-gaming" is converted to pattern "Mark II"' in p.stdout
          and "servitor sanction trial-gaming" in p.stdout,
          "trial-gaming is converted, and the tidings bid it be sanctioned anew", p.stdout + p.stderr)
    check(converted.get("pattern") == "Mark II" and os.path.exists(f"{RITES}/.mark-i/trial-gaming.json")
          and [(s["name"], s["designation"], s.get("exalted")) for s in converted["subjects"]]
          == [("incantation-1", "machine-spirit", True), ("incantation-3", "machine-spirit", True)],
          "the converted trial-gaming holds two exalted machine-spirits, its elder scripture kept in .mark-i",
          json.dumps(converted))
    spirits = {s["name"]: s for s in converted["subjects"]}
    spirits["incantation-1"]["auspex"] = {
        "scan": '[ "$(cat /proc/sys/kernel/split_lock_mitigate)" = 0 ] && echo on || echo off'}
    spirits["incantation-3"]["auspex"] = {"scan": f"cat {HIDDEN}", "exalted": True}
    with open(path, "w") as f:
        json.dump(converted, f, indent=1)
    t = servitor("invoke", "trial-gaming", "on")
    check(t.code == 2 and t.proofs == 0 and "stands unsanctioned" in t.text(),
          "the converted trial-gaming is refused before any proof until it is sanctioned", t.text())
    t = servitor("sanction", "trial-gaming")
    check(t.code == 0 and t.proofs == 1, "sanction trial-gaming: one proof, exit 0", t.text())


def tell_the_machine():
    say("# servitor: " + subprocess.run(["servitor", "--version"], capture_output=True, text=True).stdout.strip())
    s = subprocess.run(["loginctl", "show-session", os.environ.get("XDG_SESSION_ID", "self"),
                        "-p", "Active", "-p", "Type", "-p", "TTY"], capture_output=True, text=True).stdout
    say("# session: " + " ".join(s.split()))
    for argv in (["sudo", "-V"], ["pkaction", "--version"], ["systemctl", "--version"]):
        say("# " + subprocess.run(argv, capture_output=True, text=True).stdout.splitlines()[0])


def refusals_before_any_proof(when):
    before = knob()
    for name in ("split-lock", "trial-unsanctioned"):
        t = servitor("invoke", name, "on")
        check(t.code == 2 and t.proofs == 0 and f"servitor sanction {name}" in t.text()
              and "stands unsanctioned" in t.text(),
              f"{when}: the unsanctioned {name} is refused with 2 before any proof", t.text())
    check(knob() == before, f"{when}: the knob is untouched")


def sanction_split_lock():
    t = servitor("invoke", "split-lock", "on", "--foresee")
    check(t.code == 0 and t.proofs == 0 and "✠" in t.text() and "Nothing was performed" in t.text(),
          "foresight marks the exalted verses ✠ and asks no proof", t.text())
    t = servitor("sanction", "split-lock")
    check(t.code == 0 and t.proofs == 1, "sanction split-lock: one proof through sudo, exit 0", t.text())
    t = servitor("sanction", "split-lock")
    check(t.code == 0 and t.proofs == 0, "sanctioned anew unchanged: no proof, exit 0", t.text())
    t = servitor("augury", "split-lock", "sanction")
    check(t.text().strip() == "sanctioned", "the augury tells split-lock sanctioned", t.text())
    t = servitor("sanction")
    check(t.code == 0 and re.search(r"split-lock\s.*sanctioned", t.text()) is not None,
          "servitor sanction lists split-lock sanctioned", t.text())


def unchanged_through(tool, agent):
    """split-lock already bears "on": nothing exalted changes, and nothing is
    asked; its auspex is the invoker's own."""
    lines = chronicle_count()
    t = servitor("invoke", "split-lock", "on", agent=agent)
    entry = last_chronicle()
    check(t.code == 0 and t.proofs == 0, f"{tool}: split-lock invoked again into on asks no proof, exit 0", t.text())
    check(chronicle_count() == lines + 1 and entry.get("verdict") == "triumph" and entry.get("changed") == []
          and "intercessor" not in entry, f"{tool}: the unchanged invocation is chronicled, no intercessor called",
          json.dumps(entry))


def gaming_through(tool, agent):
    """The converted trial-gaming asks one proof at every invocation, for its
    exalted auspex is read by the exalted servitor alone; the subjects that
    already bear the aspect are not set again."""
    both = ["incantation-1", "incantation-3"]
    for aspect, value, changed in (("on", "0", both), ("on", "0", []), ("off", "1", both)):
        lines = chronicle_count()
        t = servitor("invoke", "trial-gaming", aspect, agent=agent)
        entry = last_chronicle()
        again = " again" if not changed else ""
        check(t.code == 0 and t.proofs == 1,
              f"{tool}: invoke trial-gaming {aspect}{again} asks exactly one proof, exit 0", t.text())
        check(knob() == value, f"{tool}: trial-gaming {aspect}{again}: split_lock_mitigate is {value}", knob())
        check(chronicle_count() == lines + 1 and entry.get("verdict") == "triumph"
              and entry.get("changed") == changed and entry.get("intercessor") == tool,
              f"{tool}: trial-gaming {aspect}{again} is chronicled changing {changed or 'nothing'}", json.dumps(entry))
        recorded = slate("trial-gaming").get("omens", {}).get("incantation-3", {})
        check(recorded.get("omen") == aspect and recorded.get("exalted") is True,
              f"{tool}: the data-slate records the hidden omen {aspect}, read by the exalted servitor",
              json.dumps(slate("trial-gaming")))


def invoke_through(tool, agent=None):
    settings(tool)
    for aspect, value in (("on", "0"), ("off", "1")):
        lines = chronicle_count()
        t = servitor("invoke", "split-lock", aspect, agent=agent)
        entry = last_chronicle()
        check(t.code == 0 and t.proofs == 1, f"{tool}: invoke split-lock {aspect} asks exactly one proof, exit 0",
              t.text())
        check(knob() == value, f"{tool}: the kernel's split_lock_mitigate is {value}", knob())
        kept = os.path.exists(KEPT) and open(KEPT).read().strip() == "kernel.split_lock_mitigate = 0"
        check(kept if aspect == "on" else not os.path.exists(KEPT),
              f"{tool}: {KEPT} is {'written' if aspect == 'on' else 'struck'}")
        check(chronicle_count() == lines + 1 and entry.get("verdict") == "triumph"
              and entry.get("intercessor") == tool and len(entry.get("sigil", "")) == 12,
              f"{tool}: the chronicle names the intercessor and the sigil", json.dumps(entry))
        if aspect == "on":
            unchanged_through(tool, agent)
    gaming_through(tool, agent)
    settings()


def pkexec_at_a_terminal():
    """polkit 127's socket-activated helper refuses pkexec's own textual agent
    ("No session for cookie"), whatever pkexec is bidden run; with an agent of
    the session, as on a desktop, pkexec asks through it."""
    t = Term(["pkexec", "/usr/bin/true"])
    t.wait(60)
    if t.code != 0:
        say(f"# known: pkexec /usr/bin/true at a terminal with no agent of the session fails here too "
            f"(exit {t.code}): " + " / ".join(t.text().strip().splitlines()[-3:]))
    agent = session_agent()
    invoke_through("pkexec", agent)
    dismiss(agent)


def lapsed():
    path = f"{RITES}/split-lock.json"
    with open(path) as f:
        sanctioned = f.read()
    with open(path, "w") as f:
        f.write(sanctioned.replace('"sysctl -wq kernel.split_lock_mitigate={{omen}}"',
                                   '"sysctl -wq kernel.split_lock_mitigate={{omen}} # amended"'))
    before = knob()
    t = servitor("invoke", "split-lock", "on")
    check(t.code == 2 and t.proofs == 0 and "has lapsed" in t.text() and "servitor sanction split-lock" in t.text(),
          "a lapsed split-lock is refused with 2 before any proof", t.text())
    check(knob() == before, "the lapsed invocation touched nothing")
    t = servitor("augury", "split-lock", "sanction")
    check(t.text().strip() == "lapsed", "the augury tells split-lock lapsed", t.text())
    with open(path, "w") as f:
        f.write(sanctioned)
    t = servitor("augury", "split-lock", "sanction")
    check(t.text().strip() == "sanctioned", "restored, split-lock stands sanctioned again", t.text())


def a_proof_fails():
    t = servitor("sanction", "trial-fall")
    check(t.code == 0 and t.proofs == 1, "sanction trial-fall: one proof", t.text())
    t = servitor("invoke", "trial-fall", "on")
    entry = last_chronicle()
    check(t.code == 1 and t.proofs == 1,
          "the exalted subject wrong of trial-fall fails its proof: exit 1, and no second proof", t.text())
    check('the subject "wrong" shows the omen "off"; "on" was commanded' in t.text(),
          "the lament names the subject, the omen found and the omen commanded", t.text())
    check(not os.path.exists("/etc/servitor-trial-fall.conf"), "the exalted vessel set before it is struck again")
    mark = open(f"{RITES}/fall.mark").read().strip() if os.path.exists(f"{RITES}/fall.mark") else ""
    check(mark == "reverted", "the common subject between them is restored to the omen read", mark)
    check(entry.get("rite") == "trial-fall" and entry.get("verdict") == "reverted"
          and entry.get("fell_at") == {"subject": "wrong"} and entry.get("intercessor") == "sudo",
          "the chronicle tells trial-fall reverted at the subject wrong", json.dumps(entry))


def the_parent_is_slain():
    t = servitor("sanction", "trial-slain")
    check(t.code == 0 and t.proofs == 1, "sanction trial-slain: one proof", t.text())
    lines = chronicle_count()
    vessel, mark = "/etc/servitor-trial-slain.conf", f"{RITES}/slain.mark"
    t = Term(["servitor", "invoke", "trial-slain", "on"])
    pump_until(lambda: os.path.exists(mark), 120, t)
    check(os.path.exists(vessel), "trial-slain: the exalted vessel stands before its invoker is slain", t.text())
    os.kill(t.pid, signal.SIGKILL)
    t.wait(10)
    gone = pump_until(lambda: not os.path.exists(vessel), 60, t)
    check(gone, "the invoker slain mid-liturgy, the exalted servitor reverts its verse by itself")
    check(t.proofs == 1, "trial-slain asked exactly one proof", t.text())
    check(chronicle_count() == lines, "the slain invocation leaves no line in the chronicle")


def under_sudo():
    t = Term(["sudo", "servitor", "census"])
    t.wait(60)
    check(t.code == 0 and "split-lock" in t.text() and "✠" in t.text() and t.proofs <= 1,
          "sudo servitor census reads tester's Librarium and sanctions", t.text())
    before = knob()
    t = Term(["sudo", "servitor", "invoke", "split-lock", "on"])
    t.wait(60)
    check(t.code == 2 and knob() == before, "sudo servitor invoke is refused as root with 2", t.text())
    subprocess.run(["sudo", "-K"])


def cogitator_sudo():
    settings()
    before, lines = knob(), chronicle_count()
    c = Term(["servitor", "cogitator"])
    pump_for(2, c)
    c.send("/")
    c.send("split-lock")
    c.send("\r", 1)
    c.send(" ")
    turned = pump_until(lambda: knob() != before, 90, c)
    pump_for(2, c)
    entry = last_chronicle()
    check(turned and c.proofs == 1, "cogitator: space lends sudo the terminal, one proof, the aspect turns",
          c.text()[-3000:])
    check(chronicle_count() == lines + 1 and entry.get("intercessor") == "sudo",
          "cogitator: the chronicle names sudo", json.dumps(entry))
    # A proof refused at sudo's own prompt: Ctrl-C there, nothing is touched.
    c.answer = False
    now, lines = knob(), chronicle_count()
    c.send(" ")
    asked = pump_until(c.asking, 60, c)
    c.send("\x03", 3)
    pump_for(2, c)
    check(asked and knob() == now and chronicle_count() == lines,
          "cogitator: Ctrl-C at sudo's prompt touches nothing", c.text()[-3000:])
    check("No exalted servitor was raised" in c.text(), "cogitator: the refusal is shown as a heresy",
          c.text()[-3000:])
    c.answer = True
    c.send("\x1b", 1)
    c.send("q", 1)
    if c.alive():
        c.send("q", 1)
    c.wait(10)
    check(c.code == 0, "the cogitator retreats with 0", c.code)
    if knob() != "1":
        servitor("invoke", "split-lock", "off")


def cogitator_through_an_agent(tool):
    """run0 and pkexec are called upon apart from the terminal, so the desktop's
    agent asks: agent.py stands in for it."""
    settings(tool)
    agent = session_agent()
    c = Term(["servitor", "cogitator"])
    pump_for(2, c, agent)
    c.send("/")
    c.send("split-lock")
    c.send("\r", 1)
    for turn in (1, 2):
        before, asked, lines = knob(), agent.proofs, chronicle_count()
        c.send(" ")
        turned = pump_until(lambda: knob() != before and chronicle_count() > lines, 90, c, agent)
        pump_for(3, c, agent)
        entry = last_chronicle()
        check(turned and agent.proofs - asked == 1 and c.proofs == 0,
              f"cogitator through {tool}, act {turn}: the agent asks exactly one proof, the aspect turns",
              agent.text()[-1500:] + "\n----\n" + c.text()[-2000:])
        check(entry.get("intercessor") == tool, f"cogitator through {tool}, act {turn}: the chronicle names {tool}",
              json.dumps(entry))
        # after an act through run0 the cogitator speaks run0 -k; should it
        # fail, the toast would add its lament ("Yet: …")
        check("Yet:" not in c.text(), f"cogitator through {tool}, act {turn}: no lament follows the act"
              + (" (run0 -k spoken)" if tool == "run0" else ""), c.text()[-2000:])
    c.send("\x1b", 1)
    c.send("q", 1)
    c.wait(10)
    check(c.code == 0, f"the cogitator through {tool} retreats with 0", c.code)
    dismiss(agent)
    settings()


def abjure():
    settings("run0")
    t = servitor("abjure", "split-lock")
    check(t.code == 0 and t.proofs == 1, "abjure split-lock through run0: one proof, exit 0", t.text())
    settings()
    t = servitor("sanction")
    check(t.code == 0 and "split-lock" not in t.text(), "servitor sanction lists split-lock no more", t.text())
    refusals_before_any_proof("abjured")


def main():
    say("# the driver begins, %s" % time.strftime("%FT%TZ", time.gmtime()))
    tell_the_machine()
    lay_the_librarium()
    if not check(os.path.exists(KNOB), f"the kernel of the VM offers {KNOB}"):
        return
    subprocess.run(["sudo", "-K"])
    refusals_before_any_proof("unsanctioned")
    sanction_split_lock()
    convert_trial_gaming()
    check(not os.access(HIDDEN, os.R_OK), f"the invoker may not read {HIDDEN}")
    for tool in ("sudo", "run0"):
        invoke_through(tool)
    pkexec_at_a_terminal()
    lapsed()
    a_proof_fails()
    the_parent_is_slain()
    under_sudo()
    cogitator_sudo()
    for tool in ("run0", "pkexec"):
        cogitator_through_an_agent(tool)
    abjure()
    check(knob() == "1" and not os.path.exists(KEPT), "the machine ends as it began")
    t = servitor("augury", "trial-gaming")
    omen = next((o for o in json.loads(t.text() or "{}").get("omens", []) if o.get("subject") == "incantation-3"), {})
    check(omen.get("omen") == "off" and omen.get("stale") is True and "read_at" in omen,
          "outside an invocation the augury tells the hidden omen as recorded, stale", t.text())


if __name__ == "__main__":
    try:
        main()
    except Exception as ex:  # noqa: BLE001 - every crash is a failed assertion
        check(False, "the driver ran to its end", repr(ex))
        raise
