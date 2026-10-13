package exalted

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

// TestChannel_HearsStrictLinesOfBoundedLength: a line is one strict value;
// unknown members, trailing words, overlong lines and a line cut short by
// the end of the channel are refused.
func TestChannel_HearsStrictLinesOfBoundedLength(t *testing.T) {
	var spoken bytes.Buffer
	if err := NewChannel(nil, &spoken).Send(Request{Perform: numbered(3)}); err != nil {
		t.Fatal(err)
	}
	if spoken.String() != `{"perform":3}`+"\n" {
		t.Fatalf("spoke %q", spoken.String())
	}
	var r Request
	if err := NewChannel(&spoken, nil).Receive(&r); err != nil || r.Perform == nil || r.Perform.Verse != 3 {
		t.Fatalf("heard %+v, %v", r, err)
	}

	long := `{"former":"` + strings.Repeat("x", LineCap) + `"}`
	for what, line := range map[string]string{
		"an unknown member": `{"perform":3,"blessing":true}` + "\n",
		"trailing words":    `{"perform":3} {"abort":true}` + "\n",
		"no value":          "\n",
		"an overlong line":  long + "\n",
		"a line cut short":  `{"perform":3}`,
		"a doubled member":  `{"perform":3,"perform":4}` + "\n",
	} {
		var r Request
		if err := NewChannel(strings.NewReader(line), nil).Receive(&r); err == nil {
			t.Errorf("%s was heard: %+v", what, r)
		}
	}
	var r2 Request
	if err := NewChannel(strings.NewReader(""), nil).Receive(&r2); !errors.Is(err, io.EOF) {
		t.Errorf("the end of the channel was %v", err)
	}
	fits := `{"former":"` + strings.Repeat("x", LineCap-len(`{"former":""}`)) + `"}` + "\n"
	var h Handshake
	if err := NewChannel(strings.NewReader(fits), nil).Receive(&h); err != nil {
		t.Errorf("a line of exactly %d bytes was refused: %v", LineCap, err)
	}
	if err := NewChannel(nil, io.Discard).Send(Handshake{Former: strings.Repeat("x", LineCap)}); err == nil {
		t.Error("an overlong line was spoken")
	}
}

// TestChannel_NamesTheItemsOfAStateRite: a verse is named by its number, a
// subject or observance of a state rite by an object of its name and the
// omen of restoration, read strictly.
func TestChannel_NamesTheItemsOfAStateRite(t *testing.T) {
	var spoken bytes.Buffer
	ch := NewChannel(&spoken, &spoken)
	for _, c := range []struct {
		r    Request
		line string
	}{
		{Request{Perform: &Item{Subject: "smt", Omen: "on"}}, `{"perform":{"subject":"smt","omen":"on"}}`},
		{Request{Revert: &Item{Observance: "herald"}}, `{"revert":{"observance":"herald"}}`},
		{Request{Revert: &Item{Verse: 2}}, `{"revert":2}`},
	} {
		if err := ch.Send(c.r); err != nil || strings.TrimSpace(spoken.String()) != c.line {
			t.Fatalf("spoke %q, %v; want %q", spoken.String(), err, c.line)
		}
		var r Request
		if err := ch.Receive(&r); err != nil || !reflect.DeepEqual(r, c.r) {
			t.Fatalf("heard %+v, %v; want %+v", r, err, c.r)
		}
	}
	for _, line := range []string{`{"perform":{"subject":"smt","blessing":true}}`, `{"perform":{}}`,
		`{"perform":{"omen":"on"}}`, `{"perform":"smt"}`} {
		var r Request
		if err := NewChannel(strings.NewReader(line+"\n"), nil).Receive(&r); err == nil {
			t.Errorf("%s was heard as %+v", line, r)
		}
	}
}
