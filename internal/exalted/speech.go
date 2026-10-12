package exalted

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/hallowed"
	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/placeholder"
	"github.com/nerdwave-nick/servitor/internal/sanction"
)

// HallRoot is where the exalted servitor raises the working hall of each
// session that speaks: HallRoot/<sigil>/.
const HallRoot = "/run/servitor"

// Speech is the world of exalted speech, whose Voice the exalted servitor
// speaks with. MachineSpeech is the machine's own; the trials raise worlds
// of their own in temporary halls.
type Speech struct {
	// Halls is where the working halls are raised; Sovereign reports
	// whether a uid is the root's, whom they must belong to.
	Halls     string
	Sovereign func(uid uint32) bool
	// Tongues is the ground the programs of the tongues are judged upon by
	// the trusted-path rule, and awakened from.
	Tongues hallowed.Ground
	// Getenv reads the environment the intercessor left, whence the locale
	// alone is taken.
	Getenv func(key string) (string, bool)
}

// MachineSpeech is the exalted speech of the machine itself.
func MachineSpeech() Speech {
	return Speech{Halls: HallRoot, Sovereign: func(uid uint32) bool { return uid == 0 },
		Tongues: hallowed.Machine, Getenv: os.LookupEnv}
}

// rootPath is the PATH exalted speech hears.
const rootPath = "/usr/local/sbin:/usr/local/bin:/usr/bin"

// locales are the variables of the locale exalted speech hears, when the
// environment it was roused in holds them as a true name of a locale.
var locales = []string{"LANG", "LC_ALL", "LC_ADDRESS", "LC_COLLATE", "LC_CTYPE", "LC_IDENTIFICATION",
	"LC_MEASUREMENT", "LC_MESSAGES", "LC_MONETARY", "LC_NAME", "LC_NUMERIC", "LC_PAPER", "LC_TELEPHONE", "LC_TIME"}

// localeName is the true name of a locale: never a path to one.
var localeName = regexp.MustCompile(`^[A-Za-z0-9_.@+-]{1,64}$`)

// granted are the variables of a sanction's environment the exalted
// servitor grants exalted speech, the inscriptions besides.
var (
	granted     = []string{"SERVITOR_RITE", "SERVITOR_ASPECT", "SERVITOR_FORMER_ASPECT"}
	inscription = regexp.MustCompile(`^SERVITOR_INSCRIPTION_[A-Z0-9_]+$`)
)

// Voice raises the speech of an accepted session: it judges every tongue
// and shebang of the plan by the trusted-path rule, builds the environment
// from what the exalted servitor grants alone, and raises a fresh working
// hall sealed 0700 holding the relics and scrolls of the plan. Whatever it
// refuses, nothing is touched and no hall is left.
func (w Speech) Voice(s Sitting) (Speaker, error) {
	sp, laid, err := w.prepare(s)
	if err != nil {
		return nil, err
	}
	if sp.hall, err = w.raise(s.Sigil, laid); err != nil {
		return nil, err
	}
	return sp, nil
}

// Weigh judges the speech of a sitting as Voice does — its environment,
// every tongue and shebang, every relic and scroll — and raises nothing.
func (w Speech) Weigh(s Sitting) error {
	_, _, err := w.prepare(s)
	return err
}

// prepare judges the speech of a sitting and readies its speaker, and what
// is to be laid in its working hall, which it does not raise.
func (w Speech) prepare(s Sitting) (*speaker, map[string]relic, error) {
	env, err := w.environ(s)
	if err != nil {
		return nil, nil, err
	}
	sp := &speaker{env: env, programs: map[string]string{}, biddings: map[string][]string{},
		scrolls: map[int]string{}}
	laid := map[string]relic{}
	for _, v := range s.Plan.Liturgy {
		if v.Kind != librarium.KindIncantation && v.Kind != librarium.KindLitany {
			continue
		}
		if err := sp.tongue(w.Tongues, v.Verse, v.Tongue); err != nil {
			return nil, nil, err
		}
		for _, r := range v.Relics {
			if err := lay(laid, v.Verse, r.Relic, r.Scripture, 0o600); err != nil {
				return nil, nil, err
			}
		}
		if v.Kind == librarium.KindLitany {
			if err := sp.scroll(w.Tongues, laid, v); err != nil {
				return nil, nil, err
			}
		}
	}
	return sp, laid, nil
}

// environ is the whole environment of the session's speech: PATH and HOME
// of the root, the locale of its invoker, and the variables the sanction
// declares, valued from the session.
func (w Speech) environ(s Sitting) ([]string, error) {
	e := s.Sanction.Environment
	if e.Pattern != sanction.EnvironmentPattern {
		return nil, fmt.Errorf("the sanction of %q grants its speech the environment of the pattern %q, which "+
			"this exalted servitor knows not", s.Sanction.Rite, e.Pattern)
	}
	words := map[string]string{"SERVITOR_RITE": s.Sanction.Rite, "SERVITOR_ASPECT": s.Plan.Aspect,
		"SERVITOR_FORMER_ASPECT": s.Former}
	for key, v := range s.Inscriptions {
		words[placeholder.InscriptionVariable(key)] = v
	}
	env := []string{"PATH=" + rootPath, "HOME=/root"}
	for _, name := range locales {
		if v, ok := w.Getenv(name); ok && localeName.MatchString(v) {
			env = append(env, name+"="+v)
		}
	}
	for _, key := range e.Keys {
		if !slices.Contains(granted, key) && !inscription.MatchString(key) {
			return nil, fmt.Errorf("the sanction of %q names %s for its speech to hear, which the exalted servitor "+
				"never grants", s.Sanction.Rite, key)
		}
		env = append(env, key+"="+words[key])
	}
	return env, nil
}

// relic is a scripture laid in the working hall: its words, its seal, and
// the verse that named it first.
type relic struct {
	scripture []byte
	seal      fs.FileMode
	verse     int
}

// lay names the scripture to be laid in the working hall under name, which
// must lie within it. A name laid twice must bear the same words.
func lay(laid map[string]relic, verse int, name string, scripture []byte, seal fs.FileMode) error {
	clean := filepath.Clean(name)
	if !filepath.IsLocal(clean) {
		return fmt.Errorf("verse %d names the relic %q, which lies beyond the hall of its rite", verse, name)
	}
	if was, ok := laid[clean]; ok {
		if !bytes.Equal(was.scripture, scripture) {
			return fmt.Errorf("verse %d lays %q in the working hall, yet verse %d laid other words there", verse,
				clean, was.verse)
		}
		seal = max(seal, was.seal)
		verse = was.verse
	}
	laid[clean] = relic{scripture: scripture, seal: seal, verse: verse}
	return nil
}

// raise raises the working hall of the sigil afresh, sealed 0700 and the
// root's, beneath halls of the root's, and lays in it what was named;
// whatever falls, no hall is left.
func (w Speech) raise(sigil string, laid map[string]relic) (string, error) {
	if err := os.MkdirAll(w.Halls, 0o755); err != nil {
		return "", fmt.Errorf("the halls of exalted speech cannot be raised: %w", err)
	}
	if err := w.sovereign(w.Halls, 0o755); err != nil {
		return "", err
	}
	hall := filepath.Join(w.Halls, sigil)
	// a hall that stands is the remnant of a session that fell unawares
	if err := os.RemoveAll(hall); err != nil {
		return "", fmt.Errorf("the remnant of a former working hall %s cannot be cast down: %w", hall, err)
	}
	if err := os.Mkdir(hall, 0o700); err != nil {
		return "", fmt.Errorf("the working hall %s cannot be raised: %w", hall, err)
	}
	err := w.sovereign(hall, 0o700)
	if err == nil {
		err = fill(hall, laid)
	}
	if err != nil {
		_ = os.RemoveAll(hall)
		return "", err
	}
	return hall, nil
}

// sovereign checks that the hall at path is a hall of the root's, sealed
// no wider than seal.
func (w Speech) sovereign(path string, seal fs.FileMode) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("the hall %s cannot be beheld: %w", path, err)
	}
	st, ok := fi.Sys().(*sysStat)
	switch {
	case !fi.IsDir():
		return fmt.Errorf("%s is no hall", path)
	case !ok || w.Sovereign == nil || !w.Sovereign(st.Uid):
		return fmt.Errorf("%s is a hall the root does not own", path)
	case fi.Mode().Perm()&^seal != 0:
		return fmt.Errorf("%s is a hall sealed %04o, wider than %04o", path, fi.Mode().Perm(), seal)
	}
	return nil
}

// fill lays every scripture named in the hall, never beyond it.
func fill(hall string, laid map[string]relic) error {
	root, err := os.OpenRoot(hall)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	for _, name := range slices.Sorted(maps.Keys(laid)) {
		r := laid[name]
		if dir := filepath.Dir(name); dir != "." {
			if err := root.MkdirAll(dir, 0o700); err != nil {
				return fmt.Errorf("the hall of the relic %q cannot be raised: %w", name, err)
			}
		}
		f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, r.seal)
		if err != nil {
			return fmt.Errorf("the relic %q cannot be laid: %w", name, err)
		}
		_, err = f.Write(r.scripture)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return fmt.Errorf("the relic %q cannot be laid: %w", name, err)
		}
	}
	return nil
}

// errNoTongue is the program of a tongue that does not begin at the root.
var errNoTongue = errors.New("does not begin at the root")

// judge judges the program of tongue (or of an interpreter) by the
// trusted-path rule, once, and keeps where it is awakened from.
func (sp *speaker) judge(g hallowed.Ground, verse int, tongue string) error {
	if _, ok := sp.programs[tongue]; ok {
		return nil
	}
	if !filepath.IsAbs(tongue) {
		return fmt.Errorf("the tongue %q of verse %d %v", tongue, verse, errNoTongue)
	}
	program, err := g.Program(tongue)
	if err != nil {
		return fmt.Errorf("the tongue of verse %d is not spoken under exaltation: %w", verse, err)
	}
	sp.programs[tongue] = filepath.Join(g.Root, program)
	return nil
}

// tongue judges the program of the tongue of verse as judge does, and as
// one of the tongues exalted speech knows, by the name of the program it
// truly is; it keeps how that program is bidden to read no scripture of its
// own.
func (sp *speaker) tongue(g hallowed.Ground, verse int, tongue string) error {
	if err := sp.judge(g, verse, tongue); err != nil {
		return err
	}
	program := sp.programs[tongue]
	bidding, known := librarium.ExaltedTongue(program)
	if !known {
		return fmt.Errorf("the tongue %s of verse %d is %s, which exalted speech is never spoken in: it knows "+
			"only %s, each bidden to read no scripture of its own as it awakens", tongue, verse,
			filepath.Base(program), librarium.ExaltedTongueNames)
	}
	sp.biddings[program] = bidding
	return nil
}

// spoken is how the program of the tongue of v is awakened, and bidden to
// read no scripture of its own.
func (sp *speaker) spoken(v sanction.Verse) ([]string, error) {
	program := sp.programs[v.Tongue]
	bidding, ok := sp.biddings[program]
	if program == "" || !ok {
		return nil, fmt.Errorf("the tongue of verse %d was never judged", v.Verse)
	}
	return append([]string{program}, bidding...), nil
}

// scroll names the scroll of a litany to be laid in the working hall: under
// the name it is written by when that lies within the rite's hall, else
// under its last name. A scroll whose words begin with a shebang is
// recited by the interpreter it names, which is judged as a tongue.
func (sp *speaker) scroll(g hallowed.Ground, laid map[string]relic, v sanction.Verse) error {
	name := filepath.Clean(v.Target)
	if !filepath.IsLocal(name) {
		name = filepath.Base(name)
	}
	if err := lay(laid, v.Verse, name, v.Scroll, 0o600); err != nil {
		return err
	}
	sp.scrolls[v.Verse] = name
	if interpreter, _, ok := shebang(v.Scroll); ok {
		return sp.judge(g, v.Verse, interpreter)
	}
	return nil
}

// shebang is the interpreter a scroll names in its first line and the one
// word it is offered, as the machine reads it.
func shebang(scroll []byte) (interpreter, word string, ok bool) {
	line, found := bytes.CutPrefix(scroll, []byte("#!"))
	if !found {
		return "", "", false
	}
	line, _, _ = bytes.Cut(line, []byte("\n"))
	rest := strings.Trim(string(line), " \t\r")
	interpreter = rest
	if i := strings.IndexAny(rest, " \t"); i >= 0 {
		interpreter, word = rest[:i], strings.Trim(rest[i:], " \t")
	}
	return interpreter, word, interpreter != ""
}
