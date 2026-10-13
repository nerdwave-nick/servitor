package exalted

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unicode/utf8"

	"github.com/nerdwave-nick/servitor/internal/invocation"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

type sysStat = syscall.Stat_t

// heardCap is the most exalted speech tells of its words: the last of them.
const heardCap = 64 << 10

// speaker is the exalted speech of one session: its working hall, its
// environment, the programs of its tongues, where the scroll of each litany
// lies, and the sessions its speech summoned, slain when it ends.
type speaker struct {
	hall     string
	env      []string
	programs map[string]string   // tongue → program as it is awakened
	biddings map[string][]string // program of a tongue → how it reads no scripture of its own
	scrolls  map[string]string   // verse or item, named → the name of its scroll in the hall

	mu       sync.Mutex
	sessions []int
}

// Speak speaks the incantation or litany v in the working hall, under its
// patience, in a session of its own; its reversion, when it has one, is
// spoken in the same tongue, hall and environment. The words of a
// machine-spirit (its Target) are spoken so too, hearing its Omen as
// SERVITOR_OMEN; they revert nothing of their own.
func (sp *speaker) Speak(ctx context.Context, v sanction.Verse) (string, Undo, error) {
	tongue, err := sp.spoken(v)
	if err != nil {
		return "", nil, err
	}
	argv := append(slices.Clone(tongue), "-c", v.Target)
	if v.Kind == librarium.KindLitany {
		argv = sp.recital(v, tongue)
	}
	if v.Spirit() {
		heard, err := sp.utterIn(ctx, argv, v, append(slices.Clone(sp.env), sanction.OmenVariable+"="+v.Omen))
		return heard, nil, err
	}
	var undo Undo
	if v.Reversion != "" {
		undo = func(ctx context.Context) (string, error) {
			heard, err := sp.utter(ctx, append(slices.Clone(tongue), "-c", v.Reversion), v)
			if err != nil {
				return heard, fmt.Errorf("the reversion fell: %w", err)
			}
			return heard, nil
		}
	}
	heard, err := sp.utter(ctx, argv, v)
	return heard, undo, err
}

// recital is how the scroll of the litany v is recited: by the interpreter
// its shebang names, else through its tongue; it is offered exactly its
// offerings.
func (sp *speaker) recital(v sanction.Verse, tongue []string) []string {
	scroll := filepath.Join(sp.hall, sp.scrolls[v.Named()])
	argv := append(slices.Clone(tongue), scroll)
	if interpreter, word, ok := shebang(v.Scroll); ok {
		argv = []string{sp.programs[interpreter]}
		if word != "" {
			argv = append(argv, word)
		}
		argv = append(argv, scroll)
	}
	return append(argv, v.Offerings...)
}

// utter speaks argv for the verse v and tells its last words.
func (sp *speaker) utter(ctx context.Context, argv []string, v sanction.Verse) (string, error) {
	return sp.utterIn(ctx, argv, v, sp.env)
}

// utterIn speaks argv for the verse v in the environment env.
func (sp *speaker) utterIn(ctx context.Context, argv []string, v sanction.Verse, env []string) (string, error) {
	heard, err := invocation.Utter(ctx, invocation.Utterance{Argv: argv, Dir: sp.hall, Env: env,
		Patience: v.Patience, Session: true, Summoned: sp.summoned})
	return lastWords(heard), err
}

func (sp *speaker) summoned(session int) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.sessions = append(sp.sessions, session)
}

// lastWords is heard as the channel may carry it: true words, the last
// heardCap bytes of them.
func lastWords(heard string) string {
	heard = strings.ToValidUTF8(heard, string(utf8.RuneError))
	for len(heard) > heardCap {
		_, size := utf8.DecodeRuneInString(heard)
		heard = heard[size:]
	}
	return heard
}

// Close slays every process the speech of the session left behind and
// casts down its working hall.
func (sp *speaker) Close() error {
	sp.mu.Lock()
	sessions := slices.Clone(sp.sessions)
	sp.mu.Unlock()
	slay(sessions)
	if err := os.RemoveAll(sp.hall); err != nil {
		return fmt.Errorf("the working hall %s cannot be cast down: %w", sp.hall, err)
	}
	return nil
}

// slayRounds is how often the machine is searched for what a session left
// behind, lest one summoned while it was slain escape.
const slayRounds = 50

// slay slays every process of the sessions: each process group, then every
// process of the machine that still dwells in one of them, even one led
// into a process group of its own.
func slay(sessions []int) {
	if len(sessions) == 0 {
		return
	}
	for _, s := range sessions {
		_ = syscall.Kill(-s, syscall.SIGKILL)
	}
	for range slayRounds {
		left := dwellers(sessions)
		if len(left) == 0 {
			return
		}
		for _, pid := range left {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}
}

// dwellers are the living processes of the machine that dwell in one of
// the sessions.
func dwellers(sessions []int) []int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var pids []int
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		stat, err := os.ReadFile("/proc/" + e.Name() + "/stat")
		if err != nil {
			continue
		}
		// after the name: state, ppid, pgrp, session
		s := string(stat)
		f := strings.Fields(s[strings.LastIndexByte(s, ')')+1:])
		if len(f) < 4 || f[0] == "Z" || f[0] == "X" {
			continue
		}
		if session, err := strconv.Atoi(f[3]); err == nil && slices.Contains(sessions, session) {
			pids = append(pids, pid)
		}
	}
	return pids
}
