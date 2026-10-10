package tui

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/tailscale/hujson"

	"github.com/nerdwave-nick/servitor/internal/config"
)

// review is the final wizard step: the rendered definition and its
// validation result.
type review struct {
	data   []byte
	path   string
	diags  config.Diagnostics
	errs   bool
	lost   bool // saving drops comments of the original file
	offset int
}

func newReview(m *model, d draft) *review {
	sw := d.toSwitch()
	data, err := config.Marshal(sw)
	r := &review{data: data, path: config.NewPath(m.dir, d.name)}
	if err != nil {
		r.diags, r.errs = config.Diagnostics{{Severity: config.SevError, Message: err.Error()}}, true
		return r
	}
	if d.origName == d.name && d.origPath != "" {
		r.path = d.origPath // keep the existing file in place
	}
	set := config.Preview(m.dir, config.Overlay{Name: d.name, Data: data, Replace: d.origName})
	for _, dg := range set.Diags {
		if dg.Switch == d.name {
			r.diags = append(r.diags, dg)
			r.errs = r.errs || dg.Severity == config.SevError
		}
	}
	if d.origPath != "" {
		if old, err := os.ReadFile(d.origPath); err == nil {
			if v, err := hujson.Parse(old); err == nil && !v.IsStandard() {
				r.lost = true
			}
		}
	}
	return r
}

func (w *wizard) updateReview(m *model, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return w, nil
	}
	r := w.review
	switch k.String() {
	case "esc", "left", "h":
		w.step = stepVessels
	case "down", "j":
		r.offset++
	case "up", "k":
		r.offset = max(0, r.offset-1)
	case "enter", "ctrl+s":
		if r.errs {
			return w, m.notify(toastErr, "Heresy remains. The rite cannot be sealed.")
		}
		if err := w.save(m); err != nil {
			return w, m.notify(toastErr, err.Error())
		}
		m.reload(w.d.name)
		verb := "consecrated"
		if w.d.origName != "" {
			verb = "amended"
		}
		return nil, m.notify(toastOK, "The rite "+w.d.name+" is "+verb+". Glory to the Omnissiah.")
	}
	return w, nil
}

// save writes the definition and removes the old file after a rename.
func (w *wizard) save(m *model) error {
	r := w.review
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	tmp := r.path + ".servitor-tmp"
	if err := os.WriteFile(tmp, r.data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, r.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if w.d.origName != "" && w.d.origName != w.d.name {
		for _, p := range m.set.Files[w.d.origName] {
			if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	return nil
}

func (w *wizard) reviewView(m *model, width, height int) string {
	t, r := m.t, w.review
	var head []string
	switch {
	case r.errs:
		head = append(head, t.danger.Render("✠ Heresy detected. Return and purify the rite:"))
	default:
		head = append(head, t.ok.Render("✔ The rite is pure. Press enter to seal it into the Librarium."))
	}
	for _, d := range r.diags {
		st := t.warn
		if d.Severity == config.SevError {
			st = t.danger
		}
		head = append(head, st.Render("  "+truncate(formatLoc(d)+": "+d.Message, width-4)))
	}
	if r.lost {
		head = append(head, t.warn.Render("  Beware: comments in the original scripture will not survive the amendment."))
	}
	head = append(head, t.dim.Render("  → "+truncateLeft(shortPath(r.path), width-6)), "")
	lines := strings.Split(strings.TrimRight(string(r.data), "\n"), "\n")
	avail := max(1, height-len(head))
	r.offset = min(r.offset, max(0, len(lines)-avail))
	body := lines[r.offset:min(len(lines), r.offset+avail)]
	for i, line := range body {
		body[i] = "  " + highlightJSON(t, line)
	}
	return strings.Join(append(head, body...), "\n")
}

// highlightJSON colors keys and strings of one line of indented JSON.
func highlightJSON(t *theme, line string) string {
	trimmed := strings.TrimLeft(line, " ")
	indent := line[:len(line)-len(trimmed)]
	if strings.HasPrefix(trimmed, `"`) {
		if i := bytes.Index([]byte(trimmed), []byte(`": `)); i > 0 {
			return indent + t.accent.Render(trimmed[:i+1]) + t.dim.Render(":") + t.text.Render(trimmed[i+2:])
		}
		return indent + t.text.Render(trimmed)
	}
	return indent + t.dim.Render(trimmed)
}
