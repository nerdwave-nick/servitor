package librarium

import (
	"strings"

	"github.com/tailscale/hujson"
)

// isTome tells a tome ({"tome": …, "illuminate": …}) from an aspect map.
func isTome(v *hujson.Value) bool {
	for _, m := range v.Value.(*hujson.Object).Members {
		if name := m.Name.Value.(hujson.Literal).String(); name == "tome" || name == "illuminate" {
			return true
		}
	}
	return false
}

// scripture decodes the scripture of a sanctum or (nullable) transcription.
func (s stepDecoder) scripture(v *hujson.Value, nullable bool) AspectMap[Scripture] {
	return aspectMap(s.d, v, s.field("scripture"), true, isTome, func(v *hujson.Value, what string) (Scripture, bool) {
		return s.d.oneScripture(v, what, nullable)
	})
}

func (d *decoder) oneScripture(v *hujson.Value, what string, nullable bool) (Scripture, bool) {
	if lit, ok := literal(v); ok && lit.Kind() == 'n' {
		if !nullable {
			d.heresy(v, "the scripture of a sanctum may not be null: a sanctum is never struck from its "+
				"vessel, for its marker records the aspect — to leave the sanctum empty, write \"\" instead")
			return Scripture{}, false
		}
		return Scripture{Null: true}, true
	}
	switch v.Value.(type) {
	case *hujson.Array:
		return d.lines(v, what)
	case *hujson.Object:
		return d.tome(v, what)
	}
	if lit, ok := literal(v); ok && lit.Kind() == '"' {
		text, ok := d.templated(v, what, true)
		return Inline(text), ok
	}
	want := "a string, a list of lines or a tome"
	if nullable {
		want += ", or null"
	}
	d.wrongForm(v, what, want)
	return Scripture{}, false
}

// lines decodes inline scripture written as a list of lines.
func (d *decoder) lines(v *hujson.Value, what string) (Scripture, bool) {
	arr := v.Value.(*hujson.Array)
	vs := make([]*hujson.Value, 0, len(arr.Elements))
	texts := make([]string, 0, len(arr.Elements))
	ok := true
	for i := range arr.Elements {
		line, isStr := d.str(&arr.Elements[i], "each line of "+what)
		ok = ok && isStr
		if isStr {
			vs = append(vs, &arr.Elements[i])
			texts = append(texts, line)
		}
	}
	if ok {
		d.lineMarks(vs, texts)
	}
	return Inline(strings.Join(texts, "\n")), ok
}

// tome decodes scripture drawn from a tome.
func (d *decoder) tome(v *hujson.Value, what string) (Scripture, bool) {
	fs, _ := d.object(v, "a tome", TomeKeys)
	var sc Scripture
	var ok bool
	if m, has := fs.get("tome"); has {
		sc.Tome, ok = d.templated(m.val, "the \"tome\" of "+what, false)
	} else {
		d.heresy(v, "a tome must name the scripture it is drawn from under \"tome\", such as {\"tome\": "+
			"\"snippets/visage.kdl\"}")
	}
	if m, has := fs.get("illuminate"); has {
		sc.Illuminate = d.boolean(m.val, "the \"illuminate\" of "+what)
	}
	return sc, ok
}
