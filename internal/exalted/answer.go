package exalted

import (
	"encoding/json/v2"
	"fmt"
)

// Installed is where the exalted servitor is installed, the only servitor
// ever roused in its hidden mode.
const Installed = "/usr/bin/servitor"

// Mismatch is the lament of an installed exalted servitor that speaks
// another protocol than the servitor that roused it: a servitor forged anew
// drives only the servitor installed from the same forging. It is a refusal
// before anything was touched.
type Mismatch struct {
	Installed int // the protocol the installed servitor speaks; 0 when it names none
}

func (m *Mismatch) Error() string {
	spoken := fmt.Sprintf("speaks the protocol %d", m.Installed)
	if m.Installed == 0 {
		spoken = "names no protocol it speaks"
	}
	return fmt.Sprintf("the exalted servitor installed at %s %s, yet this servitor speaks the protocol %d: they "+
		"are not of one forging — install the servitor anew from its package (makepkg -si), so that both are",
		Installed, spoken, Protocol)
}

// Answer hears, as the invoker's servitor, the exalted servitor's answer to
// the handshake. Every forging of the exalted servitor names its protocol in
// that answer, accepting or refusing, so the protocol alone is heard first,
// whatever else the line holds: an answer of another protocol, or naming
// none, is *Mismatch; only an answer of this protocol is heard strictly. An
// answer that refuses is no lament: its Verdict and Lament tell it. The end
// of the channel before any answer is io.EOF: the intercessor refused, or
// the servitor installed knows no exalted servitor at all.
func (c *Channel) Answer() (Reply, error) {
	line, err := c.line()
	if err != nil {
		return Reply{}, err
	}
	var named struct {
		Protocol int `json:"protocol"`
	}
	if err := json.Unmarshal(line, &named); err != nil {
		return Reply{}, fmt.Errorf("the exalted servitor installed at %s answered no true word of any protocol: %v",
			Installed, err)
	}
	if named.Protocol != Protocol {
		return Reply{}, &Mismatch{Installed: named.Protocol}
	}
	var r Reply
	return r, Decode(line, &r)
}
