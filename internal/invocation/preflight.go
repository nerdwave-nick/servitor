package invocation

import (
	"errors"
	"fmt"
	"os"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
)

// preflight examines the steps of a rite one after another without
// changing anything, planning each against a shadow of the machine.
type preflight struct {
	rite     *librarium.Rite
	opts     Options
	values   placeholder.Values
	world    *shadow
	index    int // index of the step being examined
	heresies Heresies
}

// prepare examines one step; it returns nil when the step is heretical.
func (p *preflight) prepare(s librarium.Step) performer {
	switch s := s.(type) {
	case *librarium.Sanctum:
		return p.sanctum(s)
	case *librarium.Transcription:
		return p.transcription(s)
	case *librarium.Tether:
		return p.tether(s)
	case *librarium.Incantation:
		return p.incantation(s)
	case *librarium.Litany:
		return p.litany(s)
	case *librarium.VoxCast:
		return voxCast{v: p.verse(s.Kind(), s.Tidings), tidings: Tidings(s.Tidings)}
	}
	return nil
}

// verse names the step being examined.
func (p *preflight) verse(k librarium.Kind, target string) Verse {
	return Verse{Number: p.index + 1, Kind: k, Target: target}
}

// pointer is the JSON pointer of field within the step being examined.
func (p *preflight) pointer(field string) string {
	return fmt.Sprintf("/liturgy/%d/%s", p.index, field)
}

// mapField is the field of an aspect map that answers for the invoked
// aspect: the field itself when written as one value, else its entry.
func mapField[T any](p *preflight, field string, m librarium.AspectMap[T]) string {
	if key, ok := m.Key(p.opts.Aspect); ok && !m.IsUniform() {
		return field + "/" + key
	}
	return field
}

// denounce records a heresy at field of the step being examined.
func (p *preflight) denounce(field, format string, args ...any) {
	p.heresies = append(p.heresies, Heresy{Verse: p.index + 1, Finding: librarium.Finding{
		Severity: librarium.Heresy, Scripture: p.rite.Path, Rite: p.rite.Name,
		Position: p.rite.Locate(p.pointer(field)), Message: fmt.Sprintf(format, args...),
	}})
}

// render illuminates text written at field.
func (p *preflight) render(field, text string) (string, bool) {
	out, err := p.rite.Scope().Render(text, p.values)
	if err != nil {
		p.denounce(field, "%s", firstHeresy(err))
		return "", false
	}
	return out, true
}

// path illuminates the path written at field and resolves it against the
// rite's directory.
func (p *preflight) path(field, text string) (string, bool) {
	out, ok := p.render(field, text)
	if !ok {
		return "", false
	}
	return p.rite.ResolvePath(out), true
}

func firstHeresy(err error) string {
	var hs placeholder.Heresies
	if errors.As(err, &hs) && len(hs) > 0 {
		return hs[0].Message()
	}
	return err.Error()
}

// scripture renders the scripture the invoked aspect demands of the step's
// field. null is true for a null scripture.
func (p *preflight) scripture(field string, m librarium.AspectMap[librarium.Scripture]) (text string, null, ok bool) {
	sc, has := m.For(p.opts.Aspect)
	field = mapField(p, field, m)
	if !has {
		p.denounce(field, "no scripture is written for the aspect %q, and no \"*\" serves in its stead", p.opts.Aspect)
		return "", false, false
	}
	if sc.Null {
		return "", true, true
	}
	if !sc.IsTome() {
		text, ok := p.render(field, sc.Text)
		return text, false, ok
	}
	tomePath, ok := p.path(field+"/tome", sc.Tome)
	if !ok {
		return "", false, false
	}
	data, err := os.ReadFile(tomePath)
	if err != nil {
		p.denounce(field+"/tome", "the tome cannot be opened: %v", lament(tomePath, err))
		return "", false, false
	}
	if !sc.Illuminate {
		return string(data), false, true
	}
	text, err = p.rite.Scope().Render(string(data), p.values)
	var hs placeholder.Heresies
	if errors.As(err, &hs) {
		for _, h := range hs {
			p.heresies = append(p.heresies, Heresy{Verse: p.index + 1, Finding: librarium.Finding{
				Severity: librarium.Heresy, Scripture: tomePath, Rite: p.rite.Name,
				Position: librarium.Position{Line: h.Pos.Line, Column: h.Pos.Column},
				Message:  "within the illuminated tome: " + h.Message(),
			}})
		}
		return "", false, false
	}
	return text, false, true
}

// known renders the scripture of every aspect of m, with the inscriptions
// of this invocation and those on the data-slate, by which a transcription
// recognises a vessel it wrote itself. Scripture that cannot be rendered is
// left out; a null scripture is not scripture.
func (p *preflight) known(m librarium.AspectMap[librarium.Scripture]) []string {
	sets := []map[string]string{p.opts.Inscriptions}
	if p.opts.Recorded != nil {
		sets = append(sets, p.opts.Recorded)
	}
	var out []string
	for _, aspect := range p.rite.Aspects {
		sc, ok := m.For(aspect)
		if !ok || sc.Null {
			continue
		}
		for _, ins := range sets {
			v := values(p.rite, Options{Aspect: aspect}, ins)
			if text, ok := renderQuietly(p.rite, sc, v); ok {
				out = append(out, text)
			}
		}
	}
	return out
}

func renderQuietly(r *librarium.Rite, sc librarium.Scripture, v placeholder.Values) (string, bool) {
	scope := r.Scope()
	if !sc.IsTome() {
		text, err := scope.Render(sc.Text, v)
		return text, err == nil
	}
	tomePath, err := scope.Render(sc.Tome, v)
	if err != nil {
		return "", false
	}
	data, err := os.ReadFile(r.ResolvePath(tomePath))
	if err != nil {
		return "", false
	}
	if !sc.Illuminate {
		return string(data), true
	}
	text, err := scope.Render(string(data), v)
	return text, err == nil
}
