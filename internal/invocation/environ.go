package invocation

import (
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// Environ is the environment every command of r is spoken in — its
// incantations, litanies and its auspex: the servitor's own with
// SERVITOR_ASPECT, SERVITOR_FORMER_ASPECT, SERVITOR_RITE and every declared
// SERVITOR_INSCRIPTION_<KEY> taken from v, and PWD the rite's directory.
func Environ(r *librarium.Rite, v placeholder.Values) []string {
	return environ(r.Dir(), v, r.InscriptionKeys())
}
