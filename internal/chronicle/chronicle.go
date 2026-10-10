// Package chronicle keeps the chronicle: one line of JSON for every
// invocation (Entry), appended to a file whose path the caller resolves
// (librarium.Orders.Chronicle).
//
// Before a line would carry the chronicle beyond Limit, Append renames it
// to Rotated(path), replacing the rotated file before it, so at most about
// two limits of history remain. Concurrent invocations take turns through
// an exclusive lock on the chronicle and write each line at once, so lines
// neither interleave nor vanish in a rotation. Read passes over lines that
// a crash tore or a hand garbled.
package chronicle

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"syscall"
)

// Limit is the size in bytes the chronicle may reach before Append rotates
// it.
const Limit = 1 << 20

// Rotated is the path of the chronicle at path once it has been rotated.
func Rotated(path string) string { return path + ".1" }

// Append records e as one line at the end of the chronicle at path,
// creating it and its directories when absent and rotating it first when
// the line would carry it beyond Limit.
func Append(path string, e Entry) error {
	line, err := json.Marshal(e, json.Deterministic(true))
	if err != nil {
		return err
	}
	line = append(line, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return lament(path, err)
	}
	f, held, err := seize(path)
	if err != nil {
		return lament(path, err)
	}
	defer func() { _ = f.Close() }()
	if held > 0 && held+int64(len(line)) > Limit {
		if err := os.Rename(path, Rotated(path)); err != nil {
			return lament(path, err)
		}
		_ = f.Close() // frees those waiting upon the rotated chronicle
		if f, held, err = seize(path); err != nil {
			return lament(path, err)
		}
	}
	if held > 0 {
		last := make([]byte, 1)
		if _, err := f.ReadAt(last, held-1); err != nil {
			return lament(path, err)
		}
		if last[0] != '\n' { // a torn line must not swallow this one
			line = append([]byte{'\n'}, line...)
		}
	}
	if _, err := f.Write(line); err != nil {
		return lament(path, err)
	}
	return nil
}

// seize opens the chronicle at path for appending under an exclusive lock
// and returns it with its size. Should the chronicle be rotated away while
// seize waits for the lock, it seizes the new one.
func seize(path string) (*os.File, int64, error) {
	for {
		f, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0o644)
		if err != nil {
			return nil, 0, err
		}
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
			_ = f.Close()
			return nil, 0, err
		}
		held, err := f.Stat()
		if err != nil {
			_ = f.Close()
			return nil, 0, err
		}
		named, err := os.Stat(path)
		if err == nil && os.SameFile(held, named) {
			return f, held.Size(), nil
		}
		_ = f.Close()
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, 0, err
		}
	}
}

// Read returns the entries of the chronicle at path and of its rotated
// predecessor, newest first. Lines that hold no entry are passed over; a
// chronicle not yet written reads as empty.
func Read(path string) ([]Entry, error) {
	var entries []Entry
	for _, p := range []string{Rotated(path), path} {
		data, err := os.ReadFile(p)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, lament(p, err)
		}
		for line := range bytes.Lines(data) {
			var e Entry
			if json.Unmarshal(line, &e) == nil {
				entries = append(entries, e)
			}
		}
	}
	slices.Reverse(entries)
	return entries, nil
}

// lament speaks the grimdark words for what the machine answered when the
// chronicle at path was touched.
func lament(path string, err error) error {
	switch {
	case errors.Is(err, fs.ErrPermission):
		return fmt.Errorf("the machine spirit denies the servitor the chronicle at %s", path)
	case errors.Is(err, fs.ErrNotExist), errors.Is(err, syscall.ENOTDIR):
		return fmt.Errorf("no hall can be raised to keep the chronicle at %s", path)
	case errors.Is(err, syscall.EISDIR):
		return fmt.Errorf("a hall stands where the chronicle at %s should be", path)
	}
	return fmt.Errorf("the machine resists every attempt to keep the chronicle at %s", path)
}
