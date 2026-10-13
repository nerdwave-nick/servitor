// Package tui implements the cogitator, the interactive interface of servitor.
package tui

import (
	"io"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/nerdwave-nick/servitor/internal/lexicon"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// Options configure the TUI.
type Options struct {
	Dir   string
	Runes librarium.Runes // the servitor's runes that sway the settings
	// Convert converts every scripture of pattern Mark I the cogitator
	// reads to pattern Mark II, on disk, and tells it in a toast.
	Convert bool
	Input   io.Reader // defaults to the terminal
	Output  io.Writer
}

// Run starts the TUI and blocks until it exits.
func Run(opt Options) error {
	var popts []tea.ProgramOption
	if opt.Input != nil {
		popts = append(popts, tea.WithInput(opt.Input))
	}
	if opt.Output != nil {
		popts = append(popts, tea.WithOutput(opt.Output))
	}
	m := newModel(opt.Dir, opt.Runes, opt.Convert)
	p := tea.NewProgram(m, popts...)
	m.send = p.Send
	_, err := p.Run()
	close(m.gone)
	m.running.Wait() // a rite is never abandoned half performed
	return err
}

// row is one entry of the overview list.
type row struct {
	name     string
	reading  rituals.Reading    // its augury; Rite is nil when it is heretical
	heresies librarium.Findings // the heresies of a heretical rite
	paths    []string           // its scriptures
	// sanction is how the rite stands towards its sanction: as its augury
	// tells, or forsaken when it holds no exalted verse yet a sanction is
	// kept under its name; kept is that sanction as listed, if any.
	sanction sanction.Standing
	kept     *rituals.Sanctioned
}

// rite is the rite of the row, or nil when it is heretical.
func (r *row) rite() *librarium.Rite { return r.reading.Rite }

type toastKind int

const (
	toastInfo toastKind = iota
	toastOK
	toastErr
)

type toast struct {
	text string
	kind toastKind
	id   int
}

type (
	toastExpiredMsg struct{ id int }
	reloadMsg       struct {
		selected string
		toast    *toast
	}
)

// screen is a modal dialog or the wizard, drawn above the overview.
type screen interface {
	update(m *model, msg tea.Msg) (screen, tea.Cmd)
	view(m *model) string
	fullscreen() bool
}

type model struct {
	dir     string
	runes   librarium.Runes
	convert bool // scripture of pattern Mark I is converted at every reading
	t       *theme
	s       *rituals.Servitor // the Librarium as last read
	send    func(tea.Msg)     // delivers what a running invocation tells
	last    *rituals.Result   // the last invocation performed, for its words
	all     []row
	rows    []row // all rows matching the filter
	cursor  int
	filter  textinput.Model
	typing  bool // filter input focused
	width   int
	height  int
	bodyH   int // height available between header and footer
	toast   toast
	thought int
	screen  screen
	running sync.WaitGroup // the invocation, sanctioning or abjuring being performed, if any
	gone    chan struct{}  // closed once the cogitator has ended
	// exec lends the terminal to a command, as tea.Exec does.
	exec func(tea.ExecCommand, tea.ExecCallback) tea.Cmd
}

func newModel(dir string, runes librarium.Runes, convert bool) *model {
	f := textinput.New()
	f.Prompt = "/ "
	m := &model{dir: dir, runes: runes, convert: convert, t: newTheme(), filter: f, send: func(tea.Msg) {},
		thought: int(time.Now().UnixNano() % int64(len(lexicon.Thoughts))), width: 100, height: 30,
		gone: make(chan struct{}), exec: tea.Exec}
	m.reload("")
	return m
}

// reload re-reads the configuration and keeps the selection on name.
func (m *model) reload(selected string) {
	if selected == "" {
		if r := m.current(); r != nil {
			selected = r.name
		}
	}
	if m.convert {
		m.s = rituals.OpenConverting(m.dir, m.runes)
		m.converted(m.s.Convert())
	} else {
		m.s = rituals.Open(m.dir, m.runes)
	}
	lib := m.s.Librarium
	kept := map[string]*rituals.Sanctioned{}
	list, _ := m.s.ListSanctions() // a sanction that cannot be read lapses its rite in the augury
	for i := range list {
		kept[list[i].Rite] = &list[i]
	}
	m.all = m.all[:0]
	for _, name := range lib.Names() {
		reading, _ := m.s.Augur(name, false)
		r := row{name: name, reading: reading, paths: lib.Scriptures[name], sanction: reading.Sanction,
			kept: kept[name]}
		if r.kept != nil && r.kept.Standing == sanction.Forsaken {
			r.sanction = sanction.Forsaken
		}
		for _, f := range lib.Findings {
			if f.Rite == name && f.Severity == librarium.Heresy {
				r.heresies = append(r.heresies, f)
			}
		}
		m.all = append(m.all, r)
	}
	m.applyFilter(selected)
}

func (m *model) applyFilter(selected string) {
	q := strings.ToLower(m.filter.Value())
	m.rows = m.rows[:0]
	for _, r := range m.all {
		if q == "" || strings.Contains(strings.ToLower(r.name), q) {
			m.rows = append(m.rows, r)
		}
	}
	m.cursor = min(m.cursor, max(0, len(m.rows)-1))
	for i, r := range m.rows {
		if r.name == selected {
			m.cursor = i
		}
	}
}

func (m *model) current() *row {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return nil
	}
	return &m.rows[m.cursor]
}

func (m *model) notify(kind toastKind, text string) tea.Cmd {
	m.toast = toast{text: text, kind: kind, id: m.toast.id + 1}
	return fade(m.toast.id)
}

// fade lets the toast id fade after a while.
func fade(id int) tea.Cmd {
	return tea.Tick(6*time.Second, func(time.Time) tea.Msg { return toastExpiredMsg{id: id} })
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case toastExpiredMsg:
		if msg.id == m.toast.id {
			m.toast.text = ""
		}
		return m, nil
	case reloadMsg:
		m.reload(msg.selected)
		if msg.toast != nil {
			return m, m.notify(msg.toast.kind, msg.toast.text)
		}
		return m, nil
	case invokedMsg:
		cmd := m.invoked(msg)
		if msg.hungUp {
			return m, tea.Quit // the terminal is gone; the rite has ended
		}
		return m, cmd
	case provenMsg:
		return m, m.proven(msg)
	case lendMsg:
		return m, m.lend(msg.l)
	case reclaimedMsg:
		msg.l.forgo(msg.err)
		return m, nil
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" && !m.isRunning() {
			return m, tea.Quit
		}
	}
	if m.screen != nil {
		next, cmd := m.screen.update(m, msg)
		m.screen = next
		return m, cmd
	}
	return m, m.updateOverview(msg)
}

func (m *model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.WindowTitle = "servitor ⚙ cogitator"
	return v
}

func (m *model) render() string {
	if m.width < 50 || m.height < 12 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			m.t.warn.Render("The cogitator demands a larger viewscreen (50×12)."))
	}
	header := m.header()
	footer := m.footer()
	bodyH := m.height - lipgloss.Height(header) - lipgloss.Height(footer)
	m.bodyH = bodyH
	var body string
	if m.screen != nil && m.screen.fullscreen() {
		body = m.screen.view(m)
		body = lipgloss.NewStyle().Height(bodyH).MaxHeight(bodyH).Render(body)
	} else {
		body = m.overview(bodyH)
	}
	base := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	if m.screen == nil || m.screen.fullscreen() {
		return base
	}
	box := m.screen.view(m)
	x := max(0, (m.width-lipgloss.Width(box))/2)
	y := max(1, (m.height-lipgloss.Height(box))/2)
	return lipgloss.NewCompositor(lipgloss.NewLayer(base), lipgloss.NewLayer(box).X(x).Y(y).Z(1)).Render()
}

func (m *model) header() string {
	t := m.t
	left := t.barAccent.Render(" "+t.glyphLogo+" SERVITOR ") + t.bar.Render("▸ COGITATOR ")
	label := "Librarium: "
	path := truncateLeft(shortPath(m.dir), m.width-lipgloss.Width(left)-lipgloss.Width(label)-3)
	right := t.bar.Render(" " + label + path + " ")
	gap := max(0, m.width-lipgloss.Width(left)-lipgloss.Width(right))
	return left + t.bar.Render(strings.Repeat(" ", gap)) + right
}

func (m *model) footer() string {
	t := m.t
	var status string
	switch m.toast.kind {
	case toastOK:
		status = t.ok.Render("✔ " + m.toast.text)
	case toastErr:
		status = t.danger.Render("✖ " + m.toast.text)
	default:
		status = t.text.Render(m.toast.text)
	}
	if m.toast.text == "" {
		status = ""
	}
	lines := []string{fit(status, m.width), fit(m.keyHints(), m.width)}
	lines = append(lines, fit(t.dim.Render("+++ Thought for the day: "+lexicon.Thought(m.thought)+" +++"), m.width))
	return strings.Join(lines, "\n")
}

// shortPath replaces the home directory with "~".
func shortPath(p string) string {
	if home, err := userHome(); err == nil && home != "" && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}
