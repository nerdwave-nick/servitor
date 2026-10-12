package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/nerdwave-nick/servitor/internal/exalted"
	"github.com/nerdwave-nick/servitor/internal/hallowed"
	"github.com/nerdwave-nick/servitor/internal/rituals"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// altarHall names, to a servitor the trials awaken as a process of its own,
// the hall of its altar: bin/ holds a stub intercessor that vouches for the
// master, store/ the sanctions, ground/ the world of the exalted servitor
// (with the servitor installed, which exe names as /proc/self/exe would),
// halls/ its working halls and dominions/ its dominions; journal is its
// journal. Its exalted servitor is awakened as a process of its own, in a
// session of its own, so that it outlives the servitor that raised it.
const altarHall = "SERVITOR_TEST_ALTAR"

// raiseAltarOfTheProcess lets a servitor awakened by the trials with an
// altar raise its exalted servitor upon that altar alone.
func raiseAltarOfTheProcess() {
	dir := os.Getenv(altarHall)
	if dir == "" {
		return
	}
	bin := filepath.Join(dir, "bin")
	w := rituals.Intercession
	w.LookPath = func(name string) (string, error) {
		p := filepath.Join(bin, name)
		if _, err := os.Stat(p); err != nil {
			return "", exec.ErrNotFound
		}
		return p, nil
	}
	w.Attended = func() bool { return true }
	rituals.Intercession, rituals.SanctionRoot = w, filepath.Join(dir, "store")
	rituals.Summon = func(cmd *exec.Cmd, end *os.File) (func() error, error) {
		if filepath.Dir(cmd.Path) != bin {
			return nil, fmt.Errorf("the trials never call upon %s", cmd.Path)
		}
		if err := exec.Command(cmd.Path, cmd.Args[1:]...).Run(); err != nil {
			return func() error { return err }, nil
		}
		i := slices.Index(cmd.Args, exalted.Mode)
		helper := exec.Command(os.Args[0], cmd.Args[i+1:]...)
		helper.Env = append(os.Environ(), awakenAs+"=exalted")
		helper.Stdin, helper.Stdout, helper.Stderr = end, end, os.Stderr
		helper.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := helper.Start(); err != nil {
			return nil, err
		}
		return helper.Wait, nil
	}
}

// fileJournal journals the acts of an exalted servitor, one line each.
type fileJournal string

func (j fileJournal) Record(e exalted.Entry) error {
	f, err := os.OpenFile(string(j), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(f, e.String())
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// serveAsExalted is the exalted servitor awakened by the trials upon the
// altar its environment names, the invoker of the trials standing in for
// the root.
func serveAsExalted(args []string) int {
	dir := os.Getenv(altarHall)
	me := uint32(os.Getuid())
	mine := func(uid uint32) bool { return uid == me }
	getenv := func(k string) (string, bool) {
		return strconv.Itoa(int(me)), k == "SUDO_UID"
	}
	cfg := exalted.Config{
		Exalted:   func() bool { return true },
		Getenv:    getenv,
		Store:     filepath.Join(dir, "store"),
		Sovereign: mine,
		Ground: hallowed.Ground{Root: filepath.Join(dir, "ground"), Sovereign: mine,
			Lord: hallowed.Owner{UID: me, GID: uint32(os.Getgid())}},
		Self:      filepath.Join(dir, "exe"),
		Dominions: filepath.Join(dir, "dominions"),
		Journal:   fileJournal(filepath.Join(dir, "journal")),
		Lament:    os.Stderr,
		Voice: exalted.Speech{Halls: filepath.Join(dir, "halls"), Sovereign: mine, Tongues: hallowed.Machine,
			Getenv: getenv}.Voice,
	}
	return exalted.Serve(args, os.Stdin, os.Stdout, cfg)
}

// TestInvoke_TheExaltedServitorRevertsWhenItsInvokerIsSlain: the servitor
// that invoked is slain mid-liturgy, after the exalted verse was performed;
// the exalted servitor, a process of its own, hears the end of its channel,
// reverts the verse it performed by itself and journals it. No chronicle
// line is written.
func TestInvoke_TheExaltedServitorRevertsWhenItsInvokerIsSlain(t *testing.T) {
	e := newEnv(t)
	dir := t.TempDir()
	for _, hall := range []string{"bin", "store", "halls"} {
		if err := os.Mkdir(filepath.Join(dir, hall), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "bin", "sudo"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "ground"), 0o755); err != nil {
		t.Fatal(err)
	}
	w, err := enshrine(filepath.Join(dir, "ground"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "ground", w.Servitor), filepath.Join(dir, "exe")); err != nil {
		t.Fatal(err)
	}
	marker, release := filepath.Join(e.state, "slain.spoken"), filepath.Join(e.state, "slain.release")
	e.addRite("rites/slain.json", `{"pattern": "Mark I", "aspects": ["on"], "liturgy": [
	  {"incantation": "echo performed >> $TARGET.exalted", "reversion": "echo reverted >> $TARGET.exalted",
	   "exalted": true},
	  {"incantation": "`+gate(marker, release)+`"}]}`)
	e.sanctify(sanction.Store{Root: filepath.Join(dir, "store"), UID: os.Getuid()}, "slain", "slain")

	cmd := e.servitorProcess("invoke", "slain", "on")
	cmd.Env = append(cmd.Env, altarHall+"="+dir)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	await(t, marker, "the gate was never reached")
	if got, _ := os.ReadFile(e.target + ".exalted"); string(got) != "performed\n" {
		t.Fatalf("the exalted verse said %q before the gate", got)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	var exit *exec.ExitError
	if err := cmd.Wait(); !errors.As(err, &exit) || !exit.Sys().(syscall.WaitStatus).Signaled() {
		t.Fatalf("the servitor was not slain: %v", err)
	}
	touch(t, release)

	journal := filepath.Join(dir, "journal")
	deadline := time.Now().Add(10 * time.Second)
	for {
		got, _ := os.ReadFile(journal)
		if strings.Contains(string(got), "deed=fall") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the exalted servitor never reverted; it journaled:\n%s", got)
		}
		time.Sleep(10 * time.Millisecond)
	}
	got, _ := os.ReadFile(journal)
	lines := strings.Split(strings.TrimSpace(string(got)), "\n")
	if len(lines) != 4 || !strings.Contains(lines[1], "verse=1 sigil=") || !strings.Contains(lines[1], "deed=perform") ||
		!strings.Contains(lines[2], "deed=revert verdict=reverted") || !strings.Contains(lines[3], "deed=fall verdict=reverted") {
		t.Errorf("journaled:\n%s", got)
	}
	if said, _ := os.ReadFile(e.target + ".exalted"); string(said) != "performed\nreverted\n" {
		t.Errorf("the exalted verse said %q", said)
	}
	if lines := e.chronicleLines(); lines != nil {
		t.Errorf("chronicled %q", lines)
	}
}
