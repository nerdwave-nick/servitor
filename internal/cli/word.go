package cli

import "github.com/spf13/pflag"

// word is the value of a rune that takes one word, named in the liturgy
// ("hall", "scroll", "aspect") rather than by the type the machine keeps it
// in.
type word struct {
	p    *string
	name string
}

var _ pflag.Value = word{}

// wordRune makes p, starting at value, the word of a rune named name.
func wordRune(p *string, value, name string) word {
	*p = value
	return word{p: p, name: name}
}

func (w word) String() string {
	if w.p == nil {
		return ""
	}
	return *w.p
}

func (w word) Set(s string) error { *w.p = s; return nil }

func (w word) Type() string { return w.name }
