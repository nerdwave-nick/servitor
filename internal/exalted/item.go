package exalted

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"

	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// Item names what a request performs or reverts: a verse of a liturgy by
// its number, written as that number, or a subject or observance of a
// state rite of pattern Mark II by its name, written as
// {"subject": "smt", "omen": "on"} or {"observance": "herald"}. A
// machine-spirit also names a declared omen it is restored to: when it is
// set, the omen the exalted servitor restores by itself should the session
// fall ("" when none is known, and it cannot be restored); when it is
// reverted, the omen to restore now ("" for the omen named when it was
// set).
type Item struct {
	Verse               int
	Subject, Observance string
	Omen                string
}

// itemWriting is an item of a state rite as it is written.
type itemWriting struct {
	Subject    string `json:"subject,omitzero"`
	Observance string `json:"observance,omitzero"`
	Omen       string `json:"omen,omitzero"`
}

// MarshalJSON writes a verse as its number, a subject or observance as an
// object.
func (it Item) MarshalJSON() ([]byte, error) {
	if it.Subject == "" && it.Observance == "" {
		return json.Marshal(it.Verse)
	}
	return json.Marshal(itemWriting{it.Subject, it.Observance, it.Omen}, json.Deterministic(true))
}

// UnmarshalJSON reads a number as a verse, an object strictly as a subject
// or observance.
func (it *Item) UnmarshalJSON(data []byte) error {
	*it = Item{}
	if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) {
		return json.Unmarshal(data, &it.Verse)
	}
	var w itemWriting
	if err := json.Unmarshal(data, &w, json.RejectUnknownMembers(true)); err != nil {
		return err
	}
	if w.Subject == "" && w.Observance == "" {
		return errors.New("an item names a subject or an observance")
	}
	*it = Item{Subject: w.Subject, Observance: w.Observance, Omen: w.Omen}
	return nil
}

// Named names the item as laments name it.
func (it Item) Named() string {
	return sanction.Verse{Verse: it.Verse, Subject: it.Subject, Observance: it.Observance}.Named()
}

// names tells whether the item names v.
func (it Item) names(v sanction.Verse) bool {
	return it.Verse == v.Verse && it.Subject == v.Subject && it.Observance == v.Observance
}

// unnamed is the lament for an item a plan does not hold.
func unnamed(it Item) error { return fmt.Errorf("%s is no exalted item of the plan", it.Named()) }
