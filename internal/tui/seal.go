package tui

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/tailscale/hujson"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
)

// sealing is the last station of the wizard: the rite rendered as
// scripture and the Inquisition's verdict upon it.
type sealing struct {
	rite   *librarium.Rite
	data   []byte
	path   string
	found  librarium.Findings // the verdict upon the rite
	heresy bool
	lost   bool // sealing drops the comments of the amended scripture
	offset int
}

func newSealing(m *model, d riteDraft) *sealing {
	path := librarium.NewPath(m.dir, d.name)
	if d.origPath != "" && d.origName == d.name {
		path = d.origPath // the scripture stays where it was recorded
	}
	s := &sealing{path: path, rite: d.toRite(path), lost: commentsLost(d.origPath)}
	data, err := librarium.Marshal(s.rite)
	if err != nil {
		s.heresy = true
		s.found = librarium.Findings{{Severity: librarium.Heresy, Scripture: path, Rite: d.name, Message: err.Error()}}
		return s
	}
	s.data = data
	lib := librarium.Preview(m.dir, librarium.Overlay{Name: d.name, Data: data, Replace: d.origName})
	found, _ := (&rituals.Servitor{Librarium: lib, Runes: m.runes}).Inquire([]string{d.name}, false)
	for _, f := range found {
		if f.Rite == d.name {
			s.found = append(s.found, f)
			s.heresy = s.heresy || f.Severity == librarium.Heresy
		}
	}
	return s
}

// commentsLost reports whether the scripture at path holds what only
// JSONC can say — comments or trailing commas — which sealing drops.
func commentsLost(path string) bool {
	if path == "" {
		return false
	}
	old, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	v, err := hujson.Parse(old)
	return err == nil && !v.IsStandard()
}

func (w *wizard) updateSeal(m *model, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return w, nil
	}
	s := w.seal
	switch k.String() {
	case "esc", "left", "h":
		w.station = stationLiturgy
	case "down", "j":
		s.offset++
	case "up", "k":
		s.offset = max(0, s.offset-1)
	case "g", "home":
		s.offset = 0
	case "G", "end":
		s.offset = math.MaxInt // the view holds it at the last page
	case "enter", "ctrl+s":
		if s.heresy {
			return w, m.notify(toastErr, "Heresy remains. The rite cannot be sealed.")
		}
		if err := w.sealRite(m); err != nil {
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

// sealRite writes the scripture, makes sure it reads back as the rite the
// wizard holds, and strikes the old scripture of a renamed rite.
func (w *wizard) sealRite(m *model) error {
	s := w.seal
	if back, _ := librarium.Parse(w.d.name, s.path, s.data); !back.Equal(s.rite) {
		return errors.New("the scripture would not read back as the rite the wizard holds; nothing is sealed")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".servitor-tmp"
	if err := os.WriteFile(tmp, s.data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if back, _ := librarium.LoadFile(s.path); !back.Equal(s.rite) {
		return fmt.Errorf("the scripture sealed at %s reads back otherwise than it was written", shortPath(s.path))
	}
	if w.d.origName != "" && w.d.origName != w.d.name {
		for _, p := range m.s.Librarium.Scriptures[w.d.origName] {
			if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	return nil
}

// sealView shows the verdict, then the scripture, scrolled by offset.
func (w *wizard) sealView(m *model, width, height int) string {
	t, s := m.t, w.seal
	var lines []string
	if s.heresy {
		lines = append(lines, t.danger.Render("✠ Heresy detected. Return and purify the rite:"))
	} else {
		lines = append(lines, t.ok.Render("✔ The rite is pure. Press enter to seal it into the Librarium."))
	}
	for _, f := range s.found {
		sev, st := f.Severity.String(), t.warn
		if f.Severity == librarium.Heresy {
			st = t.danger
		}
		head := fmt.Sprintf("%s %d:%d ", sev, f.Line, f.Column)
		for i, l := range wrap([]string{st.Render(head) + t.text.Render(f.Message)}, width-6) {
			lines = append(lines, map[bool]string{true: "  ", false: "    "}[i == 0]+l)
		}
	}
	if s.lost {
		lines = append(lines, t.warn.Render("  Beware: the comments of the amended scripture will not survive the sealing."))
	}
	lines = append(lines, t.dim.Render("  → "+truncateLeft(shortPath(s.path), width-6)), "")
	for _, line := range strings.Split(strings.TrimRight(string(s.data), "\n"), "\n") {
		lines = append(lines, "  "+highlightJSON(t, line))
	}
	avail := max(1, height)
	s.offset = min(s.offset, max(0, len(lines)-avail))
	return strings.Join(lines[s.offset:min(len(lines), s.offset+avail)], "\n")
}

// highlightJSON colors one line of indented JSON: every name of a member
// as an accent, every other string as text, and all else dim.
func highlightJSON(t *theme, line string) string {
	var b strings.Builder
	plain := 0 // the start of the dim run not yet written
	dim := func(end int) {
		if strings.TrimSpace(line[plain:end]) == "" {
			b.WriteString(line[plain:end])
		} else {
			b.WriteString(t.dim.Render(line[plain:end]))
		}
	}
	for i := 0; i < len(line); i++ {
		if line[i] != '"' {
			continue
		}
		end := i + 1
		for end < len(line) && line[end] != '"' {
			if line[end] == '\\' {
				end++
			}
			end++
		}
		end = min(end+1, len(line))
		dim(i)
		word := line[i:end]
		if strings.HasPrefix(strings.TrimLeft(line[end:], " "), ":") {
			b.WriteString(t.accent.Render(word))
		} else {
			b.WriteString(t.text.Render(word))
		}
		plain, i = end, end-1
	}
	dim(len(line))
	return b.String()
}
