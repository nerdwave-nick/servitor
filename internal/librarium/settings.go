package librarium

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/tailscale/hujson"
)

// SettingsName is the name of the settings' scripture within a Librarium.
const SettingsName = "servitor.json"

// SettingsKeys are the keys of the settings, in the order of the codex.
var SettingsKeys = []string{"pattern", "tongue", "chronicle", "vox", "patience"}

// The vox of the settings: whether tidings reach the desktop.
const (
	VoxNotifySend = "notify-send" // tidings are sent through notify-send
	VoxOff        = "off"         // the desktop stays silent
)

// Voxes are the vox the settings may command.
var Voxes = []string{VoxNotifySend, VoxOff}

// Settings are the servitor's standing orders as written in a Librarium's
// servitor.json (pattern Mark I). An order not written — or written in
// heresy — stays at its zero value; Resolve applies runes, the environment
// and the defaults.
type Settings struct {
	Path      string        // where the settings are kept, whether or not they exist
	Tongue    string        // "" when not written
	Chronicle string        // as written; "" when not written
	Vox       string        // VoxNotifySend, VoxOff, or "" when not written
	Patience  time.Duration // 0 when not written
}

// SettingsPath returns where the settings of the Librarium dir are kept.
func SettingsPath(dir string) string { return filepath.Join(dir, SettingsName) }

// LoadSettings reads and examines the settings of the Librarium dir. A
// Librarium without settings yields unwritten settings and no finding.
func LoadSettings(dir string) (*Settings, Findings) {
	path := SettingsPath(dir)
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return &Settings{Path: path}, nil
	case err != nil:
		return &Settings{Path: path}, Findings{readHeresy(path, "", err)}
	}
	return ParseSettings(path, data)
}

// ParseSettings reads the settings' scripture data kept at path. The
// settings are never nil: whatever cannot be read in pattern Mark I stays
// unwritten, and the findings denounce why.
func ParseSettings(path string, data []byte) (*Settings, Findings) {
	s := &Settings{Path: path}
	src := &source{path: path, data: data}
	// The settings belong to no rite: findings carry no rite's name.
	d := &decoder{src: src, rite: &Rite{}}
	ast, err := hujson.Parse(data)
	if err != nil {
		return s, Findings{{Severity: Heresy, Scripture: path, Position: syntaxPosition(err),
			Message: "the settings are malformed beyond reading here: their brackets, quotes, colons or commas " +
				"break the holy form of JSONC, and the servitor cannot heed standing orders whose very letters " +
				"are corrupt"}}
	}
	src.ast = ast
	d.decodeSettings(&src.ast, s)
	d.findings.Sort()
	return s, d.findings
}

func (d *decoder) decodeSettings(root *hujson.Value, s *Settings) {
	obj, ok := root.Value.(*hujson.Object)
	if !ok {
		d.wrongForm(root, "the settings", "one object of keys between { and }")
		return
	}
	if !d.decodePattern(root, obj) {
		return
	}
	fs, _ := d.object(root, "the settings", SettingsKeys)
	if m, ok := fs.get("tongue"); ok {
		if t, ok := d.word(m.val, `the "tongue" of the settings`); ok {
			s.Tongue = t
		}
	}
	if m, ok := fs.get("chronicle"); ok {
		if c, ok := d.word(m.val, `the "chronicle" of the settings`); ok {
			s.Chronicle = c
		}
	}
	if m, ok := fs.get("vox"); ok {
		s.Vox = d.vox(m.val)
	}
	if m, ok := fs.get("patience"); ok {
		s.Patience = d.patience(m.val, `the "patience" of the settings`)
	}
}

// vox decodes the vox of the settings; "" when heretical.
func (d *decoder) vox(v *hujson.Value) string {
	s, ok := d.str(v, `the "vox" of the settings`)
	if !ok {
		return ""
	}
	if slices.Contains(Voxes, s) {
		return s
	}
	d.heresy(v, "%q is no vox the codex knows; the \"vox\" of the settings must be %q, that tidings of "+
		"every invocation reach the desktop through notify-send, or %q, that the desktop stays silent "+
		"(a terminal still hears them)", s, VoxNotifySend, VoxOff)
	return ""
}
