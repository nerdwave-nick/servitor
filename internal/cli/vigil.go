package cli

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/nerdwave-nick/servitor/internal/rituals"
)

// The heralds of a vigil's lines.
const (
	heraldRite      = "rite"
	heraldLament    = "lament"
	heraldConcluded = "concluded"
)

// riteLine is a vigil's line telling a rite whole: its census entry, flat,
// beneath its herald.
type riteLine struct {
	Herald string `json:"herald"`
	rituals.Entry
}

// lamentLine is a vigil's line telling a lament, and the rite it concerns.
type lamentLine struct {
	Herald string `json:"herald"`
	Lament string `json:"lament"`
	Rite   string `json:"rite,omitempty"`
}

// concludedLine is a vigil's last line.
type concludedLine struct {
	Herald string `json:"herald"`
}

// keepVigil keeps the census of s as a vigil upon cmd's stdout, one
// binharic line at a time, and concludes it. Should the reader of stdout
// depart, the vigil ends at once, slaying the auspexes still awake, and
// the servitor departs without a word.
func keepVigil(cmd *cobra.Command, s *rituals.Servitor, spareAuspex bool) error {
	ctx, end := context.WithCancel(cmd.Context())
	defer end()
	sigpipe := make(chan os.Signal, 1) // never read: a write without a reader fails instead
	signal.Notify(sigpipe, syscall.SIGPIPE)
	defer signal.Stop(sigpipe)
	h := &crier{w: cmd.OutOrStdout(), end: end}
	defer watchReader(h.w, end)()

	s.Vigil(ctx, spareAuspex, func(word rituals.Heralding) {
		if word.Lament != nil {
			h.cry(lamentLine{Herald: heraldLament, Lament: word.Lament.Error(), Rite: word.Rite})
			return
		}
		h.cry(riteLine{Herald: heraldRite, Entry: *word.Entry})
	})
	if ctx.Err() == nil {
		h.cry(concludedLine{Herald: heraldConcluded})
	}
	switch {
	case h.err != nil && !errors.Is(h.err, syscall.EPIPE):
		return h.err
	case h.err != nil || ctx.Err() != nil:
		return &ExitError{Code: 1} // the reader departed: nothing more to say
	}
	return nil
}

// crier writes a vigil's lines until a write fails, then ends the vigil.
type crier struct {
	w   io.Writer
	end context.CancelFunc
	err error // the first failed write
}

// cry writes line as one binharic line in a single write.
func (c *crier) cry(line any) {
	if c.err != nil {
		return
	}
	b, err := json.Marshal(line, json.Deterministic(true))
	if err == nil {
		_, err = c.w.Write(append(b, '\n'))
	}
	if err != nil {
		c.err = err
		c.end()
	}
}
