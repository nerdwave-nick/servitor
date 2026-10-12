package hallowed

import "golang.org/x/sys/unix"

const (
	errAlien       doom = "is a program the root does not own"
	errOpenProgram doom = "is a program that others than its owner may write"
	errInert       doom = "is a scroll the machine may not awaken"
)

// Program is the program at path, as the ground names it once its bonds are
// followed: a scroll reached through halls that pass the trusted-path rule,
// owned by the root, written by none but the root and awakened by the
// machine. The exalted servitor speaks in no other tongue.
func (g Ground) Program(path string) (string, error) {
	p, err := g.walk(path, vessel)
	if err != nil {
		return "", err
	}
	defer p.close()
	switch st := p.stat; {
	case st == nil:
		err = errMissing
	case st.Mode&unix.S_IFMT != unix.S_IFREG:
		err = errNoScroll
	case !g.Sovereign(st.Uid):
		err = errAlien
	case st.Mode&0o022 != 0:
		err = errOpenProgram
	case st.Mode&0o111 == 0:
		err = errInert
	}
	if err != nil {
		return "", lament(p.path(), err)
	}
	return p.path(), nil
}
