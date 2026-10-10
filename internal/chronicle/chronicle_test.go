package chronicle

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// entry is a triumphant invocation of rite with a reason of size bytes.
func entry(rite string, size int) Entry {
	return New(moment, rite, invocation.Options{
		Aspect:       "porpl",
		Former:       "default",
		Inscriptions: map[string]string{"reason": strings.Repeat("x", size)},
	}, invocation.Outcome{Verdict: invocation.Triumph})
}

// lineOf is e as Append writes it.
func lineOf(t *testing.T, e Entry) []byte {
	t.Helper()
	b, err := json.Marshal(e, json.Deterministic(true))
	if err != nil {
		t.Fatal(err)
	}
	return append(b, '\n')
}

func mustAppend(t *testing.T, path string, e Entry) {
	t.Helper()
	if err := Append(path, e); err != nil {
		t.Fatal(err)
	}
}

// rites names the rites of entries in order.
func rites(entries []Entry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Rite
	}
	return names
}

func mustRead(t *testing.T, path string) []Entry {
	t.Helper()
	entries, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func size(t *testing.T, path string) int64 {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Size()
}

func TestAppend_KeepsTheChronicleInTheStateHome(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	t.Setenv(librarium.EnvChronicle, "")
	path := (*librarium.Settings)(nil).Resolve(librarium.Runes{}, os.LookupEnv).Chronicle

	first, second := entry("theme", 3), entry("mouse", 3)
	mustAppend(t, path, first)
	mustAppend(t, path, second)

	want := filepath.Join(state, "servitor", "chronicle.jsonl")
	got, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("no chronicle at %s: %v", want, err)
	}
	if lines := append(lineOf(t, first), lineOf(t, second)...); !bytes.Equal(got, lines) {
		t.Fatalf("chronicle\n%s\nwant\n%s", got, lines)
	}
	if got := rites(mustRead(t, path)); strings.Join(got, " ") != "mouse theme" {
		t.Fatalf("read %v, want newest first", got)
	}
}

func TestAppend_RotatesOnceTheChronicleWouldOutgrowItsLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chronicle.jsonl")
	e := entry("theme", 10)
	line := lineOf(t, e)
	filler := bytes.Repeat([]byte("x"), Limit-len(line)-1)
	if err := os.WriteFile(path, append(filler, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	mustAppend(t, path, e)
	if got := size(t, path); got != Limit {
		t.Fatalf("a line that fits rotated the chronicle: size %d", got)
	}
	if _, err := os.Stat(path + ".1"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("rotated before the limit: %v", err)
	}

	mustAppend(t, path, entry("mouse", 10))
	if got := size(t, path+".1"); got != Limit {
		t.Fatalf("rotated chronicle holds %d bytes, want %d", got, Limit)
	}
	if got := size(t, path); got != int64(len(lineOf(t, entry("mouse", 10)))) {
		t.Fatalf("current chronicle holds %d bytes, want one line", got)
	}
	if got := rites(mustRead(t, path)); strings.Join(got, " ") != "mouse theme" {
		t.Fatalf("read %v, want both files newest first", got)
	}
}

func TestAppend_ForgetsWhatOutlivesTheRotatedChronicle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chronicle.jsonl")
	for i := range 30 {
		mustAppend(t, path, entry(fmt.Sprintf("rite-%02d", i), 100<<10))
	}
	names, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil || len(names) != 2 {
		t.Fatalf("files %v (%v), want the chronicle and one rotated file", names, err)
	}
	for _, n := range names {
		if s := size(t, n); s > Limit {
			t.Errorf("%s holds %d bytes, beyond the limit", n, s)
		}
	}
	got := mustRead(t, path)
	if len(got) < 10 || len(got) >= 30 {
		t.Fatalf("read %d entries, want more than one file's worth and fewer than all", len(got))
	}
	for i, e := range got {
		if want := fmt.Sprintf("rite-%02d", 29-i); e.Rite != want {
			t.Fatalf("entry %d is %s, want %s: %v", i, e.Rite, want, rites(got))
		}
	}
}

func TestAppend_MendsATornLastLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chronicle.jsonl")
	torn := lineOf(t, entry("theme", 3))
	if err := os.WriteFile(path, torn[:len(torn)/2], 0o644); err != nil {
		t.Fatal(err)
	}
	mustAppend(t, path, entry("mouse", 3))
	if got := rites(mustRead(t, path)); strings.Join(got, " ") != "mouse" {
		t.Fatalf("read %v, want the new entry despite the torn line", got)
	}
}

func TestAppend_RefusesAPlaceThatCannotHoldIt(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "servitor")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	err := Append(filepath.Join(blocker, "chronicle.jsonl"), entry("theme", 3))
	if err == nil {
		t.Fatal("appended beneath a file")
	}
	if !strings.Contains(err.Error(), "chronicle") || strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("heresy %q must name the chronicle in the liturgy", err)
	}
}

func TestAppend_ConcurrentInvocationsLoseNoLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chronicle.jsonl")
	const workers, each = 8, 40
	var wg sync.WaitGroup
	for w := range workers {
		wg.Go(func() {
			for i := range each {
				if err := Append(path, entry(fmt.Sprintf("w%d-%02d", w, i), 50)); err != nil {
					t.Error(err)
				}
			}
		})
	}
	wg.Wait()
	got := mustRead(t, path)
	if len(got) != workers*each {
		t.Fatalf("read %d entries, want %d", len(got), workers*each)
	}
	seen := map[string]bool{}
	for _, e := range got {
		if seen[e.Rite] {
			t.Fatalf("%s recorded twice", e.Rite)
		}
		seen[e.Rite] = true
	}
}

func TestAppend_ConcurrentRotationsKeepOneFullPredecessor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chronicle.jsonl")
	const workers, each, reason = 8, 25, 20 << 10
	var wg sync.WaitGroup
	for w := range workers {
		wg.Go(func() {
			for i := range each {
				if err := Append(path, entry(fmt.Sprintf("w%d-%02d", w, i), reason)); err != nil {
					t.Error(err)
				}
			}
		})
	}
	wg.Wait()
	for _, p := range []string{path, path + ".1"} {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) > Limit {
			t.Errorf("%s holds %d bytes, beyond the limit", p, len(data))
		}
		for n, l := range bytes.Split(bytes.TrimSuffix(data, []byte("\n")), []byte("\n")) {
			var e Entry
			if err := json.Unmarshal(l, &e); err != nil {
				t.Fatalf("%s line %d is garbled: %v", p, n+1, err)
			}
		}
	}
	perFile := Limit / len(lineOf(t, entry("w0-00", reason)))
	if got := len(mustRead(t, path)); got <= perFile {
		t.Fatalf("read %d entries, want more than the %d of one full file", got, perFile)
	}
}

func TestRead_PassesOverGarbledLinesNewestFirst(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chronicle.jsonl")
	older := append(lineOf(t, entry("first", 3)), "garbled by an unsteady hand\n"...)
	current := bytes.Join([][]byte{
		lineOf(t, entry("second", 3)),
		[]byte("{\"at\":\"2026-10\n"),
		[]byte("\n"),
		[]byte("[1, 2]\n"),
		lineOf(t, entry("third", 3)),
		[]byte(`{"at":"2026-10-10T13:30:00Z","rite":"tor`),
	}, nil)
	if err := os.WriteFile(path+".1", older, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, current, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := rites(mustRead(t, path)); strings.Join(got, " ") != "third second first" {
		t.Fatalf("read %v", got)
	}
}

func TestRead_NothingYetChronicled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chronicle.jsonl")
	if got := mustRead(t, path); len(got) != 0 {
		t.Fatalf("read %v from nothing", got)
	}
	mustAppend(t, path, entry("theme", 3))
	if got := rites(mustRead(t, path)); strings.Join(got, " ") != "theme" {
		t.Fatalf("read %v without a rotated file", got)
	}
	if _, err := Read(dir); err == nil || !strings.Contains(err.Error(), "chronicle") {
		t.Fatalf("a hall read as a chronicle: %v", err)
	}
}
