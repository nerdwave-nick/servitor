package exalted

import (
	"bufio"
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
)

// Protocol is the version of the words spoken over the channel. The
// invoker's servitor and the exalted servitor must speak the same: a
// servitor built anew drives only the servitor installed from the same
// forging. Protocol 2 names the subjects and observances of state rites of
// pattern Mark II; protocol 3 reads their privileged subjects (Request.Read)
// and tells the omens that proved them.
const Protocol = 3

// LineCap is the most a line of the channel may hold, its newline aside.
const LineCap = 1 << 20

// errOverlong is a line of the channel longer than LineCap.
var errOverlong = fmt.Errorf("a line of the channel exceeds %d bytes", LineCap)

// Channel speaks strict JSON lines over a channel: one value per line, at
// most LineCap bytes, every member known. The exalted servitor hears its
// invoker through one, and the invoker's servitor may hear the exalted
// servitor through another.
type Channel struct {
	r *bufio.Reader
	w io.Writer
}

// NewChannel speaks over r and w; either may be nil when the channel only
// hears or only speaks.
func NewChannel(r io.Reader, w io.Writer) *Channel {
	c := &Channel{w: w}
	if r != nil {
		c.r = bufio.NewReaderSize(r, LineCap+1)
	}
	return c
}

// Send speaks v as one line.
func (c *Channel) Send(v any) error {
	data, err := json.Marshal(v, json.Deterministic(true))
	if err != nil {
		return err
	}
	if len(data) > LineCap {
		return errOverlong
	}
	_, err = c.w.Write(append(data, '\n'))
	return err
}

// Receive hears the next line into v, strictly. The end of the channel
// before a line begins is io.EOF; within a line, io.ErrUnexpectedEOF.
func (c *Channel) Receive(v any) error {
	line, err := c.line()
	if err != nil {
		return err
	}
	return Decode(line, v)
}

// line hears the next line, its newline cut away.
func (c *Channel) line() ([]byte, error) {
	line, err := c.r.ReadSlice('\n')
	switch {
	case errors.Is(err, bufio.ErrBufferFull):
		return nil, errOverlong
	case errors.Is(err, io.EOF) && len(line) > 0:
		return nil, io.ErrUnexpectedEOF
	case err != nil:
		return nil, err
	}
	return bytes.Clone(line[:len(line)-1]), nil
}

// Decode reads one line of the channel into v: strict JSON, every member
// known, nothing after it.
func Decode(line []byte, v any) error {
	if err := json.Unmarshal(line, v, json.RejectUnknownMembers(true)); err != nil {
		return fmt.Errorf("a line of the channel is no true word of the protocol: %v", err)
	}
	return nil
}
