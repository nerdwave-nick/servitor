package rituals

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/sanctum"
)

// Excommunication tells what was struck and purged.
type Excommunication struct {
	Struck []string // the scriptures struck from the Librarium
	Purged []string // the vessels whose sanctums were purged, their bonds followed
}

// Excommunicate strikes the rite name from the Librarium. With purge, its
// sanctums are first removed from their vessels — the vessel of every
// sanctum step rendered for every aspect, with that aspect's decrees —
// while transcribed vessels and tethers are left as they stand. A purge
// examines every vessel before touching any: a vessel whose sanctum markers
// are broken refuses the whole excommunication. A heretical rite may be
// struck but not purged. The error is *Unrecorded, *Heretical, or the
// lament of a vessel or scripture that would not yield.
func (s *Servitor) Excommunicate(name string, purge bool) (Excommunication, error) {
	var ex Excommunication
	if _, ok := s.Librarium.Scriptures[name]; !ok {
		return ex, &Unrecorded{Name: name, Librarium: s.Librarium.Dir}
	}
	if purge {
		r, err := s.Rite(name)
		if err != nil {
			return ex, err
		}
		plans, err := purgePlans(r)
		if err != nil {
			return ex, err
		}
		for _, p := range plans {
			if err := writeVessel(p.path, p.after, p.mode); err != nil {
				return ex, fmt.Errorf("the purge of the rite %q halts at the vessel %s, which the machine "+
					"spirit will not let be rewritten; the vessels purged before it stay purged", name, p.path)
			}
			ex.Purged = append(ex.Purged, p.path)
		}
	}
	for _, p := range s.Librarium.Scriptures[name] {
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return ex, fmt.Errorf("the scripture %s resists being struck from the Librarium", p)
		}
		ex.Struck = append(ex.Struck, p)
	}
	return ex, nil
}

// purge is one vessel rewritten without the rite's sanctums.
type purge struct {
	path  string
	after string
	mode  fs.FileMode
}

// purgePlans reads every vessel of the rite's sanctums and removes the
// sanctums from what was read, touching nothing; vessels that do not stand,
// or hold no sanctum of the rite, are passed over.
func purgePlans(r *librarium.Rite) ([]purge, error) {
	e := &examiner{rite: r, seen: map[string]bool{}}
	byPath := map[string]*purge{}
	var order []string
	for _, step := range r.Liturgy {
		st, ok := step.(*librarium.Sanctum)
		if !ok {
			continue
		}
		glyph, closing := st.Glyphs()
		m := sanctum.Marker{Glyph: glyph, ClosingGlyph: closing, Ward: st.WardFor(r.Name)}
		for _, aspect := range r.Aspects {
			written, ok := e.path(aspect, st.Vessel)
			if !ok {
				continue
			}
			real, err := filepath.EvalSymlinks(written)
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err != nil {
				return nil, unyielding(r.Name, written)
			}
			p := byPath[real]
			if p == nil {
				info, err := os.Stat(real)
				if err != nil || !info.Mode().IsRegular() {
					return nil, unyielding(r.Name, real)
				}
				data, err := os.ReadFile(real)
				if err != nil {
					return nil, unyielding(r.Name, real)
				}
				p = &purge{path: real, after: string(data), mode: info.Mode().Perm()}
				byPath[real] = p
				order = append(order, real)
			}
			after, err := sanctum.Remove(p.after, m)
			if err != nil {
				return nil, fmt.Errorf("the vessel %s is defiled at %v — mend its markers by hand, for the "+
					"sanctums of the rite %q cannot be purged from it; nothing was touched", real, err, r.Name)
			}
			p.after = after
		}
	}
	var plans []purge
	for _, path := range order {
		plans = append(plans, *byPath[path])
	}
	return plans, nil
}

func unyielding(rite, path string) error {
	return fmt.Errorf("the vessel %s will not yield its scripture, so the sanctums of the rite %q cannot be "+
		"purged from it; nothing was touched", path, rite)
}

// writeVessel writes content to path atomically, sealed with mode.
func writeVessel(path, content string, mode fs.FileMode) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".servitor-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.Remove(tmp.Name())
		}
	}()
	if _, err = tmp.WriteString(content); err == nil {
		err = tmp.Chmod(mode)
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
