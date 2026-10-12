package exalted

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/hallowed"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanction"
	sanctums "github.com/nerdwave-nick/servitor/internal/sanctum"
)

// act performs the exalted verse v from the sanction, telling its verdict
// and, when it changed the machine, the deed by which it is reverted. A
// fallen verse that changed the machine all the same comes with its deed.
func (s *session) act(ctx context.Context, v sanction.Verse) (Reply, *deed) {
	var (
		r   Reply
		mem *hallowed.Memory
		err error
	)
	switch v.Kind {
	case librarium.KindSanctum:
		r, mem, err = s.sanctum(v)
	case librarium.KindTranscription:
		r, mem, err = s.transcription(v)
	case librarium.KindTether:
		r, mem, err = s.tether(v)
	case librarium.KindIncantation, librarium.KindLitany:
		return s.speak(ctx, v)
	default:
		err = fmt.Errorf("a %s is never exalted", v.Kind.Key())
	}
	r.To, r.Verse, r.Verdict = "perform", v.Verse, Triumph
	var d *deed
	switch {
	case mem != nil && mem.Changed():
		r.Changed = true
		d = &deed{verse: v, undo: func(context.Context) (string, error) { return "", mem.Revert() }}
	case err == nil:
		d = &deed{verse: v}
	}
	if err != nil {
		r.Verdict, r.Lament = Fell, err.Error()
	}
	return r, d
}

// speak performs exalted speech through the voice of the session.
func (s *session) speak(ctx context.Context, v sanction.Verse) (Reply, *deed) {
	r := Reply{To: "perform", Verse: v.Verse, Verdict: Triumph}
	heard, undo, err := s.speaker.Speak(ctx, v)
	r.Heard = heard
	d := &deed{verse: v, undo: undo}
	r.Changed = undo != nil
	if err != nil {
		r.Verdict, r.Lament = Fell, err.Error()
		if undo == nil {
			d = nil
		}
	}
	return r, d
}

// seen is what the servitor tells of scripture.
func seen(scripture []byte) *Seen {
	sum := sha256.Sum256(scripture)
	return &Seen{Size: len(scripture), SHA256: hex.EncodeToString(sum[:])}
}

// read reads the vessel at path through the trusted-path walk; absent is
// true, with no error, when no vessel stands there.
func (s *session) read(path string) (scripture []byte, absent bool, err error) {
	scripture, err = s.cfg.Ground.Read(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, true, nil
	}
	return scripture, false, err
}

// sanctum rewrites the sanctum of the verse for the aspect of the plan,
// consecrating its vessel when the sanction allows. A vessel that already
// holds it is not touched.
func (s *session) sanctum(v sanction.Verse) (Reply, *hallowed.Memory, error) {
	m := sanctums.Marker{Glyph: v.Glyph, ClosingGlyph: v.ClosingGlyph, Ward: v.Ward}
	for _, line := range strings.Split(string(v.Scripture), "\n") {
		if m.IsMarkerLine(line) {
			return Reply{}, nil, fmt.Errorf("the scripture of verse %d would carry a marker of its own sanctum %q "+
				"into %s", v.Verse, m.Ward, v.Target)
		}
	}
	before, absent, err := s.read(v.Target)
	switch {
	case err != nil:
		return Reply{}, nil, err
	case absent && !v.Consecrate:
		return Reply{}, nil, fmt.Errorf("no vessel stands at %s, and its sanction does not consecrate it", v.Target)
	}
	after, err := sanctums.Upsert(string(before), m, s.plan.Aspect, string(v.Scripture))
	if err != nil {
		return Reply{}, nil, fmt.Errorf("the vessel %s: %w", v.Target, err)
	}
	r := Reply{After: seen([]byte(after))}
	if !absent {
		r.Before = seen(before)
		if after == string(before) {
			return r, nil, nil
		}
	}
	mem, err := s.cfg.Ground.Transcribe(v.Target, []byte(after), nil)
	return r, mem, err
}

// transcription transcribes the scripture of the verse, or strikes its
// vessel. A standing vessel gives way only when it holds the scripture of
// an aspect of the sanction, or with zeal, and takes the seal of the verse
// as every transcription does; one that already holds the scripture under
// that seal is not touched.
func (s *session) transcription(v sanction.Verse) (Reply, *hallowed.Memory, error) {
	before, absent, err := s.read(v.Target)
	if err != nil {
		return Reply{}, nil, err
	}
	var r Reply
	if !absent {
		r.Before = seen(before)
		if !v.Zeal && !s.known(v.Verse, before) {
			verb := map[bool]string{true: "strike", false: "replace"}[v.Null]
			return r, nil, fmt.Errorf("the vessel %s holds scripture of no aspect of this sanction; the servitor "+
				"will %s it only with \"zeal\": true", v.Target, verb)
		}
	}
	if v.Null {
		if absent {
			return r, nil, nil
		}
		mem, err := s.cfg.Ground.Strike(v.Target)
		return r, mem, err
	}
	r.After = seen(v.Scripture)
	mem, err := s.cfg.Ground.Transcribe(v.Target, v.Scripture, v.Seal) // changes nothing that stands as decreed
	return r, mem, err
}

// known reports whether scripture is that of verse n for some aspect of the
// sanction; one final newline is forgiven.
func (s *session) known(n int, scripture []byte) bool {
	got := strings.TrimSuffix(string(scripture), "\n")
	for _, p := range s.sn.Aspects {
		for _, v := range p.Liturgy {
			if v.Verse == n && !v.Null && strings.TrimSuffix(string(v.Scripture), "\n") == got {
				return true
			}
		}
	}
	return false
}

// tether binds the name of the verse to its anchor, or unbinds it.
func (s *session) tether(v sanction.Verse) (Reply, *hallowed.Memory, error) {
	var (
		mem *hallowed.Memory
		err error
	)
	if v.Null {
		mem, err = s.cfg.Ground.Unbind(v.Target, v.Zeal)
	} else {
		mem, err = s.cfg.Ground.Bind(v.Target, v.Anchor, v.Zeal)
	}
	return Reply{}, mem, err
}
