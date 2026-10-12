package hallowed

import (
	"errors"
	"testing"
)

// altar raises a temple with halls of the root's own: /etc for bonds, and
// /usr/share/themes holding two anchors.
func altar(t *testing.T) *temple {
	tp := newTemple(t)
	tp.hall("/etc", 0o755)
	tp.hall("/usr/share/themes", 0o755)
	tp.scroll("/usr/share/themes/on", "on", 0o644)
	tp.scroll("/usr/share/themes/off", "off", 0o644)
	return tp
}

func TestBind_BindsRebindsAndUnbinds(t *testing.T) {
	tp := altar(t)
	bound, err := tp.g.Bind("/etc/theme", "/usr/share/themes/on", false)
	if err != nil {
		t.Fatal(err)
	}
	if tp.anchor("/etc/theme") != "/usr/share/themes/on" {
		t.Fatalf("bound to %q", tp.anchor("/etc/theme"))
	}
	rebound, err := tp.g.Bind("/etc/theme", "../usr/share/themes/off", false)
	if err != nil {
		t.Fatal(err)
	}
	if tp.anchor("/etc/theme") != "../usr/share/themes/off" {
		t.Fatalf("rebound to %q", tp.anchor("/etc/theme"))
	}
	unbound, err := tp.g.Unbind("/etc/theme", false)
	if err != nil {
		t.Fatal(err)
	}
	if !tp.absent("/etc/theme") {
		t.Fatal("the bond still stands")
	}

	if err := unbound.Revert(); err != nil || tp.anchor("/etc/theme") != "../usr/share/themes/off" {
		t.Fatalf("unbinding reverted to %q: %v", tp.anchor("/etc/theme"), err)
	}
	if err := rebound.Revert(); err != nil || tp.anchor("/etc/theme") != "/usr/share/themes/on" {
		t.Fatalf("rebinding reverted to %q: %v", tp.anchor("/etc/theme"), err)
	}
	if err := bound.Revert(); err != nil || !tp.absent("/etc/theme") {
		t.Fatalf("binding reverted to presence: %v", err)
	}
	if names := tp.names("/etc"); len(names) != 0 {
		t.Fatalf("the hall holds %v", names)
	}
}

// TestBind_RefusesAnAnchorThroughAnOpenHall: /etc/x → ~/dotfiles/x is
// heresy under exaltation.
func TestBind_RefusesAnAnchorThroughAnOpenHall(t *testing.T) {
	tp := altar(t)
	tp.hall("/home/faithful/dotfiles", 0o755)
	tp.scroll("/home/faithful/dotfiles/theme", "mine", 0o644)
	tp.hall("/home/faithful", 0o775)
	tp.bond("/etc/dotfiles", "/home/faithful/dotfiles")
	for _, anchor := range []string{"/home/faithful/dotfiles/theme", "../home/faithful/dotfiles/theme",
		"/etc/dotfiles/theme", "/home/faithful/dotfiles/absent"} {
		if _, err := tp.g.Bind("/etc/theme", anchor, false); !errors.Is(err, errOpenHall) {
			t.Errorf("bound to %s: %v", anchor, err)
		}
	}
	if !tp.absent("/etc/theme") {
		t.Fatal("a bond was laid")
	}
}

// TestBind_RefusesANameInAnOpenHall: the hall of the bond is judged; the
// bonded name itself is never followed.
func TestBind_RefusesANameInAnOpenHall(t *testing.T) {
	tp := altar(t)
	tp.hall("/var/open", 0o777)
	tp.bond("/etc/open", "/var/open")
	if _, err := tp.g.Bind("/etc/open/theme", "/usr/share/themes/on", false); !errors.Is(err, errOpenHall) {
		t.Fatalf("bound in an open hall: %v", err)
	}
	tp.bond("/etc/theme", "/var/open/theme")
	if _, err := tp.g.Bind("/etc/theme", "/usr/share/themes/on", false); err != nil {
		t.Fatalf("rebinding a bond that leads into an open hall: %v", err)
	}
	if tp.anchor("/etc/theme") != "/usr/share/themes/on" || !tp.absent("/var/open/theme") {
		t.Fatal("the bond was followed")
	}
}

// TestBind_DisplacesAScrollOnlyInZeal: what gave way is raised again
// exactly; a hall never gives way.
func TestBind_DisplacesAScrollOnlyInZeal(t *testing.T) {
	tp := altar(t)
	tp.scroll("/etc/theme", "\x00mine", 0o600)
	if _, err := tp.g.Bind("/etc/theme", "/usr/share/themes/on", false); !errors.Is(err, errNoBond) {
		t.Fatalf("displaced a scroll without zeal: %v", err)
	}
	if _, err := tp.g.Unbind("/etc/theme", false); !errors.Is(err, errNoBond) {
		t.Fatalf("struck a scroll without zeal: %v", err)
	}
	m, err := tp.g.Bind("/etc/theme", "/usr/share/themes/on", true)
	if err != nil {
		t.Fatal(err)
	}
	if tp.anchor("/etc/theme") != "/usr/share/themes/on" {
		t.Fatal("the scroll did not give way")
	}
	if err := m.Revert(); err != nil || tp.content("/etc/theme") != "\x00mine" || seal(tp.stat("/etc/theme")) != 0o600 {
		t.Fatalf("raised again as %q: %v", tp.content("/etc/theme"), err)
	}
	m, err = tp.g.Unbind("/etc/theme", true)
	if err != nil || !tp.absent("/etc/theme") {
		t.Fatalf("unbound in zeal: %v", err)
	}
	if err := m.Revert(); err != nil || tp.content("/etc/theme") != "\x00mine" {
		t.Fatalf("raised again: %v", err)
	}

	tp.hall("/etc/hall", 0o755)
	if _, err := tp.g.Bind("/etc/hall", "/usr/share/themes/on", true); !errors.Is(err, errHall) {
		t.Fatalf("cast down a hall: %v", err)
	}
	if _, err := tp.g.Unbind("/etc/hall", true); !errors.Is(err, errHall) {
		t.Fatalf("cast down a hall: %v", err)
	}
}

func TestUnbind_LeavesAnAbsentNameAbsent(t *testing.T) {
	tp := altar(t)
	m, err := tp.g.Unbind("/etc/theme", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Revert(); err != nil || !tp.absent("/etc/theme") {
		t.Fatalf("reverted to presence: %v", err)
	}
}
