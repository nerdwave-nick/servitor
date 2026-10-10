package invocation

import (
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// Illuminate renders scripture sc of rite r with v as an invocation would
// write it: inline scripture illuminated, a tome read afresh from its place
// and illuminated only when it says so. ok is false for scripture that
// cannot be rendered or read, and for a null scripture.
func Illuminate(r *librarium.Rite, sc librarium.Scripture, v placeholder.Values) (text string, ok bool) {
	if sc.Null {
		return "", false
	}
	return renderQuietly(r, sc, v)
}
