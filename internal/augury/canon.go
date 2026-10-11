package augury

import "github.com/nerdwave-nick/servitor/internal/librarium"

// Declaration is one inscription of a rite's canon, as the augury and the
// census speak it: every field is always present.
type Declaration struct {
	Key       string `json:"key"`
	Purpose   string `json:"purpose"` // "" when unwritten
	Mandatory bool   `json:"mandatory"`
	// Decrees are the decree of every aspect that has one, "*" resolved
	// for each aspect it serves; never nil. An edict's are empty.
	Decrees map[string]string `json:"decrees"`
	// Decree is an edict's one plain decree; nil for a rite, and for an
	// edict that decrees nothing.
	Decree *string `json:"decree"`
}

// CanonOf is the canon of the rite r: its inscriptions as declared, in
// written order; never nil.
func CanonOf(r *librarium.Rite) []Declaration {
	canon := make([]Declaration, 0, len(r.Inscriptions))
	for _, in := range r.Inscriptions {
		d := Declaration{Key: in.Key, Purpose: in.Purpose, Mandatory: in.Mandatory, Decrees: map[string]string{}}
		if r.Edict {
			// an edict's decree is one plain word, read as its proclaiming
			// reads it: for no aspect at all
			if word, ok := in.Decrees.For(""); ok {
				d.Decree = &word
			}
		} else {
			for _, aspect := range r.Aspects {
				if word, ok := in.Decrees.For(aspect); ok {
					d.Decrees[aspect] = word
				}
			}
		}
		canon = append(canon, d)
	}
	return canon
}
