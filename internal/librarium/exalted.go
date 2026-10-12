package librarium

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tailscale/hujson"

	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// IsExalted reports whether the step is exalted: performed not with its
// invoker's authority but with the machine's own, from a sanction.
func IsExalted(s Step) bool {
	switch s := s.(type) {
	case *Sanctum:
		return s.Exalted
	case *Transcription:
		return s.Exalted
	case *Tether:
		return s.Exalted
	case *Incantation:
		return s.Exalted
	case *Litany:
		return s.Exalted
	}
	return false
}

// ExaltedVerses returns the verses of the rite's exalted steps, counted
// from one as written; nil when it holds none.
func (r *Rite) ExaltedVerses() []int {
	var verses []int
	for i, s := range r.Liturgy {
		if IsExalted(s) {
			verses = append(verses, r.verse(i))
		}
	}
	return verses
}

// Exalted reports whether the rite holds an exalted verse.
func (r *Rite) Exalted() bool { return len(r.ExaltedVerses()) > 0 }

// ExaltedScope is the placeholder scope of the words of the rite's exalted
// verses, fixed when its master sanctions them: {{former}} is heresy there,
// and so is every mandatory inscription that some aspect (or, for an edict,
// the edict itself) does not decree.
func (r *Rite) ExaltedScope() placeholder.Scope {
	sc := r.Scope()
	sc.Exalted = true
	aspects := r.Aspects
	if r.Edict {
		aspects = []string{""}
	}
	for _, in := range r.Inscriptions {
		if in.Mandatory && len(in.Decrees.unserved(aspects)) > 0 {
			sc.Fleeting = append(sc.Fleeting, in.Key)
		}
	}
	return sc
}

// ScopeOf is the placeholder scope of the words of the step s of the rite.
func (r *Rite) ScopeOf(s Step) placeholder.Scope {
	if IsExalted(s) {
		return r.ExaltedScope()
	}
	return r.Scope()
}

// exaltation decodes the step's "exalted"; a vox-cast is never exalted, and
// a verse is exalted for every aspect or for none.
func (s stepDecoder) exaltation() bool {
	m, ok := s.fs.get("exalted")
	switch {
	case !ok:
		return false
	case s.kind == KindVoxCast:
		s.d.heresy(m.key, "a vox-cast is never exalted: it only sends word of the invocation, which needs no "+
			"authority but its invoker's — strike \"exalted\" from verse %d", s.verse)
		return false
	}
	if _, isMap := m.val.Value.(*hujson.Object); isMap {
		s.d.heresy(m.val, "%s is written as an aspect map, yet a verse is exalted for every aspect or for none "+
			"— write true or false", s.field("exalted"))
		return false
	}
	return s.d.boolean(m.val, s.field("exalted"))
}

// relics decodes the relics of an incantation or litany: paths within the
// rite's hall, borne only by an exalted step.
func (s stepDecoder) relics() []string {
	m, ok := s.fs.get("relics")
	if !ok {
		return nil
	}
	if !s.exalted {
		s.d.heresy(m.key, "%s names \"relics\", yet it is not exalted: relics are laid beside exalted speech "+
			"alone, in the hall of the exalted servitor — strike them, or write \"exalted\": true", s.what)
	}
	arr, ok := m.val.Value.(*hujson.Array)
	if !ok {
		s.d.wrongForm(m.val, s.field("relics"), "a list of paths within the hall of the rite")
		return nil
	}
	relics := []string{}
	for i := range arr.Elements {
		e := &arr.Elements[i]
		what := fmt.Sprintf("relic %d of %s", i+1, s.what)
		p, ok := s.d.word(e, what)
		if !ok {
			continue
		}
		if !withinHall(p) {
			s.d.heresy(e, "%s, %q, lies outside the hall of the rite: a relic is named by its place beside the "+
				"rite's scripture, within its hall, and the exalted servitor lays it under that same name in a "+
				"hall of its own — lay it there and name it so", what, p)
		}
		relics = append(relics, p)
	}
	return relics
}

// withinHall reports whether the relative path p names a place within the
// hall it is reckoned from, and not the hall itself.
func withinHall(p string) bool {
	c := filepath.Clean(p)
	return !filepath.IsAbs(c) && c != "." && c != ".." && !strings.HasPrefix(c, "../")
}

// realms are the halls of the kernel's own knobs, which hold no vessel.
var realms = []string{"/proc", "/sys", "/dev"}

// rooted denounces an exalted vessel or bound name that, rendered for some
// aspect, does not begin at the root or lies within one of the realms; own
// is where it is written.
func (s stepDecoder) rooted(step Step, own *hujson.Value) {
	var path, noun string
	switch st := step.(type) {
	case *Sanctum:
		path, noun = st.Vessel, "vessel"
	case *Transcription:
		path, noun = st.Vessel, "vessel"
	case *Tether:
		path, noun = st.Name, "bound name"
	default:
		return
	}
	r := s.d.rite
	for _, aspect := range s.d.aspects {
		inscriptions := map[string]string{}
		for _, in := range r.Inscriptions {
			inscriptions[in.Key], _ = in.Decrees.For(aspect)
		}
		out, err := s.d.scope.Render(path, placeholder.Values{Aspect: aspect, Rite: r.Name, Inscriptions: inscriptions})
		if err != nil {
			return // denounced already
		}
		if !filepath.IsAbs(out) {
			s.d.heresy(own, "the %s %s of the exalted %s of verse %d does not begin at the root: the exalted "+
				"servitor reckons no path from the invoker's home or hall, and finds its vessels from \"/\" alone "+
				"— write the whole path", noun, out, s.kind.Key(), s.verse)
			return
		}
		for _, realm := range realms {
			if c := filepath.Clean(out); c == realm || strings.HasPrefix(c, realm+"/") {
				s.d.heresy(own, "%s lies beneath %s, where no vessel is kept: the knobs of the machine beneath "+
					"/proc, /sys and /dev cannot be written anew and restored as vessels are — write it by an "+
					"exalted incantation, such as \"echo 1 > %s\"", out, realm, c)
				return
			}
		}
	}
}

// judgeSpeech notes the impurities of the words every incantation and
// litany of the rite speaks: its commands and reversions.
func (d *decoder) judgeSpeech() {
	r := d.rite
	amid := r.Exalted()
	for i, step := range r.Liturgy {
		var u Utterance
		var words []AspectMap[string]
		var keys []string
		switch st := step.(type) {
		case *Incantation:
			u, words, keys = st.Utterance, []AspectMap[string]{st.Command}, []string{"incantation"}
		case *Litany:
			u = st.Utterance
		default:
			continue
		}
		words, keys = append(words, u.Reversion), append(keys, "reversion")
		sp := Speech{Verse: r.verse(i), Exalted: u.Exalted, Relics: u.Relics, AmidExalted: amid}
		for j, m := range words {
			for _, e := range m.Entries() {
				ptr := r.stepPointer(i) + "/" + keys[j]
				if !m.IsUniform() {
					ptr += "/" + e.Aspect
				}
				for _, msg := range sp.Impurities(e.Value) {
					d.findings = append(d.findings, Finding{Severity: Impurity, Scripture: d.src.path, Rite: r.Name,
						Position: d.src.locate(ptr), Message: msg})
				}
			}
		}
	}
}

// exaltedTongues are the tongues exalted speech may be spoken in, by the
// last name of the program each truly is, and the words that bid each read
// no scripture of its own as it awakens: a non-interactive sh or dash reads
// none, and ENV never reaches it.
var exaltedTongues = map[string][]string{
	"bash": {"--noprofile", "--norc"},
	"zsh":  {"-f"},
	"fish": {"--no-config"},
	"sh":   {},
	"dash": {},
}

// ExaltedTongueNames names the tongues exalted speech may be spoken in.
const ExaltedTongueNames = "bash, zsh, fish, sh or dash"

// ExaltedTongue tells how program, the path of a tongue with its bonds
// followed, is bidden to read no scripture of its own as it awakens; known
// is false for a program that is none of the tongues exalted speech knows,
// judged by its last name.
func ExaltedTongue(program string) (bidding []string, known bool) {
	bidding, known = exaltedTongues[filepath.Base(program)]
	return append([]string{}, bidding...), known
}
