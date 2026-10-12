package librarium

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Speech is one verse that speaks, as the inquisition weighs its words:
// a command, a reversion, or the scroll of a litany.
type Speech struct {
	Verse       int      // counted from one
	Exalted     bool     // the verse is exalted
	Relics      []string // the relics of an exalted verse, as written
	AmidExalted bool     // the rite holds exalted verses
}

// vouchers are the programs that vouch for a master before the machine's
// own authority is lent; guarded are the programs the machine guards by an
// intercessor of its own, asking for the master's proof by themselves.
var (
	vouchers = []string{"sudo", "run0", "pkexec", "doas"}
	guarded  = []string{"systemctl", "hostnamectl", "timedatectl", "localectl", "loginctl", "machinectl",
		"resolvectl", "networkctl", "homectl", "udisksctl"}
)

var (
	// simpleCalls splits words into the calls they make.
	simpleCalls = regexp.MustCompile("[;&|()\n`]+")
	// homeward finds $HOME, ${HOME} and a leading ~.
	homeward = regexp.MustCompile("\\$\\{?HOME\\b|(?:^|[\\s=:;&|(\"'`])~")
	// beside finds ./<name>, a scripture beside the speech.
	beside = regexp.MustCompile("(?:^|[\\s=:;&|(\"'`])\\./([^\\s;&|()<>\"'`]*)")
	// sourced finds the scripture of source <name> and . <name>.
	sourced = regexp.MustCompile("(?:^|[;&|(\n`])\\s*(?:source|\\.)\\s+(\"[^\"]*\"|'[^']*'|[^\\s;&|()<>`]+)")
)

// Impurities returns the impurities of words the verse speaks. Words that
// are not exalted should not call upon an intercessor, nor — beside
// exalted verses — upon a guarded program. Exalted words should reach
// neither into the invoker's home nor for scripture beside them that is not
// among their relics.
func (s Speech) Impurities(words string) []string {
	var out []string
	if !s.Exalted {
		if n := calls(words, vouchers, false); n != "" {
			out = append(out, fmt.Sprintf("verse %d calls upon the intercessor %q within words that are not "+
				"exalted: such words ask for the master's proof by themselves, fall where no terminal attends, and "+
				"leave that proof lingering for whatever follows — strike %q and write \"exalted\": true upon the "+
				"verse, that the servitor itself vouches for it once", s.Verse, n, n))
		}
		if n := calls(words, guarded, true); n != "" && s.AmidExalted {
			out = append(out, fmt.Sprintf("verse %d calls %q, which the machine guards by an intercessor of its "+
				"own: it asks for the master's proof by itself, beside the one proof the exalted verses of this "+
				"rite already demand — write \"exalted\": true upon the verse, that one proof serves both", s.Verse, n))
		}
		return out
	}
	if homeward.MatchString(words) {
		out = append(out, fmt.Sprintf("the exalted speech of verse %d reaches into the home of its invoker "+
			"($HOME or ~): exalted speech is spoken with the machine's own home, and what lies in the invoker's "+
			"may be altered by the invoker's hand after its sanction — lay what it reads beside the rite's "+
			"scripture and name it under \"relics\"", s.Verse))
	}
	if strays := s.strays(words); len(strays) > 0 {
		is, it := "is", "it"
		if len(strays) > 1 {
			is, it = "are", "them"
		}
		out = append(out, fmt.Sprintf("the exalted speech of verse %d reaches for %s, which %s not among its "+
			"relics: exalted speech is spoken in a hall of the exalted servitor's own that holds its relics "+
			"alone — lay %s beside the rite's scripture and name %s under \"relics\"", s.Verse,
			strings.Join(strays, ", "), is, it, it))
	}
	return out
}

// calls returns the first of names the words call upon as a program, ""
// for none; with mastersOwn, a call bearing --user is the master's own and
// is passed over.
func calls(words string, names []string, mastersOwn bool) string {
	for _, call := range simpleCalls.Split(words, -1) {
		fields := strings.Fields(call)
		if mastersOwn && slices.Contains(fields, "--user") {
			continue
		}
		for _, f := range fields {
			if n := path.Base(strings.Trim(f, `"'`)); slices.Contains(names, n) {
				return n
			}
		}
	}
	return ""
}

// strays returns the scriptures the words read beside them, or source,
// that are not among the relics, each once.
func (s Speech) strays(words string) []string {
	var relics, found []string
	for _, r := range s.Relics {
		relics = append(relics, filepath.Clean(r))
	}
	add := func(p string) {
		if c := filepath.Clean(p); !slices.Contains(relics, c) && !slices.Contains(found, c) {
			found = append(found, c)
		}
	}
	for _, m := range beside.FindAllStringSubmatch(words, -1) {
		add(m[1])
	}
	for _, m := range sourced.FindAllStringSubmatch(words, -1) {
		add(strings.Trim(m[1], `"'`))
	}
	return found
}
