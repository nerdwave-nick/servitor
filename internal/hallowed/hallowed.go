// Package hallowed performs the acts of the exalted servitor upon vessels and
// tethers safely, though it acts with the root's authority upon places that
// others may influence.
//
// Every act begins with the walk of the trusted-path rule: from the root of
// the machine, one hall after another, on descriptors that are never
// followed by the machine itself. Bonds met on the way are read and walked
// anew, at most forty of them. Every hall the path leads through must be
// owned by the root and written by none but its owner; the vessel itself is
// a scroll, and the name of a tether a bond. What the walk reached is then
// acted upon relative to its hall, so that no bond laid afterwards can turn
// the act aside.
//
// A vessel is transcribed atomically: a fresh scroll beside it in the same
// hall receives the scripture, the owner and the seal of the vessel it
// replaces, and is renamed into its place. Each act remembers what stood
// before it, in memory only, so that it can be undone exactly.
package hallowed

import (
	"errors"
	"io/fs"
)

// Owner is the owner of a vessel: its uid and gid.
type Owner struct{ UID, GID uint32 }

// Ground is where the trusted-path rule stands: the hall every walk begins
// at, whom it holds to be the root, and who owns a vessel brought into
// being. Machine is the machine itself; the trials of the servitor raise
// grounds of their own in temporary halls.
type Ground struct {
	// Root is the hall every walk begins at. Paths given to the acts begin
	// at it, and so do absolute bonds; ".." never leads above it.
	Root string
	// Sovereign reports whether uid is the root's, whom every hall along a
	// path must belong to.
	Sovereign func(uid uint32) bool
	// Lord owns every vessel brought into being.
	Lord Owner
}

// Machine is the ground of the machine itself: walks begin at /, and the
// root is uid 0.
var Machine = Ground{
	Root:      "/",
	Sovereign: func(uid uint32) bool { return uid == 0 },
	Lord:      Owner{UID: 0, GID: 0},
}

// Capacity is the most scripture an act reads from a vessel, in bytes.
const Capacity = 16 << 20

// newSeal is the seal of a vessel brought into being without one.
const newSeal fs.FileMode = 0o644

// maxBonds is how many bonds a walk follows before giving up.
const maxBonds = 40

// doom is what the rule or the servitor's own caution finds at a place,
// spoken after its name.
type doom string

func (d doom) Error() string { return string(d) }

const (
	errAstray      doom = "does not begin at the root"
	errUnsovereign doom = "is a hall the root does not own"
	errOpenHall    doom = "is a hall that others than its owner may write"
	errBondLoop    doom = "leads through bonds in endless circles"
	errNoScroll    doom = "is no scroll the servitor may touch under exaltation"
	errNoBond      doom = "is neither bond nor absent, and gives way only with zeal"
	errHall        doom = "is a hall, which the servitor never casts down"
	errManyNames   doom = "is known to the machine under more than one name"
	errAttributes  doom = "bears attributes the servitor cannot keep, such as access lists or labels"
	errOverflow    doom = "holds more scripture than the servitor reads at once"
	errMissing     doom = "is not to be found"
)

// lament speaks the grimdark words for err, met at place, unless they are
// spoken already. The machine's own answer is kept beneath them, unspoken.
func lament(place string, err error) error {
	var (
		l *lamentation
		d doom
	)
	switch {
	case err == nil:
		return nil
	case errors.As(err, &l):
		return err
	case errors.As(err, &d):
		return &lamentation{place + " " + string(d), err}
	case errors.Is(err, fs.ErrPermission):
		return &lamentation{"the machine spirit denies the servitor access to " + place, err}
	case errors.Is(err, fs.ErrNotExist):
		return &lamentation{place + " " + string(errMissing), errors.Join(errMissing, err)}
	}
	return &lamentation{"the machine resists every attempt to touch " + place, err}
}

// lamentation is the grimdark words for what the machine answered.
type lamentation struct {
	words string
	cause error
}

func (l *lamentation) Error() string { return l.words }
func (l *lamentation) Unwrap() error { return l.cause }
