package sanction

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Sanction is a unit its master sanctioned: the unit, its sigil, and the
// hour it was sanctioned. Its JSON is strict, as the store keeps it.
type Sanction struct {
	Unit         `json:",inline"`
	Sigil        string    `json:"sigil"`
	SanctionedAt time.Time `json:"sanctioned_at"`
}

// New is the sanction of u, sanctioned at the hour at.
func New(u *Unit, at time.Time) *Sanction {
	return &Sanction{Unit: *u, Sigil: u.Sigil(), SanctionedAt: at}
}

// DefaultRoot is where the store of sanctions lies.
const DefaultRoot = "/var/lib/servitor/sanctions"

// limit is how much of a sanction the store reads.
const limit = 64 << 20

// UnitCap is the most bytes the canonical JSON of a unit may hold to be
// sanctioned, so that its sanction, a little longer, is read whole.
const UnitCap = limit - 1<<20

// Store is the store of one master's sanctions: <Root>/<UID>/<rite>.json.
// A Store without Root holds none. Reading it changes nothing.
type Store struct {
	Root string
	UID  int
}

// hall is where the master's sanctions lie.
func (s Store) hall() string { return filepath.Join(s.Root, strconv.Itoa(s.UID)) }

// Path is where the sanction of the rite lies.
func (s Store) Path(rite string) string { return filepath.Join(s.hall(), rite+".json") }

// Read reads the sanction of the rite: nil, and no lament, when none is
// kept, or the store itself is not. A sanction that cannot be read, or is no
// true sanction of the rite and master, is lamented.
func (s Store) Read(rite string) (*Sanction, error) {
	if s.Root == "" {
		return nil, nil
	}
	p := s.Path(rite)
	f, err := os.Open(p)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("the sanction of %q at %s cannot be read: %w", rite, p, err)
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, limit))
	if err != nil {
		return nil, fmt.Errorf("the sanction of %q at %s cannot be read: %w", rite, p, err)
	}
	return s.parse(data, rite)
}

// parse reads data, laid at the place of the rite's sanction, strictly as
// a true sanction of the rite and master.
func (s Store) parse(data []byte, rite string) (*Sanction, error) {
	p := s.Path(rite)
	var sn Sanction
	if err := json.Unmarshal(data, &sn, json.RejectUnknownMembers(true)); err != nil {
		return nil, fmt.Errorf("the sanction of %q at %s is corrupt beyond reading: %v", rite, p, err)
	}
	if why := sn.untrue(rite, s.UID); why != "" {
		return nil, fmt.Errorf("the sanction at %s is no true sanction of %q: %s", p, rite, why)
	}
	return &sn, nil
}

// untrue tells why sn is no true sanction of the rite for uid; "" when it
// is one.
func (sn *Sanction) untrue(rite string, uid int) string {
	switch {
	case !slices.Contains(Patterns, sn.Pattern):
		return fmt.Sprintf("its pattern is %q, yet this servitor reads only %q and %q", sn.Pattern, Pattern,
			PatternMarkI)
	case sn.Misshapen() != "":
		return sn.Misshapen()
	case sn.Rite != rite:
		return fmt.Sprintf("it names the rite %q", sn.Rite)
	case sn.UID != uid:
		return fmt.Sprintf("it was sanctioned by uid %d", sn.UID)
	case sn.Sigil != sn.Unit.Sigil():
		return fmt.Sprintf("its sigil %s is not the sigil of what it holds", Short(sn.Sigil))
	}
	return ""
}

// List reads every sanction of the master, by the name of its rite. A store
// or hall that is not kept lists none; a sanction that cannot be read is
// lamented and left out.
func (s Store) List() ([]*Sanction, []error) {
	if s.Root == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(s.hall())
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, []error{fmt.Errorf("the sanctions at %s cannot be read: %w", s.hall(), err)}
	}
	var names []string
	for _, e := range entries {
		if name, ok := strings.CutSuffix(e.Name(), ".json"); ok && !e.IsDir() {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	var out []*Sanction
	var laments []error
	for _, name := range names {
		sn, err := s.Read(name)
		switch {
		case err != nil:
			laments = append(laments, err)
		case sn != nil:
			out = append(out, sn)
		}
	}
	return out, laments
}
