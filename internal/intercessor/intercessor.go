// Package intercessor calls upon the intercessor, who vouches for the
// master before the exalted servitor is raised with the machine's own
// authority.
//
// Find judges the installed servitor first: the exalted servitor is always
// /usr/bin/servitor, never the servitor that asks, and it must stand where
// the root alone may write, it and every hall above it. It then finds the
// intercessor the settings name upon the PATH: for "auto", sudo, else run0,
// else pkexec when a terminal attends (/dev/tty opens), and run0, else
// pkexec, when none does; a value of its own calls upon that one alone.
// Command speaks the exact words that raise the exalted servitor through
// it for an invocation, Sanction and Abjure those that raise it to keep the
// store of sanctions. Every lament of Find is a refusal before anything was
// touched.
package intercessor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/exalted"
	"github.com/nerdwave-nick/servitor/internal/hallowed"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// Servitor is the installed servitor, the only one ever raised as the
// exalted servitor.
const Servitor = exalted.Installed

// World is where the intercessor is sought. Machine is the machine's own;
// the trials raise worlds of their own.
type World struct {
	// LookPath finds an intercessor upon the PATH.
	LookPath func(file string) (string, error)
	// Attended tells whether a terminal attends the invocation.
	Attended func() bool
	// Ground is where the installed servitor is judged, and Servitor its
	// path within it.
	Ground   hallowed.Ground
	Servitor string
}

// Machine is the world of the machine itself.
func Machine() World {
	return World{LookPath: exec.LookPath, Attended: attended, Ground: hallowed.Machine, Servitor: Servitor}
}

// attended tells whether a terminal attends the servitor: its controlling
// terminal, /dev/tty, opens.
func attended() bool {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

// sought are the intercessors "auto" seeks, in turn, with a terminal
// attending and without one. sudo asks at a terminal; without one it asks
// only through a helper its master laid outside the servitor, so only a
// settings naming it calls upon it there.
var sought = map[bool][]string{
	true:  {librarium.IntercessorSudo, librarium.IntercessorRun0, librarium.IntercessorPkexec},
	false: {librarium.IntercessorRun0, librarium.IntercessorPkexec},
}

// Intercessor is an intercessor found upon the machine.
type Intercessor struct {
	Name     string // sudo, run0 or pkexec
	Path     string // where the PATH holds it
	servitor string
}

// Enshrined judges the installed servitor; the lament is *Unenshrined.
func (w World) Enshrined() error {
	if err := w.Ground.Enshrined(w.Servitor); err != nil {
		return &Unenshrined{Servitor: w.Servitor, Err: err}
	}
	return nil
}

// Find judges the installed servitor and finds the intercessor setting
// names ("" for the settings unwritten, which is "auto"). The lament is
// *Unenshrined or *Absent, or an intercessor the codex does not know.
func (w World) Find(setting string) (*Intercessor, error) {
	if err := w.Enshrined(); err != nil {
		return nil, err
	}
	var names []string
	attended := w.Attended()
	switch setting {
	case "", librarium.IntercessorAuto:
		setting, names = librarium.IntercessorAuto, sought[attended]
	case librarium.IntercessorSudo, librarium.IntercessorRun0, librarium.IntercessorPkexec:
		names = []string{setting}
	default:
		return nil, fmt.Errorf("%q is no intercessor the codex knows; the \"intercessor\" of the settings must be "+
			"%q, %q, %q or %q", setting, librarium.IntercessorAuto, librarium.IntercessorSudo,
			librarium.IntercessorRun0, librarium.IntercessorPkexec)
	}
	for _, name := range names {
		if path, err := w.LookPath(name); err == nil {
			return &Intercessor{Name: name, Path: path, servitor: w.Servitor}, nil
		}
	}
	return nil, &Absent{Setting: setting, Attended: attended, Sought: names}
}

// Command is the intercessor called upon to raise the exalted servitor for
// the aspect of the rite, its plan bearing sigil (an edict names no
// aspect). Its words are exact:
//
//	sudo -k -p "servitor: authenticate to perform <rite> (sigil <sigil12>): " -- <servitor> __exalt …
//	run0 --unit=servitor-<rite>-<sigil8> --pipe --chdir=/ <servitor> __exalt …
//	pkexec <servitor> __exalt …
//
// where … is <rite> <sigil12> [<aspect>]. A rite's name never bears the %
// that sudo would read in its prompt. The caller lays the channel upon it.
func (i *Intercessor) Command(rite, sigil, aspect string) *exec.Cmd {
	short := sanction.Short(sigil)
	exalt := []string{rite, short}
	if aspect != "" {
		exalt = append(exalt, aspect)
	}
	return i.call(fmt.Sprintf("perform %s (sigil %s)", rite, short),
		fmt.Sprintf("servitor-%s-%s", rite, short[:min(8, len(short))]), exalt)
}

// Sanction is the intercessor called upon to raise the exalted servitor
// that sanctions the unit of the rite bearing sigil (the unit's whole
// sigil, which it cuts as Command does). Its words are exact:
//
//	sudo -k -p "servitor: authenticate to sanction <rite> (sigil <sigil12>): " -- <servitor> __exalt --sanction <rite> <sigil12>
//	run0 --unit=servitor-sanction-<rite>-<sigil8> --pipe --chdir=/ <servitor> __exalt --sanction <rite> <sigil12>
//	pkexec <servitor> __exalt --sanction <rite> <sigil12>
func (i *Intercessor) Sanction(rite, sigil string) *exec.Cmd {
	short := sanction.Short(sigil)
	return i.call(fmt.Sprintf("sanction %s (sigil %s)", rite, short),
		fmt.Sprintf("servitor-sanction-%s-%s", rite, short[:min(8, len(short))]),
		[]string{exalted.SanctionWord, rite, short})
}

// Abjure is the intercessor called upon to raise the exalted servitor that
// abjures the sanction of the rite, or every sanction when rite is "". Its
// words are exact:
//
//	sudo -k -p "servitor: authenticate to abjure the sanction of <rite>: " -- <servitor> __exalt --abjure <rite>
//	sudo -k -p "servitor: authenticate to abjure every sanction: " -- <servitor> __exalt --abjure --all
//	run0 --unit=servitor-abjure-<rite> --pipe --chdir=/ <servitor> __exalt --abjure <rite>
//	run0 --unit=servitor-abjure --pipe --chdir=/ <servitor> __exalt --abjure --all
//	pkexec <servitor> __exalt --abjure <rite> | --all
func (i *Intercessor) Abjure(rite string) *exec.Cmd {
	if rite == "" {
		return i.call("abjure every sanction", "servitor-abjure", []string{exalted.AbjureWord, exalted.AllWord})
	}
	return i.call("abjure the sanction of "+rite, "servitor-abjure-"+rite, []string{exalted.AbjureWord, rite})
}

// call is the intercessor called upon to raise the exalted servitor with
// the words exalt after its Mode: sudo asking the master's proof "to
// <deed>", run0 in the unit of the machine named unit.
func (i *Intercessor) call(deed, unit string, exalt []string) *exec.Cmd {
	var args []string
	switch i.Name {
	case librarium.IntercessorSudo:
		args = []string{"-k", "-p", "servitor: authenticate to " + deed + ": ", "--"}
	case librarium.IntercessorRun0:
		args = []string{"--unit=" + unit, "--pipe", "--chdir=/"}
	}
	args = append(append(append([]string{i.Name}, args...), i.servitor, exalted.Mode), exalt...)
	return &exec.Cmd{Path: i.Path, Args: args}
}

// Absent is the lament of an intercessor nowhere to be found upon the PATH.
type Absent struct {
	Setting  string   // as the settings name it; "auto" when unwritten
	Attended bool     // whether a terminal attended
	Sought   []string // the intercessors sought, in turn
}

func (a *Absent) Error() string {
	if a.Setting != librarium.IntercessorAuto {
		return fmt.Sprintf("the intercessor %q the settings name is nowhere to be found upon the PATH, and no other is "+
			"called upon in its stead: install it, or name another as the \"intercessor\" of the settings",
			a.Setting)
	}
	at := "with a terminal attending, the servitor calls upon sudo, else run0, else pkexec"
	if !a.Attended {
		at = "without a terminal, the servitor calls upon run0, else pkexec, for sudo asks for the master's proof " +
			"only at a terminal"
	}
	return fmt.Sprintf("no intercessor is found upon the PATH to vouch for the master: %s, and neither %s stands "+
		"there — install one, or name the one to call upon as the \"intercessor\" of the settings", at,
		strings.Join(a.Sought, " nor "))
}

// Unenshrined is the lament of an installed servitor that does not stand
// where the root alone may write.
type Unenshrined struct {
	Servitor string
	Err      error // what the rule found
}

func (u *Unenshrined) Error() string {
	return fmt.Sprintf("the exalted servitor cannot be raised, for %v: it is always %s, never the servitor that asks, "+
		"and it must stand where the root alone may write, it and every hall above it — install the servitor "+
		"anew from its package (makepkg -si)", u.Err, u.Servitor)
}

func (u *Unenshrined) Unwrap() error { return u.Err }
