package rituals

import (
	"testing"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanctum"
)

// TestInquire_NamesTheTaintedSubject: a sanctum subject other hands altered
// is denounced by its name, where the scripture writes it, never by a verse
// a state rite does not have.
func TestInquire_NamesTheTaintedSubject(t *testing.T) {
	fx := newFixture(t)
	fx.vessel("niri.kdl", sanctum.Render(sanctum.Marker{Glyph: "#", Ward: "gaming"}, "", "on", "altered"))
	fx.rite("gaming", `{"pattern": "Mark II", "aspects": ["on", "off"], "subjects": [
	  {"name": "theme", "designation": "tether", "tether": "$DATA/current", "omens": {"*": "$DATA/{{aspect}}"}},
	  {"name": "niri", "designation": "sanctum", "vessel": "$DATA/niri.kdl", "ward": "gaming", "glyph": "#",
	   "omens": {"on": "hide", "off": "show"}}]}`)

	fs := inquire(t, fx.servitor(), false)

	tainted := denounced(fs, "gaming", librarium.Impurity, "tainted", `the sanctum subject "niri"`, fx.data+"/niri.kdl")
	if len(tainted) != 1 || tainted[0].Line != 3 {
		t.Errorf("the tainted subject: %v\nin\n%v", tainted, fs)
	}
	if len(denounced(fs, "gaming", librarium.Impurity, "verse")) != 0 {
		t.Errorf("a state rite was denounced by verse:\n%v", fs)
	}
}
