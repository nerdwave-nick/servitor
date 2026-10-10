package invocation

import (
	"fmt"
	"sort"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// ResolveInscriptions resolves the inscriptions of an invocation of r into
// aspect: a rune given (runes holds every --<key> written, "" for one
// written empty) outranks the decree for aspect, and a rune written empty
// clears the decree. Only inscriptions with a value are returned.
//
// The error, of type Heresies, denounces a rune of no declared inscription
// and every mandatory inscription left without value; Prepare refuses the
// latter too, so callers that resolve inscriptions otherwise are held to it
// as well. The inscriptions resolved so far are returned even then.
func ResolveInscriptions(r *librarium.Rite, aspect string, runes map[string]string) (map[string]string, error) {
	var hs Heresies
	undeclared := make([]string, 0, len(runes))
	for key := range runes {
		if _, ok := r.Inscription(key); !ok {
			undeclared = append(undeclared, key)
		}
	}
	sort.Strings(undeclared)
	for _, key := range undeclared {
		hs = append(hs, inscriptionHeresy(r, "/inscriptions", fmt.Sprintf("the rite %q declares no inscription %q, "+
			"so no rune --%s may be given to its invocation; declare it under \"inscriptions\" first", r.Name, key, key)))
	}
	out := map[string]string{}
	for _, in := range r.Inscriptions {
		v, given := runes[in.Key]
		if !given {
			v, _ = in.Decrees.For(aspect)
		}
		if v != "" {
			out[in.Key] = v
		}
	}
	hs = append(hs, unmet(r, aspect, out)...)
	if len(hs) > 0 {
		return out, hs
	}
	return out, nil
}

// unmet denounces every mandatory inscription that inscriptions leave
// without value for an invocation into aspect.
func unmet(r *librarium.Rite, aspect string, inscriptions map[string]string) Heresies {
	var hs Heresies
	for _, in := range r.Inscriptions {
		if in.Mandatory && inscriptions[in.Key] == "" {
			hs = append(hs, inscriptionHeresy(r, "/inscriptions/"+in.Key, fmt.Sprintf("the inscription %q is "+
				"mandatory, yet the invocation into %q bears no word for it: give the rune --%s, or let the rite "+
				"decree it for this aspect", in.Key, aspect, in.Key)))
		}
	}
	return hs
}

// inscriptionHeresy is a heresy of the invocation as a whole, found at the
// JSON pointer ptr of r's scripture.
func inscriptionHeresy(r *librarium.Rite, ptr, message string) Heresy {
	return Heresy{Finding: librarium.Finding{
		Severity: librarium.Heresy, Scripture: r.Path, Rite: r.Name, Position: r.Locate(ptr), Message: message,
	}}
}
