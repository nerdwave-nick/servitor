package librarium

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/tailscale/hujson"
)

// SchemaKey names the schema a scripture is written against, for the
// faithful's editors; rites and settings accept it, and the servitor heeds
// it not.
const SchemaKey = "$schema"

// Keys of the objects of a Mark I scripture, in the order of the codex.
var (
	RiteKeys        = []string{"pattern", "purpose", "aspects", "inscriptions", "auspex", "tongue", "liturgy", SchemaKey}
	InscriptionKeys = []string{"purpose", "mandatory", "decrees"}
	AuspexKeys      = []string{"rite", "patience"}
	TomeKeys        = []string{"tome", "illuminate"}
	// Aliases maps every alias key to the key it stands for.
	Aliases = map[string]string{"force": "zeal"}
)

// ReservedRunes are runes of the servitor's own rituals; no inscription may
// bear their names.
var ReservedRunes = []string{"foresee", "silence", "librarium", "chronicle", "log", "help", "binharic", "json"}

var (
	nameRe    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	patternRe = regexp.MustCompile(`^Mark (M{0,3}(CM|CD|D?C{0,3})(XC|XL|L?X{0,3})(IX|IV|V?I{0,3}))$`)
)

// ValidName reports whether name may name a rite, an aspect or an
// inscription.
func ValidName(name string) bool { return nameRe.MatchString(name) }

// Parse reads the scripture data of the rite name recorded at path. The rite
// is nil when the scripture cannot be read as pattern Mark I at all;
// otherwise it holds everything that could be read, even when findings
// denounce heresy.
func Parse(name, path string, data []byte) (*Rite, Findings) {
	src := &source{path: path, data: data}
	d := &decoder{src: src, rite: &Rite{Name: name, Path: path, src: src}}
	ast, err := hujson.Parse(data)
	if err != nil {
		d.findings = Findings{{Severity: Heresy, Scripture: path, Rite: name, Position: syntaxPosition(err),
			Message: "the scripture is malformed beyond reading here: its brackets, quotes, colons or commas " +
				"break the holy form of JSONC, and the cogitator cannot parse a rite whose very letters are corrupt"}}
		return nil, d.findings
	}
	src.ast = ast
	r := d.decodeRite(&src.ast)
	d.findings.Sort()
	return r, d.findings
}

func (d *decoder) decodeRite(root *hujson.Value) *Rite {
	obj, ok := root.Value.(*hujson.Object)
	if !ok {
		d.wrongForm(root, "a rite's scripture", "one object of keys between { and }")
		return nil
	}
	if !d.decodePattern(root, obj) {
		return nil
	}
	r := d.rite
	if !ValidName(r.Name) {
		d.heresy(root, "%q is no fit name for a rite: the name of its scripture must begin with a letter "+
			"or digit and continue only with letters, digits, '.', '_' or '-'", r.Name)
	}
	fs, _ := d.object(root, "a rite", RiteKeys)
	if m, ok := fs.get(SchemaKey); ok {
		r.Schema, _ = d.str(m.val, `the "$schema" of a rite`)
	}
	if m, ok := fs.get("purpose"); ok {
		r.Purpose, _ = d.str(m.val, `the "purpose" of a rite`)
	}
	m, ok := fs.get("aspects")
	if !ok {
		d.heresy(root, "the rite declares no \"aspects\"; list every aspect the rite can bring the machine "+
			"into, for a rite without aspects can never be invoked")
	} else {
		d.decodeAspects(m.val)
	}
	if m, ok := fs.get("inscriptions"); ok {
		d.decodeInscriptions(m.val)
	}
	d.scope = r.Scope()
	if m, ok := fs.get("auspex"); ok {
		d.decodeAuspex(m.val)
	}
	if m, ok := fs.get("tongue"); ok {
		r.Tongue, _ = d.word(m.val, `the "tongue" of a rite`)
	}
	if m, ok := fs.get("liturgy"); ok {
		d.decodeLiturgy(m.val)
	}
	return r
}

// decodePattern reports whether the scripture is of pattern Mark I.
func (d *decoder) decodePattern(root *hujson.Value, obj *hujson.Object) bool {
	for i := range obj.Members {
		m := &obj.Members[i]
		if m.Name.Value.(hujson.Literal).String() != "pattern" {
			continue
		}
		lit, ok := literal(&m.Value)
		switch {
		case !ok || lit.Kind() != '"' || !patternRe.MatchString(lit.String()) || lit.String() == "Mark ":
			written := formOf(&m.Value)
			if ok {
				written = string(lit)
			}
			d.heresy(&m.Value, "the \"pattern\" must be \"Mark\" followed by numerals of the old tongue, such "+
				"as \"Mark I\"; %s is no Mark the Adeptus Mechanicus recognises", written)
			return false
		case lit.String() != Pattern:
			d.heresy(&m.Value, "this scripture is written in pattern %q, but this servitor was forged to read "+
				"only %q; it will not attempt a form it does not know", lit.String(), Pattern)
			return false
		}
		return true
	}
	d.heresy(root, "the scripture bears no \"pattern\"; every scripture of the Librarium must declare the "+
		"Mark of its form, and "+
		"this servitor reads only \"pattern\": %q — scripture of no pattern is heresy, and the servitor "+
		"will not guess at its meaning%s", Pattern, elderLament(obj))
	return false
}

// elderKeys are the keys by which a rite of the elder form, which knew no
// pattern, is recognised.
var elderKeys = []string{"states", "files"}

// elderLament names the elder form when obj bears its keys, so that its
// keeper knows why the rite no longer speaks; it is empty otherwise.
func elderLament(obj *hujson.Object) string {
	var found []string
	for i := range obj.Members {
		if name := obj.Members[i].Name.Value.(hujson.Literal).String(); slices.Contains(elderKeys, name) {
			found = append(found, fmt.Sprintf("%q", name))
		}
	}
	if len(found) == 0 {
		return ""
	}
	return fmt.Sprintf("; its %s are the words of the elder scripture that knew no Mark, a form this "+
		"servitor has abjured: not one of its vessels will be read or rewritten, and nothing of it will be "+
		"converted — rewrite the rite by hand in pattern %q, its vessels bound as steps of a \"liturgy\", "+
		"or strike it from the Librarium", strings.Join(found, " and "), Pattern)
}

func (d *decoder) decodeAspects(v *hujson.Value) {
	arr, ok := v.Value.(*hujson.Array)
	if !ok {
		d.wrongForm(v, `the "aspects" of a rite`, "a list of names")
		return
	}
	if len(arr.Elements) == 0 {
		d.heresy(v, "the rite declares no aspects; list every aspect the rite can bring the machine into, "+
			"for a rite without aspects can never be invoked")
	}
	for i := range arr.Elements {
		e := &arr.Elements[i]
		a, ok := d.str(e, "an aspect")
		if !ok {
			continue
		}
		d.rite.Aspects = append(d.rite.Aspects, a)
		switch {
		case !ValidName(a):
			d.heresy(e, "%q is no fit name for an aspect: an aspect's name must begin with a letter or digit "+
				"and continue only with letters, digits, '.', '_' or '-' (the %q is reserved for the aspect "+
				"maps, where it serves every aspect not named)", a, Fallback)
		case slices.Contains(d.aspects, a):
			d.heresy(e, "the aspect %q is declared twice; each aspect must be named once and only once", a)
		default:
			d.aspects = append(d.aspects, a)
		}
	}
}

func (d *decoder) decodeInscriptions(v *hujson.Value) {
	fs, _ := d.object(v, `the "inscriptions" of a rite`, nil)
	for _, m := range fs {
		in := Inscription{Key: m.name}
		what := "the inscription " + quoteAll([]string{m.name})
		switch {
		case !ValidName(m.name):
			d.heresy(m.key, "%q is no fit name for an inscription: it is spoken as the rune --%s and illuminated "+
				"as {{inscription.%s}}, so it must begin with a letter or digit and continue only with "+
				"letters, digits, '.', '_' or '-'", m.name, m.name, m.name)
		case slices.Contains(ReservedRunes, m.name):
			d.heresy(m.key, "the inscription %q would be spoken as the rune --%s, which the servitor already "+
				"bears for its own rituals; choose another name", m.name, m.name)
		}
		decl, _ := d.object(m.val, what, InscriptionKeys)
		if f, ok := decl.get("purpose"); ok {
			in.Purpose, _ = d.str(f.val, `the "purpose" of `+what)
		}
		if f, ok := decl.get("mandatory"); ok {
			in.Mandatory = d.boolean(f.val, `the "mandatory" of `+what)
		}
		if f, ok := decl.get("decrees"); ok {
			in.Decrees = aspectMap(d, f.val, `the "decrees" of `+what, false, nil, func(v *hujson.Value, what string) (string, bool) {
				return d.str(v, what)
			})
		}
		if in.Mandatory {
			d.unDecreed(m.key, in)
		}
		d.rite.Inscriptions = append(d.rite.Inscriptions, in)
	}
}

func (d *decoder) decodeAuspex(v *hujson.Value) {
	const what = "the auspex"
	a := &Auspex{}
	if lit, ok := literal(v); ok && lit.Kind() == '"' {
		a.Rite, _ = d.templated(v, what, false)
		d.rite.Auspex = a
		return
	}
	if _, ok := v.Value.(*hujson.Object); !ok {
		d.wrongForm(v, what, `a command, or {"rite": "<command>", "patience": "2s"}`)
		return
	}
	fs, _ := d.object(v, what, AuspexKeys)
	if m, ok := fs.get("rite"); ok {
		a.Rite, _ = d.templated(m.val, `the "rite" of the auspex`, false)
	} else {
		d.heresy(v, "the long form of the auspex must name its \"rite\", the command that reports the "+
			"aspect: {\"rite\": \"<command>\", \"patience\": \"5s\"}")
	}
	if m, ok := fs.get("patience"); ok {
		a.Patience = d.patience(m.val, `the "patience" of the auspex`)
	}
	d.rite.Auspex = a
}

func (d *decoder) decodeLiturgy(v *hujson.Value) {
	arr, ok := v.Value.(*hujson.Array)
	if !ok {
		d.wrongForm(v, `the "liturgy" of a rite`, "a list of steps")
		return
	}
	var success *hujson.Value
	for i := range arr.Elements {
		e := &arr.Elements[i]
		if success != nil {
			d.impurity(success, "a \"success\" vox-cast proclaims triumph, yet further steps follow it in the "+
				"liturgy; should one of them fail, triumph will have been proclaimed falsely — place it last")
			success = nil
		}
		s := d.decodeStep(e, i+1)
		if s == nil {
			continue
		}
		d.rite.Liturgy = append(d.rite.Liturgy, s)
		d.rite.verses = append(d.rite.verses, i+1)
		if vc, ok := s.(*VoxCast); ok && vc.Tidings == VoxSuccess {
			success = stepValue(e, "vox-cast")
		}
	}
}

// stepValue returns the value of the step's own key.
func stepValue(step *hujson.Value, key string) *hujson.Value {
	for i, m := range step.Value.(*hujson.Object).Members {
		if m.Name.Value.(hujson.Literal).String() == key {
			return &step.Value.(*hujson.Object).Members[i].Value
		}
	}
	return step
}
