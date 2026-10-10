package vox

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/nerdwave-nick/servitor/internal/invocation"
)

// notifySend is the program that carries vox-casts to the desktop.
const notifySend = "notify-send"

// patience is how long notify-send may labour before it is abandoned; a
// desktop without a notification daemon must not stall the invocation.
const patience = 5 * time.Second

// desktop is the one notification of an invocation.
type desktop struct {
	id        string   // the notification's id; "" before the first is shown
	lingering *Message // the last progress sent, which never expires
}

// send shows m, replacing the notification shown before.
//
// Every call asks notify-send to print the id (-p): the first call learns
// it, later ones replace it (-r <id>) and follow the id answered, should
// the daemon have shown a new notification instead. Progress is shown with
// "-t 0", so it does not expire while the rite runs however long its
// steps labour, and normal urgency, so it neither stays nor alarms once
// replaced; rest lets a lingering progress expire. Triumph keeps the
// daemon's own expiry. A fall is critical: the freedesktop specification
// lets critical notifications stay until they are dismissed, so a fall
// cannot pass unseen.
func (d *desktop) send(m Message, resting bool) {
	program, err := exec.LookPath(notifySend)
	if err != nil {
		return
	}
	args := []string{"-a", "servitor", "-p"}
	if d.id != "" {
		args = append(args, "-r", d.id)
	}
	d.lingering = nil
	switch m.Tidings {
	case invocation.Progress:
		args = append(args, "-u", "normal")
		if !resting {
			args = append(args, "-t", "0", "-h", "int:value:"+strconv.Itoa(m.Percent))
			d.lingering = &m
		}
	case invocation.Success:
		args = append(args, "-u", "normal")
	case invocation.Failure:
		args = append(args, "-u", "critical")
	}
	args = append(args, "--", m.Summary, m.Body)
	ctx, cancel := context.WithTimeout(context.Background(), patience)
	defer cancel()
	out, err := exec.CommandContext(ctx, program, args...).Output()
	if err != nil {
		return
	}
	if id := strings.TrimSpace(string(out)); isID(id) {
		d.id = id
	}
}

// rest sends a lingering progress once more, so that it may expire.
func (d *desktop) rest() {
	if d.lingering != nil {
		d.send(*d.lingering, true)
	}
}

func isID(s string) bool {
	n, err := strconv.ParseUint(s, 10, 32)
	return err == nil && n > 0
}
