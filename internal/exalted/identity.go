package exalted

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

// identify tells the uid of the master at the other end of the channel in.
// Upon a socket it is the uid the machine knows its peer by (SO_PEERCRED),
// and SUDO_UID and PKEXEC_UID, when present, must name the same. Upon pipes,
// which sudo lays between them when it records what passes, it is SUDO_UID.
// Without either the master is unknown, and nothing is accepted. The root
// itself is never the master of an exalted servitor.
func identify(in *os.File, getenv func(string) (string, bool)) (int, error) {
	uid, err := peer(in)
	switch {
	case errors.Is(err, unix.ENOTSOCK):
		v, ok := getenv("SUDO_UID")
		if !ok {
			return 0, errors.New("the channel is no socket, and no SUDO_UID names the master; the master " +
				"stays unknown")
		}
		if uid, err = parseUID("SUDO_UID", v); err != nil {
			return 0, err
		}
	case err != nil:
		return 0, fmt.Errorf("the master at the other end of the channel cannot be known: %v", err)
	default:
		for _, key := range []string{"SUDO_UID", "PKEXEC_UID"} {
			v, ok := getenv(key)
			if !ok {
				continue
			}
			named, err := parseUID(key, v)
			if err != nil {
				return 0, err
			}
			if named != uid {
				return 0, fmt.Errorf("the channel's master is uid %d, yet %s names uid %d", uid, key, named)
			}
		}
	}
	if uid == 0 {
		return 0, errors.New("the master of an exalted servitor is never the root itself")
	}
	return uid, nil
}

// peer is the uid of the peer of the socket f.
func peer(f *os.File) (int, error) {
	raw, err := f.SyscallConn()
	if err != nil {
		return 0, err
	}
	var cred *unix.Ucred
	var credErr error
	if err := raw.Control(func(fd uintptr) {
		cred, credErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil {
		return 0, err
	}
	if credErr != nil {
		return 0, credErr
	}
	return int(cred.Uid), nil
}

// parseUID reads the uid the variable key holds.
func parseUID(key, v string) (int, error) {
	uid, err := strconv.ParseUint(v, 10, 31)
	if err != nil {
		return 0, fmt.Errorf("%s=%q names no uid", key, v)
	}
	return int(uid), nil
}
